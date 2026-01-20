// Package analyzer provides a public API for analyzing Go struct dependencies.
//
// Basic usage:
//
//	a := analyzer.New(analyzer.Options{
//	    ProjectPath: "./myproject",
//	    StartStruct: "UserService",
//	    MaxDepth:    2,
//	})
//
//	result, err := a.Analyze()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Found %d structs\n", result.TotalStructs)
//
// With LLM support:
//
//	a := analyzer.New(analyzer.Options{
//	    ProjectPath: "./myproject",
//	    StartStruct: "UserService",
//	    MaxDepth:    2,
//	    LLMProvider: "glm",
//	    APIKey:      os.Getenv("GLM_API_KEY"),
//	    EnableCache: true,
//	})
package analyzer

import (
	"fmt"
	"path/filepath"

	internalAnalyzer "github.com/user/go-struct-analyzer/internal/analyzer"
	"github.com/user/go-struct-analyzer/internal/llm"
	"github.com/user/go-struct-analyzer/internal/parser"
	"github.com/user/go-struct-analyzer/internal/reporter"
	"github.com/user/go-struct-analyzer/internal/types"
)

// Options 配置分析器选项
type Options struct {
	// ProjectPath 项目路径（必需）
	ProjectPath string

	// StartStruct 起点结构体名称（必需）
	StartStruct string

	// MaxDepth 分析深度，默认为 2
	MaxDepth int

	// BlacklistFile 黑名单文件路径（可选）
	BlacklistFile string

	// BlacklistTypes 黑名单类型列表（可选）
	BlacklistTypes []string

	// BlacklistPackages 黑名单包列表（可选）
	BlacklistPackages []string

	// LLMProvider LLM 提供商: "glm", "claude"（可选）
	LLMProvider string

	// APIKey LLM API Key（可选）
	APIKey string

	// EnableCache 是否启用 LLM 缓存，默认 true
	EnableCache bool

	// Verbose 详细输出模式
	Verbose bool
}

// Analyzer 结构体依赖分析器
type Analyzer struct {
	opts       Options
	parser     *parser.Parser
	traverser  *internalAnalyzer.Traverser
	blacklist  *internalAnalyzer.Blacklist
	cache      *internalAnalyzer.AnalysisCache
	llmClient  llm.LLMClient
	lastResult *Result
}

// New 创建新的分析器实例
func New(opts Options) (*Analyzer, error) {
	// 设置默认值
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 2
	}

	// 验证必需参数
	if opts.ProjectPath == "" {
		return nil, fmt.Errorf("ProjectPath is required")
	}
	if opts.StartStruct == "" {
		return nil, fmt.Errorf("StartStruct is required")
	}

	// 解析项目路径
	absPath, err := filepath.Abs(opts.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("invalid project path: %w", err)
	}
	opts.ProjectPath = absPath

	a := &Analyzer{
		opts: opts,
	}

	return a, nil
}

// Analyze 执行依赖分析
func (a *Analyzer) Analyze() (*Result, error) {
	// 1. 解析项目
	a.parser = parser.NewParser(a.opts.Verbose)
	if err := a.parser.ParseProject(a.opts.ProjectPath); err != nil {
		return nil, fmt.Errorf("failed to parse project: %w", err)
	}

	// 验证起点结构体存在
	if a.parser.GetStruct(a.opts.StartStruct) == nil {
		available := make([]string, 0)
		for name := range a.parser.GetAllStructs() {
			available = append(available, name)
		}
		return nil, fmt.Errorf("start struct '%s' not found, available: %v", a.opts.StartStruct, available)
	}

	// 2. 加载黑名单
	a.blacklist = internalAnalyzer.NewBlacklist()
	if a.opts.BlacklistFile != "" {
		if err := a.blacklist.LoadFromFile(a.opts.BlacklistFile); err != nil {
			return nil, fmt.Errorf("failed to load blacklist: %w", err)
		}
	}
	for _, t := range a.opts.BlacklistTypes {
		a.blacklist.AddType(t)
	}
	for _, p := range a.opts.BlacklistPackages {
		a.blacklist.AddPackage(p)
	}

	// 3. 创建 LLM 客户端（可选）
	if a.opts.APIKey != "" && a.opts.LLMProvider != "" {
		a.llmClient = llm.NewLLMClient(a.opts.LLMProvider, a.opts.APIKey)
	}

	// 4. 创建过滤器和遍历器
	filter := internalAnalyzer.NewScopeFilter(a.parser, a.blacklist)
	a.traverser = internalAnalyzer.NewTraverser(a.parser, filter, a.llmClient, a.opts.Verbose)

	// 5. 创建缓存（如果启用）
	if a.opts.EnableCache && a.llmClient != nil && a.llmClient.IsConfigured() {
		a.cache = internalAnalyzer.NewAnalysisCache(a.opts.ProjectPath)
		a.traverser.SetCache(a.cache)
	}

	// 6. 执行分析
	internalResult := a.traverser.Analyze(a.opts.StartStruct, a.opts.MaxDepth, a.opts.ProjectPath)

	// 7. 保存缓存
	if err := a.traverser.SaveCache(); err != nil && a.opts.Verbose {
		fmt.Printf("Warning: failed to save cache: %v\n", err)
	}

	// 8. 转换结果
	a.lastResult = convertResult(internalResult)

	return a.lastResult, nil
}

// GetResult 获取上次分析结果
func (a *Analyzer) GetResult() *Result {
	return a.lastResult
}

// GenerateMarkdown 生成 Markdown 报告
func (a *Analyzer) GenerateMarkdown() (string, error) {
	if a.lastResult == nil {
		return "", fmt.Errorf("no analysis result, call Analyze() first")
	}

	mdReporter := reporter.NewMarkdownReporter()
	content := mdReporter.Generate(a.lastResult.raw, a.blacklist.GetBlockedTypes())
	return content, nil
}

// GenerateJSON 生成 JSON 报告
func (a *Analyzer) GenerateJSON() (string, error) {
	if a.lastResult == nil {
		return "", fmt.Errorf("no analysis result, call Analyze() first")
	}

	jsonReporter := reporter.NewJSONReporter()
	return jsonReporter.Generate(a.lastResult.raw)
}

// GenerateMermaid 生成 Mermaid 依赖图
func (a *Analyzer) GenerateMermaid() (string, error) {
	if a.lastResult == nil {
		return "", fmt.Errorf("no analysis result, call Analyze() first")
	}

	mermaidGen := reporter.NewMermaidGenerator()
	return mermaidGen.Generate(a.lastResult.raw), nil
}

// GenerateVisualizerJSON 生成可视化工具 JSON 字符串
func (a *Analyzer) GenerateVisualizerJSON() (string, error) {
	if a.lastResult == nil {
		return "", fmt.Errorf("no analysis result, call Analyze() first")
	}

	vizReporter := reporter.NewVisualizerReporter()
	vizOutput := vizReporter.Generate(a.lastResult.raw)
	return vizReporter.ToJSON(vizOutput)
}

// SaveMarkdown 保存 Markdown 报告到文件
func (a *Analyzer) SaveMarkdown(path string) error {
	content, err := a.GenerateMarkdown()
	if err != nil {
		return err
	}

	mdReporter := reporter.NewMarkdownReporter()
	return mdReporter.SaveToFile(content, path)
}

// SaveJSON 保存 JSON 报告到文件
func (a *Analyzer) SaveJSON(path string) error {
	if a.lastResult == nil {
		return fmt.Errorf("no analysis result, call Analyze() first")
	}

	jsonReporter := reporter.NewJSONReporter()
	return jsonReporter.SaveToFile(a.lastResult.raw, path)
}

// SaveMermaid 保存 Mermaid 图到文件
func (a *Analyzer) SaveMermaid(path string) error {
	if a.lastResult == nil {
		return fmt.Errorf("no analysis result, call Analyze() first")
	}

	mermaidGen := reporter.NewMermaidGenerator()
	return mermaidGen.GenerateToFile(a.lastResult.raw, path)
}

// SaveVisualizerJSON 保存可视化工具 JSON 到文件
func (a *Analyzer) SaveVisualizerJSON(path string) error {
	if a.lastResult == nil {
		return fmt.Errorf("no analysis result, call Analyze() first")
	}

	vizReporter := reporter.NewVisualizerReporter()
	vizOutput := vizReporter.Generate(a.lastResult.raw)
	return vizReporter.SaveToFile(vizOutput, path)
}

// GetAllStructs 获取项目中所有结构体名称
func (a *Analyzer) GetAllStructs() []string {
	if a.parser == nil {
		return nil
	}

	structs := a.parser.GetAllStructs()
	names := make([]string, 0, len(structs))
	for name := range structs {
		names = append(names, name)
	}
	return names
}

// convertResult 将内部结果转换为公共 API 结果
func convertResult(r *types.AnalysisResult) *Result {
	result := &Result{
		ProjectPath:  r.ProjectPath,
		StartStruct:  r.StartStruct,
		MaxDepth:     r.MaxDepth,
		TotalStructs: r.TotalStructs,
		TotalDeps:    r.TotalDeps,
		GeneratedAt:  r.GeneratedAt,
		Cycles:       r.Cycles,
		Blacklist:    r.Blacklist,
		raw:          r,
	}

	// 转换结构体分析
	for _, s := range r.Structs {
		sa := StructAnalysis{
			Name:        s.Name,
			Package:     s.Package,
			Description: s.Description,
			Depth:       s.Depth,
		}

		// 转换字段
		for _, f := range s.Fields {
			sa.Fields = append(sa.Fields, FieldAnalysis{
				Name:        f.Name,
				Type:        f.Type,
				Description: f.Description,
				IsExported:  f.IsExported,
				IsEmbedded:  f.IsEmbedded,
			})
		}

		// 转换方法
		for _, m := range s.Methods {
			sa.Methods = append(sa.Methods, MethodAnalysis{
				Name:        m.Name,
				Signature:   m.Signature,
				Description: m.Description,
				IsExported:  m.IsExported,
			})
		}

		// 转换依赖
		for _, d := range s.Dependencies {
			sa.Dependencies = append(sa.Dependencies, Dependency{
				From:    d.From,
				To:      d.To,
				Type:    DependencyType(d.Type),
				Context: d.Context,
				Depth:   d.Depth,
			})
		}

		result.Structs = append(result.Structs, sa)
	}

	return result
}

