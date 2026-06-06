package widget

import (
	"strings"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Cascader 级联选择器（1:1 复刻 el-cascader 单选核心）──────────
// 触发器(显示选中路径 "省 / 市 / 区") + 点击弹出多列联动浮层(复用 Overlay 地基)。
// 点某项：有子节点 → 右侧展开下一列(列数变→relayout)；叶子 → 回填整条路径并关闭。
// 当前为单选；多选/可搜索/动态加载为进阶。

// CascaderOption 级联树的一个节点。
type CascaderOption struct {
	Label    string
	Value    string
	Children []CascaderOption
	Icon     string // 可选左侧图标字符(任意 Unicode/emoji)
}

// Cascader 级联选择器。
type Cascader struct {
	StatelessWidget
	pseudoColors // CSS 伪类：交互态边框色覆盖（:hover/:focus）
	Options     []CascaderOption
	Value       []string // 选中路径的 value 序列
	Placeholder string
	OnChange    func([]string)
	Width       float64
	Disabled    bool
}

func NewCascader(options []CascaderOption) *Cascader {
	return &Cascader{Options: options, Width: 280} // Placeholder 空→渲染取 i18n 默认
}

func (c *Cascader) WithValue(v []string) *Cascader          { c.Value = v; return c }
func (c *Cascader) WithPlaceholder(p string) *Cascader      { c.Placeholder = p; return c }
func (c *Cascader) WithOnChange(fn func([]string)) *Cascader { c.OnChange = fn; return c }
func (c *Cascader) WithWidth(w float64) *Cascader           { c.Width = w; return c }
func (c *Cascader) WithDisabled(b bool) *Cascader           { c.Disabled = b; return c }

// selectedLabels 沿 Value 路径解析出各级 Label。
func (c *Cascader) selectedLabels() []string {
	var labels []string
	node := c.Options
	for _, v := range c.Value {
		found := false
		for _, o := range node {
			if o.Value == v {
				labels = append(labels, o.Label)
				node = o.Children
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return labels
}

func (c *Cascader) CreateElement() Element {
	return &CascaderElement{BaseElement: BaseElement{widget: c}, cas: c}
}

// CascaderElement 触发器运行时 Element。
type CascaderElement struct {
	BaseElement
	cas       *Cascader
	overlayID int
	open      bool
	hovered   bool
}

func (e *CascaderElement) Build() []Element { return nil }

func (e *CascaderElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := e.cas.Width
	if w <= 0 {
		w = 280
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: 32})
	return layout.LayoutResult{Size: e.size}
}

func (e *CascaderElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	c := e.cas
	W, H := e.size.Width, e.size.Height

	border := c.borderOr(elBorder())
	if (e.hovered || e.open) && !c.Disabled {
		if e.open {
			border = c.focusBorderOr(elPrimary())
		} else {
			border = c.hoverBorderOr(elPrimary())
		}
	}
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	if c.Disabled {
		bg.Color = elFill()
	}
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	// 文本：选中路径 "省 / 市 / 区" 或 placeholder
	label := strings.Join(c.selectedLabels(), " / ")
	textColor := elTextRegular()
	if label == "" {
		label = orI18n(c.Placeholder, "el.cascader.placeholder")
		textColor = elPlaceholder()
	}
	if c.Disabled {
		textColor = elPlaceholder()
	}
	font := canvas.DefaultFont()
	font.Size = 14
	tr := types.Rect{X: pos.X + 12, Y: pos.Y, Width: W - 12 - 28, Height: H}
	canvas.DrawTextAligned(cvs, label, tr, font, textColor, canvas.HAlignLeft, canvas.VAlignMiddle)

	// 下拉箭头
	arrow := paint.DefaultStrokePaint()
	arrow.Color = elPlaceholder()
	arrow.StrokeWidth = 1.4
	acx := pos.X + W - 16
	acy := pos.Y + H/2
	if e.open {
		cvs.DrawLine(acx-4, acy+2, acx, acy-2, arrow)
		cvs.DrawLine(acx, acy-2, acx+4, acy+2, arrow)
	} else {
		cvs.DrawLine(acx-4, acy-2, acx, acy+2, arrow)
		cvs.DrawLine(acx, acy+2, acx+4, acy-2, arrow)
	}
}

func (e *CascaderElement) HandleEvent(ev event.Event) bool {
	if e.cas.Disabled {
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

func (e *CascaderElement) show() {
	if e.open {
		return
	}
	off := e.Offset()
	panel := &cascaderPanel{
		options: e.cas.Options,
		value:   e.cas.Value,
		onSelect: func(values, labels []string) {
			e.cas.Value = values
			if e.cas.OnChange != nil {
				e.cas.OnChange(values)
			}
			e.hide()
			e.MarkNeedsPaint()
		},
	}
	entry := &OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: e.size.Width, Height: e.size.Height},
		Content:        panel,
		Placement:      PlacementBottomStart,
		Gap:            4,
		Transition:     "fade",
		OnClickOutside: func() { e.hide(); e.MarkNeedsPaint() },
	}
	e.overlayID = ShowOverlay(entry)
	e.open = true
}

func (e *CascaderElement) hide() {
	if !e.open {
		return
	}
	HideOverlay(e.overlayID)
	e.open = false
}

func (e *CascaderElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*Cascader); ok {
		e.cas = nc
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── cascaderPanel 多列联动浮层 ─────────────────────────────────

type cascaderPanel struct {
	StatelessWidget
	options  []CascaderOption
	value    []string
	onSelect func(values, labels []string)
}

func (p *cascaderPanel) CreateElement() Element {
	return &cascaderPanelElement{BaseElement: BaseElement{widget: p}, panel: p, hoverCol: -1, hoverIdx: -1}
}

const (
	cascaderColW  = 160.0
	cascaderItemH = 34.0
	cascaderPadV  = 6.0
)

type cascaderPanelElement struct {
	BaseElement
	panel      *cascaderPanel
	activePath []int // 各列当前选中的索引
	inited     bool
	hoverCol   int
	hoverIdx   int
}

func (e *cascaderPanelElement) Build() []Element { return nil }

// ensureInit 首次按 value 还原展开路径。
func (e *cascaderPanelElement) ensureInit() {
	if e.inited {
		return
	}
	e.inited = true
	node := e.panel.options
	for _, v := range e.panel.value {
		idx := -1
		for i, o := range node {
			if o.Value == v {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		e.activePath = append(e.activePath, idx)
		node = node[idx].Children
		if len(node) == 0 {
			break
		}
	}
}

// columns 依据 activePath 推出当前要显示的各列选项。
func (e *cascaderPanelElement) columns() [][]CascaderOption {
	cols := [][]CascaderOption{e.panel.options}
	node := e.panel.options
	for _, idx := range e.activePath {
		if idx < 0 || idx >= len(node) || len(node[idx].Children) == 0 {
			break
		}
		node = node[idx].Children
		cols = append(cols, node)
	}
	return cols
}

func (e *cascaderPanelElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.ensureInit()
	cols := e.columns()
	maxRows := 1
	for _, c := range cols {
		if len(c) > maxRows {
			maxRows = len(c)
		}
	}
	w := float64(len(cols)) * cascaderColW
	h := float64(maxRows)*cascaderItemH + cascaderPadV*2
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *cascaderPanelElement) Paint(cvs canvas.Canvas, offset types.Point) {
	e.ensureInit()
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height

	// 阴影 + 白底 + 边框
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

	font := canvas.DefaultFont()
	font.Size = 14
	for ci, col := range e.columns() {
		colX := pos.X + float64(ci)*cascaderColW
		// 列间分隔竖线
		if ci > 0 {
			sp := paint.DefaultStrokePaint()
			sp.Color = elBorderLighter()
			sp.StrokeWidth = 1
			cvs.DrawLine(colX, pos.Y+4, colX, pos.Y+H-4, sp)
		}
		for ri, opt := range col {
			iy := pos.Y + cascaderPadV + float64(ri)*cascaderItemH
			active := ci < len(e.activePath) && e.activePath[ci] == ri
			hover := e.hoverCol == ci && e.hoverIdx == ri

			if active {
				ap := paint.DefaultPaint()
				ap.Color = types.ColorFromRGB(236, 245, 255)
				cvs.DrawRect(colX+1, iy, cascaderColW-2, cascaderItemH, ap)
			} else if hover {
				hp := paint.DefaultPaint()
				hp.Color = elFill()
				cvs.DrawRect(colX+1, iy, cascaderColW-2, cascaderItemH, hp)
			}

			color := elTextRegular()
			f := font
			if active {
				color = elPrimary()
				f.Weight = canvas.FontWeightBold
			}
			textX := colX + 14
			if opt.Icon != "" {
				canvas.DrawTextAligned(cvs, opt.Icon, types.Rect{X: colX + 12, Y: iy, Width: 18, Height: cascaderItemH}, f, color, canvas.HAlignLeft, canvas.VAlignMiddle)
				textX += 20
			}
			tr := types.Rect{X: textX, Y: iy, Width: colX + cascaderColW - 22 - textX, Height: cascaderItemH}
			canvas.DrawTextAligned(cvs, opt.Label, tr, f, color, canvas.HAlignLeft, canvas.VAlignMiddle)

			// 有子节点 → 右箭头 ›
			if len(opt.Children) > 0 {
				arr := paint.DefaultStrokePaint()
				arr.Color = elPlaceholder()
				if active {
					arr.Color = elPrimary()
				}
				arr.StrokeWidth = 1.3
				ax := colX + cascaderColW - 16
				ay := iy + cascaderItemH/2
				cvs.DrawLine(ax, ay-3, ax+3, ay, arr)
				cvs.DrawLine(ax+3, ay, ax, ay+3, arr)
			}
		}
	}
}

// cellAt 返回局部坐标命中的 (列, 行)，落空返回 (-1,-1)。
func (e *cascaderPanelElement) cellAt(lx, ly float64) (int, int) {
	cols := e.columns()
	col := int(lx / cascaderColW)
	idx := int((ly - cascaderPadV) / cascaderItemH)
	if col < 0 || col >= len(cols) {
		return -1, -1
	}
	if idx < 0 || idx >= len(cols[col]) {
		return -1, -1
	}
	return col, idx
}

func (e *cascaderPanelElement) clickAt(col, idx int) {
	cols := e.columns()
	if col < 0 || col >= len(cols) || idx < 0 || idx >= len(cols[col]) {
		return
	}
	opt := cols[col][idx]
	// 截断到本列再追加（三索引切片限 cap，避免污染原底层数组）
	e.activePath = append(e.activePath[:col:col], idx)
	if len(opt.Children) > 0 {
		// 展开下一列：列数/宽度变 → relayout
		if OnNeedsLayout != nil {
			OnNeedsLayout()
		}
		e.MarkNeedsPaint()
	} else {
		// 叶子 → 回填整条路径并关闭
		values, labels := e.pathFrom(e.activePath)
		if e.panel.onSelect != nil {
			e.panel.onSelect(values, labels)
		}
	}
}

// pathFrom 沿索引路径收集各级 value/label。
func (e *cascaderPanelElement) pathFrom(path []int) (values, labels []string) {
	node := e.panel.options
	for _, idx := range path {
		if idx < 0 || idx >= len(node) {
			break
		}
		values = append(values, node[idx].Value)
		labels = append(labels, node[idx].Label)
		node = node[idx].Children
	}
	return values, labels
}

func (e *cascaderPanelElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	lx, ly := me.X-pos.X, me.Y-pos.Y
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		col, idx := e.cellAt(lx, ly)
		if col != e.hoverCol || idx != e.hoverIdx {
			e.hoverCol, e.hoverIdx = col, idx
			e.MarkNeedsPaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hoverCol, e.hoverIdx = -1, -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		col, idx := e.cellAt(lx, ly)
		if col >= 0 {
			e.clickAt(col, idx)
		}
		return true
	}
	return false
}

func (e *cascaderPanelElement) Update(newWidget Widget) {
	if np, ok := newWidget.(*cascaderPanel); ok {
		e.panel = np
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// CascaderPanelPreview 返回独立的多列面板 widget，仅供 demo/测试预览外观
// （正常使用时面板由 CascaderElement 点击弹出、挂在 Overlay 顶层）。
func CascaderPanelPreview(options []CascaderOption, value []string) Widget {
	return &cascaderPanel{options: options, value: value}
}
