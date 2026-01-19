package analyzer

import (
	"go/ast"

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
			if ident, ok := node.Fun.(*ast.Ident); ok && ident.Name == "new" {
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
			}

			// 检查方法调用: b.Method()
			if selExpr, ok := node.Fun.(*ast.SelectorExpr); ok {
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
						Context: methodName + " -> " + selExpr.Sel.Name,
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
