package widget

import (
	"strings"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Select 下拉选择器（1:1 复刻 el-select）─────────────────────────
// 触发器(显示选中项/placeholder + 下拉箭头) + 点击弹出选项浮层(复用 Overlay 地基)。
// 支持：单选 / 多选(选中变 mini-tag、可删) / 可搜索(下拉顶部输入过滤) / 可清除(×)。

// SelectOption 一个下拉选项。
type SelectOption struct {
	Label    string
	Value    string
	Disabled bool
	Icon     string // 可选左侧图标字符(任意 Unicode/emoji，如 "★"/"📁")
}

// Select 下拉选择器。
type Select struct {
	StatelessWidget
	pseudoColors // CSS 伪类：交互态边框色覆盖（:hover/:focus）
	Options        []SelectOption
	Value          string   // 当前选中值(单选)
	Values         []string // 当前选中值(多选)
	Placeholder    string
	OnChanged      func(string)
	OnChangedMulti func([]string)
	Multiple       bool // 多选
	Filterable     bool // 可搜索(下拉顶部输入过滤)
	Clearable      bool // hover 显示 × 清除
	Disabled       bool
	Width          float64
	Size           string // large / "" (default) / small
}

// NewSelect 创建下拉选择器。
func NewSelect(options []SelectOption) *Select {
	return &Select{Options: options, Width: 220} // Placeholder 空→渲染取 i18n 默认
}

func (s *Select) WithValue(v string) *Select           { s.Value = v; return s }
func (s *Select) WithPlaceholder(p string) *Select      { s.Placeholder = p; return s }
func (s *Select) WithOnChanged(fn func(string)) *Select { s.OnChanged = fn; return s }
func (s *Select) WithDisabled(d bool) *Select           { s.Disabled = d; return s }
func (s *Select) WithWidth(w float64) *Select           { s.Width = w; return s }
func (s *Select) WithSize(sz string) *Select            { s.Size = sz; return s }

func (s *Select) WithMultiple(b bool) *Select                  { s.Multiple = b; return s }
func (s *Select) WithValues(v []string) *Select               { s.Values = v; return s }
func (s *Select) WithOnChangedMulti(fn func([]string)) *Select { s.OnChangedMulti = fn; return s }
func (s *Select) WithFilterable(b bool) *Select               { s.Filterable = b; return s }
func (s *Select) WithClearable(b bool) *Select                { s.Clearable = b; return s }

func (s *Select) height() float64 {
	switch s.Size {
	case "large":
		return 40
	case "small":
		return 24
	default:
		return 32
	}
}

// selectedLabel 返回当前选中项的 Label（单选，无选中返回空）。
func (s *Select) selectedLabel() string { return s.labelOf(s.Value) }

func (s *Select) labelOf(value string) string {
	for _, o := range s.Options {
		if o.Value == value {
			return o.Label
		}
	}
	return ""
}

func selectContains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func (s *Select) CreateElement() Element {
	return &SelectElement{BaseElement: BaseElement{widget: s}, sel: s}
}

// SelectElement 下拉选择器的运行时 Element（触发器）。
type SelectElement struct {
	BaseElement
	sel        *Select
	overlayID  int
	open       bool
	hovered    bool
	tagEls     []Element // 多选：复用 Tag 的子元素（可关闭）
	multiH     float64   // 多选自适应高度
	clearShown bool      // 当前是否显示清除 ×
}

// Build：多选时为每个选中值建一个可关闭 Tag 子元素（复用 Tag 自带的 ×/OnClose），
// 单选无子。Values 变 → OnNeedsLayout → 重建。
func (e *SelectElement) Build() []Element {
	for _, el := range e.tagEls {
		el.Unmount()
	}
	e.tagEls = e.tagEls[:0]
	if !e.sel.Multiple {
		e.children = nil
		return nil
	}
	for _, v := range e.sel.Values {
		val := v
		tag := NewTag(e.sel.labelOf(v)).WithClosable(true).OnCloseFn(func() { e.toggleValue(val) })
		el := CreateElementFor(tag)
		el.Mount(e, len(e.tagEls))
		e.tagEls = append(e.tagEls, el)
	}
	e.children = e.tagEls
	return e.children
}

// layoutMultiTags 把多选 Tag 子元素横排换行布局，并算触发器自适应高度。
func (e *SelectElement) layoutMultiTags(W float64) {
	x := 8.0
	lineTop := 5.0
	const rowGap = 6.0
	maxX := W - 28 // 右侧留给箭头/清除
	tagH := 24.0
	for _, el := range e.tagEls {
		res := el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: W - 16, MinHeight: 0, MaxHeight: 24,
		}})
		tw := res.Size.Width
		tagH = res.Size.Height
		if x > 8 && x+tw > maxX {
			x = 8
			lineTop += tagH + rowGap
		}
		el.SetPosition(types.Point{X: x, Y: lineTop})
		x += tw + rowGap
	}
	e.multiH = lineTop + tagH + 5
	if e.multiH < e.sel.height() {
		e.multiH = e.sel.height()
	}
}

func (e *SelectElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	s := e.sel
	w := s.Width
	if w <= 0 {
		w = 220
	}
	if s.Multiple {
		e.layoutMultiTags(w)
		e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: e.multiH})
	} else {
		e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: s.height()})
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *SelectElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	s := e.sel
	W, H := e.size.Width, e.size.Height

	border := s.borderOr(elBorder())
	if (e.hovered || e.open) && !s.Disabled {
		if e.open {
			border = s.focusBorderOr(elPrimary()) // 展开≈聚焦激活态
		} else {
			border = s.hoverBorderOr(elPrimary())
		}
	}
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	if s.Disabled {
		bg.Color = elFill()
	}
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	font := canvas.DefaultFont()
	font.Size = 14
	placeholder := orI18n(s.Placeholder, "el.select.placeholder") // 空→i18n 默认「请选择/Select」

	if s.Multiple {
		if len(s.Values) == 0 {
			ph := elPlaceholder()
			canvas.DrawTextAligned(cvs, placeholder, types.Rect{X: pos.X + 12, Y: pos.Y, Width: W - 40, Height: s.height()},
				font, ph, canvas.HAlignLeft, canvas.VAlignMiddle)
		} else {
			for _, el := range e.tagEls {
				el.Paint(cvs, offset)
			}
		}
	} else {
		label := s.selectedLabel()
		textColor := elTextRegular()
		if label == "" {
			label = placeholder
			textColor = elPlaceholder()
		}
		if s.Disabled {
			textColor = elPlaceholder()
		}
		textRect := types.Rect{X: pos.X + 12, Y: pos.Y, Width: W - 12 - 28, Height: H}
		canvas.DrawTextAligned(cvs, label, textRect, font, textColor, canvas.HAlignLeft, canvas.VAlignMiddle)
	}

	// 右侧：清除 × 或下拉箭头
	hasVal := s.Value != "" || len(s.Values) > 0
	e.clearShown = s.Clearable && hasVal && e.hovered && !s.Disabled
	acx := pos.X + W - 16
	acy := pos.Y + H/2
	if e.clearShown {
		xp := paint.DefaultStrokePaint()
		xp.Color = elTextSecondary()
		xp.StrokeWidth = 1.4
		cvs.DrawLine(acx-4, acy-4, acx+4, acy+4, xp)
		cvs.DrawLine(acx-4, acy+4, acx+4, acy-4, xp)
	} else {
		arrow := paint.DefaultStrokePaint()
		arrow.Color = elPlaceholder()
		arrow.StrokeWidth = 1.4
		if e.open {
			cvs.DrawLine(acx-4, acy+2, acx, acy-2, arrow)
			cvs.DrawLine(acx, acy-2, acx+4, acy+2, arrow)
		} else {
			cvs.DrawLine(acx-4, acy-2, acx, acy+2, arrow)
			cvs.DrawLine(acx, acy+2, acx+4, acy-2, arrow)
		}
	}
}

func (e *SelectElement) HandleEvent(ev event.Event) bool {
	if e.sel.Disabled {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			return false
		}
		pos := e.Offset()
		lx := me.X - pos.X
		// 多选 tag 的 × 删除由 Tag 子元素自带的 OnClose 处理（事件自动路由），此处无需命中测试。
		// 清除 ×
		if e.clearShown && lx >= e.size.Width-28 {
			e.clearAll()
			return true
		}
		if e.open {
			e.hide()
		} else {
			e.show()
		}
		e.MarkNeedsPaint()
		return true
	}
	return false
}

func (e *SelectElement) toggleValue(v string) {
	found := -1
	for i, x := range e.sel.Values {
		if x == v {
			found = i
			break
		}
	}
	if found >= 0 {
		e.sel.Values = append(e.sel.Values[:found], e.sel.Values[found+1:]...)
	} else {
		e.sel.Values = append(e.sel.Values, v)
	}
	if e.sel.OnChangedMulti != nil {
		e.sel.OnChangedMulti(e.sel.Values)
	}
	if OnNeedsLayout != nil {
		OnNeedsLayout() // 触发器 tag 重排 + 下拉打勾刷新
	}
}

func (e *SelectElement) clearAll() {
	e.sel.Value = ""
	e.sel.Values = nil
	if e.sel.OnChanged != nil {
		e.sel.OnChanged("")
	}
	if e.sel.OnChangedMulti != nil {
		e.sel.OnChangedMulti(nil)
	}
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	}
}

func (e *SelectElement) show() {
	if e.open {
		return
	}
	off := e.Offset()
	dd := &selectDropdown{
		options:    e.sel.Options,
		value:      e.sel.Value,
		multiple:   e.sel.Multiple,
		filterable: e.sel.Filterable,
		width:      e.size.Width,
		getValues:  func() []string { return e.sel.Values },
		onSelect: func(v string) {
			if e.sel.Multiple {
				e.toggleValue(v)
			} else {
				e.sel.Value = v
				if e.sel.OnChanged != nil {
					e.sel.OnChanged(v)
				}
				e.hide()
				e.MarkNeedsPaint()
			}
		},
	}
	entry := &OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: e.size.Width, Height: e.size.Height},
		Content:        dd,
		Placement:      PlacementBottomStart,
		Gap:            4,
		Transition:     "fade",
		OnClickOutside: func() { e.hide(); e.MarkNeedsPaint() },
	}
	e.overlayID = ShowOverlay(entry)
	e.open = true
}

func (e *SelectElement) hide() {
	if !e.open {
		return
	}
	HideOverlay(e.overlayID)
	e.open = false
}

func (e *SelectElement) Update(newWidget Widget) {
	if ns, ok := newWidget.(*Select); ok {
		e.sel = ns
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── selectDropdown 选项浮层 ────────────────────────────────────

type selectDropdown struct {
	StatelessWidget
	options    []SelectOption
	value      string
	multiple   bool
	filterable bool
	width      float64
	getValues  func() []string
	onSelect   func(string)
}

func (d *selectDropdown) CreateElement() Element {
	return &selectDropdownElement{BaseElement: BaseElement{widget: d}, dd: d, hoverIdx: -1}
}

const (
	selectItemH   = 34.0
	selectSearchH = 40.0
)

type selectDropdownElement struct {
	BaseElement
	dd       *selectDropdown
	hoverIdx int
	searchEl Element
	filter   string
}

func (e *selectDropdownElement) Build() []Element {
	if !e.dd.filterable {
		e.children = nil
		return nil
	}
	if e.searchEl == nil {
		in := NewInput("搜索", func(s string) {
			e.filter = s
			e.hoverIdx = -1
			if OnNeedsLayout != nil {
				OnNeedsLayout()
			}
		})
		e.searchEl = CreateElementFor(in)
		e.searchEl.Mount(e, 0)
	}
	e.children = []Element{e.searchEl}
	return e.children
}

// visible 返回经搜索过滤后的可见选项。
func (e *selectDropdownElement) visible() []SelectOption {
	if !e.dd.filterable || e.filter == "" {
		return e.dd.options
	}
	lf := strings.ToLower(e.filter)
	var out []SelectOption
	for _, o := range e.dd.options {
		if strings.Contains(strings.ToLower(o.Label), lf) {
			out = append(out, o)
		}
	}
	return out
}

func (e *selectDropdownElement) searchH() float64 {
	if e.dd.filterable {
		return selectSearchH
	}
	return 0
}

func (e *selectDropdownElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	sh := e.searchH()
	if e.dd.filterable && e.searchEl != nil {
		e.searchEl.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: e.dd.width - 16, MinHeight: 0, MaxHeight: 30,
		}})
		e.searchEl.SetPosition(types.Point{X: 8, Y: 6})
	}
	n := len(e.visible())
	h := sh + float64(n)*selectItemH + 8
	e.size = ctx.Constraints.Constrain(types.Size{Width: e.dd.width, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *selectDropdownElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height

	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 22)
	cvs.DrawRoundedRect(pos.X, pos.Y+2, W, H, 4, shadow)
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = elBorderLight()
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	sh := e.searchH()
	if e.dd.filterable {
		// 搜索框分隔线
		cvs.DrawLine(pos.X+1, pos.Y+sh, pos.X+W-1, pos.Y+sh, bp)
	}

	font := canvas.DefaultFont()
	font.Size = 14
	vis := e.visible()
	var values []string
	if e.dd.multiple && e.dd.getValues != nil {
		values = e.dd.getValues()
	}
	for i, opt := range vis {
		iy := pos.Y + sh + 4 + float64(i)*selectItemH
		selected := opt.Value == e.dd.value
		if e.dd.multiple {
			selected = selectContains(values, opt.Value)
		}

		if i == e.hoverIdx && !opt.Disabled {
			hp := paint.DefaultPaint()
			hp.Color = elFill()
			cvs.DrawRect(pos.X+1, iy, W-2, selectItemH, hp)
		}

		color := elTextRegular()
		if selected {
			color = elPrimary()
		}
		if opt.Disabled {
			color = elPlaceholder()
		}
		f := font
		if selected {
			f.Weight = canvas.FontWeightBold
		}
		textX := pos.X + 14
		if opt.Icon != "" {
			canvas.DrawTextAligned(cvs, opt.Icon, types.Rect{X: pos.X + 12, Y: iy, Width: 18, Height: selectItemH}, f, color, canvas.HAlignLeft, canvas.VAlignMiddle)
			textX += 22
		}
		tr := types.Rect{X: textX, Y: iy, Width: pos.X + W - 30 - textX, Height: selectItemH}
		canvas.DrawTextAligned(cvs, opt.Label, tr, f, color, canvas.HAlignLeft, canvas.VAlignMiddle)

		if selected {
			cp := paint.DefaultStrokePaint()
			cp.Color = elPrimary()
			cp.StrokeWidth = 1.6
			cx := pos.X + W - 22
			cy := iy + selectItemH/2
			cvs.DrawLine(cx, cy, cx+3, cy+3, cp)
			cvs.DrawLine(cx+3, cy+3, cx+8, cy-4, cp)
		}
	}

	// 空状态
	if len(vis) == 0 {
		canvas.DrawTextAligned(cvs, "无匹配项", types.Rect{X: pos.X, Y: pos.Y + sh + 4, Width: W, Height: selectItemH},
			font, elPlaceholder(), canvas.HAlignCenter, canvas.VAlignMiddle)
	}

	if e.dd.filterable && e.searchEl != nil {
		e.searchEl.Paint(cvs, offset)
	}
}

// itemAt 返回局部 y 对应的可见选项索引（-1 表示空白/搜索框区）。
func (e *selectDropdownElement) itemAt(localY float64) int {
	idx := int((localY - e.searchH() - 4) / selectItemH)
	if idx < 0 || idx >= len(e.visible()) {
		return -1
	}
	return idx
}

func (e *selectDropdownElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		if me, ok := ev.(*event.MouseEvent); ok {
			e.hoverIdx = e.itemAt(me.Y - e.Offset().Y)
			e.MarkNeedsPaint()
			return true
		}
	case event.TypeMouseLeave:
		e.hoverIdx = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if me, ok := ev.(*event.MouseEvent); ok {
			idx := e.itemAt(me.Y - e.Offset().Y)
			vis := e.visible()
			if idx >= 0 && idx < len(vis) && !vis[idx].Disabled && e.dd.onSelect != nil {
				e.dd.onSelect(vis[idx].Value)
			}
			return true
		}
	}
	return false
}

func (e *selectDropdownElement) Update(newWidget Widget) {
	if nd, ok := newWidget.(*selectDropdown); ok {
		e.dd = nd
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// SelectDropdownPreview 返回独立的下拉面板 widget，仅供 demo/测试预览下拉外观
// （正常使用时下拉由 SelectElement 点击弹出、挂在 Overlay 顶层）。
func SelectDropdownPreview(options []SelectOption, value string, width float64) Widget {
	return &selectDropdown{options: options, value: value, width: width}
}

// SelectDropdownPreviewMulti 预览多选/可搜索下拉外观（多选打勾 + 顶部搜索框）。
func SelectDropdownPreviewMulti(options []SelectOption, values []string, filterable bool, width float64) Widget {
	return &selectDropdown{
		options:    options,
		multiple:   true,
		filterable: filterable,
		width:      width,
		getValues:  func() []string { return values },
	}
}
