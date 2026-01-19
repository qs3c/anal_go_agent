package reporter

import (
	"encoding/json"
	"os"

	"github.com/user/go-struct-analyzer/internal/types"
)

// JSONReporter 生成 JSON 格式的报告
type JSONReporter struct{}

// NewJSONReporter 创建 JSON 报告生成器
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{}
}

// Generate 生成 JSON 报告
func (r *JSONReporter) Generate(result *types.AnalysisResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveToFile 保存报告到文件
func (r *JSONReporter) SaveToFile(result *types.AnalysisResult, filePath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
