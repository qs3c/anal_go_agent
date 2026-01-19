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

// ClaudeClient 是 Claude API 客户端
type ClaudeClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
	maxRetries int
}

// NewClaudeClient 创建 Claude 客户端
func NewClaudeClient(apiKey string) *ClaudeClient {
	return &ClaudeClient{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model:      "claude-sonnet-4-20250514",
		maxRetries: 3,
	}
}

// ClaudeAPIRequest 表示 Claude API 请求体
type ClaudeAPIRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// Message 表示消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeAPIResponse 表示 Claude API 响应
type ClaudeAPIResponse struct {
	Content []ContentBlock `json:"content"`
	Error   *APIError      `json:"error,omitempty"`
}

// ContentBlock 表示内容块
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// APIError 表示 API 错误
type APIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// AnalyzeStruct 分析结构体并返回描述
func (c *ClaudeClient) AnalyzeStruct(info *types.StructInfo) (*types.LLMAnalysisResult, error) {
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

// callAPI 调用 Claude API
func (c *ClaudeClient) callAPI(prompt string) (string, error) {
	request := ClaudeAPIRequest{
		Model:     c.model,
		MaxTokens: 2000,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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

	var apiResp ClaudeAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response content")
	}

	return apiResp.Content[0].Text, nil
}

// IsConfigured 检查客户端是否已配置 API Key
func (c *ClaudeClient) IsConfigured() bool {
	return c.apiKey != ""
}

// Name 返回提供商名称
func (c *ClaudeClient) Name() string {
	return "Claude"
}
