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
