package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/user/go-struct-analyzer/internal/types"
)

// GLMClient 是智谱 GLM API 客户端
type GLMClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
	maxRetries int
}

// DefaultGLMModel 默认 GLM 模型
const DefaultGLMModel = "glm-4-flash"

// NewGLMClient 创建 GLM 客户端
func NewGLMClient(apiKey string) *GLMClient {
	return NewGLMClientWithModel(apiKey, "")
}

// NewGLMClientWithModel 创建指定模型的 GLM 客户端
func NewGLMClientWithModel(apiKey, model string) *GLMClient {
	if model == "" {
		model = DefaultGLMModel
	}
	return &GLMClient{
		apiKey:  apiKey,
		baseURL: "https://open.bigmodel.cn/api/paas/v4",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model:      model,
		maxRetries: 3,
	}
}

// GLMAPIRequest 表示 GLM API 请求体
type GLMAPIRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// GLMAPIResponse 表示 GLM API 响应
type GLMAPIResponse struct {
	Choices []GLMChoice `json:"choices"`
	Error   *GLMError   `json:"error,omitempty"`
}

// GLMChoice 表示 GLM 响应选项
type GLMChoice struct {
	Message GLMMessage `json:"message"`
}

// GLMMessage 表示 GLM 消息
type GLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GLMError 表示 GLM API 错误
type GLMError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// AnalyzeStruct 分析结构体并返回描述
func (c *GLMClient) AnalyzeStruct(info *types.StructInfo) (*types.LLMAnalysisResult, error) {
	prompt := buildPrompt(info)

	var lastErr error
	for i := 0; i < c.maxRetries; i++ {
		response, err := c.callAPI(prompt)
		if err == nil {
			return parseResponse(response)
		}

		lastErr = err
		// 指数退避
		sleepDuration := time.Duration(math.Pow(2, float64(i))) * time.Second
		time.Sleep(sleepDuration)
	}

	// 降级处理：返回空描述
	return &types.LLMAnalysisResult{
		StructDescription: "分析失败：" + lastErr.Error(),
	}, nil
}

// callAPI 调用 GLM API
func (c *GLMClient) callAPI(prompt string) (string, error) {
	request := GLMAPIRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 2000,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp GLMAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error [%s]: %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response choices")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// IsConfigured 检查客户端是否已配置 API Key
func (c *GLMClient) IsConfigured() bool {
	return c.apiKey != ""
}

// Name 返回提供商名称
func (c *GLMClient) Name() string {
	return "GLM"
}

// Model 返回当前使用的模型
func (c *GLMClient) Model() string {
	return c.model
}
