package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// ─── Breadcrumb 面包屑（1:1 复刻 el-breadcrumb）──────────────────
// 路径项横排，分隔符连接；最后一项为当前(浅灰不可点)，前面项为链接(可点 + hover 变蓝)。

// BreadcrumbItem 一个路径项。
type BreadcrumbItem struct {
	Label   string
	OnClick func()
}

// Breadcrumb 面包屑导航。
type Breadcrumb struct {
	StatelessWidget
	Items     []BreadcrumbItem
	Separator string // 分隔符(默认 "/")
}

func NewBreadcrumb(items ...BreadcrumbItem) *Breadcrumb {
	return &Breadcrumb{Items: items, Separator: "/"}
}
func (b *Breadcrumb) WithSeparator(s string) *Breadcrumb { b.Separator = s; return b }

func (b *Breadcrumb) sep() string {
	if b.Separator == "" {
		return "/"
	}
	return b.Separator
}

func (b *Breadcrumb) CreateElement() Element {
	return &BreadcrumbElement{BaseElement: BaseElement{widget: b}, bc: b, hoverIdx: -1}
}

type BreadcrumbElement struct {
	BaseElement
	bc        *Breadcrumb
	itemRects []types.Rect // 每项矩形(相对本元素左上)
	hoverIdx  int
}

const breadcrumbH = 22.0

func (e *BreadcrumbElement) Build() []Element { return nil }

func (e *BreadcrumbElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	font := canvas.DefaultFont()
	font.Size = 14
	sepW := canvas.MeasureTextGlobal(e.bc.sep(), font).Width + 16 // 分隔符 + 左右各 8
	x := 0.0
	e.itemRects = e.itemRects[:0]
	for i, item := range e.bc.Items {
		w := canvas.MeasureTextGlobal(item.Label, font).Width
		e.itemRects = append(e.itemRects, types.Rect{X: x, Y: 0, Width: w, Height: breadcrumbH})
		x += w
		if i < len(e.bc.Items)-1 {
			x += sepW
		}
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: x, Height: breadcrumbH})
	return layout.LayoutResult{Size: e.size}
}

func (e *BreadcrumbElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	font := canvas.DefaultFont()
	font.Size = 14
	sepW := canvas.MeasureTextGlobal(e.bc.sep(), font).Width + 16
	n := len(e.bc.Items)

	for i, item := range e.bc.Items {
		r := e.itemRects[i]
		last := i == n-1
		color := elTextRegular() // 链接深灰
		switch {
		case last:
			color = elPlaceholder() // 当前浅灰
		case i == e.hoverIdx:
			color = elPrimary() // hover 蓝
		}
		tr := types.Rect{X: pos.X + r.X, Y: pos.Y, Width: r.Width, Height: breadcrumbH}
		canvas.DrawTextAligned(cvs, item.Label, tr, font, color, canvas.HAlignLeft, canvas.VAlignMiddle)

		if !last {
			sr := types.Rect{X: pos.X + r.X + r.Width, Y: pos.Y, Width: sepW, Height: breadcrumbH}
			canvas.DrawTextAligned(cvs, e.bc.sep(), sr, font, elPlaceholder(), canvas.HAlignCenter, canvas.VAlignMiddle)
		}
	}
}

// itemAtX 返回局部 x 命中的可点击项索引(最后一项当前态不可点，返回 -1)。
func (e *BreadcrumbElement) itemAtX(localX float64) int {
	for i, r := range e.itemRects {
		if i < len(e.bc.Items)-1 && localX >= r.X && localX < r.X+r.Width {
			return i
		}
	}
	return -1
}

func (e *BreadcrumbElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	localX := me.X - e.Offset().X
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		e.hoverIdx = e.itemAtX(localX)
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hoverIdx = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if idx := e.itemAtX(localX); idx >= 0 {
			if e.bc.Items[idx].OnClick != nil {
				e.bc.Items[idx].OnClick()
			}
			return true
		}
	}
	return false
}

func (e *BreadcrumbElement) Update(newWidget Widget) {
	if nb, ok := newWidget.(*Breadcrumb); ok {
		e.bc = nb
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
