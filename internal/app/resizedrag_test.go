package app

import (
	"testing"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// 完整复现实机拖动：真 routeMouseEvent + OverlayHost + 模拟主循环 +
// **companion 完全相同的布局结构**（VBox + expand(body) + flexRow + 嵌套 flexCol + Expanded），
// 排查复杂嵌套是否破坏 Element 复用 / capture。断言拖动持续改变尺寸。

var appDragW float64

type appDragShell struct{ widget.StatefulWidget }

func (s *appDragShell) CreateState() widget.State { return &appDragState{} }

type appDragState struct{ widget.BaseState }

func expandW(w widget.Widget) widget.Widget {
	return &widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: w}, Flex: 1}
}

func flexRowW(children ...widget.Widget) widget.Widget {
	args := []interface{}{widget.Style{FlexDirection: "row", AlignItems: "stretch"}}
	for _, c := range children {
		args = append(args, c)
	}
	return widget.Div(args...)
}

func flexColW(children ...widget.Widget) widget.Widget {
	args := []interface{}{widget.Style{FlexDirection: "column", AlignItems: "stretch"}}
	for _, c := range children {
		args = append(args, c)
	}
	return widget.Div(args...)
}

func (s *appDragState) Build(ctx widget.BuildContext) widget.Widget {
	return widget.VBox(
		widget.Div(widget.Style{Height: 36, BackgroundColor: types.ColorRef(60, 60, 61)}), // titleBar
		expandW(s.body()),
		widget.Div(widget.Style{Height: 26, BackgroundColor: types.ColorRef(0, 122, 204)}), // status
	)
}

func (s *appDragState) body() widget.Widget {
	return flexRowW(
		widget.Div(widget.Style{Width: appDragW, BackgroundColor: types.ColorRef(37, 37, 38)}),
		widget.VResize(types.ColorFromRGB(45, 45, 45), types.ColorFromRGB(0, 122, 204), func(d float64) {
			appDragW += d
			s.SetState()
		}),
		expandW(s.mid()),
	)
}

func (s *appDragState) mid() widget.Widget {
	return flexColW(
		expandW(widget.Div(widget.Style{BackgroundColor: types.ColorRef(30, 30, 30)})),
		widget.Div(widget.Style{Height: 100, BackgroundColor: types.ColorRef(37, 37, 38)}),
	)
}

func TestRouteMouseEventDrag(t *testing.T) {
	appDragW = 200
	app := NewApplication()
	app.Pipeline = render.NewPipeline(800, 400, canvas.NewSoftCanvas(800, 400))
	app.RootElement = widget.CreateElementFor(widget.NewOverlayHost(&appDragShell{}))
	app.RootElement.Mount(nil, 0)
	app.Pipeline.SetRootElement(app.RootElement)

	widget.OnNeedsRepaint = func() { app.Pipeline.MarkNeedsRepaint() }
	widget.OnNeedsLayout = func() { app.Pipeline.MarkNeedsLayout() }
	widget.RequestPointerCapture = func(el widget.Element) { app.capturedElement = el }
	defer func() {
		widget.OnNeedsRepaint = nil
		widget.OnNeedsLayout = nil
		widget.RequestPointerCapture = nil
	}()

	app.Pipeline.MarkNeedsLayout()
	app.Pipeline.EnsureLayout()

	feed := func(typ event.Type, x, y float64) {
		app.Pipeline.EnsureLayout()
		app.routeMouseEvent(event.NewMouseEvent(typ, x, y, 0, 0))
	}

	hx := appDragW + 3 // handle 中线（紧贴 sidePanel 右侧）
	hy := 200.0        // 落在 body 区（titleBar 36 与 status 之间）

	// 真实序列：先 hover 到 handle 再按下（使 hoveredElement=handle）。
	feed(event.TypeMouseMove, hx, hy)
	feed(event.TypeMouseDown, hx, hy)
	if app.capturedElement == nil {
		t.Fatal("down 后无 capture（HitTest 没命中 handle）")
	}
	t.Logf("after down: captured=%T appDragW=%v", app.capturedElement, appDragW)

	// 关键复现：拖动中鼠标快速移出这条 6px 窄条（落到右侧 editor 区），
	// app 据 hover 变化给 handle 派发 MouseLeave。修复前 → dragging 被清 → 拖动卡死。
	feed(event.TypeMouseMove, hx+100, hy)
	t.Logf("after move-OUT(+100): appDragW=%v captured=%T", appDragW, app.capturedElement)
	feed(event.TypeMouseMove, hx+160, hy)
	t.Logf("after move(+160): appDragW=%v captured=%T", appDragW, app.capturedElement)

	if appDragW <= 200 {
		t.Errorf("MouseLeave 中断了拖动：appDragW=%v（应 >200——移出窄条后拖动仍须生效）", appDragW)
	}
}
