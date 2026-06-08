//go:build windows

package widget

import (
	"strings"
	"testing"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// TestToggleWrapRebuilds 运行时 toggleWrap 是否真把长行折成多视觉行（对照 WordWrap 配置路径）。
func TestToggleWrapRebuilds(t *testing.T) {
	long := strings.Repeat("word ", 60) // ~300 字符长行
	ce := NewCodeEditor("go", "package main\n"+long+"\n").WithSize(320, 200)
	e := ce.CreateElement().(*CodeEditorElement)
	ctx := &layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: 320, MaxWidth: 320, MinHeight: 200, MaxHeight: 200}}
	e.Layout(ctx)
	sk := canvas.NewSkiaCanvas(320, 200)
	defer sk.Release()
	e.Paint(sk, types.Point{}) // 关闭换行下构建 wrapSegs
	before := len(e.wrapSegs)
	e.runCommand("toggleWrap")
	e.Layout(ctx)
	e.Paint(sk, types.Point{}) // 开启换行后重建 wrapSegs
	after := len(e.wrapSegs)
	if !e.wrap {
		t.Fatal("toggleWrap 后 e.wrap 应为 true")
	}
	if after <= before {
		t.Errorf("toggleWrap 应把长行折成多视觉行：before=%d after=%d", before, after)
	}
}

// TestMenuTogglesAfterBlur 复刻 bug：右键菜单(覆盖层)弹出令编辑器失焦后，菜单里的命令/勾选仍要命中
// 最近聚焦的编辑器（否则 toggleWrap 不生效、对勾永远不亮）。
func TestMenuTogglesAfterBlur(t *testing.T) {
	focusedCodeEditor, lastFocusedCodeEditor = nil, nil // 清全局，避免受其它测试影响
	defer func() { focusedCodeEditor, lastFocusedCodeEditor = nil, nil }()
	ce := NewCodeEditor("go", "package main\n").WithSize(320, 200)
	e := ce.CreateElement().(*CodeEditorElement)
	e.Focus() // 聚焦：focusedCodeEditor=e、lastFocused=e
	e.Blur()  // 失焦（模拟菜单覆盖层抢焦点）：focusedCodeEditor=nil，lastFocused 仍是 e
	if !HasFocusedEditor() {
		t.Error("失焦后菜单仍应认为有可操作编辑器（last 兜底）")
	}
	if EditorWrapEnabled() {
		t.Error("初始换行应为关")
	}
	if !RunEditorCommand("toggleWrap") {
		t.Fatal("失焦后 RunEditorCommand 应仍命中最近聚焦编辑器")
	}
	if !e.wrap {
		t.Error("toggleWrap 应翻转 e.wrap=true")
	}
	if !EditorWrapEnabled() {
		t.Error("toggleWrap 后菜单勾选状态应为 true（经 last 兜底读到）")
	}
}
