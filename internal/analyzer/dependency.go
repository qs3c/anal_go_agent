package analyzer

import (
	"go/ast"
	"strings"

	"github.com/user/go-struct-analyzer/internal/parser"
	"github.com/user/go-struct-analyzer/internal/types"
)

// DependencyAnalyzer 分析结构体之间的依赖关系
type DependencyAnalyzer struct {
	parser       *parser.Parser
	typeResolver *parser.TypeResolver
	filter       *ScopeFilter
	verbose      bool
}

// NewDependencyAnalyzer 创建依赖分析器
func NewDependencyAnalyzer(p *parser.Parser, filter *ScopeFilter, verbose bool) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		parser:       p,
		typeResolver: parser.NewTypeResolver(p),
		filter:       filter,
		verbose:      verbose,
	}
}

// AnalyzeStruct 分析单个结构体的依赖关系
func (a *DependencyAnalyzer) AnalyzeStruct(structInfo *types.StructInfo) []types.Dependency {
	var deps []types.Dependency

	// 1. 分析字段依赖
	deps = append(deps, a.analyzeFieldDeps(structInfo)...)

	// 2. 分析方法内的依赖
	deps = append(deps, a.analyzeMethodDeps(structInfo)...)

	// 3. 分析接口实现关系
	deps = append(deps, a.analyzeInterfaceImpl(structInfo)...)

	// 去重
	deps = a.deduplicateDeps(deps)

	return deps
}

// analyzeFieldDeps 分析字段依赖
func (a *DependencyAnalyzer) analyzeFieldDeps(structInfo *types.StructInfo) []types.Dependency {
	var deps []types.Dependency

	for _, field := range structInfo.Fields {
		baseType := parser.ExtractBaseType(field.Type)

		if !a.filter.ShouldAnalyze(baseType) {
			continue
		}

		depType := types.DepTypeField
		if field.IsEmbedded {
			depType = types.DepTypeEmbed
		}

		deps = append(deps, types.Dependency{
			From:    structInfo.Name,
			To:      baseType,
			Type:    depType,
			Context: field.Name + " 字段",
		})
	}

	return deps
}

// analyzeMethodDeps 分析方法内的依赖
func (a *DependencyAnalyzer) analyzeMethodDeps(structInfo *types.StructInfo) []types.Dependency {
	var deps []types.Dependency

	// 遍历项目中的所有文件，找到该结构体的方法
	for _, file := range a.getFilesForStruct(structInfo) {
		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil || funcDecl.Body == nil {
				return true
			}

			// 检查是否是该结构体的方法
			receiverType := a.getReceiverTypeName(funcDecl.Recv)
			if receiverType != structInfo.Name && receiverType != "*"+structInfo.Name {
				return true
			}

			// 分析方法体
			methodDeps := a.analyzeMethodBody(structInfo.Name, funcDecl)
			deps = append(deps, methodDeps...)

			return true
		})
	}

	return deps
}

// getFilesForStruct 获取包含该结构体的文件
func (a *DependencyAnalyzer) getFilesForStruct(structInfo *types.StructInfo) []*ast.File {
	var files []*ast.File

	// 首先添加结构体定义所在的文件
	if file := a.parser.GetFile(structInfo.FilePath); file != nil {
		files = append(files, file)
	}

	// 遍历所有结构体，找到同包的文件（方法可能在不同文件中）
	for _, s := range a.parser.GetAllStructs() {
		if s.Package == structInfo.Package {
			if f := a.parser.GetFile(s.FilePath); f != nil {
				// 避免重复添加
				found := false
				for _, existing := range files {
					if existing == f {
						found = true
						break
					}
				}
				if !found {
					files = append(files, f)
				}
			}
		}
	}

	return files
}

// analyzeMethodBody 分析方法体内的依赖
func (a *DependencyAnalyzer) analyzeMethodBody(structName string, funcDecl *ast.FuncDecl) []types.Dependency {
	var deps []types.Dependency
	methodName := funcDecl.Name.Name

	// 构建类型上下文
	ctx := a.typeResolver.BuildTypeContext(funcDecl.Body)

	// 遍历方法体
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		// 复合字面量: B{}
		case *ast.CompositeLit:
			typeName := a.typeResolver.InferTypeFromExpr(node)
			baseType := parser.ExtractBaseType(typeName)
			if a.filter.ShouldAnalyze(baseType) {
				deps = append(deps, types.Dependency{
					From:    structName,
					To:      baseType,
					Type:    types.DepTypeInit,
					Context: methodName + " 方法",
				})
			}

		// 函数调用
		case *ast.CallExpr:
			// 检查 new(B)
			if ident, ok := node.Fun.(*ast.Ident); ok {
				if ident.Name == "new" {
					if len(node.Args) > 0 {
						typeName := a.parser.GetStruct(a.getTypeName(node.Args[0]))
						if typeName != nil && a.filter.ShouldAnalyze(typeName.Name) {
							deps = append(deps, types.Dependency{
								From:    structName,
								To:      typeName.Name,
								Type:    types.DepTypeInit,
								Context: methodName + " 方法",
							})
						}
					}
				} else if strings.HasPrefix(ident.Name, "New") {
					// 检查构造函数调用: NewXxx()
					dep := a.analyzeConstructorCall(structName, methodName, ident.Name, "")
					if dep != nil {
						deps = append(deps, *dep)
					}
				}
			}

			// 检查方法调用: b.Method() 或包调用: pkg.NewXxx()
			if selExpr, ok := node.Fun.(*ast.SelectorExpr); ok {
				methodOrFuncName := selExpr.Sel.Name

				// 检查是否是构造函数调用: pkg.NewXxx()
				if strings.HasPrefix(methodOrFuncName, "New") {
					if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
						dep := a.analyzeConstructorCall(structName, methodName, methodOrFuncName, pkgIdent.Name)
						if dep != nil {
							deps = append(deps, *dep)
							return true
						}
					}
				}

				// 检查方法调用: b.Method()
				receiverType := a.typeResolver.InferType(selExpr.X, ctx)
				// 跳过无法推断类型的情况（避免将变量名误识别为类型名）
				if receiverType == "" {
					return true
				}
				baseType := parser.ExtractBaseType(receiverType)
				if a.filter.ShouldAnalyze(baseType) {
					deps = append(deps, types.Dependency{
						From:    structName,
						To:      baseType,
						Type:    types.DepTypeMethodCall,
						Context: methodName + " -> " + methodOrFuncName,
					})
				}
			}
		}

		return true
	})

	return deps
}

// getReceiverTypeName 获取接收者类型名
func (a *DependencyAnalyzer) getReceiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	return a.getTypeName(recv.List[0].Type)
}

// getTypeName 获取类型名称
func (a *DependencyAnalyzer) getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + a.getTypeName(t.X)
	case *ast.SelectorExpr:
		return a.getTypeName(t.X) + "." + t.Sel.Name
	default:
		return ""
	}
}

// deduplicateDeps 去除重复的依赖
func (a *DependencyAnalyzer) deduplicateDeps(deps []types.Dependency) []types.Dependency {
	seen := make(map[string]bool)
	var result []types.Dependency

	for _, dep := range deps {
		key := dep.From + "->" + dep.To + ":" + dep.Type
		if !seen[key] {
			seen[key] = true
			result = append(result, dep)
		}
	}

	return result
}

// analyzeInterfaceImpl 分析结构体实现的接口
func (a *DependencyAnalyzer) analyzeInterfaceImpl(structInfo *types.StructInfo) []types.Dependency {
	var deps []types.Dependency

	// 获取结构体的所有方法
	structMethods := make(map[string]string) // 方法名 -> 签名
	for _, method := range structInfo.Methods {
		structMethods[method.Name] = method.Signature
	}

	// 检查所有接口
	for _, iface := range a.parser.GetAllInterfaces() {
		// 跳过不在分析范围内的接口
		if !a.filter.ShouldAnalyze(iface.Name) {
			continue
		}

		// 检查结构体是否实现了该接口的所有方法
		if a.implementsInterface(structMethods, iface) {
			deps = append(deps, types.Dependency{
				From:    structInfo.Name,
				To:      iface.Name,
				Type:    types.DepTypeInterface,
				Context: "实现接口",
			})
		}
	}

	return deps
}

// implementsInterface 检查结构体是否实现了接口
func (a *DependencyAnalyzer) implementsInterface(structMethods map[string]string, iface *types.InterfaceInfo) bool {
	if len(iface.Methods) == 0 {
		// 空接口，所有类型都实现
		return false // 不记录空接口的实现关系
	}

	for _, ifaceMethod := range iface.Methods {
		structSig, exists := structMethods[ifaceMethod.Name]
		if !exists {
			return false
		}

		// 比较方法签名（简化比较，只比较方法名存在性）
		// 完整的签名比较需要更复杂的类型匹配
		_ = structSig // 暂时只检查方法名
	}

	return true
}

// analyzeConstructorCall 分析构造函数调用
func (a *DependencyAnalyzer) analyzeConstructorCall(structName, methodName, funcName, pkgAlias string) *types.Dependency {
	// 尝试从函数名推断返回类型
	// NewUserService -> UserService
	// NewCache -> Cache
	inferredType := strings.TrimPrefix(funcName, "New")
	if inferredType == "" {
		return nil
	}

	// 如果有包别名，尝试查找完整的函数信息
	if pkgAlias != "" {
		// 在已解析的函数中查找
		for key, fn := range a.parser.GetAllFunctions() {
			if strings.HasSuffix(key, "."+funcName) {
				baseType := parser.ExtractBaseType(fn.ReturnType)
				if a.filter.ShouldAnalyze(baseType) {
					return &types.Dependency{
						From:    structName,
						To:      baseType,
						Type:    types.DepTypeConstructor,
						Context: methodName + " -> " + funcName,
					}
				}
			}
		}
	}

	// 直接使用推断的类型名
	if a.filter.ShouldAnalyze(inferredType) {
		// 验证推断的类型确实存在
		if a.parser.GetStruct(inferredType) != nil {
			return &types.Dependency{
				From:    structName,
				To:      inferredType,
				Type:    types.DepTypeConstructor,
				Context: methodName + " -> " + funcName,
			}
		}
	}

	return nil
}
