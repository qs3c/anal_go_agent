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

// Client 是 Claude API 客户端
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
	maxRetries int
}

// NewClient 创建新的 LLM 客户端
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model:      "claude-sonnet-4-20250514",
		maxRetries: 3,
	}
}

// APIRequest 表示 API 请求体
type APIRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// Message 表示消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// APIResponse 表示 API 响应
type APIResponse struct {
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
func (c *Client) AnalyzeStruct(info *types.StructInfo) (*types.LLMAnalysisResult, error) {
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
func (c *Client) callAPI(prompt string) (string, error) {
	request := APIRequest{
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

	var apiResp APIResponse
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
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}
