package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestDefinitionLive 真机：起 gopls，解析跨函数调用 Foo() 的定义位置。gopls 不在 PATH 则跳过。
func TestDefinitionLive(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls 不在 PATH，跳过")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n\ngo 1.21\n"), 0o644)
	src := "package x\n\nfunc Foo() int { return 1 }\n\nfunc Bar() int { return Foo() }\n"
	fp := filepath.Join(dir, "a.go")
	os.WriteFile(fp, []byte(src), 0o644)

	c, err := NewClient("gopls")
	if err != nil {
		t.Skip("gopls 启动失败")
	}
	defer c.Close()
	toURI := func(p string) string { return "file:///" + filepath.ToSlash(p) }
	if err := c.Initialize(toURI(dir)); err != nil {
		t.Fatal(err)
	}
	c.DidOpen(toURI(fp), "go", src)

	// gopls 首次索引需时间：轮询 Definition 直到拿到或超时。
	var locs []Location
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		locs, err = c.Definition(toURI(fp), 4, 25) // 行 4(0 基) "func Bar() ... Foo()"，Foo 在列 24-26
		if err == nil && len(locs) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if len(locs) == 0 {
		t.Fatalf("应解析到 Foo 定义，err=%v", err)
	}
	if locs[0].Range.Start.Line != 2 { // func Foo 在行 2(0 基)
		t.Errorf("Foo 定义应在行 2(0 基)，得 %d", locs[0].Range.Start.Line)
	}
	t.Logf("✓ 转到定义真机通过：Foo() 调用 → 定义在 %s 行 %d", uriBase(locs[0].URI), locs[0].Range.Start.Line+1)

	// 查找引用：Foo 声明处（行 2 char 5）→ 应含声明 + Bar 里的调用（≥2）。
	refs, _ := c.References(toURI(fp), 2, 5, true)
	if len(refs) < 2 {
		t.Errorf("Foo 引用应 ≥2（声明+调用），得 %d", len(refs))
	}

	// 文档符号大纲：应含 Foo、Bar。
	syms, _ := c.DocumentSymbol(toURI(fp))
	var hasFoo, hasBar bool
	for _, s := range syms {
		if s.Name == "Foo" {
			hasFoo = true
		}
		if s.Name == "Bar" {
			hasBar = true
		}
	}
	if !hasFoo || !hasBar {
		t.Errorf("大纲应含 Foo+Bar，得 %d 个符号", len(syms))
	}

	// 悬停：Foo 定义处应有签名/文档。
	if h, _ := c.HoverAt(toURI(fp), 2, 5); h == nil || h.Contents.Value == "" {
		t.Error("Foo 悬停应有内容")
	}
	t.Logf("✓ 查找引用 %d 处、大纲 %d 符号、悬停有内容", len(refs), len(syms))
}

func uriBase(uri string) string { return filepath.Base(uri) }
