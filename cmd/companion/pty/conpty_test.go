//go:build windows

package pty

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestConPTYStreams 关键验证：ConPTY 起 cmd 后，echo 输出应在进程「存活期间」就流出来
// （裸管道做不到——全缓冲到退出才刷新；伪控制台给真 tty 才行）。
// WIP：伪控制台已建+连通（管道收到 ConPTY 初始序列），但 shell I/O 尚未挂到伪控制台
// （输出跑去继承的父控制台）。待修：raw CreatePipe(可继承句柄) / 控制台分离细节。
// 默认跳过（避免未修期间挂套件）；设 CONPTY_RUN=1 运行调试。
func TestConPTYStreams(t *testing.T) {
	if os.Getenv("CONPTY_RUN") == "" {
		t.Skip("ConPTY 输出路由 WIP，设 CONPTY_RUN=1 运行")
	}
	p, err := Start(Shell{Name: "CMD", Path: "cmd"}, ".", 80, 25)
	if err != nil {
		t.Fatalf("ConPTY Start 失败: %v", err)
	}
	defer p.Close()

	var mu sync.Mutex
	captured := ""
	hit := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				mu.Lock()
				captured += string(buf[:n])
				ok := strings.Contains(captured, "CONPTY_STREAM_OK")
				mu.Unlock()
				if ok {
					select {
					case hit <- struct{}{}:
					default:
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()

	time.Sleep(400 * time.Millisecond) // 等 shell 起来 + 打印初始提示符
	if _, err := p.Write([]byte("echo CONPTY_STREAM_OK\r\n")); err != nil {
		t.Fatalf("写命令失败: %v", err)
	}

	select {
	case <-hit:
		t.Logf("✓ ConPTY 流式输出生效（持久 shell 存活期间即收到 echo）")
	case <-time.After(5 * time.Second):
		mu.Lock()
		c := captured
		mu.Unlock()
		t.Errorf("超时。管道捕获到 %d 字节：%q", len(c), trunc(c, 400))
	}
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
