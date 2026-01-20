package llm

import (
	"strings"
	"testing"

	"github.com/user/go-struct-analyzer/internal/types"
)

// ==================== cleanResponse 测试 ====================

func TestCleanResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "pure JSON",
			input:    `{"struct_description": "test"}`,
			expected: `{"struct_description": "test"}`,
		},
		{
			name:     "JSON with spaces",
			input:    `  {"struct_description": "test"}  `,
			expected: `{"struct_description": "test"}`,
		},
		{
			name: "JSON with markdown code block",
			input: "```json\n{\"struct_description\": \"test\"}\n```",
			expected: `{"struct_description": "test"}`,
		},
		{
			name: "JSON with plain code block",
			input: "```\n{\"struct_description\": \"test\"}\n```",
			expected: `{"struct_description": "test"}`,
		},
		{
			name:     "JSON with prefix text",
			input:    "Here is the result: {\"struct_description\": \"test\"}",
			expected: `{"struct_description": "test"}`,
		},
		{
			name:     "JSON with suffix text",
			input:    "{\"struct_description\": \"test\"} That's the analysis.",
			expected: `{"struct_description": "test"}`,
		},
		{
			name:     "JSON with both prefix and suffix",
			input:    "Analysis: {\"struct_description\": \"test\"} Done.",
			expected: `{"struct_description": "test"}`,
		},
		{
			name: "multiline JSON",
			input: `{
  "struct_description": "test",
  "fields": []
}`,
			expected: `{
  "struct_description": "test",
  "fields": []
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanResponse(tt.input)
			if result != tt.expected {
				t.Errorf("cleanResponse() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// ==================== parseResponse 测试 ====================

func TestParseResponse_ValidJSON(t *testing.T) {
	input := `{
		"struct_description": "用户服务，处理用户业务",
		"fields": [
			{"name": "repo", "description": "用户数据仓库"},
			{"name": "cache", "description": "缓存服务"}
		],
		"methods": [
			{"name": "CreateUser", "description": "创建新用户"},
			{"name": "GetUser", "description": "获取用户信息"}
		]
	}`

	result, err := parseResponse(input)
	if err != nil {
		t.Fatalf("parseResponse failed: %v", err)
	}

	if result.StructDescription != "用户服务，处理用户业务" {
		t.Errorf("StructDescription = %q, want %q", result.StructDescription, "用户服务，处理用户业务")
	}

	if len(result.Fields) != 2 {
		t.Errorf("Fields count = %d, want 2", len(result.Fields))
	}

	if len(result.Methods) != 2 {
		t.Errorf("Methods count = %d, want 2", len(result.Methods))
	}

	// 验证字段内容
	if result.Fields[0].Name != "repo" {
		t.Errorf("First field name = %q, want %q", result.Fields[0].Name, "repo")
	}
	if result.Fields[0].Description != "用户数据仓库" {
		t.Errorf("First field description = %q, want %q", result.Fields[0].Description, "用户数据仓库")
	}

	// 验证方法内容
	if result.Methods[0].Name != "CreateUser" {
		t.Errorf("First method name = %q, want %q", result.Methods[0].Name, "CreateUser")
	}
}

func TestParseResponse_WithMarkdown(t *testing.T) {
	input := "```json\n{\"struct_description\": \"测试服务\", \"fields\": [], \"methods\": []}\n```"

	result, err := parseResponse(input)
	if err != nil {
		t.Fatalf("parseResponse failed with markdown: %v", err)
	}

	if result.StructDescription != "测试服务" {
		t.Errorf("StructDescription = %q, want %q", result.StructDescription, "测试服务")
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	input := "This is not valid JSON"

	_, err := parseResponse(input)
	if err == nil {
		t.Error("parseResponse should return error for invalid JSON")
	}
}

func TestParseResponse_EmptyFields(t *testing.T) {
	input := `{"struct_description": "简单结构体", "fields": [], "methods": []}`

	result, err := parseResponse(input)
	if err != nil {
		t.Fatalf("parseResponse failed: %v", err)
	}

	if len(result.Fields) != 0 {
		t.Errorf("Fields should be empty, got %d", len(result.Fields))
	}
	if len(result.Methods) != 0 {
		t.Errorf("Methods should be empty, got %d", len(result.Methods))
	}
}

// ==================== buildPrompt 测试 ====================

func TestBuildPrompt(t *testing.T) {
	info := &types.StructInfo{
		Name:    "UserService",
		Package: "service",
		SourceCode: `type UserService struct {
	repo *UserRepository
	cache *Cache
}`,
		Methods: []types.MethodInfo{
			{
				Name:       "CreateUser",
				Signature:  "func (s *UserService) CreateUser(name string) error",
				SourceCode: "func (s *UserService) CreateUser(name string) error {\n\treturn s.repo.Save(name)\n}",
			},
		},
	}

	prompt := buildPrompt(info)

	// 验证包含关键信息
	checks := []string{
		"UserService",
		"service",
		"type UserService struct",
		"repo *UserRepository",
		"CreateUser",
		"JSON",
		"struct_description",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("Prompt should contain %q", check)
		}
	}
}

func TestBuildPrompt_NoMethods(t *testing.T) {
	info := &types.StructInfo{
		Name:       "SimpleStruct",
		Package:    "model",
		SourceCode: "type SimpleStruct struct {\n\tID int\n}",
		Methods:    []types.MethodInfo{},
	}

	prompt := buildPrompt(info)

	if !strings.Contains(prompt, "SimpleStruct") {
		t.Error("Prompt should contain struct name")
	}
	if !strings.Contains(prompt, "// 无方法") {
		t.Error("Prompt should indicate no methods")
	}
}

// ==================== buildMethodsCode 测试 ====================

func TestBuildMethodsCode(t *testing.T) {
	methods := []types.MethodInfo{
		{
			Name:       "Method1",
			SourceCode: "func (s *S) Method1() {}",
		},
		{
			Name:       "Method2",
			SourceCode: "func (s *S) Method2() {}",
		},
	}

	result := buildMethodsCode(methods)

	if !strings.Contains(result, "Method1") {
		t.Error("Should contain Method1")
	}
	if !strings.Contains(result, "Method2") {
		t.Error("Should contain Method2")
	}
}

func TestBuildMethodsCode_Empty(t *testing.T) {
	result := buildMethodsCode([]types.MethodInfo{})

	if result != "// 无方法" {
		t.Errorf("Empty methods should return '// 无方法', got %q", result)
	}
}

func TestBuildMethodsCode_SingleMethod(t *testing.T) {
	methods := []types.MethodInfo{
		{
			Name:       "OnlyMethod",
			SourceCode: "func (s *S) OnlyMethod() error { return nil }",
		},
	}

	result := buildMethodsCode(methods)

	if result != "func (s *S) OnlyMethod() error { return nil }" {
		t.Errorf("Single method result mismatch: %q", result)
	}
}

// ==================== buildSimplePrompt 测试 ====================

func TestBuildSimplePrompt(t *testing.T) {
	info := &types.StructInfo{
		Name:       "TestStruct",
		Package:    "test",
		SourceCode: "type TestStruct struct {}",
		Methods:    []types.MethodInfo{},
	}

	prompt := buildSimplePrompt(info)

	if !strings.Contains(prompt, "TestStruct") {
		t.Error("Simple prompt should contain struct name")
	}
	if !strings.Contains(prompt, "test") {
		t.Error("Simple prompt should contain package name")
	}
	if !strings.Contains(prompt, "struct_description") {
		t.Error("Simple prompt should contain output format hint")
	}
}

func TestBuildSimplePrompt_WithMethods(t *testing.T) {
	info := &types.StructInfo{
		Name:       "TestStruct",
		Package:    "test",
		SourceCode: "type TestStruct struct {}",
		Methods: []types.MethodInfo{
			{Name: "Do", SourceCode: "func (t *TestStruct) Do() {}"},
		},
	}

	prompt := buildSimplePrompt(info)

	if !strings.Contains(prompt, "方法:") {
		t.Error("Simple prompt with methods should contain method section")
	}
	if !strings.Contains(prompt, "func (t *TestStruct) Do()") {
		t.Error("Simple prompt should contain method source code")
	}
}

// ==================== LLMClient 接口测试 ====================

func TestNewLLMClient_GLM(t *testing.T) {
	client := NewLLMClient("glm", "test-api-key")
	if client == nil {
		t.Fatal("NewLLMClient returned nil for glm")
	}
	if client.Name() != "GLM" {
		t.Errorf("GLM client name = %q, want %q", client.Name(), "GLM")
	}
}

func TestNewLLMClient_Claude(t *testing.T) {
	client := NewLLMClient("claude", "test-api-key")
	if client == nil {
		t.Fatal("NewLLMClient returned nil for claude")
	}
	if client.Name() != "Claude" {
		t.Errorf("Claude client name = %q, want %q", client.Name(), "Claude")
	}
}

func TestNewLLMClient_Aliases(t *testing.T) {
	tests := []struct {
		provider     string
		expectedName string
	}{
		{"glm", "GLM"},
		{"zhipu", "GLM"},
		{"claude", "Claude"},
		{"anthropic", "Claude"},
		{"unknown", "Claude"}, // 默认使用 Claude
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			client := NewLLMClient(tt.provider, "test-key")
			if client.Name() != tt.expectedName {
				t.Errorf("NewLLMClient(%q).Name() = %q, want %q", tt.provider, client.Name(), tt.expectedName)
			}
		})
	}
}

// ==================== PromptData 结构测试 ====================

func TestPromptData_Fields(t *testing.T) {
	data := PromptData{
		StructName:  "Test",
		Package:     "pkg",
		StructCode:  "type Test struct {}",
		MethodsCode: "// methods",
	}

	if data.StructName != "Test" {
		t.Error("StructName mismatch")
	}
	if data.Package != "pkg" {
		t.Error("Package mismatch")
	}
	if data.StructCode != "type Test struct {}" {
		t.Error("StructCode mismatch")
	}
	if data.MethodsCode != "// methods" {
		t.Error("MethodsCode mismatch")
	}
}
