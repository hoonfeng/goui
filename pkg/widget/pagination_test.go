package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
)

func layoutPagination(p *Pagination) *PaginationElement {
	el := p.CreateElement().(*PaginationElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 600, MaxHeight: 50}})
	return el
}

// TestPaginationPageCount 页数 = ceil(Total/PageSize)，至少 1。
func TestPaginationPageCount(t *testing.T) {
	if got := NewPagination(50, 10, 1).pageCount(); got != 5 {
		t.Errorf("50/10 pageCount=%d, want 5", got)
	}
	if got := NewPagination(55, 10, 1).pageCount(); got != 6 {
		t.Errorf("55/10 pageCount=%d, want 6(ceil)", got)
	}
	if got := NewPagination(0, 10, 1).pageCount(); got != 1 {
		t.Errorf("空数据 pageCount=%d, want 1", got)
	}
}

// TestPaginationGoTo 跳页 + clamp 边界 + 回调。
func TestPaginationGoTo(t *testing.T) {
	prev := OnNeedsLayout
	OnNeedsLayout = func() {}
	defer func() { OnNeedsLayout = prev }()

	changed := -1
	el := layoutPagination(NewPagination(50, 10, 2).WithOnChange(func(p int) { changed = p }))
	el.goTo(4)
	if el.pag.CurrentPage != 4 || changed != 4 {
		t.Errorf("goTo(4): page=%d changed=%d", el.pag.CurrentPage, changed)
	}
	el.goTo(99) // clamp 到 5
	if el.pag.CurrentPage != 5 {
		t.Errorf("clamp 上界: page=%d, want 5", el.pag.CurrentPage)
	}
	el.goTo(0) // clamp 到 1
	if el.pag.CurrentPage != 1 {
		t.Errorf("clamp 下界: page=%d, want 1", el.pag.CurrentPage)
	}
}

// TestPaginationPrevArrow 点 < 上一页。
func TestPaginationPrevArrow(t *testing.T) {
	prev := OnNeedsLayout
	OnNeedsLayout = func() {}
	defer func() { OnNeedsLayout = prev }()

	el := layoutPagination(NewPagination(50, 10, 3))
	c0 := el.cells[0] // < 上一页
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, c0.rect.X+c0.rect.Width/2, 16, event.ButtonLeft, event.ModNone))
	if el.pag.CurrentPage != 2 {
		t.Errorf("点 < 应到第 2 页, got %d", el.pag.CurrentPage)
	}
}
