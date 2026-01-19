package analyzer

import (
	"testing"
)

func TestIsBuiltinType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		{"string", "string", true},
		{"int", "int", true},
		{"int64", "int64", true},
		{"bool", "bool", true},
		{"error", "error", true},
		{"any", "any", true},
		{"interface{}", "interface{}", true},
		{"custom type", "User", false},
		{"custom lower", "user", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBuiltinType(tt.typeName)
			if result != tt.expected {
				t.Errorf("isBuiltinType(%q) = %v, want %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestIsStandardLibrary(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		{"context type", "context.Context", true},
		{"time type", "time.Time", true},
		{"sync type", "sync.Mutex", true},
		{"http type", "http.Client", true},
		{"io type", "io.Reader", true},
		{"custom type", "model.User", false},
		{"simple name", "User", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStandardLibrary(tt.typeName)
			if result != tt.expected {
				t.Errorf("isStandardLibrary(%q) = %v, want %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestShouldAnalyze_EmptyTypeName(t *testing.T) {
	sf := &ScopeFilter{
		blacklist: NewBlacklist(),
	}

	// 空类型名应该返回 false
	if sf.ShouldAnalyze("") {
		t.Error("ShouldAnalyze(\"\") should return false")
	}
}

func TestShouldAnalyze_LowercaseTypeName(t *testing.T) {
	sf := &ScopeFilter{
		blacklist: NewBlacklist(),
	}

	// 小写开头的类型名应该返回 false（可能是变量名误识别）
	lowercaseNames := []string{"cache", "user", "db", "repo", "service"}

	for _, name := range lowercaseNames {
		if sf.ShouldAnalyze(name) {
			t.Errorf("ShouldAnalyze(%q) should return false for lowercase type name", name)
		}
	}
}

func TestShouldAnalyze_UppercaseValidation(t *testing.T) {
	// 测试首字母大写验证逻辑
	// 通过测试会被其他条件（如内置类型、标准库）阻止的大写类型
	// 来间接验证大写验证不会误阻止

	sf := &ScopeFilter{
		blacklist: NewBlacklist(),
	}

	// 测试大写开头的内置类型名（虽然 Go 没有，但测试逻辑）
	// 这些类型名会在 isBuiltinType 阶段被拒绝，而不是在首字母检查阶段
	// 所以我们只需验证小写类型名会被首字母检查阻止

	// 确认小写名字被拒绝
	if sf.ShouldAnalyze("cache") {
		t.Error("lowercase 'cache' should be rejected")
	}

	// 带包前缀的小写类型名也应该被拒绝
	if sf.ShouldAnalyze("model.user") {
		t.Error("lowercase 'model.user' should be rejected")
	}
}

func TestShouldAnalyze_BuiltinTypes(t *testing.T) {
	sf := &ScopeFilter{
		blacklist: NewBlacklist(),
	}

	builtinTypes := []string{"string", "int", "bool", "error", "any"}

	for _, typeName := range builtinTypes {
		if sf.ShouldAnalyze(typeName) {
			t.Errorf("ShouldAnalyze(%q) should return false for builtin type", typeName)
		}
	}
}

func TestShouldAnalyze_StandardLibraryTypes(t *testing.T) {
	sf := &ScopeFilter{
		blacklist: NewBlacklist(),
	}

	stdTypes := []string{"context.Context", "time.Time", "sync.Mutex", "http.Client"}

	for _, typeName := range stdTypes {
		if sf.ShouldAnalyze(typeName) {
			t.Errorf("ShouldAnalyze(%q) should return false for standard library type", typeName)
		}
	}
}

func TestShouldAnalyze_BlacklistedTypes(t *testing.T) {
	bl := NewBlacklist()
	bl.AddType("Logger")
	bl.AddPackage("thirdparty")

	sf := &ScopeFilter{
		blacklist: bl,
	}

	// 黑名单中的类型应该返回 false
	if sf.ShouldAnalyze("Logger") {
		t.Error("ShouldAnalyze(\"Logger\") should return false for blacklisted type")
	}

	// 黑名单包中的类型应该返回 false
	if sf.ShouldAnalyze("thirdparty.Client") {
		t.Error("ShouldAnalyze(\"thirdparty.Client\") should return false for blacklisted package")
	}
}

func TestShouldAnalyze_PointerAndSliceTypes(t *testing.T) {
	sf := &ScopeFilter{
		blacklist: NewBlacklist(),
	}

	// 指针和切片前缀应该被正确处理
	// 这些都是小写的，所以应该返回 false
	if sf.ShouldAnalyze("*cache") {
		t.Error("ShouldAnalyze(\"*cache\") should return false")
	}

	if sf.ShouldAnalyze("[]user") {
		t.Error("ShouldAnalyze(\"[]user\") should return false")
	}

	// 指针到标准库类型也应该返回 false
	if sf.ShouldAnalyze("*time.Time") {
		t.Error("ShouldAnalyze(\"*time.Time\") should return false for std lib type")
	}
}

func TestShouldAnalyze_MapTypes(t *testing.T) {
	sf := &ScopeFilter{
		blacklist: NewBlacklist(),
	}

	// map 类型应该返回 false
	if sf.ShouldAnalyze("map[string]int") {
		t.Error("ShouldAnalyze(\"map[string]int\") should return false")
	}

	if sf.ShouldAnalyze("map[int64]User") {
		t.Error("ShouldAnalyze(\"map[int64]User\") should return false")
	}
}

func TestShouldAnalyze_PackageQualifiedTypes(t *testing.T) {
	sf := &ScopeFilter{
		blacklist: NewBlacklist(),
	}

	// 带包前缀且小写类型名应该返回 false
	if sf.ShouldAnalyze("model.user") {
		t.Error("ShouldAnalyze(\"model.user\") should return false for lowercase type")
	}
}
