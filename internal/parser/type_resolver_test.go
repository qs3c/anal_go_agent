package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestExtractBaseType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple type", "User", "User"},
		{"pointer type", "*User", "User"},
		{"slice type", "[]User", "User"},
		{"pointer to slice", "*[]User", "User"},
		{"package type", "model.User", "User"},
		{"pointer package type", "*model.User", "User"},
		{"double pointer", "**User", "*User"}, // ExtractBaseType only removes one level
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBaseType(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractBaseType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTypeContext(t *testing.T) {
	ctx := NewTypeContext()

	// 测试设置和获取类型
	ctx.SetType("user", "*User")
	ctx.SetType("cache", "*Cache")

	if got := ctx.GetType("user"); got != "*User" {
		t.Errorf("GetType(user) = %q, want %q", got, "*User")
	}

	if got := ctx.GetType("cache"); got != "*Cache" {
		t.Errorf("GetType(cache) = %q, want %q", got, "*Cache")
	}

	// 测试获取不存在的类型
	if got := ctx.GetType("notexist"); got != "" {
		t.Errorf("GetType(notexist) = %q, want empty string", got)
	}
}

func TestInferType_VariableNotInContext(t *testing.T) {
	// 创建一个简单的解析器
	p := NewParser(false)
	resolver := NewTypeResolver(p)
	ctx := NewTypeContext()

	// 创建一个标识符表达式（模拟变量名）
	ident := &ast.Ident{Name: "cache"}

	// 当变量不在上下文中时，应返回空字符串
	result := resolver.InferType(ident, ctx)
	if result != "" {
		t.Errorf("InferType for unknown variable should return empty string, got %q", result)
	}
}

func TestInferType_VariableInContext(t *testing.T) {
	p := NewParser(false)
	resolver := NewTypeResolver(p)
	ctx := NewTypeContext()

	// 设置变量类型
	ctx.SetType("cache", "*Cache")

	ident := &ast.Ident{Name: "cache"}

	result := resolver.InferType(ident, ctx)
	if result != "*Cache" {
		t.Errorf("InferType for known variable should return type, got %q, want %q", result, "*Cache")
	}
}

func TestInferTypeFromExpr_CompositeLit(t *testing.T) {
	p := NewParser(false)
	resolver := NewTypeResolver(p)

	// 解析一段代码来获取 AST
	src := `package test
func foo() {
	x := User{}
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// 找到复合字面量
	var compositeLit *ast.CompositeLit
	ast.Inspect(file, func(n ast.Node) bool {
		if cl, ok := n.(*ast.CompositeLit); ok {
			compositeLit = cl
			return false
		}
		return true
	})

	if compositeLit == nil {
		t.Fatal("Failed to find composite literal")
	}

	result := resolver.InferTypeFromExpr(compositeLit)
	if result != "User" {
		t.Errorf("InferTypeFromExpr(User{}) = %q, want %q", result, "User")
	}
}

func TestInferTypeFromExpr_NewCall(t *testing.T) {
	p := NewParser(false)
	resolver := NewTypeResolver(p)

	// 解析 new(User) 调用
	src := `package test
func foo() {
	x := new(User)
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// 找到函数调用
	var callExpr *ast.CallExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			callExpr = ce
			return false
		}
		return true
	})

	if callExpr == nil {
		t.Fatal("Failed to find call expression")
	}

	result := resolver.InferTypeFromExpr(callExpr)
	if result != "*User" {
		t.Errorf("InferTypeFromExpr(new(User)) = %q, want %q", result, "*User")
	}
}

func TestInferTypeFromExpr_AddressOf(t *testing.T) {
	p := NewParser(false)
	resolver := NewTypeResolver(p)

	// 解析 &User{} 表达式
	src := `package test
func foo() {
	x := &User{}
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// 找到一元表达式
	var unaryExpr *ast.UnaryExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if ue, ok := n.(*ast.UnaryExpr); ok {
			unaryExpr = ue
			return false
		}
		return true
	})

	if unaryExpr == nil {
		t.Fatal("Failed to find unary expression")
	}

	result := resolver.InferTypeFromExpr(unaryExpr)
	if result != "*User" {
		t.Errorf("InferTypeFromExpr(&User{}) = %q, want %q", result, "*User")
	}
}

func TestBuildTypeContext(t *testing.T) {
	p := NewParser(false)
	resolver := NewTypeResolver(p)

	// 解析包含变量声明的代码
	src := `package test
func foo() {
	user := User{}
	cache := &Cache{}
	var db Database
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// 找到函数体
	var body *ast.BlockStmt
	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			body = fd.Body
			return false
		}
		return true
	})

	if body == nil {
		t.Fatal("Failed to find function body")
	}

	ctx := resolver.BuildTypeContext(body)

	// 验证变量类型
	tests := []struct {
		varName      string
		expectedType string
	}{
		{"user", "User"},
		{"cache", "*Cache"},
		{"db", "Database"},
	}

	for _, tt := range tests {
		got := ctx.GetType(tt.varName)
		if got != tt.expectedType {
			t.Errorf("ctx.GetType(%q) = %q, want %q", tt.varName, got, tt.expectedType)
		}
	}
}
