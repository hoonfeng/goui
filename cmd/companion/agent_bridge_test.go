//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/goui/cmd/companion/agent"
	"github.com/user/goui/cmd/companion/state"
	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/widget"
)

// TestAutonomousParams 自主模式：追加计划提示 + 放宽迭代上限；非自主：原样 + 默认上限。
func TestAutonomousParams(t *testing.T) {
	tn, n := autonomousParams("做点事", false)
	if tn != "做点事" || n != 30 {
		t.Errorf("非自主 = (%q,%d)，期望原样 + 30", tn, n)
	}
	ta, a := autonomousParams("做点事", true)
	if !strings.Contains(ta, "做点事") || !strings.Contains(ta, "update_plan") || a != 60 {
		t.Errorf("自主 = (%q,%d)，期望含原任务 + 计划提示 + 60", ta, a)
	}
}

// TestAskUserResolve ask_user：parseAsk 解析问题/选项；askUser 阻塞 → resolveAsk 送回答案。
func TestAskUserResolve(t *testing.T) {
	pa := parseAsk(`{"question":"继续吗","options":["是","否"]}`)
	if pa.Question != "继续吗" || len(pa.Options) != 2 {
		t.Fatalf("parseAsk = %+v", pa)
	}

	theChatState = &chatState{store: state.NewChatStore()}
	b := &agentBridge{cs: theChatState}
	theChatState.bridge = b

	ansCh := make(chan string, 1)
	go func() {
		a, _ := b.askUser(context.Background(), map[string]any{"question": "继续吗"})
		ansCh <- a
	}()
	for i := 0; i < 200; i++ { // 等 askCh 就绪
		b.mu.Lock()
		ready := b.askCh != nil
		b.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	b.resolveAsk("是")
	select {
	case a := <-ansCh:
		if a != "是" {
			t.Errorf("answer = %q，期望『是』", a)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("askUser 未返回")
	}
}

// 端到端（无窗口、无网络）：注入 MockProvider 的 loop，send 一条任务 → goroutine 跑 TAOR →
// 事件经动画帧泵 drain → 流式写进当前助手消息。手动 animation.Tick + EnsureLayout 模拟主循环。
func TestAgentBridgeStreamsIntoChat(t *testing.T) {
	// 复位聊天单例（ChatPanel.CreateState 返回它）。
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true}

	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "note.txt"), []byte("AGENT_SEES_ME"), 0o644)
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "c1", Type: "function", Function: agent.FunctionCall{Name: "read_file", Arguments: `{"path":"note.txt"}`}}}},
		{Content: "笔记内容是 AGENT_SEES_ME [FINAL]"},
	}}
	// 预置 loop（带 mock），跳过 buildProvider（无 key）。
	theChatState.bridge = &agentBridge{cs: theChatState,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}

	// 挂 pipeline：SetState 需挂载的 Element + 全局钩子。
	animation.ResetScheduler()
	pipe := render.NewPipeline(360, 600, canvas.NewSoftCanvas(360, 600))
	rootEl := widget.CreateElementFor(&ChatPanel{})
	rootEl.Mount(nil, 0)
	pipe.SetRootElement(rootEl)
	widget.OnNeedsRepaint = func() { pipe.MarkNeedsRepaint() }
	widget.OnNeedsLayout = func() { pipe.MarkNeedsLayout() }
	defer func() { widget.OnNeedsRepaint = nil; widget.OnNeedsLayout = nil }()
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()

	theChatState.store.Draft = "读 note.txt 告诉我内容"
	theChatState.send()

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		animation.Tick(time.Now()) // 推进帧泵 → drain（应用事件 + SetState）
		pipe.EnsureLayout()        // 消费 relayout

		th := theChatState.store.Active()
		last := th.Messages[len(th.Messages)-1]
		gotContent := strings.Contains(last.Text, "AGENT_SEES_ME")
		gotActivity := false
		for _, a := range last.Activities {
			if a.Tool == "read_file" && a.Done && strings.Contains(a.Result, "AGENT_SEES_ME") {
				gotActivity = true
			}
		}
		idle := !theChatState.bridge.isRunning()
		if gotContent && gotActivity && idle && !last.Streaming {
			if err := pipe.Render(); err != nil { // 含工具活动+正文的消息卡渲染不报错
				t.Fatalf("含 Agent 回复的聊天渲染失败: %v", err)
			}
			if theChatState.bridge.pump != nil {
				t.Error("结束后帧泵应已停")
			}
			return
		}
		time.Sleep(16 * time.Millisecond)
	}
	last := theChatState.store.Active().Messages
	t.Errorf("超时：Agent 回复未完整流入聊天。末条=%+v", last[len(last)-1])
}

// 手动审核：写类工具执行前出现「待批准」活动；用户「允许」后才写盘、活动完成。
func TestAgentBridgeApprovalApprove(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: false} // 手动审核
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "w1", Type: "function", Function: agent.FunctionCall{Name: "write_file", Arguments: `{"path":"out.txt","content":"APPROVED"}`}}}},
		{Content: "已写入 [FINAL]"},
	}}
	theChatState.bridge = &agentBridge{cs: theChatState,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	theChatState.store.Draft = "写 out.txt"
	theChatState.send()

	// 阶段一：等到 write_file 活动进入「待批准」，且尚未写盘。
	var callID string
	if !pumpUntil(t, pipe, func() bool {
		for _, a := range lastActivities() {
			if a.Tool == "write_file" && a.AwaitingApproval {
				callID = a.CallID
				return true
			}
		}
		return false
	}, 5*time.Second) {
		t.Fatal("超时：未出现待批准的 write_file 活动")
	}
	if _, e := os.Stat(filepath.Join(root, "out.txt")); e == nil {
		t.Error("批准前不应写盘")
	}

	// 阶段二：批准 → 写盘、循环结算。
	resolveApprovalUI(callID, true)
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(theChatState.bridge) }, 5*time.Second) {
		t.Fatal("超时：批准后未结算")
	}
	if b, _ := os.ReadFile(filepath.Join(root, "out.txt")); string(b) != "APPROVED" {
		t.Errorf("写入内容 = %q", b)
	}
	if err := pipe.Render(); err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
}

// 手动审核：用户「拒绝」后不写盘，拒绝结果回填到活动。
func TestAgentBridgeApprovalReject(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: false}
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "w1", Type: "function", Function: agent.FunctionCall{Name: "write_file", Arguments: `{"path":"out.txt","content":"X"}`}}}},
		{Content: "好的，不写了 [FINAL]"},
	}}
	theChatState.bridge = &agentBridge{cs: theChatState,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	theChatState.store.Draft = "写 out.txt"
	theChatState.send()

	var callID string
	if !pumpUntil(t, pipe, func() bool {
		for _, a := range lastActivities() {
			if a.AwaitingApproval {
				callID = a.CallID
				return true
			}
		}
		return false
	}, 5*time.Second) {
		t.Fatal("超时：未出现待批准活动")
	}

	resolveApprovalUI(callID, false) // 拒绝
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(theChatState.bridge) }, 5*time.Second) {
		t.Fatal("超时：拒绝后未结算")
	}
	if _, e := os.Stat(filepath.Join(root, "out.txt")); e == nil {
		t.Error("拒绝后不应写盘")
	}
	var rejected bool
	for _, a := range lastActivities() {
		if a.Done && strings.Contains(a.Result, "拒绝") {
			rejected = true
		}
	}
	if !rejected {
		t.Error("拒绝结果应回填到活动（含『拒绝』字样）")
	}
}

// Agent 成功写文件后，已打开且无未存改动的编辑器标签应被重载为新内容（IDE 闭环）。
func TestAgentBridgeReloadsEditorAfterEdit(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true} // 自动审核：写直接执行
	theEditor = &editorState{}                                               // 复位编辑器单例
	theFileTree = &fileTreeState{}                                           // 复位文件树单例
	root := t.TempDir()
	docAbs := filepath.Join(root, "doc.txt")
	os.WriteFile(docAbs, []byte("OLD"), 0o644)
	theEditor.open(docAbs) // 编辑器打开（读入 "OLD"）

	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "w1", Type: "function", Function: agent.FunctionCall{Name: "write_file", Arguments: `{"path":"doc.txt","content":"NEW"}`}}}},
		{Content: "已更新 [FINAL]"},
	}}
	theChatState.bridge = &agentBridge{cs: theChatState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	theChatState.store.Draft = "把 doc.txt 改成 NEW"
	theChatState.send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(theChatState.bridge) }, 5*time.Second) {
		t.Fatal("超时：未结算")
	}
	if b, _ := os.ReadFile(docAbs); string(b) != "NEW" {
		t.Fatalf("磁盘应为 NEW，得 %q", b)
	}
	var tab *editorTab
	for _, tt := range theEditor.tabs {
		if tt.path == docAbs {
			tab = tt
		}
	}
	if tab == nil {
		t.Fatal("编辑器应仍打开 doc.txt 标签")
	}
	if tab.content != "NEW" {
		t.Errorf("编辑器标签应被重载为 NEW，得 %q", tab.content)
	}
}

// reloadIfOpen：已打开且无改动→重载；dirty→不覆盖；未打开→不动。
func TestEditorReloadIfOpen(t *testing.T) {
	theEditor = &editorState{}
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	os.WriteFile(p, []byte("V1"), 0o644)
	theEditor.open(p)

	os.WriteFile(p, []byte("V2"), 0o644)
	if !theEditor.reloadIfOpen(p) {
		t.Fatal("无改动的已打开标签应被重载")
	}
	if theEditor.tabs[0].content != "V2" {
		t.Errorf("重载后内容应为 V2，得 %q", theEditor.tabs[0].content)
	}

	theEditor.tabs[0].dirty = true
	os.WriteFile(p, []byte("V3"), 0o644)
	if theEditor.reloadIfOpen(p) {
		t.Error("dirty 标签不应被覆盖")
	}
	if theEditor.tabs[0].content != "V2" {
		t.Errorf("dirty 后内容不应变，得 %q", theEditor.tabs[0].content)
	}

	if theEditor.reloadIfOpen(filepath.Join(dir, "nope.txt")) {
		t.Error("未打开的文件不应重载")
	}
}

// blockingProvider 的 Chat 阻塞到 ctx 取消（模拟长任务，用于测停止）。
type blockingProvider struct{}

func (blockingProvider) Name() string { return "block" }
func (blockingProvider) Chat(ctx context.Context, m []agent.Message, td []agent.ToolDefinition, oc func(agent.Chunk)) (agent.Message, error) {
	<-ctx.Done()
	return agent.Message{}, ctx.Err()
}

// 停止按钮：运行中 stop() 取消 ctx → loop 退出、消息收尾标 [已停止]（不显示底层取消错误）。
func TestAgentBridgeStop(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true}
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, t.TempDir())
	theChatState.bridge = &agentBridge{cs: theChatState, root: t.TempDir(),
		loop: &agent.Loop{Provider: blockingProvider{}, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	theChatState.store.Draft = "跑个长任务"
	theChatState.send()
	if !pumpUntil(t, pipe, func() bool { return theChatState.bridge.isRunning() }, 3*time.Second) {
		t.Fatal("应进入运行态")
	}
	theChatState.bridge.stop()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(theChatState.bridge) }, 3*time.Second) {
		t.Fatal("停止后应结算")
	}
	msgs := theChatState.store.Active().Messages
	if last := msgs[len(msgs)-1]; !strings.Contains(last.Text, "已停止") {
		t.Errorf("停止后末条应含 [已停止]，得 %q", last.Text)
	}
	if strings.Contains(msgs[len(msgs)-1].Text, "[错误]") {
		t.Error("用户主动停止不应显示 [错误]")
	}
}

// Agent 卡 Markdown 正文（标题/列表/行内/代码块）+ 折叠两态都能无错渲染。
func TestAgentCardMarkdownAndCollapseRender(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true}
	th := theChatState.store.Active()
	th.Messages = append(th.Messages, state.Message{
		Role:       state.Assistant,
		Thinking:   "先想一下方案…",
		Activities: []state.Activity{{CallID: "1", Tool: "read_file", Args: `{"path":"x.go"}`, Result: "ok", Done: true}},
		Text:       "# 标题\n\n说明文字 **加粗** 与 `行内代码`。\n\n- 项目一\n- 项目二\n\n```go\nfunc main() {}\n```\n",
	})
	pipe := newChatPipe(t)
	if err := pipe.Render(); err != nil {
		t.Fatalf("展开态渲染失败: %v", err)
	}
	// 折叠后再渲染（走折叠摘要路径）
	theChatState.store.Active().Messages[1].Collapsed = true
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()
	if err := pipe.Render(); err != nil {
		t.Fatalf("折叠态渲染失败: %v", err)
	}
}

// autoCollapse(收缩) 开：一轮完成后该助手消息应自动折叠。
func TestAgentBridgeAutoCollapse(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true, autoCollapse: true}
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, t.TempDir())
	mock := &agent.MockProvider{Responses: []agent.Message{{Content: "完成了 [FINAL]"}}}
	theChatState.bridge = &agentBridge{cs: theChatState, root: t.TempDir(),
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	theChatState.store.Draft = "干活"
	theChatState.send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(theChatState.bridge) }, 5*time.Second) {
		t.Fatal("超时：未结算")
	}
	msgs := theChatState.store.Active().Messages
	if last := msgs[len(msgs)-1]; !last.Collapsed {
		t.Error("autoCollapse 开，完成后该消息应自动折叠")
	}
}

// 消息操作：删除移除该条。
func TestChatMessageDelete(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true, hoveredMsg: -1}
	th := theChatState.store.Active()
	th.Messages = append(th.Messages, state.Message{Role: state.User, Text: "hi"}, state.Message{Role: state.Assistant, Text: "yo"})
	n := len(th.Messages)
	theChatState.deleteMessage(th, n-1)
	if len(th.Messages) != n-1 {
		t.Fatalf("删后应 %d 条，得 %d", n-1, len(th.Messages))
	}
	if th.Messages[len(th.Messages)-1].Text == "yo" {
		t.Error("末条 yo 应已删除")
	}
}

// 消息操作：重新生成删掉末条助手回复并重跑一轮。
func TestChatRegenerate(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true, hoveredMsg: -1}
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{{Content: "重新生成的回复 [FINAL]"}}}
	theChatState.bridge = &agentBridge{cs: theChatState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)
	th := theChatState.store.Active()
	th.Messages = append(th.Messages, state.Message{Role: state.User, Text: "原任务"}, state.Message{Role: state.Assistant, Text: "旧回复"})

	theChatState.regenerate(th, len(th.Messages)-1)
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(theChatState.bridge) }, 5*time.Second) {
		t.Fatal("超时：重新生成未结算")
	}
	last := theChatState.store.Active().Messages
	if l := last[len(last)-1]; l.Role != state.Assistant || !strings.Contains(l.Text, "重新生成") {
		t.Errorf("末条应为新助手回复，得 %+v", l)
	}
	for _, m := range last {
		if m.Text == "旧回复" {
			t.Error("旧回复应被删除替换")
		}
	}
}

// Ctrl+F 搜索：匹配计数 + 过滤渲染无错；hover 操作叠加渲染无错。
func TestChatSearchAndHoverRender(t *testing.T) {
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true, hoveredMsg: -1}
	th := theChatState.store.Active()
	th.Messages = []state.Message{
		{Role: state.User, Text: "实现快速排序"},
		{Role: state.Assistant, Text: "好的，用 Go 写 quicksort"},
		{Role: state.User, Text: "再加个二分查找"},
	}
	theChatState.searchQuery = "quicksort"
	if c := theChatState.searchMatchCount(); c != 1 {
		t.Errorf("quicksort 应匹配 1 条，得 %d", c)
	}
	theChatState.searchQuery = "查找"
	if c := theChatState.searchMatchCount(); c != 1 {
		t.Errorf("『查找』应匹配 1 条，得 %d", c)
	}
	theChatState.searchQuery = "不存在XYZ"
	if c := theChatState.searchMatchCount(); c != 0 {
		t.Errorf("无匹配应 0，得 %d", c)
	}
	// 开搜索 + 过滤渲染
	theChatState.showSearch = true
	theChatState.searchQuery = "查找"
	widget.ClipboardWrite = func(string) {}
	defer func() { widget.ClipboardWrite = nil }()
	pipe := newChatPipe(t)
	if err := pipe.Render(); err != nil {
		t.Fatalf("搜索过滤渲染失败: %v", err)
	}
	// hover 末条 → 操作按钮叠加渲染
	theChatState.showSearch = false
	theChatState.searchQuery = ""
	theChatState.hoveredMsg = len(th.Messages) - 1
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()
	if err := pipe.Render(); err != nil {
		t.Fatalf("hover 操作叠加渲染失败: %v", err)
	}
}

// ─── 测试辅助 ─────────────────────────────────────────────

// newChatPipe 挂一个最小渲染管线（SetState 需挂载 Element + 全局钩子），返回管线。
func newChatPipe(t *testing.T) *render.Pipeline {
	t.Helper()
	animation.ResetScheduler()
	pipe := render.NewPipeline(360, 600, canvas.NewSoftCanvas(360, 600))
	rootEl := widget.CreateElementFor(&ChatPanel{})
	rootEl.Mount(nil, 0)
	pipe.SetRootElement(rootEl)
	widget.OnNeedsRepaint = func() { pipe.MarkNeedsRepaint() }
	widget.OnNeedsLayout = func() { pipe.MarkNeedsLayout() }
	t.Cleanup(func() { widget.OnNeedsRepaint = nil; widget.OnNeedsLayout = nil })
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()
	return pipe
}

// pumpUntil 手动逐帧推进（animation.Tick→帧泵 drain + EnsureLayout），直到 cond 成立或超时。
func pumpUntil(t *testing.T, pipe *render.Pipeline, cond func() bool, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		animation.Tick(time.Now())
		pipe.EnsureLayout()
		if cond() {
			return true
		}
		time.Sleep(8 * time.Millisecond)
	}
	return false
}

// lastActivities 取当前会话最后一条消息的工具活动。
func lastActivities() []state.Activity {
	th := theChatState.store.Active()
	if th == nil || len(th.Messages) == 0 {
		return nil
	}
	return th.Messages[len(th.Messages)-1].Activities
}

// bridgeSettled 桥彻底结算：循环已结束(running=false)且帧泵已停(pump=nil)。stopPump 只在
// 「done 帧」里、把所有 pending 事件 drain 应用之后调用，故 pump==nil ⟹ 全部事件已落到消息上。
// （注意：单看 !isRunning() 会过早——goroutine 在 Tick 与 cond 间隙置 running=false 时，
// 末批事件可能还在 pending 未 drain；真实 app 主循环会继续出帧直到泵自停，测试须同样等到结算。）
func bridgeSettled(b *agentBridge) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.running && b.pump == nil
}

// 无 API key 时 send 给出明确提示、不挂起（不建 loop、不进循环）。
func TestAgentBridgeNoKeyHint(t *testing.T) {
	if buildProvider() != nil {
		t.Skip("环境配了 API key，跳过无-key 提示测试（避免真网络调用）")
	}
	theChatState = &chatState{store: state.NewChatStore(), autoReview: true}
	// 确保环境无 key（测试环境本就无，双保险不依赖）。
	theChatState.bridge = &agentBridge{cs: theChatState} // loop=nil → start 走 buildProvider
	theChatState.store.Draft = "你好"
	theChatState.send()

	th := theChatState.store.Active()
	last := th.Messages[len(th.Messages)-1]
	if last.Role != state.Assistant || !strings.Contains(last.Text, "API key") {
		t.Errorf("无 key 应给出配置提示，得 %+v", last)
	}
	if theChatState.bridge.isRunning() {
		t.Error("无 key 不应进入运行态")
	}
}
