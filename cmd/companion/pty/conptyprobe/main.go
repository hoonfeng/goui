//go:build windows

// 无控制台（windowsgui）下验证 ConPTY：模拟 GUI 环境，结果写文件（没有控制台可打印）。
// 用法：go build -ldflags="-H windowsgui" -o conptyprobe.exe ./...  然后运行，看 conpty_probe_result.txt。
package main

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/goui/cmd/companion/pty"
)

func main() {
	f, _ := os.Create("conpty_probe_result.txt")
	defer f.Close()

	p, err := pty.Start(pty.Shell{Name: "CMD", Path: "cmd"}, ".", 80, 25)
	if err != nil {
		f.WriteString("Start err: " + err.Error())
		return
	}
	defer p.Close()

	var mu sync.Mutex
	var buf strings.Builder
	go func() {
		b := make([]byte, 4096)
		for {
			n, e := p.Read(b)
			if n > 0 {
				mu.Lock()
				buf.Write(b[:n])
				mu.Unlock()
			}
			if e != nil {
				return
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	p.Write([]byte("echo PROBE_OK_99\r\n"))
	time.Sleep(1500 * time.Millisecond)

	mu.Lock()
	s := buf.String()
	mu.Unlock()
	f.WriteString("captured " + strconv.Itoa(len(s)) + " bytes\n")
	if strings.Contains(s, "PROBE_OK_99") {
		f.WriteString(">>> STREAMED OK —— ConPTY 在无控制台环境生效（GUI 会工作）\n")
	} else {
		f.WriteString(">>> NO STREAM —— 无控制台也不行，是代码 bug\n")
	}
	f.WriteString("--- raw ---\n")
	f.WriteString(s)
}
