package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

func layoutDropdownMenu(items []DropdownItem, onSelect func(string)) *dropdownMenuElement {
	m := &dropdownMenu{items: items, onSelect: onSelect}
	el := m.CreateElement().(*dropdownMenuElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 300, MaxHeight: 400}})
	return el
}

// TestDropdownMenuClick 点菜单项触发对应 Command。
func TestDropdownMenuClick(t *testing.T) {
	got := ""
	el := layoutDropdownMenu([]DropdownItem{
		{Label: "编辑", Command: "edit"},
		{Label: "删除", Command: "delete"},
	}, func(c string) { got = c })
	// 第二项(删除)：y = 4 + 1*34 + 17 = 55
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 50, 55, event.ButtonLeft, event.ModNone))
	if got != "delete" {
		t.Errorf("点第二项 Command=%q, want delete", got)
	}
}

// TestDropdownDisabledItem 禁用项不触发。
func TestDropdownDisabledItem(t *testing.T) {
	got := "none"
	el := layoutDropdownMenu([]DropdownItem{
		{Label: "a", Command: "a"},
		{Label: "b", Command: "b", Disabled: true},
	}, func(c string) { got = c })
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 50, 55, event.ButtonLeft, event.ModNone))
	if got != "none" {
		t.Errorf("禁用项不应触发, got %q", got)
	}
}

// TestDropdownTriggerToggle 点触发器弹出/关闭(click 模式)。
func TestDropdownTriggerToggle(t *testing.T) {
	prev := OnOverlayChanged
	OnOverlayChanged = func() {}
	defer func() { OnOverlayChanged = prev }()
	overlayEntries = nil

	d := NewDropdown(NewText("菜单", types.ColorFromRGB(0, 0, 0)),
		DropdownItem{Label: "a", Command: "a"})
	el := d.CreateElement().(*DropdownElement)
	el.Build()
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 200, MaxHeight: 50}})
	if el.shown {
		t.Fatal("初始未显示")
	}
	click := event.NewMouseEvent(event.TypeMouseDown, 5, 5, event.ButtonLeft, event.ModNone)
	el.HandleEvent(click)
	if !el.shown {
		t.Error("click 应弹出菜单")
	}
	el.HandleEvent(click)
	if el.shown {
		t.Error("再 click 应关闭")
	}
}
