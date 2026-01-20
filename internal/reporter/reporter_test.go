package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/go-struct-analyzer/internal/types"
)

// 创建测试用的 AnalysisResult
func createTestAnalysisResult() *types.AnalysisResult {
	return &types.AnalysisResult{
		ProjectPath:  "/test/project",
		StartStruct:  "UserService",
		MaxDepth:     2,
		TotalStructs: 3,
		TotalDeps:    4,
		GeneratedAt:  "2026-01-20 10:00:00",
		Cycles:       [][]string{},
		Blacklist:    []string{"Context", "Logger"},
		Structs: []types.StructAnalysis{
			{
				Name:        "UserService",
				Package:     "service",
				Description: "用户服务，处理用户业务逻辑",
				Depth:       0,
				Fields: []types.FieldAnalysis{
					{Name: "repo", Type: "*UserRepository", Description: "用户数据仓库", IsExported: false},
					{Name: "Cache", Type: "*Cache", Description: "缓存服务", IsExported: true},
				},
				Methods: []types.MethodAnalysis{
					{Name: "CreateUser", Signature: "func (s *UserService) CreateUser(name string) error", Description: "创建新用户", IsExported: true},
					{Name: "getUser", Signature: "func (s *UserService) getUser(id int) *User", Description: "内部获取用户", IsExported: false},
				},
				Dependencies: []types.Dependency{
					{From: "UserService", To: "UserRepository", Type: types.DepTypeField, Context: "repo", Depth: 1},
					{From: "UserService", To: "Cache", Type: types.DepTypeField, Context: "cache", Depth: 1},
				},
			},
			{
				Name:        "UserRepository",
				Package:     "repository",
				Description: "用户数据仓库",
				Depth:       1,
				Fields: []types.FieldAnalysis{
					{Name: "db", Type: "*Database", Description: "数据库连接", IsExported: false},
				},
				Methods: []types.MethodAnalysis{
					{Name: "Save", Signature: "func (r *UserRepository) Save(user *User) error", Description: "保存用户", IsExported: true},
				},
				Dependencies: []types.Dependency{
					{From: "UserRepository", To: "Database", Type: types.DepTypeField, Context: "db", Depth: 2},
				},
			},
			{
				Name:        "Cache",
				Package:     "cache",
				Description: "缓存服务",
				Depth:       1,
				Fields:      []types.FieldAnalysis{},
				Methods:     []types.MethodAnalysis{},
				Dependencies: []types.Dependency{},
			},
		},
	}
}

// ==================== MarkdownReporter 测试 ====================

func TestMarkdownReporter_Generate(t *testing.T) {
	reporter := NewMarkdownReporter()
	result := createTestAnalysisResult()
	blacklist := []string{"Context", "Logger"}

	content := reporter.Generate(result, blacklist)

	// 验证包含关键内容
	tests := []struct {
		name     string
		expected string
	}{
		{"header", "# Go 项目结构体依赖分析报告"},
		{"project path", "/test/project"},
		{"start struct", "UserService"},
		{"depth", "深度 0"},
		{"struct name", "### UserService"},
		{"description", "用户服务，处理用户业务逻辑"},
		{"field table header", "| 字段名 | 类型 | 导出 | 描述 |"},
		{"method table header", "| 方法名 | 签名 | 导出 | 描述 |"},
		{"dependency section", "#### 依赖关系"},
		{"mermaid graph", "```mermaid"},
		{"statistics", "## 统计信息"},
		{"blacklist", "Context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(content, tt.expected) {
				t.Errorf("Generated markdown should contain %q", tt.expected)
			}
		})
	}
}

func TestMarkdownReporter_SaveToFile(t *testing.T) {
	reporter := NewMarkdownReporter()
	content := "# Test Report\n\nThis is a test."

	// 使用临时目录
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_report.md")

	err := reporter.SaveToFile(content, filePath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// 读取并验证内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content mismatch: got %q, want %q", string(data), content)
	}
}

// ==================== JSONReporter 测试 ====================

func TestJSONReporter_Generate(t *testing.T) {
	reporter := NewJSONReporter()
	result := createTestAnalysisResult()

	jsonStr, err := reporter.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 验证是有效的 JSON
	var parsed types.AnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// 验证关键字段
	if parsed.ProjectPath != result.ProjectPath {
		t.Errorf("ProjectPath mismatch: got %q, want %q", parsed.ProjectPath, result.ProjectPath)
	}
	if parsed.StartStruct != result.StartStruct {
		t.Errorf("StartStruct mismatch: got %q, want %q", parsed.StartStruct, result.StartStruct)
	}
	if len(parsed.Structs) != len(result.Structs) {
		t.Errorf("Structs count mismatch: got %d, want %d", len(parsed.Structs), len(result.Structs))
	}
}

func TestJSONReporter_SaveToFile(t *testing.T) {
	reporter := NewJSONReporter()
	result := createTestAnalysisResult()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_result.json")

	err := reporter.SaveToFile(result, filePath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// 读取并验证
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	var parsed types.AnalysisResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Saved JSON is invalid: %v", err)
	}

	if parsed.StartStruct != result.StartStruct {
		t.Errorf("StartStruct mismatch: got %q, want %q", parsed.StartStruct, result.StartStruct)
	}
}

// ==================== MermaidGenerator 测试 ====================

func TestMermaidGenerator_Generate(t *testing.T) {
	generator := NewMermaidGenerator()
	result := createTestAnalysisResult()

	content := generator.Generate(result)

	// 验证基本结构
	tests := []struct {
		name     string
		expected string
	}{
		{"graph declaration", "graph TD"},
		{"node UserService", "UserService["},
		{"node UserRepository", "UserRepository["},
		{"edge with label", "-->|"},
		{"style declaration", "style UserService fill:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(content, tt.expected) {
				t.Errorf("Generated mermaid should contain %q\nGot:\n%s", tt.expected, content)
			}
		})
	}
}

func TestMermaidGenerator_GenerateToFile(t *testing.T) {
	generator := NewMermaidGenerator()
	result := createTestAnalysisResult()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_graph.mmd")

	err := generator.GenerateToFile(result, filePath)
	if err != nil {
		t.Fatalf("GenerateToFile failed: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if !strings.Contains(string(data), "graph TD") {
		t.Error("Saved file should contain 'graph TD'")
	}
}

// ==================== 辅助函数测试 ====================

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserService", "UserService"},
		{"model.User", "model_User"},
		{"*User", "ptr_User"},
		{"[]User", "__User"},
		{"User-Service", "User_Service"},
		{"User Service", "User_Service"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeID(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Hello", 10, "Hello"},
		{"Hello World", 5, "Hello..."},
		{"", 5, ""},
		{"你好世界", 2, "你好..."},
		{"Test", 4, "Test"},
		{"Test", 3, "Tes..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text|with|pipes", "text\\|with\\|pipes"},
		{"*bold*", "\\*bold\\*"},
		{"text|and*mix", "text\\|and\\*mix"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDepTypeLabel(t *testing.T) {
	tests := []struct {
		depType  string
		expected string
	}{
		{types.DepTypeField, "字段依赖"},
		{types.DepTypeInit, "方法内初始化"},
		{types.DepTypeMethodCall, "方法调用"},
		{types.DepTypeInterface, "接口实现"},
		{types.DepTypeEmbed, "结构体嵌入"},
		{types.DepTypeConstructor, "构造函数调用"},
		{"unknown", "依赖"},
	}

	for _, tt := range tests {
		t.Run(tt.depType, func(t *testing.T) {
			result := getDepTypeLabel(tt.depType)
			if result != tt.expected {
				t.Errorf("getDepTypeLabel(%q) = %q, want %q", tt.depType, result, tt.expected)
			}
		})
	}
}

// ==================== MermaidGenerator 边标签测试 ====================

func TestMermaidGenerator_getEdgeLabel(t *testing.T) {
	generator := NewMermaidGenerator()

	tests := []struct {
		depType  string
		expected string
	}{
		{types.DepTypeField, "字段"},
		{types.DepTypeInit, "初始化"},
		{types.DepTypeMethodCall, "调用"},
		{types.DepTypeInterface, "实现"},
		{types.DepTypeEmbed, "嵌入"},
		{types.DepTypeConstructor, "构造"},
		{"unknown", "依赖"},
	}

	for _, tt := range tests {
		t.Run(tt.depType, func(t *testing.T) {
			result := generator.getEdgeLabel(tt.depType)
			if result != tt.expected {
				t.Errorf("getEdgeLabel(%q) = %q, want %q", tt.depType, result, tt.expected)
			}
		})
	}
}

// ==================== 循环依赖测试 ====================

func TestMarkdownReporter_WithCycles(t *testing.T) {
	reporter := NewMarkdownReporter()
	result := createTestAnalysisResult()
	result.Cycles = [][]string{
		{"A", "B", "C", "A"},
		{"X", "Y", "X"},
	}

	content := reporter.Generate(result, nil)

	if !strings.Contains(content, "### 循环依赖") {
		t.Error("Should contain cycle section when cycles exist")
	}
	if !strings.Contains(content, "A -> B -> C -> A") {
		t.Error("Should contain cycle path")
	}
}

// ==================== 嵌入字段测试 ====================

func TestMarkdownReporter_EmbeddedField(t *testing.T) {
	reporter := NewMarkdownReporter()
	result := &types.AnalysisResult{
		ProjectPath:  "/test",
		StartStruct:  "Test",
		GeneratedAt:  "2026-01-20",
		TotalStructs: 1,
		Structs: []types.StructAnalysis{
			{
				Name:    "Test",
				Package: "test",
				Depth:   0,
				Fields: []types.FieldAnalysis{
					{Name: "BaseModel", Type: "BaseModel", IsEmbedded: true, Description: "基础模型"},
				},
			},
		},
	}

	content := reporter.Generate(result, nil)

	if !strings.Contains(content, "(嵌入)") {
		t.Error("Embedded field should be marked with (嵌入)")
	}
}
