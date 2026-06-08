// 终端面板 —— 真终端：ConPTY 伪终端(pty) + VT/ANSI 屏幕模型(vterm) + 可聚焦渲染/输入(TerminalView)。
// 多标签：每标签一个持久 PTY 会话喂 vterm；TerminalView 抓原始按键转 VT 写 PTY、自绘 vterm 网格。
// 线程模型：读协程读 PTY 原始字节进 pending(加锁)；帧泵每帧在 UI 线程把 pending 喂进 vterm(单线程更新网格)
// + 重绘；久无输出停泵省电。键盘/resize/cd 都在 UI 线程。
//
//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/user/goui/cmd/companion/pty"
	"github.com/user/goui/cmd/companion/vterm"
	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const termIdleFrames = 180 // 连续 ~3s 无输出 → 停泵省电；窗口放宽以接住 ping/慢构建等稀疏输出。
// 注：停泵后纯后台输出（无按键）会延到下次交互才刷——根治需读协程跨线程唤醒 UI 循环(PostMessage)，列为后续。

// 多实例终端：theTermMgr 管多个标签，theTerminal 始终指向「当前活动标签」
// （外部 openDir/copyAll/clearScreen/shell 等照旧用 theTerminal，自动作用于活动标签）。
var (
	theTermMgr  *termManager
	theTerminal *terminalState
)

func init() { theTermMgr = newTermManager() }

func newTerminalState() *terminalState {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return &terminalState{cwd: cwd, shell: "cmd", vt: vterm.New(80, 24), cols: 80, rows: 24}
}

// ─── 多标签管理器 ───────────────────────────────────────────────
type termManager struct {
	widget.BaseState
	tabs   []*terminalState
	active int
}

func newTermManager() *termManager {
	m := &termManager{tabs: []*terminalState{newTerminalState()}}
	theTerminal = m.tabs[0]
	return m
}

// newTab 新建标签（继承当前标签的 shell/cwd），切到它。
func (m *termManager) newTab() {
	t := newTerminalState()
	if cur := m.tabs[m.active]; cur != nil {
		t.shell, t.cwd = cur.shell, cur.cwd
	}
	m.tabs = append(m.tabs, t)
	m.active = len(m.tabs) - 1
	theTerminal = t
	m.SetState()
}

// switchTab 切到第 i 个标签。
func (m *termManager) switchTab(i int) {
	if i < 0 || i >= len(m.tabs) || i == m.active {
		return
	}
	m.active = i
	theTerminal = m.tabs[i]
	m.SetState()
}

// closeTab 关闭第 i 个标签（至少留一个），杀掉它的 PTY。
func (m *termManager) closeTab(i int) {
	if i < 0 || i >= len(m.tabs) || len(m.tabs) == 1 {
		return
	}
	m.tabs[i].killPTY()
	m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
	if m.active >= len(m.tabs) {
		m.active = len(m.tabs) - 1
	} else if m.active > i {
		m.active--
	}
	theTerminal = m.tabs[m.active]
	m.SetState()
}

func (m *termManager) Build(ctx widget.BuildContext) widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch", BackgroundColor: cEditor},
		m.tabBar(),
		expand(&termInstance{st: m.tabs[m.active]}),
	)
}

// tabBar 顶部标签栏：每标签 shell 徽标 + 目录名 + 关闭×；末尾「+」新建。
func (m *termManager) tabBar() widget.Widget {
	kids := make([]widget.Widget, 0, len(m.tabs)+1)
	for i, t := range m.tabs {
		idx := i
		bg, tc := cStatusBar, cTextDim
		if i == m.active {
			bg, tc = cEditor, cText
		}
		kids = append(kids, &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
				widget.Style{BackgroundColor: bg, Padding: types.EdgeInsetsLTRB(10, 5, 10, 5)},
				label(shellLabel(t.shell)+" "+filepath.Base(t.cwd), tc, 11),
			)},
			OnClick: func() { m.switchTab(idx) },
		})
		if len(m.tabs) > 1 {
			kids = append(kids, &widget.Clickable{
				SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
					widget.Style{BackgroundColor: bg, Padding: types.EdgeInsetsLTRB(0, 5, 8, 5)},
					widget.Lucide("x", widget.IconSize(11), widget.IconColor(cTextDim)),
				)},
				OnClick: func() { m.closeTab(idx) },
			})
		}
	}
	kids = append(kids, &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(8, 5, 8, 5)},
			widget.Lucide("plus", widget.IconSize(13), widget.IconColor(*cStatus)),
		)},
		OnClick: func() { m.newTab() },
	})
	return widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "center", BackgroundColor: cStatusBar, Height: 28},
		kids,
	)
}

// termInstance 把某个终端标签作为子 StatefulWidget 挂载（活动标签才挂载渲染；
// 切换=挂另一标签；后台标签的 vt/读协程仍在其 *terminalState 里活着，切回即见）。
type termInstance struct {
	widget.StatefulWidget
	st *terminalState
}

func (w *termInstance) CreateState() widget.State { return w.st }

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

// ptyShellFor 把内部 shell 码（cmd/powershell/gitbash）映射到探测到的 pty.Shell。
func ptyShellFor(code string) pty.Shell {
	name := map[string]string{"cmd": "CMD", "powershell": "PowerShell", "gitbash": "Git Bash"}[code]
	if name == "" {
		return pty.DefaultShell()
	}
	return pty.ShellByName(name)
}

// ─── 单个终端标签（PTY + vterm）───────────────────────────────────
type terminalState struct {
	widget.BaseState
	mu         sync.Mutex            // 护 pending / alive / sess / idleFrames / pump
	vt         *vterm.Terminal       // 屏幕模型（仅 UI 线程读写）
	sess       pty.PTY               // 伪终端会话
	pending    []byte                // 读协程写、帧泵取（原始 PTY 字节）
	alive      bool                  // PTY 在跑
	failed     bool                  // 启动失败（不再反复重试）
	shell      string                // cmd / powershell / gitbash
	cwd        string                // 初始/cd 目录
	cols, rows int                   // 当前终端列/行
	idleFrames int                   // 连续无输出帧数
	pump       *animation.Controller // 帧泵
}

// TerminalPanel 终端面板组件。
type TerminalPanel struct{ widget.StatefulWidget }

func (t *TerminalPanel) CreateState() widget.State { return theTermMgr }

func (t *terminalState) Build(ctx widget.BuildContext) widget.Widget {
	tv := &widget.TerminalView{
		OnPaint: func(cvs canvas.Canvas, x, y, w, h float64) {
			font := termGridFontNow()
			cw, ch := termCellSize(cvs, font)
			cols, rows := 1, 1
			if cw > 0 {
				cols = int(w / cw)
			}
			if ch > 0 {
				rows = int(h / ch)
			}
			if cols < 1 {
				cols = 1
			}
			if rows < 1 {
				rows = 1
			}
			t.ensurePTY(cols, rows)
			t.resizeTo(cols, rows)
			paintVTGrid(cvs, x, y, w, h, t.vt, font)
		},
		OnKey: t.handleKey,
	}
	return &widget.ContextArea{ // 右键：终端菜单（复制全部/粘贴/添加到对话/清屏/切 shell）
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{BackgroundColor: cEditor, FlexDirection: "column", AlignItems: "stretch"},
			expand(tv),
		)},
		OnContextMenu: func(x, y float64) { terminalMenu(x, y) },
	}
}

// ensurePTY 懒启动伪终端（首帧拿到真实尺寸时）。
func (t *terminalState) ensurePTY(cols, rows int) {
	t.mu.Lock()
	if t.alive || t.failed {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()
	sess, err := pty.Start(ptyShellFor(t.shell), t.cwd, cols, rows)
	if err != nil {
		t.mu.Lock()
		t.failed = true
		t.mu.Unlock()
		t.vt = vterm.New(cols, rows)
		t.vt.Write([]byte("[终端启动失败: " + err.Error() + "]\r\n"))
		widget.OnNeedsRepaint()
		return
	}
	t.vt = vterm.New(cols, rows)
	t.mu.Lock()
	t.sess, t.alive, t.cols, t.rows = sess, true, cols, rows
	t.mu.Unlock()
	go t.reader(sess)
	t.startPump()
}

// reader 持续读 PTY 原始字节进 pending（读协程，会话存活期间常驻）。
func (t *terminalState) reader(sess pty.PTY) {
	buf := make([]byte, 4096)
	for {
		n, err := sess.Read(buf)
		if n > 0 {
			t.mu.Lock()
			t.pending = append(t.pending, buf[:n]...)
			t.mu.Unlock()
		}
		if err != nil {
			t.mu.Lock()
			t.alive = false
			t.mu.Unlock()
			return
		}
	}
}

// drain 每帧把 pending 喂进 vterm（UI 线程）+ 重绘；久无输出停泵。
func (t *terminalState) drain() {
	t.mu.Lock()
	var data []byte
	if len(t.pending) > 0 {
		data = t.pending
		t.pending = nil
		t.idleFrames = 0
	} else {
		t.idleFrames++
	}
	idle := t.idleFrames > termIdleFrames
	t.mu.Unlock()

	if len(data) > 0 {
		t.vt.Write(data) // UI 线程更新网格
		if widget.OnNeedsRepaint != nil {
			widget.OnNeedsRepaint() // 仅重绘（网格尺寸未变，无需 relayout）
		}
	}
	if idle {
		t.stopPump()
	}
}

// handleKey 按键 → VT 字节 → 写 PTY（起泵接住 shell 响应）。
func (t *terminalState) handleKey(ev *event.KeyEvent) {
	data := keyToVT(ev)
	if len(data) == 0 {
		return
	}
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write(data)
		t.startPump()
	}
}

// resizeTo 面板尺寸变 → 同步 vterm + PTY（伪控制台据此重排）。
func (t *terminalState) resizeTo(cols, rows int) {
	t.mu.Lock()
	if !t.alive || (cols == t.cols && rows == t.rows) {
		t.mu.Unlock()
		return
	}
	t.cols, t.rows = cols, rows
	sess := t.sess
	t.mu.Unlock()
	t.vt.Resize(cols, rows)
	if sess != nil {
		sess.Resize(cols, rows)
	}
}

func (t *terminalState) startPump() {
	t.mu.Lock()
	t.idleFrames = 0
	if t.pump != nil {
		t.mu.Unlock()
		return
	}
	p := animation.NewController(time.Second, animation.Linear)
	p.Repeat = true
	p.OnUpdate = func(float64) { t.drain() }
	t.pump = p
	t.mu.Unlock()
	p.Start()
}

func (t *terminalState) stopPump() {
	t.mu.Lock()
	p := t.pump
	t.pump = nil
	t.mu.Unlock()
	if p != nil {
		p.Stop()
	}
}

// killPTY 杀掉本标签的 PTY 会话（关闭标签 / 切 shell 时）。
func (t *terminalState) killPTY() {
	t.mu.Lock()
	sess := t.sess
	t.sess, t.alive, t.failed = nil, false, false
	t.mu.Unlock()
	if sess != nil {
		sess.Close()
	}
	t.stopPump()
}

// cycleShell 循环切 shell：cmd → powershell → gitbash → cmd。杀旧 PTY，下帧起新 shell。
func (t *terminalState) cycleShell() {
	switch t.shell {
	case "cmd":
		t.shell = "powershell"
	case "powershell":
		t.shell = "gitbash"
	default:
		t.shell = "cmd"
	}
	t.killPTY()
	t.SetState()
}

// ─── 右键菜单动作（UI 线程）────────────────────────────────

// openDir 把终端切到某目录（文件树「在终端打开」）：cd 当前 shell；未起则记为初始目录。
func (t *terminalState) openDir(dir string) {
	t.cwd = dir
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write([]byte("cd /d \"" + dir + "\"\r")) // cmd 用 cd /d；其它 shell 多余的 /d 会忽略或报小错
		t.startPump()
	}
	t.SetState()
}

// copyAll 取屏幕全部文本（按行，去尾空白）。
func (t *terminalState) copyAll() string {
	cols, rows := t.vt.Size()
	var b strings.Builder
	for r := 0; r < rows; r++ {
		line := make([]rune, 0, cols)
		for c := 0; c < cols; c++ {
			line = append(line, t.vt.Cell(r, c).Ch)
		}
		b.WriteString(strings.TrimRight(string(line), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// pasteToInput 粘贴：把剪贴板内容写进 PTY（真终端=直接送给 shell）。
func (t *terminalState) pasteToInput() {
	if widget.ClipboardRead == nil {
		return
	}
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	if sess != nil {
		sess.Write([]byte(widget.ClipboardRead()))
		t.startPump()
	}
}

// clearScreen 清屏：发 shell 的清屏命令（cmd→cls / 其它→clear）。
func (t *terminalState) clearScreen() {
	t.mu.Lock()
	sess := t.sess
	t.mu.Unlock()
	cmd := "clear\r"
	if t.shell == "cmd" {
		cmd = "cls\r"
	}
	if sess != nil {
		sess.Write([]byte(cmd))
		t.startPump()
	} else {
		t.vt = vterm.New(t.cols, t.rows)
		if widget.OnNeedsRepaint != nil {
			widget.OnNeedsRepaint()
		}
	}
}

// terminalArea 终端面板入口（panelBody 调用）。
func terminalArea() widget.Widget { return &TerminalPanel{} }
