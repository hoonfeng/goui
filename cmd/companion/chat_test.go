//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAttachmentContext 附件内容拼成 agent 上下文；文件名用于显示；空附件返回空。
func TestAttachmentContext(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(p, []byte("hello attach"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := attachmentContext([]string{p})
	if !strings.Contains(ctx, "hello attach") || !strings.Contains(ctx, "用户附件") {
		t.Errorf("context = %q", ctx)
	}
	if got := attachmentNames([]string{p}); got != "note.txt" {
		t.Errorf("names = %q，期望 note.txt", got)
	}
	if attachmentContext(nil) != "" {
		t.Error("空附件应返回空串")
	}
}
