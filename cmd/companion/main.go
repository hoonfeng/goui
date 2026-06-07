// 伴随式 CodeAgent —— goui 重构。入口 + 主窗壳（标题栏 / 停靠区 / 状态栏）。
// 详见同目录 AGENTS.md（开发铁律）。
//
//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/user/goui/cmd/companion/state"
	"github.com/user/goui/internal/app"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"

	_ "github.com/user/goui/internal/platform"
)

var application *app.Application

// VS Code Dark+ 风格配色（IDE 深色）。
var (
	cTitle   = types.ColorRef(60, 60, 61)  // 标题栏 #3c3c3d
	cSide    = types.ColorRef(37, 37, 38)  // 侧栏/面板 #252526
	cEditor  = types.ColorRef(30, 30, 30)  // 编辑区 #1e1e1e
	cStatus    = types.ColorRef(0, 122, 204) // 强调色：分隔条 hover/拖动高亮 #007acc
	cStatusBar = types.ColorRef(45, 45, 48)  // 状态栏底色（深灰、低调，不用刺眼蓝）
	cBorder    = types.ColorRef(45, 45, 45)  // 分隔线 #2d2d2d
	cText    = types.ColorFromRGB(204, 204, 204)
	cTextDim = types.ColorFromRGB(140, 140, 140)
)

const (
	titleBarH = 36
	statusH   = 26
	toggleW   = 40 // 标题栏面板开关按钮宽
	winBtnW   = 46 // 标题栏窗口按钮宽
	dividerW  = 4  // 停靠分隔条粗细（细、低调；hover/拖动显蓝）
)

func main() {
	if os.Getenv("COMPANION_SHOT") != "" {
		renderShot()
		return
	}
	loadSettings() // 读用户配置目录的设置（LLM 服务商/Key/模型），供 buildProvider 用

	application = app.NewApplication()
	application.SetRootWidget(&Shell{})

	// 统一所有菜单（右键上下文菜单 / 编辑器右键 / 标题栏下拉）为 GitHub 深色，匹配应用主题。
	widget.SetMenuTheme(*ghBgTertiary, ghText, *ghAccentEmph, *ghBorder, ghTextMuted)
	// 对话框/抽屉也统一深色（设置面板、新建/重命名输入框、确认框等），否则白底面板与深色内容冲突。
	widget.SetDialogTheme(*ghBgSecondary, ghText, ghTextMuted)

	// Ctrl+S 保存当前编辑器标签（全局快捷键，优先于焦点 Widget）。VK_S=0x53。
	application.ShortcutManager.Register(0x53, event.ModCtrl, func() { theEditor.save() }, "Ctrl+S 保存")
	application.ShortcutManager.Register(0x46, event.ModCtrl, func() { theChatState.toggleSearch() }, "Ctrl+F 搜索对话") // VK_F
	application.ShortcutManager.Register(0xBC, event.ModCtrl, func() { openSettings() }, "Ctrl+, 打开设置")               // VK_OEM_COMMA

	application.Ready = func() {
		// 标题栏命中区：顶部 titleBarH 高、右侧 6 个按钮（3 面板开关 + 3 窗口）宽除外 → 系统接管拖动/双击最大化。
		application.SetTitleBar(titleBarH, 3*toggleW+3*winBtnW)
		application.EnableWindowEffects() // DWM 阴影 + Win11 圆角
	}

	cfg := app.Config{Title: "伴随式 CodeAgent", Width: 1200, Height: 760, Resizable: true, Borderless: true}
	if err := application.Run(cfg); err != nil {
		fmt.Println("运行失败:", err)
	}
}

// renderShot 无窗口渲染整个主窗壳到 PNG（环境变量 COMPANION_SHOT 触发），用于布局/视觉自检。
func renderShot() {
	shot("companion_shot.png", state.DefaultPanels())
}

func shot(name string, p *state.Panels) {
	const w, h = 1200, 760
	sk := canvas.NewSkiaCanvas(w, h)
	defer sk.Release()
	pipe := render.NewPipeline(w, h, sk)
	pipe.SetRootElement(widget.CreateElementFor(&Shell{initial: p}))
	if err := pipe.Render(); err != nil {
		fmt.Println("render:", err)
		return
	}
	if err := sk.SaveToPNG(name); err != nil {
		fmt.Println("save:", err)
		return
	}
	fmt.Println("✅", name, "已保存")
}

// ─── 主窗壳（有状态）：标题栏 + 主体（左|中|右）+ 状态栏 ───────

type Shell struct {
	widget.StatefulWidget
	initial *state.Panels // 可选初始布局（renderShot 注入不同状态自检；nil=默认全开）
}

func (sh *Shell) CreateState() widget.State {
	p := sh.initial
	if p == nil {
		p = state.DefaultPanels()
	}
	return &shellState{panels: p}
}

type shellState struct {
	widget.BaseState
	panels   *state.Panels // 停靠布局的唯一真相来源
	leftView string        // 左栏当前视图标签："files" / "git"（复刻参考左区多面板 tab 切换）
}

func (s *shellState) Build(ctx widget.BuildContext) widget.Widget {
	return widget.VBox(
		s.titleBar(),
		expand(s.body()),
		statusBar(),
	)
}

// titleBar 自绘标题栏：左 logo+标题（拖动区）| 右 面板开关 ×3 + 窗口按钮 ×3。
func (s *shellState) titleBar() widget.Widget {
	p := s.panels
	// logo（固定宽，不撑开）+ 菜单（嵌在标题栏）+ 弹性拖动空白 + 面板开关 + 窗口按钮。
	// 菜单/按钮经 ClickTarget 命中判定为可点(HTCLIENT)，空白区交系统拖动(HTCAPTION，见 app.SetTitleBar)。
	kids := []widget.Widget{
		widget.Div( // app 图标徽标（仅图标、无文字，复刻参考 icon.svg 的 Pair 标志）
			widget.Style{Height: titleBarH, Padding: types.EdgeInsetsLTRB(10, 0, 4, 0),
				FlexDirection: "row", AlignItems: "center"},
			pairLogo(),
		),
	}
	kids = append(kids, s.titleMenus()...)
	// 居中标题：两侧等弹性占位夹住标题 → 居中（复刻参考的居中 "Pair CodeAgent"）；占位空白兼作拖动区。
	kids = append(kids, expand(widget.Div(widget.Style{})))
	kids = append(kids, label("Pair CodeAgent", cText, 12))
	kids = append(kids, expand(widget.Div(widget.Style{})))
	kids = append(kids,
		s.toggleBtn("panel-left", state.ZoneLeft, p.Left),
		s.toggleBtn("panel-bottom", state.ZoneBottom, p.Bottom),
		s.toggleBtn("panel-right", state.ZoneRight, p.Right),
		winButton("minus", func() { application.Minimize() }),
		winButton(maxRestoreIcon(), func() { application.ToggleMaximize() }),
		winButton("x", func() { application.Close() }, winCloseRed),
	)
	return widget.Div(
		widget.Style{Height: titleBarH, BackgroundColor: cTitle, FlexDirection: "row", AlignItems: "center"},
		kids,
	)
}

// toggleBtn 面板开关：图标亮=展开、暗=收起，点击翻转该停靠区。
func (s *shellState) toggleBtn(icon string, z state.Zone, on bool) widget.Widget {
	col := cTextDim
	if on {
		col = cText
	}
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Lucide(icon, widget.IconSize(16), widget.IconColor(col))},
		OnClick:           func() { s.panels.Toggle(z); s.SetState() },
		Color:             *cTitle,
		MinWidth:          toggleW,
		MinHeight:         titleBarH,
	}
}

// winCloseRed 关闭键悬停红（复刻参考 TitleBar 的 #e81123）。
var winCloseRed = types.ColorFromHex("#e81123")

// winButton 标题栏窗口按钮；可选 hover 传悬停底色（关闭键传红，其余省略＝自动变暗）。
func winButton(icon string, onClick func(), hover ...types.Color) widget.Widget {
	b := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Lucide(icon, widget.IconSize(15), widget.IconColor(cText))},
		OnClick:           onClick,
		Color:             *cTitle, // 同标题栏
		MinWidth:          winBtnW,
		MinHeight:         titleBarH,
	}
	if len(hover) > 0 {
		b.HoverColor = hover[0]
	}
	return b
}

// maxRestoreIcon 据最大化状态切「最大化(单方块)/还原(叠层方块)」图标（复刻参考）。
func maxRestoreIcon() string {
	if application != nil && application.IsMaximized() {
		return "minimize" // 还原（复刻参考用 lucide Minimize：角内收）
	}
	return "square" // 最大化：单方块
}

// titleMenus 标题栏内嵌菜单——**1:1 复刻参考源 TitleBar.tsx 的菜单结构**（文件/编辑/视图/终端/Agent/帮助，
// 项/顺序/分隔/快捷键/勾选照搬）。companion 暂无对应功能的项置灰 Disabled（诚实保真，非删项）。
// 嵌在标题栏内可点靠 ClickTarget 命中判定（见 app.SetTitleBar / win32 NCHITTEST）。
func (s *shellState) titleMenus() []widget.Widget {
	p := s.panels
	return []widget.Widget{
		menuBarBtn("文件", []widget.DropdownItem{
			{Label: "新建项目", Disabled: true},
			{Label: "新建文件", Shortcut: "Ctrl+N", Command: "file.new"},
			{Label: "打开文件", Shortcut: "Ctrl+O", Command: "file.open"},
			{Label: "打开文件夹", Shortcut: "Ctrl+K Ctrl+O", Command: "file.openFolder"},
			{Label: "添加项目到工作区", Shortcut: "Ctrl+Shift+O", Disabled: true},
			{Label: "保存", Shortcut: "Ctrl+S", Command: "file.save", Divided: true},
			{Label: "保存工作区", Disabled: true, Divided: true},
			{Label: "管理工作区...", Disabled: true},
			{Label: "关闭项目", Disabled: true, Divided: true},
			{Label: "关闭工作区", Disabled: true},
			{Label: "退出", Shortcut: "Alt+F4", Command: "file.quit", Divided: true},
		}, s.onFileMenu),
		menuBarBtn("编辑", []widget.DropdownItem{
			{Label: "撤销", Shortcut: "Ctrl+Z", Command: "edit.undo"},
			{Label: "重做", Shortcut: "Ctrl+Y", Command: "edit.redo"},
			{Label: "剪切", Shortcut: "Ctrl+X", Command: "edit.cut", Divided: true},
			{Label: "复制", Shortcut: "Ctrl+C", Command: "edit.copy"},
			{Label: "粘贴", Shortcut: "Ctrl+V", Command: "edit.paste"},
			{Label: "跨文件搜索", Shortcut: "Ctrl+Shift+F", Command: "edit.searchfiles", Divided: true},
			{Label: "对话内搜索", Shortcut: "Ctrl+F", Command: "edit.chatsearch"},
		}, s.onEditMenu),
		menuBarBtn("视图", []widget.DropdownItem{
			{Label: "专注模式", Shortcut: "Ctrl+K", Disabled: true},
			{Label: "文件树", Shortcut: "Ctrl+B", Command: "view.files", Checked: p.Left && s.leftView != "git", Divided: true},
			{Label: "搜索", Command: "view.search", Checked: p.Left && s.leftView == "search"},
			{Label: "Git", Command: "view.git", Checked: p.Left && s.leftView == "git"},
			{Label: "对话", Shortcut: "Ctrl+Shift+C", Command: "view.chat", Checked: p.Right},
			{Label: "终端", Shortcut: "Ctrl+J", Command: "view.terminal", Checked: p.Bottom},
			{Label: "放大", Shortcut: "Ctrl+=", Disabled: true, Divided: true},
			{Label: "缩小", Shortcut: "Ctrl+-", Disabled: true},
			{Label: "切换 Minimap", Disabled: true},
			{Label: "导出当前对话", Command: "view.export"},
		}, s.onViewMenu),
		menuBarBtn("终端", []widget.DropdownItem{
			{Label: "新建 CMD", Command: "term.cmd"},
			{Label: "新建 PowerShell", Disabled: true},
			{Label: "新建 Git Bash", Disabled: true},
		}, s.onTerminalMenu),
		menuBarBtn("Agent", []widget.DropdownItem{
			{Label: "Agent 监控面板", Shortcut: "Ctrl+Shift+M", Disabled: true},
			{Label: "性能监控", Disabled: true, Divided: true},
			{Label: "进化图（EvoMap）", Disabled: true},
			{Label: "探索项目知识库", Disabled: true, Divided: true},
		}, func(string) {}),
		menuBarBtn("帮助", []widget.DropdownItem{
			{Label: "扩展市场", Disabled: true},
			{Label: "打开设置", Shortcut: "Ctrl+,", Command: "help.settings"},
			{Label: "更新日志", Disabled: true, Divided: true},
			{Label: "关于", Command: "help.about", Divided: true},
			{Label: "开发者工具", Shortcut: "F12", Disabled: true},
		}, s.onHelpMenu),
	}
}

func (s *shellState) onFileMenu(cmd string) {
	switch cmd {
	case "file.new":
		newEntryIn(theFileTree.rootPath, false)
	case "file.open":
		openFileViaDialog()
	case "file.openFolder":
		openFolderViaDialog()
	case "file.save":
		theEditor.save()
	case "file.quit":
		application.Close()
	}
}

func (s *shellState) onEditMenu(cmd string) {
	switch cmd {
	case "edit.undo":
		widget.RunEditorCommand("undo")
	case "edit.redo":
		widget.RunEditorCommand("redo")
	case "edit.cut":
		widget.RunEditorCommand("cut")
	case "edit.copy":
		widget.RunEditorCommand("copy")
	case "edit.paste":
		widget.RunEditorCommand("paste")
	case "edit.chatsearch":
		theChatState.toggleSearch()
	case "edit.searchfiles":
		s.showLeft("search")
	}
}

func (s *shellState) onViewMenu(cmd string) {
	switch cmd {
	case "view.files":
		s.showLeft("files")
	case "view.search":
		s.showLeft("search")
	case "view.git":
		s.showLeft("git")
	case "view.chat":
		s.panels.Toggle(state.ZoneRight)
		s.SetState()
	case "view.terminal":
		s.panels.Toggle(state.ZoneBottom)
		s.SetState()
	case "view.export":
		theChatState.exportActive()
	}
}

func (s *shellState) onTerminalMenu(cmd string) {
	if cmd == "term.cmd" && !s.panels.Bottom { // 显示终端面板（companion 单终端）
		s.panels.Toggle(state.ZoneBottom)
		s.SetState()
	}
}

func (s *shellState) onHelpMenu(cmd string) {
	switch cmd {
	case "help.settings":
		openSettings()
	case "help.about":
		widget.ShowAlert("关于", "伴随式 CodeAgent —— 用 goui（Go 自绘 UI、Skia GPU）全 Go 重写的 IDE 式 AI 编码助手。", widget.MsgInfo, nil)
	}
}

// menuBarBtn 菜单栏一项：标签触发 + 下拉（Dropdown 锚到触发器，深色主题）。
func menuBarBtn(name string, items []widget.DropdownItem, onCmd func(string)) widget.Widget {
	// 用 Child=label（非 Text）：Button 对纯文本按钮(child==nil)会强加 64 最小宽 → 菜单项过宽；
	// 给 child 则按内容紧凑（label+padding，约 40px）。
	trigger := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: label(name, cText, 12)},
		Color:             *cTitle,
		MinHeight:         titleBarH,
		Padding:           types.EdgeInsetsLTRB(9, 0, 9, 0),
	}
	return widget.NewDropdown(trigger, items...).WithOnCommand(onCmd).WithPlacement(widget.PlacementBottomStart)
}

// body 主体：左栏 | 中列（编辑区+底部）| 右栏。按面板状态增减，分隔条可拖动调尺寸（兼作分隔线）。
func (s *shellState) body() widget.Widget {
	p := s.panels
	cols := []widget.Widget{}
	if p.Left {
		cols = append(cols,
			s.leftPanel(p.LeftW),
			vDivide(func(d float64) {
				p.LeftW = state.Clamp(p.LeftW+d, state.MinSideW, state.MaxSideW)
				s.SetState()
			}),
		)
	}
	cols = append(cols, expand(s.midColumn()))
	if p.Right {
		theChatState.inputAreaH = p.BottomH // 输入区高 = 终端面板高，使两者底部等高对齐（拖动终端高度时跟随）
		cols = append(cols,
			vDivide(func(d float64) {
				p.RightW = state.Clamp(p.RightW-d, state.MinSideW, state.MaxSideW) // 右栏左侧条：右拖变窄
				s.SetState()
			}),
			widget.Div( // 右栏：对话面板（GitHub 深色）。展开对话列表时右栏整体加宽，列表在对话右侧腾出，对话主区不变。
				widget.Style{Width: rightColW(p.RightW), BackgroundColor: ghBgPrimary, FlexDirection: "column", AlignItems: "stretch"},
				&ChatPanel{},
			),
		)
	}
	return flexRow(cols...)
}

// midColumn 中间列：编辑区（撑满）+ 底部面板（按状态）。
func (s *shellState) midColumn() widget.Widget {
	p := s.panels
	rows := []widget.Widget{expand(editorArea())}
	if p.Bottom {
		rows = append(rows,
			hDivide(func(d float64) {
				p.BottomH = state.Clamp(p.BottomH-d, state.MinBotH, state.MaxBotH) // 底栏上侧条：下拖变矮
				s.SetState()
			}),
			bottomPanel("terminal", "终端", "terminal", p.BottomH),
		)
	}
	return flexCol(rows...)
}

// showLeft 显示左栏并切到 view；若已可见且正是该 view，则隐藏左栏（toggle）。
func (s *shellState) showLeft(view string) {
	cur := s.leftView
	if cur == "" {
		cur = "files"
	}
	if s.panels.Left && cur == view {
		s.panels.Left = false
	} else {
		s.panels.Left = true
		s.leftView = view
	}
	s.SetState()
}

// leftPanel 左栏：顶部视图标签条（文件 / Git）+ 当前视图内容（复刻参考左区多面板 tab 切换）。
func (s *shellState) leftPanel(w float64) widget.Widget {
	view := s.leftView
	if view == "" {
		view = "files"
	}
	var content widget.Widget = &FileTreePanel{}
	switch view {
	case "git":
		content = &GitPanel{}
	case "search":
		content = &SearchPanel{}
	}
	return widget.Div(
		widget.Style{Width: w, BackgroundColor: cSide, FlexDirection: "column", AlignItems: "stretch"},
		widget.Div( // 标签条
			widget.Style{Height: 32, BackgroundColor: cSide, BorderColor: cBorder, BorderWidth: 1,
				FlexDirection: "row", AlignItems: "stretch"},
			s.tabItem("files", "文件", "folder", view),
			s.tabItem("search", "搜索", "search", view),
			s.tabItem("git", "Git", "git-branch", view),
		),
		expand(content),
	)
}

// tabItem 单个标签（图标+标题，激活态高亮）。
func (s *shellState) tabItem(id, title, icon, active string) widget.Widget {
	on := id == active
	tc, bg := cTextDim, *cSide
	if on {
		tc, bg = cText, *cTitle
	}
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(12, 0, 12, 0), FlexDirection: "row", AlignItems: "center",
				BackgroundColor: &bg},
			widget.Lucide(icon, widget.IconSize(14), widget.IconColor(tc)),
			widget.Div(widget.Style{Width: 6}),
			label(title, tc, 12),
		)},
		OnClick:    func() { s.leftView = id; s.SetState() },
		HoverColor: *ftHover,
	}
}

// sidePanel 左/右停靠面板：固定宽，高度随父拉伸。
func sidePanel(id, title, icon string, w float64) widget.Widget {
	return widget.Div(
		widget.Style{Width: w, BackgroundColor: cSide, FlexDirection: "column", AlignItems: "stretch"},
		panelHeader(title, icon),
		expand(panelBody(id)),
	)
}

// bottomPanel 底部停靠面板：固定高，宽度随父拉伸。
func bottomPanel(id, title, icon string, h float64) widget.Widget {
	return widget.Div(
		widget.Style{Height: h, BackgroundColor: cSide, FlexDirection: "column", AlignItems: "stretch"},
		panelHeader(title, icon),
		expand(panelBody(id)),
	)
}

func panelBody(id string) widget.Widget {
	switch id {
	case "files":
		return &FileTreePanel{} // 左栏「文件」：真实文件树
	case "terminal":
		return terminalArea() // 中列底部「终端」：命令运行器
	}
	return widget.Div(
		widget.Style{Padding: types.EdgeInsets(12)},
		label("〔"+id+" 面板占位〕", cTextDim, 12),
	)
}

// panelHeader 面板顶部标签条（图标+标题，垂直居中）。
func panelHeader(title, icon string) widget.Widget {
	return widget.Div(
		// 显式行容器 + 垂直居中：否则 HBox 默认 CrossStart 会把内容顶到 30px 条的顶部。
		widget.Style{Height: 30, Padding: types.EdgeInsetsLTRB(10, 0, 8, 0), BackgroundColor: cSide,
			FlexDirection: "row", AlignItems: "center"},
		widget.Lucide(icon, widget.IconSize(15), widget.IconColor(cTextDim)),
		widget.Div(widget.Style{Width: 7}),
		label(title, cText, 12),
	)
}

// statusBar 底部状态栏：左 agent 状态灯 + Git 分支，右 模型 + 编码（VS Code 风）。
// 读实时单例（运行态/分支/模型）；随 shell 重建刷新（面板开关等触发；agent 实时态对话面板已直显）。
func statusBar() widget.Widget {
	running := theChatState != nil && theChatState.bridge != nil && theChatState.bridge.isRunning()
	agentText, dotCol := "就绪", gitGreen
	if running {
		agentText, dotCol = "运行中", gitOrange
	}
	branch := "—"
	if theGit != nil && theGit.isRepo && theGit.branch != "" {
		branch = theGit.branch
	}
	model := "未配置模型"
	if theSettings.Model != "" {
		model = theSettings.Model
	} else if theSettings.Provider != "" {
		model = theSettings.Provider
	}
	return widget.Div(
		widget.Style{Height: statusH, BackgroundColor: cStatusBar, Padding: types.EdgeInsetsLTRB(10, 0, 10, 0),
			FlexDirection: "row", AlignItems: "center"},
		widget.Div(widget.Style{Width: 8, Height: 8, BackgroundColor: &dotCol, BorderRadius: 4}), // agent 状态灯
		widget.Div(widget.Style{Width: 6}),
		label(agentText, cText, 12),
		widget.Div(widget.Style{Width: 16}),
		statusItem("git-branch", branch, cText),
		expand(widget.Div(widget.Style{})), // 中间撑满，把右侧项推到最右
		statusItem("", model, cText),
		widget.Div(widget.Style{Width: 16}),
		statusItem("", "UTF-8", cText),
	)
}

// statusItem 状态栏一项：可选图标 + 文本，水平排列、垂直居中。
func statusItem(icon, text string, c types.Color) widget.Widget {
	kids := []widget.Widget{}
	if icon != "" {
		kids = append(kids, widget.Lucide(icon, widget.IconSize(13), widget.IconColor(c)), widget.Div(widget.Style{Width: 5}))
	}
	kids = append(kids, label(text, c, 12))
	return rowCenter(kids...)
}

// rowCenter 行容器、交叉轴居中（接受动态子节点）。
func rowCenter(children ...widget.Widget) widget.Widget {
	args := make([]interface{}, 0, len(children)+1)
	args = append(args, widget.Style{FlexDirection: "row", AlignItems: "center"})
	for _, c := range children {
		args = append(args, c)
	}
	return widget.Div(args...)
}

// ─── 小工具 ───────────────────────────────────────────────

func expand(w widget.Widget) widget.Widget {
	return &widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: w}, Flex: 1}
}

func label(s string, c types.Color, size float64) widget.Widget {
	t := widget.NewText(s, c)
	t.Font.Size = size
	return t
}

// rightColW 右栏宽度：展开对话列表时额外加宽（列表在对话右侧腾出，对话主区不变）。
func rightColW(base float64) float64 {
	if theChatState.showThreads {
		return base + 190
	}
	return base
}

// flexRow / flexCol：带交叉轴拉伸的行/列容器，接受动态子节点列表。
// （Div 只收 ...interface{}，无法直接 spread []Widget，这里封一层省去手拼。）
func flexRow(children ...widget.Widget) widget.Widget { return flexBox("row", children) }
func flexCol(children ...widget.Widget) widget.Widget { return flexBox("column", children) }

func flexBox(dir string, children []widget.Widget) widget.Widget {
	args := make([]interface{}, 0, len(children)+1)
	args = append(args, widget.Style{FlexDirection: dir, AlignItems: "stretch"})
	for _, c := range children {
		args = append(args, c)
	}
	return widget.Div(args...)
}

// vDivide / hDivide：停靠分隔条。分隔条的样式（粗细 dividerW + 分隔色/高亮色）全部集中在此处
// 参数化——要改样式改这两行即可、无需动 goui 组件；组件只负责"能力"（hover 高亮 / 拖动 / 填充背景）。
func vDivide(onDrag func(float64)) widget.Widget {
	return widget.VResize(*cBorder, *cStatus, onDrag).WithThickness(dividerW)
}
func hDivide(onDrag func(float64)) widget.Widget {
	return widget.HResize(*cBorder, *cStatus, onDrag).WithThickness(dividerW)
}
