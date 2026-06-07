package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// EventType 循环对外广播的事件类型（供 UI 流式展示）。
type EventType string

const (
	EventThinking   EventType = "thinking"    // LLM 思考链增量
	EventContent    EventType = "content"     // LLM 正文增量
	EventToolCall   EventType = "tool_call"   // 即将执行某工具
	EventToolResult EventType = "tool_result" // 工具结果回来
	EventFinal      EventType = "final"       // 任务完成（最终答复）
	EventError      EventType = "error"       // 出错/止损
	// EventApproval 等待用户审批某次写类工具调用。由宿主（UI 桥）在 Approve 钩子里 emit，
	// loop 自身不直接发——loop 只通过 Approve 回调阻塞等待裁决（见 agent_bridge.go）。
	EventApproval EventType = "approval"
)

// Event 一条循环事件。
type Event struct {
	Type    EventType
	Content string // thinking/content/final/error/tool_result 的文本
	Tool    string // tool_call/tool_result 的工具名
	Args    string // tool_call 的参数 JSON
	CallID  string
}

const finalMarker = "[FINAL]"

// Loop TAOR 编排器：think(LLM 决策)→act(执行工具)→observe(结果回灌)→repeat。
// 停止：assistant 输出 [FINAL] 或无工具调用（即给出文本答复）/ 连续 3 轮工具全错 / 达最大迭代 / 外部取消。
type Loop struct {
	Provider      Provider
	Registry      *Registry
	System        string // 系统提示词
	MaxIterations int    // 默认 30
	OnEvent       func(Event)
	// Approve 审批钩子（可空）。设置后，每次执行 RequiresApproval 的写类工具前调用它，
	// 返回 false 即拒绝执行（把拒绝作为观察回灌、让模型改道）。只读工具永不经过它。
	// nil = 自动审核（全部放行）。宿主可在此阻塞等待用户点「允许/拒绝」（见 agent_bridge.go）。
	Approve func(ctx context.Context, tc ToolCall) bool
}

func (l *Loop) emit(e Event) {
	if l.OnEvent != nil {
		l.OnEvent(e)
	}
}

// Run 跑一轮任务。history 为先前对话（可空，不含本次 task）。
// 返回在 history 基础上追加了 system(首轮)/user/assistant/tool 等本轮全部消息的完整对话。
func (l *Loop) Run(ctx context.Context, task string, history []Message) ([]Message, error) {
	max := l.MaxIterations
	if max <= 0 {
		max = 30
	}
	msgs := make([]Message, 0, len(history)+4)
	if l.System != "" && !hasSystem(history) {
		msgs = append(msgs, Message{Role: RoleSystem, Content: l.System})
	}
	msgs = append(msgs, history...)
	msgs = append(msgs, Message{Role: RoleUser, Content: task})

	tools := l.Registry.Definitions()
	consecErr := 0

	for iter := 0; iter < max; iter++ {
		if err := ctx.Err(); err != nil {
			return msgs, err // 外部取消
		}

		// ── THINK：LLM 决策（流式 thinking/content 经事件外发）──
		assistant, err := l.Provider.Chat(ctx, msgs, tools, func(c Chunk) {
			if c.Reasoning != "" {
				l.emit(Event{Type: EventThinking, Content: c.Reasoning})
			}
			if c.Content != "" {
				l.emit(Event{Type: EventContent, Content: c.Content})
			}
		})
		if err != nil {
			l.emit(Event{Type: EventError, Content: err.Error()})
			return msgs, err
		}
		msgs = append(msgs, assistant)

		// ── 完成判定：[FINAL] 标记，或无工具调用（视作已给出文本答复）──
		if strings.Contains(assistant.Content, finalMarker) || len(assistant.ToolCalls) == 0 {
			l.emit(Event{Type: EventFinal, Content: stripFinal(assistant.Content)})
			return msgs, nil
		}

		// ── ACT + OBSERVE：依次执行工具，结果作 role=tool 消息回灌 ──
		iterErr := false
		for _, tc := range assistant.ToolCalls {
			l.emit(Event{Type: EventToolCall, Tool: tc.Function.Name, Args: tc.Function.Arguments, CallID: tc.ID})

			// 审批门：写类工具（RequiresApproval）在手动审核下需用户批准。被拒则不执行，
			// 把拒绝作为观察回灌（让模型改道，而非当成工具错误计入连续失败）。
			if l.Approve != nil {
				if tool, ok := l.Registry.Get(tc.Function.Name); ok && tool.RequiresApproval && !l.Approve(ctx, tc) {
					rej := "用户拒绝了此操作。请勿重试该操作；改用其他方式达成目标，或先向用户说明你为何需要它。"
					l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: rej, CallID: tc.ID})
					msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: rej})
					continue
				}
			}

			result, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if terr != nil {
				result = "Error: " + terr.Error()
				iterErr = true
			}
			l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: result, CallID: tc.ID})
			msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: result})
		}

		// 连续 3 轮工具全有错 → 止损停（复刻参考源 3-consecutive-error）。
		if iterErr {
			if consecErr++; consecErr >= 3 {
				l.emit(Event{Type: EventError, Content: "连续 3 轮工具执行失败，已停止"})
				return msgs, nil
			}
		} else {
			consecErr = 0
		}
	}
	l.emit(Event{Type: EventError, Content: "已达最大迭代数，停止"})
	return msgs, nil
}

func hasSystem(msgs []Message) bool {
	for _, m := range msgs {
		if m.Role == RoleSystem {
			return true
		}
	}
	return false
}

func stripFinal(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, finalMarker, ""))
}

// DefaultSystemPrompt 复刻参考源核心铁律的系统提示词（中文 lock / 改前 read / 工作区限定 / [FINAL]）。
func DefaultSystemPrompt(workspaceRoot string) string {
	return "你是「伴随式 CodeAgent」，一个在用户项目里结对编程的 AI 编码助手。\n\n" +
		"# 工作区\n根目录: " + workspaceRoot + "\n\n" +
		"# 核心铁律（不可违背）\n" +
		"- 所有思考与回答均使用中文。\n" +
		"- 文件操作只用工作区内路径；修改文件前必须先 read_file 确认当前内容。\n" +
		"- 每次工具调用后，依据真实结果决定下一步，绝不臆测结果。\n" +
		"- 禁止破坏性命令（如 rm -rf、强制 push main），禁止修改工作区外文件。\n" +
		"- 【完成标记】任务彻底完成时，在最终答复末尾输出 [FINAL]。\n\n" +
		"# 工具\n" +
		"- 浏览定位：search_files（按通配符找文件）、search_content（按正则搜内容，返回 路径:行号）、list_files。\n" +
		"- 读改：read_file（改前必读）、edit_file（小处精确替换，首选）、multi_edit（一个文件多处替换、一次搞定）、write_file（整文件覆盖/新建）、move_file（移动/重命名）、delete_file（删文件）。\n" +
		"- 运行：run_command（构建/测试等同步命令，会等结果）；长命令（dev server/watch）改用 run_background 后台启动，再 read_output 看输出、kill_process 停。\n" +
		"- 联网：web_fetch（抓网页转纯文本）、web_search（搜索引擎查资料）——查文档/报错/库用法时用。\n" +
		"- 仓库：只读 git_status / git_diff / git_log / git_show / git_blame；写类 git_add / git_commit / git_branch / git_checkout / git_stash（需审批）。\n" +
		"- 记忆：memory_write 记下项目知识/用户偏好/教训，memory_search / memory_read / memory_list 取用——跨会话持久。\n" +
		"- 规划：复杂任务先用 update_plan 列出步骤清单（每步 pending/in_progress/done），执行中更新状态——清单会展示给用户。\n" +
		"- 提问：关键决策或需求有歧义时用 ask_user 问用户（可给 options 选项），别自己瞎猜；但别滥用、能自查就自查。\n\n" +
		"# 工作方式\n" +
		"按「思考 → 调用工具 → 观察结果 → 再决策」循环推进，直至完成。\n" +
		"复杂或多步任务先用 update_plan 列计划，再逐步执行并更新状态。\n" +
		"先用 search_* 定位、read_file 细读，再动手；改动优先 edit_file/multi_edit（小而准），大改才 write_file。\n" +
		"不确定的库用法/报错/最新信息，用 web_search / web_fetch 查证，别凭记忆臆测。\n" +
		"写类操作（写/改/删/移文件、运行命令）在手动审核模式下需用户批准；若被拒绝，换思路或先解释原因，勿反复重试同一操作。"
}

// ProjectRules 读工作区根的项目约定，拼成系统提示附加段供 agent 遵守：
// 项目文档（AGENTS.md / CLAUDE.md 取首个）+ 用户在设置「指令」tab 写的 .companion/rules.md（两者都注入）。
// 都没有则返回空串。每份内容超长截断。
func ProjectRules(root string) string {
	var b strings.Builder
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} { // 项目文档取首个
		if s := readCapped(root, name); s != "" {
			b.WriteString("\n\n# 项目约定（来自 " + name + "，务必遵守）\n" + s)
			break
		}
	}
	if s := readCapped(root, ".companion/rules.md"); s != "" { // 设置「指令」tab 写的
		b.WriteString("\n\n# 项目指令（务必遵守）\n" + s)
	}
	return b.String()
}

// readCapped 读 root/name 并裁到 8000 字；不存在/空返回 ""。
func readCapped(root, name string) string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	if len(s) > 8000 {
		s = s[:8000] + "\n…（已截断）"
	}
	return s
}
