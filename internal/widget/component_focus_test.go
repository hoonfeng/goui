package widget

import (
	"testing"

	"github.com/user/goui/internal/types"
)

// TestInputFocusHoverColor 验证 Input 聚焦/悬停边框色可配（链式 setter + Style 伪类注入）。
func TestInputFocusHoverColor(t *testing.T) {
	red := types.ColorFromRGB(255, 0, 0)
	green := types.ColorFromRGB(0, 255, 0)

	// 链式 setter
	in := DefaultInput().WithFocusColor(red).WithHoverColor(green)
	if in.FocusBorderColor != red {
		t.Errorf("WithFocusColor 应设聚焦边框红，得 %v", in.FocusBorderColor)
	}
	if in.HoverBorderColor != green {
		t.Errorf("WithHoverColor 应设悬停边框绿，得 %v", in.HoverBorderColor)
	}

	// Style 伪类注入（Styled / Div 路径）
	in2 := DefaultInput()
	Style{
		Focus: &Style{BorderColor: &red},
		Hover: &Style{BorderColor: &green},
	}.applyToInput(in2)
	if in2.FocusBorderColor != red || in2.HoverBorderColor != green {
		t.Errorf("Style :focus/:hover 应注入 Input 交互态边框色，得 focus=%v hover=%v",
			in2.FocusBorderColor, in2.HoverBorderColor)
	}
}

// TestButtonFocusRing 验证 Button 聚焦态切换 + focus ring 色可配。
func TestButtonFocusRing(t *testing.T) {
	blue := types.ColorFromRGB(0, 0, 255)
	b := &Button{Text: "OK"}
	Style{Focus: &Style{BorderColor: &blue}}.applyToButton(b)
	if b.FocusRingColor != blue {
		t.Errorf("Style :focus 应注入 Button.FocusRingColor，得 %v", b.FocusRingColor)
	}

	e := &ButtonElement{button: b}
	if e.focused {
		t.Error("初始不应聚焦")
	}
	e.Focus()
	if !e.focused {
		t.Error("Focus() 后内部 focused 应为真")
	}
	if e.IsFocused() {
		t.Error("IsFocused() 应保持 false（避免触发持续重绘），聚焦态用内部 focused")
	}
	e.Blur()
	if e.focused {
		t.Error("Blur() 后应取消聚焦")
	}
}

// TestButtonStyleTypography 验证 applyToButton 真正应用 TextColor/FontSize（修复此前空注释）。
func TestButtonStyleTypography(t *testing.T) {
	red := types.ColorFromRGB(255, 0, 0)
	b := &Button{}
	Style{TextColor: &red, FontSize: 18}.applyToButton(b)
	if b.TextColor != red {
		t.Errorf("applyToButton 应应用 TextColor，得 %v", b.TextColor)
	}
	if b.FontSize != 18 {
		t.Errorf("applyToButton 应应用 FontSize，得 %v", b.FontSize)
	}
}

// TestTriggerComponentsPseudo 验证触发器/选择类组件接入 CSS 伪类：
// 内嵌 pseudoColors → 实现 interactionStyler → 注入后 hover/focus 取色覆盖生效。
func TestTriggerComponentsPseudo(t *testing.T) {
	red := types.ColorFromRGB(255, 0, 0)
	green := types.ColorFromRGB(0, 255, 0)
	inject := Style{Hover: &Style{BorderColor: &green}, Focus: &Style{BorderColor: &red}}

	// 自绘边框组件都应实现 interactionStyler，注入后取色生效
	comps := []Widget{
		&Select{}, &SelectV2{}, &DatePicker{}, &TimePicker{}, &TreeSelect{}, &Cascader{},
		&Checkbox{}, &RadioButton{},
	}
	for _, w := range comps {
		is, ok := w.(interactionStyler)
		if !ok {
			t.Errorf("%T 应实现 interactionStyler（内嵌 pseudoColors）", w)
			continue
		}
		is.applyInteraction(inject)
		pc, ok := w.(interface {
			hoverBorderOr(types.Color) types.Color
			focusBorderOr(types.Color) types.Color
		})
		if !ok {
			t.Errorf("%T 应有 hoverBorderOr/focusBorderOr", w)
			continue
		}
		if pc.hoverBorderOr(elBorder()) != green {
			t.Errorf("%T :hover 边框应被覆盖为绿", w)
		}
		if pc.focusBorderOr(elBorder()) != red {
			t.Errorf("%T :focus 边框应被覆盖为红", w)
		}
	}
}

// TestDeclarativeComponentFocus 验证声明式 JSON 路径：StyleSpec 伪类经
// applyStyleToWidgetConfig 注入 Input/Button 的交互态色字段（gate 放行 + switch 分派）。
func TestDeclarativeComponentFocus(t *testing.T) {
	in := NewInput("", nil)
	applyStyleToWidgetConfig(in, ComponentSpec{Style: &StyleSpec{Focus: &StyleSpec{BorderColor: "#ff0000"}}})
	if want := types.ColorFromRGB(255, 0, 0); in.FocusBorderColor != want {
		t.Errorf("声明式 :focus 应注入 Input.FocusBorderColor=红，得 %v", in.FocusBorderColor)
	}

	b := &Button{Text: "OK"}
	applyStyleToWidgetConfig(b, ComponentSpec{Style: &StyleSpec{Focus: &StyleSpec{BorderColor: "#0000ff"}}})
	if want := types.ColorFromRGB(0, 0, 255); b.FocusRingColor != want {
		t.Errorf("声明式 :focus 应注入 Button.FocusRingColor=蓝，得 %v", b.FocusRingColor)
	}
}
