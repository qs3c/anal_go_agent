# Claude Code 项目指南

## 项目概述

`go-struct-analyzer` 是一个 Go 语言结构体依赖分析 CLI 工具，用于分析 Go 项目中结构体之间的依赖关系，并可选使用 Claude API 生成代码描述。

## 项目结构

```
go-struct-analyzer/
├── cmd/analyzer/main.go           # CLI 入口，使用 cobra
├── internal/
│   ├── types/models.go            # 核心数据模型定义
│   ├── parser/
│   │   ├── parser.go              # AST 解析器，提取结构体/方法
│   │   └── type_resolver.go       # 类型推断和解析
│   ├── analyzer/
│   │   ├── dependency.go          # 依赖关系分析
│   │   ├── traverser.go           # BFS 遍历器
│   │   ├── blacklist.go           # 黑名单过滤
│   │   └── scope_filter.go        # 范围过滤（排除标准库/第三方）
│   ├── llm/
│   │   ├── client.go              # Claude API 客户端
│   │   ├── prompt.go              # Prompt 模板构建
│   │   └── response.go            # 响应解析
│   └── reporter/
│       ├── markdown.go            # Markdown 报告生成
│       ├── mermaid.go             # Mermaid 依赖图生成
│       └── json.go                # JSON 输出
├── testdata/sample_project/       # 测试用的示例 Go 项目
├── blacklist.yaml                 # 默认黑名单配置
└── requirement.md                 # 原始需求文档（在上级目录）
```

## 技术决策

1. **CLI 框架**: 使用 `github.com/spf13/cobra`
2. **配置格式**: 黑名单使用 YAML 格式 (`gopkg.in/yaml.v3`)
3. **AST 解析**: 使用 Go 标准库 `go/ast`, `go/parser`, `go/token`
4. **遍历算法**: BFS（广度优先搜索）按深度分析依赖
5. **LLM 集成**: 直接调用 Claude API，支持重试和降级

## 开发规范

- 编译命令: `go build -o go-struct-analyzer.exe ./cmd/analyzer`
- 测试运行: `./go-struct-analyzer.exe -p ./testdata/sample_project -s UserService -d 2 -v`
- 依赖管理: `go mod tidy`

## 关键命令行参数

| 参数 | 必需 | 说明 |
|------|------|------|
| -p, --project | 是 | 项目路径 |
| -s, --start | 是 | 起点结构体名称 |
| -d, --depth | 否 | 分析深度，默认 2 |
| -o, --output | 否 | 输出文件，默认 ./analysis_report.md |
| -k, --api-key | 否 | Claude API Key |
| -b, --blacklist | 否 | 黑名单文件路径 |
| --mermaid | 否 | Mermaid 图输出路径 |
| -v, --verbose | 否 | 详细输出模式 |

## Claude Code 工作流程

### 开始工作时
1. 先阅读 `PROGRESS.md` 了解当前进度、待办事项和已知问题
2. 根据用户指示或 PROGRESS.md 中的"下次工作建议"继续工作

### 结束工作时
1. 更新 `PROGRESS.md`：
   - 将完成的任务标记为 `[x]`
   - 添加新发现的问题到"已知问题"
   - 更新"下次工作建议"
   - 更新"最后更新"日期
2. 提交并推送更改：
   ```bash
   git add .
   git commit -m "描述本次工作内容"
   git push
   ```

### 重要原则
- 每次会话结束前必须更新 PROGRESS.md 并推送
- 保持 PROGRESS.md 的准确性，方便其他设备上的 Claude Code 接手
- 遇到重大技术决策时，更新 CLAUDE.md 的"技术决策"部分

## 注意事项

- 需求文档在上级目录: `../requirement.md`
- 测试时不需要 API Key，描述会显示"待分析"
- Windows 环境开发，注意路径分隔符
