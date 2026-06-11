package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// 复现「拖动 ResizeHandle → OnDrag 改尺寸 + SetState → relayout → 继续拖」全链路，
// 断言：Element 跨 relayout 复用（capture 不脱钩）、dragging 不丢、尺寸持续变化。

var rhTestW float64

type rhTestWidget struct{ StatefulWidget }

func (w *rhTestWidget) CreateState() State { return &rhTestState{} }

type rhTestState struct{ BaseState }

func (s *rhTestState) Build(ctx BuildContext) Widget {
	return HBox(
		Div(Style{Width: rhTestW, Height: 200}),
		VResize(types.ColorFromRGB(0, 0, 0), types.ColorFromRGB(0, 0, 0), func(d float64) {
			rhTestW += d
			s.SetState()
		}),
		Div(Style{Width: 50, Height: 200}),
	)
}

func rhLayoutTree(root Element, w, h float64) {
	var build func(Element)
	build = func(el Element) {
		for _, c := range el.Build() {
			build(c)
		}
	}
	build(root)
	root.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: w, MaxWidth: w, MinHeight: h, MaxHeight: h}})
}

func rhFind(el Element) *resizeHandleElement {
	if r, ok := el.(*resizeHandleElement); ok {
		return r
	}
	for _, c := range el.Children() {
		if r := rhFind(c); r != nil {
			return r
		}
	}
	return nil
}

func TestResizeHandleDragSurvivesRelayout(t *testing.T) {
	rhTestW = 100
	OnNeedsLayout = func() {}
	OnNeedsRepaint = func() {}
	var captured Element
	RequestPointerCapture = func(el Element) { captured = el }
	defer func() { OnNeedsLayout = nil; OnNeedsRepaint = nil; RequestPointerCapture = nil }()

	root := CreateElementFor(&rhTestWidget{})
	rhLayoutTree(root, 400, 200)

	h1 := rhFind(root)
	if h1 == nil {
		t.Fatal("ResizeHandle Element 没找到")
	}
	t.Logf("handle offset=%v size=%v", h1.Offset(), h1.size)
	if h1.size.Width <= 0 || h1.size.Height <= 0 {
		t.Fatalf("handle 尺寸非法 %v（HitTest 会命中不到）", h1.size)
	}

	// MouseDown 命中 handle 中线
	downX := h1.Offset().X + 3
	h1.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, downX, 100, 0, 0))
	if captured != Element(h1) {
		t.Fatalf("capture 未指向 handle: captured=%p h1=%p", captured, h1)
	}
	if !h1.dragging {
		t.Fatal("MouseDown 后 dragging 未置位")
	}

	// MouseMove +10（绝对坐标）
	h1.HandleEvent(event.NewMouseEvent(event.TypeMouseMove, downX+10, 100, 0, 0))
	if rhTestW != 110 {
		t.Errorf("move1 后 rhTestW=%v want 110", rhTestW)
	}

	// 模拟下一帧 relayout
	rhLayoutTree(root, 400, 200)

	h2 := rhFind(root)
	if h2 != h1 {
		t.Errorf("ResizeHandle 跨 relayout 未复用: h1=%p h2=%p", h1, h2)
	}
	if !h1.dragging {
		t.Error("relayout 后 dragging 丢失")
	}

	// 继续拖（app 会把 move 发给 captured）
	before := rhTestW
	rh := captured.(*resizeHandleElement)
	rh.HandleEvent(event.NewMouseEvent(event.TypeMouseMove, h1.Offset().X+9, 100, 0, 0))
	if rhTestW == before {
		t.Errorf("relayout 后拖动卡死: rhTestW 停在 %v", rhTestW)
	}
	t.Logf("最终 rhTestW=%v（从 100 起，三次 move 应持续增长）", rhTestW)
}
