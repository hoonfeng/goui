package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Pagination 分页（1:1 复刻 el-pagination）────────────────────
// < 上一页 + 页码(当前高亮，多页省略号) + 下一页 >。点击切换页。

type Pagination struct {
	StatelessWidget
	Total       int // 总条数
	PageSize    int // 每页条数
	CurrentPage int // 当前页(1-indexed)
	OnChange    func(int)
}

func NewPagination(total, pageSize, current int) *Pagination {
	if pageSize <= 0 {
		pageSize = 10
	}
	if current <= 0 {
		current = 1
	}
	return &Pagination{Total: total, PageSize: pageSize, CurrentPage: current}
}

func (p *Pagination) WithOnChange(fn func(int)) *Pagination { p.OnChange = fn; return p }

func (p *Pagination) pageCount() int {
	if p.PageSize <= 0 {
		return 1
	}
	n := (p.Total + p.PageSize - 1) / p.PageSize
	if n < 1 {
		n = 1
	}
	return n
}

const (
	pageItemW = 34.0
	pageItemH = 32.0
)

func (p *Pagination) CreateElement() Element {
	return &PaginationElement{BaseElement: BaseElement{widget: p}, pag: p, hoverIdx: -1}
}

// pageCell 一个可点单元：page>0 页码，-1 省略号，-2 上一页，-3 下一页。
type pageCell struct {
	rect types.Rect
	page int
}

type PaginationElement struct {
	BaseElement
	pag      *Pagination
	cells    []pageCell
	hoverIdx int
}

func (e *PaginationElement) Build() []Element { return nil }

// buildCells 计算 < / 页码(含省略号) / > 的布局。
func (e *PaginationElement) buildCells() {
	e.cells = e.cells[:0]
	pc := e.pag.pageCount()
	cur := e.pag.CurrentPage
	x := 0.0
	add := func(page int) {
		e.cells = append(e.cells, pageCell{rect: types.Rect{X: x, Y: 0, Width: pageItemW, Height: pageItemH}, page: page})
		x += pageItemW
	}
	add(-2) // <
	add(1)
	if pc <= 7 {
		for i := 2; i <= pc; i++ {
			add(i)
		}
	} else {
		if cur > 4 {
			add(-1)
		}
		start := cur - 2
		if start < 2 {
			start = 2
		}
		end := cur + 2
		if end > pc-1 {
			end = pc - 1
		}
		for i := start; i <= end; i++ {
			add(i)
		}
		if cur < pc-3 {
			add(-1)
		}
		add(pc)
	}
	add(-3) // >
}

func (e *PaginationElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.buildCells()
	w := 0.0
	if n := len(e.cells); n > 0 {
		last := e.cells[n-1]
		w = last.rect.X + last.rect.Width
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: pageItemH})
	return layout.LayoutResult{Size: e.size}
}

func (e *PaginationElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	pc := e.pag.pageCount()
	cur := e.pag.CurrentPage
	font := canvas.DefaultFont()
	font.Size = 14

	blue := elPrimary()
	normal := elTextRegular()
	muted := elPlaceholder()

	for idx, cell := range e.cells {
		cr := types.Rect{X: pos.X + cell.rect.X, Y: pos.Y, Width: cell.rect.Width, Height: pageItemH}
		hovered := idx == e.hoverIdx

		switch {
		case cell.page > 0:
			if cell.page == cur {
				bp := paint.DefaultPaint()
				bp.Color = blue
				cvs.DrawRoundedRect(cr.X+2, cr.Y+2, cr.Width-4, cr.Height-4, 4, bp)
				canvas.DrawTextAligned(cvs, itoa(cell.page), cr, font, types.ColorWhite, canvas.HAlignCenter, canvas.VAlignMiddle)
			} else {
				color := normal
				if hovered {
					hp := paint.DefaultPaint()
					hp.Color = elFill()
					cvs.DrawRoundedRect(cr.X+2, cr.Y+2, cr.Width-4, cr.Height-4, 4, hp)
					color = blue
				}
				canvas.DrawTextAligned(cvs, itoa(cell.page), cr, font, color, canvas.HAlignCenter, canvas.VAlignMiddle)
			}
		case cell.page == -1:
			canvas.DrawTextAligned(cvs, "···", cr, font, muted, canvas.HAlignCenter, canvas.VAlignMiddle)
		case cell.page == -2: // <
			color := normal
			if cur <= 1 {
				color = muted
			} else if hovered {
				color = blue
			}
			e.paintArrow(cvs, cr, true, color)
		case cell.page == -3: // >
			color := normal
			if cur >= pc {
				color = muted
			} else if hovered {
				color = blue
			}
			e.paintArrow(cvs, cr, false, color)
		}
	}
}

// paintArrow 画 < (left=true) 或 > 箭头。
func (e *PaginationElement) paintArrow(cvs canvas.Canvas, cr types.Rect, left bool, color types.Color) {
	ap := paint.DefaultStrokePaint()
	ap.Color = color
	ap.StrokeWidth = 1.5
	cx := cr.X + cr.Width/2
	cy := cr.Y + cr.Height/2
	if left {
		cvs.DrawLine(cx+3, cy-5, cx-3, cy, ap)
		cvs.DrawLine(cx-3, cy, cx+3, cy+5, ap)
	} else {
		cvs.DrawLine(cx-3, cy-5, cx+3, cy, ap)
		cvs.DrawLine(cx+3, cy, cx-3, cy+5, ap)
	}
}

// goTo 切换到指定页(clamp + 回调)。
func (e *PaginationElement) goTo(page int) {
	pc := e.pag.pageCount()
	if page < 1 {
		page = 1
	}
	if page > pc {
		page = pc
	}
	if page == e.pag.CurrentPage {
		return
	}
	e.pag.CurrentPage = page
	if e.pag.OnChange != nil {
		e.pag.OnChange(page)
	}
	if OnNeedsLayout != nil {
		OnNeedsLayout() // 页码集合可能变(省略号位置)
	}
}

func (e *PaginationElement) cellAt(localX float64) int {
	for i, c := range e.cells {
		if localX >= c.rect.X && localX < c.rect.X+c.rect.Width {
			return i
		}
	}
	return -1
}

func (e *PaginationElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	localX := me.X - e.Offset().X
	idx := e.cellAt(localX)
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		e.hoverIdx = idx
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hoverIdx = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if idx >= 0 {
			switch p := e.cells[idx].page; {
			case p > 0:
				e.goTo(p)
			case p == -2:
				e.goTo(e.pag.CurrentPage - 1)
			case p == -3:
				e.goTo(e.pag.CurrentPage + 1)
			}
			return true
		}
	}
	return false
}

func (e *PaginationElement) Update(newWidget Widget) {
	if np, ok := newWidget.(*Pagination); ok {
		e.pag = np
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
