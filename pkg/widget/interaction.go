package widget

import "github.com/hoonfeng/goui/pkg/types"

// ── CSS 伪类：组件级交互态色覆盖基建 ──────────────────────────
//
// 触发器 / 选择类组件（Select / DatePicker / Cascader / TimePicker …）的边框在
// 常态 / 悬停 / 聚焦(激活) 三态间变色，默认走调色板（elBorder / elPrimary，可换肤）。
// 组件 Widget 嵌入 pseudoColors 后，这些色可被 Style 的 :hover/:focus 或链式 setter
// 按组件覆盖 —— 与 Container / Input / Button 同一套伪类语义。
//
// 接入一个组件只需两步：
//  1. 在 Widget struct 内嵌 `pseudoColors`（自动获得 applyInteraction + 链式 setter，
//     从而实现 interactionStyler 接口，被声明式/Styled/Div 三入口分派）。
//  2. Paint 取边框色处把 `elBorder()` 换成 `w.borderOr(elBorder())`、
//     悬停色换 `w.hoverBorderOr(elPrimary())`、激活/聚焦色换 `w.focusBorderOr(elPrimary())`。

// pseudoColors 可内嵌组件 Widget 的交互态边框色覆盖（nil 字段=用默认调色板）。
type pseudoColors struct {
	ovBorder      *types.Color // 常态边框覆盖
	ovHoverBorder *types.Color // :hover 边框覆盖
	ovFocusBorder *types.Color // :focus / 激活态边框覆盖
}

// applyInteraction 从 Style 的伪类提取交互态边框色（实现 interactionStyler）。
func (p *pseudoColors) applyInteraction(s Style) {
	if s.BorderColor != nil {
		p.ovBorder = s.BorderColor
	}
	if s.Hover != nil && s.Hover.BorderColor != nil {
		p.ovHoverBorder = s.Hover.BorderColor
	}
	if s.Focus != nil && s.Focus.BorderColor != nil {
		p.ovFocusBorder = s.Focus.BorderColor
	}
}

// borderOr / hoverBorderOr / focusBorderOr 返回覆盖色，未设则回退 def（调色板默认）。
func (p *pseudoColors) borderOr(def types.Color) types.Color {
	if p.ovBorder != nil {
		return *p.ovBorder
	}
	return def
}

func (p *pseudoColors) hoverBorderOr(def types.Color) types.Color {
	if p.ovHoverBorder != nil {
		return *p.ovHoverBorder
	}
	return def
}

func (p *pseudoColors) focusBorderOr(def types.Color) types.Color {
	if p.ovFocusBorder != nil {
		return *p.ovFocusBorder
	}
	return def
}

// SetHoverBorderColor / SetFocusBorderColor 设置 :hover / :focus(激活) 边框覆盖色。
// 内嵌 pseudoColors 的组件（Select/DatePicker/Cascader/Checkbox…）自动提升这两个方法，
// 可直接 `sel.SetHoverBorderColor(c)` 按组件定制，无需经 Style/Styled。
func (p *pseudoColors) SetHoverBorderColor(c types.Color) { p.ovHoverBorder = &c }
func (p *pseudoColors) SetFocusBorderColor(c types.Color) { p.ovFocusBorder = &c }

// interactionStyler 组件实现它即接受 CSS 伪类（:hover/:focus/:active）交互态色覆盖。
// 内嵌 pseudoColors 即自动实现。Input/Button 走各自的 applyToXxx（含排版），不走此接口。
type interactionStyler interface {
	applyInteraction(s Style)
}
