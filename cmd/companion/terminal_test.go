//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/widget"
)

// TestShellCmd 据 shell 类型构造正确命令 + 携带工作目录；shellLabel 短标签。
func TestShellCmd(t *testing.T) {
	for _, c := range []struct{ shell, want string }{
		{"cmd", "cmd"}, {"powershell", "powershell"}, {"gitbash", "bash"},
	} {
		cmd := shellCmd(c.shell, "echo hi", `C:\`)
		if !strings.Contains(cmd.Args[0], c.want) {
			t.Errorf("shell %s → Args[0]=%q，期望含 %s", c.shell, cmd.Args[0], c.want)
		}
		if !strings.Contains(strings.Join(cmd.Args, " "), "echo hi") {
			t.Errorf("shell %s 命令缺 'echo hi'：%v", c.shell, cmd.Args)
		}
		if cmd.Dir != `C:\` {
			t.Errorf("shell %s Dir=%q", c.shell, cmd.Dir)
		}
	}
	if shellLabel("powershell") != "PS" || shellLabel("gitbash") != "Bash" || shellLabel("cmd") != "CMD" {
		t.Error("shellLabel 标签错")
	}
}

// TestTerminalHistory 上下键命令历史：prev 回溯（顶部停住）、next 前进（底部清空）、空历史不响应。
func TestTerminalHistory(t *testing.T) {
	ts := &terminalState{history: []string{"go build", "go test", "git status"}, histIdx: 3}
	for _, want := range []string{"git status", "go test", "go build", "go build"} { // prev×4（最后到顶停住）
		if c, ok := ts.historyPrev(); !ok || c != want {
			t.Errorf("prev=%q,%v 期望 %q", c, ok, want)
		}
	}
	for _, want := range []string{"go test", "git status", ""} { // next×3（最后到底清空）
		if c, ok := ts.historyNext(); !ok || c != want {
			t.Errorf("next=%q,%v 期望 %q", c, ok, want)
		}
	}
	if _, ok := (&terminalState{}).historyPrev(); ok {
		t.Error("空历史 prev 应返回 false")
	}
}

// run() 是读协程体：cmd /C 执行 + 把 stdout 行写进 pending。直接同步调用验证 exec 闭环
// （chcp 65001 + echo → 捕获到输出 + running 复位），不依赖 GUI。
func TestTerminalRunCapturesOutput(t *testing.T) {
	ts := &terminalState{cwd: "."}
	ts.running = true
	ts.run("echo HELLO_TERM_42")

	if ts.running {
		t.Error("run 结束后 running 应为 false")
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	var sb strings.Builder
	for _, r := range ts.pending {
		sb.WriteString(r.text)
		sb.WriteByte('\n')
	}
	if !strings.Contains(sb.String(), "HELLO_TERM_42") {
		t.Errorf("终端未捕获 echo 输出，pending=%q", sb.String())
	}
}

// changeDir 纯逻辑：cd .. 应回到父目录；cd 不存在目录应报错且 cwd 不变。
func TestTerminalChangeDir(t *testing.T) {
	cwd, _ := os.Getwd()
	ts := &terminalState{cwd: cwd}
	ts.changeDir("..")
	if ts.cwd != filepath.Dir(cwd) {
		t.Errorf("cd .. 后 cwd=%q，期望父目录 %q", ts.cwd, filepath.Dir(cwd))
	}
	before := ts.cwd
	ts.changeDir("绝不存在的目录_xyz")
	if ts.cwd != before {
		t.Errorf("cd 到不存在目录不应改变 cwd：%q → %q", before, ts.cwd)
	}
}

// 端到端（无窗口）：submit 一条命令 → 读协程 exec → 帧泵 drain → SetState → relayout，
// 输出最终被搬进 lines。手动用 animation.Tick + EnsureLayout 模拟主循环逐帧推进。
func TestTerminalStreamsToUI(t *testing.T) {
	animation.ResetScheduler()
	// 复位单例（TerminalPanel.CreateState 返回 theTerminal，挂到下面 mount 的 Element）。
	theTerminal.mu.Lock()
	theTerminal.lines, theTerminal.pending, theTerminal.running = nil, nil, false
	theTerminal.cwd = "."
	theTerminal.mu.Unlock()
	theTerminal.stopPump()

	pipe := render.NewPipeline(420, 320, canvas.NewSoftCanvas(420, 320))
	root := widget.CreateElementFor(&TerminalPanel{})
	root.Mount(nil, 0)
	pipe.SetRootElement(root)
	widget.OnNeedsRepaint = func() { pipe.MarkNeedsRepaint() }
	widget.OnNeedsLayout = func() { pipe.MarkNeedsLayout() }
	defer func() { widget.OnNeedsRepaint = nil; widget.OnNeedsLayout = nil }()
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()

	theTerminal.submit("echo STREAM_OK_7")

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		animation.Tick(time.Now()) // 推进帧泵 → drain（搬 pending→lines + SetState）
		pipe.EnsureLayout()        // 消费 SetState 标的 relayout：Build 重读 lines
		theTerminal.mu.Lock()
		done := !theTerminal.running && len(theTerminal.pending) == 0
		has := false
		for _, r := range theTerminal.lines {
			if strings.Contains(r.text, "STREAM_OK_7") {
				has = true
			}
		}
		theTerminal.mu.Unlock()
		if has && done {
			if err := pipe.Render(); err != nil { // 带输出渲染不报错
				t.Fatalf("终端含输出渲染失败: %v", err)
			}
			if theTerminal.pump != nil {
				t.Error("命令结束且缓冲清空后帧泵应已停（pump=nil）")
			}
			return
		}
		time.Sleep(16 * time.Millisecond)
	}
	t.Error("超时：终端未把命令输出 STREAM_OK_7 搬进 lines")
}
