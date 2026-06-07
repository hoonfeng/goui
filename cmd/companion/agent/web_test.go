package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWebFetch 离线（httptest）验证：抓取 HTML → 去 script/style/标签、解实体、保留正文文本。
func TestWebFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<html><head><style>.x{color:red}</style></head>`+
			`<body><h1>标题</h1><p>Hello &amp; world</p><script>alert('bad')</script></body></html>`)
	}))
	defer srv.Close()

	out, err := webFetch(context.Background(), map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "标题") || !strings.Contains(out, "Hello & world") {
		t.Errorf("正文缺失：%q", out)
	}
	if strings.Contains(out, "alert('bad')") || strings.Contains(out, "color:red") || strings.Contains(out, "<h1>") {
		t.Errorf("未去除 script/style/标签：%q", out)
	}
}

// TestWebFetchRejectsNonHTTP 非 http(s) URL 应拒绝（挡 file:// 等）。
func TestWebFetchRejectsNonHTTP(t *testing.T) {
	if _, err := webFetch(context.Background(), map[string]any{"url": "file:///etc/passwd"}); err == nil {
		t.Error("应拒绝非 http(s) URL")
	}
}

// TestWebFetchRegistered web_fetch 应在默认工具集中（只读）。
func TestWebFetchRegistered(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	tool, ok := r.Get("web_fetch")
	if !ok {
		t.Fatal("web_fetch 未注册")
	}
	if !tool.ReadOnly {
		t.Error("web_fetch 应为只读（免审）")
	}
}
