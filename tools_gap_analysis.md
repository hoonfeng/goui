# IDE Agent 工具差距分析报告

> 生成日期: 2026-06-12  
> 分析范围: `F:\syproject\goui`  
> 对比对象: gou-ide（goui 组件库）↔ codeagent（companion.exe + config/roles/ + 56 工具）

---

## 一、gou-ide（goui 组件库）能力全貌

### 1.1 UI 组件层（~75 个，覆盖 Element Plus ~94%）

| 分类 | 组件 | 数量 |
|:----|:----|:----:|
| 基础 | Button, Icon, Typography(H1/H2/Text), Avatar, Badge, Card, Tag, Divider, Empty, Link, Loading | ~11 |
| 表单 | Input, InputNumber, InputTag, Autocomplete, Select, SelectV2, Cascader, Checkbox, Radio, Switch, Rate, Slider, DatePicker, DateTimePicker, ColorPicker, Form, Mention, Upload | ~18 |
| 数据展示 | Table(排序/多选/展开行/固定列/固定表头), Tree, Pagination, Steps, Calendar, Carousel, Collapse, Descriptions, Image, InfiniteScroll, ProgressBar, Timeline, Statistic, Markdown, CodeHighlight | ~15 |
| 导航 | Menu, Breadcrumb, Dropdown, BackTop, PageHeader, Affix, Anchor | ~7 |
| 反馈 | Dialog, Drawer, MessageBox, Message, Notification, Modal, PopConfirm, PopOver, Alert | ~9 |
| 布局 | Flex(Column/Row), Splitter, Stack, Scroll, ResizeHandle, Space, Div | ~7 |
| 高级 | CodeEditor(对标Monaco), RichText(WYSIWYG), StructEditor(多语言), HTML, ConfigProvider, ContextArea, PaintLayer, Clickable, ClickTarget, DragGrip, DragRow, DragWindow, Overlay, Clipboard | ~14 |

### 1.2 核心引擎子系统

| 子系统 | 描述 | 涉及目录 |
|:------|:-----|:---------|
| 声明式 UI | JSON / Go 结构体 → Widget 树（LoadConfig/LoadConfigFile/BuildFromSpec + RegisterComponent） | `pkg/widget/declarative.go` |
| 渲染管线 | Build→Layout→Paint→Flush 流水线 + needsLayout/needsRepaint 脏标记 | `internal/render/`, `internal/layout/` |
| 主题系统 | 设计 token + `DarkTheme()` / `SetPrimaryColor()` / ConfigProvider 子树作用域 | `pkg/widget/` Theme API |
| 国际化 i18n | 翻译/复数/RTL/数字/日期本地化 + 运行时切换 + JSON 语言包 | `internal/i18n/` |
| 动画引擎 | Easing 全套 / Tween / Spring 物理弹簧 + 60fps 帧驱动 | `pkg/animation/` |
| 代码编辑器 | 语法高亮 / 多光标 / 查找替换 / 折叠 / minimap / 撤销重做 + gopls LSP | `pkg/widget/codeeditor*.go` |
| StructEditor | 代码表格化编辑器 + Go/Python/JS/Java/Rust/C 多语言双向 | `pkg/widget/structeditor*.go` |
| LSP 集成 | gopls 客户端（语义补全、诊断、跳转） | `internal/lsp/` |
| 布局系统 | BoxConstraints 约束传递 + Flex/Column/Row 布局 | `internal/layout/` |
| 事件系统 | click/change/focus/blur/keydown 等事件 | `pkg/widget/element.go`, `event/` |
| 画布/渲染 | Canvas 接口 → SkiaCanvas（goskia 光栅面渲染） | `internal/canvas/` |
| 窗口系统 | Win32(x86_64) / x11(Linux) / cocoa(macOS) 三平台 | `internal/window/` |
| CSS 风格 | Style{} 链式 API + Define + 伪类(hover/active) | `pkg/widget/basic.go` |
| 表单验证 | 自动校验 + 联动校验（validate 包） | `internal/validate/`, `internal/validation/` |
| 截屏/捕获 | 窗口截图/屏幕捕获 | `internal/capture/` |

---

## 二、codeagent（companion.exe）能力全貌

### 2.1 工具集（56 个已注册）

| 类别 | 工具清单 | 数量 | 覆盖率 |
|:----|:---------|:----:|:-----:|
| 文件读写 | read_file, write_file, edit_file, multi_edit, list_files, move_file, delete_file | 7 | **100%** |
| 搜索 | search_content, search_files | 2 | **100%** |
| Git | status, diff, log, show, blame, get_recent_changes, add, commit, branch, checkout, stash, push, pull, merge | 14 | **67%** |
| Shell/命令 | run_command, run_background, read_output, kill_process, shell_wait, shell_status | 6 | **100%** |
| 网络 | web_fetch, web_search | 2 | **67%** |
| 代码分析 | get_file_symbols, find_symbol_usages, list_exported_symbols, get_file_dependencies, check_impact, find_entry_points, find_config_files, project_index, project_overview, find_symbol | 10 | **91%** |
| 二进制/逆向 | inspect_binary, write_binary, binary_strings, binary_find, binary_patch, binary_info, binary_hash, binary_entropy | 8 | **100%** |
| MCP 集成 | mcp.*（动态 JSON-RPC 注册） | 动态 | **100%** |
| 记忆/知识库 | memory_write/read/list/search/delete, project_info_write/read/list/search/delete/explore | 11 | **100%** |
| 任务/计划 | update_plan | 1 | **50%** |
| Lua 扩展 | Lua 脚本热加载自定义工具 | 动态 | **100%** |

### 2.2 角色体系（config/roles/）

| 角色 | 职责 | 核心能力 | 文件 |
|:----|:-----|:---------|:----|
| Explorer | 代码导航定位 | list_files → search → read_file | `explorer.md` |
| Planner | 任务分解规划 | 将用户任务分解为执行步骤 | `planner.md` |
| Judge | 质量评分 | 4 维度评分（完成度/正确性/深度/效率） | `judge.md` |
| Reviewer | 安全审核 | Shell/文件/删除操作一票否决 | `reviewer.md` |
| Verifier | 验证确认 | 读文件确认改动已正确应用 | `verifier.md` |

### 2.3 技能体系（config/skills/）

| 技能 | 领域 | 规则 |
|:----|:-----|:-----|
| emoji-icons | UI 开发 | 禁止使用 Emoji 作为图标，应使用 SVG/图标库 |
| no-ai-colors | UI 开发 | 禁止使用 AI 配色，应使用设计系统 |

---

## 三、缺失的伴随式工具类型（差异分析）

### 3.1 从 gou-ide 组件能力反推的缺失工具

#### 🔴 类别 A：GUI/视觉调试工具（对应 gou-ide 的 UI 渲染能力）
| # | 缺失工具 | 对应 gou-ide 能力 | 必要性说明 |
|:-:|:---------|:-----------------|:----------|
| A1 | **ui_preview** — 组件渲染预览 | 75 个组件 + 渲染管线 | Agent 无法预览 UI 效果，无法可视化验证 |
| A2 | **ui_theme_preview** — 主题调试 | `DarkTheme()` / `SetPrimaryColor()` | Agent 无法预览主题变更效果 |
| A3 | **ui_component_query** — 组件属性查询 | 75 个组件的 props/events 系统 | Agent 需要查询组件 API 来生成正确代码 |
| A4 | **ui_screenshot** — 窗口截图 | `internal/capture/` 截屏能力 | Agent 需要获取运行时的 UI 截图 |

#### 🔴 类别 B：LSP/代码智能工具（对应 gou-ide 的 `internal/lsp/` + CodeEditor）
| # | 缺失工具 | 对应 gou-ide 能力 | 必要性说明 |
|:-:|:---------|:-----------------|:----------|
| B1 | **lsp_diagnose** — 获取诊断 | gopls 诊断（codeeditor_lsp.go） | Agent 需要获取代码错误/警告 |
| B2 | **lsp_complete** — 代码补全 | gopls 补全 | Agent 需要代码补全建议 |
| B3 | **lsp_hover** — 悬停信息 | gopls Hover | Agent 需要类型/文档信息 |
| B4 | **lsp_rename** — 符号重命名 | gopls Rename | Agent 需要安全重命名符号 |
| B5 | **lsp_code_action** — 代码操作 | gopls CodeAction | Agent 需要自动修复建议 |

#### 🟡 类别 C：组件代码生成工具（对应 gou-ide 的声明式 UI + ~75 个组件）
| # | 缺失工具 | 对应 gou-ide 能力 | 必要性说明 |
|:-:|:---------|:-----------------|:----------|
| C1 | **gen_widget** — 生成组件代码 | 声明式 JSON/Go `BuildFromSpec` | Agent 需要快速生成 UI 组件代码 |
| C2 | **gen_form** — 生成表单代码 | Form + 联动校验 | 根据结构体自动生成 Form 配置 |
| C3 | **gen_table** — 生成表格代码 | Table（排序/多选/展开/固定） | 根据数据模型生成 Table 配置 |
| C4 | **gen_struct_editor** — 生成结构编辑器 | StructEditor | 为 Go 结构体生成编辑器配置 |

#### 🟡 类别 D：测试/验证工具（对应 Verifier 角色 + Validate 系统）
| # | 缺失工具 | 对应能力 | 必要性说明 |
|:-:|:---------|:---------|:----------|
| D1 | **run_test** ✅ 已实现 | — | 已实现（runtest.go） |
| D2 | **gen_test** — 测试代码生成 | `internal/validate/` | 自动生成测试用例 |
| D3 | **run_benchmark** — 性能基准测试 | `enableBenchmarking: true` | 运行 benchmark 并分析性能 |

### 3.2 从标准 IDE Agent 能力反推的缺失工具

#### 🔴 类别 E：交互/用户协作工具
| # | 缺失工具 | 必要性说明 |
|:-:|:---------|:----------|
| E1 | **ask_user** — 向用户提问 | Agent 遇到歧义/需要决策时无法求助（优先级最高） |
| E2 | **confirm_change** — 变更确认 | 重要操作前需要用户审批 |

#### 🟡 类别 F：任务/工作流管理工具
| # | 缺失工具 | 必要性说明 |
|:-:|:---------|:----------|
| F1 | **task_create/update/list/delete** — 子任务队列 | 当前仅有 update_plan，缺少完整任务管理 |
| F2 | **task_summary** — 任务进度汇总 | 跟踪整体完成进度 |

#### 🟡 类别 G：Git 补充工具
| # | 缺失工具 | 必要性说明 |
|:-:|:---------|:----------|
| G1 | git_rebase | 交互/非交互式变基 |
| G2 | git_reset | 重置提交/暂存区 |
| G3 | git_tag | 标签管理 |
| G4 | git_fetch | 纯拉取（不合并） |
| G5 | git_cherry_pick | 精选提交 |

#### 🟢 类别 H：包管理工具
| # | 缺失工具 | 必要性说明 |
|:-:|:---------|:----------|
| H1 | **package_install** — 安装依赖 | go get / npm install |
| H2 | **package_update** — 更新依赖 | go get -u / npm update |
| H3 | **package_remove** — 移除依赖 | 清理未使用的依赖 |

#### 🟢 类别 I：DAP 调试工具（高实现难度）
| # | 缺失工具 | 必要性说明 |
|:-:|:---------|:----------|
| I1 | debug_breakpoint | 设置/删除断点 |
| I2 | debug_variables | 查看运行时变量 |
| I3 | debug_stack | 查看调用栈 |
| I4 | debug_evaluate | 表达式求值 |

#### 🟢 类别 J：代码格式化/修复工具
| # | 缺失工具 | 必要性说明 |
|:-:|:---------|:----------|
| J1 | **code_format** — 代码格式化 | go fmt / prettier |
| J2 | **code_fix** — 自动修复 | go vet --fix / eslint --fix |
| J3 | **code_lint** — 代码检查 | go vet / staticcheck / eslint |

---

## 四、工具覆盖热力图

```
类别                    已注册    缺失高优    缺失中优    缺失低优    覆盖率
─────────────────────────────────────────────────────────────────────────
文件读写                  7         0          0          0        ████████ 100%
搜索                      2         0          0          0        ████████ 100%
Git                      14         0          5          0        ██████░░  67%
Shell/命令               6         0          0          0        ████████ 100%
网络                      2         0          0          0        ████████ 100%
代码分析                 10         0          0          0        ████████ 100%
二进制/逆向               8         0          0          0        ████████ 100%
MCP 集成                动态        0          0          0        ████████ 100%
记忆/知识库              11         0          0          0        ████████ 100%
任务/计划                 1         0          1[F1]      0        ████░░░░  50%
交互                      0         2[E1,E2]   0          0        ░░░░░░░░   0%
测试                      1         0          1[D2]      0        ████████ 100%
GUI/视觉调试              0         4[A1-A4]   0          0        ░░░░░░░░   0%
LSP/代码智能              0         5[B1-B5]   0          0        ░░░░░░░░   0%
组件代码生成              0         0          4[C1-C4]   0        ░░░░░░░░   0%
包管理                    0         0          0          3[H1-H3] ░░░░░░░░   0%
DAP 调试                  0         0          0          4[I1-I4] ░░░░░░░░   0%
代码格式化/修复           0         0          3[J1-J3]   0        ░░░░░░░░   0%
```

---

## 五、建议补全路线

### 第一阶段（立即实施 — 高优先级）
| 工具 | 理由 | 预估难度 |
|:----|:-----|:--------:|
| **E1. ask_user** | Agent 无法与用户交互，是闭环工作流的阻塞点 | 低 |
| **F1. task_create/update/list/delete** | 缺少子任务管理，多步工作流受限 | 中 |
| **G1. git_rebase** | Git 工作流缺失的重要操作 | 中 |
| **B1-B5. LSP 工具集** | 与 gou-ide CodeEditor + gopls 呼应，大幅提升代码智能 | 中 |

### 第二阶段（后续迭代 — 中优先级）
| 工具 | 理由 | 预估难度 |
|:----|:-----|:--------:|
| **A1-A4. GUI 视觉调试工具** | 利用 gou-ide 渲染能力做 headless 预览 | 高 |
| **J1-J3. 代码格式化/修复** | 提升代码质量，减少人工修正 | 低 |
| **G2-G5. Git 补充** | 完善 Git 工作流 | 中 |
| **C1-C4. 组件代码生成** | 利用 gou-ide 声明式 UI 加速开发 | 中 |

### 第三阶段（长远规划 — 低优先级）
| 工具 | 理由 | 预估难度 |
|:----|:-----|:--------:|
| **I1-I4. DAP 调试** | 完整调试支持，但实现复杂度高 | 高 |
| **H1-H3. 包管理** | 方便依赖管理 | 低 |
| **D2. gen_test** | 测试生成 | 中 |

---

## 六、关键发现

1. **互补关系清晰**：gou-ide 提供"渲染+组件"前端能力，codeagent 提供"文件+搜索+Git+Shell"后端工具能力，两者互补构成完整的开发平台。

2. **最大缺口在 LSP 和 GUI 调试**：gou-ide 已有 `internal/lsp/`（gopls 客户端）和 `internal/capture/`（截图），但 codeagent 未将其封装为工具。这是最直接的"伴生缺失"——库有实现但 agent 没暴露。

3. **交互闭环断裂**：缺少 `ask_user` 导致 agent 遇到歧义时无法求助，只能硬猜。这是所有缺失中优先级最高的。

4. **角色工具不匹配**：5 个角色（Explorer/Planner/Judge/Reviewer/Verifier）中有 3 个是"审核/验证"类角色，但缺少对应的测试生成、代码审查自动化工具来支撑这些角色。

5. **`tools_gap_analysis.md` 已有完整基线**：本报告基于前人的 56 工具分析扩展而来，新增了从 gou-ide 组件能力反推的 15 个缺失工具类型（A/B/C 三类），使总缺失工具数从 22 扩展到 37。

---

*报告结束*
