//go:build windows

package main

import (
	"strings"
	"testing"

	"github.com/user/goui/internal/widget"
)

func findItem(items []widget.MenuItem, label string) *widget.MenuItem {
	for i := range items {
		if items[i].Label == label {
			return &items[i]
		}
	}
	return nil
}
func labelsOf(items []widget.MenuItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Label
	}
	return out
}

// 文件树文件菜单：项齐全 + 复制名称/相对/绝对路径动作正确。
func TestFileNodeMenuItems(t *testing.T) {
	theFileTree = &fileTreeState{rootPath: `C:\proj`}
	n := &fileNode{name: "main.go", path: `C:\proj\pkg\main.go`, isDir: false}
	items := fileNodeMenuItems(n)
	for _, want := range []string{"打开", "新建文件", "新建文件夹", "重命名", "删除", "复制名称", "复制相对路径", "复制绝对路径", "在终端打开", "在资源管理器中打开", "刷新"} {
		if findItem(items, want) == nil {
			t.Errorf("文件菜单缺项 %q；实有 %v", want, labelsOf(items))
		}
	}
	var got string
	widget.ClipboardWrite = func(s string) { got = s }
	defer func() { widget.ClipboardWrite = nil }()
	findItem(items, "复制名称").OnClick()
	if got != "main.go" {
		t.Errorf("复制名称=%q", got)
	}
	findItem(items, "复制相对路径").OnClick()
	if got != "pkg/main.go" {
		t.Errorf("复制相对路径=%q", got)
	}
	findItem(items, "复制绝对路径").OnClick()
	if got != `C:\proj\pkg\main.go` {
		t.Errorf("复制绝对路径=%q", got)
	}

	// 目录菜单首项应为 展开/折叠。
	d := &fileNode{name: "pkg", path: `C:\proj\pkg`, isDir: true}
	if l := fileNodeMenuItems(d)[0].Label; l != "展开" && l != "折叠" {
		t.Errorf("目录菜单首项应为 展开/折叠，得 %q", l)
	}
}

// 编辑器标签菜单：项齐全 + 关闭所有清空标签。
func TestEditorTabMenuItems(t *testing.T) {
	theEditor = &editorState{}
	theEditor.tabs = []*editorTab{{path: `C:\a.go`}, {path: `C:\b.go`}}
	items := editorTabItems(0)
	for _, want := range []string{"关闭", "关闭其他", "关闭所有", "复制路径", "复制目录路径", "添加到对话"} {
		if findItem(items, want) == nil {
			t.Errorf("标签菜单缺项 %q；实有 %v", want, labelsOf(items))
		}
	}
	findItem(items, "关闭所有").OnClick()
	if len(theEditor.tabs) != 0 {
		t.Errorf("关闭所有后应无标签，得 %d", len(theEditor.tabs))
	}
}

// 终端菜单：复制全部含全部输出 + 清屏清空。
func TestTerminalMenuItems(t *testing.T) {
	theTerminal = newTerminalState()
	theTerminal.lines = []termRow{{text: "hello"}, {text: "world"}}
	var got string
	widget.ClipboardWrite = func(s string) { got = s }
	defer func() { widget.ClipboardWrite = nil }()
	items := terminalItems()
	for _, want := range []string{"复制全部", "粘贴", "添加到对话", "清屏"} {
		if findItem(items, want) == nil {
			t.Errorf("终端菜单缺项 %q", want)
		}
	}
	findItem(items, "复制全部").OnClick()
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") {
		t.Errorf("复制全部=%q", got)
	}
	findItem(items, "清屏").OnClick()
	if len(theTerminal.lines) != 0 {
		t.Errorf("清屏后应无行，得 %d", len(theTerminal.lines))
	}
}
