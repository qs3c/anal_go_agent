package llm

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/user/go-struct-analyzer/internal/types"
)

const promptTemplate = `你是一个 Go 语言代码分析专家。请分析以下结构体并生成简洁的功能描述。

结构体名称: {{.StructName}}
所属包: {{.Package}}

结构体定义:
` + "```go" + `
{{.StructCode}}
` + "```" + `

方法实现:
` + "```go" + `
{{.MethodsCode}}
` + "```" + `

请以 JSON 格式返回分析结果，包括：
1. struct_description: 结构体的功能简述（1-2句话，直接说明这个结构体是做什么的）
2. fields: 数组，每个字段包含 name 和 description（简短说明字段用途）
3. methods: 数组，每个方法包含 name 和 description（简短说明方法功能）

要求：
- 描述要简洁明了，每个描述控制在 20 字以内
- 直接说明"做什么"而不是"怎么做"
- 不要包含代码片段
- 只返回纯 JSON，不要包含任何其他文本或 Markdown 标记

输出格式示例：
{
  "struct_description": "用户服务层，处理用户注册、登录等业务逻辑",
  "fields": [
    {"name": "repo", "description": "用户数据仓库，负责数据持久化"},
    {"name": "cache", "description": "缓存服务，提升查询性能"}
  ],
  "methods": [
    {"name": "CreateUser", "description": "创建新用户并发送欢迎邮件"},
    {"name": "GetUserByID", "description": "根据ID查询用户信息"}
  ]
}`

// PromptData 表示 prompt 模板数据
type PromptData struct {
	StructName  string
	Package     string
	StructCode  string
	MethodsCode string
}

// buildPrompt 构建 LLM 提示词
func buildPrompt(info *types.StructInfo) string {
	data := PromptData{
		StructName:  info.Name,
		Package:     info.Package,
		StructCode:  info.SourceCode,
		MethodsCode: buildMethodsCode(info.Methods),
	}

	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		// 降级处理：返回简化的提示词
		return buildSimplePrompt(info)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return buildSimplePrompt(info)
	}

	return buf.String()
}

// buildMethodsCode 构建方法代码字符串
func buildMethodsCode(methods []types.MethodInfo) string {
	if len(methods) == 0 {
		return "// 无方法"
	}

	var sb strings.Builder
	for i, method := range methods {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(method.SourceCode)
	}
	return sb.String()
}

// buildSimplePrompt 构建简化的提示词
func buildSimplePrompt(info *types.StructInfo) string {
	var sb strings.Builder
	sb.WriteString("分析以下 Go 结构体，返回 JSON 格式的描述：\n\n")
	sb.WriteString("结构体名称: ")
	sb.WriteString(info.Name)
	sb.WriteString("\n包名: ")
	sb.WriteString(info.Package)
	sb.WriteString("\n\n代码:\n")
	sb.WriteString(info.SourceCode)

	if len(info.Methods) > 0 {
		sb.WriteString("\n\n方法:\n")
		for _, m := range info.Methods {
			sb.WriteString(m.SourceCode)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n返回格式: {\"struct_description\": \"...\", \"fields\": [{\"name\": \"...\", \"description\": \"...\"}], \"methods\": [{\"name\": \"...\", \"description\": \"...\"}]}")

	return sb.String()
}
