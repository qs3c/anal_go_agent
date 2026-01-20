package analyzer

import "github.com/user/go-struct-analyzer/internal/types"

// Result 分析结果
type Result struct {
	// ProjectPath 项目路径
	ProjectPath string

	// StartStruct 起点结构体
	StartStruct string

	// MaxDepth 分析深度
	MaxDepth int

	// TotalStructs 分析的结构体总数
	TotalStructs int

	// TotalDeps 依赖关系总数
	TotalDeps int

	// GeneratedAt 生成时间
	GeneratedAt string

	// Structs 结构体分析列表
	Structs []StructAnalysis

	// Cycles 检测到的循环依赖
	Cycles [][]string

	// Blacklist 使用的黑名单
	Blacklist []string

	// raw 内部原始结果（用于生成报告）
	raw *types.AnalysisResult
}

// StructAnalysis 单个结构体的分析结果
type StructAnalysis struct {
	// Name 结构体名称
	Name string

	// Package 所属包名
	Package string

	// Description 功能描述（来自 LLM 或默认值）
	Description string

	// Depth 在依赖树中的深度
	Depth int

	// Fields 字段列表
	Fields []FieldAnalysis

	// Methods 方法列表
	Methods []MethodAnalysis

	// Dependencies 依赖列表
	Dependencies []Dependency
}

// FieldAnalysis 字段分析结果
type FieldAnalysis struct {
	// Name 字段名
	Name string

	// Type 字段类型
	Type string

	// Description 字段描述
	Description string

	// IsExported 是否导出
	IsExported bool

	// IsEmbedded 是否为嵌入字段
	IsEmbedded bool
}

// MethodAnalysis 方法分析结果
type MethodAnalysis struct {
	// Name 方法名
	Name string

	// Signature 方法签名
	Signature string

	// Description 方法描述
	Description string

	// IsExported 是否导出
	IsExported bool
}

// DependencyType 依赖类型
type DependencyType string

const (
	// DepTypeField 字段依赖
	DepTypeField DependencyType = "field"

	// DepTypeInit 方法内初始化
	DepTypeInit DependencyType = "init"

	// DepTypeMethodCall 方法调用
	DepTypeMethodCall DependencyType = "method_call"

	// DepTypeInterface 接口实现
	DepTypeInterface DependencyType = "interface"

	// DepTypeEmbed 结构体嵌入
	DepTypeEmbed DependencyType = "embed"

	// DepTypeConstructor 构造函数调用
	DepTypeConstructor DependencyType = "constructor"
)

// Dependency 依赖关系
type Dependency struct {
	// From 依赖来源结构体
	From string

	// To 依赖目标结构体
	To string

	// Type 依赖类型
	Type DependencyType

	// Context 上下文信息（如字段名、方法名）
	Context string

	// Depth 深度
	Depth int
}

// GetStructByName 根据名称获取结构体分析
func (r *Result) GetStructByName(name string) *StructAnalysis {
	for i := range r.Structs {
		if r.Structs[i].Name == name {
			return &r.Structs[i]
		}
	}
	return nil
}

// GetStructsByDepth 获取指定深度的结构体
func (r *Result) GetStructsByDepth(depth int) []StructAnalysis {
	var result []StructAnalysis
	for _, s := range r.Structs {
		if s.Depth == depth {
			result = append(result, s)
		}
	}
	return result
}

// HasCycles 是否存在循环依赖
func (r *Result) HasCycles() bool {
	return len(r.Cycles) > 0
}

// GetAllDependencies 获取所有依赖关系
func (r *Result) GetAllDependencies() []Dependency {
	var deps []Dependency
	for _, s := range r.Structs {
		deps = append(deps, s.Dependencies...)
	}
	return deps
}

// GetDependenciesOf 获取指定结构体的依赖
func (r *Result) GetDependenciesOf(structName string) []Dependency {
	for _, s := range r.Structs {
		if s.Name == structName {
			return s.Dependencies
		}
	}
	return nil
}

// GetDependentsOf 获取依赖指定结构体的结构体
func (r *Result) GetDependentsOf(structName string) []string {
	var dependents []string
	seen := make(map[string]bool)

	for _, s := range r.Structs {
		for _, d := range s.Dependencies {
			if d.To == structName && !seen[s.Name] {
				dependents = append(dependents, s.Name)
				seen[s.Name] = true
			}
		}
	}
	return dependents
}
