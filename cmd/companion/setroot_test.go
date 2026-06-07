//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWorkspaceBuildRoots 单/多文件夹工作区都正确构建各根并加载内容（VS Code 多根模型）。
func TestWorkspaceBuildRoots(t *testing.T) {
	a := t.TempDir()
	if err := os.WriteFile(filepath.Join(a, "x.txt"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(a, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	b := t.TempDir()
	if err := os.WriteFile(filepath.Join(b, "y.txt"), []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}

	prevFT, prevWF := theFileTree, workspaceFolders
	defer func() { theFileTree, workspaceFolders = prevFT, prevWF }()
	theFileTree = &fileTreeState{}

	// 单文件夹工作区
	workspaceFolders = []string{a}
	theFileTree.buildRoots()
	if len(theFileTree.roots) != 1 || theFileTree.roots[0].path != a {
		t.Fatalf("单根失败：%+v", theFileTree.roots)
	}
	if len(theFileTree.roots[0].children) != 2 {
		t.Errorf("根 a 应 2 子，got %d", len(theFileTree.roots[0].children))
	}

	// 多文件夹工作区
	workspaceFolders = []string{a, b}
	theFileTree.buildRoots()
	if len(theFileTree.roots) != 2 {
		t.Fatalf("多根应 2 个，got %d", len(theFileTree.roots))
	}
	if theFileTree.roots[1].path != b || len(theFileTree.roots[1].children) != 1 {
		t.Errorf("根 b 失败：%+v", theFileTree.roots[1])
	}
}

// TestProjectName 工作区显示名：单根=文件夹名；多根=「工作区 (N)」。
func TestProjectName(t *testing.T) {
	prev := workspaceFolders
	defer func() { workspaceFolders = prev }()

	workspaceFolders = []string{`F:\my-proj`}
	if got := projectName(); got != "my-proj" {
		t.Errorf("单根 projectName = %q，want my-proj", got)
	}
	workspaceFolders = []string{`F:\a`, `F:\b`}
	if got := projectName(); got != "工作区 (2 个文件夹)" {
		t.Errorf("多根 projectName = %q", got)
	}
}
