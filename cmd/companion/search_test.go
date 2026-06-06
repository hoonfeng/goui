//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSearchRun 验证跨文件搜索：大小写不敏感(默认)→敏感的命中数变化。
func TestSearchRun(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "package main\nfunc Foo() {}\n// foo bar\n")
	mustWrite(t, filepath.Join(dir, "b.txt"), "hello FOO world\n")

	pf, ps := theFileTree, theSearch
	defer func() { theFileTree, theSearch = pf, ps }()
	theFileTree = &fileTreeState{rootPath: dir}
	theSearch = &searchState{collapsed: map[string]bool{}, query: "foo"}

	theSearch.run() // 不区分大小写：Foo + foo + FOO = 3 命中，2 文件
	if theSearch.totalMatches != 3 {
		t.Errorf("insensitive matches=%d, want 3", theSearch.totalMatches)
	}
	if len(theSearch.files) != 2 {
		t.Errorf("files=%d, want 2", len(theSearch.files))
	}

	theSearch.caseSensitive = true
	theSearch.run() // 区分大小写：只有 a.go 的 "foo" = 1 命中
	if theSearch.totalMatches != 1 {
		t.Errorf("sensitive matches=%d, want 1", theSearch.totalMatches)
	}
}

// TestSearchReplace 验证全部替换：foo(不敏感)→bar，整文件字面替换。
func TestSearchReplace(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "foo and Foo and FOO\n")

	pf, ps, pe := theFileTree, theSearch, theEditor
	defer func() { theFileTree, theSearch, theEditor = pf, ps, pe }()
	theFileTree = &fileTreeState{rootPath: dir}
	theEditor = &editorState{}
	theSearch = &searchState{collapsed: map[string]bool{}, query: "foo", replaceText: "bar"}

	theSearch.run()
	if theSearch.totalMatches != 3 {
		t.Fatalf("matches=%d, want 3", theSearch.totalMatches)
	}
	if n := theSearch.doReplace(); n != 1 {
		t.Errorf("replaced files=%d, want 1", n)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "a.go"))
	if string(data) != "bar and bar and bar\n" {
		t.Errorf("content=%q, want all bar", string(data))
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
