package parser

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/go-struct-analyzer/internal/types"
)

// Parser 是 Go 源码解析器
type Parser struct {
	fset       *token.FileSet
	files      map[string]*ast.File      // 文件路径 -> AST
	packages   map[string]*ast.Package   // 包名 -> 包
	structs    map[string]*types.StructInfo // 结构体名 -> 结构体信息
	methods    map[string][]types.MethodInfo // 结构体名 -> 方法列表
	interfaces map[string]*types.InterfaceInfo // 接口名 -> 接口信息
	functions  map[string]*types.FunctionInfo  // 函数名 -> 函数信息（用于构造函数检测）
	imports    map[string]map[string]string  // 文件路径 -> (别名 -> 导入路径)
	moduleName string                        // 项目模块名
	verbose    bool
}

// NewParser 创建一个新的解析器
func NewParser(verbose bool) *Parser {
	return &Parser{
		fset:       token.NewFileSet(),
		files:      make(map[string]*ast.File),
		packages:   make(map[string]*ast.Package),
		structs:    make(map[string]*types.StructInfo),
		methods:    make(map[string][]types.MethodInfo),
		interfaces: make(map[string]*types.InterfaceInfo),
		functions:  make(map[string]*types.FunctionInfo),
		imports:    make(map[string]map[string]string),
		verbose:    verbose,
	}
}

// ParseProject 解析整个项目
func (p *Parser) ParseProject(projectPath string) error {
	// 1. 获取模块名
	moduleName, err := p.getModuleName(projectPath)
	if err != nil {
		// 如果没有 go.mod，使用项目目录名
		moduleName = filepath.Base(projectPath)
	}
	p.moduleName = moduleName

	// 2. 递归扫描所有 .go 文件
	goFiles, err := p.findGoFiles(projectPath)
	if err != nil {
		return err
	}

	// 3. 解析每个文件
	for _, filePath := range goFiles {
		astFile, err := parser.ParseFile(p.fset, filePath, nil, parser.ParseComments)
		if err != nil {
			if p.verbose {
				println("Warning: failed to parse file:", filePath, err.Error())
			}
			continue
		}
		p.files[filePath] = astFile
		p.imports[filePath] = p.buildImportMap(astFile)
	}

	// 4. 提取所有结构体
	for filePath, file := range p.files {
		p.extractStructs(file, filePath)
	}

	// 5. 提取所有方法
	for filePath, file := range p.files {
		p.extractMethods(file, filePath)
	}

	// 6. 关联方法到结构体
	for structName, methods := range p.methods {
		if structInfo, ok := p.structs[structName]; ok {
			structInfo.Methods = methods
		}
	}

	// 7. 提取所有接口
	for filePath, file := range p.files {
		p.extractInterfaces(file, filePath)
	}

	// 8. 提取所有函数（用于构造函数检测）
	for filePath, file := range p.files {
		p.extractFunctions(file, filePath)
	}

	return nil
}

// GetModuleName 返回项目模块名
func (p *Parser) GetModuleName() string {
	return p.moduleName
}

// GetStruct 根据名称获取结构体信息
func (p *Parser) GetStruct(name string) *types.StructInfo {
	// 去掉可能的指针符号和包前缀
	name = strings.TrimPrefix(name, "*")
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		name = parts[len(parts)-1]
	}
	return p.structs[name]
}

// GetAllStructs 获取所有结构体信息
func (p *Parser) GetAllStructs() map[string]*types.StructInfo {
	return p.structs
}

// GetFile 获取文件的 AST
func (p *Parser) GetFile(path string) *ast.File {
	return p.files[path]
}

// GetFileSet 获取 FileSet
func (p *Parser) GetFileSet() *token.FileSet {
	return p.fset
}

// getModuleName 从 go.mod 读取模块名
func (p *Parser) getModuleName(projectPath string) (string, error) {
	goModPath := filepath.Join(projectPath, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", os.ErrNotExist
}

// findGoFiles 递归查找所有 .go 文件
func (p *Parser) findGoFiles(root string) ([]string, error) {
	var goFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏目录和 vendor 目录
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// 只处理 .go 文件，跳过测试文件
		if filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go") {
			goFiles = append(goFiles, path)
		}

		return nil
	})

	return goFiles, err
}

// buildImportMap 构建导入映射
func (p *Parser) buildImportMap(file *ast.File) map[string]string {
	importMap := make(map[string]string)

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		var pkgName string
		if imp.Name != nil {
			// 有别名: import foo "github.com/bar/baz"
			pkgName = imp.Name.Name
		} else {
			// 无别名: import "github.com/bar/baz"
			parts := strings.Split(path, "/")
			pkgName = parts[len(parts)-1]
		}

		importMap[pkgName] = path
	}

	return importMap
}

// extractStructs 从文件中提取结构体
func (p *Parser) extractStructs(file *ast.File, filePath string) {
	packageName := file.Name.Name

	ast.Inspect(file, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			info := &types.StructInfo{
				Name:       typeSpec.Name.Name,
				Package:    packageName,
				FilePath:   filePath,
				SourceCode: p.nodeToString(genDecl),
				Fields:     p.extractFields(structType),
			}

			p.structs[info.Name] = info
		}

		return true
	})
}

// extractFields 从结构体中提取字段
func (p *Parser) extractFields(structType *ast.StructType) []types.FieldInfo {
	var fields []types.FieldInfo

	if structType.Fields == nil {
		return fields
	}

	for _, field := range structType.Fields.List {
		typeName := p.getTypeName(field.Type)
		tag := ""
		if field.Tag != nil {
			tag = field.Tag.Value
		}

		if len(field.Names) == 0 {
			// 嵌入字段
			fields = append(fields, types.FieldInfo{
				Name:       typeName,
				Type:       typeName,
				Tag:        tag,
				IsExported: isExported(typeName),
				IsEmbedded: true,
			})
		} else {
			for _, name := range field.Names {
				fields = append(fields, types.FieldInfo{
					Name:       name.Name,
					Type:       typeName,
					Tag:        tag,
					IsExported: isExported(name.Name),
					IsEmbedded: false,
				})
			}
		}
	}

	return fields
}

// extractMethods 从文件中提取方法
func (p *Parser) extractMethods(file *ast.File, filePath string) {
	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			return true
		}

		// 获取接收者类型
		receiverType := p.getReceiverType(funcDecl.Recv)
		baseType := strings.TrimPrefix(receiverType, "*")

		methodInfo := types.MethodInfo{
			Name:       funcDecl.Name.Name,
			Signature:  p.getMethodSignature(funcDecl),
			Receiver:   receiverType,
			IsExported: isExported(funcDecl.Name.Name),
			SourceCode: p.nodeToString(funcDecl),
		}

		p.methods[baseType] = append(p.methods[baseType], methodInfo)

		return true
	})
}

// getReceiverType 获取方法接收者类型
func (p *Parser) getReceiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	return p.getTypeName(recv.List[0].Type)
}

// getMethodSignature 获取方法签名
func (p *Parser) getMethodSignature(funcDecl *ast.FuncDecl) string {
	var sig bytes.Buffer

	// 参数
	sig.WriteString("(")
	if funcDecl.Type.Params != nil {
		for i, param := range funcDecl.Type.Params.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			for j, name := range param.Names {
				if j > 0 {
					sig.WriteString(", ")
				}
				sig.WriteString(name.Name)
			}
			if len(param.Names) > 0 {
				sig.WriteString(" ")
			}
			sig.WriteString(p.getTypeName(param.Type))
		}
	}
	sig.WriteString(")")

	// 返回值
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		sig.WriteString(" ")
		if len(funcDecl.Type.Results.List) > 1 {
			sig.WriteString("(")
		}
		for i, result := range funcDecl.Type.Results.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(p.getTypeName(result.Type))
		}
		if len(funcDecl.Type.Results.List) > 1 {
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// getTypeName 获取类型名称
func (p *Parser) getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.getTypeName(t.X)
	case *ast.SelectorExpr:
		return p.getTypeName(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + p.getTypeName(t.Elt)
		}
		return "[...]" + p.getTypeName(t.Elt)
	case *ast.MapType:
		return "map[" + p.getTypeName(t.Key) + "]" + p.getTypeName(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func"
	case *ast.ChanType:
		return "chan " + p.getTypeName(t.Value)
	case *ast.Ellipsis:
		return "..." + p.getTypeName(t.Elt)
	default:
		return "unknown"
	}
}

// nodeToString 将 AST 节点转换为字符串
func (p *Parser) nodeToString(node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, p.fset, node)
	return buf.String()
}

// isExported 判断名称是否导出
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// GetImports 获取文件的导入映射
func (p *Parser) GetImports(filePath string) map[string]string {
	return p.imports[filePath]
}

// extractInterfaces 从文件中提取接口
func (p *Parser) extractInterfaces(file *ast.File, filePath string) {
	packageName := file.Name.Name

	ast.Inspect(file, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			info := &types.InterfaceInfo{
				Name:       typeSpec.Name.Name,
				Package:    packageName,
				FilePath:   filePath,
				Methods:    p.extractInterfaceMethods(interfaceType),
				SourceCode: p.nodeToString(genDecl),
			}

			p.interfaces[info.Name] = info
		}

		return true
	})
}

// extractInterfaceMethods 提取接口方法
func (p *Parser) extractInterfaceMethods(interfaceType *ast.InterfaceType) []types.InterfaceMethod {
	var methods []types.InterfaceMethod

	if interfaceType.Methods == nil {
		return methods
	}

	for _, field := range interfaceType.Methods.List {
		// 检查是否是方法（有函数类型）
		funcType, ok := field.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		for _, name := range field.Names {
			methods = append(methods, types.InterfaceMethod{
				Name:      name.Name,
				Signature: p.getFuncSignature(funcType),
			})
		}
	}

	return methods
}

// getFuncSignature 获取函数签名（参数和返回值）
func (p *Parser) getFuncSignature(funcType *ast.FuncType) string {
	var sig bytes.Buffer

	// 参数
	sig.WriteString("(")
	if funcType.Params != nil {
		for i, param := range funcType.Params.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			for j, name := range param.Names {
				if j > 0 {
					sig.WriteString(", ")
				}
				sig.WriteString(name.Name)
			}
			if len(param.Names) > 0 {
				sig.WriteString(" ")
			}
			sig.WriteString(p.getTypeName(param.Type))
		}
	}
	sig.WriteString(")")

	// 返回值
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		sig.WriteString(" ")
		if len(funcType.Results.List) > 1 {
			sig.WriteString("(")
		}
		for i, result := range funcType.Results.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(p.getTypeName(result.Type))
		}
		if len(funcType.Results.List) > 1 {
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// extractFunctions 从文件中提取函数（用于构造函数检测）
func (p *Parser) extractFunctions(file *ast.File, filePath string) {
	packageName := file.Name.Name

	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Recv != nil { // 跳过方法，只处理函数
			return true
		}

		// 只处理 New 开头的函数（构造函数模式）
		funcName := funcDecl.Name.Name
		if !strings.HasPrefix(funcName, "New") {
			return true
		}

		// 获取返回类型
		returnType := p.getReturnType(funcDecl)
		if returnType == "" {
			return true
		}

		info := &types.FunctionInfo{
			Name:       funcName,
			Package:    packageName,
			FilePath:   filePath,
			ReturnType: returnType,
			Signature:  p.getMethodSignature(funcDecl),
		}

		// 使用包名+函数名作为 key，避免跨包冲突
		key := packageName + "." + funcName
		p.functions[key] = info

		return true
	})
}

// getReturnType 获取函数的主要返回类型
func (p *Parser) getReturnType(funcDecl *ast.FuncDecl) string {
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) == 0 {
		return ""
	}

	// 获取第一个返回值类型（通常是主要类型，error 一般在后面）
	firstResult := funcDecl.Type.Results.List[0]
	return p.getTypeName(firstResult.Type)
}

// GetAllInterfaces 获取所有接口信息
func (p *Parser) GetAllInterfaces() map[string]*types.InterfaceInfo {
	return p.interfaces
}

// GetInterface 根据名称获取接口信息
func (p *Parser) GetInterface(name string) *types.InterfaceInfo {
	return p.interfaces[name]
}

// GetFunction 根据名称获取函数信息
func (p *Parser) GetFunction(name string) *types.FunctionInfo {
	return p.functions[name]
}

// GetAllFunctions 获取所有函数信息
func (p *Parser) GetAllFunctions() map[string]*types.FunctionInfo {
	return p.functions
}

// GetFunctionByReturnType 根据返回类型查找构造函数
func (p *Parser) GetFunctionByReturnType(returnType string) *types.FunctionInfo {
	// 标准化返回类型（去除指针符号）
	baseType := strings.TrimPrefix(returnType, "*")

	for _, fn := range p.functions {
		fnBaseType := strings.TrimPrefix(fn.ReturnType, "*")
		if fnBaseType == baseType || fnBaseType == returnType {
			return fn
		}
	}
	return nil
}
