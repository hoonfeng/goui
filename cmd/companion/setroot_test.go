//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSetRootSwitchesWorkspace 验证「打开文件夹」→ setRoot 切换根并加载新目录内容。
func TestSetRootSwitchesWorkspace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	prev := theFileTree
	defer func() { theFileTree = prev }()
	theFileTree = &fileTreeState{}
	theFileTree.setRoot(dir)

	if theFileTree.rootPath != dir {
		t.Fatalf("rootPath = %q, want %q", theFileTree.rootPath, dir)
	}
	if theFileTree.root == nil || len(theFileTree.root.children) != 2 {
		t.Fatalf("want 2 children after setRoot, got %+v", theFileTree.root)
	}
}
