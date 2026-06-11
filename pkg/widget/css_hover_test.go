package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/types"
)

// TestContainerPseudoPriority 验证 CSS :hover / :focus / :active 状态优先级：
// Style 伪类经 applyToContainer 挂到 Container，ContainerElement 按 hover<focus<active 解析生效属性。
func TestContainerPseudoPriority(t *testing.T) {
	base := types.ColorFromRGB(255, 0, 0)  // 红
	hov := types.ColorFromRGB(0, 255, 0)   // 绿
	foc := types.ColorFromRGB(0, 0, 255)   // 蓝
	act := types.ColorFromRGB(255, 255, 0) // 黄
	bd := types.ColorFromRGB(10, 20, 30)

	c := &Container{}
	Style{
		BackgroundColor: &base,
		Hover:           &Style{BackgroundColor: &hov, BorderColor: &bd},
		Focus:           &Style{BackgroundColor: &foc},
		Active:          &Style{BackgroundColor: &act},
	}.applyToContainer(c)

	if c.Hover == nil || c.Focus == nil || c.Active == nil {
		t.Fatal("applyToContainer 应把三个伪类样式挂到 Container")
	}
	e := &ContainerElement{container: c}

	// 默认：基础背景红、无边框
	if got := e.effBg(); got == nil || *got != base {
		t.Errorf("默认背景应红，得 %v", got)
	}
	if e.effBorder() != nil {
		t.Errorf("默认无边框，effBorder 应 nil")
	}

	// :hover —— 背景绿 + 从无到有长出边框
	e.hovered = true
	if got := e.effBg(); got == nil || *got != hov {
		t.Errorf("hover 背景应绿，得 %v", got)
	}
	if b := e.effBorder(); b == nil || b.Top.Color != bd {
		t.Errorf("hover 应长出边框色 %v，得 %v", bd, b)
	}

	// :focus 优先级高于 :hover（背景蓝）
	e.cssFocused = true
	if got := e.effBg(); got == nil || *got != foc {
		t.Errorf("focus 应盖过 hover，背景蓝，得 %v", got)
	}

	// :active 优先级最高（背景黄）
	e.pressed = true
	if got := e.effBg(); got == nil || *got != act {
		t.Errorf("active 应盖过 focus，背景黄，得 %v", got)
	}
}

// TestContainerPseudoEvents 验证 HandleEvent / Focus / Blur 驱动状态切换。
func TestContainerPseudoEvents(t *testing.T) {
	base := types.ColorFromRGB(200, 200, 200)
	hov := types.ColorFromRGB(100, 100, 100)
	act := types.ColorFromRGB(50, 50, 50)
	ring := types.ColorFromRGB(64, 158, 255)
	c := &Container{Background: &PaintWidget{Color: &base}}
	Style{
		Hover:  &Style{BackgroundColor: &hov},
		Active: &Style{BackgroundColor: &act},
		Focus:  &Style{BorderColor: &ring},
	}.applyToContainer(c)
	e := &ContainerElement{container: c}

	md := func(typ event.Type) { e.HandleEvent(event.NewMouseEvent(typ, 0, 0, event.ButtonLeft, 0)) }

	// MouseEnter → hover 背景
	md(event.TypeMouseEnter)
	if !e.hovered || *e.effBg() != hov {
		t.Errorf("MouseEnter 后应 hover")
	}
	// MouseDown → active（最高优先）
	md(event.TypeMouseDown)
	if !e.pressed || *e.effBg() != act {
		t.Errorf("MouseDown 后应 active")
	}
	// MouseUp → 退回 hover
	md(event.TypeMouseUp)
	if e.pressed || *e.effBg() != hov {
		t.Errorf("MouseUp 后应退回 hover")
	}
	// MouseLeave → 退回基础
	md(event.TypeMouseLeave)
	if e.hovered || *e.effBg() != base {
		t.Errorf("MouseLeave 后应退回基础背景")
	}
	// Focus → cssFocused（focus 有 BorderColor，长出 focus ring）
	e.Focus()
	if !e.cssFocused || e.effBorder() == nil {
		t.Errorf("Focus 后应 focus 态并长出边框")
	}
	// Blur → 清除
	e.Blur()
	if e.cssFocused || e.effBorder() != nil {
		t.Errorf("Blur 后应清除 focus 边框")
	}
}
