package render

import (
	"testing"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

func buildAndLayout(el widget.Element, w, h float64) {
	var rec func(e widget.Element)
	rec = func(e widget.Element) {
		for _, c := range e.Build() {
			rec(c)
		}
	}
	rec(el)
	// 宽松约束：让固定尺寸的 Container 保持自身大小，而非被 Tight 撑满
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: w, MaxHeight: h}})
}

// TestHitTestThroughTransformScale 验证命中测试能穿过 Transform：
// 缩放 2 倍后，点击放大区域（超出原始矩形）应命中子元素，区域外则落空。
func TestHitTestThroughTransformScale(t *testing.T) {
	target := &widget.Container{Width: 20, Height: 20}
	tr := &widget.Transform{Scale: 2, Child: target}
	root := widget.CreateElementFor(tr)
	root.Mount(nil, 0)
	root.SetPosition(types.Point{X: 0, Y: 0})
	buildAndLayout(root, 100, 100)

	p := &Pipeline{rootElement: root}

	// 原始矩形 (0,0)-(20,20)、中心 (10,10)；缩放 2 后视觉占 (-10,-10)-(30,30)。
	// 视觉点 (28,28) 在放大区域内、但在原始矩形外 → 仍应命中子元素。
	if hit := p.HitTest(28, 28); hit == nil {
		t.Error("expected hit on scaled child at (28,28)")
	}
	// 视觉点 (40,40) 在放大区域外 → 落空。
	if hit := p.HitTest(40, 40); hit != nil {
		t.Error("expected miss outside scaled region at (40,40)")
	}
}
