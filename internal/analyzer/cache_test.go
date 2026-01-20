package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/go-struct-analyzer/internal/types"
)

func TestAnalysisCache_SetGet(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewAnalysisCache(tmpDir)

	// 创建测试数据
	structName := "TestStruct"
	sourceCode := "type TestStruct struct { Name string }"
	llmProvider := "test-provider"
	llmResult := &types.LLMAnalysisResult{
		StructDescription: "测试结构体描述",
		Fields: []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}{
			{Name: "Name", Description: "名称字段"},
		},
	}

	// 测试缓存未命中
	result := cache.Get(structName, sourceCode, llmProvider)
	if result != nil {
		t.Error("Expected cache miss, got hit")
	}

	// 设置缓存
	cache.Set(structName, sourceCode, llmProvider, llmResult)

	// 测试缓存命中
	result = cache.Get(structName, sourceCode, llmProvider)
	if result == nil {
		t.Fatal("Expected cache hit, got miss")
	}
	if result.StructDescription != llmResult.StructDescription {
		t.Errorf("Description mismatch: got %q, want %q", result.StructDescription, llmResult.StructDescription)
	}

	// 测试 Size
	if cache.Size() != 1 {
		t.Errorf("Size mismatch: got %d, want 1", cache.Size())
	}
}

func TestAnalysisCache_SourceCodeChange(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewAnalysisCache(tmpDir)

	structName := "TestStruct"
	originalSource := "type TestStruct struct { Name string }"
	modifiedSource := "type TestStruct struct { Name string; Age int }"
	llmProvider := "test-provider"
	llmResult := &types.LLMAnalysisResult{
		StructDescription: "测试结构体描述",
	}

	// 设置缓存
	cache.Set(structName, originalSource, llmProvider, llmResult)

	// 使用原始源码应该命中
	result := cache.Get(structName, originalSource, llmProvider)
	if result == nil {
		t.Error("Expected cache hit with original source")
	}

	// 使用修改后的源码应该不命中
	result = cache.Get(structName, modifiedSource, llmProvider)
	if result != nil {
		t.Error("Expected cache miss with modified source")
	}
}

func TestAnalysisCache_LLMProviderChange(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewAnalysisCache(tmpDir)

	structName := "TestStruct"
	sourceCode := "type TestStruct struct { Name string }"
	provider1 := "provider1"
	provider2 := "provider2"
	llmResult := &types.LLMAnalysisResult{
		StructDescription: "测试结构体描述",
	}

	// 使用 provider1 设置缓存
	cache.Set(structName, sourceCode, provider1, llmResult)

	// 使用 provider1 应该命中
	result := cache.Get(structName, sourceCode, provider1)
	if result == nil {
		t.Error("Expected cache hit with same provider")
	}

	// 使用 provider2 应该不命中
	result = cache.Get(structName, sourceCode, provider2)
	if result != nil {
		t.Error("Expected cache miss with different provider")
	}
}

func TestAnalysisCache_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建并填充缓存
	cache1 := NewAnalysisCache(tmpDir)
	cache1.Set("Struct1", "source1", "provider", &types.LLMAnalysisResult{
		StructDescription: "描述1",
	})
	cache1.Set("Struct2", "source2", "provider", &types.LLMAnalysisResult{
		StructDescription: "描述2",
	})

	// 保存缓存
	if err := cache1.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 验证缓存文件存在
	cachePath := filepath.Join(tmpDir, CacheFileName)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Cache file not created")
	}

	// 创建新缓存实例并加载
	cache2 := NewAnalysisCache(tmpDir)

	// 验证加载的数据
	if cache2.Size() != 2 {
		t.Errorf("Loaded cache size: got %d, want 2", cache2.Size())
	}

	result := cache2.Get("Struct1", "source1", "provider")
	if result == nil {
		t.Error("Expected to find Struct1 in loaded cache")
	} else if result.StructDescription != "描述1" {
		t.Errorf("Description mismatch: got %q, want %q", result.StructDescription, "描述1")
	}
}

func TestAnalysisCache_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewAnalysisCache(tmpDir)

	cache.Set("Struct1", "source1", "provider", &types.LLMAnalysisResult{})
	cache.Set("Struct2", "source2", "provider", &types.LLMAnalysisResult{})

	if cache.Size() != 2 {
		t.Errorf("Size before clear: got %d, want 2", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Size after clear: got %d, want 0", cache.Size())
	}
}

func TestHashSource(t *testing.T) {
	// 相同内容应该产生相同哈希
	hash1 := hashSource("test content")
	hash2 := hashSource("test content")
	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	// 不同内容应该产生不同哈希
	hash3 := hashSource("different content")
	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}

	// 哈希长度应该是 16（8字节的十六进制表示）
	if len(hash1) != 16 {
		t.Errorf("Hash length: got %d, want 16", len(hash1))
	}
}
