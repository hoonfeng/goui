package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
)

// TestSelectSelectedLabel 选中值映射到对应 Label。
func TestSelectSelectedLabel(t *testing.T) {
	opts := []SelectOption{{Label: "北京", Value: "bj"}, {Label: "上海", Value: "sh"}}
	if got := NewSelect(opts).WithValue("sh").selectedLabel(); got != "上海" {
		t.Errorf("selectedLabel()=%q, want 上海", got)
	}
	if got := NewSelect(opts).selectedLabel(); got != "" {
		t.Errorf("无选中应返回空，得 %q", got)
	}
}

// TestSelectItemAt 下拉局部 y → 选项索引(itemH=34, 上 padding 4)。
func TestSelectItemAt(t *testing.T) {
	opts := []SelectOption{{Label: "a"}, {Label: "b"}, {Label: "c"}}
	el := (&selectDropdown{options: opts, width: 200}).CreateElement().(*selectDropdownElement)
	if el.itemAt(20) != 0 {
		t.Errorf("itemAt(20)=%d, want 0", el.itemAt(20))
	}
	if el.itemAt(50) != 1 {
		t.Errorf("itemAt(50)=%d, want 1", el.itemAt(50))
	}
	if el.itemAt(400) != -1 {
		t.Errorf("itemAt(400)=%d, want -1(越界)", el.itemAt(400))
	}
}

// TestSelectClickSelects 点击选项触发 onSelect。
func TestSelectClickSelects(t *testing.T) {
	got := ""
	opts := []SelectOption{{Label: "北京", Value: "bj"}, {Label: "上海", Value: "sh"}}
	el := (&selectDropdown{options: opts, onSelect: func(v string) { got = v }}).CreateElement().(*selectDropdownElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 200, MaxHeight: 200}})
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 50, 50, event.ButtonLeft, event.ModNone)) // 第 2 项
	if got != "sh" {
		t.Errorf("点第 2 项 onSelect=%q, want sh", got)
	}
}

// TestSelectDisabledOptionNotSelectable 禁用项点击不触发选择。
func TestSelectDisabledOptionNotSelectable(t *testing.T) {
	got := "none"
	opts := []SelectOption{{Label: "a", Value: "a"}, {Label: "b", Value: "b", Disabled: true}}
	el := (&selectDropdown{options: opts, onSelect: func(v string) { got = v }}).CreateElement().(*selectDropdownElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 200, MaxHeight: 200}})
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 50, 50, event.ButtonLeft, event.ModNone)) // 禁用的第 2 项
	if got != "none" {
		t.Errorf("禁用项不应被选中，得 %q", got)
	}
}
