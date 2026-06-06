package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
)

func enterLeave(el *ClickableElement, typ event.Type) {
	el.HandleEvent(event.NewMouseEvent(typ, 0, 0, event.ButtonNone, event.ModifierKeys(0)))
}

// OnHoverChange 在进入/离开时各回调一次；重复进入不重复回调。
func TestClickableOnHoverChange(t *testing.T) {
	var states []bool
	el := (&Clickable{OnHoverChange: func(h bool) { states = append(states, h) }}).CreateElement().(*ClickableElement)

	enterLeave(el, event.TypeMouseEnter)
	enterLeave(el, event.TypeMouseLeave)
	if len(states) != 2 || !states[0] || states[1] {
		t.Fatalf("应依次 true,false，得 %v", states)
	}
	enterLeave(el, event.TypeMouseEnter)
	enterLeave(el, event.TypeMouseEnter) // 已 hover，不重复
	if len(states) != 3 {
		t.Errorf("重复 enter 不应重复回调，得 %v", states)
	}
}

// Cursor：无 OnClick → 默认（不误导成可点）；有 OnClick → 手型。
func TestClickableCursorConditional(t *testing.T) {
	hoverOnly := (&Clickable{OnHoverChange: func(bool) {}}).CreateElement().(*ClickableElement)
	if hoverOnly.Cursor() != CursorDefault {
		t.Error("无 OnClick 的 Clickable 应用默认光标")
	}
	clickable := (&Clickable{OnClick: func() {}}).CreateElement().(*ClickableElement)
	if clickable.Cursor() != CursorHand {
		t.Error("有 OnClick 的 Clickable 应用手型光标")
	}
}
