package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/user/go-struct-analyzer/internal/types"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	StructName  string                   `json:"struct_name"`
	SourceHash  string                   `json:"source_hash"`
	LLMResult   *types.LLMAnalysisResult `json:"llm_result"`
	CachedAt    time.Time                `json:"cached_at"`
	LLMProvider string                   `json:"llm_provider"`
}

// AnalysisCache 分析结果缓存
type AnalysisCache struct {
	Entries   map[string]*CacheEntry `json:"entries"` // key: structName
	Version   string                 `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
	mu        sync.RWMutex
	filePath  string
	dirty     bool // 是否有未保存的更改
}

const (
	CacheVersion  = "1.0"
	CacheFileName = ".struct-analyzer-cache.json"
)

// NewAnalysisCache 创建新的分析缓存
func NewAnalysisCache(projectPath string) *AnalysisCache {
	cache := &AnalysisCache{
		Entries:   make(map[string]*CacheEntry),
		Version:   CacheVersion,
		UpdatedAt: time.Now(),
		filePath:  filepath.Join(projectPath, CacheFileName),
	}

	// 尝试加载已有缓存
	cache.Load()

	return cache
}

// Load 从文件加载缓存
func (c *AnalysisCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 缓存文件不存在是正常的
		}
		return err
	}

	var loaded AnalysisCache
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	// 版本检查
	if loaded.Version != CacheVersion {
		// 版本不匹配，清空缓存
		c.Entries = make(map[string]*CacheEntry)
		return nil
	}

	c.Entries = loaded.Entries
	c.UpdatedAt = loaded.UpdatedAt
	return nil
}

// Save 保存缓存到文件
func (c *AnalysisCache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.dirty {
		return nil // 没有更改，不需要保存
	}

	c.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		return err
	}

	c.dirty = false
	return nil
}

// Get 获取缓存的 LLM 分析结果
func (c *AnalysisCache) Get(structName, sourceCode, llmProvider string) *types.LLMAnalysisResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.Entries[structName]
	if !ok {
		return nil
	}

	// 检查源码是否变化
	currentHash := hashSource(sourceCode)
	if entry.SourceHash != currentHash {
		return nil // 源码已变化，缓存失效
	}

	// 检查 LLM 提供商是否一致
	if entry.LLMProvider != llmProvider {
		return nil // LLM 提供商不同，缓存失效
	}

	return entry.LLMResult
}

// Set 设置 LLM 分析结果缓存
func (c *AnalysisCache) Set(structName, sourceCode, llmProvider string, result *types.LLMAnalysisResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Entries[structName] = &CacheEntry{
		StructName:  structName,
		SourceHash:  hashSource(sourceCode),
		LLMResult:   result,
		CachedAt:    time.Now(),
		LLMProvider: llmProvider,
	}
	c.dirty = true
}

// Clear 清空缓存
func (c *AnalysisCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Entries = make(map[string]*CacheEntry)
	c.dirty = true
}

// Size 返回缓存条目数
func (c *AnalysisCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Entries)
}

// hashSource 计算源码哈希
func hashSource(source string) string {
	hash := sha256.Sum256([]byte(source))
	return hex.EncodeToString(hash[:8]) // 使用前8字节（16字符）
}
