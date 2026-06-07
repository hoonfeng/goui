// 终端面板 —— 中列底部：命令运行器（cmd /C 执行 + stdout/stderr 实时流式回显）。
// 输入命令回车执行；cd / cls 内建；输出强制 UTF-8（chcp 65001）。真 PTY（vim 等交互式
// 程序的 VT 全解析）待办，对 agent「跑构建/测试看输出」的场景，命令运行器已够用。
//
// 线程模型（复刻 CodeEditor LSP 同款，见 AGENTS.md）：exec 读协程把输出写进 pending（加锁），
// 一个 Repeat 动画控制器作「帧泵」——命令运行期间 animation.HasActive() 为真使主循环 60fps
// 出帧，每帧在 UI 线程 drain 把 pending 搬进 lines + 滚到底 + 重绘；命令结束且缓冲清空即停泵，
// 主循环回到 WaitMessage 阻塞省电。animation.Tick 对活跃集合取快照迭代，故 OnUpdate 里 SetState
// （会增减 Input 光标闪烁控制器）是安全的。
//
//go:build windows

package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const maxTermLines = 5000 // 输出行上限，超出丢弃最旧（防长跑命令吃内存）

// 终端等宽字体 + 配色（编辑器深色主题内的终端）。
var (
	termFont = canvas.Font{Family: "Consolas", Size: 13}
	cTermErr = types.ColorFromRGB(240, 105, 98) // stderr 红
)

// theTerminal 终端状态（包级单例，跨 relayout 存活，理由同 theChatState）。
var theTerminal = newTerminalState()

func newTerminalState() *terminalState {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return &terminalState{cwd: cwd, shell: "cmd"}
}

// shellLabel 当前 shell 的短标签。
func shellLabel(shell string) string {
	switch shell {
	case "powershell":
		return "PS"
	case "gitbash":
		return "Bash"
	default:
		return "CMD"
	}
}

// shellCmd 据 shell 类型构造命令（cmd 强制 UTF-8；powershell -NoProfile；gitbash 走 bash -c）。
func shellCmd(shell, line, dir string) *exec.Cmd {
	var c *exec.Cmd
	switch shell {
	case "powershell":
		c = exec.Command("powershell", "-NoProfile", "-Command", line)
	case "gitbash":
		c = exec.Command("bash", "-c", line) // 需 git bash 的 bash 在 PATH
	default:
		c = exec.Command("cmd", "/C", "chcp 65001 >nul & "+line)
	}
	c.Dir = dir
	return c
}

// termRow 一行输出 + 颜色（命令回显蓝 / stdout 常规 / stderr 红 / 提示灰）。
type termRow struct {
	text string
	col  types.Color
}

type terminalState struct {
	widget.BaseState
	mu        sync.Mutex            // 护 lines / pending / running（跨读协程↔UI 线程）
	lines     []termRow             // 已显示输出（含命令回显）
	pending   []termRow             // 读协程写、帧泵 drain 取
	running   bool                  // 有命令在跑
	shell     string                // 当前 shell：cmd / powershell / gitbash
	history   []string              // 命令历史（上下键回溯）
	histIdx   int                   // 历史游标（== len 表示「当前空输入」）
	cwd       string                // 当前工作目录（cd 改、仅 UI 线程）
	draft     string                // 命令输入镜像（防 relayout 丢，仅 UI 线程）
	inputTok  int                   // Input.ResetToken：执行后清空输入框
	scrollTok int                   // ScrollView.ScrollEndToken：新输出滚到底
	pump      *animation.Controller // 帧泵（仅 UI 线程持有/起停）
}

// TerminalPanel 终端面板组件。
type TerminalPanel struct{ widget.StatefulWidget }

func (t *TerminalPanel) CreateState() widget.State { return theTerminal }

func (t *terminalState) Build(ctx widget.BuildContext) widget.Widget {
	if theSettings.TermFontSize > 0 { // 外观/终端设置：字号（单终端，直接调共享 termFont）
		termFont.Size = float64(theSettings.TermFontSize)
	}
	// ── 输出区：等宽彩色行，撑满，新输出滚到底 ──
	t.mu.Lock()
	rows := make([]widget.Widget, 0, len(t.lines)+1)
	if len(t.lines) == 0 {
		rows = append(rows, termLine("终端 — 输入命令回车执行（cd / cls 内建）。当前目录："+t.cwd, cTextDim))
	}
	for _, r := range t.lines {
		rows = append(rows, termLine(r.text, r.col))
	}
	t.mu.Unlock()
	out := widget.NewScrollView(widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(10, 6, 10, 6)},
		rows,
	))
	out.ScrollEndToken = t.scrollTok

	// ── 输入行：提示符（Lucide chevron-right）+ 命令输入（回车执行）──
	in := widget.NewInput("", nil)
	in.Text = t.draft // relayout 回填镜像（仅 ResetToken 变化时真正复位为空）
	in.ResetToken = t.inputTok
	in.OnTextChanged = func(s string) { t.draft = s }
	in.OnSubmit = t.submit
	in.OnArrowUp = t.historyPrev   // ↑ 回溯历史命令
	in.OnArrowDown = t.historyNext // ↓ 前进历史命令
	in.Placeholder = "输入命令，回车执行"
	in.Font = termFont
	in.Color = cText
	in.BGColor = *cEditor // 与行底融为一体（无独立输入框边框）
	in.BorderColor = *cEditor
	in.FocusBorderColor = *cEditor
	in.HoverBorderColor = *cEditor
	in.PlaceholderColor = cTextDim
	in.CursorColor = cText

	shellBadge := &widget.Clickable{ // 点击循环切换 shell（CMD→PS→Bash）
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{BackgroundColor: cStatusBar, BorderRadius: 3, Padding: types.EdgeInsetsLTRB(6, 2, 6, 2)},
			label(shellLabel(t.shell), cTextDim, 10),
		)},
		OnClick: t.cycleShell,
	}
	inputRow := widget.Div(
		widget.Style{Height: 30, BackgroundColor: cEditor, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0),
			FlexDirection: "row", AlignItems: "center", BorderColor: cBorder, BorderWidth: 1},
		shellBadge,
		widget.Div(widget.Style{Width: 8}),
		widget.Lucide("chevron-right", widget.IconSize(14), widget.IconColor(*cStatus)),
		widget.Div(widget.Style{Width: 6}),
		expand(in),
	)

	return &widget.ContextArea{ // 右键：终端菜单（复制全部/粘贴/添加到对话/清屏）
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{BackgroundColor: cEditor, FlexDirection: "column", AlignItems: "stretch"},
			expand(out),
			inputRow,
		)},
		OnContextMenu: func(x, y float64) { terminalMenu(x, y) },
	}
}

// termLine 一行等宽文本（空行用空格占位以保行高）。
func termLine(s string, col types.Color) widget.Widget {
	if s == "" {
		s = " "
	}
	t := widget.NewText(s, col)
	t.Font = termFont
	return t
}

// submit 提交一条命令（Input 回车回调，UI 线程）。
func (t *terminalState) submit(line string) {
	t.inputTok++ // 清空输入框
	t.draft = ""
	line = strings.TrimSpace(line)
	if line == "" {
		t.SetState()
		return
	}
	if n := len(t.history); n == 0 || t.history[n-1] != line { // 入历史（去连续重复）
		t.history = append(t.history, line)
	}
	t.histIdx = len(t.history) // 游标重置到「当前空输入」
	t.appendLine(t.cwd+"> "+line, *cStatus) // 回显命令（含目录上下文，蓝）

	// 内建命令：cls/clear 清屏、cd 改目录（exec 子进程的 cwd 不持久，必须在此处理）。
	switch {
	case line == "cls" || line == "clear":
		t.mu.Lock()
		t.lines = nil
		t.mu.Unlock()
		t.scrollTok++
		t.SetState()
		return
	case line == "cd" || strings.HasPrefix(line, "cd "):
		t.changeDir(strings.TrimSpace(strings.TrimPrefix(line, "cd")))
		t.scrollTok++
		t.SetState()
		return
	}

	t.mu.Lock()
	busy := t.running
	t.mu.Unlock()
	if busy {
		t.appendLine("[上一条命令仍在运行，请稍候]", cTextDim)
		t.scrollTok++
		t.SetState()
		return
	}
	t.mu.Lock()
	t.running = true
	t.mu.Unlock()
	t.scrollTok++
	go t.run(line)
	t.startPump()
	t.SetState()
}

// historyPrev ↑：回溯到更早的历史命令（Input.OnArrowUp 回调）。
func (t *terminalState) historyPrev() (string, bool) {
	if len(t.history) == 0 {
		return "", false
	}
	if t.histIdx > 0 {
		t.histIdx--
	}
	t.draft = t.history[t.histIdx]
	return t.draft, true
}

// historyNext ↓：前进到更新的历史命令；越过最后一条→清空（Input.OnArrowDown 回调）。
func (t *terminalState) historyNext() (string, bool) {
	if len(t.history) == 0 {
		return "", false
	}
	if t.histIdx < len(t.history)-1 {
		t.histIdx++
		t.draft = t.history[t.histIdx]
		return t.draft, true
	}
	t.histIdx = len(t.history)
	t.draft = ""
	return "", true // 到底→清空输入
}

// cycleShell 循环切换 shell：cmd → powershell → gitbash → cmd。
func (t *terminalState) cycleShell() {
	switch t.shell {
	case "cmd":
		t.shell = "powershell"
	case "powershell":
		t.shell = "gitbash"
	default:
		t.shell = "cmd"
	}
	t.SetState()
}

// changeDir 处理 cd：改 t.cwd（仅 UI 线程）。
func (t *terminalState) changeDir(arg string) {
	if arg == "" || arg == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			t.cwd = h
		}
		return
	}
	target := strings.Trim(arg, `"`)
	if !filepath.IsAbs(target) {
		target = filepath.Join(t.cwd, target)
	}
	if fi, err := os.Stat(target); err == nil && fi.IsDir() {
		if abs, err := filepath.Abs(target); err == nil {
			t.cwd = abs
		}
	} else {
		t.appendLine("cd: 系统找不到指定的路径: "+arg, cTermErr)
	}
}

// run 执行命令（读协程）：cmd /C 起子进程，chcp 65001 统一 UTF-8 输出，
// stdout/stderr 各起一读协程把行写进 pending；全部读完 + 进程退出 → running=false。
func (t *terminalState) run(line string) {
	// 据所选 shell 构造命令（cmd 强制 UTF-8 防中文乱码；powershell/gitbash 见 shellCmd）。
	c := shellCmd(t.shell, line, t.cwd)
	stdout, err1 := c.StdoutPipe()
	stderr, err2 := c.StderrPipe()
	if err1 != nil || err2 != nil {
		t.finish("[管道创建失败]")
		return
	}
	if err := c.Start(); err != nil {
		t.finish("[启动失败: " + err.Error() + "]")
		return
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go t.pipe(&wg, stdout, cText)
	go t.pipe(&wg, stderr, cTermErr)
	wg.Wait()
	if err := c.Wait(); err != nil {
		t.push("[进程退出: "+err.Error()+"]", cTextDim)
	}
	t.mu.Lock()
	t.running = false
	t.mu.Unlock()
}

// pipe 把一路输出按行写进 pending（读协程）。
func (t *terminalState) pipe(wg *sync.WaitGroup, r io.Reader, col types.Color) {
	defer wg.Done()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 容长行（默认 64KB 上限不够）
	for sc.Scan() {
		t.push(sc.Text(), col)
	}
}

func (t *terminalState) push(s string, col types.Color) {
	t.mu.Lock()
	t.pending = append(t.pending, termRow{s, col})
	t.mu.Unlock()
}

func (t *terminalState) appendLine(s string, col types.Color) {
	t.mu.Lock()
	t.lines = append(t.lines, termRow{s, col})
	t.mu.Unlock()
}

func (t *terminalState) finish(msg string) {
	t.push(msg, cTermErr)
	t.mu.Lock()
	t.running = false
	t.mu.Unlock()
}

// startPump 启动帧泵（Repeat 控制器，UI 线程）。命令运行期间使主循环持续出帧。
func (t *terminalState) startPump() {
	if t.pump != nil {
		return
	}
	p := animation.NewController(time.Second, animation.Linear)
	p.Repeat = true // 不为插值，只为「保持 HasActive 为真」让主循环持续出帧
	p.OnUpdate = func(float64) { t.drain() }
	t.pump = p
	p.Start()
}

// drain 每帧把 pending 搬进 lines（UI 线程，animation.Tick 调）。命令结束且缓冲清空则停泵。
func (t *terminalState) drain() {
	t.mu.Lock()
	had := len(t.pending) > 0
	if had {
		t.lines = append(t.lines, t.pending...)
		t.pending = t.pending[:0]
		if len(t.lines) > maxTermLines {
			t.lines = append([]termRow(nil), t.lines[len(t.lines)-maxTermLines:]...)
		}
	}
	done := !t.running
	t.mu.Unlock()

	if had {
		t.scrollTok++
		t.SetState()
	}
	if done { // 进程已退出且本帧已把残余 drain 干净 → 停泵，主循环回阻塞
		t.stopPump()
		if !had {
			t.SetState() // 收尾重绘一次，确保最终态呈现
		}
	}
}

func (t *terminalState) stopPump() {
	if t.pump != nil {
		t.pump.Stop()
		t.pump = nil
	}
}

// ─── 右键菜单动作（UI 线程）────────────────────────────────

// openDir 把终端切到某目录（文件树「在终端打开」）。
func (t *terminalState) openDir(dir string) {
	t.cwd = dir
	t.appendLine("[切换目录] "+dir, cTextDim)
	t.scrollTok++
	t.SetState()
}

// copyAll 取全部输出文本（含命令回显，按行）。
func (t *terminalState) copyAll() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var b strings.Builder
	for _, r := range t.lines {
		b.WriteString(r.text)
		b.WriteByte('\n')
	}
	return b.String()
}

// pasteToInput 把剪贴板内容追加到命令输入框。
func (t *terminalState) pasteToInput() {
	if widget.ClipboardRead == nil {
		return
	}
	t.draft += widget.ClipboardRead()
	t.inputTok++ // 受控刷新输入框（Input.Text=t.draft）
	t.SetState()
}

// clearScreen 清屏（等价 cls）。
func (t *terminalState) clearScreen() {
	t.mu.Lock()
	t.lines = nil
	t.mu.Unlock()
	t.scrollTok++
	t.SetState()
}

// terminalArea 终端面板入口（panelBody 调用）。
func terminalArea() widget.Widget { return &TerminalPanel{} }
