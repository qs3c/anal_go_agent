package types

// StructInfo 表示解析阶段提取的结构体原始信息
type StructInfo struct {
	Name       string       // 结构体名称
	Package    string       // 所属包名
	FilePath   string       // 所在文件路径
	SourceCode string       // 结构体源代码
	Fields     []FieldInfo  // 字段列表
	Methods    []MethodInfo // 方法列表
}

// FieldInfo 表示字段信息
type FieldInfo struct {
	Name       string // 字段名
	Type       string // 字段类型（完整类型名）
	Tag        string // 字段标签
	IsExported bool   // 是否导出（首字母大写）
	IsEmbedded bool   // 是否为嵌入字段
}

// MethodInfo 表示方法信息
type MethodInfo struct {
	Name       string // 方法名
	Signature  string // 完整签名
	Receiver   string // 接收者类型
	IsExported bool   // 是否导出
	SourceCode string // 方法源代码
}

// InterfaceInfo 表示接口信息
type InterfaceInfo struct {
	Name       string              // 接口名称
	Package    string              // 所属包名
	FilePath   string              // 所在文件路径
	Methods    []InterfaceMethod   // 方法列表
	SourceCode string              // 接口源代码
}

// InterfaceMethod 表示接口方法签名
type InterfaceMethod struct {
	Name      string // 方法名
	Signature string // 完整签名（参数和返回值）
}

// FunctionInfo 表示函数信息（用于构造函数检测）
type FunctionInfo struct {
	Name       string // 函数名
	Package    string // 所属包名
	FilePath   string // 所在文件路径
	ReturnType string // 返回类型
	Signature  string // 完整签名
}

// StructAnalysis 表示分析后的结构体信息（包含LLM描述）
type StructAnalysis struct {
	Name         string           // 结构体名称
	Package      string           // 所属包名
	Description  string           // 功能简述（Claude 生成）
	Fields       []FieldAnalysis  // 字段列表
	Methods      []MethodAnalysis // 方法列表
	Dependencies []Dependency     // 依赖关系
	Depth        int              // 在依赖树中的深度
}

// FieldAnalysis 表示分析后的字段信息
type FieldAnalysis struct {
	Name        string // 字段名
	Type        string // 字段类型
	Description string // 功能简述（Claude 生成）
	IsExported  bool   // 是否导出
	IsEmbedded  bool   // 是否为嵌入字段
}

// MethodAnalysis 表示分析后的方法信息
type MethodAnalysis struct {
	Name        string // 方法名
	Signature   string // 完整签名
	Description string // 功能简述（Claude 生成）
	IsExported  bool   // 是否导出
	Receiver    string // 接收者类型
}

// Dependency 表示依赖关系
type Dependency struct {
	From    string // 源结构体
	To      string // 目标结构体
	Type    string // 依赖类型："field", "init", "method_call", "interface", "embed"
	Context string // 上下文（字段名/方法名）
	Depth   int    // 依赖深度
}

// AnalysisResult 表示完整的分析结果
type AnalysisResult struct {
	ProjectPath  string           // 项目路径
	StartStruct  string           // 起始结构体
	MaxDepth     int              // 最大深度
	Structs      []StructAnalysis // 分析的结构体列表
	TotalStructs int              // 总结构体数
	TotalDeps    int              // 总依赖关系数
	Cycles       [][]string       // 循环依赖
	Blacklist    []string         // 黑名单类型
	GeneratedAt  string           // 生成时间
}

// AnalysisTask 表示分析任务（用于BFS遍历）
type AnalysisTask struct {
	StructName string // 结构体名称
	Depth      int    // 当前深度
}

// BlacklistConfig 表示黑名单配置
type BlacklistConfig struct {
	Types    []string `yaml:"types"`    // 忽略的类型列表
	Packages []string `yaml:"packages"` // 忽略的包列表
}

// LLMAnalysisResult 表示 LLM 分析结果
type LLMAnalysisResult struct {
	StructDescription string `json:"struct_description"`
	Fields            []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"fields"`
	Methods []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"methods"`
}

// DependencyType 定义依赖类型常量
const (
	DepTypeField       = "field"       // 字段依赖
	DepTypeInit        = "init"        // 方法内初始化
	DepTypeMethodCall  = "method_call" // 方法调用
	DepTypeInterface   = "interface"   // 接口实现
	DepTypeEmbed       = "embed"       // 结构体嵌入
	DepTypeConstructor = "constructor" // 构造函数调用
)
