package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/user/go-struct-analyzer/internal/analyzer"
	"github.com/user/go-struct-analyzer/internal/llm"
	"github.com/user/go-struct-analyzer/internal/parser"
	"github.com/user/go-struct-analyzer/internal/reporter"
)

var (
	projectPath    string
	startStruct    string
	depth          int
	outputPath     string
	format         string
	blacklistPath  string
	apiKey         string
	llmProvider    string
	llmModel       string
	mermaidPath    string
	visualizerPath string
	noCache        bool
	verbose        bool
)

var rootCmd = &cobra.Command{
	Use:   "go-struct-analyzer",
	Short: "Go 项目结构体依赖关系分析工具",
	Long: `分析 Go 语言项目中结构体之间的依赖关系，
并使用 LLM API 生成功能描述和可视化依赖图。

支持的 LLM 后端：
  - glm: 智谱 GLM（默认）
  - claude: Anthropic Claude

示例:
  go-struct-analyzer --project ./myapp --start UserService --depth 2
  go-struct-analyzer -p ./myapp -s UserService --llm glm -k $GLM_API_KEY
  go-struct-analyzer -p ./myapp -s UserService --llm claude -k $CLAUDE_API_KEY
  go-struct-analyzer -p ./myapp -s UserService -b ./blacklist.yaml -v
  go-struct-analyzer -p ./myapp -s UserService --visualizer ./output.json`,
	Run: runAnalyzer,
}

func init() {
	rootCmd.Flags().StringVarP(&projectPath, "project", "p", "", "项目路径（必需）")
	rootCmd.Flags().StringVarP(&startStruct, "start", "s", "", "起点结构体名称（必需）")
	rootCmd.Flags().IntVarP(&depth, "depth", "d", 2, "分析深度")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "./analysis_report.md", "输出文件路径")
	rootCmd.Flags().StringVarP(&format, "format", "f", "markdown", "输出格式：markdown, json")
	rootCmd.Flags().StringVarP(&blacklistPath, "blacklist", "b", "", "黑名单文件路径")
	rootCmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "LLM API Key（可选，也可通过环境变量设置）")
	rootCmd.Flags().StringVar(&llmProvider, "llm", "glm", "LLM 后端：glm（默认）, claude")
	rootCmd.Flags().StringVarP(&llmModel, "model", "m", "", "LLM 模型（可选，默认: glm-4-flash / claude-sonnet-4-20250514）")
	rootCmd.Flags().StringVar(&mermaidPath, "mermaid", "", "Mermaid 图输出路径（可选）")
	rootCmd.Flags().StringVar(&visualizerPath, "visualizer", "", "可视化工具 JSON 输出路径（可选）")
	rootCmd.Flags().BoolVar(&noCache, "no-cache", false, "禁用 LLM 分析结果缓存")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "详细输出模式")

	rootCmd.MarkFlagRequired("project")
	rootCmd.MarkFlagRequired("start")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runAnalyzer(cmd *cobra.Command, args []string) {
	// 验证项目路径
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法解析项目路径: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(absProjectPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "错误: 项目路径不存在: %s\n", absProjectPath)
		os.Exit(1)
	}

	if verbose {
		fmt.Println("=== Go 结构体依赖分析器 ===")
		fmt.Printf("项目路径: %s\n", absProjectPath)
		fmt.Printf("起点结构体: %s\n", startStruct)
		fmt.Printf("分析深度: %d\n", depth)
		fmt.Println()
	}

	// 1. 解析项目
	if verbose {
		fmt.Println("正在解析项目...")
	}

	p := parser.NewParser(verbose)
	if err := p.ParseProject(absProjectPath); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析项目失败: %v\n", err)
		os.Exit(1)
	}

	// 验证起点结构体存在
	if p.GetStruct(startStruct) == nil {
		fmt.Fprintf(os.Stderr, "错误: 未找到起点结构体 '%s'\n", startStruct)
		fmt.Fprintln(os.Stderr, "可用的结构体:")
		for name := range p.GetAllStructs() {
			fmt.Fprintf(os.Stderr, "  - %s\n", name)
		}
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("解析完成，共发现 %d 个结构体\n\n", len(p.GetAllStructs()))
	}

	// 2. 加载黑名单
	blacklist := analyzer.NewBlacklist()
	if blacklistPath != "" {
		if err := blacklist.LoadFromFile(blacklistPath); err != nil {
			fmt.Fprintf(os.Stderr, "警告: 加载黑名单失败: %v\n", err)
		} else if verbose {
			fmt.Printf("已加载黑名单: %v\n", blacklist.GetBlockedTypes())
		}
	}

	// 3. 创建 LLM 客户端（可选）
	var llmClient llm.LLMClient
	effectiveAPIKey := apiKey

	// 根据 LLM 后端选择环境变量
	if effectiveAPIKey == "" {
		switch llmProvider {
		case "glm", "zhipu":
			effectiveAPIKey = os.Getenv("GLM_API_KEY")
		case "claude", "anthropic":
			effectiveAPIKey = os.Getenv("CLAUDE_API_KEY")
		default:
			effectiveAPIKey = os.Getenv("GLM_API_KEY")
		}
	}

	if effectiveAPIKey != "" {
		llmClient = llm.NewLLMClientWithModel(llmProvider, effectiveAPIKey, llmModel)
		if verbose {
			fmt.Printf("已启用 LLM 分析功能 (后端: %s, 模型: %s)\n", llmClient.Name(), llmClient.Model())
		}
	} else if verbose {
		fmt.Println("未配置 API Key，将跳过 LLM 分析")
	}

	// 4. 创建过滤器和遍历器
	filter := analyzer.NewScopeFilter(p, blacklist)
	traverser := analyzer.NewTraverser(p, filter, llmClient, verbose)

	// 5. 创建缓存（如果未禁用且有 LLM 客户端）
	if !noCache && llmClient != nil && llmClient.IsConfigured() {
		cache := analyzer.NewAnalysisCache(absProjectPath)
		traverser.SetCache(cache)
		if verbose {
			fmt.Printf("LLM 缓存已启用 (缓存条目: %d)\n", cache.Size())
		}
	}

	// 6. 执行分析
	if verbose {
		fmt.Println("\n正在分析依赖关系...")
	}

	result := traverser.Analyze(startStruct, depth, absProjectPath)

	// 保存缓存
	if err := traverser.SaveCache(); err != nil && verbose {
		fmt.Printf("警告: 保存缓存失败: %v\n", err)
	}

	if verbose {
		fmt.Printf("分析完成，共分析 %d 个结构体，%d 个依赖关系\n\n", result.TotalStructs, result.TotalDeps)
	}

	// 6. 生成报告
	if verbose {
		fmt.Println("正在生成报告...")
	}

	switch format {
	case "json":
		jsonReporter := reporter.NewJSONReporter()
		if err := jsonReporter.SaveToFile(result, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 保存 JSON 报告失败: %v\n", err)
			os.Exit(1)
		}
	default: // markdown
		mdReporter := reporter.NewMarkdownReporter()
		content := mdReporter.Generate(result, blacklist.GetBlockedTypes())
		if err := mdReporter.SaveToFile(content, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 保存 Markdown 报告失败: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("报告已保存至: %s\n", outputPath)

	// 7. 生成 Mermaid 图（可选）
	if mermaidPath != "" {
		mermaidGen := reporter.NewMermaidGenerator()
		if err := mermaidGen.GenerateToFile(result, mermaidPath); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 保存 Mermaid 图失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Mermaid 图已保存至: %s\n", mermaidPath)
	}

	// 8. 生成可视化工具 JSON（可选）
	if visualizerPath != "" {
		vizReporter := reporter.NewVisualizerReporter()
		vizOutput := vizReporter.Generate(result)
		if err := vizReporter.SaveToFile(vizOutput, visualizerPath); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 保存可视化 JSON 失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("可视化 JSON 已保存至: %s\n", visualizerPath)
	}

	// 9. 输出摘要
	if len(result.Cycles) > 0 {
		fmt.Printf("\n警告: 发现 %d 个循环依赖\n", len(result.Cycles))
		for _, cycle := range result.Cycles {
			fmt.Printf("  %v\n", cycle)
		}
	}

	fmt.Println("\n分析完成！")
}
