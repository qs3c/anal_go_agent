package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewBlacklist(t *testing.T) {
	bl := NewBlacklist()

	if bl == nil {
		t.Fatal("NewBlacklist() returned nil")
	}

	if bl.types == nil {
		t.Error("Blacklist types map is nil")
	}

	if bl.packages == nil {
		t.Error("Blacklist packages map is nil")
	}
}

func TestBlacklist_AddType(t *testing.T) {
	bl := NewBlacklist()

	bl.AddType("Logger")
	bl.AddType("Tracer")

	if !bl.types["Logger"] {
		t.Error("Logger should be in types blacklist")
	}

	if !bl.types["Tracer"] {
		t.Error("Tracer should be in types blacklist")
	}
}

func TestBlacklist_AddPackage(t *testing.T) {
	bl := NewBlacklist()

	bl.AddPackage("thirdparty")
	bl.AddPackage("vendor")

	if !bl.packages["thirdparty"] {
		t.Error("thirdparty should be in packages blacklist")
	}

	if !bl.packages["vendor"] {
		t.Error("vendor should be in packages blacklist")
	}
}

func TestBlacklist_IsBlocked_Type(t *testing.T) {
	bl := NewBlacklist()
	bl.AddType("Logger")

	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		{"exact match", "Logger", true},
		{"pointer type", "*Logger", true},
		{"slice type", "[]Logger", true},
		{"different type", "User", false},
		{"similar name", "LoggerFactory", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bl.IsBlocked(tt.typeName)
			if result != tt.expected {
				t.Errorf("IsBlocked(%q) = %v, want %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestBlacklist_IsBlocked_Package(t *testing.T) {
	bl := NewBlacklist()
	bl.AddPackage("thirdparty")

	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		{"package qualified", "thirdparty.Client", true},
		{"pointer package qualified", "*thirdparty.Client", true},
		{"different package", "internal.Client", false},
		{"no package", "Client", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bl.IsBlocked(tt.typeName)
			if result != tt.expected {
				t.Errorf("IsBlocked(%q) = %v, want %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestBlacklist_IsBlocked_TypeWithPackagePrefix(t *testing.T) {
	bl := NewBlacklist()
	bl.AddType("Logger")

	// 即使带包前缀，类型名匹配也应该被阻止
	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		{"type with package", "log.Logger", true},
		{"type with different package", "mylog.Logger", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bl.IsBlocked(tt.typeName)
			if result != tt.expected {
				t.Errorf("IsBlocked(%q) = %v, want %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestBlacklist_GetBlockedTypes(t *testing.T) {
	bl := NewBlacklist()
	bl.AddType("Logger")
	bl.AddType("Tracer")

	types := bl.GetBlockedTypes()

	if len(types) != 2 {
		t.Errorf("GetBlockedTypes() returned %d types, want 2", len(types))
	}

	// 检查类型是否在列表中
	typeSet := make(map[string]bool)
	for _, typ := range types {
		typeSet[typ] = true
	}

	if !typeSet["Logger"] {
		t.Error("Logger should be in blocked types")
	}

	if !typeSet["Tracer"] {
		t.Error("Tracer should be in blocked types")
	}
}

func TestBlacklist_GetBlockedPackages(t *testing.T) {
	bl := NewBlacklist()
	bl.AddPackage("thirdparty")
	bl.AddPackage("vendor")

	packages := bl.GetBlockedPackages()

	if len(packages) != 2 {
		t.Errorf("GetBlockedPackages() returned %d packages, want 2", len(packages))
	}

	// 检查包是否在列表中
	pkgSet := make(map[string]bool)
	for _, pkg := range packages {
		pkgSet[pkg] = true
	}

	if !pkgSet["thirdparty"] {
		t.Error("thirdparty should be in blocked packages")
	}

	if !pkgSet["vendor"] {
		t.Error("vendor should be in blocked packages")
	}
}

func TestBlacklist_LoadFromFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "blacklist.yaml")

	configContent := `types:
  - Logger
  - Tracer
packages:
  - thirdparty
  - vendor
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	bl := NewBlacklist()
	if err := bl.LoadFromFile(configPath); err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	// 验证加载的配置
	if !bl.IsBlocked("Logger") {
		t.Error("Logger should be blocked after loading config")
	}

	if !bl.IsBlocked("Tracer") {
		t.Error("Tracer should be blocked after loading config")
	}

	if !bl.IsBlocked("thirdparty.Client") {
		t.Error("thirdparty.Client should be blocked after loading config")
	}

	if !bl.IsBlocked("vendor.Library") {
		t.Error("vendor.Library should be blocked after loading config")
	}
}

func TestBlacklist_LoadFromFile_EmptyPath(t *testing.T) {
	bl := NewBlacklist()

	// 空路径应该不报错
	if err := bl.LoadFromFile(""); err != nil {
		t.Errorf("LoadFromFile(\"\") should not return error, got %v", err)
	}
}

func TestBlacklist_LoadFromFile_NotExist(t *testing.T) {
	bl := NewBlacklist()

	// 不存在的文件应该报错
	err := bl.LoadFromFile("/nonexistent/path/blacklist.yaml")
	if err == nil {
		t.Error("LoadFromFile() should return error for non-existent file")
	}
}

func TestBlacklist_LoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// 写入无效的 YAML
	invalidContent := `types: [
  - Logger
  invalid yaml here
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	bl := NewBlacklist()
	err := bl.LoadFromFile(configPath)
	if err == nil {
		t.Error("LoadFromFile() should return error for invalid YAML")
	}
}
