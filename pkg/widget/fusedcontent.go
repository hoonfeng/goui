// FusedContent — 内容分段虚加载容器。
//
// 设计意图：
//   聊天面板中，一条 Agent 消息可能包含大量子内容段（思考块 × N + 工具调用 × N +
//   正文 chunk × N）。FusedContent 像 VirtualList 一样只对「可见」的段创建 Widget 树，
//   超出视口的段只占高度，不产生 Widget 节点。
//
//   与 VirtualList 的区别：
//     - FusedContent 本身不滚动；它通过 ViewportProvider 接口从外层 VirtualList
//       获知当前视口信息（scrollOffset + viewportH）
//     - FusedContent 产出自己的总高度（所有 segment 高度之和），让父容器正确布局
//     - 内部的 segment Element 由槽池管理（首次创建后永不销毁），
//       滚动时只改变映射索引，不触发 Mount/Unmount
//
// 使用模式（chat.go）：
//
//	content = &widget.FusedContent{
//	    Segments:      segs,
//	    DefaultHeight: 30,
//	    Overscan:      3,
//	    RenderSegment: func(i int) widget.Widget { return renderSubItemData(segData, i, t) },
//	    SegmentVersion: func(i int) int { ... },
//	}
package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// ContentSegment 单个内容段的高度占位信息。
type ContentSegment struct {
	Height float64 // 估算高度（Layout 后会被实测高度覆盖）
}

// ViewportProvider 视口信息提供者接口。
// VirtualListElement 实现了此接口，FusedContentElement 通过遍历父链找到它。
type ViewportProvider interface {
	GetViewport() (scrollOffset, viewportH float64)
}

// ─── segment 槽位 ─────────────────────────────────
// 每个 slot 持有一个 Element（首次创建后永不销毁），
// 滚动时只改变 slot.index，不触发 Mount/Unmount。
type segSlot struct {
	el          Element // 槽内 Element
	index       int     // 当前映射的 segment 索引（-1 = 空闲）
	version     int     // 上次 Update 时的 SegmentVersion
	height      float64 // 上次 Layout 测量出的实际高度
	layoutWidth float64 // 上次 Layout 时的约束宽度（宽度未变+版本未变→跳过 Layout）
}

// FusedContent 内容分段虚加载容器。
//
// ★ 关键设计：
//   - Build() 返回 nil（隐藏子节点，不被管道递归 Build）
//   - Layout() 计算可见范围，分配槽位，只布局可见 segment
//   - Paint() 只绘制可见 segment
//   - Children() 返回当前可见 segment 的 Element（供 hitTest 遍历）
//   - 总高度 = 所有 segment 高度之和（不可见 segment 用估算高度占位）
//   - 通过 ViewportProvider 接口从外层 VirtualList 获取视口
type FusedContent struct {
	StatelessWidget

	Segments      []ContentSegment // 内容段列表
	DefaultHeight float64          // 默认段高（Segments 中 Height=0 时回退至此）
	Overscan      int              // 视口外额外渲染段数（默认 3）
	Height        float64          // 显式容器高度（0 = 由内容撑开）

	RenderSegment  func(i int) Widget        // 渲染第 i 段
	SegmentVersion func(i int) int            // 可选：返回段内容版本；相同则跳过 Update
	SegmentHeight  func(i int) float64        // 可选：返回第 i 段的实时估算高度；nil 用 Segments[i].Height
}

// CreateElement 创建 FusedContentElement
func (f *FusedContent) CreateElement() Element {
	os := f.Overscan
	if os <= 0 {
		os = 3
	}
	return &FusedContentElement{
		BaseElement:    BaseElement{widget: f},
		fc:             f,
		overscan:       os,
		prefixDirty:    true,
		segHeightCache: make(map[int]float64),
	}
}

// FusedContentElement FusedContent 的运行时 Element
type FusedContentElement struct {
	BaseElement
	fc       *FusedContent
	overscan int

	slots    []*segSlot
	slotBuf  []int   // 当前可见 slot 索引
	childBuf []Element // Children() 返回

	// 前缀和缓存
	prefixSum   []float64 // prefixSum[i] = 从 0 到 i-1 的高度累加
	prefixDirty bool

	// 高度缓存
	segHeightCache map[int]float64 // segment 实测高度缓存
	totalH         float64         // 总高度（所有 segment 之和）

	// 视口缓存（遍历父链获取）
	viewportProvider ViewportProvider
	lastScrollOff    float64 // 上次布局时的滚动偏移（变化时才重算可见范围）
	lastViewportH    float64
}

// ─── 前缀和 ────────────────────────────────────────

// segH 返回第 i 段的当前高度（实测优先，估算次之，默认回退）。
func (e *FusedContentElement) segH(i int) float64 {
	if h, ok := e.segHeightCache[i]; ok && h > 0 {
		return h
	}
	if e.fc.Segments != nil && i < len(e.fc.Segments) && e.fc.Segments[i].Height > 0 {
		return e.fc.Segments[i].Height
	}
	if e.fc.SegmentHeight != nil {
		if h := e.fc.SegmentHeight(i); h > 0 {
			return h
		}
	}
	if e.fc.DefaultHeight > 0 {
		return e.fc.DefaultHeight
	}
	return 30
}

// ensurePrefix 确保前缀和有效。
func (e *FusedContentElement) ensurePrefix() {
	if !e.prefixDirty && e.prefixSum != nil {
		return
	}
	n := len(e.fc.Segments)
	if n <= 0 {
		e.prefixSum = e.prefixSum[:0]
		e.totalH = 0
		e.prefixDirty = false
		return
	}
	if cap(e.prefixSum) < n+1 {
		e.prefixSum = make([]float64, n+1, n+1)
	}
	e.prefixSum = e.prefixSum[:n+1]
	cum := 0.0
	for i := 0; i < n; i++ {
		e.prefixSum[i] = cum
		cum += e.segH(i)
	}
	e.prefixSum[n] = cum
	e.totalH = cum
	e.prefixDirty = false
}

// rebuildPrefixWithMeasured 用 slot 实测高度重建前缀和。
func (e *FusedContentElement) rebuildPrefixWithMeasured() {
	n := len(e.fc.Segments)
	if n <= 0 {
		return
	}
	if cap(e.prefixSum) < n+1 {
		e.prefixSum = make([]float64, n+1, n+1)
	}
	e.prefixSum = e.prefixSum[:n+1]
	cum := 0.0
	for i := 0; i < n; i++ {
		e.prefixSum[i] = cum
		cum += e.segH(i)
	}
	e.prefixSum[n] = cum
	e.totalH = cum
	e.prefixDirty = false
}

// segOffset 返回第 i 段顶部距内容区顶部的累积偏移（前缀和 O(1)）。
func (e *FusedContentElement) segOffset(i int) float64 {
	e.ensurePrefix()
	if i <= 0 {
		return 0
	}
	if i >= len(e.prefixSum) {
		i = len(e.prefixSum) - 1
	}
	return e.prefixSum[i]
}

// segOffsetEnd 返回第 i 段底部偏移（顶部+高度）。
func (e *FusedContentElement) segOffsetEnd(i int) float64 {
	return e.segOffset(i) + e.segH(i)
}

// ─── 视口查找 ──────────────────────────────────────

// findViewportProvider 从父链查找 ViewportProvider（VirtualListElement）。
func (e *FusedContentElement) findViewportProvider() ViewportProvider {
	if e.viewportProvider != nil {
		return e.viewportProvider
	}
	p := e.Parent()
	for p != nil {
		if vp, ok := p.(ViewportProvider); ok {
			e.viewportProvider = vp
			return vp
		}
		// 有些 Element 的 Parent() 可能返回的是 container/flex 中间层，
		// 需要继续向上查找
		if be, ok := p.(*BaseElement); ok && be.parent != nil {
			p = be.parent
			continue
		}
		// 尝试通过接口方法获取 parent（更通用的方式）
		type parenter interface{ Parent() Element }
		if pp, ok := p.(parenter); ok {
			p = pp.Parent()
		} else {
			break
		}
	}
	return nil
}

// ─── 可见范围 ──────────────────────────────────────

// visibleRange 计算当前可见的 segment 范围 [start, end)。
//
// 坐标系说明：
//   VirtualList 把 item 定位在 Y = itemContentY - scrollOff（相对于 VirtualList 原点）。
//   FusedContent 挂在 item 内部，其 screen Y 由父链 e.Offset() 给出。
//   FusedContent 在 VirtualList 内容空间的 Y = e.Offset().Y - vlScreenY + scrollOff。
//   用此值做视口可见性判断。
func (e *FusedContentElement) visibleRange() (int, int) {
	n := len(e.fc.Segments)
	if n <= 0 {
		return 0, 0
	}

	vp := e.findViewportProvider()
	if vp == nil {
		return 0, n
	}
	scrollOff, viewportH := vp.GetViewport()
	if viewportH <= 0 {
		return 0, n
	}

	// 取 VirtualList Element 的屏幕 Y
	var vlScreenY float64
	if vlEl, ok := vp.(Element); ok {
		vlScreenY = vlEl.Offset().Y
	} else {
		return 0, n
	}

	fcScreenY := e.Offset().Y
	// FusedContent 在 VirtualList 内容空间的 Y（该空间原点 = scrollOff）
	fcContentY := (fcScreenY - vlScreenY) + scrollOff

	// 内容空间中的视口范围：[scrollOff, scrollOff+viewportH)
	// Segment 在内容空间的范围：[fcContentY+segOffset(i), fcContentY+segOffset(i+1))
	// 相交即可见

	// 快速拒绝
	if fcContentY+e.totalH <= scrollOff || fcContentY >= scrollOff+viewportH {
		return 0, 0
	}

	// 二分查找第一个 segOffset(i+1)+fcContentY > scrollOff 的 i
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		if e.segOffset(mid+1)+fcContentY > scrollOff {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	start := lo

	// 二分查找第一个 segOffset(i)+fcContentY >= scrollOff+viewportH 的 i
	lo, hi = start, n
	for lo < hi {
		mid := (lo + hi) / 2
		if e.segOffset(mid)+fcContentY >= scrollOff+viewportH {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	end := lo

	start -= e.overscan
	end += e.overscan
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}
	return start, end
}

// searchIndex 二分查找第一个 segOffset(i+1) > target 的 i。
func (e *FusedContentElement) searchIndex(target float64) int {
	n := len(e.fc.Segments)
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		if e.segOffset(mid+1) > target {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo
}

// ─── Build ────────────────────────────────────────

func (e *FusedContentElement) Build() []Element {
	return nil // 隐藏子节点，不被管道递归 Build
}

// Children 返回当前可见 segment 的 Element（仅用于 hitTest）。
func (e *FusedContentElement) Children() []Element {
	return e.childBuf
}

// ─── Layout ────────────────────────────────────────

func (e *FusedContentElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	fc := e.fc

	// 不可见时返回零尺寸（安全兜底）
	if len(fc.Segments) == 0 {
		e.totalH = 0
		e.size = ctx.Constraints.Constrain(types.Size{Width: 0, Height: 0})
		return layout.LayoutResult{Size: e.size}
	}

	w := ctx.Constraints.MaxWidth
	// 如果 MaxWidth 是无穷大，用显式宽或默认宽
	if w >= float64(1<<30) {
		if fc.Height > 0 {
			w = ctx.Constraints.MinWidth
			if w <= 0 {
				w = 300
			}
		} else {
			w = 300
		}
	}

	// 确保前缀和
	e.ensurePrefix()

	// 检测视口变化（只有变化时才重算可见范围）
	vp := e.findViewportProvider()
	if vp != nil {
		so, vh := vp.GetViewport()
		if so != e.lastScrollOff || vh != e.lastViewportH {
			e.lastScrollOff = so
			e.lastViewportH = vh
		}
	}

	// 第一阶段：分配所有 segment 的 slot（全部可见，确保每个 segment 都有对应的 Element 被创建+Layout）。
	// ★ 为什么不在此时就按 viewport 裁剪：Layout 阶段 e.Offset() 可能是旧帧的折叠态位置（父链未更新），
	// 会导致 visibleRange 算出 (0,0)，segment 的 Element 压根不创建、永不可见。
	// 等到 repositionAll 修正位置后，第二阶段 refreshVisibleSlots 再按正确 offset 裁剪。
	e.assignAllSlots()

	// 布局每个可见 slot
	for _, s := range e.slots {
		if s.index < 0 {
			continue
		}

		// 跳过 Layout：宽度未变且版本未变且已有高度
		if s.layoutWidth == w && s.height > 0 {
			versionSame := true
			if fc.SegmentVersion != nil {
				v := fc.SegmentVersion(s.index)
				if v != s.version || v == 0 {
					versionSame = false
				}
			}
			if versionSame {
				continue
			}
		}

		childCtx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{
				MinWidth:  w,
				MaxWidth:  w,
				MinHeight: 0,
				MaxHeight: 1 << 20,
			},
		}
		s.el.Layout(childCtx)
		s.layoutWidth = w
		actualH := s.el.Size().Height
		if s.height != actualH {
			s.height = actualH
			e.segHeightCache[s.index] = actualH
			e.prefixDirty = true
		}
	}

	// 用实测值重建前缀和
	if e.prefixDirty {
		e.rebuildPrefixWithMeasured()
	}

	// 定位所有 slot（设位置，用于 hitTest）
	e.repositionAll()

	// 取下帧的可见范围供 Paint 裁剪（仅用于性能优化，不影响 slot 分配）
	// ★ 不再调用 refreshVisibleSlots：所有 segment slot 保持分配，不释放。
	// Paint 时按 visibleRange 裁剪绘制，避免坐标计算错误导致内容消失。

	// 总高度 = 内容总高（不可见 segment 用估算高度占位）
	totalH := e.totalH

	// 确定最终尺寸
	h := totalH
	if fc.Height > 0 {
		h = fc.Height
	}

	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

// assignAllSlots 为所有 segment 分配 slot（第一阶段：全部可见）。
// 确保每个 segment 都有对应的 Element 被创建+Layout，不受陈旧 e.Offset() 干扰。
// repositionAll 后再用 refreshVisibleSlots 按真实 offset 裁剪。
func (e *FusedContentElement) assignAllSlots() {
	n := len(e.fc.Segments)
	if n <= 0 || e.fc.RenderSegment == nil {
		e.slotBuf = nil
		e.childBuf = nil
		return
	}

	// 确保槽池大小 = 全量 segment（首次 Layout 时全量创建，后续 Layout slot 已存在→复用 Update）
	e.ensureSlotPoolFull(n)

	// 为每个 segment 分配/复用 slot
	for i := 0; i < n; i++ {
		// 查找是否已有 slot 映射到此 segment
		found := -1
		for si, s := range e.slots {
			if s.index == i {
				found = si
				break
			}
		}
		if found >= 0 {
			// 复用：更新
			s := e.slots[found]
			shouldUpdate := true
			if e.fc.SegmentVersion != nil {
				v := e.fc.SegmentVersion(i)
				if s.version == v && v != 0 {
					shouldUpdate = false
				}
				s.version = v
			}
			if shouldUpdate {
				w := e.fc.RenderSegment(i)
				if w != nil && s.el != nil {
					s.el.Update(w)
					s.layoutWidth = 0 // 强制重测 Layout
					e.buildSubtree(s.el)
				}
			}
			e.slotBuf = append(e.slotBuf, found)
			continue
		}
		// 找空闲 slot
		free := -1
		for si, s := range e.slots {
			if s.index < 0 {
				free = si
				break
			}
		}
		if free < 0 {
			continue // 池满，不应发生
		}
		s := e.slots[free]
		s.index = i
		w := e.fc.RenderSegment(i)
		if w == nil {
			continue
		}
		if s.el == nil {
			s.el = CreateElementFor(w)
			s.el.Mount(e, i)
			e.buildSubtree(s.el)
		} else {
			s.el.Update(w)
			s.layoutWidth = 0
			e.buildSubtree(s.el)
		}
		e.slotBuf = append(e.slotBuf, free)
	}

	// 刷新 childBuf
	e.childBuf = e.childBuf[:0]
	for _, si := range e.slotBuf {
		if s := e.slots[si]; s != nil && s.el != nil {
			e.childBuf = append(e.childBuf, s.el)
		}
	}
}

// ensureSlotPoolFull 确保槽池大小为 segment 总数（确保每个 segment 都有槽位）。
func (e *FusedContentElement) ensureSlotPoolFull(n int) {
	if len(e.slots) >= n {
		return
	}
	start := len(e.slots)
	for i := start; i < n; i++ {
		e.slots = append(e.slots, &segSlot{index: -1})
	}
}

// refreshVisibleSlots 用当前正确 offset 重新计算可见范围，释放视口外的 slot。
// 在 repositionAll 之后调用，此时 e.Offset() 已反映正确位置。
func (e *FusedContentElement) refreshVisibleSlots() {
	start, end := e.visibleRange()

	// 释放离开可见区的 slot
	for _, s := range e.slots {
		if s.index >= 0 && (s.index < start || s.index >= end) {
			s.index = -1
		}
	}

	// 重建 slotBuf
	e.slotBuf = e.slotBuf[:0]
	for si, s := range e.slots {
		if s.index >= 0 {
			e.slotBuf = append(e.slotBuf, si)
		}
	}

	// 刷新 childBuf
	e.childBuf = e.childBuf[:0]
	for _, si := range e.slotBuf {
		if s := e.slots[si]; s != nil && s.el != nil {
			e.childBuf = append(e.childBuf, s.el)
		}
	}
}

// assignVisibleSlots 分配可见 segment 到槽池（保留原接口供其他调用方）。
func (e *FusedContentElement) assignVisibleSlots() {
	n := len(e.fc.Segments)
	if n <= 0 || e.fc.RenderSegment == nil {
		e.slotBuf = nil
		e.childBuf = nil
		return
	}

	// 确保槽池已初始化（Layout 可能先于 Update 调用）
	e.ensureSlotPool()

	start, end := e.visibleRange()
	if start >= end {
		e.slotBuf = nil
		e.childBuf = nil
		return
	}

	// Step 1: 释放离开可见区的 slot
	for _, s := range e.slots {
		if s.index >= 0 && (s.index < start || s.index >= end) {
			s.index = -1
		}
	}

	// Step 2: 分配可见区间 slot
	e.slotBuf = e.slotBuf[:0]
	for i := start; i < end; i++ {
		found := -1
		for si, s := range e.slots {
			if s.index == i {
				found = si
				break
			}
		}
		if found >= 0 {
			// 复用：Update 该 slot
			s := e.slots[found]
			shouldUpdate := true
			if e.fc.SegmentVersion != nil {
				v := e.fc.SegmentVersion(i)
				if s.version == v && v != 0 {
					shouldUpdate = false
				}
				s.version = v
			}
			if shouldUpdate {
				w := e.fc.RenderSegment(i)
				if w != nil && s.el != nil {
					s.el.Update(w)
					s.layoutWidth = 0 // 强制重测 Layout
					e.buildSubtree(s.el)
				}
			}
			e.slotBuf = append(e.slotBuf, found)
			continue
		}
		// 找空闲 slot
		free := -1
		for si, s := range e.slots {
			if s.index < 0 {
				free = si
				break
			}
		}
		if free < 0 {
			free = e.furthestSlot(start, end)
		}
		if free < 0 {
			// 槽池为空（不应发生，安全兜底）
			continue
		}
		s := e.slots[free]
		s.index = i
		w := e.fc.RenderSegment(i)
		if w == nil {
			continue
		}
		shouldUpdate := true
		if e.fc.SegmentVersion != nil {
			v := e.fc.SegmentVersion(i)
			if s.version == v && v != 0 {
				shouldUpdate = false
			}
			s.version = v
		}
		if s.el == nil {
			s.el = CreateElementFor(w)
			s.el.Mount(e, i)
			e.buildSubtree(s.el)
		} else if shouldUpdate {
			s.el.Update(w)
			s.layoutWidth = 0
			e.buildSubtree(s.el)
		}
		e.slotBuf = append(e.slotBuf, free)
	}

	// Step 3: 刷新 childBuf
	e.childBuf = e.childBuf[:0]
	for _, si := range e.slotBuf {
		if s := e.slots[si]; s != nil && s.el != nil {
			e.childBuf = append(e.childBuf, s.el)
		}
	}
}
// furthestSlot 返回距当前可见区间最远的 slot 索引。
func (e *FusedContentElement) furthestSlot(start, end int) int {
	if len(e.slots) == 0 {
		return -1
	}
	worst := 0
	worstDist := -1
	for si, s := range e.slots {
		if s.index < 0 {
			return si
		}
		dist := 0
		if s.index < start {
			dist = start - s.index
		} else {
			dist = s.index - end + 1
		}
		if dist > worstDist {
			worstDist = dist
			worst = si
		}
	}
	return worst
}

// ─── Paint ────────────────────────────────────────

// Paint 绘制 FusedContent 的全部已分配 slot。
//
// 1. 先从 state 刷新版本变化的内容（SegmentVersion + RenderSegment）
// 2. 计算可见范围，跳过视口外的 segment
// 3. 可见范围内调用 slot.el.Paint 绘制
//
// 可见范围由 corrected visibleRange 计算（修复了混用 content-space/screen-space 的 bug）。
// 所有 slot Element 保持分配（不释放），Paint 时按可见性裁剪绘制。
func (e *FusedContentElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if len(e.fc.Segments) == 0 || e.fc.RenderSegment == nil {
		return
	}

	// Step 1: 从 state 刷新版本变化的内容
	// 当 buildTree 不运行时（SetStateNoRebuild），slot 不会收到 Update()，
	// 因此需要在 Paint 阶段自行检查 SegmentVersion 并更新内容。
	for _, s := range e.slots {
		if s == nil || s.index < 0 || s.el == nil {
			continue
		}
		if e.fc.SegmentVersion != nil {
			v := e.fc.SegmentVersion(s.index)
			if s.version != v || v == 0 {
				s.version = v
				w := e.fc.RenderSegment(s.index)
				if w != nil {
					s.el.Update(w)
					s.layoutWidth = 0 // 下次 Layout 强制重测
					e.buildSubtree(s.el)
				}
			}
		}
	}

	// Step 2: 计算可见范围（用修正后的坐标公式）
	start, end := e.visibleRange()
	if start >= end {
		return
	}

	// Step 3: 绘制可见 segment
	for _, s := range e.slots {
		if s == nil || s.index < 0 || s.el == nil || s.height <= 0 {
			continue
		}
		if s.index < start || s.index >= end {
			continue
		}
		segTop := e.segOffset(s.index)
		s.el.Paint(cvs, types.Point{X: offset.X, Y: offset.Y + segTop})
	}
}

// ─── Update ────────────────────────────────────────

func (e *FusedContentElement) Update(newWidget Widget) {
	// ★ 关键优化：相同 Widget 指针 ⇒ 内容未变 ⇒ 跳过所有更新。
	// 当 getOrBuildMessage 返回缓存的消息 Widget（指针与上一帧相同）时，
	// VirtualList slot 仍会调用 Update，但此处快速返回，避免 prefix 重建 + Layout。
	if newWidget == e.widget || newWidget == e.fc {
		return
	}

	if nf, ok := newWidget.(*FusedContent); ok {
		oldN := 0
		if e.fc != nil {
			oldN = len(e.fc.Segments)
		}
		e.fc = nf
		e.BaseElement.widget = newWidget

		// Segment 数量变化 → 清槽池 + 高度缓存
		newN := len(nf.Segments)
		if newN != oldN {
			for _, s := range e.slots {
				if s.el != nil {
					s.el.Unmount()
				}
			}
			e.slots = nil
			e.slotBuf = nil
			e.childBuf = nil
			e.segHeightCache = make(map[int]float64)
			e.prefixDirty = true
			e.ensureSlotPool()
		} else {
			e.prefixDirty = true
		}

		// 视口提供者重置（下次查找）
		e.viewportProvider = nil
	}
}

// ─── 槽池管理 ──────────────────────────────────────

// ensureSlotPool 确保槽池存在且大小合理。
func (e *FusedContentElement) ensureSlotPool() {
	n := len(e.fc.Segments)
	if n <= 0 {
		return
	}

	// 槽池大小：min(n, 可见数 + overscan*2, 硬上限 30)
	poolSize := n
	if vp := e.findViewportProvider(); vp != nil {
		_, vh := vp.GetViewport()
		if vh > 0 {
			defH := e.fc.DefaultHeight
			if defH <= 0 {
				defH = 30
			}
			visibleCount := int(vh/defH) + 1
			poolSize = visibleCount + 2*e.overscan
		}
	}
	if poolSize > n {
		poolSize = n
	}
	if poolSize < 5 {
		poolSize = 5
	}
	if poolSize > 30 {
		poolSize = 30
	}

	if len(e.slots) >= poolSize {
		return
	}
	start := len(e.slots)
	for i := start; i < poolSize; i++ {
		e.slots = append(e.slots, &segSlot{index: -1})
	}
}

// ─── 调试支持 ────────────────────────────────────

// FusedContentSlotInfo 单个 slot 的调试信息
type FusedContentSlotInfo struct {
	SlotIndex  int
	SegIndex   int
	Height     float64
	SegOffset  float64
	Element    Element
}

// DebugSlots 返回当前所有 slot 的调试信息，供测试和诊断使用。
func (e *FusedContentElement) DebugSlots() []FusedContentSlotInfo {
	info := make([]FusedContentSlotInfo, 0, len(e.slots))
	for si, s := range e.slots {
		if s.index >= 0 {
			info = append(info, FusedContentSlotInfo{
				SlotIndex: si,
				SegIndex:  s.index,
				Height:    s.height,
				SegOffset: e.segOffset(s.index),
				Element:   s.el,
			})
		}
	}
	return info
}

// TotalContentH 返回所有 segment 的总高度。
func (e *FusedContentElement) TotalContentH() float64 {
	return e.totalH
}

// buildSubtree 递归构建 Element 子树。
// ★ 核心：FusedContentElement.Build() 返回 nil，管线的 buildTree 不会进入 FusedContent
// 内部。因此 slot element 的子树必须由 FusedContent 自行构建，否则 slot 的子控件树为空→
// Paint 无内容可画→内容不渲染（这是之前 FusedContent 不渲染的根因）。
func (e *FusedContentElement) buildSubtree(el Element) {
	children := el.Build()
	for _, child := range children {
		e.buildSubtree(child)
	}
}

// ─── 定位 ──────────────────────────────────────────

// repositionAll 设置所有 slot 子 Element 的绝对位置。
// 包含 FusedContentElement 自身的偏移（由父级通过 SetPosition 设置）。
// 位置影响 hitTest 命中判定和 ContainerElement 的子控件绘制坐标。
func (e *FusedContentElement) repositionAll() {
	// ★ 关键：slot 位置必须是 RELATIVE（相对于 FusedContentElement 自己）。
	// 之前用了 e.Offset()（绝对坐标），但 layout 期间父级尚未完全定位，
	// e.Offset() 不准确，加上 BaseElement.Offset() 链式累加父偏移，
	// 导致 slot 最终位置双倍计数，内容画出屏幕不可见。
	// 改为相对 (0, segOff)，父链的 Offset() 会自动累加 FusedContent 的偏移，
	// 得到正确的绝对位置。
	for _, s := range e.slots {
		if s.index < 0 || s.el == nil {
			continue
		}
		off := e.segOffset(s.index)
		s.el.SetPosition(types.Point{X: 0, Y: off})
	}
}

// SetPosition 覆写 BaseElement.SetPosition。
// 由于 slot 位置是相对的（相对 FusedContentElement），父级设置 FC 位置时
// slot 位置不随之变化（BaseElement.Offset() 链自动累加父偏移）。
// 因此不再需要在此处调用 repositionAll。
func (e *FusedContentElement) SetPosition(pos types.Point) {
	e.BaseElement.SetPosition(pos)
}

// ─── 事件处理 ──────────────────────────────────────
// FusedContent 本身不需要处理滚动/鼠标事件（由外层 VirtualList 处理）。
func (e *FusedContentElement) HandleEvent(ev event.Event) bool { return false }
