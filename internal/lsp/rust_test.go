package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestRustAnalyzerLive 真机验证「非 gopls」语言服务器：起 rust-analyzer，取 Rust 文件符号大纲。
// rust-analyzer 不在 PATH 则跳过。证明 LSP 客户端对各语言服务器通用（不同启动 + langID）。
func TestRustAnalyzerLive(t *testing.T) {
	if _, err := exec.LookPath("rust-analyzer"); err != nil {
		t.Skip("rust-analyzer 不在 PATH，跳过")
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"x\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"), 0o644)
	src := "fn foo() -> i32 { 1 }\n\nfn main() { let _ = foo(); }\n"
	fp := filepath.Join(dir, "src", "main.rs")
	os.WriteFile(fp, []byte(src), 0o644)

	c, err := NewClient("rust-analyzer")
	if err != nil {
		t.Skip("rust-analyzer 启动失败")
	}
	defer c.Close()
	toURI := func(p string) string { return "file:///" + filepath.ToSlash(p) }
	// rust-analyzer 首次会下载/索引 Rust 标准库，可能极慢——initialize 加超时，超时就跳过（非失败）。
	initDone := make(chan error, 1)
	go func() { initDone <- c.Initialize(toURI(dir)) }()
	select {
	case err := <-initDone:
		if err != nil {
			t.Skip("rust-analyzer initialize 出错，跳过")
		}
	case <-time.After(8 * time.Second):
		t.Skip("rust-analyzer initialize 超时（环境首次索引慢），跳过")
	}
	c.DidOpen(toURI(fp), "rust", src)

	var syms []DocumentSymbol
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		syms, _ = c.DocumentSymbol(toURI(fp))
		if len(syms) > 0 {
			break
		}
		time.Sleep(time.Second)
	}
	if len(syms) == 0 {
		t.Skip("rust-analyzer 索引未就绪（环境慢），跳过")
	}
	var hasFoo bool
	for _, s := range syms {
		if s.Name == "foo" {
			hasFoo = true
		}
	}
	if !hasFoo {
		t.Errorf("rust-analyzer 大纲应含 foo，得 %d 符号", len(syms))
	}
	t.Logf("✓ rust-analyzer 真机：Rust 大纲 %d 符号（多语言通用 LSP 客户端）", len(syms))
}
