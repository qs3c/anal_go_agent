package reporter

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/user/go-struct-analyzer/internal/types"
)

// MarkdownReporter 生成 Markdown 格式的报告
type MarkdownReporter struct {
	builder strings.Builder
}

// NewMarkdownReporter 创建 Markdown 报告生成器
func NewMarkdownReporter() *MarkdownReporter {
	return &MarkdownReporter{}
}

// Generate 生成 Markdown 报告
func (r *MarkdownReporter) Generate(result *types.AnalysisResult, blacklist []string) string {
	r.builder.Reset()

	r.writeHeader(result)
	r.writeOverview(result, blacklist)
	r.writeStructsByDepth(result)
	r.writeDependencyGraph(result)
	r.writeStatistics(result, blacklist)
	r.writeFooter(result)

	return r.builder.String()
}

// SaveToFile 保存报告到文件
func (r *MarkdownReporter) SaveToFile(content string, filePath string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// writeHeader 写入报告头部
func (r *MarkdownReporter) writeHeader(result *types.AnalysisResult) {
	r.builder.WriteString("# Go 项目结构体依赖分析报告\n\n")
	r.builder.WriteString(fmt.Sprintf("**项目路径**: %s\n", result.ProjectPath))
	r.builder.WriteString(fmt.Sprintf("**分析起点**: %s\n", result.StartStruct))
	r.builder.WriteString(fmt.Sprintf("**分析深度**: %d\n", result.MaxDepth))
	r.builder.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", result.GeneratedAt))
	r.builder.WriteString("---\n\n")
}

// writeOverview 写入概览部分
func (r *MarkdownReporter) writeOverview(result *types.AnalysisResult, blacklist []string) {
	r.builder.WriteString("## 分析概览\n\n")

	// 按深度统计
	depthCount := make(map[int]int)
	for _, s := range result.Structs {
		depthCount[s.Depth]++
	}

	r.builder.WriteString(fmt.Sprintf("- **总结构体数**: %d\n", result.TotalStructs))
	r.builder.WriteString("- **分析深度分布**:\n")

	// 按深度排序
	var depths []int
	for d := range depthCount {
		depths = append(depths, d)
	}
	sort.Ints(depths)

	for _, d := range depths {
		r.builder.WriteString(fmt.Sprintf("  - 深度 %d: %d 个\n", d, depthCount[d]))
	}

	r.builder.WriteString(fmt.Sprintf("- **总依赖关系数**: %d\n", result.TotalDeps))
	r.builder.WriteString(fmt.Sprintf("- **循环依赖**: %d 个\n", len(result.Cycles)))

	if len(blacklist) > 0 {
		r.builder.WriteString(fmt.Sprintf("- **忽略类型**: %s\n", strings.Join(blacklist, ", ")))
	}

	r.builder.WriteString("\n---\n\n")
}

// writeStructsByDepth 按深度写入结构体信息
func (r *MarkdownReporter) writeStructsByDepth(result *types.AnalysisResult) {
	// 按深度分组
	structsByDepth := make(map[int][]types.StructAnalysis)
	for _, s := range result.Structs {
		structsByDepth[s.Depth] = append(structsByDepth[s.Depth], s)
	}

	// 按深度排序
	var depths []int
	for d := range structsByDepth {
		depths = append(depths, d)
	}
	sort.Ints(depths)

	for _, depth := range depths {
		r.builder.WriteString(fmt.Sprintf("## 深度 %d\n\n", depth))

		for _, s := range structsByDepth[depth] {
			r.writeStructDetail(s)
		}
	}
}

// writeStructDetail 写入结构体详情
func (r *MarkdownReporter) writeStructDetail(s types.StructAnalysis) {
	r.builder.WriteString(fmt.Sprintf("### %s\n\n", s.Name))
	r.builder.WriteString(fmt.Sprintf("**功能**: %s\n\n", s.Description))
	r.builder.WriteString(fmt.Sprintf("**所属包**: `%s`\n\n", s.Package))

	// 字段列表
	if len(s.Fields) > 0 {
		r.builder.WriteString("#### 字段列表\n\n")
		r.builder.WriteString("| 字段名 | 类型 | 导出 | 描述 |\n")
		r.builder.WriteString("|--------|------|------|------|\n")

		for _, field := range s.Fields {
			exported := "✗"
			if field.IsExported {
				exported = "✓"
			}
			fieldName := field.Name
			if field.IsEmbedded {
				fieldName = fmt.Sprintf("*%s* (嵌入)", field.Name)
			}
			r.builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				fieldName, escapeMarkdown(field.Type), exported, field.Description))
		}
		r.builder.WriteString("\n")
	}

	// 方法列表
	if len(s.Methods) > 0 {
		r.builder.WriteString("#### 方法列表\n\n")
		r.builder.WriteString("| 方法名 | 签名 | 导出 | 描述 |\n")
		r.builder.WriteString("|--------|------|------|------|\n")

		for _, method := range s.Methods {
			exported := "✗"
			if method.IsExported {
				exported = "✓"
			}
			r.builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				method.Name, escapeMarkdown(method.Signature), exported, method.Description))
		}
		r.builder.WriteString("\n")
	}

	// 依赖关系
	if len(s.Dependencies) > 0 {
		r.builder.WriteString("#### 依赖关系\n\n")
		r.builder.WriteString("| 目标结构体 | 依赖类型 | 上下文 | 深度 |\n")
		r.builder.WriteString("|-----------|---------|--------|------|\n")

		for _, dep := range s.Dependencies {
			depTypeLabel := getDepTypeLabel(dep.Type)
			r.builder.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n",
				dep.To, depTypeLabel, dep.Context, dep.Depth))
		}
		r.builder.WriteString("\n")
	}

	r.builder.WriteString("---\n\n")
}

// writeDependencyGraph 写入依赖关系图
func (r *MarkdownReporter) writeDependencyGraph(result *types.AnalysisResult) {
	r.builder.WriteString("## 依赖关系图\n\n")
	r.builder.WriteString("```mermaid\n")

	mermaid := NewMermaidGenerator()
	r.builder.WriteString(mermaid.Generate(result))

	r.builder.WriteString("```\n\n")
	r.builder.WriteString("---\n\n")
}

// writeStatistics 写入统计信息
func (r *MarkdownReporter) writeStatistics(result *types.AnalysisResult, blacklist []string) {
	r.builder.WriteString("## 统计信息\n\n")

	// 深度分布
	r.builder.WriteString("### 依赖深度分布\n")
	depthCount := make(map[int]int)
	for _, s := range result.Structs {
		depthCount[s.Depth]++
	}

	var depths []int
	for d := range depthCount {
		depths = append(depths, d)
	}
	sort.Ints(depths)

	for _, d := range depths {
		r.builder.WriteString(fmt.Sprintf("- 深度 %d: %d 个结构体\n", d, depthCount[d]))
	}
	r.builder.WriteString("\n")

	// 被依赖次数排行
	r.builder.WriteString("### 被依赖次数排行\n")
	depCount := make(map[string]int)
	for _, s := range result.Structs {
		for _, dep := range s.Dependencies {
			depCount[dep.To]++
		}
	}

	// 排序
	type depItem struct {
		name  string
		count int
	}
	var items []depItem
	for name, count := range depCount {
		items = append(items, depItem{name, count})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].count > items[j].count
	})

	for i, item := range items {
		if i >= 10 { // 只显示前 10
			break
		}
		r.builder.WriteString(fmt.Sprintf("%d. %s - 被依赖 %d 次\n", i+1, item.name, item.count))
	}
	r.builder.WriteString("\n")

	// 黑名单类型
	if len(blacklist) > 0 {
		r.builder.WriteString("### 黑名单类型\n")
		for _, b := range blacklist {
			r.builder.WriteString(fmt.Sprintf("- %s (已忽略)\n", b))
		}
		r.builder.WriteString("\n")
	}

	// 循环依赖
	if len(result.Cycles) > 0 {
		r.builder.WriteString("### 循环依赖\n")
		for i, cycle := range result.Cycles {
			r.builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.Join(cycle, " -> ")))
		}
		r.builder.WriteString("\n")
	}

	r.builder.WriteString("---\n\n")
}

// writeFooter 写入页脚
func (r *MarkdownReporter) writeFooter(result *types.AnalysisResult) {
	r.builder.WriteString(fmt.Sprintf("生成于: %s\n", result.GeneratedAt))
}

// getDepTypeLabel 获取依赖类型的中文标签
func getDepTypeLabel(depType string) string {
	switch depType {
	case types.DepTypeField:
		return "字段依赖"
	case types.DepTypeInit:
		return "方法内初始化"
	case types.DepTypeMethodCall:
		return "方法调用"
	case types.DepTypeInterface:
		return "接口实现"
	case types.DepTypeEmbed:
		return "结构体嵌入"
	default:
		return "依赖"
	}
}

// escapeMarkdown 转义 Markdown 特殊字符
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "*", "\\*")
	return s
}
