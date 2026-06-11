package app

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// 复现「按钮内有图标+文字容器子，点击落在子元素上 → OnClick 不触发」（与标题栏同源）。
// MouseDown 把最深命中元素（内层 Div/Text）设为 capturedElement；MouseUp 走捕获路径，
// 修复前只发给捕获元素本身、不冒泡 → 祖先 Button 收不到 up、OnClick 不触发。
// 修复后捕获路径沿父链冒泡，Button 正常触发。
func TestButtonClickThroughChildContainer(t *testing.T) {
	clicked := 0
	btn := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("send", widget.IconSize(12)),
			widget.Div(widget.Style{Width: 4}),
			widget.NewText("发送", types.ColorWhite),
		)},
		OnClick:  func() { clicked++ },
		MinWidth: 120, MinHeight: 40,
	}
	app := NewApplication()
	app.Pipeline = render.NewPipeline(400, 200, canvas.NewSoftCanvas(400, 200))
	// 按钮作为根，布局在 (0,0)，尺寸 = MinWidth×MinHeight=120×40，确定可控。
	app.RootElement = widget.CreateElementFor(btn)
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

	// 按钮 (0,0)~(120,40)，子内容（图标+文字）居中其中。点 (60,20) 落在内层装饰子上，
	// 正是要复现的「点子元素而非 Button 本身」场景。
	cx, cy := 60.0, 20.0
	hit := app.Pipeline.HitTest(cx, cy)
	if hit == nil {
		t.Fatal("HitTest 未命中任何元素（按钮没布局到预期位置）")
	}
	if _, isBtn := hit.(*widget.ButtonElement); isBtn {
		t.Fatalf("HitTest 命中 Button 本身（%T），无法复现「点击落在子元素」；换个坐标", hit)
	}
	t.Logf("命中子元素 = %T（非 Button，符合复现前提）", hit)
	foundBtn := false
	for e := hit; e != nil; e = e.Parent() {
		t.Logf("  父链: %T", e)
		if _, ok := e.(*widget.ButtonElement); ok {
			foundBtn = true
		}
	}
	if !foundBtn {
		t.Fatalf("命中元素的父链里没有 Button —— 冒泡到不了 Button（布局/结构问题）")
	}

	feed := func(typ event.Type) {
		app.Pipeline.EnsureLayout()
		app.routeMouseEvent(event.NewMouseEvent(typ, cx, cy, 0, 0))
	}
	feed(event.TypeMouseDown)
	feed(event.TypeMouseUp)

	if clicked != 1 {
		t.Errorf("OnClick 触发 %d 次，期望 1——MouseUp 在捕获路径未冒泡到祖先 Button", clicked)
	}
}
