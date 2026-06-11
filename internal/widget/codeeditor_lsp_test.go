package widget

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestEditorGoToDefLive CodeEditor 接 gopls 全链路：requestDefinition → 解析 → 回调宿主拿到目标文件/行。
func TestEditorGoToDefLive(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls 不在 PATH，跳过")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n\ngo 1.21\n"), 0o644)
	src := "package x\n\nfunc Foo() int { return 1 }\n\nfunc Bar() int { return Foo() }\n"
	fp := filepath.Join(dir, "a.go")
	os.WriteFile(fp, []byte(src), 0o644)
	toURI := func(p string) string { return "file:///" + filepath.ToSlash(p) }

	ed := NewCodeEditor("go", src).WithLSP("gopls", toURI(dir), toURI(fp))
	var gotFile string
	var gotLine int
	ed.OnGoToDefinition = func(f string, l, c int) { gotFile, gotLine = f, l }
	el := ed.CreateElement().(*CodeEditorElement) // 非嵌入 → 立即 go startLSP

	for i := 0; i < 60 && !el.lspReady; i++ { // 等 gopls 就绪（initialize ~2.5s）
		time.Sleep(250 * time.Millisecond)
	}
	if !el.lspReady {
		t.Skip("gopls 未就绪（环境慢），跳过")
	}
	el.cursor = cePos{4, 25} // Bar 里 Foo() 调用处

	var ok bool
	for i := 0; i < 60; i++ { // 轮询：请求定义 → 等 pendingDef
		el.requestDefinition()
		for j := 0; j < 8; j++ {
			el.lspMu.Lock()
			has := el.pendingDef != nil
			el.lspMu.Unlock()
			if has {
				ok = true
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if ok {
			break
		}
	}
	if !ok {
		t.Fatal("requestDefinition 未产生 pendingDef（gopls 未解析）")
	}
	el.drainLSPCompletion() // UI 线程消费 → 触发 OnGoToDefinition
	if filepath.Base(gotFile) != "a.go" || gotLine != 3 {
		t.Errorf("应跳到 a.go 行 3，得 %s 行 %d", gotFile, gotLine)
	}

	// 查找引用：Foo 声明处 → ≥2（声明 + 调用）。
	var refsGot []CodeLoc
	ed.OnReferences = func(rs []CodeLoc) { refsGot = rs }
	el.cursor = cePos{2, 5}
	for i := 0; i < 40 && len(refsGot) == 0; i++ {
		el.requestReferences()
		for j := 0; j < 8 && pendingNil(el, "refs"); j++ {
			time.Sleep(250 * time.Millisecond)
		}
		el.drainLSPCompletion()
	}
	if len(refsGot) < 2 {
		t.Errorf("查找引用应 ≥2，得 %d", len(refsGot))
	}

	// 文档符号大纲：应含 Foo。
	var symsGot []CodeSym
	ed.OnDocumentSymbols = func(ss []CodeSym) { symsGot = ss }
	for i := 0; i < 40 && len(symsGot) == 0; i++ {
		el.requestDocumentSymbol()
		for j := 0; j < 8 && pendingNil(el, "syms"); j++ {
			time.Sleep(250 * time.Millisecond)
		}
		el.drainLSPCompletion()
	}
	var foundFoo bool
	for _, s := range symsGot {
		if s.Name == "Foo" {
			foundFoo = true
		}
	}
	if !foundFoo {
		t.Errorf("大纲应含 Foo，得 %d 符号", len(symsGot))
	}

	// 悬停：Foo 声明处 → hoverText 有内容（编辑器内画浮层）。
	el.cursor = cePos{2, 5}
	for i := 0; i < 40 && el.hoverText == ""; i++ {
		el.requestHover()
		for j := 0; j < 8 && pendingNil(el, "hover"); j++ {
			time.Sleep(250 * time.Millisecond)
		}
		el.drainLSPCompletion()
	}
	if el.hoverText == "" {
		t.Error("悬停应有内容")
	}
	t.Logf("✓ 查找引用 %d 处、大纲 %d 符号、悬停 %d 字", len(refsGot), len(symsGot), len([]rune(el.hoverText)))
	el.Unmount()
}

func pendingNil(el *CodeEditorElement, which string) bool {
	el.lspMu.Lock()
	defer el.lspMu.Unlock()
	switch which {
	case "refs":
		return el.pendingRefs == nil
	case "hover":
		return el.pendingHover == nil
	}
	return el.pendingSyms == nil
}
