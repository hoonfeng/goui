# goui — Go 跨平台原生 UI 库

goui 是用 Go 编写的跨平台原生 UI 库，采用 **类 Flutter 的自绘引擎**：每个控件由库自己用 **Skia（经 CGO）** 绘制，不依赖系统原生控件，实现跨平台 UI 一致性与像素级可控。模块路径 `github.com/user/goui`（Go 1.24.2），代码注释为中文。

> 历史说明：早期版本用 OpenGL 自绘，现已切换到 Skia（`SkiaCanvas`）；OpenGL 仅作为「Skia 渲染结果上屏」的纹理上传通道保留。

## 项目状态

成熟可用（当前平台：Windows / Win32）。已具备：

- **~75 个组件**（对标 Element Plus，覆盖 ~94%）：按钮 / 输入 / 表单 / 表格 / 日期 / 级联 / 树 / 穿梭框 / 上传 / 富文本…… 含 Table 五大增强（排序 / 多选 / 可展开行 / 固定列 / 固定表头）、Select 多选搜索清除、Form 联动校验、富文本 WYSIWYG（粗斜下划删除线 / 字号 / 颜色 / 对齐 / 图片 / **撤销重做**）。
- **声明式 UI**：JSON 配置 / Go 结构体驱动，**86 个组件 type 全覆盖**（`LoadConfig` / `LoadConfigFile` / `BuildFromSpec` + `RegisterComponent` 注册自定义组件）。
- **主题与配色**：全站 Theme（设计 token）+ 一键换主色（`SetPrimaryColor`）+ **深色模式**（`DarkTheme()`）+ **局部差异化**（`ConfigProvider` 子树主题作用域）。
- **i18n 国际化**：翻译 / 复数 / RTL / 数字 / 日期本地化 + 运行时切换语言 + JSON 语言包加载（`i18n.LoadLocaleFile`）。
- **动画引擎**：Easing 全套 / Tween / Spring 物理弹簧 + 60fps 帧驱动（空闲阻塞省电）。
- **代码编辑器**（对标 Monaco）：语法高亮 / 多光标 / 查找替换 / 折叠 / minimap / 撤销重做 + 接 **gopls** 语义补全与诊断。
- **StructEditor**：代码表格化编辑器（导入 / 程序集变量 / 函数·参数·返回值·局部变量表格化 + 代码区）+ Go 双向（`ParseGo`/`ToGo`，import / 指针参数↔参考 / `:=` 类型推导 / 注释 / 空行保留）+ 去语言化的 `LanguageProvider`。

## 构建前提（重要——非显而易见）

渲染后端是 **Skia via CGO**，几乎所有包都传递性依赖 `internal/canvas`：

1. **`CGO_ENABLED=1`** + C 工具链（gcc/mingw）在 `PATH`。
2. 同级仓库 **`goskia`** 须存在（`go.mod` 里 `replace github.com/hoonfeng/goskia => ../goskia`）。
3. **`libSkiaSharp.dll`**（仓库根的原生 Skia 库）须在运行时 DLL 搜索路径（工作目录或 exe 旁）——故预编译 exe 与 DLL 同放仓库根。
4. **`fonts/`** 须在运行时可达（exe 旁或 cwd）。
5. **当前仅 Windows / Win32 可用**；`x11`（Linux）/ `cocoa`（macOS）在 build tag 后但尚未功能化。

## 快速开始（PowerShell）

```powershell
$env:CGO_ENABLED='1'   # 下面每条都需要

# 组件测试应用（Web 式组件 + 主题 + CSS）——从仓库根运行（需 DLL + fonts/）
go run ./cmd/guitest/

# 声明式 JSON 驱动 + Web 组件 demo
go run ./examples/demo/

# headless 渲染到 PNG（无窗口，可视验证主路径）
go run ./examples/skiawidget/   # 完整 Widget→Element→Layout→Paint→Skia 管线
go run ./examples/skiapaint/    # 原始 SkiaCanvas 绘制

# 测试（同样需要 CGO + DLL + goskia 同级仓库）
go test ./internal/...
```

## 架构

分层（上到下）：**app → widget → layout → render → canvas → window/platform**，`event` 与 `types` 横切。

- **`widget`**：`Widget`（不可变配置）/ `Element`（可变运行时）分离（Flutter 模型）；`StatefulWidget` + `State`（`SetState` → 重绘）。HTML 式构造（`Div/H1/P`…）+ CSS 式 `Style`/`Define` + `Theme` 设计 token + 声明式 JSON。
- **`layout`**：`BoxConstraints` 约束系统（父传约束、子返尺寸）。
- **`render`**：`Pipeline` 拥有根 Element 与最终 Canvas（Build→Layout→Paint→Flush，`needsLayout`/`needsRepaint` 脏标记）。
- **`canvas`**：`Canvas` 接口；主实现 **`SkiaCanvas`**（goskia 光栅面渲染进 `image.RGBA`）。
- **`window`**：平台接口；`win32` 可用（raw `syscall` 到 user32/gdi32/opengl32/imm32）。

## 声明式 / Web 式 API

```go
ui, _ := widget.LoadConfig([]byte(`{
  "type": "Column",
  "style": {"gap": 12, "padding": "16"},
  "children": [
    {"type": "H1", "text": "Hello"},
    {"type": "Button", "text": "OK", "events": {"click": "onOk"}}
  ]
}`), widget.Handlers{"onOk": func(ctx widget.EventContext) { /* ... */ }})
```

主题 / 换肤 / 深色 / 局部差异化：

```go
widget.SetTheme(widget.DarkTheme())             // 整体深色
widget.SetPrimaryColor(green)                    // 一键换主色
widget.NewConfigProvider(panel).WithTheme(dark)  // 仅这块子树深色
```

## 子目录说明

- `cmd/guitest`、`examples/demo`、`examples/skiapaint`、`examples/skiawidget` 是当前有效示例；其余 `examples/*` 多为各组件/特性的验证 demo。
- `svg/` 是独立的 SVG 渲染库（自带文档），**不属于** 核心 UI 引擎的 import 图。
- `.trae/documents/` 是 PRD / 架构规划文档（背景参考）。
