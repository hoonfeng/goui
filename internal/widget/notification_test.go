package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
)

// TestNotificationLayout 通知卡片固定尺寸。
func TestNotificationLayout(t *testing.T) {
	el := (&notificationCard{title: "标题", message: "内容", ntype: MsgSuccess}).CreateElement().(*notificationCardElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 500, MaxHeight: 200}})
	if el.size.Width != notifyW || el.size.Height != notifyH {
		t.Errorf("size=%.0fx%.0f, want %.0fx%.0f", el.size.Width, el.size.Height, notifyW, notifyH)
	}
}

// TestNotificationCloseButton 点右上角 ✕ 触发 onClose。
func TestNotificationCloseButton(t *testing.T) {
	closed := false
	card := &notificationCard{title: "t", message: "m", ntype: MsgInfo, onClose: func() { closed = true }}
	el := card.CreateElement().(*notificationCardElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 500, MaxHeight: 200}})
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, notifyW-16, 16, event.ButtonLeft, event.ModNone))
	if !closed {
		t.Error("点右上角 ✕ 应触发 onClose")
	}
}

// TestNotificationBodyNoClose 点卡片主体(非✕)不关闭。
func TestNotificationBodyNoClose(t *testing.T) {
	closed := false
	card := &notificationCard{title: "t", message: "m", ntype: MsgInfo, onClose: func() { closed = true }}
	el := card.CreateElement().(*notificationCardElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 500, MaxHeight: 200}})
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 60, 40, event.ButtonLeft, event.ModNone)) // 主体中部
	if closed {
		t.Error("点主体不应关闭")
	}
}
