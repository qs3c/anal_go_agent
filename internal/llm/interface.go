package llm

import "github.com/user/go-struct-analyzer/internal/types"

// LLMClient 定义 LLM 客户端接口
type LLMClient interface {
	// AnalyzeStruct 分析结构体并返回描述
	AnalyzeStruct(info *types.StructInfo) (*types.LLMAnalysisResult, error)

	// IsConfigured 检查客户端是否已配置
	IsConfigured() bool

	// Name 返回 LLM 提供商名称
	Name() string
}

// NewLLMClient 根据提供商类型创建 LLM 客户端
func NewLLMClient(provider, apiKey string) LLMClient {
	switch provider {
	case "glm", "zhipu":
		return NewGLMClient(apiKey)
	case "claude", "anthropic":
		return NewClaudeClient(apiKey)
	default:
		// 默认使用 Claude
		return NewClaudeClient(apiKey)
	}
}
