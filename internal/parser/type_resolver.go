package parser

import (
	"go/ast"
	"go/token"
	"strings"
)

// TypeResolver 用于解析和推断类型
type TypeResolver struct {
	parser *Parser
}

// NewTypeResolver 创建类型解析器
func NewTypeResolver(p *Parser) *TypeResolver {
	return &TypeResolver{parser: p}
}

// TypeContext 表示方法体中的变量类型上下文
type TypeContext struct {
	variables map[string]string // 变量名 -> 类型名
}

// NewTypeContext 创建类型上下文
func NewTypeContext() *TypeContext {
	return &TypeContext{
		variables: make(map[string]string),
	}
}

// SetType 设置变量类型
func (ctx *TypeContext) SetType(name, typeName string) {
	ctx.variables[name] = typeName
}

// GetType 获取变量类型
func (ctx *TypeContext) GetType(name string) string {
	return ctx.variables[name]
}

// BuildTypeContext 从方法体构建类型上下文
func (r *TypeResolver) BuildTypeContext(body *ast.BlockStmt) *TypeContext {
	ctx := NewTypeContext()

	if body == nil {
		return ctx
	}

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		// 短变量声明: b := B{}
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				r.handleAssignment(node, ctx)
			}
		// 变量声明: var b B
		case *ast.DeclStmt:
			if genDecl, ok := node.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
				r.handleVarDecl(genDecl, ctx)
			}
		}
		return true
	})

	return ctx
}

// handleAssignment 处理赋值语句
func (r *TypeResolver) handleAssignment(assign *ast.AssignStmt, ctx *TypeContext) {
	for i, lhs := range assign.Lhs {
		if i >= len(assign.Rhs) {
			break
		}

		if ident, ok := lhs.(*ast.Ident); ok {
			typeName := r.InferTypeFromExpr(assign.Rhs[i])
			if typeName != "" {
				ctx.SetType(ident.Name, typeName)
			}
		}
	}
}

// handleVarDecl 处理变量声明
func (r *TypeResolver) handleVarDecl(genDecl *ast.GenDecl, ctx *TypeContext) {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		typeName := ""
		if valueSpec.Type != nil {
			typeName = r.parser.getTypeName(valueSpec.Type)
		}

		for i, name := range valueSpec.Names {
			if typeName == "" && i < len(valueSpec.Values) {
				typeName = r.InferTypeFromExpr(valueSpec.Values[i])
			}
			if typeName != "" {
				ctx.SetType(name.Name, typeName)
			}
		}
	}
}

// InferTypeFromExpr 从表达式推断类型
func (r *TypeResolver) InferTypeFromExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	// 复合字面量: B{}
	case *ast.CompositeLit:
		if e.Type != nil {
			return r.parser.getTypeName(e.Type)
		}
	// 函数调用: new(B), NewB()
	case *ast.CallExpr:
		return r.inferTypeFromCall(e)
	// 取地址: &B{}
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return "*" + r.InferTypeFromExpr(e.X)
		}
	// 类型断言: x.(B)
	case *ast.TypeAssertExpr:
		if e.Type != nil {
			return r.parser.getTypeName(e.Type)
		}
	// 标识符: 可能是已知变量
	case *ast.Ident:
		// 返回空，让调用者使用上下文查找
		return ""
	}
	return ""
}

// inferTypeFromCall 从函数调用推断类型
func (r *TypeResolver) inferTypeFromCall(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		// new(B) -> *B
		if fun.Name == "new" && len(call.Args) > 0 {
			return "*" + r.parser.getTypeName(call.Args[0])
		}
		// make([]B, n) -> []B
		if fun.Name == "make" && len(call.Args) > 0 {
			return r.parser.getTypeName(call.Args[0])
		}
		// NewXxx() -> 可能返回 *Xxx
		if len(fun.Name) > 3 && strings.HasPrefix(fun.Name, "New") {
			return "*" + fun.Name[3:]
		}
	case *ast.SelectorExpr:
		// pkg.NewXxx() -> 可能返回 *pkg.Xxx
		if len(fun.Sel.Name) > 3 && strings.HasPrefix(fun.Sel.Name, "New") {
			pkgName := r.parser.getTypeName(fun.X)
			return "*" + pkgName + "." + fun.Sel.Name[3:]
		}
	}
	return ""
}

// InferType 从表达式和上下文推断类型
func (r *TypeResolver) InferType(expr ast.Expr, ctx *TypeContext) string {
	// 先尝试直接推断
	if typeName := r.InferTypeFromExpr(expr); typeName != "" {
		return typeName
	}

	// 如果是标识符，从上下文查找
	if ident, ok := expr.(*ast.Ident); ok {
		if typeName := ctx.GetType(ident.Name); typeName != "" {
			return typeName
		}
		// 变量名无法推断类型时，返回空字符串而不是变量名
		// 避免将变量名误识别为类型名
		return ""
	}

	// 对于非标识符表达式（如选择器表达式 pkg.Type），尝试获取类型名
	return r.parser.getTypeName(expr)
}

// ExtractBaseType 提取基础类型名（去掉指针、切片等修饰符）
func ExtractBaseType(typeName string) string {
	// 去掉指针
	typeName = strings.TrimPrefix(typeName, "*")
	// 去掉切片
	typeName = strings.TrimPrefix(typeName, "[]")
	// 去掉 map
	if strings.HasPrefix(typeName, "map[") {
		// 简化处理，取值类型
		if idx := strings.Index(typeName, "]"); idx != -1 {
			typeName = typeName[idx+1:]
		}
	}
	// 取最后一部分（结构体名）
	parts := strings.Split(typeName, ".")
	return parts[len(parts)-1]
}
