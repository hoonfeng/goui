// Agent 引擎 ↔ 聊天面板 桥接：把 agent.Loop 接到对话 UI。
// send → 异步跑 TAOR 循环（goroutine）；事件（thinking/工具调用/结果/final）经**动画帧泵**
// 每帧在 UI 线程 drain，流式写进当前助手消息（复刻终端面板跨线程模式，见 terminal.go / AGENTS.md）。
// 本文件还含 Agent 消息卡的富渲染（思考块 + 工具活动行 + 正文）。
//
//go:build windows

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/goui/cmd/companion/agent"
	"github.com/user/goui/cmd/companion/state"
	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// agentBridge 持有 Agent 引擎与一次流式回复的运行态。挂在 chatState 上（懒建）。
type agentBridge struct {
	cs   *chatState
	loop *agent.Loop // 懒建（provider 来自环境变量）；测试可预置 mock
	root string      // 工作区根（= 工具/文件树/编辑器同根），用于把工具 path 解析为绝对路径

	mu        sync.Mutex
	running   bool
	pending   []agent.Event // loop 协程写、帧泵 drain
	pump      *animation.Controller
	runThread *state.Thread // 当前流式回复所在会话（捕获，防用户切会话）
	runIdx    int           // 流式回复在该会话的消息索引

	cancel  context.CancelFunc // 取消当前 loop（停止按钮）；start 置、stop 调
	stopped bool               // 本轮被用户主动停止（UI 线程读写）：抑制错误显示、收尾标[已停止]

	// 审批（手动审核模式）：loop 协程在 approve() 里登记裁决通道并阻塞，UI 线程点「允许/拒绝」经 resolveApproval 送回。
	approvalCh     chan bool
	approvalCallID string

	// ask_user：loop 协程在 askUser() 里登记回答通道并阻塞，UI 线程选项/输入经 resolveAsk 送回（同一时刻仅一个）。
	askCh chan string
}

func (b *agentBridge) isRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// runningThread 返回当前正被运行的会话（未运行返回 nil）。会话列表状态灯着色用。
func (b *agentBridge) runningThread() *state.Thread {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return b.runThread
	}
	return nil
}

// stop 用户主动停止当前运行：取消 ctx（loop 在下个迭代/Chat/审批等待处停下）。收尾标记 [已停止]。
func (b *agentBridge) stop() {
	b.mu.Lock()
	b.stopped = true
	cancel := b.cancel
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// resetForNewRoot 项目根切换后清掉已建 loop，下条消息用新根重建（运行中则不动，避免打断）。
func (b *agentBridge) resetForNewRoot() {
	if b.isRunning() {
		return
	}
	b.loop = nil
	b.root = ""
}

// autonomousParams 据自主开关算（实际下发给 LLM 的任务文本, 迭代上限）。
// 自主：追加「列计划→连续完成所有步骤→全部完成再 [FINAL]」提示 + 放宽迭代上限（一气呵成多步任务）。
func autonomousParams(task string, autonomous bool) (string, int) {
	base := theSettings.MaxIterations // 设置面板「最大迭代步数」；未设=30
	if base <= 0 {
		base = 30
	}
	if autonomous {
		return task + "\n\n（自主模式：先用 update_plan 列出完整计划，然后连续完成所有步骤、中途不要停下等我，全部完成后再输出 [FINAL]。）", base * 2
	}
	return task, base
}

// start 异步跑一轮 Agent 任务（UI 线程调用）。无 API key 则只提示、不跑。
func (b *agentBridge) start(task string) {
	task = strings.TrimSpace(task)
	if task == "" {
		return
	}
	th := b.cs.store.Active()
	if th == nil {
		return
	}
	// 懒建 loop：provider 来自环境变量；无 key → 给一条提示、不进循环。
	if b.loop == nil {
		prov := buildProvider()
		if prov == nil {
			th.Messages = append(th.Messages, state.Message{Role: state.Assistant,
				Text: "未配置 API key。请设置环境变量 DEEPSEEK_API_KEY / OPENAI_API_KEY / DASHSCOPE_API_KEY / MOONSHOT_API_KEY / OPENROUTER_API_KEY 之一后重启，即可与我对话。"})
			return
		}
		root := currentRoot() // 当前项目根（与文件树/终端统一）
		b.root = root
		reg := agent.NewRegistry()
		agent.RegisterDefaultTools(reg, root)
		b.registerAskTool(reg) // ask_user：handler 闭包持有 bridge（需 UI 交互），故在此注册而非默认集
		if cfgs := loadMCPConfigs(); len(cfgs) > 0 { // 外部 MCP 服务器（mcp.json；失败跳过、不阻断；首条消息时一次性连接）
			agent.RegisterMCPServers(reg, cfgs)
		}
		sys := agent.DefaultSystemPrompt(root)
		if si := strings.TrimSpace(theSettings.SystemInstructions); si != "" { // 设置里的系统级指令
			sys += "\n\n# 系统级指令（务必遵守）\n" + si
		}
		sys += agent.ProjectRules(root)
		b.loop = &agent.Loop{Provider: prov, Registry: reg, System: sys, MaxIterations: 30}
	}
	hist := b.history(th)

	// 审批门：手动审核（审核=手动 且 非自主模式）时，写类工具执行前阻塞等用户批准；
	// 自动审核或自主模式 → 全部放行（Approve=nil）。每次发送都按当前开关状态重设。
	if !b.cs.autoReview && !b.cs.autonomous {
		b.loop.Approve = b.approve
	} else {
		b.loop.Approve = nil
	}

	// 自主模式：放宽迭代上限 + 提示连续完成整份计划（配合 update_plan 清单一气呵成，不中途停等）。
	task, b.loop.MaxIterations = autonomousParams(task, b.cs.autonomous)

	// 流式助手消息（占位，事件到来时填充）。
	th.Messages = append(th.Messages, state.Message{Role: state.Assistant, Streaming: true})
	b.runThread = th
	b.runIdx = len(th.Messages) - 1

	ctx, cancel := context.WithCancel(context.Background())
	b.mu.Lock()
	b.running = true
	b.stopped = false
	b.cancel = cancel
	b.pending = nil
	b.mu.Unlock()

	b.loop.OnEvent = func(e agent.Event) { // loop 协程调用 → 缓冲（加锁），不碰 UI/Element
		b.mu.Lock()
		b.pending = append(b.pending, e)
		b.mu.Unlock()
	}
	loop := b.loop
	go func() {
		loop.Run(ctx, task, hist)
		cancel() // 释放 ctx 资源
		b.mu.Lock()
		b.running = false
		b.mu.Unlock()
	}()
	b.startPump()
}

// history 把当前会话的既往消息（不含末尾刚加的 user task）转成 LLM 上下文。
func (b *agentBridge) history(th *state.Thread) []agent.Message {
	var h []agent.Message
	msgs := th.Messages
	for i := 0; i < len(msgs)-1; i++ { // 末条是本次 user task，loop 内部会再加 → 这里排除
		m := msgs[i]
		if m.Streaming || strings.TrimSpace(m.Text) == "" {
			continue
		}
		role := agent.RoleAssistant
		if m.Role == state.User {
			role = agent.RoleUser
		}
		h = append(h, agent.Message{Role: role, Content: m.Text})
	}
	return h
}

// startPump 帧泵：运行期间使主循环持续出帧，每帧 drain 把事件应用到流式消息（同终端面板）。
func (b *agentBridge) startPump() {
	if b.pump != nil {
		return
	}
	p := animation.NewController(time.Second, animation.Linear)
	p.Repeat = true
	p.OnUpdate = func(float64) { b.drain() }
	b.pump = p
	p.Start()
}

func (b *agentBridge) stopPump() {
	if b.pump != nil {
		b.pump.Stop()
		b.pump = nil
	}
}

// approve 审批钩子（loop 协程调用，**阻塞**）：登记裁决通道 + 推一条 EventApproval（让帧泵把对应
// 活动标为「待批准」并渲染允许/拒绝按钮），然后阻塞等用户点击或 ctx 取消。
// 跨线程铁律：只在锁内写 pending/通道、不碰 Element/store。通道与事件在同一把锁内设置，
// 保证 drain 看到 EventApproval 时通道已就绪（resolveApproval 必能找到它）。
func (b *agentBridge) approve(ctx context.Context, tc agent.ToolCall) bool {
	ch := make(chan bool, 1)
	b.mu.Lock()
	b.approvalCh = ch
	b.approvalCallID = tc.ID
	b.pending = append(b.pending, agent.Event{Type: agent.EventApproval, CallID: tc.ID, Tool: tc.Function.Name, Args: tc.Function.Arguments})
	b.mu.Unlock()
	select {
	case d := <-ch:
		return d
	case <-ctx.Done():
		b.mu.Lock()
		if b.approvalCh == ch {
			b.approvalCh = nil
			b.approvalCallID = ""
		}
		b.mu.Unlock()
		return false
	}
}

// resolveApproval UI 线程（用户点「允许/拒绝」）：把裁决送给阻塞中的 loop 协程，清审批态 + 隐藏按钮。
// 结果（执行输出或「已拒绝」观察）随后由 loop 的 tool_result 事件回填。
func (b *agentBridge) resolveApproval(callID string, ok bool) {
	b.mu.Lock()
	ch := b.approvalCh
	match := ch != nil && b.approvalCallID == callID
	if match {
		b.approvalCh = nil
		b.approvalCallID = ""
	}
	b.mu.Unlock()
	if !match {
		return
	}
	ch <- ok // 缓冲=1，非阻塞
	if m := b.streamingMsg(); m != nil {
		for i := range m.Activities {
			if m.Activities[i].CallID == callID {
				m.Activities[i].AwaitingApproval = false
			}
		}
	}
	b.cs.SetState()
}

// askUser ask_user 工具处理器（loop 协程调用，**阻塞**）：登记回答通道，阻塞等用户回答或 ctx 取消。
// 问答卡 UI 由 drain 处理 EventToolCall(ask_user) 时据参数渲染（见 applyEvent），故此处只管阻塞取答。
func (b *agentBridge) askUser(ctx context.Context, args map[string]any) (string, error) {
	ch := make(chan string, 1)
	b.mu.Lock()
	b.askCh = ch
	b.mu.Unlock()
	select {
	case ans := <-ch:
		return ans, nil
	case <-ctx.Done():
		b.mu.Lock()
		if b.askCh == ch {
			b.askCh = nil
		}
		b.mu.Unlock()
		return "（用户未回答，已取消提问）", nil
	}
}

// registerAskTool 注册 ask_user（handler 闭包持有 bridge → 能阻塞等 UI 回答）。
func (b *agentBridge) registerAskTool(r *agent.Registry) {
	r.Register(&agent.Tool{
		Name: "ask_user",
		Description: "向用户提问并等待回答（用于关键决策、歧义澄清，别滥用）。question 必填；options 可选(给用户快捷选项)；" +
			"用户也可自由输入。调用会阻塞直到用户回答。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{"type": "string", "description": "要问用户的问题"},
				"options":  map[string]any{"type": "array", "description": "可选：快捷选项", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"question"},
		},
		ReadOnly: true, // 提问非写操作，免审批
		Handler:  b.askUser,
	})
}

// parseAsk 从 ask_user 工具参数解析出问答卡数据。
func parseAsk(argsJSON string) *pendingAsk {
	var pa pendingAsk
	if json.Unmarshal([]byte(argsJSON), &pa) != nil || strings.TrimSpace(pa.Question) == "" {
		return &pendingAsk{Question: "（Agent 提问，但问题为空）"}
	}
	return &pa
}

// resolveAsk UI 线程（用户点选项/输入回答）：把回答送给阻塞中的 loop 协程，清问答卡。
func (b *agentBridge) resolveAsk(answer string) {
	b.mu.Lock()
	ch := b.askCh
	b.askCh = nil
	b.mu.Unlock()
	b.cs.ask = nil
	if ch != nil {
		ch <- answer // 缓冲=1，非阻塞
	}
	b.cs.SetState()
}

// resolveAskUI 把问答卡的回答路由到单例对话面板的 bridge。
func resolveAskUI(answer string) {
	if theChatState != nil && theChatState.bridge != nil {
		theChatState.bridge.resolveAsk(answer)
	}
}

// drain 每帧（UI 线程，animation.Tick 调）把缓冲事件应用到流式消息 + 重绘；结束即停泵。
func (b *agentBridge) drain() {
	b.mu.Lock()
	evs := b.pending
	b.pending = nil
	done := !b.running
	b.mu.Unlock()

	if len(evs) > 0 {
		for _, e := range evs {
			b.applyEvent(e)
		}
		b.syncWorkspaceEdits(evs) // Agent 成功写/改文件 → 刷新文件树 + 重载已打开文件（IDE 闭环）
		b.cs.sendSeq++            // 滚到底
		b.cs.SetState()
	}
	if done {
		if msg := b.streamingMsg(); msg != nil {
			msg.Streaming = false
			if b.stopped && !strings.Contains(msg.Text, "[已停止]") {
				if strings.TrimSpace(msg.Text) != "" {
					msg.Text += "\n\n"
				}
				msg.Text += "[已停止]"
			}
			if b.cs.autoCollapse { // 收缩开关：完成即折叠本轮
				msg.Collapsed = true
			}
		}
		b.cs.ask = nil // 本轮结束：清掉残留问答卡（如被停止）
		b.stopPump()
		if len(evs) == 0 || b.stopped {
			b.cs.SetState()
		}
	}
}

// streamingMsg 取当前流式消息指针（越界返回 nil）。
func (b *agentBridge) streamingMsg() *state.Message {
	if b.runThread == nil || b.runIdx < 0 || b.runIdx >= len(b.runThread.Messages) {
		return nil
	}
	return &b.runThread.Messages[b.runIdx]
}

// applyPlan 解析 update_plan 工具参数里的计划清单，存入 chatState（置顶渲染）。
func (b *agentBridge) applyPlan(argsJSON string) {
	var p struct {
		Plan []planStep `json:"plan"`
	}
	if json.Unmarshal([]byte(argsJSON), &p) == nil && len(p.Plan) > 0 {
		b.cs.plan = p.Plan
	}
}

func (b *agentBridge) applyEvent(e agent.Event) {
	m := b.streamingMsg()
	if m == nil {
		return
	}
	switch e.Type {
	case agent.EventThinking:
		m.Thinking += e.Content
	case agent.EventContent:
		m.Text += e.Content
	case agent.EventToolCall:
		switch e.Tool {
		case "update_plan": // 计划单独渲染为置顶清单卡，不作通用工具活动行
			b.applyPlan(e.Args)
		case "ask_user": // 提问单独渲染为问答卡（输入区上方），不作通用活动行
			b.cs.ask = parseAsk(e.Args)
		default:
			m.Activities = append(m.Activities, state.Activity{CallID: e.CallID, Tool: e.Tool, Args: e.Args})
		}
	case agent.EventApproval:
		// 标记对应活动为「待批准」（tool_call 已建活动，按 CallID 找；兜底补建）。
		found := false
		for i := range m.Activities {
			if m.Activities[i].CallID == e.CallID {
				m.Activities[i].AwaitingApproval = true
				found = true
				break
			}
		}
		if !found {
			m.Activities = append(m.Activities, state.Activity{CallID: e.CallID, Tool: e.Tool, Args: e.Args, AwaitingApproval: true})
		}
	case agent.EventToolResult:
		for i := len(m.Activities) - 1; i >= 0; i-- {
			if m.Activities[i].CallID == e.CallID || (m.Activities[i].Tool == e.Tool && !m.Activities[i].Done) {
				m.Activities[i].Result = e.Content
				m.Activities[i].Done = true
				m.Activities[i].AwaitingApproval = false
				break
			}
		}
	case agent.EventFinal:
		if strings.TrimSpace(e.Content) != "" {
			m.Text = e.Content
		}
		m.Streaming = false
	case agent.EventError:
		if b.stopped {
			return // 用户主动停止：抑制底层取消错误，由 drain 收尾标 [已停止]
		}
		if strings.TrimSpace(m.Text) != "" {
			m.Text += "\n\n"
		}
		m.Text += "[错误] " + e.Content
		m.Streaming = false
	}
}

// syncWorkspaceEdits 在 Agent 成功写/改文件后，刷新文件树并重载已打开（且无未存改动）的对应文件，
// 让改动在 IDE 里即时可见（闭环）。仅 UI 线程（drain 内）调用。文件树未构建/文件未打开时均安全无操作。
func (b *agentBridge) syncWorkspaceEdits(evs []agent.Event) {
	root := b.root
	if root == "" {
		root = currentRoot()
	}
	changed := false
	for _, e := range evs {
		if e.Type != agent.EventToolResult || (e.Tool != "write_file" && e.Tool != "edit_file") {
			continue
		}
		r := strings.TrimSpace(e.Content)
		if strings.HasPrefix(r, "Error:") || strings.Contains(r, "拒绝") {
			continue // 失败或被拒，磁盘未变
		}
		changed = true
		if p := b.toolPath(root, e.CallID); p != "" {
			theEditor.reloadIfOpen(p)
		}
	}
	if changed {
		theFileTree.refresh()
	}
}

// toolPath 从活动（按 CallID 找）参数 JSON 取 path，解析为工作区内绝对路径（与编辑器标签路径同形可比）。
func (b *agentBridge) toolPath(root, callID string) string {
	m := b.streamingMsg()
	if m == nil {
		return ""
	}
	for _, a := range m.Activities {
		if a.CallID != callID {
			continue
		}
		var args map[string]any
		if json.Unmarshal([]byte(a.Args), &args) != nil {
			return ""
		}
		p, _ := args["path"].(string)
		if strings.TrimSpace(p) == "" {
			return ""
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		return filepath.Clean(p)
	}
	return ""
}

// buildProvider 选 Provider：优先用「设置」里配的（帮助→打开设置），否则回退环境变量。无 key 返回 nil。
func buildProvider() agent.Provider {
	if settingsConfigured() {
		return &agent.OpenAIProvider{BaseURL: theSettings.BaseURL, APIKey: theSettings.APIKey, Model: mainModel(),
			Temperature: settingsTemperature(), MaxTokens: theSettings.MaxTokens}
	}
	for _, c := range []struct{ env, base, model string }{
		{"DEEPSEEK_API_KEY", "https://api.deepseek.com/v1", "deepseek-chat"},
		{"OPENAI_API_KEY", "https://api.openai.com/v1", "gpt-4o"},
		{"DASHSCOPE_API_KEY", "https://dashscope.aliyuncs.com/compatible-mode/v1", "qwen-plus"},
		{"MOONSHOT_API_KEY", "https://api.moonshot.cn/v1", "moonshot-v1-8k"},
		{"OPENROUTER_API_KEY", "https://openrouter.ai/api/v1", "anthropic/claude-3.5-sonnet"},
	} {
		if k := os.Getenv(c.env); k != "" {
			return &agent.OpenAIProvider{BaseURL: c.base, APIKey: k, Model: c.model, Temperature: -1}
		}
	}
	return nil
}

// ─── Agent 消息卡富渲染（头 + 思考 + 工具活动 + 正文）──────────

func agentMessageCard(m state.Message, onToggleCollapse, onToggleThinking func(), onToggleActivity func(int)) widget.Widget {
	collapsed := !m.Streaming && m.Collapsed
	kids := []widget.Widget{agentHeaderCollapsible(m, onToggleCollapse)}
	if !collapsed {
		if strings.TrimSpace(m.Thinking) != "" {
			kids = append(kids, vgap(6), thinkingBlock(m, onToggleThinking))
		}
		for i, a := range m.Activities {
			ai := i
			kids = append(kids, vgap(6), activityRow(a, func() { onToggleActivity(ai) }))
		}
		if txt := strings.TrimSpace(m.Text); txt != "" {
			kids = append(kids, vgap(8), mdRender(m.Text)) // 正文走 Markdown 渲染（代码块/标题/列表）
		} else if m.Streaming && len(m.Activities) == 0 && strings.TrimSpace(m.Thinking) == "" {
			kids = append(kids, vgap(6), label("思考中…", ghTextMuted, 12))
		}
	}
	card := widget.Div(
		widget.Style{BackgroundColor: ghBgTertiary, BorderColor: ghBorder, BorderWidth: 1, BorderRadius: 6,
			Padding: types.EdgeInsetsLTRB(14, 10, 14, 10), FlexDirection: "column", AlignItems: "stretch"},
		kids,
	)
	// 复刻参考 border-left:3px 状态色（运行中黄 / 出错红 / 完成绿）：3px 竖条 + 卡片并排撑同高。
	return widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "stretch"},
		widget.Div(widget.Style{Width: 3, BackgroundColor: agentStatusColor(m), BorderRadius: 1.5}),
		widget.Div(widget.Style{Width: 6}),
		expand(card),
	)
}

// agentStatusColor Agent 卡左条状态色：运行中黄 / 出错红 / 完成绿。
func agentStatusColor(m state.Message) *types.Color {
	switch {
	case m.Streaming:
		return ghWarning
	case strings.Contains(m.Text, "[错误]"):
		return ghDanger
	default:
		return ghSuccess
	}
}

// agentHeaderCollapsible Agent 卡头：bot + Agent + 运行中点；完成后变可点折叠头（chevron + 折叠时摘要）。
func agentHeaderCollapsible(m state.Message, onToggle func()) widget.Widget {
	done := !m.Streaming
	var row []widget.Widget
	if done {
		ic := "chevron-down"
		if m.Collapsed {
			ic = "chevron-right"
		}
		row = append(row, widget.Lucide(ic, widget.IconSize(12), widget.IconColor(ghTextMuted)), hgap(4))
	}
	row = append(row,
		widget.Lucide("bot", widget.IconSize(13), widget.IconColor(*ghAccent)),
		hgap(6),
		label("Agent", ghTextSec, 11),
	)
	switch {
	case m.Streaming:
		row = append(row, hgap(8), statusDot(ghWarning), hgap(4), label("运行中", ghTextMuted, 10))
	case m.Collapsed:
		row = append(row, hgap(8), expand(label(collapseSummary(m), ghTextMuted, 11)))
	}
	header := widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, row)
	if !done {
		return header
	}
	return &widget.Clickable{SingleChildWidget: widget.SingleChildWidget{Child: header}, OnClick: onToggle}
}

// collapseSummary 折叠态摘要：正文首行（去标题号）；无正文则按活动数。
func collapseSummary(m state.Message) string {
	if s := strings.TrimSpace(strings.TrimLeft(firstLine(m.Text), "# ")); s != "" {
		return truncRunes(s, 56)
	}
	if n := len(m.Activities); n > 0 {
		return "已完成 · " + strconv.Itoa(n) + " 步"
	}
	return "已完成"
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// thinkingBlock 思考块：可折叠头（chevron + 思考）；展开看全文、折叠看首行；流式时强制展开看实时。
func thinkingBlock(m state.Message, onToggle func()) widget.Widget {
	expanded := m.Streaming || m.ThinkingExpanded
	ic := "chevron-right"
	if expanded {
		ic = "chevron-down"
	}
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(ic, widget.IconSize(11), widget.IconColor(ghTextMuted)),
			hgap(4),
			label("思考", ghTextMuted, 10),
		)},
		OnClick: onToggle,
	}
	kids := []widget.Widget{header}
	if expanded {
		kids = append(kids, vgap(3), label(strings.TrimSpace(m.Thinking), ghTextSec, 11))
	} else {
		kids = append(kids, vgap(2), label(truncRunes(firstLine(m.Thinking), 48), ghTextMuted, 10))
	}
	return widget.Div(
		widget.Style{BackgroundColor: ghBgPrimary, BorderRadius: 4, Padding: types.EdgeInsetsLTRB(8, 6, 8, 6),
			FlexDirection: "column", AlignItems: "stretch"},
		kids,
	)
}

// activityRow 一次工具调用：[chevron] 工具图标(进行蓝/完成绿/待批准黄) + 名 + 参数预览；
// 待批准时附「允许/拒绝」按钮条；有结果时头可点折叠——折叠看首行预览、展开看全量(等宽)。
func activityRow(a state.Activity, onToggle func()) widget.Widget {
	iconCol := *ghAccent
	switch {
	case a.AwaitingApproval:
		iconCol = *ghWarning
	case a.Done:
		iconCol = *ghSuccess
	}
	hasResult := a.Done && strings.TrimSpace(a.Result) != ""
	var headRow []widget.Widget
	if hasResult {
		ic := "chevron-right"
		if a.Expanded {
			ic = "chevron-down"
		}
		headRow = append(headRow, widget.Lucide(ic, widget.IconSize(11), widget.IconColor(ghTextMuted)), hgap(4))
	} else {
		headRow = append(headRow, widget.Div(widget.Style{Width: 15})) // 无 chevron 时占位对齐
	}
	headRow = append(headRow,
		widget.Lucide(iconForTool(a.Tool), widget.IconSize(12), widget.IconColor(iconCol)),
		hgap(6),
		label(a.Tool, ghText, 11),
		hgap(6),
		expand(label(argPreview(a.Args), ghTextMuted, 11)),
	)
	headDiv := widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, headRow)
	var head widget.Widget = headDiv
	if hasResult { // 头可点折叠/展开
		head = &widget.Clickable{SingleChildWidget: widget.SingleChildWidget{Child: headDiv}, OnClick: onToggle}
	}

	kids := []widget.Widget{head}
	switch {
	case a.AwaitingApproval:
		kids = append(kids, vgap(6), approvalBar(a.CallID))
	case hasResult && a.Expanded:
		kids = append(kids, vgap(4), activityResultBody(a.Result))
	case hasResult:
		kids = append(kids, vgap(3), label(truncRunes(mdExpandTabs(firstLine(a.Result)), 88), ghTextSec, 10))
	}
	border := ghBorder
	if a.AwaitingApproval {
		border = ghWarning
	}
	return widget.Div(
		widget.Style{BackgroundColor: ghBgPrimary, BorderColor: border, BorderWidth: 1, BorderRadius: 4,
			Padding: types.EdgeInsetsLTRB(8, 6, 8, 6), FlexDirection: "column", AlignItems: "stretch"},
		kids,
	)
}

// activityResultBody 展开态工具结果：等宽多行（tab→空格、截断 4000 防撑爆）。
func activityResultBody(result string) widget.Widget {
	t := widget.NewText(mdExpandTabs(truncRunes(strings.TrimSpace(result), 4000)), ghTextSec)
	t.Font = mdMonoFont
	return t
}

// approvalBar 待批准操作的提示 + 允许/拒绝按钮（手动审核模式）。点击经单例 bridge 把裁决送回阻塞的 loop。
func approvalBar(callID string) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
		widget.Lucide("shield-alert", widget.IconSize(12), widget.IconColor(*ghWarning)),
		hgap(6),
		label("等待批准", *ghWarning, 11),
		expand(widget.Div(widget.Style{})),
		&widget.Button{
			Icon: "check", IconSize: 12, IconGap: 4, Text: "允许", TextColor: cWhite, FontSize: 11,
			OnClick: func() { resolveApprovalUI(callID, true) },
			Color:   *ghSuccess, BorderRadius: 5, Padding: types.EdgeInsetsLTRB(10, 3, 11, 3), MinHeight: 12,
		},
		hgap(6),
		&widget.Button{
			Icon: "x", IconSize: 12, IconGap: 4, Text: "拒绝", TextColor: cWhite, FontSize: 11,
			OnClick: func() { resolveApprovalUI(callID, false) },
			Color:   *ghDanger, BorderRadius: 5, Padding: types.EdgeInsetsLTRB(10, 3, 11, 3), MinHeight: 12,
		},
	)
}

// resolveApprovalUI 把按钮点击路由到单例对话面板的 bridge（与 theChatState 单例约定一致）。
func resolveApprovalUI(callID string, ok bool) {
	if theChatState != nil && theChatState.bridge != nil {
		theChatState.bridge.resolveApproval(callID, ok)
	}
}

func iconForTool(name string) string {
	switch { // 前缀族：git_* / memory_* / mcp.*
	case strings.HasPrefix(name, "git_"):
		return "git-branch"
	case strings.HasPrefix(name, "memory_"):
		return "file-text"
	}
	switch name {
	case "read_file", "write_file":
		return "file-text"
	case "edit_file", "multi_edit":
		return "file-code"
	case "list_files":
		return "folder"
	case "search_content", "search_files":
		return "search"
	case "run_command", "run_background", "read_output":
		return "terminal"
	case "kill_process":
		return "circle-x"
	case "web_fetch", "web_search":
		return "globe"
	case "move_file":
		return "square-pen"
	case "delete_file":
		return "trash-2"
	}
	return "braces"
}

// argPreview 取关键参数（path/command/pattern）作预览，截断。
func argPreview(argsJSON string) string {
	var m map[string]any
	if json.Unmarshal([]byte(argsJSON), &m) == nil {
		for _, k := range []string{"path", "command", "pattern", "old_string"} {
			if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
				return truncRunes(strings.TrimSpace(v), 64)
			}
		}
	}
	return truncRunes(strings.TrimSpace(argsJSON), 64)
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func vgap(h float64) widget.Widget { return widget.Div(widget.Style{Height: h}) }
func hgap(w float64) widget.Widget { return widget.Div(widget.Style{Width: w}) }
