package analyzer

import (
	"sync"
	"time"

	"github.com/user/go-struct-analyzer/internal/llm"
	"github.com/user/go-struct-analyzer/internal/parser"
	"github.com/user/go-struct-analyzer/internal/types"
)

// LLMConcurrency LLM 并发调用数限制
const LLMConcurrency = 3

// llmTask 表示一个 LLM 分析任务
type llmTask struct {
	index int
	info  *types.StructInfo
}

// Traverser 使用 BFS 遍历结构体依赖
type Traverser struct {
	parser      *parser.Parser
	depAnalyzer *DependencyAnalyzer
	filter      *ScopeFilter
	llmClient   llm.LLMClient
	cache       *AnalysisCache
	verbose     bool
}

// NewTraverser 创建遍历器
func NewTraverser(p *parser.Parser, filter *ScopeFilter, llmClient llm.LLMClient, verbose bool) *Traverser {
	return &Traverser{
		parser:      p,
		depAnalyzer: NewDependencyAnalyzer(p, filter, verbose),
		filter:      filter,
		llmClient:   llmClient,
		verbose:     verbose,
	}
}

// SetCache 设置缓存
func (t *Traverser) SetCache(cache *AnalysisCache) {
	t.cache = cache
}

// SaveCache 保存缓存
func (t *Traverser) SaveCache() error {
	if t.cache != nil {
		return t.cache.Save()
	}
	return nil
}

// Analyze 从起始结构体开始进行 BFS 分析
func (t *Traverser) Analyze(startStruct string, maxDepth int, projectPath string) *types.AnalysisResult {
	visited := make(map[string]bool)
	queue := []types.AnalysisTask{{StructName: startStruct, Depth: 0}}

	result := &types.AnalysisResult{
		ProjectPath: projectPath,
		StartStruct: startStruct,
		MaxDepth:    maxDepth,
		Structs:     []types.StructAnalysis{},
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 收集需要 LLM 分析的结构体信息
	var llmTasks []llmTask

	for len(queue) > 0 {
		task := queue[0]
		queue = queue[1:]

		// 深度检查
		if task.Depth > maxDepth {
			continue
		}

		// 去重检查
		if visited[task.StructName] {
			continue
		}
		visited[task.StructName] = true

		// 获取结构体信息
		structInfo := t.parser.GetStruct(task.StructName)
		if structInfo == nil {
			if t.verbose {
				println("Warning: struct not found:", task.StructName)
			}
			continue
		}

		if t.verbose {
			println("Analyzing struct:", task.StructName, "at depth", task.Depth)
		}

		// 分析依赖关系
		deps := t.depAnalyzer.AnalyzeStruct(structInfo)

		// 设置依赖深度
		for i := range deps {
			deps[i].Depth = task.Depth + 1
		}

		// 构建分析结果（不包含 LLM 分析）
		structAnalysis := t.buildStructAnalysisWithoutLLM(structInfo, deps, task.Depth)
		result.Structs = append(result.Structs, structAnalysis)
		result.TotalDeps += len(deps)

		// 记录需要 LLM 分析的任务
		if t.llmClient != nil && t.llmClient.IsConfigured() {
			llmTasks = append(llmTasks, llmTask{
				index: len(result.Structs) - 1,
				info:  structInfo,
			})
		}

		// 将依赖加入队列
		for _, dep := range deps {
			if !visited[dep.To] && t.filter.ShouldAnalyze(dep.To) {
				queue = append(queue, types.AnalysisTask{
					StructName: dep.To,
					Depth:      task.Depth + 1,
				})
			}
		}
	}

	result.TotalStructs = len(result.Structs)

	// 并发调用 LLM 分析
	if len(llmTasks) > 0 {
		t.enrichWithLLMConcurrently(result, llmTasks)
	}

	// 检测循环依赖
	result.Cycles = t.detectCycles(result.Structs)

	return result
}

// enrichWithLLMConcurrently 并发调用 LLM 分析（支持缓存）
func (t *Traverser) enrichWithLLMConcurrently(result *types.AnalysisResult, tasks []llmTask) {
	llmProvider := ""
	if t.llmClient != nil {
		llmProvider = t.llmClient.Name()
	}

	// 先检查缓存，分离出需要调用 LLM 的任务
	var llmCallTasks []llmTask
	var mu sync.Mutex // 保护 result.Structs

	cacheHits := 0
	for _, task := range tasks {
		if t.cache != nil {
			cached := t.cache.Get(task.info.Name, task.info.SourceCode, llmProvider)
			if cached != nil {
				// 缓存命中
				mu.Lock()
				t.applyLLMResult(&result.Structs[task.index], cached)
				mu.Unlock()
				cacheHits++
				continue
			}
		}
		llmCallTasks = append(llmCallTasks, task)
	}

	if t.verbose {
		println("LLM analysis: cache hits:", cacheHits, ", need to call:", len(llmCallTasks))
	}

	if len(llmCallTasks) == 0 {
		return
	}

	if t.verbose {
		println("Starting concurrent LLM analysis for", len(llmCallTasks), "structs (max concurrency:", LLMConcurrency, ")")
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, LLMConcurrency) // 限制并发数

	for _, task := range llmCallTasks {
		wg.Add(1)
		go func(idx int, info *types.StructInfo) {
			defer wg.Done()
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			if t.verbose {
				println("Calling LLM for:", info.Name)
			}

			llmResult, err := t.llmClient.AnalyzeStruct(info)
			if err != nil {
				if t.verbose {
					println("LLM analysis failed for", info.Name, ":", err.Error())
				}
				return
			}

			// 更新结果
			mu.Lock()
			t.applyLLMResult(&result.Structs[idx], llmResult)
			mu.Unlock()

			// 保存到缓存
			if t.cache != nil {
				t.cache.Set(info.Name, info.SourceCode, llmProvider, llmResult)
			}
		}(task.index, task.info)
	}

	wg.Wait()

	if t.verbose {
		println("LLM analysis completed")
	}
}

// applyLLMResult 应用 LLM 分析结果到结构体分析
func (t *Traverser) applyLLMResult(analysis *types.StructAnalysis, llmResult *types.LLMAnalysisResult) {
	// 更新结构体描述
	analysis.Description = llmResult.StructDescription

	// 更新字段描述
	fieldDescMap := make(map[string]string)
	for _, f := range llmResult.Fields {
		fieldDescMap[f.Name] = f.Description
	}
	for i := range analysis.Fields {
		if desc, ok := fieldDescMap[analysis.Fields[i].Name]; ok {
			analysis.Fields[i].Description = desc
		}
	}

	// 更新方法描述
	methodDescMap := make(map[string]string)
	for _, m := range llmResult.Methods {
		methodDescMap[m.Name] = m.Description
	}
	for i := range analysis.Methods {
		if desc, ok := methodDescMap[analysis.Methods[i].Name]; ok {
			analysis.Methods[i].Description = desc
		}
	}
}

// buildStructAnalysisWithoutLLM 构建结构体分析结果（不包含 LLM 分析）
func (t *Traverser) buildStructAnalysisWithoutLLM(info *types.StructInfo, deps []types.Dependency, depth int) types.StructAnalysis {
	analysis := types.StructAnalysis{
		Name:         info.Name,
		Package:      info.Package,
		Description:  "待分析",
		Fields:       make([]types.FieldAnalysis, 0, len(info.Fields)),
		Methods:      make([]types.MethodAnalysis, 0, len(info.Methods)),
		Dependencies: deps,
		Depth:        depth,
	}

	// 转换字段
	for _, field := range info.Fields {
		analysis.Fields = append(analysis.Fields, types.FieldAnalysis{
			Name:        field.Name,
			Type:        field.Type,
			Description: "待分析",
			IsExported:  field.IsExported,
			IsEmbedded:  field.IsEmbedded,
		})
	}

	// 转换方法
	for _, method := range info.Methods {
		analysis.Methods = append(analysis.Methods, types.MethodAnalysis{
			Name:        method.Name,
			Signature:   method.Signature,
			Description: "待分析",
			IsExported:  method.IsExported,
			Receiver:    method.Receiver,
		})
	}

	return analysis
}

// detectCycles 检测循环依赖
func (t *Traverser) detectCycles(structs []types.StructAnalysis) [][]string {
	// 构建依赖图
	graph := make(map[string][]string)
	for _, s := range structs {
		for _, dep := range s.Dependencies {
			graph[s.Name] = append(graph[s.Name], dep.To)
		}
	}

	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				dfs(neighbor, path)
			} else if recStack[neighbor] {
				// 找到循环
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart != -1 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
			}
		}

		recStack[node] = false
	}

	for node := range graph {
		if !visited[node] {
			dfs(node, []string{})
		}
	}

	return cycles
}
