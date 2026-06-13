package tools

import (
	"strings"
	"testing"
)

func TestRunCommand(t *testing.T) {
	// 测试简单命令
	got, err := RunCommand("echo hello", "", 30)
	if err != nil {
		t.Fatalf("RunCommand echo failed: %v", err)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("RunCommand echo want 'hello', got %q", got)
	}

	// 测试空命令
	_, err = RunCommand("", "", 30)
	if err == nil {
		t.Error("RunCommand should fail on empty command")
	}

	// 测试无效命令
	got, err = RunCommand("nonexistent_cmd_xyz", "", 10)
	if err != nil {
		// 错误也可能在输出中
		if !strings.Contains(got, "退出") && !strings.Contains(got, "not recognized") {
			t.Logf("RunCommand bad cmd returned: %q, err: %v", got, err)
		}
	}
}

func TestCapOutput(t *testing.T) {
	short := "hello"
	if got := capOutput(short, 100); got != short {
		t.Errorf("capOutput short want %q, got %q", short, got)
	}

	long := strings.Repeat("x", 1000)
	capped := capOutput(long, 100)
	if len(capped) >= len(long) {
		t.Error("capOutput should truncate long strings")
	}
	if !strings.Contains(capped, "...[输出截断") {
		t.Error("capOutput should contain truncation marker")
	}
}
