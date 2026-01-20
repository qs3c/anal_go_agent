// Example: Using go-struct-analyzer as a library
//
// This example demonstrates how to use the analyzer package
// to analyze Go struct dependencies programmatically.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/user/go-struct-analyzer/pkg/analyzer"
)

func main() {
	// 示例 1: 基本使用
	basicUsage()

	// 示例 2: 使用 LLM 分析
	// llmUsage()

	// 示例 3: 遍历分析结果
	// traverseResult()
}

// basicUsage 展示基本使用方法
func basicUsage() {
	fmt.Println("=== 示例 1: 基本使用 ===")

	// 创建分析器
	a, err := analyzer.New(analyzer.Options{
		ProjectPath: "../../testdata/sample_project", // 你的项目路径
		StartStruct: "UserService",                   // 起点结构体
		MaxDepth:    2,                               // 分析深度
	})
	if err != nil {
		log.Fatalf("创建分析器失败: %v", err)
	}

	// 执行分析
	result, err := a.Analyze()
	if err != nil {
		log.Fatalf("分析失败: %v", err)
	}

	// 打印基本信息
	fmt.Printf("项目路径: %s\n", result.ProjectPath)
	fmt.Printf("分析结构体总数: %d\n", result.TotalStructs)
	fmt.Printf("依赖关系总数: %d\n", result.TotalDeps)
	fmt.Printf("是否存在循环依赖: %v\n", result.HasCycles())

	// 打印所有分析的结构体
	fmt.Println("\n分析的结构体:")
	for _, s := range result.Structs {
		fmt.Printf("  - %s (深度: %d, 包: %s)\n", s.Name, s.Depth, s.Package)
	}

	// 生成 Markdown 报告
	md, err := a.GenerateMarkdown()
	if err != nil {
		log.Fatalf("生成 Markdown 失败: %v", err)
	}
	fmt.Printf("\n生成的 Markdown 长度: %d 字符\n", len(md))

	// 保存到文件
	if err := a.SaveMarkdown("./output_report.md"); err != nil {
		log.Printf("保存文件失败: %v", err)
	} else {
		fmt.Println("报告已保存到 ./output_report.md")
	}
}

// llmUsage 展示使用 LLM 分析
func llmUsage() {
	fmt.Println("\n=== 示例 2: 使用 LLM 分析 ===")

	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		fmt.Println("未设置 GLM_API_KEY，跳过 LLM 示例")
		return
	}

	a, err := analyzer.New(analyzer.Options{
		ProjectPath: "../../testdata/sample_project",
		StartStruct: "UserService",
		MaxDepth:    2,
		LLMProvider: "glm",       // 或 "claude"
		APIKey:      apiKey,
		EnableCache: true,        // 启用缓存，避免重复调用
	})
	if err != nil {
		log.Fatalf("创建分析器失败: %v", err)
	}

	result, err := a.Analyze()
	if err != nil {
		log.Fatalf("分析失败: %v", err)
	}

	// 打印 LLM 生成的描述
	for _, s := range result.Structs {
		fmt.Printf("\n%s:\n", s.Name)
		fmt.Printf("  描述: %s\n", s.Description)
		for _, f := range s.Fields {
			fmt.Printf("  字段 %s: %s\n", f.Name, f.Description)
		}
	}
}

// traverseResult 展示遍历分析结果
func traverseResult() {
	fmt.Println("\n=== 示例 3: 遍历分析结果 ===")

	a, err := analyzer.New(analyzer.Options{
		ProjectPath: "../../testdata/sample_project",
		StartStruct: "UserService",
		MaxDepth:    2,
	})
	if err != nil {
		log.Fatalf("创建分析器失败: %v", err)
	}

	result, err := a.Analyze()
	if err != nil {
		log.Fatalf("分析失败: %v", err)
	}

	// 按深度获取结构体
	fmt.Println("按深度分组:")
	for depth := 0; depth <= result.MaxDepth; depth++ {
		structs := result.GetStructsByDepth(depth)
		if len(structs) > 0 {
			fmt.Printf("  深度 %d:\n", depth)
			for _, s := range structs {
				fmt.Printf("    - %s\n", s.Name)
			}
		}
	}

	// 获取特定结构体的依赖
	fmt.Println("\nUserService 的依赖:")
	deps := result.GetDependenciesOf("UserService")
	for _, d := range deps {
		fmt.Printf("  -> %s (%s: %s)\n", d.To, d.Type, d.Context)
	}

	// 获取依赖某结构体的其他结构体
	fmt.Println("\n依赖 UserRepository 的结构体:")
	dependents := result.GetDependentsOf("UserRepository")
	for _, name := range dependents {
		fmt.Printf("  <- %s\n", name)
	}

	// 获取所有依赖关系
	fmt.Println("\n所有依赖关系:")
	allDeps := result.GetAllDependencies()
	for _, d := range allDeps {
		fmt.Printf("  %s -> %s (%s)\n", d.From, d.To, d.Type)
	}
}
