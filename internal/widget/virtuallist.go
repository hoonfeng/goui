package widget

import (
	"reflect"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── VirtualList 虚拟滚动地基 ───────────────────────────────────
// 只渲染可视区内的行(按 index 复用 Element)，支撑大数据(上万项)流畅滚动。
// SelectV2/TreeV2/InfiniteScroll 复用它。RenderItem(index) 按需产出每行 Widget。

type VirtualList struct {
	StatelessWidget
	ItemCount  int
	ItemHeight float64
	Height     float64
	Width      float64
	RenderItem func(index int) Widget
	OnReachEnd func() // 滚到接近底部时回调（供 InfiniteScroll 加载更多）
}

func NewVirtualList(count int, itemH float64, render func(int) Widget) *VirtualList {
	return &VirtualList{ItemCount: count, ItemHeight: itemH, Height: 280, RenderItem: render}
}
func (v *VirtualList) WithHeight(h float64) *VirtualList     { v.Height = h; return v }
func (v *VirtualList) WithWidth(w float64) *VirtualList      { v.Width = w; return v }
func (v *VirtualList) WithOnReachEnd(fn func()) *VirtualList { v.OnReachEnd = fn; return v }

func (v *VirtualList) CreateElement() Element {
	return &VirtualListElement{BaseElement: BaseElement{widget: v}, vl: v, itemEls: map[int]Element{}}
}

type VirtualListElement struct {
	BaseElement
	vl           *VirtualList
	scrollOffset float64
	maxScroll    float64
	itemEls      map[int]Element
}

// visibleRange 依据当前 scrollOffset 算可视区 [start,end)。
func (e *VirtualListElement) visibleRange() (int, int) {
	ih := e.vl.ItemHeight
	if ih <= 0 {
		ih = 30
	}
	start := int(e.scrollOffset / ih)
	end := int((e.scrollOffset+e.vl.Height)/ih) + 1
	if start < 0 {
		start = 0
	}
	if end > e.vl.ItemCount {
		end = e.vl.ItemCount
	}
	return start, end
}

func (e *VirtualListElement) Build() []Element {
	if e.itemEls == nil {
		e.itemEls = map[int]Element{}
	}
	start, end := e.visibleRange()
	used := map[int]bool{}
	var kids []Element
	for i := start; i < end; i++ {
		w := e.vl.RenderItem(i)
		if w == nil {
			continue
		}
		el, ok := e.itemEls[i]
		if ok && reflect.TypeOf(el.Widget()) == reflect.TypeOf(w) {
			el.Update(w)
		} else {
			if ok {
				el.Unmount()
			}
			el = CreateElementFor(w)
			el.Mount(e, i)
			e.itemEls[i] = el
		}
		used[i] = true
		kids = append(kids, el)
	}
	for k, el := range e.itemEls {
		if !used[k] {
			el.Unmount()
			delete(e.itemEls, k)
		}
	}
	e.children = kids
	return kids
}

func (e *VirtualListElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	vl := e.vl
	w := vl.Width
	if w <= 0 {
		w = ctx.Constraints.MaxWidth
		if w >= float64(1<<30) {
			w = 300
		}
	}
	e.maxScroll = max(0, float64(vl.ItemCount)*vl.ItemHeight-vl.Height)
	e.scrollOffset = clamp(e.scrollOffset, 0, e.maxScroll)
	for i, el := range e.itemEls {
		el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: w, MaxWidth: w, MinHeight: vl.ItemHeight, MaxHeight: vl.ItemHeight}})
		el.SetPosition(types.Point{X: 0, Y: float64(i)*vl.ItemHeight - e.scrollOffset})
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: vl.Height})
	return layout.LayoutResult{Size: e.size}
}

func (e *VirtualListElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	cvs.Save()
	cvs.ClipRect(pos.X, pos.Y, e.size.Width, e.size.Height)
	for _, el := range e.itemEls {
		el.Paint(cvs, offset)
	}
	cvs.Restore()

	// 滚动条
	if e.maxScroll > 0 {
		contentH := e.size.Height + e.maxScroll
		thumbH := e.size.Height * e.size.Height / contentH
		if thumbH < 24 {
			thumbH = 24
		}
		thumbY := pos.Y + (e.size.Height-thumbH)*(e.scrollOffset/e.maxScroll)
		tp := paint.DefaultPaint()
		tp.Color = types.ColorFromRGBA(144, 147, 153, 120)
		cvs.DrawRoundedRect(pos.X+e.size.Width-6, thumbY, 4, thumbH, 2, tp)
	}
}

func (e *VirtualListElement) HandleEvent(ev event.Event) bool {
	if ev.Type() != event.TypeMouseWheel {
		return false
	}
	me := ev.(*event.MouseEvent)
	e.scrollOffset = clamp(e.scrollOffset-me.DeltaY*scrollWheelStep, 0, e.maxScroll)
	// 接近底部 → 触发加载更多
	if e.vl.OnReachEnd != nil && e.scrollOffset >= e.maxScroll-e.vl.ItemHeight*2 {
		e.vl.OnReachEnd()
	}
	ev.StopPropagation()
	if OnNeedsLayout != nil {
		OnNeedsLayout() // 可视区变→重建可视行
	}
	return true
}

func (e *VirtualListElement) Update(newWidget Widget) {
	if nv, ok := newWidget.(*VirtualList); ok {
		e.vl = nv
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
