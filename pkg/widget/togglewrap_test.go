//go:build windows

package widget

import (
	"strings"
	"testing"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// TestGlobalWordWrap 软自动换行是全局开关：ToggleWordWrap 翻转后，任意编辑器（无需聚焦）绘制即跟随。
// 复刻 bug：之前做成 per-editor + 靠 focusedCodeEditor 派发，菜单覆盖层一弹出编辑器就失焦 → 切不动、对勾不亮。
func TestGlobalWordWrap(t *testing.T) {
	globalWordWrap = false
	defer func() { globalWordWrap = false }()
	if WordWrapEnabled() {
		t.Fatal("初始全局换行应为关")
	}
	long := strings.Repeat("word ", 60) // 长行
	ce := NewCodeEditor("go", "package main\n"+long+"\n").WithSize(320, 200)
	e := ce.CreateElement().(*CodeEditorElement) // 注意：不调用 Focus()，验证与焦点无关
	ctx := &layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: 320, MaxWidth: 320, MinHeight: 200, MaxHeight: 200}}
	e.Layout(ctx)
	sk := canvas.NewSkiaCanvas(320, 200)
	defer sk.Release()
	e.Paint(sk, types.Point{})
	if e.wrap {
		t.Fatal("全局关时未聚焦编辑器不应换行")
	}
	before := len(e.wrapSegs)

	ToggleWordWrap() // 全局开（不碰任何编辑器、不需要焦点）
	if !WordWrapEnabled() {
		t.Fatal("ToggleWordWrap 后全局换行应为开")
	}
	e.Layout(ctx)
	e.Paint(sk, types.Point{}) // 编辑器绘制时经 ensureWrapSegs 跟随全局
	if !e.wrap {
		t.Error("全局开后，未聚焦的编辑器绘制也应跟随 e.wrap=true")
	}
	if len(e.wrapSegs) <= before {
		t.Errorf("全局开后长行应折成更多视觉行：before=%d after=%d", before, len(e.wrapSegs))
	}
}

// TestEditorCmdAfterBlur 编辑器专属命令（撤销/复制/全选…仍 per-editor）：菜单覆盖层令编辑器失焦后，
// 经「最近聚焦」兜底仍要命中刚才那个编辑器。
func TestEditorCmdAfterBlur(t *testing.T) {
	focusedCodeEditor, lastFocusedCodeEditor = nil, nil
	defer func() { focusedCodeEditor, lastFocusedCodeEditor = nil, nil }()
	ce := NewCodeEditor("go", "package main\nfunc x() {}\n").WithSize(320, 200)
	e := ce.CreateElement().(*CodeEditorElement)
	e.Focus() // 聚焦
	e.Blur()  // 失焦（模拟菜单覆盖层抢焦点）：focusedCodeEditor=nil，lastFocused 仍是 e
	if !HasFocusedEditor() {
		t.Error("失焦后菜单仍应认为有可操作编辑器（last 兜底）")
	}
	if !RunEditorCommand("selectAll") {
		t.Fatal("失焦后编辑命令应仍命中最近聚焦编辑器")
	}
	if !e.hasSel() {
		t.Error("selectAll 应产生选区（说明命中了最近编辑器）")
	}
}

func BenchmarkRebuildWrapSegs(b *testing.B) {
	globalWordWrap = true
	defer func() { globalWordWrap = false }()
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString(strings.Repeat("word ", 30)) // ~150 字符/行
		sb.WriteByte('\n')
	}
	ce := NewCodeEditor("go", sb.String()).WithSize(600, 400)
	e := ce.CreateElement().(*CodeEditorElement)
	e.wrap = true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.rebuildWrapSegs(500 + float64(i%80)) // 变宽强制重建（模拟拖拽 resize）
	}
}
