package reporter

import (
	"fmt"
	"os"
	"strings"

	"github.com/user/go-struct-analyzer/internal/types"
)

// MermaidGenerator 生成 Mermaid 流程图
type MermaidGenerator struct {
	builder strings.Builder
}

// NewMermaidGenerator 创建 Mermaid 生成器
func NewMermaidGenerator() *MermaidGenerator {
	return &MermaidGenerator{}
}

// Generate 生成 Mermaid 流程图代码
func (m *MermaidGenerator) Generate(result *types.AnalysisResult) string {
	m.builder.Reset()
	m.builder.WriteString("graph TD\n")

	// 收集所有节点
	nodes := make(map[string]types.StructAnalysis)
	for _, s := range result.Structs {
		nodes[s.Name] = s
	}

	// 生成节点定义
	for _, s := range result.Structs {
		nodeID := sanitizeID(s.Name)
		label := fmt.Sprintf("%s<br/>%s", s.Name, truncate(s.Description, 15))
		m.builder.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", nodeID, label))
	}

	m.builder.WriteString("\n")

	// 生成边
	edgeSet := make(map[string]bool) // 用于去重
	for _, s := range result.Structs {
		for _, dep := range s.Dependencies {
			fromID := sanitizeID(s.Name)
			toID := sanitizeID(dep.To)
			edgeKey := fromID + "->" + toID

			// 避免重复边和自环
			if edgeSet[edgeKey] || fromID == toID {
				continue
			}
			edgeSet[edgeKey] = true

			edgeLabel := m.getEdgeLabel(dep.Type)
			m.builder.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", fromID, edgeLabel, toID))
		}
	}

	m.builder.WriteString("\n")

	// 添加样式 - 按深度着色
	m.addStyles(result)

	return m.builder.String()
}

// GenerateToFile 生成并保存到文件
func (m *MermaidGenerator) GenerateToFile(result *types.AnalysisResult, filePath string) error {
	content := m.Generate(result)
	return os.WriteFile(filePath, []byte(content), 0644)
}

// getEdgeLabel 获取边的标签
func (m *MermaidGenerator) getEdgeLabel(depType string) string {
	switch depType {
	case types.DepTypeField:
		return "字段"
	case types.DepTypeInit:
		return "初始化"
	case types.DepTypeMethodCall:
		return "调用"
	case types.DepTypeInterface:
		return "实现"
	case types.DepTypeEmbed:
		return "嵌入"
	case types.DepTypeConstructor:
		return "构造"
	default:
		return "依赖"
	}
}

// addStyles 添加节点样式
func (m *MermaidGenerator) addStyles(result *types.AnalysisResult) {
	// 根据深度分配颜色
	colors := []string{
		"#ff9999", // 深度 0 - 红色
		"#99ccff", // 深度 1 - 蓝色
		"#99ff99", // 深度 2 - 绿色
		"#ffcc99", // 深度 3 - 橙色
		"#cc99ff", // 深度 4 - 紫色
		"#ffff99", // 深度 5 - 黄色
	}

	for _, s := range result.Structs {
		nodeID := sanitizeID(s.Name)
		colorIdx := s.Depth
		if colorIdx >= len(colors) {
			colorIdx = len(colors) - 1
		}
		m.builder.WriteString(fmt.Sprintf("    style %s fill:%s\n", nodeID, colors[colorIdx]))
	}
}

// sanitizeID 清理节点 ID，移除特殊字符
func sanitizeID(name string) string {
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "*", "ptr_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
