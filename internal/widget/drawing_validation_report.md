# goui 组件绘制设计验证报告

**生成日期**: 2026-06-01 17:29  
**编译文件**: `guitest.exe` (3.7MB)  
**运行命令**: `go run ./cmd/guitest/` 或直接运行 `./guitest.exe`  
**窗口尺寸**: 680x880，可调整大小  

---

## 一、验证范围

本报告验证 `cmd/guitest/main.go` 中使用的所有 Web 风格组件，以及所有已实现的核心 GUI 组件。

| 组件 | 文件 | 是否在 guitest 中展示 | 绘制行数 |
|------|------|:---:|:---:|
| Div / Container | html.go / basic.go | ✅ | 128-176 (Container.Paint) |
| H1~H4 标题 | html.go (122-171) | ✅ | 359-392 (TextElement.Paint) |
| P 段落 | html.go (176-185) | ✅ | 同上 (TextElement.Paint) |
| Small 小号文本 | html.go (188-197) | ✅ | 同上 |
| Strong 加粗文本 | html.go (200-210) | ✅ | 同上 |
| Button 按钮 | basic.go (536-568) | ✅ | ButtonElement.Paint |
| Checkbox 复选框 | checkbox.go (127-219) | ✅ | CheckboxElement.Paint |
| Input 输入框 | input.go (134-228) | ✅ | InputElement.Paint |
| Separator 分割线 | divider.go (91-118) | ✅ | DividerElement.Paint |
| SpacerDiv 弹性占位 | html.go (218-220) | ✅ | — |
| Card 卡片 | card.go (117-206) | ❌ (仅通过 Div 模拟) | CardElement.Paint |
| Switch 开关 | switch.go (81-125) | ❌ | SwitchElement.Paint |
| RadioButton 单选 | radio.go (82-133) | ❌ | RadioButtonElement.Paint |
| Slider 滑块 | slider.go (93-173) | ❌ | SliderElement.Paint |
| ProgressBar 进度条 | progressbar.go (82-166) | ❌ | ProgressBarElement.Paint |

---

## 二、组件绘制设计验证详情

### 2.1 Container / Div

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 背景色 | 支持纯色背景 | `Background.Color != nil` 时填充 `DrawRect` | ✅ |
| 边框 | 四边独立颜色和宽度 | 分别绘制上/下/左/右边线 | ✅ |
| 内边距 | 子控件偏移 padding 后绘制 | `childOffset = pos + padding` | ✅ |
| 固定尺寸 | Width/Height 可控制大小 | Layout 中约束 | ✅ |
| 圆角 | 支持 BorderRadius | **已修复**: Container.BorderRadius + DrawRoundedRect + ClipRect 子内容裁剪 | ✅ |

> **修复**: Container 已添加 `BorderRadius` 字段，Paint 方法在 `br>0` 时使用 `DrawRoundedRect` 绘制背景，边框使用 `DrawRoundedRect(Stroke)`，并使用 `Save/ClipRect/Restore` 裁剪子内容防止溢出。Div() 和 applyToContainer 已传递 BorderRadius。

---

### 2.2 Text / H1~H4 / P / Small / Strong

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 多行换行 | 按宽度自动换行 | `splitLines()` 实现单词换行 | ✅ |
| 基线定位 | 使用字体 ascent 正确定位 | `y = pos.Y + lineIdx*lineH + ascent` | ✅ |
| 颜色 | 各自预设独立颜色 | H1=#212121, P=#3C3C3C, Small=#808080 | ✅ |
| 字号 | H1=28, H2=22, H3=18, H4=16, P=14, Small=12 | 正确设置 | ✅ |
| 字重 | 标题 Bold，正文 Regular | ✅ | ✅ |
| 对齐方式 | 支持左/中/右对齐 | Align 字段已定义，但 Paint 中始终左对齐 | ⚠️ |

> **说明**: Text.Align 字段存在，但在 Paint() 中未根据 Align 调整绘制 X 坐标。当前始终从 pos.X 开始左对齐绘制。

---

### 2.3 Button

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 背景 | 主题色圆角矩形 (radius=4) | `DrawRoundedRect(x, y, w, h, 4, bgPaint)` | ✅ |
| 文本颜色 | 白色 | `types.ColorWhite` | ✅ |
| 文本居中 | 水平+垂直居中 | `(容器宽-文本宽)/2` + 基线计算 | ✅ |
| 悬停效果 | 悬停时颜色变深 | **已修复**: `NewButton` 设置 `HoverColor` | ✅ |
| 交互反馈 | 按下时触发 OnClick | `TypeMouseDown/Up` 处理 | ✅ |

> **修复**: `NewButton()` 已添加 `HoverColor: t.Button.HoverColor`，按钮悬停时颜色从 `#4285F4` 变为 `#3878DC`。

---

### 2.4 Checkbox

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 选中态 | 主题色填充 + 白色勾 ✓ | ActiveColor 填充 + 白线（2笔路径） | ✅ |
| 未选中态 | 白底 + 灰色边框 | InactiveBgColor 填充 + BorderColor 边框 | ✅ |
| 悬停效果 | 未选中时显示浅色背景 | HoverBgColor 填充 + HoverBorderColor 边框 | ✅ |
| 勾选标记 | 标准 ✓ 形状 | 两条折线: 左下→中→右上 | ✅ |
| 标签文字 | 主题色，方框右侧8px | `textPaint.Color = LabelColor`，`textX = boxX+boxSize+8` | ✅ |
| 标签基线 | 与方框垂直居中对齐 | `centerY ± ascent-lineH/2` 正确定位 | ✅ |

> **说明**: Checkbox 是绘制最完善的组件之一。✓ 标记使用粗线（2.5px）清晰可见，标签文字通过字体 ascent 准确对齐。

---

### 2.5 Input

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 背景 | 白色填充 | `BGColor` 填充矩形 | ✅ |
| 边框 | 灰色，聚焦变主题色 | `BorderColor` / `FocusBorderColor` | ✅ |
| 占位文本 | 灰色，在无输入时显示 | `PlaceholderColor` + 颜色区分 | ✅ |
| 光标 | 竖线闪烁，每30帧一次 | `DrawLine` + goroutine 切换可见性 | ✅ |
| 文本输入 | 支持键盘输入 | `TypeKeyChar/KeyDown` 处理 | ⚠️ |

> **问题**: Input 的文本输入处理在 `state_test.go` 中有 7 个测试用例失败，表明事件处理存在较老的问题（可能是 StatefulWidget 重构导致的事件传递变化）。但这些不影响静态绘制验证。

---

### 2.6 Separator / Divider

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 水平线 | 灰色水平线，宽度占满容器 | `DrawLine(x1, y, x2, y)` | ✅ |
| 间距 | 上下有 margin 空间 | Layout 计算 margin | ✅ |
| 缩进 | 可选起始/末端缩进 | Indent/EndIndent 支持 | ✅ |

---

### 2.7 Switch

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 轨道 | 圆角矩形，24x44 | `DrawRoundedRect` (radius=12) | ✅ |
| 滑块 | 白色圆形，半径10px | `DrawCircle` | ✅ |
| 状态切换 | 滑块左右滑动 + 颜色变化 | 位置和颜色根据 Value 变化 | ✅ |
| 标签 | 轨道右侧文字 | `DrawText` | ✅ |

> **注意**: 标签基线使用近似值 `trackY + trackH/2 + font.Size/3`，未使用字体 ascent 计算，可能在不同字号下有微小偏移。

---

### 2.8 RadioButton

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 外圆 | 灰色描边圆（选中时主题色） | `DrawCircle` 描边 (StrokeWidth=2) | ✅ |
| 内圆 | 选中时填充半经圆 | `DrawCircle` 填充 (radius=outer*0.5) | ✅ |
| 悬停 | 边框变深 | `HoverBorderColor` | ✅ |
| 分组互斥 | 同组只能选一个 | `notifyGroup()` 遍历兄弟节点 | ✅ |

> **说明**: RadioButton 实现完整，标签定位使用近似值 `centerY + font.Size/3`，和 Switch 类似。

---

### 2.9 Slider

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 轨道 | 左右分段着色 | 激活部分/未激活部分分别绘制 | ✅ |
| 滑块 | 圆形，可拖拽 | `DrawCircle` | ✅ |
| 悬停/拖拽 | 滑块外围半透明圈 | `DrawCircle` (stroke, alpha=60) | ✅ |
| 值标签 | 滑块上方显示当前值 | `DrawText` 居中于滑块 | ✅ |
| 拖拽交互 | 鼠标拖拽改变值 | `updateValueFromEvent` | ✅ |

---

### 2.10 ProgressBar

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 背景轨道 | 灰色圆角矩 | BgColor 填充 | ✅ |
| 进度填充 | 主题色，按比例宽度 | `filledW = barW * value` | ✅ |
| 不确定模式 | 移动的方块 | `animOffset` 驱动的滑动块 | ✅ |
| 百分比标签 | 可选显示 xx% | `ShowLabel` 控制 | ✅ |

---

### 2.11 Card

| 项目 | 设计意图 | 实际实现 | 一致? |
|------|---------|---------|:---:|
| 背景色 | 白色 | `Color` 填充 | ✅ |
| 圆角 | 8px | `DrawRoundedRect(radius)` | ✅ |
| 阴影 | 多层半透明堆叠模拟模糊 | `layers` 循环绘制 | ✅ |
| 边框 | 可选颜色+宽度 | `BorderColor/BorderWidth` | ✅ |
| 内边距 | 子控件偏移 | `childOffset = pos + padding` | ✅ |

> **说明**: guitest 没有直接使用 Card 组件，而是用 Div + Style 模拟卡片效果。Card 的阴影实现是简化版（多层半透明矩形堆叠），不是真正的模糊阴影。

---

## 三、发现的问题汇总

### 严重问题（已修复）

| # | 问题 | 文件 | 行 | 影响 |
|---|------|------|---|------|
| P1-P2 | Container 圆角支持 | basic.go + html.go | **已修复** | ✅ Container.BorderRadius + Paint 圆角绘制 + Div 传递 |

### 一般问题

| # | 问题 | 文件 | 行 | 影响 |
|---|------|------|---|------|
| P3 | NewButton HoverColor | basic.go | 430-439 | **已修复**，从主题读取 HoverColor |
| P4 | Text.Align 字段未在 Paint 中使用 | basic.go | 359-392 | 对齐设置无效 |
| P5 | Switch/Radio 标签使用近似基线而非 ascent | switch.go:122, radio.go:130 | — | 字号变化时对齐偏差 |
| P6 | Input 文本输入有 7 个测试失败 | state_test.go | 多个 | 功能不完整（预先存在） |
| P7 | Button 文字显示效果不佳 | basic.go:577-593 | **已修复** | 见下方详细说明 |

### P7 修复详情：按钮文字显示效果不佳

**问题分析**：按钮文字在视觉上看起来"显示效果不佳"的原因有三点：

| 问题 | 原因 | 修复方案 |
|------|------|---------|
| ① 文字颜色硬编码为白色 | `ButtonElement.Paint()` 中 `textPaint.Color = types.ColorWhite`，未读取 `ButtonTheme.TextColor` | ✅ 添加 `Button.TextColor` 字段，`NewButton()` 从主题读取 |
| ② 缺少内边距 | `Button.Padding` 字段已定义但 Layout 和 Paint 均未使用，文字紧贴按钮背景边缘 | ✅ Paint 中计算 content area（扣除 padding），在内容区域中居中文字 |
| ③ 字体大小硬编码 | `font.Size = 14` 固定，无可配置性 | ✅ 添加 `Button.FontSize` 字段，默认 14，可通过 `WithFontSize()` 链式调用配置 |

**新增字段**：
- `Button.TextColor` — 从 `ButtonTheme.TextColor` 读取默认值
- `Button.FontSize` — 默认 14px，可通过 `WithFontSize()` 设置
- `Button.Padding` — 默认 `{12, 4, 12, 4}`（左右12px，上下4px）

**新增链式方法**：
- `WithTextColor(color)` — 设置按钮文字颜色
- `WithFontSize(size)` — 设置按钮字体大小  
- `WithPadding(padding)` — 设置按钮内边距

**修改文件**：`internal/widget/basic.go`（NewButton、Button 结构体、Layout、Paint）

### P8 修复详情：字体提示缺失导致文字笔画宽度不均匀

**问题根源**：`internal/canvas/fontutil.go` 中 `loadFontFromFile()` 创建 OpenType face 时使用了 `Hinting: gofont.HintingNone`，导致小字号（14px）下字形笔画未对齐像素网格，产生不均匀的笔画宽度。

**为什么按钮更明显**：
- 按钮文字为白色（RGB 255,255,255）在深蓝色背景上，对比度极高，无提示的字体抗锯齿边缘更明显，视觉上"笔画粗细不均"
- 标签/复选框文字为深灰色（RGB 33,33,33）在浅色/白色背景上，问题被对比度掩盖

**修复**：`fontutil.go:102` `HintingNone` → `HintingFull`

**修改文件**：`internal/canvas/fontutil.go`

---

## 四、设计一致性评估

### 4.1 颜色系统
**状态**: ✅ 一致  
主题系统（`theme.go`）提供 16 种通用颜色 + 8 个组件子主题，所有组件从主题读取默认值。`cmd/guitest/main.go` 中的自定义主题修改后，所有组件颜色统一变化。

### 4.2 字体系统  
**状态**: ⚠️ 部分一致  
Text 和 Input 使用 `canvas.DefaultFont()`，标题使用独立定义（sans-serif + 字号 + Bold）。但所有组件都硬编码了 `Family: "sans-serif"`，没有统一的字体管理。

### 4.3 间距系统
**状态**: ⚠️ 部分一致  
使用 `types.EdgeInsets`/`EdgeInsetsLTRB` 统一管理间距，但无类似 CSS 的 spacing scale（如 `SpaceXS/SM/MD/LG/XL`）。

### 4.4 阴影
**状态**: ⚠️ 简化实现  
Card 使用多层半透明矩形堆叠模拟阴影，不是真正的模糊阴影。但在此类自绘引擎中属常见简化方案。

---

## 五、如何手动验证

1. **编译运行**：
   ```powershell
   cd F:\syproject\goui
   go run ./cmd/guitest/
   ```

2. **直接运行已编译的 exe**：
   ```powershell
   cd F:\syproject\goui
   .\guitest.exe
   ```

3. **预期看到的界面内容**（从上到下）：
   - 标题区：goui 组件测试（H1）
   - 卡片1：按钮与复选框（H2）
     - 三个按钮：确定 / 取消 / 危险操作
     - 三个复选框：显示完成状态(选中) / 记住登录信息 / 同意用户协议
   - 卡片2：输入框（H2）
     - 两个输入框：用户名 / 密码
     - 带标签的行内输入：邮箱
   - 卡片3：排版系统（H2）
     - H1~H4 标题
     - P 段落 / Small 小号文本 / Strong 强调文本
   - 卡片4：CSS 风格样式系统（H2）
     - 高亮提示卡片
     - 内联样式卡片（浅蓝背景）
   - 卡片5：分隔线与间距（H2）
     - 水平分割线
     - HBox 左右布局 (SpacerDiv 弹性占位)
   - 底部成功提示卡片

4. **对比检查清单**：
   - [ ] 所有卡片背景是白色，右侧有轻微边框阴影
   - [ ] 按钮有主题色（蓝色）背景
   - [ ] 选中复选框：蓝色方框 + 白色 ✓
   - [ ] 未选中复选框：白色方框 + 灰色边框
   - [ ] 输入框：白色背景 + 灰色边框
   - [ ] 标题字号大小递减层级：H1→H4
   - [ ] 水平分割线为灰色细线
   - [ ] 间隔和边距一致、不拥挤

---

## 六、构建验证摘要

| 验证项 | 状态 | 说明 |
|-------|:---:|------|
| `go build ./cmd/guitest/` | ✅ | 成功 -> guitest.exe (3.7MB) |
| `go build ./internal/...` | ✅ | 全部通过 |
| `go vet ./internal/widget/...` | ✅ | 全部通过 |
| `go test ./internal/widget/... -run TestWeb` | ✅ | 30+ 测试全部通过 |
| `guitest.exe` 存在 | ✅ | 已编译 |

---

## 七、修复状态

1. **✅ P1/P2**: Container 圆角裁剪 -- **已修复**
2. **✅ P3**: NewButton HoverColor -- **已修复**
3. **P4**: Text 对齐支持 -- 待完善
4. **P5**: Switch/Radio 基线 -- 待优化

---

*报告结束。用户可运行 `.\guitest.exe` 手动对比实际绘制效果与上述设计描述。*
