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

	// Model 返回当前使用的模型
	Model() string
}

// NewLLMClient 根据提供商类型创建 LLM 客户端（使用默认模型）
func NewLLMClient(provider, apiKey string) LLMClient {
	return NewLLMClientWithModel(provider, apiKey, "")
}

// NewLLMClientWithModel 根据提供商类型创建指定模型的 LLM 客户端
func NewLLMClientWithModel(provider, apiKey, model string) LLMClient {
	switch provider {
	case "glm", "zhipu":
		return NewGLMClientWithModel(apiKey, model)
	case "claude", "anthropic":
		return NewClaudeClientWithModel(apiKey, model)
	default:
		// 默认使用 GLM
		return NewGLMClientWithModel(apiKey, model)
	}
}

// GetDefaultModel 获取指定提供商的默认模型
func GetDefaultModel(provider string) string {
	switch provider {
	case "glm", "zhipu":
		return DefaultGLMModel
	case "claude", "anthropic":
		return DefaultClaudeModel
	default:
		return DefaultGLMModel
	}
}
