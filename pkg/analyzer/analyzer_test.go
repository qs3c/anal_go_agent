package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getTestProjectPath() string {
	// 从 pkg/analyzer 目录向上找到 testdata
	return "../../testdata/sample_project"
}

func TestNew(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: Options{
				ProjectPath: projectPath,
				StartStruct: "UserService",
			},
			wantErr: false,
		},
		{
			name: "missing project path",
			opts: Options{
				StartStruct: "UserService",
			},
			wantErr: true,
		},
		{
			name: "missing start struct",
			opts: Options{
				ProjectPath: projectPath,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyzer_Analyze(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	a, err := New(Options{
		ProjectPath: projectPath,
		StartStruct: "UserService",
		MaxDepth:    2,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	result, err := a.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	// 验证基本结果
	if result.TotalStructs == 0 {
		t.Error("Expected to find structs")
	}
	if result.StartStruct != "UserService" {
		t.Errorf("StartStruct = %q, want %q", result.StartStruct, "UserService")
	}
	if result.MaxDepth != 2 {
		t.Errorf("MaxDepth = %d, want %d", result.MaxDepth, 2)
	}
}

func TestAnalyzer_AnalyzeInvalidStartStruct(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	a, err := New(Options{
		ProjectPath: projectPath,
		StartStruct: "NonExistentStruct",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = a.Analyze()
	if err == nil {
		t.Error("Expected error for non-existent start struct")
	}
}

func TestAnalyzer_GenerateMarkdown(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	a, err := New(Options{
		ProjectPath: projectPath,
		StartStruct: "UserService",
		MaxDepth:    1,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if _, err := a.Analyze(); err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	md, err := a.GenerateMarkdown()
	if err != nil {
		t.Fatalf("GenerateMarkdown() failed: %v", err)
	}

	// 验证 Markdown 内容
	if !strings.Contains(md, "UserService") {
		t.Error("Markdown should contain UserService")
	}
	if !strings.Contains(md, "# Go 项目结构体依赖分析报告") {
		t.Error("Markdown should contain header")
	}
}

func TestAnalyzer_GenerateJSON(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	a, err := New(Options{
		ProjectPath: projectPath,
		StartStruct: "UserService",
		MaxDepth:    1,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if _, err := a.Analyze(); err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	jsonStr, err := a.GenerateJSON()
	if err != nil {
		t.Fatalf("GenerateJSON() failed: %v", err)
	}

	if !strings.Contains(jsonStr, "UserService") {
		t.Error("JSON should contain UserService")
	}
}

func TestAnalyzer_GenerateMermaid(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	a, err := New(Options{
		ProjectPath: projectPath,
		StartStruct: "UserService",
		MaxDepth:    1,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if _, err := a.Analyze(); err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	mermaid, err := a.GenerateMermaid()
	if err != nil {
		t.Fatalf("GenerateMermaid() failed: %v", err)
	}

	if !strings.Contains(mermaid, "graph TD") {
		t.Error("Mermaid should contain 'graph TD'")
	}
}

func TestAnalyzer_SaveToFile(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	a, err := New(Options{
		ProjectPath: projectPath,
		StartStruct: "UserService",
		MaxDepth:    1,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if _, err := a.Analyze(); err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	tmpDir := t.TempDir()

	// 测试保存 Markdown
	mdPath := filepath.Join(tmpDir, "report.md")
	if err := a.SaveMarkdown(mdPath); err != nil {
		t.Fatalf("SaveMarkdown() failed: %v", err)
	}
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Error("Markdown file not created")
	}

	// 测试保存 JSON
	jsonPath := filepath.Join(tmpDir, "report.json")
	if err := a.SaveJSON(jsonPath); err != nil {
		t.Fatalf("SaveJSON() failed: %v", err)
	}
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("JSON file not created")
	}

	// 测试保存 Mermaid
	mermaidPath := filepath.Join(tmpDir, "graph.mmd")
	if err := a.SaveMermaid(mermaidPath); err != nil {
		t.Fatalf("SaveMermaid() failed: %v", err)
	}
	if _, err := os.Stat(mermaidPath); os.IsNotExist(err) {
		t.Error("Mermaid file not created")
	}
}

func TestResult_GetStructByName(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	a, err := New(Options{
		ProjectPath: projectPath,
		StartStruct: "UserService",
		MaxDepth:    2,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	result, err := a.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	// 测试获取存在的结构体
	s := result.GetStructByName("UserService")
	if s == nil {
		t.Error("GetStructByName(UserService) returned nil")
	}

	// 测试获取不存在的结构体
	s = result.GetStructByName("NonExistent")
	if s != nil {
		t.Error("GetStructByName(NonExistent) should return nil")
	}
}

func TestResult_GetStructsByDepth(t *testing.T) {
	projectPath := getTestProjectPath()
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	a, err := New(Options{
		ProjectPath: projectPath,
		StartStruct: "UserService",
		MaxDepth:    2,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	result, err := a.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	// 深度 0 应该只有起点结构体
	depth0 := result.GetStructsByDepth(0)
	if len(depth0) != 1 {
		t.Errorf("GetStructsByDepth(0) = %d structs, want 1", len(depth0))
	}
	if len(depth0) > 0 && depth0[0].Name != "UserService" {
		t.Errorf("Depth 0 struct name = %q, want %q", depth0[0].Name, "UserService")
	}
}
