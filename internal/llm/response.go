package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/user/go-struct-analyzer/internal/types"
)

// parseResponse 解析 LLM 响应
func parseResponse(response string) (*types.LLMAnalysisResult, error) {
	// 清理可能的 Markdown 标记
	response = cleanResponse(response)

	var result types.LLMAnalysisResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w\nResponse: %s", err, response)
	}

	return &result, nil
}

// cleanResponse 清理响应中的 Markdown 标记
func cleanResponse(response string) string {
	response = strings.TrimSpace(response)

	// 移除开头的 ```json 或 ```
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}

	// 移除结尾的 ```
	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
	}

	// 再次清理空白
	response = strings.TrimSpace(response)

	// 尝试找到 JSON 对象的开始和结束
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start != -1 && end != -1 && end > start {
		response = response[start : end+1]
	}

	return response
}
