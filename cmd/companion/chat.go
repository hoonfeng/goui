// 对话面板 —— 1:1 复刻参考源（GitHub 深色主题）：布局工具栏 + 可折叠对话侧栏 +
// 用户/Agent 消息卡 + 输入区（包裹盒 + textarea + 工具按钮栏）。详见 AGENTS.md。
//
//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/goui/cmd/companion/state"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// ─── GitHub 深色主题（复刻参考源 :root 变量，解析到 hex）──────
var (
	ghBgPrimary   = types.ColorRef(13, 17, 23)        // #0d1117 主背景
	ghBgSecondary = types.ColorRef(22, 27, 34)        // #161b22 次背景（侧栏）
	ghBgTertiary  = types.ColorRef(33, 38, 45)        // #21262d 三级（agent 卡）
	ghBgHover     = types.ColorRef(48, 54, 61)        // #30363d hover
	ghBgActive    = types.ColorRef(28, 45, 74)        // bg-active 近似（#1f6feb22 叠在深底）
	ghBorder      = types.ColorRef(48, 54, 61)        // #30363d 边框
	ghAccent      = types.ColorRef(88, 166, 255)      // #58a6ff 强调蓝
	ghAccentEmph  = types.ColorRef(31, 111, 235)      // #1f6feb 主按钮底
	ghBlue        = types.ColorRef(59, 130, 246)      // #3b82f6 自主模式（刻意区别于 accent）
	ghText        = types.ColorFromRGB(230, 237, 243) // #e6edf3 主文字
	ghTextSec     = types.ColorFromRGB(139, 148, 158) // #8b949e 次文字
	ghTextMuted   = types.ColorFromRGB(110, 118, 129) // #6e7681 弱文字
	ghSuccess     = types.ColorRef(63, 185, 80)       // #3fb950 完成绿
	ghWarning     = types.ColorRef(210, 153, 34)      // #d29922 警告黄
	ghDanger      = types.ColorRef(248, 81, 73)       // #f85149 错误红
	ghUserBg      = types.ColorRef(24, 23, 18)        // 用户卡黄底（#0d1117 上叠 warning 4%）
	ghUserBorder  = types.ColorRef(38, 35, 23)        // 用户卡边（warning 10%）
	cWhite        = types.ColorFromRGB(255, 255, 255)
)

// ChatPanel 对话面板。
type ChatPanel struct{ widget.StatefulWidget }

// theChatState 是对话面板的持久状态（包级单例）。原因：companion 每次 relayout 会重跑
// shell.Build 重建整棵右栏（含本面板的 StatefulElement），若状态挂在 Element 上会随之
// 被重置（showThreads/会话/草稿全丢）。单例确保跨重建存活。companion 只有一个对话面板，安全。
var theChatState = &chatState{store: state.NewChatStore(), autoReview: true, hoveredMsg: -1}

func (c *ChatPanel) CreateState() widget.State { return theChatState }

// planStep Agent 任务计划的一步（update_plan 工具传入；status: pending/in_progress/done）。
type planStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

type chatState struct {
	widget.BaseState
	store        *state.ChatStore
	showThreads  bool         // 对话侧栏展开
	threadLeft   bool         // 对话侧栏靠左（默认右），由换边按钮切换
	autoReview   bool         // 自动/手动审核
	autonomous   bool         // 全自主模式
	autoCollapse bool         // 最后一条自动收缩
	sendSeq      int          // 递增 → 输入框清空 + 滚到底
	inputAreaH   float64      // 输入区固定高（由 main.go 设为终端面板高 BottomH，使两者等高对齐）
	bridge       *agentBridge // Agent 引擎接入（懒建，见 agent_bridge.go）
	plan         []planStep   // 当前 Agent 任务计划清单（update_plan 工具更新；置顶可视）

	hoveredMsg  int    // 当前 hover 的消息索引（-1=无）→ 揭示该消息的操作按钮
	showSearch  bool   // Ctrl+F 搜索栏开
	searchQuery string // 搜索词（实时过滤消息）
	searchSeq   int    // 搜索框受控清空令牌
}

func (s *chatState) Build(ctx widget.BuildContext) widget.Widget {
	mainKids := []widget.Widget{s.layoutToolbar()}
	if s.showSearch {
		mainKids = append(mainKids, s.searchBar())
	}
	if len(s.plan) > 0 {
		mainKids = append(mainKids, s.planCard())
	}
	mainKids = append(mainKids, expand(s.scrollMessages()), s.inputArea())
	main := widget.Div(
		widget.Style{BackgroundColor: ghBgPrimary, FlexDirection: "column", AlignItems: "stretch"},
		mainKids,
	)
	if !s.showThreads {
		return main
	}
	// 对话列表：与主区并排分隔（不覆盖）。右栏整体会相应加宽（见 main.go rightColW），
	// 故对话主区宽度不变、列表独立腾出。换边按钮可让列表靠左/靠右（复刻 threadListSide）。
	if s.threadLeft {
		return flexRow(s.sidebarContent(), vlineGh(), expand(main))
	}
	return flexRow(expand(main), vlineGh(), s.sidebarContent())
}

func (s *chatState) scrollMessages() widget.Widget {
	sv := widget.NewScrollView(s.messageList())
	sv.ScrollEndToken = s.sendSeq
	return sv
}

// planCard 置顶任务计划清单（update_plan 工具更新）：标题 + 进度 + 每步状态图标。
func (s *chatState) planCard() widget.Widget {
	done := 0
	for _, p := range s.plan {
		if p.Status == "done" {
			done++
		}
	}
	rows := []widget.Widget{
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 0, 0, 5)},
			widget.Lucide("list-checks", widget.IconSize(13), widget.IconColor(*ghAccent)),
			widget.Div(widget.Style{Width: 6}),
			label("计划", ghText, 12),
			widget.Div(widget.Style{Width: 6}),
			label(fmt.Sprintf("%d/%d", done, len(s.plan)), ghTextMuted, 10),
		),
	}
	for _, p := range s.plan {
		rows = append(rows, planRow(p))
	}
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(8, 6, 8, 6)},
		widget.Div(
			widget.Style{Padding: types.EdgeInsets(8), BackgroundColor: ghBgSecondary, BorderColor: ghBorder,
				BorderWidth: 1, BorderRadius: 6, FlexDirection: "column", AlignItems: "stretch"},
			rows,
		),
	)
}

func planRow(p planStep) widget.Widget {
	icon, col := "circle", ghTextMuted // pending
	switch p.Status {
	case "done":
		icon, col = "circle-check", gitGreen
	case "in_progress":
		icon, col = "loader-circle", *ghAccent
	}
	txtCol := ghText
	if p.Status == "done" {
		txtCol = ghTextMuted
	}
	return widget.Div(
		widget.Style{Height: 20, FlexDirection: "row", AlignItems: "center"},
		widget.Lucide(icon, widget.IconSize(13), widget.IconColor(col)),
		widget.Div(widget.Style{Width: 6}),
		expand(label(p.Step, txtCol, 11.5)),
	)
}

// ─── 顶部布局工具栏（List 显隐侧栏，靠右）─────────────────
func (s *chatState) layoutToolbar() widget.Widget {
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(6, 3, 6, 3), BackgroundColor: ghBgPrimary,
			BorderColor: ghBorder, BorderWidth: 1, FlexDirection: "row", AlignItems: "center"},
		expand(widget.Div(widget.Style{})), // 弹性占位把按钮顶到右侧（justify-end 单子项不撑空间）
		toolBtn("list", s.showThreads, func() { s.showThreads = !s.showThreads; s.SetState() }),
	)
}

// ─── 对话侧栏 ─────────────────────────────────────────────

// strip 6px 可点击竖条：折叠/展开对话侧栏。
func (s *chatState) strip() widget.Widget {
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(widget.Style{Width: 6})},
		OnClick:           func() { s.showThreads = !s.showThreads; s.SetState() },
		Color:             *ghBgSecondary,
		MinWidth:          6,
	}
}

// sidebarContent 侧栏内容（180px）：头部（标题 + 切换/导出/新建）+ 会话列表。
func (s *chatState) sidebarContent() widget.Widget {
	rows := []widget.Widget{}
	for _, t := range s.store.Threads {
		rows = append(rows, s.threadItem(t))
	}
	return widget.Div(
		widget.Style{Width: 180, BackgroundColor: ghBgSecondary, BorderColor: ghBorder, BorderWidth: 1,
			FlexDirection: "column", AlignItems: "stretch"},
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(10, 8, 8, 8), BorderColor: ghBorder, BorderWidth: 1,
				FlexDirection: "row", AlignItems: "center"},
			expand(label("对话", ghTextSec, 12)),
			toolBtn("arrow-left-right", s.threadLeft, func() { s.threadLeft = !s.threadLeft; s.SetState() }),
			toolBtn("download", false, s.exportActive),
			toolBtn("plus", false, func() { s.store.NewThread(); s.SetState() }),
		),
		expand(widget.NewScrollView(flexCol(rows...))),
	)
}

// threadItem 会话项：左强调条(当前) + 状态灯(运行黄/就绪绿) + 标题 + 关闭×。整行可点、悬停高亮。
func (s *chatState) threadItem(t *state.Thread) widget.Widget {
	tt := t
	active := t.ID == s.store.ActiveID
	txt := ghTextMuted
	bg := types.Color{} // 透明
	hover := *ghBgHover
	if active {
		txt = ghText
		bg = *ghBgActive
		hover = *ghBgActive // 选中态悬停不再变色
	}
	// 状态灯：该会话正被 Agent 运行→运行黄，否则就绪绿（复刻 thread.status 着色）。
	dot := ghSuccess
	if s.threadRunning(tt) {
		dot = ghWarning
	}
	// 左侧 3px 强调条（仅当前会话），复刻参考 borderLeft:3px accent；非当前留占位保持对齐。
	var bar widget.Widget
	if active {
		bar = widget.Div(widget.Style{Width: 3, Height: 18, BackgroundColor: ghAccent, BorderRadius: 1.5})
	} else {
		bar = widget.Div(widget.Style{Width: 3})
	}
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 36, FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 0, 6, 0)},
			bar,
			widget.Div(widget.Style{Width: 7}),
			statusDot(dot),
			widget.Div(widget.Style{Width: 8}),
			expand(label(tt.Title, txt, 12)),
			s.threadCloseBtn(tt),
		)},
		OnClick:    func() { s.store.Switch(tt.ID); s.SetState() },
		Color:      bg,
		HoverColor: hover,
	}
}

// threadCloseBtn 会话关闭×：StopPropagation 使点×只删会话、不触发外层切换。
func (s *chatState) threadCloseBtn(t *state.Thread) widget.Widget {
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Padding: types.EdgeInsets(4)},
			widget.Lucide("x", widget.IconSize(13), widget.IconColor(ghTextMuted)),
		)},
		OnClick:         func() { s.store.Delete(t.ID); s.SetState() },
		StopPropagation: true,
		HoverColor:      *ghBgHover,
	}
}

// threadRunning 该会话是否正被 Agent 引擎运行（状态灯着色用）。
func (s *chatState) threadRunning(t *state.Thread) bool {
	return s.bridge != nil && s.bridge.runningThread() == t
}

// exportActive 把当前会话导出为 Markdown 写到工作区根，并在对话里回执路径（复刻 download/导出）。
func (s *chatState) exportActive() {
	th := s.store.Active()
	if th == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", th.Title)
	for _, m := range th.Messages {
		who := "用户"
		if m.Role == state.Assistant {
			who = "Agent"
		}
		fmt.Fprintf(&b, "## %s\n\n", who)
		if t := strings.TrimSpace(m.Thinking); t != "" {
			fmt.Fprintf(&b, "> 思考：%s\n\n", t)
		}
		for _, a := range m.Activities {
			fmt.Fprintf(&b, "- `%s` %s\n", a.Tool, argPreview(a.Args))
		}
		if len(m.Activities) > 0 {
			b.WriteString("\n")
		}
		if t := strings.TrimSpace(m.Text); t != "" {
			b.WriteString(t + "\n\n")
		}
	}
	wd, _ := os.Getwd()
	path := filepath.Join(wd, "对话导出-"+th.ID+".md")
	note := "已导出对话到 " + path
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		note = "导出失败：" + err.Error()
	}
	th.Messages = append(th.Messages, state.Message{Role: state.Assistant, Text: note})
	s.sendSeq++
	s.SetState()
}

func statusDot(c *types.Color) widget.Widget {
	return widget.Div(widget.Style{Width: 7, Height: 7, BackgroundColor: c, BorderRadius: 3.5})
}

func vlineGh() widget.Widget { return widget.Div(widget.Style{Width: 1, BackgroundColor: ghBorder}) }

// ─── 消息列表 ─────────────────────────────────────────────

func (s *chatState) messageList() widget.Widget {
	t := s.store.Active()
	if t == nil {
		return widget.Div(widget.Style{})
	}
	q := strings.ToLower(strings.TrimSpace(s.searchQuery))
	kids := []widget.Widget{}
	for i := range t.Messages {
		if q != "" && !msgMatches(t.Messages[i], q) { // Ctrl+F 过滤
			continue
		}
		kids = append(kids, s.renderMessage(t, i), widget.Div(widget.Style{Height: 8}))
	}
	if q != "" && len(kids) == 0 {
		kids = append(kids, label("无匹配消息", ghTextMuted, 12))
	}
	return widget.Div(widget.Style{Padding: types.EdgeInsets(10)}, flexCol(kids...))
}

// renderMessage 渲染单条消息：用户黄卡 / Agent 卡（折叠回调按索引改 store）；hover 揭示操作按钮（绝对定位叠加）。
func (s *chatState) renderMessage(t *state.Thread, i int) widget.Widget {
	m := t.Messages[i]
	var card widget.Widget
	if m.Role == state.User {
		card = userCard(m)
	} else {
		card = agentMessageCard(m,
			func() { t.Messages[i].Collapsed = !t.Messages[i].Collapsed; s.SetState() },
			func() { t.Messages[i].ThinkingExpanded = !t.Messages[i].ThinkingExpanded; s.SetState() },
			func(ai int) {
				if ai >= 0 && ai < len(t.Messages[i].Activities) {
					t.Messages[i].Activities[ai].Expanded = !t.Messages[i].Activities[ai].Expanded
					s.SetState()
				}
			},
		)
	}
	content := card
	if actions := s.messageActions(t, i); actions != nil { // hover 时叠加操作按钮（绝对定位，不挤布局）
		content = widget.NewStack(card, widget.NewPositioned(actions).WithTop(6).WithRight(8))
	}
	// hover 揭示：包一层只检测 hover 的 Clickable（OnClick=nil→无手型）；goui 子树 hover 语义保证移到按钮上仍算 hover。
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: content},
		OnHoverChange: func(h bool) {
			switch {
			case h:
				s.hoveredMsg = i
			case s.hoveredMsg == i:
				s.hoveredMsg = -1
			default:
				return
			}
			s.SetState()
		},
	}
}

// messageActions hover 揭示的操作按钮行：复制 / 重新生成（仅末条助手且空闲）/ 删除。未 hover 或流式中返回 nil。
func (s *chatState) messageActions(t *state.Thread, i int) widget.Widget {
	if i < 0 || i >= len(t.Messages) || s.hoveredMsg != i {
		return nil
	}
	m := t.Messages[i]
	if m.Streaming {
		return nil
	}
	var kids []widget.Widget
	if widget.ClipboardWrite != nil {
		kids = append(kids, msgActionBtn("copy", func() { widget.ClipboardWrite(m.Text) }))
	}
	if m.Role == state.Assistant && i == len(t.Messages)-1 && !s.agentBusy() {
		kids = append(kids, msgActionBtn("refresh-cw", func() { s.regenerate(t, i) }))
	}
	kids = append(kids, msgActionBtn("trash-2", func() { s.deleteMessage(t, i) }))
	return widget.Div(
		widget.Style{BackgroundColor: ghBgSecondary, BorderColor: ghBorder, BorderWidth: 1, BorderRadius: 5,
			Padding: types.EdgeInsets(2), FlexDirection: "row", AlignItems: "center", Gap: 1},
		kids,
	)
}

// msgActionBtn 消息操作小图标钮（透明底、悬停高亮）。
func msgActionBtn(icon string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: icon, IconSize: 12, TextColor: ghTextMuted,
		OnClick: onClick, Color: *ghBgSecondary, HoverColor: *ghBgHover,
		BorderRadius: 4, MinWidth: 22, MinHeight: 20,
	}
}

func (s *chatState) agentBusy() bool { return s.bridge != nil && s.bridge.isRunning() }

// deleteMessage 删除某条消息。
func (s *chatState) deleteMessage(t *state.Thread, i int) {
	if i < 0 || i >= len(t.Messages) {
		return
	}
	t.Messages = append(t.Messages[:i], t.Messages[i+1:]...)
	s.hoveredMsg = -1
	s.SetState()
}

// regenerate 重新生成末条助手回复：删掉它（末条复位为 user task）再跑一轮。
func (s *chatState) regenerate(t *state.Thread, i int) {
	if s.agentBusy() || i <= 0 || i != len(t.Messages)-1 {
		return
	}
	user := t.Messages[i-1]
	if user.Role != state.User {
		return
	}
	t.Messages = t.Messages[:i] // 删本条助手回复 → 末条变回 user task（start 的 history 排除末条）
	if s.bridge == nil {
		s.bridge = &agentBridge{cs: s}
	}
	s.hoveredMsg = -1
	s.sendSeq++
	s.bridge.start(user.Text)
	s.SetState()
}

// userCard 用户消息：黄底卡片、pre-wrap 文本（复刻 .cc-user）。
func userCard(m state.Message) widget.Widget {
	return widget.Div(
		widget.Style{BackgroundColor: ghUserBg, BorderColor: ghUserBorder, BorderWidth: 1, BorderRadius: 6,
			Padding: types.EdgeInsetsLTRB(12, 8, 12, 8), FlexDirection: "column", AlignItems: "stretch"},
		label(m.Text, ghText, 13),
	)
}

// ─── 输入区（复刻 ChatInput）──────────────────────────────

func (s *chatState) inputArea() widget.Widget {
	placeholder := "输入任务... Enter 发送"
	if s.bridge != nil && s.bridge.isRunning() {
		placeholder = "Agent 运行中…（点停止后再发）"
	}
	ta := widget.NewTextarea(placeholder, 3, func(t string) { s.store.Draft = t })
	ta.Text = s.store.Draft // 重建时回填草稿（发送后随 sendSeq 复位为空）
	ta.ResetToken = s.sendSeq
	ta.BGColor = types.Color{} // 透明：让外层圆角包裹盒的背景+圆角透出（否则 textarea 方角盖住上圆角）
	ta.Color = ghText
	ta.CursorColor = ghText // 亮光标，深背景可见
	ta.PlaceholderColor = ghTextMuted
	// 三态边框都设为包裹盒底色 → textarea 自身无边框（边框由外层圆角盒提供）。
	// 关键：HoverBorderColor 不设会默认 el #c0c4cc 浅灰，导致悬停时框内冒出一圈灰边框。
	ta.BorderColor = *ghBgPrimary
	ta.FocusBorderColor = *ghBgPrimary
	ta.HoverBorderColor = *ghBgPrimary

	// 输入框包裹盒：圆角 8、1px 边框，含 textarea（撑满剩余高）+ 底部工具按钮栏。
	box := widget.Div(
		widget.Style{BackgroundColor: ghBgPrimary, BorderColor: ghBorder, BorderWidth: 1, BorderRadius: 8,
			FlexDirection: "column", AlignItems: "stretch"},
		expand(ta), // textarea 撑满包裹盒减按钮栏的剩余高度
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(8, 4, 8, 8), FlexDirection: "row", AlignItems: "center"},
			iconGhost("paperclip", func() {}),
			expand(widget.Div(widget.Style{})),
			s.reviewToggle(),
			widget.Div(widget.Style{Width: 5}),
			toggleBtn("refresh-cw", "自主", s.autonomous, ghBlue, func() { s.autonomous = !s.autonomous; s.SetState() }),
			widget.Div(widget.Style{Width: 5}),
			toggleBtn("chevron-down", "收缩", s.autoCollapse, ghAccent, func() { s.autoCollapse = !s.autoCollapse; s.SetState() }),
			widget.Div(widget.Style{Width: 8}),
			s.sendOrStop(),
		),
	)
	// 输入区固定高 = 终端面板高（inputAreaH），与中列底部终端面板等高对齐；包裹盒撑满。
	inputH := s.inputAreaH
	if inputH < 100 {
		inputH = 200 // 兜底（= 终端面板默认高）
	}
	return widget.Div(
		widget.Style{Height: inputH, Padding: types.EdgeInsetsLTRB(12, 8, 12, 10), BorderColor: ghBorder, BorderWidth: 1,
			BackgroundColor: ghBgPrimary, FlexDirection: "column", AlignItems: "stretch"},
		expand(box),
	)
}

// ─── Ctrl+F 搜索 ──────────────────────────────────────────

// toggleSearch Ctrl+F 开关搜索栏（关闭时清空查询）。
func (s *chatState) toggleSearch() {
	s.showSearch = !s.showSearch
	if !s.showSearch {
		s.searchQuery = ""
		s.searchSeq++
	}
	s.SetState()
}

// searchBar 搜索栏：放大镜 + 输入(实时过滤) + 匹配计数 + 关闭×。
func (s *chatState) searchBar() widget.Widget {
	in := widget.NewInput("搜索对话内容…", func(q string) { s.searchQuery = q; s.SetState() })
	in.Text = s.searchQuery
	in.ResetToken = s.searchSeq
	in.Color = ghText
	in.CursorColor = ghText
	in.PlaceholderColor = ghTextMuted
	in.BGColor = *ghBgSecondary
	in.BorderColor = *ghBgSecondary
	in.FocusBorderColor = *ghBgSecondary
	in.HoverBorderColor = *ghBgSecondary
	var count widget.Widget = widget.Div(widget.Style{})
	if strings.TrimSpace(s.searchQuery) != "" {
		count = label(fmt.Sprintf("%d 条", s.searchMatchCount()), ghTextMuted, 11)
	}
	return widget.Div(
		widget.Style{Height: 34, Padding: types.EdgeInsetsLTRB(10, 0, 8, 0), BackgroundColor: ghBgSecondary,
			BorderColor: ghBorder, BorderWidth: 1, FlexDirection: "row", AlignItems: "center"},
		widget.Lucide("search", widget.IconSize(13), widget.IconColor(ghTextMuted)),
		widget.Div(widget.Style{Width: 8}),
		expand(in),
		count,
		widget.Div(widget.Style{Width: 6}),
		toolBtn("x", false, s.toggleSearch),
	)
}

// msgMatches 消息是否命中搜索词（正文/思考/工具名·参数·结果；q 须已小写）。
func msgMatches(m state.Message, q string) bool {
	if strings.Contains(strings.ToLower(m.Text), q) || strings.Contains(strings.ToLower(m.Thinking), q) {
		return true
	}
	for _, a := range m.Activities {
		if strings.Contains(strings.ToLower(a.Tool), q) ||
			strings.Contains(strings.ToLower(a.Args), q) ||
			strings.Contains(strings.ToLower(a.Result), q) {
			return true
		}
	}
	return false
}

// searchMatchCount 当前会话命中搜索词的消息数。
func (s *chatState) searchMatchCount() int {
	t := s.store.Active()
	q := strings.ToLower(strings.TrimSpace(s.searchQuery))
	if t == nil || q == "" {
		return 0
	}
	n := 0
	for _, m := range t.Messages {
		if msgMatches(m, q) {
			n++
		}
	}
	return n
}

// ─── 小部件 ───────────────────────────────────────────────

// toolBtn 工具栏图标按钮（26×26，active 高亮）。用 Button 原生 Icon。
func toolBtn(icon string, active bool, onClick func()) widget.Widget {
	col := ghTextSec
	bg := *ghBgPrimary
	if active {
		col = ghText
		bg = *ghBgTertiary
	}
	return &widget.Button{
		Icon: icon, IconSize: 14, TextColor: col,
		OnClick: onClick, Color: bg,
		MinWidth: 26, MinHeight: 26,
	}
}

// iconGhost 透明底图标按钮（输入栏附件等）。用 Button 原生 Icon。
func iconGhost(icon string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: icon, IconSize: 14, TextColor: ghTextMuted,
		OnClick: onClick, Color: *ghBgPrimary,
		MinWidth: 24, MinHeight: 24,
	}
}

// tint 把强调色按 alpha 淡化为按钮底色（开启态的低透明填充）。
func tint(c *types.Color, a uint8) types.Color {
	return types.Color{R: c.R, G: c.G, B: c.B, A: a}
}

// reviewToggle 审核模式：双态彩色切换——自动(绿盾) ↔ 手动(黄盾)，永不灰显（复刻参考源）。
// 返回的就是一个 widget.Button（原生 Icon+Text），无任何中间封装。
func (s *chatState) reviewToggle() widget.Widget {
	icon, text, c := "shield", "自动", ghSuccess
	if !s.autoReview {
		icon, text, c = "shield-off", "手动", ghWarning
	}
	return &widget.Button{
		Icon: icon, IconSize: 13, IconGap: 4, Text: text, TextColor: *c, FontSize: 11,
		OnClick:      func() { s.autoReview = !s.autoReview; s.SetState() },
		Color:        tint(c, 30),
		BorderColor:  c,
		BorderWidth:  1,
		BorderRadius: 5,
		Padding:      types.EdgeInsetsLTRB(9, 3, 10, 3),
		MinHeight:    12,
	}
}

// toggleBtn 输入栏开关按钮（图标 + 文字）：开启时染强调色，关闭时低调灰。就是一个 widget.Button。
func toggleBtn(icon, text string, on bool, onColor *types.Color, onClick func()) widget.Widget {
	fg, border, bg := ghTextMuted, ghBorder, *ghBgPrimary
	if on {
		fg, border, bg = *onColor, onColor, tint(onColor, 30)
	}
	return &widget.Button{
		Icon: icon, IconSize: 13, IconGap: 4, Text: text, TextColor: fg, FontSize: 11,
		OnClick:      onClick,
		Color:        bg,
		BorderColor:  border,
		BorderWidth:  1,
		BorderRadius: 5,
		Padding:      types.EdgeInsetsLTRB(9, 3, 10, 3),
		MinHeight:    12,
	}
}

// primaryBtn 主按钮（发送）：实心强调底 + 送出图标 + 文字。用 Button 原生 Icon。
func primaryBtn(text string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: "send", IconSize: 13, IconGap: 6,
		Text: text, TextColor: cWhite, FontSize: 12,
		OnClick:      onClick,
		Color:        *ghAccentEmph,
		BorderRadius: 5,
		Padding:      types.EdgeInsetsLTRB(12, 3, 13, 3),
		MinHeight:    12,
	}
}

// sendOrStop 运行中显示红「停止」(取消 loop)，空闲显示蓝「发送」。
func (s *chatState) sendOrStop() widget.Widget {
	if s.bridge != nil && s.bridge.isRunning() {
		return &widget.Button{
			Icon: "square", IconSize: 12, IconGap: 6,
			Text: "停止", TextColor: cWhite, FontSize: 12,
			OnClick:      func() { s.bridge.stop() },
			Color:        *ghDanger,
			BorderRadius: 5,
			Padding:      types.EdgeInsetsLTRB(12, 3, 13, 3),
			MinHeight:    12,
		}
	}
	return primaryBtn("发送", s.send)
}

func (s *chatState) send() {
	if s.bridge != nil && s.bridge.isRunning() {
		return // 上一轮还在跑，不重复发
	}
	draft := s.store.Draft
	if !s.store.Send(draft) { // 只加 user 消息（Send 内部 trim+空判）
		return
	}
	s.sendSeq++ // 清输入框 + 滚到底
	s.plan = nil // 新任务 → 清旧计划清单（Agent 会用 update_plan 重列）
	if s.bridge == nil {
		s.bridge = &agentBridge{cs: s}
	}
	s.bridge.start(draft) // 异步跑 Agent 引擎，流式回复（start 内 trim）
	s.SetState()
}
