package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
)

func looseBox() *layout.LayoutContext {
	return &layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 400, MaxHeight: 400}}
}

func TestBadgeDisplayText(t *testing.T) {
	e := (&Badge{Value: "100", Max: 99}).CreateElement().(*BadgeElement)
	if got := e.displayText(); got != "99+" {
		t.Errorf("超 max 应截断: got %q want 99+", got)
	}
	e2 := (&Badge{Value: "5", Max: 99}).CreateElement().(*BadgeElement)
	if got := e2.displayText(); got != "5" {
		t.Errorf("未超 max: got %q want 5", got)
	}
}

func TestAvatarSize(t *testing.T) {
	r := NewAvatar("A").CreateElement().(*AvatarElement).Layout(looseBox())
	if r.Size.Width != 40 || r.Size.Height != 40 {
		t.Errorf("默认应 40x40, got %vx%v", r.Size.Width, r.Size.Height)
	}
	r2 := NewAvatar("A").WithSize(60).CreateElement().(*AvatarElement).Layout(looseBox())
	if r2.Size.Width != 60 {
		t.Errorf("自定义尺寸 60, got %v", r2.Size.Width)
	}
}

func TestLinkClickDisabled(t *testing.T) {
	clicked := false
	e := NewLink("链接", func() { clicked = true }).CreateElement().(*LinkElement)
	e.Layout(looseBox())
	e.HandleEvent(event.NewMouseEvent(event.TypeMouseUp, 0, 0, event.ButtonLeft, event.ModNone))
	if !clicked {
		t.Error("点击应触发 OnClick")
	}
	clicked = false
	e2 := NewLink("禁用", func() { clicked = true }).WithDisabled(true).CreateElement().(*LinkElement)
	e2.HandleEvent(event.NewMouseEvent(event.TypeMouseUp, 0, 0, event.ButtonLeft, event.ModNone))
	if clicked {
		t.Error("disabled link 不应触发 click")
	}
}

func TestAlertClose(t *testing.T) {
	closed := false
	a := NewAlert("警告", AlertWarning).WithClosable(true)
	a.OnClose = func() { closed = true }
	e := a.CreateElement().(*AlertElement)
	e.Build()
	e.Layout(looseBox())
	cx := e.Offset().X + e.size.Width - 16
	cy := e.Offset().Y + e.size.Height/2
	e.HandleEvent(event.NewMouseEvent(event.TypeMouseUp, cx, cy, event.ButtonLeft, event.ModNone))
	if !closed || !e.alert.closed {
		t.Errorf("点 × 应关闭 alert: closed=%v elClosed=%v", closed, e.alert.closed)
	}
}

func TestEmptyLayout(t *testing.T) {
	e := NewEmpty("暂无数据").CreateElement().(*EmptyElement)
	e.Build()
	r := e.Layout(looseBox())
	if r.Size.Height <= 0 {
		t.Error("empty 高度应 > 0")
	}
}

func TestSimpleWidgetsRegistered(t *testing.T) {
	for _, name := range []string{"Badge", "Avatar", "Link", "Alert", "Empty"} {
		if _, ok := componentRegistry[name]; !ok {
			t.Errorf("%s 未注册到声明式系统", name)
		}
	}
}
