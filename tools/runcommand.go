package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RunCommand 在工作目录执行一条 shell 命令并返回输出（同步、120s 超时、UTF-8）。
// command 为要执行的命令，cwd 可选工作目录，timeoutSec 可选超时秒数（默认 120）。
func RunCommand(command, cwd string, timeoutSec int) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command 不能为空")
	}

	if timeoutSec <= 0 {
		timeoutSec = 120
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Windows 下统一 UTF-8 输出
	c := exec.CommandContext(ctx, "cmd", "/C", "chcp 65001 >nul & "+command)
	if cwd != "" {
		c.Dir = cwd
	}

	out, err := c.CombinedOutput()
	res := capOutput(string(out), 16000)
	if ctx.Err() == context.DeadlineExceeded {
		res += fmt.Sprintf("\n[超时 %ds 已终止]", timeoutSec)
	} else if err != nil {
		res += "\n[退出: " + err.Error() + "]"
	}
	return res, nil
}

// capOutput 截断过长输出（保头 3/4 + 尾 1/4），防工具结果撑爆上下文。
func capOutput(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	head := limit * 3 / 4
	tail := limit - head
	return s[:head] + "\n...[输出截断 " + fmt.Sprint(len(s)-limit) + " 字节]...\n" + s[len(s)-tail:]
}
