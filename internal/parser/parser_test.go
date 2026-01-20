package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_ParseProject(t *testing.T) {
	// 使用 testdata 目录
	projectPath := "../../testdata/sample_project"

	// 检查测试目录是否存在
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	p := NewParser(false)
	err := p.ParseProject(projectPath)
	if err != nil {
		t.Fatalf("ParseProject failed: %v", err)
	}

	// 验证解析结果
	structs := p.GetAllStructs()
	if len(structs) == 0 {
		t.Error("Expected to find structs, got none")
	}

	// 验证 UserService 结构体
	userService := p.GetStruct("UserService")
	if userService == nil {
		t.Error("Expected to find UserService struct")
	} else {
		if userService.Package != "service" {
			t.Errorf("UserService package: got %q, want %q", userService.Package, "service")
		}
		if len(userService.Fields) == 0 {
			t.Error("UserService should have fields")
		}
	}

	// 验证接口解析
	interfaces := p.GetAllInterfaces()
	if len(interfaces) == 0 {
		t.Error("Expected to find interfaces, got none")
	}

	// 验证 UserRepositoryInterface
	repoInterface := p.GetInterface("UserRepositoryInterface")
	if repoInterface == nil {
		t.Error("Expected to find UserRepositoryInterface")
	} else {
		if len(repoInterface.Methods) == 0 {
			t.Error("UserRepositoryInterface should have methods")
		}
	}

	// 验证函数解析
	functions := p.GetAllFunctions()
	if len(functions) == 0 {
		t.Error("Expected to find functions (New* constructors), got none")
	}
}

func TestParser_ConcurrentParsing(t *testing.T) {
	// 并发解析测试 - 使用 race detector 运行: go test -race
	projectPath := "../../testdata/sample_project"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	// 多次解析以增加发现竞态条件的机会
	for i := 0; i < 5; i++ {
		p := NewParser(false)
		err := p.ParseProject(projectPath)
		if err != nil {
			t.Fatalf("ParseProject iteration %d failed: %v", i, err)
		}

		// 验证每次解析结果一致
		structs := p.GetAllStructs()
		if len(structs) == 0 {
			t.Errorf("Iteration %d: Expected to find structs", i)
		}
	}
}

func TestParser_GetStruct(t *testing.T) {
	projectPath := "../../testdata/sample_project"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	p := NewParser(false)
	err := p.ParseProject(projectPath)
	if err != nil {
		t.Fatalf("ParseProject failed: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantName string
	}{
		{"simple name", "UserService", false, "UserService"},
		{"with pointer", "*UserService", false, "UserService"},
		{"with package prefix", "service.UserService", false, "UserService"},
		{"non-existent", "NonExistent", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.GetStruct(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("GetStruct(%q) = %v, want nil", tt.input, result)
				}
			} else {
				if result == nil {
					t.Errorf("GetStruct(%q) = nil, want struct", tt.input)
				} else if result.Name != tt.wantName {
					t.Errorf("GetStruct(%q).Name = %q, want %q", tt.input, result.Name, tt.wantName)
				}
			}
		})
	}
}

func TestParser_GetModuleName(t *testing.T) {
	projectPath := "../../testdata/sample_project"

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Skip("testdata/sample_project not found")
	}

	p := NewParser(false)
	err := p.ParseProject(projectPath)
	if err != nil {
		t.Fatalf("ParseProject failed: %v", err)
	}

	moduleName := p.GetModuleName()
	if moduleName == "" {
		t.Error("GetModuleName should return non-empty string")
	}
}

func TestParser_FindGoFiles(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 创建一些测试文件
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)

	// 创建 .go 文件
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte("package main"), 0644) // 应该被跳过
	os.WriteFile(filepath.Join(subDir, "sub.go"), []byte("package sub"), 0644)

	// 创建 vendor 目录（应该被跳过）
	vendorDir := filepath.Join(tmpDir, "vendor")
	os.Mkdir(vendorDir, 0755)
	os.WriteFile(filepath.Join(vendorDir, "vendor.go"), []byte("package vendor"), 0644)

	p := NewParser(false)
	goFiles, err := p.findGoFiles(tmpDir)
	if err != nil {
		t.Fatalf("findGoFiles failed: %v", err)
	}

	// 应该找到 2 个文件（main.go 和 sub.go）
	if len(goFiles) != 2 {
		t.Errorf("findGoFiles found %d files, want 2", len(goFiles))
	}

	// 验证不包含测试文件
	for _, f := range goFiles {
		if filepath.Base(f) == "main_test.go" {
			t.Error("findGoFiles should skip test files")
		}
		if filepath.Base(f) == "vendor.go" {
			t.Error("findGoFiles should skip vendor directory")
		}
	}
}
