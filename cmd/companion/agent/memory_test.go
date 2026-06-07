package agent

import (
	"context"
	"strings"
	"testing"
)

// TestMemoryTools 写记忆 → 读回 → 列出 → 搜索命中/不命中。
func TestMemoryTools(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	if _, err := reg.Execute(ctx, "memory_write",
		`{"name":"build-cmd","type":"project","description":"如何构建","content":"用 go build ./cmd/x"}`); err != nil {
		t.Fatalf("memory_write: %v", err)
	}

	got, err := reg.Execute(ctx, "memory_read", `{"name":"build-cmd"}`)
	if err != nil || !strings.Contains(got, "go build ./cmd/x") {
		t.Errorf("memory_read 应含正文：%v\n%s", err, got)
	}
	if list, _ := reg.Execute(ctx, "memory_list", `{}`); !strings.Contains(list, "build-cmd") {
		t.Errorf("memory_list 应含条目：\n%s", list)
	}
	if hit, _ := reg.Execute(ctx, "memory_search", `{"query":"构建"}`); !strings.Contains(hit, "build-cmd") {
		t.Errorf("memory_search 应命中：\n%s", hit)
	}
	if miss, _ := reg.Execute(ctx, "memory_search", `{"query":"无关词xyz"}`); strings.Contains(miss, "build-cmd") {
		t.Errorf("不该命中：\n%s", miss)
	}
	if _, err := reg.Execute(ctx, "memory_read", `{"name":"不存在"}`); err == nil {
		t.Error("读不存在的记忆应报错")
	}
}
