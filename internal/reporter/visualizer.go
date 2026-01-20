package reporter

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/user/go-struct-analyzer/internal/types"
)

// VisualizerOutput 表示前端可视化工具需要的输出格式
type VisualizerOutput struct {
	ProjectPath string              `json:"projectPath"`
	StartStruct string              `json:"startStruct"`
	GeneratedAt string              `json:"generatedAt"`
	Structs     []VisualizerStruct  `json:"structs"`
	Connections []VisualizerConnect `json:"connections"`
}

// VisualizerStruct 表示单个结构体的可视化数据
type VisualizerStruct struct {
	ID       string              `json:"id"`
	X        float64             `json:"x"`
	Y        float64             `json:"y"`
	Metadata StructBoxMetadata   `json:"metadata"`
}

// StructBoxMetadata 对应前端 StructBoxMetadata 类型
type StructBoxMetadata struct {
	Type             string       `json:"type"`
	Name             string       `json:"name"`
	Description      string       `json:"description"`
	DescriptionTitle string       `json:"descriptionTitle,omitempty"`
	Fields           []FieldInfo  `json:"fields"`
	Methods          []MethodInfo `json:"methods"`
	CurrentView      string       `json:"currentView"`
	FontSize         string       `json:"fontSize,omitempty"`
	Color            string       `json:"color,omitempty"`
}

// FieldInfo 对应前端 FieldInfo 类型
type FieldInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Expanded    bool   `json:"expanded"`
}

// MethodInfo 对应前端 MethodInfo 类型
type MethodInfo struct {
	Name        string `json:"name"`
	Params      string `json:"params"`
	ReturnType  string `json:"returnType,omitempty"`
	Description string `json:"description,omitempty"`
	Expanded    bool   `json:"expanded"`
}

// VisualizerConnect 表示连接关系
type VisualizerConnect struct {
	FromID string `json:"fromId"`
	ToID   string `json:"toId"`
	Label  string `json:"label,omitempty"` // 可选：依赖类型描述
}

// VisualizerReporter 生成前端可视化格式的报告
type VisualizerReporter struct {
	// 布局参数
	BoxWidth      float64
	BoxHeight     float64
	HorizontalGap float64
	VerticalGap   float64
	StartX        float64
	StartY        float64
}

// NewVisualizerReporter 创建可视化报告生成器
func NewVisualizerReporter() *VisualizerReporter {
	return &VisualizerReporter{
		BoxWidth:      250,
		BoxHeight:     220,
		HorizontalGap: 80,
		VerticalGap:   100,
		StartX:        100,
		StartY:        100,
	}
}

// Generate 生成可视化输出
func (r *VisualizerReporter) Generate(result *types.AnalysisResult) *VisualizerOutput {
	output := &VisualizerOutput{
		ProjectPath: result.ProjectPath,
		StartStruct: result.StartStruct,
		GeneratedAt: result.GeneratedAt,
		Structs:     make([]VisualizerStruct, 0, len(result.Structs)),
		Connections: make([]VisualizerConnect, 0),
	}

	// 按深度分组结构体
	depthGroups := r.groupByDepth(result.Structs)

	// 计算布局位置
	positions := r.calculateLayout(depthGroups)

	// 深度对应的颜色
	depthColors := []string{"red", "blue", "green", "orange", "gray", "black"}

	// 转换结构体
	for _, s := range result.Structs {
		id := "struct-" + s.Name
		pos := positions[s.Name]
		color := depthColors[s.Depth%len(depthColors)]

		vs := VisualizerStruct{
			ID: id,
			X:  pos.X,
			Y:  pos.Y,
			Metadata: StructBoxMetadata{
				Type:             "struct-box",
				Name:             s.Name,
				Description:      s.Description,
				DescriptionTitle: s.Package,
				Fields:           r.convertFields(s.Fields),
				Methods:          r.convertMethods(s.Methods),
				CurrentView:      "fields",
				FontSize:         "m",
				Color:            color,
			},
		}
		output.Structs = append(output.Structs, vs)
	}

	// 生成连接关系（去重）
	connSet := make(map[string]bool)
	for _, s := range result.Structs {
		for _, dep := range s.Dependencies {
			fromID := "struct-" + dep.From
			toID := "struct-" + dep.To
			key := fromID + "->" + toID

			if !connSet[key] {
				connSet[key] = true
				output.Connections = append(output.Connections, VisualizerConnect{
					FromID: fromID,
					ToID:   toID,
					Label:  r.depTypeToLabel(dep.Type),
				})
			}
		}
	}

	return output
}

// Position 表示位置
type Position struct {
	X float64
	Y float64
}

// groupByDepth 按深度分组
func (r *VisualizerReporter) groupByDepth(structs []types.StructAnalysis) map[int][]string {
	groups := make(map[int][]string)
	for _, s := range structs {
		groups[s.Depth] = append(groups[s.Depth], s.Name)
	}
	return groups
}

// calculateLayout 计算布局位置（按深度分层，水平排列）
func (r *VisualizerReporter) calculateLayout(depthGroups map[int][]string) map[string]Position {
	positions := make(map[string]Position)

	// 找出最大深度
	maxDepth := 0
	for depth := range depthGroups {
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	// 计算每层的 Y 位置和水平布局
	for depth := 0; depth <= maxDepth; depth++ {
		names := depthGroups[depth]
		if len(names) == 0 {
			continue
		}

		y := r.StartY + float64(depth)*(r.BoxHeight+r.VerticalGap)

		// 计算该层总宽度，使其居中
		totalWidth := float64(len(names))*(r.BoxWidth+r.HorizontalGap) - r.HorizontalGap
		startX := r.StartX + (float64(maxDepth+1)*(r.BoxWidth+r.HorizontalGap)-totalWidth)/2

		for i, name := range names {
			x := startX + float64(i)*(r.BoxWidth+r.HorizontalGap)
			positions[name] = Position{X: x, Y: y}
		}
	}

	return positions
}

// convertFields 转换字段格式
func (r *VisualizerReporter) convertFields(fields []types.FieldAnalysis) []FieldInfo {
	result := make([]FieldInfo, 0, len(fields))
	for _, f := range fields {
		result = append(result, FieldInfo{
			Name:        f.Name,
			Type:        f.Type,
			Description: f.Description,
			Expanded:    false,
		})
	}
	return result
}

// convertMethods 转换方法格式
func (r *VisualizerReporter) convertMethods(methods []types.MethodAnalysis) []MethodInfo {
	result := make([]MethodInfo, 0, len(methods))
	for _, m := range methods {
		params, returnType := parseSignature(m.Signature)
		result = append(result, MethodInfo{
			Name:        m.Name,
			Params:      params,
			ReturnType:  returnType,
			Description: m.Description,
			Expanded:    false,
		})
	}
	return result
}

// parseSignature 解析方法签名，提取参数和返回类型
// 输入格式: "(name string, age int) error" 或 "(id int64) (*User, error)"
func parseSignature(sig string) (params string, returnType string) {
	sig = strings.TrimSpace(sig)
	if sig == "" {
		return "", ""
	}

	// 找到参数部分 (...)
	if !strings.HasPrefix(sig, "(") {
		return sig, ""
	}

	// 找到匹配的右括号
	depth := 0
	paramEnd := -1
	for i, ch := range sig {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				paramEnd = i
				break
			}
		}
	}

	if paramEnd == -1 {
		return sig, ""
	}

	params = sig[1:paramEnd] // 去掉括号
	returnType = strings.TrimSpace(sig[paramEnd+1:])

	return params, returnType
}

// depTypeToLabel 将依赖类型转换为标签
func (r *VisualizerReporter) depTypeToLabel(depType string) string {
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
		return depType
	}
}

// SaveToFile 保存到文件
func (r *VisualizerReporter) SaveToFile(output *VisualizerOutput, filePath string) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// ToJSON 转换为 JSON 字符串
func (r *VisualizerReporter) ToJSON(output *VisualizerOutput) (string, error) {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
