package app

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// Clickable 整行可点：点击落在内层（图标/文字 Div）上也要触发 OnClick（靠捕获路径冒泡，
// 与 Button 同源）。文件树行用它。
func TestClickableClickThroughChild(t *testing.T) {
	clicked := 0
	cl := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsets(8)},
			widget.NewText("行内容", types.ColorWhite),
		)},
		OnClick: func() { clicked++ },
	}
	app := NewApplication()
	app.Pipeline = render.NewPipeline(300, 100, canvas.NewSoftCanvas(300, 100))
	app.RootElement = widget.CreateElementFor(cl)
	app.RootElement.Mount(nil, 0)
	app.Pipeline.SetRootElement(app.RootElement)

	widget.OnNeedsRepaint = func() { app.Pipeline.MarkNeedsRepaint() }
	widget.OnNeedsLayout = func() { app.Pipeline.MarkNeedsLayout() }
	defer func() {
		widget.OnNeedsRepaint = nil
		widget.OnNeedsLayout = nil
	}()
	app.Pipeline.MarkNeedsLayout()
	app.Pipeline.EnsureLayout()

	// Clickable 在 (0,0)；点 (30,16) 落在内层文字/Div 上（非 Clickable 本身）。
	cx, cy := 30.0, 16.0
	hit := app.Pipeline.HitTest(cx, cy)
	if hit == nil {
		t.Fatal("HitTest 未命中（布局异常）")
	}
	if _, isCl := hit.(*widget.ClickableElement); isCl {
		t.Fatalf("命中 Clickable 本身（%T），无法复现「点子元素」", hit)
	}

	feed := func(typ event.Type) {
		app.Pipeline.EnsureLayout()
		app.routeMouseEvent(event.NewMouseEvent(typ, cx, cy, 0, 0))
	}
	feed(event.TypeMouseDown)
	feed(event.TypeMouseUp)

	if clicked != 1 {
		t.Errorf("OnClick 触发 %d 次，期望 1——点击未冒泡到 Clickable", clicked)
	}
}

// 嵌套 Clickable（外=标签切换，内=关闭×带 StopPropagation）：点内层只触发内层、不触发外层。
func TestClickableStopPropagation(t *testing.T) {
	outer, inner := 0, 0
	closeBtn := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(widget.Style{Width: 20, Height: 20})},
		OnClick:           func() { inner++ },
		StopPropagation:   true,
	}
	tab := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsets(4)},
			widget.NewText("a.go", types.ColorWhite),
			closeBtn,
		)},
		OnClick: func() { outer++ },
	}
	app := NewApplication()
	app.Pipeline = render.NewPipeline(200, 60, canvas.NewSoftCanvas(200, 60))
	app.RootElement = widget.CreateElementFor(tab)
	app.RootElement.Mount(nil, 0)
	app.Pipeline.SetRootElement(app.RootElement)
	widget.OnNeedsRepaint = func() { app.Pipeline.MarkNeedsRepaint() }
	widget.OnNeedsLayout = func() { app.Pipeline.MarkNeedsLayout() }
	defer func() { widget.OnNeedsRepaint = nil; widget.OnNeedsLayout = nil }()
	app.Pipeline.MarkNeedsLayout()
	app.Pipeline.EnsureLayout()

	// closeBtn 在「a.go」文字之后（padding4 + 文字 ~28 → ×区约 x32..52）。点 (42,12) 落在内层关闭区。
	bx, by := 42.0, 12.0
	if hit := app.Pipeline.HitTest(bx, by); hit == nil {
		t.Fatal("未命中关闭区")
	}
	feed := func(typ event.Type) {
		app.Pipeline.EnsureLayout()
		app.routeMouseEvent(event.NewMouseEvent(typ, bx, by, 0, 0))
	}
	feed(event.TypeMouseDown)
	feed(event.TypeMouseUp)
	if inner != 1 || outer != 0 {
		t.Errorf("点关闭× 应只触发内层：inner=%d(期望1) outer=%d(期望0)", inner, outer)
	}
}
