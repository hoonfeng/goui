package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── SelectV2 虚拟下拉（el-select-v2，复用 VirtualList）──────────
// 触发器(复刻 Select) + 浮层用 VirtualList 渲染大量选项(只画可视行)；
// 行 selectV2Row 自绘 label/hover/选中✓，点击回填并关闭。

type SelectV2 struct {
	StatelessWidget
	pseudoColors // CSS 伪类：交互态边框色覆盖（:hover/:focus）
	Options     []SelectOption
	Value       string
	Placeholder string
	OnChange    func(string)
	Width       float64
}

func NewSelectV2(options []SelectOption) *SelectV2 {
	return &SelectV2{Options: options, Width: 240}
}
func (s *SelectV2) WithValue(v string) *SelectV2          { s.Value = v; return s }
func (s *SelectV2) WithPlaceholder(p string) *SelectV2    { s.Placeholder = p; return s }
func (s *SelectV2) WithOnChange(fn func(string)) *SelectV2 { s.OnChange = fn; return s }
func (s *SelectV2) WithWidth(w float64) *SelectV2         { s.Width = w; return s }

func (s *SelectV2) selectedLabel() string {
	for _, o := range s.Options {
		if o.Value == s.Value {
			return o.Label
		}
	}
	return ""
}

func (s *SelectV2) CreateElement() Element {
	return &SelectV2Element{BaseElement: BaseElement{widget: s}, sv: s}
}

type SelectV2Element struct {
	BaseElement
	sv        *SelectV2
	overlayID int
	open      bool
	hovered   bool
}

func (e *SelectV2Element) Build() []Element { return nil }

func (e *SelectV2Element) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := e.sv.Width
	if w <= 0 {
		w = 240
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: 32})
	return layout.LayoutResult{Size: e.size}
}

func (e *SelectV2Element) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height
	border := e.sv.borderOr(elBorder())
	if e.hovered || e.open {
		if e.open {
			border = e.sv.focusBorderOr(elPrimary())
		} else {
			border = e.sv.hoverBorderOr(elPrimary())
		}
	}
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	label := e.sv.selectedLabel()
	tc := elTextRegular()
	if label == "" {
		label, tc = orI18n(e.sv.Placeholder, "el.select.placeholder"), elPlaceholder()
	}
	font := canvas.DefaultFont()
	font.Size = 14
	canvas.DrawTextAligned(cvs, label, types.Rect{X: pos.X + 12, Y: pos.Y, Width: W - 40, Height: H},
		font, tc, canvas.HAlignLeft, canvas.VAlignMiddle)

	arrow := paint.DefaultStrokePaint()
	arrow.Color = elPlaceholder()
	arrow.StrokeWidth = 1.4
	acx, acy := pos.X+W-16, pos.Y+H/2
	if e.open {
		cvs.DrawLine(acx-4, acy+2, acx, acy-2, arrow)
		cvs.DrawLine(acx, acy-2, acx+4, acy+2, arrow)
	} else {
		cvs.DrawLine(acx-4, acy-2, acx, acy+2, arrow)
		cvs.DrawLine(acx, acy+2, acx+4, acy-2, arrow)
	}
}

func (e *SelectV2Element) HandleEvent(ev event.Event) bool {
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

const selV2RowH = 34.0

func (e *SelectV2Element) show() {
	if e.open {
		return
	}
	off := e.Offset()
	sv := e.sv
	listH := float64(len(sv.Options)) * selV2RowH
	if listH > 280 {
		listH = 280
	}
	vl := NewVirtualList(len(sv.Options), selV2RowH, func(i int) Widget {
		opt := sv.Options[i]
		return &selectV2Row{label: opt.Label, selected: opt.Value == sv.Value, onClick: func() {
			sv.Value = opt.Value
			if sv.OnChange != nil {
				sv.OnChange(opt.Value)
			}
			e.hide()
			e.MarkNeedsPaint()
		}}
	}).WithHeight(listH).WithWidth(e.size.Width - 8)

	panel := Div(Style{
		Width: e.size.Width, Padding: types.EdgeInsets(4),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderColor:     types.ColorRef(228, 231, 237), BorderWidth: 1, BorderRadius: 4,
		Shadow: &paint.Shadow{Offset: types.Point{X: 0, Y: 2}, Blur: 12, Color: types.ColorFromRGBA(0, 0, 0, 22)},
	}, vl)

	e.overlayID = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: e.size.Width, Height: e.size.Height},
		Content:        panel,
		Placement:      PlacementBottomStart,
		Gap:            4,
		Transition:     "fade",
		OnClickOutside: func() { e.hide(); e.MarkNeedsPaint() },
	})
	e.open = true
}

func (e *SelectV2Element) hide() {
	if !e.open {
		return
	}
	HideOverlay(e.overlayID)
	e.open = false
}

func (e *SelectV2Element) Update(newWidget Widget) {
	if ns, ok := newWidget.(*SelectV2); ok {
		e.sv = ns
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── selectV2Row 虚拟下拉的一行 ─────────────────────────────────

type selectV2Row struct {
	StatelessWidget
	label    string
	selected bool
	onClick  func()
}

func (r *selectV2Row) CreateElement() Element {
	return &selectV2RowElement{BaseElement: BaseElement{widget: r}, row: r}
}

type selectV2RowElement struct {
	BaseElement
	row     *selectV2Row
	hovered bool
}

func (e *selectV2RowElement) Build() []Element { return nil }

func (e *selectV2RowElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(types.Size{Width: ctx.Constraints.MaxWidth, Height: selV2RowH})
	return layout.LayoutResult{Size: e.size}
}

func (e *selectV2RowElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W := e.size.Width
	if e.hovered {
		hp := paint.DefaultPaint()
		hp.Color = elFill()
		cvs.DrawRect(pos.X, pos.Y, W, selV2RowH, hp)
	}
	color := elTextRegular()
	f := canvas.DefaultFont()
	f.Size = 14
	if e.row.selected {
		color = elPrimary()
		f.Weight = canvas.FontWeightBold
	}
	canvas.DrawTextAligned(cvs, e.row.label, types.Rect{X: pos.X + 12, Y: pos.Y, Width: W - 36, Height: selV2RowH},
		f, color, canvas.HAlignLeft, canvas.VAlignMiddle)
	if e.row.selected {
		cp := paint.DefaultStrokePaint()
		cp.Color = elPrimary()
		cp.StrokeWidth = 1.6
		cx, cy := pos.X+W-20, pos.Y+selV2RowH/2
		cvs.DrawLine(cx, cy, cx+3, cy+3, cp)
		cvs.DrawLine(cx+3, cy+3, cx+8, cy-4, cp)
	}
}

func (e *selectV2RowElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter, event.TypeMouseMove:
		if !e.hovered {
			e.hovered = true
			e.MarkNeedsPaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if e.row.onClick != nil {
			e.row.onClick()
		}
		return true
	}
	return false
}

func (e *selectV2RowElement) Update(newWidget Widget) {
	if nr, ok := newWidget.(*selectV2Row); ok {
		e.row = nr
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
