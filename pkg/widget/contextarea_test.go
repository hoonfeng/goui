package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
)

// ContextArea 捕获右键 → 回调屏幕坐标 + StopPropagation；非右键不触发。
func TestContextAreaCatchesRightClick(t *testing.T) {
	var gotX, gotY float64
	called := false
	el := (&ContextArea{OnContextMenu: func(x, y float64) { called = true; gotX, gotY = x, y }}).
		CreateElement().(*ContextAreaElement)

	ev := event.NewMouseEvent(event.TypeContextMenu, 42, 99, event.ButtonNone, event.ModifierKeys(0))
	if handled := el.HandleEvent(ev); !handled || !called || gotX != 42 || gotY != 99 {
		t.Fatalf("右键应触发 OnContextMenu(42,99)：handled=%v called=%v (%v,%v)", handled, called, gotX, gotY)
	}
	if !ev.IsPropagationStopped() {
		t.Error("右键应 StopPropagation（避免外层 ContextArea 重复触发）")
	}

	called = false
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 1, 1, event.ButtonNone, event.ModifierKeys(0)))
	if called {
		t.Error("左键/普通事件不应触发 OnContextMenu")
	}
}
