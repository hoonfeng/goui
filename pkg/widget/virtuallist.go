package widget

import (
	"reflect"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── VirtualList 虚拟滚动列表 ─────────────────────────────────────
// 只渲染可视区内的行(按 index 复用 Element)，支撑大数据(上万项)流畅滚动。
// SelectV2/TreeV2/InfiniteScroll 复用它。RenderItem(index) 按需产出每行 Widget。
//
// 增强功能：
//   - Height=0 时从父布局约束撑满（ScrollView/Splitter 内自动适配）
//   - Overscan 缓冲避免滚动闪白
//   - 滚动条可拖拽
type VirtualList struct {
	StatelessWidget
	ItemCount  int
	ItemHeight float64
	Height     float64                 // 显式高度(0=从父布局撑满)
	Width      float64                 // 显式宽度(0=从父布局撑满)
	Overscan   int                     // 视口外额外渲染行数(默认5)
	RenderItem func(index int) Widget  // 按索引创建列表项控件
	OnReachEnd func()                  // 滚到接近底部时回调（供 InfiniteScroll 加载更多）
	OnScroll   func(scrollOffset float64) // 滚动偏移回调
}

// NewVirtualList 创建固定高度的虚拟列表（兼容旧 API，Height=280）。
func NewVirtualList(count int, itemH float64, render func(int) Widget) *VirtualList {
	os := 5
	return &VirtualList{
		ItemCount: count, ItemHeight: itemH, Height: 280, Overscan: os,
		RenderItem: render,
	}
}

// WithHeight 设置显式高。
func (v *VirtualList) WithHeight(h float64) *VirtualList { v.Height = h; return v }

// WithWidth 设置显式宽。
func (v *VirtualList) WithWidth(w float64) *VirtualList { v.Width = w; return v }

// WithOverscan 设置 overscan 缓冲行数。
func (v *VirtualList) WithOverscan(n int) *VirtualList { v.Overscan = n; return v }

// WithOnReachEnd 注册滚动到底部回调。
func (v *VirtualList) WithOnReachEnd(fn func()) *VirtualList { v.OnReachEnd = fn; return v }

// WithOnScroll 注册滚动偏移回调。
func (v *VirtualList) WithOnScroll(fn func(float64)) *VirtualList { v.OnScroll = fn; return v }

// CreateElement 创建 VirtualListElement
func (v *VirtualList) CreateElement() Element {
	os := v.Overscan
	if os <= 0 {
		os = 5
	}
	return &VirtualListElement{
		BaseElement: BaseElement{widget: v},
		vl:          v,
		overscan:    os,
		itemEls:     map[int]Element{},
	}
}

// VirtualListElement VirtualList 的运行时 Element
type VirtualListElement struct {
	BaseElement
	vl           *VirtualList
	scrollOffset float64
	maxScroll    float64
	viewportH    float64 // 实际视口高度（从 Layout 约束获取）
	overscan     int
	itemEls      map[int]Element // index→Element 缓存

	// 滚动条拖拽状态
	draggingBar      bool
	dragStartMouseY  float64
	dragStartScrollY float64
}

// ── 公开接口 ──────────────────────────────────────────────

// ScrollOffset 返回当前垂直滚动偏移
func (e *VirtualListElement) ScrollOffset() float64 { return e.scrollOffset }

// ScrollTo 跳到指定偏移
func (e *VirtualListElement) ScrollTo(offset float64) {
	maxScr := vlMax(0, e.maxScroll)
	e.scrollOffset = vlClamp(offset, 0, maxScr)
	e.rebuildVisible()
	e.fireScroll()
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// visibleRange 依据当前 scrollOffset 算可视区 [start,end)。
func (e *VirtualListElement) visibleRange() (int, int) {
	ih := e.vl.ItemHeight
	if ih <= 0 {
		ih = 30
	}
	start := int(e.scrollOffset / ih)
	end := int((e.scrollOffset+e.viewportH)/ih) + 1

	// overscan
	start -= e.overscan
	end += e.overscan
	if start < 0 {
		start = 0
	}
	if end > e.vl.ItemCount {
		end = e.vl.ItemCount
	}
	return start, end
}

// Build 构建可见项（被 pipeline.buildTree 递归调用）
func (e *VirtualListElement) Build() []Element {
	if e.viewportH <= 0 {
		e.viewportH = 600 // 首帧默认值（Layout 后会修正）
	}
	e.rebuildVisible()
	return e.children
}

// rebuildVisible 重建可见项列表
func (e *VirtualListElement) rebuildVisible() {
	if e.vl.ItemCount <= 0 || e.vl.RenderItem == nil {
		e.removeAll()
		return
	}
	start, end := e.visibleRange()
	if start >= end {
		e.removeAll()
		return
	}
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
	e.repositionAll()
}

// repositionAll 重新定位所有可见项
func (e *VirtualListElement) repositionAll() {
	ih := e.vl.ItemHeight
	if ih <= 0 {
		ih = 30
	}
	for idx, el := range e.itemEls {
		el.SetPosition(types.Point{X: 0, Y: float64(idx)*ih - e.scrollOffset})
	}
}

// removeAll 移除所有项
func (e *VirtualListElement) removeAll() {
	for _, el := range e.itemEls {
		el.Unmount()
	}
	e.itemEls = map[int]Element{}
	e.children = nil
}

// Layout 执行布局
func (e *VirtualListElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	vl := e.vl

	// 宽度
	w := vl.Width
	if w <= 0 {
		w = ctx.Constraints.MaxWidth
		if w >= float64(1<<30) {
			w = 300
		}
	}

	// 视口高度（优先从父布局约束获取，其次用显式 Height）
	vh := vl.Height
	if vh <= 0 {
		vh = ctx.Constraints.MaxHeight
		if vh >= float64(1<<30) {
			vh = 600
		}
	}

	e.viewportH = vh
	ih := vl.ItemHeight
	if ih <= 0 {
		ih = 30
	}

	e.maxScroll = vlMax(0, float64(vl.ItemCount)*ih-e.viewportH)
	e.scrollOffset = vlClamp(e.scrollOffset, 0, e.maxScroll)

	// 布局可见项
	childConstraints := layout.BoxConstraints{
		MinWidth:  w,
		MaxWidth:  w,
		MinHeight: 0,
		MaxHeight: ih,
	}
	for _, el := range e.itemEls {
		el.Layout(&layout.LayoutContext{Constraints: childConstraints})
	}

	// 定位可见项
	for idx, el := range e.itemEls {
		el.SetPosition(types.Point{X: 0, Y: float64(idx)*ih - e.scrollOffset})
	}

	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: vh})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制（裁剪到视口 + 滚动条）
func (e *VirtualListElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	ih := e.vl.ItemHeight
	if ih <= 0 {
		ih = 30
	}

	// 裁剪到视口
	cvs.Save()
	cvs.ClipRect(pos.X, pos.Y, e.size.Width, e.size.Height)

	// 只绘制可见项
	for _, el := range e.itemEls {
		el.Paint(cvs, offset)
	}

	cvs.Restore()

	// 滚动条（内容超出视口时才显示）
	if e.maxScroll > 0 {
		e.paintScrollbar(cvs, pos)
	}
}

// paintScrollbar 绘制滚动条
func (e *VirtualListElement) paintScrollbar(cvs canvas.Canvas, pos types.Point) {
	contentH := e.size.Height + e.maxScroll
	thumbH := e.size.Height * e.size.Height / contentH
	if thumbH < 30 {
		thumbH = 30
	}
	thumbY := pos.Y + (e.size.Height-thumbH)*(e.scrollOffset/e.maxScroll)
	tp := paint.DefaultPaint()
	tp.Color = types.ColorFromRGBA(144, 147, 153, 120)
	cvs.DrawRoundedRect(pos.X+e.size.Width-6, thumbY, 4, thumbH, 2, tp)
}

// HandleEvent 处理滚轮滚动 + 滚动条拖动
func (e *VirtualListElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseWheel:
		me := ev.(*event.MouseEvent)
		dy := me.DeltaY * vlWheelStep
		e.scrollOffset = vlClamp(e.scrollOffset-dy, 0, e.maxScroll)
		e.rebuildVisible()
		e.fireScroll()
		// 接近底部 → 触发加载更多
		if e.vl.OnReachEnd != nil && e.scrollOffset >= e.maxScroll-e.vl.ItemHeight*2 {
			e.vl.OnReachEnd()
		}
		ev.StopPropagation()
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true

	case event.TypeMouseDown:
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			return false
		}
		if r, okk := e.thumbRect(); okk {
			if me.X >= r.X-6 && me.X < r.X+r.Width+2 && me.Y >= r.Y-2 && me.Y < r.Y+r.Height+2 {
				e.draggingBar = true
				e.dragStartMouseY = me.Y
				e.dragStartScrollY = e.scrollOffset
				if RequestPointerCapture != nil {
					RequestPointerCapture(e)
				}
				ev.StopPropagation()
				return true
			}
		}
		return false

	case event.TypeMouseMove:
		if e.draggingBar {
			if me, ok := ev.(*event.MouseEvent); ok {
				e.dragBarTo(me.Y)
			}
			return true
		}
		return false

	case event.TypeMouseUp:
		if e.draggingBar {
			e.draggingBar = false
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		return false

	case event.TypeMouseLeave:
		if !e.draggingBar {
			e.draggingBar = false
		}
		return false
	}
	return false
}

// Update 更新 widget 配置
func (e *VirtualListElement) Update(newWidget Widget) {
	if nv, ok := newWidget.(*VirtualList); ok {
		e.vl = nv
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// thumbRect 返回竖向滚动条滑块的全局矩形
func (e *VirtualListElement) thumbRect() (types.Rect, bool) {
	if e.maxScroll <= 0 {
		return types.Rect{}, false
	}
	pos := e.Offset()
	contentH := e.size.Height + e.maxScroll
	thumbH := e.size.Height * e.size.Height / contentH
	if thumbH < 30 {
		thumbH = 30
	}
	thumbY := pos.Y + (e.size.Height-thumbH)*(e.scrollOffset/e.maxScroll)
	bx := pos.X + e.size.Width - 8
	return types.Rect{X: bx, Y: thumbY, Width: 6, Height: thumbH}, true
}

// dragBarTo 按鼠标 Y 拖动滑块
func (e *VirtualListElement) dragBarTo(mouseY float64) {
	contentH := e.size.Height + e.maxScroll
	thumbH := e.size.Height * e.size.Height / contentH
	if thumbH < 30 {
		thumbH = 30
	}
	travel := e.size.Height - thumbH
	if travel <= 0 {
		return
	}
	delta := (mouseY - e.dragStartMouseY) * e.maxScroll / travel
	e.scrollOffset = vlClamp(e.dragStartScrollY+delta, 0, e.maxScroll)
	e.repositionAll()
	e.fireScroll()
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// fireScroll 触发滚动回调
func (e *VirtualListElement) fireScroll() {
	if e.vl.OnScroll != nil {
		e.vl.OnScroll(e.scrollOffset)
	}
}

// ── 工具函数 ──────────────────────────────────────────────

// vlWheelStep 每个滚轮格滚动的像素（复用 scroll.go 的 scrollWheelStep）
const vlWheelStep = 50.0

func vlMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func vlClamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
