# Tools 包 — Agent 工具函数库

`tools/` 包提供基础工具函数，供 Agent 在工作流中调用。每个工具独立可测，遵循统一的错误处理风格。

## 工具清单

### 文件操作

| 函数 | 说明 | 参数 |
|------|------|------|
| `ReadFile(path, offset, limit)` | 读取文件内容，支持按行偏移/限制读片段 | `path` 必填, `offset`/`limit` 可选 |
| `WriteFile(path, content)` | 写入文件（覆盖），自动创建父目录 | `path`, `content` 必填 |
| `DeleteFile(path)` | 删除单个文件（不删目录） | `path` 必填 |
| `ListDir(path, pattern)` | 列出目录内容，支持 glob 模式过滤 | `path` 必填, `pattern` 可选 |

#### 示例

```go
// 读取文件全文
content, err := tools.ReadFile("main.go", 0, 0)
if err != nil { /* 处理错误 */ }

// 读取第5-10行
lines, err := tools.ReadFile("main.go", 5, 6)

// 写入文件
result, err := tools.WriteFile("output.txt", "hello world")

// 列出 Go 文件
listing, err := tools.ListDir(".", "*.go")

// 删除文件
result, err := tools.DeleteFile("tmp/test.txt")
```

### 命令执行

| 函数 | 说明 | 参数 |
|------|------|------|
| `RunCommand(command, cwd, timeoutSec)` | 执行 shell 命令，返回 UTF-8 输出 | `command` 必填, `cwd`/`timeoutSec` 可选 |

#### 示例

```go
// 执行命令（默认120s超时）
output, err := tools.RunCommand("go build ./...", "", 0)

// 带工作目录和超时
output, err := tools.RunCommand("npm test", "./frontend", 60)
```

### 工具函数

| 函数 | 说明 | 参数 |
|------|------|------|
| `OpenBrowser(url)` | 在系统默认浏览器中打开 URL（跨平台） | `url` 必填 |
| `EventSimulate(eventType, target, params)` | 模拟 UI 事件（click/input/keydown 等） | `eventType`, `target` 必填 |

#### 示例

```go
// 打开浏览器
result, err := tools.OpenBrowser("https://example.com")

// 模拟点击
result, err := tools.EventSimulate("click", "#submit-btn", map[string]any{
    "x": float64(100),
    "y": float64(50),
})

// 模拟输入
result, err := tools.EventSimulate("input", "#name-input", map[string]any{
    "value": "test user",
})
```

## 支持的事件类型

| 事件 | 说明 | 典型参数 |
|------|------|----------|
| `click` | 鼠标点击 | x, y, button, modifiers |
| `focus` | 获得焦点 | — |
| `blur` | 失去焦点 | — |
| `keydown` | 按下按键 | key, modifiers |
| `keyup` | 释放按键 | key, modifiers |
| `input` | 输入文本 | value |
| `change` | 值变更 | value |
| `hover` | 鼠标悬停 | x, y |
| `scroll` | 滚动 | x, y |
| `resize` | 大小变更 | — |
| `drag` | 开始拖拽 | x, y, button |
| `drop` | 释放拖拽 | x, y |

## 运行测试

```bash
# 运行所有工具测试
go test ./tools/ -v

# 运行特定测试
go test ./tools/ -run TestReadFile -v
```
