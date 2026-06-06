package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Dropdown 下拉菜单（1:1 复刻 el-dropdown）────────────────────
// 触发元素 + 点击弹出命令菜单(复用 Overlay 浮层地基 + Popover 触发模式)。
// 菜单项点击触发 OnCommand;支持禁用项、分隔线。默认 click 触发(hover 在
// trigger 与菜单间隙易误关，click 更稳)。

type DropdownItem struct {
	Label    string
	Command  string
	Disabled bool
	Divided  bool   // 上方加分隔线
	Icon     string // 可选左侧图标字符（旧）
	Shortcut string // 右侧快捷键提示（灰显），如 "Ctrl+S"
	Checked  bool   // 勾选项（左侧显 ✓ 向量，如视图开关）
}

type Dropdown struct {
	StatelessWidget
	Trigger     Widget
	Items       []DropdownItem
	OnCommand   func(string)
	TriggerMode TooltipTrigger // click(默认) / hover
	Placement   Placement
}

func NewDropdown(trigger Widget, items ...DropdownItem) *Dropdown {
	return &Dropdown{Trigger: trigger, Items: items, TriggerMode: TooltipClick, Placement: PlacementBottomStart}
}
func (d *Dropdown) WithOnCommand(fn func(string)) *Dropdown { d.OnCommand = fn; return d }
func (d *Dropdown) WithTrigger(t TooltipTrigger) *Dropdown  { d.TriggerMode = t; return d }
func (d *Dropdown) WithPlacement(p Placement) *Dropdown     { d.Placement = p; return d }

func (d *Dropdown) CreateElement() Element {
	return &DropdownElement{BaseElement: BaseElement{widget: d}, dropdown: d}
}

type DropdownElement struct {
	BaseElement
	dropdown  *Dropdown
	triggerEl Element
	overlayID int
	shown     bool
}

func (e *DropdownElement) Build() []Element {
	if e.dropdown.Trigger != nil {
		e.triggerEl = CreateElementFor(e.dropdown.Trigger)
		e.triggerEl.Mount(e, 0)
		e.children = []Element{e.triggerEl}
		return e.children
	}
	e.children = nil
	return nil
}

func (e *DropdownElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.triggerEl != nil {
		res := e.triggerEl.Layout(ctx)
		e.triggerEl.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *DropdownElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.triggerEl != nil {
		e.triggerEl.Paint(cvs, offset)
	}
}

func (e *DropdownElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter:
		if e.dropdown.TriggerMode == TooltipHover {
			e.show()
		}
	case event.TypeMouseLeave:
		if e.dropdown.TriggerMode == TooltipHover {
			e.hide()
		}
	case event.TypeMouseDown:
		if e.dropdown.TriggerMode == TooltipClick {
			if e.shown {
				e.hide()
			} else {
				e.show()
			}
		}
	}
	return false
}

func (e *DropdownElement) show() {
	if e.shown || e.triggerEl == nil {
		return
	}
	off := e.triggerEl.Offset()
	sz := e.triggerEl.Size()
	menu := &dropdownMenu{
		items: e.dropdown.Items,
		onSelect: func(cmd string) {
			if e.dropdown.OnCommand != nil {
				e.dropdown.OnCommand(cmd)
			}
			e.hide()
		},
	}
	e.overlayID = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: sz.Width, Height: sz.Height},
		Content:        menu,
		Placement:      e.dropdown.Placement,
		Gap:            4,
		Transition:     "fade",
		OnClickOutside: func() { e.hide() },
	})
	e.shown = true
}

func (e *DropdownElement) hide() {
	if !e.shown {
		return
	}
	HideOverlay(e.overlayID)
	e.shown = false
}

func (e *DropdownElement) Update(newWidget Widget) {
	if nd, ok := newWidget.(*Dropdown); ok {
		e.dropdown = nd
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── dropdownMenu 菜单浮层 ──────────────────────────────────────

type dropdownMenu struct {
	StatelessWidget
	items    []DropdownItem
	onSelect func(string)
}

func (m *dropdownMenu) CreateElement() Element {
	return &dropdownMenuElement{BaseElement: BaseElement{widget: m}, menu: m, hoverIdx: -1}
}

const dropdownItemH = 34.0

type dropdownMenuElement struct {
	BaseElement
	menu     *dropdownMenu
	width    float64
	hoverIdx int
}

func (e *dropdownMenuElement) Build() []Element { return nil }

func (e *dropdownMenuElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	font := canvas.DefaultFont()
	font.Size = 14
	maxLabel, maxShort := 0.0, 0.0
	leftCol := 0.0
	for _, it := range e.menu.items {
		if w := canvas.MeasureTextGlobal(it.Label, font).Width; w > maxLabel {
			maxLabel = w
		}
		if it.Shortcut != "" {
			if w := canvas.MeasureTextGlobal(it.Shortcut, font).Width; w > maxShort {
				maxShort = w
			}
		}
		if it.Checked || it.Icon != "" {
			leftCol = 18 // 勾选/图标列：有则预留，标签对齐
		}
	}
	e.width = 12 + leftCol + maxLabel + 14
	if maxShort > 0 {
		e.width += maxShort + 24
	}
	if e.width < 180 {
		e.width = 180
	}
	h := float64(len(e.menu.items))*dropdownItemH + 8
	e.size = ctx.Constraints.Constrain(types.Size{Width: e.width, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *dropdownMenuElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height

	// 配色：默认 el 浅色；若设了全局菜单主题(SetMenuTheme)则用之（深色应用统一）。
	bgCol, borderCol, hoverCol := elSurface(), elBorderLight(), elFill()
	textNormal, mutedCol := elTextRegular(), elPlaceholder()
	themed := menuThemeBG != (types.Color{})
	if themed {
		bgCol = menuThemeBG
	}
	if menuThemeBorder != (types.Color{}) {
		borderCol = menuThemeBorder
	}
	if menuThemeHover != (types.Color{}) {
		hoverCol = menuThemeHover
	}
	if menuThemeText != (types.Color{}) {
		textNormal = menuThemeText
	}
	if menuThemeMuted != (types.Color{}) {
		mutedCol = menuThemeMuted
	}

	// 阴影 + 底 + 边框
	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 22)
	cvs.DrawRoundedRect(pos.X, pos.Y+2, W, H, 4, shadow)
	bg := paint.DefaultPaint()
	bg.Color = bgCol
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = borderCol
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	font := canvas.DefaultFont()
	font.Size = 14
	for i, it := range e.menu.items {
		iy := pos.Y + 4 + float64(i)*dropdownItemH
		// 分隔线
		if it.Divided && i > 0 {
			dp := paint.DefaultStrokePaint()
			dp.Color = borderCol
			dp.StrokeWidth = 1
			cvs.DrawLine(pos.X, iy, pos.X+W, iy, dp)
		}
		// hover 高亮
		if i == e.hoverIdx && !it.Disabled {
			hp := paint.DefaultPaint()
			hp.Color = hoverCol
			cvs.DrawRect(pos.X+1, iy, W-2, dropdownItemH, hp)
		}
		// 文本
		color := textNormal
		if it.Disabled {
			color = mutedCol
		} else if i == e.hoverIdx {
			if themed {
				color = types.ColorWhite // 深色主题：hover 白字（高亮底上更清晰）
			} else {
				color = elPrimary()
			}
		}
		leftCol := 0.0
		for _, j := range e.menu.items { // 任一项有勾选/图标 → 预留左列，标签对齐
			if j.Checked || j.Icon != "" {
				leftCol = 18
				break
			}
		}
		if it.Checked { // 勾选：Lucide ✓ 向量（非字符）
			PaintLucide(cvs, "check", pos.X+12, iy+(dropdownItemH-13)/2, 13, 2, color)
		} else if it.Icon != "" {
			canvas.DrawTextAligned(cvs, it.Icon, types.Rect{X: pos.X + 12, Y: iy, Width: 16, Height: dropdownItemH}, font, color, canvas.HAlignLeft, canvas.VAlignMiddle)
		}
		textX := pos.X + 12 + leftCol
		tr := types.Rect{X: textX, Y: iy, Width: pos.X + W - 14 - textX, Height: dropdownItemH}
		canvas.DrawTextAligned(cvs, it.Label, tr, font, color, canvas.HAlignLeft, canvas.VAlignMiddle)
		if it.Shortcut != "" { // 右侧快捷键（灰、小）
			sf := canvas.DefaultFont()
			sf.Size = 11
			sw := canvas.MeasureTextGlobal(it.Shortcut, sf).Width
			canvas.DrawTextAligned(cvs, it.Shortcut, types.Rect{X: pos.X + W - 14 - sw, Y: iy, Width: sw, Height: dropdownItemH}, sf, mutedCol, canvas.HAlignLeft, canvas.VAlignMiddle)
		}
	}
}

func (e *dropdownMenuElement) itemAt(localY float64) int {
	idx := int((localY - 4) / dropdownItemH)
	if idx < 0 || idx >= len(e.menu.items) {
		return -1
	}
	return idx
}

func (e *dropdownMenuElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		e.hoverIdx = e.itemAt(me.Y - e.Offset().Y)
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hoverIdx = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		idx := e.itemAt(me.Y - e.Offset().Y)
		if idx >= 0 && !e.menu.items[idx].Disabled && e.menu.onSelect != nil {
			e.menu.onSelect(e.menu.items[idx].Command)
		}
		return true
	}
	return false
}

func (e *dropdownMenuElement) Update(newWidget Widget) {
	if nm, ok := newWidget.(*dropdownMenu); ok {
		e.menu = nm
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// DropdownMenuPreview 返回独立菜单 widget，仅供 demo/测试预览(正常由 Dropdown 点击弹出)。
func DropdownMenuPreview(items []DropdownItem) Widget {
	return &dropdownMenu{items: items}
}
