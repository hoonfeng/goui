package widget

import (
	"time"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─────────── slot: 槽池中的单个槽位 ───────────
// 每个 slot 持有一个 Element（首次创建后永不销毁），
// 滚动时只改变 slot.index（映射的 item 索引）和 slot 的 position，
// 不触发 Mount/Unmount。
type slot struct {
	el          Element // 槽内 Element（创建后挂载一次，永不卸载）
	index       int     // 当前映射的 item 索引（-1 = 空闲）
	version     int     // 上次 Update 时的 ItemVersion（未变则跳过 Update）
	height      float64 // 上次 Layout 测量出的实际高度
	layoutWidth float64 // 上次 Layout 时的约束宽度；宽度未变+版本未变→跳过 Layout
}

// VirtualList 虚拟滚动列表
//
// 核心架构（参考 @tanstack/react-virtual + Flutter ListView.builder）：
//   - Item 完全私有：不暴露给 pipeline 的 buildTree（Build 返回 nil），
//     避免 pipeline 递归重建每个 item 的子树。
//   - 槽池复用：固定大小的 Slot Pool 在首次 Build 时创建，item element
//     创建后永不销毁。滚动时只改变 slot.index 映射，不触发 Mount/Unmount。
//   - Layout 驱动：可见项分配、测量、定位全部在 Layout 中完成。
//     Build 仅为首次创建池，后续不做任何工作。
//   - Children 仅用于命中测试：返回当前可见 slot 的 Element，
//     让 hitTest 能正确找到每个 item 内部的交互组件。
//   - ScrollOffset 作为 scrollOffset 的权威来源：每次 Widget Build 时传入
//     实际滚动位置，Element 的 Update 中同步此值，跨重建也从这里恢复。
//   - 前缀和（参考 @tanstack/react-virtual）：O(n) 构建一次，之后 O(1) itemOffset +
//     O(log n) visibleRange + O(1) totalContentH，与总消息数无关。
type VirtualList struct {
	StatelessWidget
	ItemCount             int
	ItemHeight            float64   // 固定行高（ItemHeights 为 nil 时回退至此）
	ItemHeights           []float64 // 可选：每项估算高度（Layout 后会测量实际值）
	Height                float64   // 显式高度（0 = 从父布局撑满）
	Width                 float64   // 显式宽度（0 = 从父布局撑满）
	Overscan              int       // 视口外额外渲染行数（默认 5）
	RenderItem            func(index int) Widget
	OnReachEnd            func()
	OnScroll              func(scrollOffset float64)
	ScrollToBottomToken   int                 // 改变此值 → 下次布局滚到底部
	InitialScrollOffset   float64             // 初始滚动偏移（跨 Element 重建恢复用，Element 首次创建时使用）
	ScrollOffset          float64             // ★ 权威滚动偏移：每次 Update() 都会同步此值到 Element，跨重建也从此恢复取。>=0 时生效，-1 表示不覆盖。
	ItemVersion           func(index int) int // 可选：返回内容的版本；相同则跳过 Update
	AutoScrollMinInterval time.Duration       // 自动滚底最小间隔（默认 1s）
	AutoScrollToBottom    bool                // 首次有内容时自动滚到底部（对话列表等需要）

	// 跨 Element 重建持久化标记（chat.go 的 scrollMessages 从 ChatState 回填这些值）
	AutoScrolledAlready     bool // 首次自动滚底已触发
	LastScrollToBottomToken int  // 上次记录的 ScrollToBottomToken
}

// NewVirtualList 创建固定高度的虚拟列表（兼容旧 API，Height=280）。
func NewVirtualList(count int, itemH float64, render func(int) Widget) *VirtualList {
	return &VirtualList{
		ItemCount: count, ItemHeight: itemH, Height: 280, Overscan: 5,
		RenderItem: render,
	}
}
func (v *VirtualList) WithHeight(h float64) *VirtualList          { v.Height = h; return v }
func (v *VirtualList) WithWidth(w float64) *VirtualList           { v.Width = w; return v }
func (v *VirtualList) WithOverscan(n int) *VirtualList            { v.Overscan = n; return v }
func (v *VirtualList) WithOnReachEnd(fn func()) *VirtualList      { v.OnReachEnd = fn; return v }
func (v *VirtualList) WithOnScroll(fn func(float64)) *VirtualList { v.OnScroll = fn; return v }
func (v *VirtualList) SetItemHeights(h []float64) *VirtualList    { v.ItemHeights = h; return v }
func (v *VirtualList) WithScrollToBottom(token int) *VirtualList {
	v.ScrollToBottomToken = token
	return v
}

// CreateElement 创建 VirtualListElement
func (v *VirtualList) CreateElement() Element {
	osVal := v.Overscan
	if osVal <= 0 {
		osVal = 5
	}
	so := v.ScrollOffset
	extSet := so >= 0
	if !extSet {
		so = v.InitialScrollOffset
	}
	if so < 0 {
		so = 0
	}
	return &VirtualListElement{
		BaseElement:          BaseElement{widget: v},
		vl:                   v,
		overscan:             osVal,
		scrollOffset:         so,
		autoScrolledToBottom: v.AutoScrolledAlready,
		lastScrollToBottom:   v.LastScrollToBottomToken,
		prefixDirty:          true,
		prefixFirstDirty:     -1,
		itemHeightMap:        make(map[int]float64, 64), // 预分配容量避免频繁 grow
		externalScrollSet:     extSet,
		slotLayoutCtx:        layout.LayoutContext{},
	}
}

// VirtualListElement VirtualList 的运行时 Element
type VirtualListElement struct {
	BaseElement
	vl           *VirtualList
	scrollOffset float64
	maxScroll    float64
	viewportH    float64
	overscan     int

	autoScrolledToBottom bool
	slots                []*slot
	visibleSlots         []int
	childrenBuf          []Element

	// slotIndexMap 提供 item 索引到 slot 的 O(1) 查找。
	// 长度 = |slots|，slotIndexMap[itemIndex] = slot索引 或 -1。
	// 在 assignVisibleSlots 中更新，避免 O(slots²) 的内层扫描。
	slotIndexMap []int

	lastScrollToBottom int
	lastAutoScroll     time.Time
	lastUserScroll     time.Time
	lastFiredOffset    float64
	userScrolled       bool

	draggingBar      bool
	dragStartMouseY  float64
	dragStartScrollY float64

	// ── 高度快速查找 ──
	itemHeightMap map[int]float64

	// heightsChanged 标记本轮 Layout 中是否有 slot 高度变化，用于判定是否需要重建前缀和。
	heightsChanged bool

	// externalScrollSet 标记当前 scrollOffset 是外部通过 Widget.ScrollOffset 设置的，
	// Layout 应跳过 anchoring 逻辑、原样保留。用户滚轮/拖拽后自动清空。
	externalScrollSet bool

	// ── 前缀和缓存 ──
	prefixSum        []float64
	prefixDirty      bool
	prefixFirstDirty int

	// ── LayoutContext 复用 ──
	// slotLayoutCtx 是每个 slot 子布局共用的 LayoutContext，避免每帧为每个 slot 分配堆对象。
	// 每次 Layout 前修改 Constraints 字段即可复用，无需 &layout.LayoutContext{}。
	slotLayoutCtx layout.LayoutContext
}

// ─── 前缀和 ────────────────────────────────────────

// invalidatePrefix 标记前缀和失效（高度变化后调用）。
func (e *VirtualListElement) invalidatePrefix() {
	e.prefixDirty = true
	e.prefixFirstDirty = 0 // 未知范围，全量重建
}

// itemHFromEstimate 返回第 i 项的外部估算高度（不扫 slot），用于 prefix sum 构建。
func (e *VirtualListElement) itemHFromEstimate(i int) float64 {
	if e.vl.ItemHeights != nil && i >= 0 && i < len(e.vl.ItemHeights) {
		return e.vl.ItemHeights[i]
	}
	if e.vl.ItemHeight > 0 {
		return e.vl.ItemHeight
	}
	return 30
}

// ensurePrefix 确保前缀和有效，失效时重建（增量：从 prefixFirstDirty 向后）。
func (e *VirtualListElement) ensurePrefix() {
	if !e.prefixDirty && e.prefixSum != nil {
		return
	}
	n := e.vl.ItemCount
	if n <= 0 {
		e.prefixSum = e.prefixSum[:0]
		e.prefixDirty = false
		e.prefixFirstDirty = -1
		return
	}
	if cap(e.prefixSum) < n+1 {
		e.prefixSum = make([]float64, n+1, n+1)
	}
	e.prefixSum = e.prefixSum[:n+1]

	// 增量构建：从 prefixFirstDirty（或 0）向后重建，保留前面未变的部分
	start := e.prefixFirstDirty
	if start < 0 || start > n {
		start = 0
	}
	var cum float64
	if start == 0 {
		cum = 0
	} else {
		cum = e.prefixSum[start] // 复用 start 之前的正确值
	}
	for i := start; i < n; i++ {
		e.prefixSum[i] = cum
		cum += e.itemH(i)
	}
	e.prefixSum[n] = cum
	e.prefixDirty = false
	e.prefixFirstDirty = -1
}

// rebuildPrefixWithMeasured 用 slot 实测高度重建前缀和（增量：从 prefixFirstDirty 向后）。
// Layout 末尾调用，使 maxScroll 和 repositionAll 反映实际测量值。
// 相比全量 O(N) 重建，仅 O(N-startIdx) 与变化项数量相关。
func (e *VirtualListElement) rebuildPrefixWithMeasured() {
	n := e.vl.ItemCount
	if n <= 0 {
		e.prefixFirstDirty = -1
		return
	}
	if cap(e.prefixSum) < n+1 {
		e.prefixSum = make([]float64, n+1, n+1)
	}
	e.prefixSum = e.prefixSum[:n+1]

	start := e.prefixFirstDirty
	if start < 0 || start > n {
		start = 0
	}
	var cum float64
	if start == 0 {
		cum = 0
	} else {
		cum = e.prefixSum[start] // 复用 start 之前未变的前缀值
	}
	for i := start; i < n; i++ {
		e.prefixSum[i] = cum
		cum += e.itemH(i) // 含 slot 实测高度
	}
	e.prefixSum[n] = cum
	e.prefixDirty = false
	e.prefixFirstDirty = -1
}

// ─── 公开接口 ──────────────────────────────────────

func (e *VirtualListElement) ScrollOffset() float64 { return e.scrollOffset }

// GetViewport 实现 ViewportProvider 接口，返回当前滚动偏移和视口高度。
func (e *VirtualListElement) GetViewport() (scrollOffset, viewportH float64) {
	return e.scrollOffset, e.viewportH
}
func (e *VirtualListElement) ViewportH() float64 { return e.viewportH }
func (e *VirtualListElement) MaxScroll() float64 { return e.maxScroll }

// ScrollTo 跳到指定偏移（外部强制滚动）。
func (e *VirtualListElement) ScrollTo(offset float64) {
	e.scrollOffset = vlClamp(offset, 0, vlMaxF(0, e.maxScroll))
	e.rebuildVisible()
	e.fireScroll()
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// Children 返回当前可见的 slot Element（仅供 hitTest 遍历，不做 layout/paint）。
func (e *VirtualListElement) Children() []Element {
	return e.childrenBuf
}

// ─── 高度计算（前缀和驱动）───────────────────────────

// itemH 返回第 i 项的当前高度（实测优先，其次估算，最后固定值）。
// 使用 itemHeightMap 做 O(1) 查找，避免扫描全部 slots。
func (e *VirtualListElement) itemH(i int) float64 {
	if i >= 0 {
		if e.itemHeightMap != nil {
			if h, ok := e.itemHeightMap[i]; ok && h > 0 {
				return h
			}
		}
	}
	if e.vl.ItemHeights != nil && i >= 0 && i < len(e.vl.ItemHeights) {
		return e.vl.ItemHeights[i]
	}
	if e.vl.ItemHeight > 0 {
		return e.vl.ItemHeight
	}
	return 30
}

// itemOffset 返回第 i 项顶部距内容区顶部的累计偏移。
// 前缀和 O(1) 查找。
func (e *VirtualListElement) itemOffset(i int) float64 {
	e.ensurePrefix()
	if i <= 0 {
		return 0
	}
	if i >= len(e.prefixSum) {
		i = len(e.prefixSum) - 1
	}
	return e.prefixSum[i]
}

// totalContentH 返回全部项的总高度。
// 前缀和 O(1) 查找。
func (e *VirtualListElement) totalContentH() float64 {
	e.ensurePrefix()
	n := len(e.prefixSum)
	if n <= 0 {
		return 0
	}
	return e.prefixSum[n-1]
}

// searchIndex 二分查找第一个 height sum > target 的 item 索引。
// 即第一个满足 itemOffset(i+1) > target 的 i。
// O(log n) 次前缀和读取。
func (e *VirtualListElement) searchIndex(target float64) int {
	n := e.vl.ItemCount
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		if e.itemOffset(mid+1) > target {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo
}

// visibleRange 依据 scrollOffset 和 viewportH 算可见区间 [start, end)。
// 二分查找 O(log n)，不因消息总数增长而变慢。
func (e *VirtualListElement) visibleRange() (int, int) {
	n := e.vl.ItemCount
	if n <= 0 || e.viewportH <= 0 {
		return 0, 0
	}
	so := e.scrollOffset
	vh := e.viewportH

	// 二分查找：第一个 itemOffset(i+1) > so 的 i
	start := e.searchIndex(so)
	// 二分查找：第一个 itemOffset(i) > so+vh 的 i
	end := e.searchIndex(so + vh)

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

// ─── Build（被 pipeline.buildTree 递归调用）───────
func (e *VirtualListElement) Build() []Element {
	e.ensureSlotPool()
	return nil
}

// ensureSlotPool 确保槽池存在且大小足够。
func (e *VirtualListElement) ensureSlotPool() {
	poolSize := e.vl.ItemCount
	if poolSize > 50 {
		poolSize = 50
		vh := e.viewportH
		if vh <= 0 {
			vh = 600
		}
		minH := e.vl.ItemHeight
		if minH <= 0 {
			minH = 30
		}
		visible := int(vh/minH) + 1
		dynSize := visible + 2*e.overscan
		if dynSize < 10 {
			dynSize = 10
		}
		if dynSize < poolSize {
			poolSize = dynSize
		}
	}
	if poolSize < 1 {
		poolSize = 1
	}
	if len(e.slots) >= poolSize {
		return
	}
	start := len(e.slots)
	for i := start; i < poolSize; i++ {
		e.slots = append(e.slots, &slot{index: -1})
	}
}

// ─── Layout（核心逻辑）─────────────────────────────
func (e *VirtualListElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	vl := e.vl

	w := vl.Width
	if w <= 0 {
		w = ctx.Constraints.MaxWidth
		if w >= float64(1<<30) {
			w = 300
		}
	}

	vh := vl.Height
	if vh <= 0 {
		vh = ctx.Constraints.MaxHeight
		if vh >= float64(1<<30) {
			vh = 600
		}
	}
	viewportChanged := e.viewportH != vh
	e.viewportH = vh

	// ★ 保存进入 Layout 时的 scrollOffset，若内容高度未变则在退出时恢复
	//   （防止鼠标移动/点击等无内容变化的 SetState→重建 意外改变滚动位置）
	scrollOffsetAtLayoutEntry := e.scrollOffset
	oldItemCount := e.vl.ItemCount

	e.ensureSlotPool()
	e.assignVisibleSlots()

	// 【优化】使用复用 LayoutContext 避免每 slot 的堆分配
	e.slotLayoutCtx.Constraints = layout.BoxConstraints{
		MinWidth:  w,
		MaxWidth:  w,
		MinHeight: 0,
		MaxHeight: 1 << 20,
	}

	// 布局每个可见 slot（仅宽度或版本变化时才重布局）
	// 保存旧 itemHeightMap 快照（scroll anchoring 需要测量前的高度）
	oldHeights := make(map[int]float64, len(e.itemHeightMap))
	for k, v := range e.itemHeightMap {
		oldHeights[k] = v
	}
	e.heightsChanged = false
	for _, s := range e.slots {
		if s.index < 0 {
			continue
		}

		// 跳过 Layout：宽度未变 且 版本未变 且 已有高度
		if s.layoutWidth == w && s.height > 0 {
			versionSame := true
			if e.vl.ItemVersion != nil {
				v := e.vl.ItemVersion(s.index)
				if v != s.version || v == 0 {
					versionSame = false
				}
			}
			if versionSame {
				continue
			}
		}

		s.el.Layout(&e.slotLayoutCtx)
		s.layoutWidth = w
		actualH := s.el.Size().Height
		if s.height-actualH < -0.5 || s.height-actualH > 0.5 {
			s.height = actualH
			e.heightsChanged = true
			if e.prefixFirstDirty < 0 || s.index < e.prefixFirstDirty {
				e.prefixFirstDirty = s.index
			}
			if e.itemHeightMap == nil {
				e.itemHeightMap = make(map[int]float64)
			}
			e.itemHeightMap[s.index] = actualH
		}
	}

	// [scroll anchoring] 内容高度变化时，记录锚点偏移差，补偿 scrollOffset
	// 使视口第一可见项停留在同一屏幕位置，避免上方消息生长时视觉抖动
	if e.heightsChanged {
		if e.externalScrollSet && e.vl.ItemCount == oldItemCount {
			// 外部控制 + ItemCount 未变：外部传入的 scrollOffset 保持原值
			e.scrollOffset = scrollOffsetAtLayoutEntry
			e.externalScrollSet = false
			e.rebuildPrefixWithMeasured()
		} else {
			// ItemCount 变化或非外部控制：标准 scroll anchoring 补偿内容变化
			e.externalScrollSet = false
			realMap := e.itemHeightMap
			e.itemHeightMap = oldHeights
			e.invalidatePrefix()

			anchorIdx := e.searchIndex(e.scrollOffset)
			anchorOldOffset := e.itemOffset(anchorIdx)

			e.itemHeightMap = realMap
			e.invalidatePrefix()
			e.rebuildPrefixWithMeasured()

			anchorNewOffset := e.itemOffset(anchorIdx)
			delta := anchorNewOffset - anchorOldOffset
			e.scrollOffset += delta
		}
	} else if e.prefixDirty || len(e.prefixSum) != e.vl.ItemCount+1 {
		e.ensurePrefix()
	}

	// 更新总高度与 maxScroll
	totalH := e.totalContentH()
	e.maxScroll = vlMaxF(0, totalH-e.viewportH)
	// ★ 安全保护：内容高度足以滚动时，不允许 scrollOffset 被错误地 clamp 到 0
	//   （防止各种边缘情况导致 scrollOffset 丢失）
	prevScroll := e.scrollOffset
	clampMax := vlMaxF(0, e.maxScroll)
	e.scrollOffset = vlClamp(e.scrollOffset, 0, clampMax)
	if e.maxScroll > 0 && e.scrollOffset == 0 && prevScroll > 0 {
		e.scrollOffset = vlMinF(prevScroll, e.maxScroll)
	}

	// 【关键】内容高度未变时，恢复 Layout 进入时的 scrollOffset（避免鼠标移动等无内容变化的 SetState 意外改变滚动位置）
	if !e.heightsChanged {
		newSO := vlClamp(scrollOffsetAtLayoutEntry, 0, vlMaxF(0, e.maxScroll))
		if newSO != e.scrollOffset {
		}
		e.scrollOffset = newSO
	}

	// 定位所有 slot
	e.repositionAll()

	// 受控滚到底部
	e.handleAutoScroll()

	if viewportChanged {
		e.fireScroll()
	}

	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: vh})
	return layout.LayoutResult{Size: e.size}
}

// assignVisibleSlots 把可见区间内的 item 分配到 slot 池。
// 使用 slotIndexMap 做 O(1) 查找，替代 O(slots²) 的线性扫描。
func (e *VirtualListElement) assignVisibleSlots() {
	n := e.vl.ItemCount
	if n <= 0 || e.vl.RenderItem == nil {
		e.visibleSlots = nil
		e.childrenBuf = nil
		return
	}

	start, end := e.visibleRange()
	if start >= end {
		e.visibleSlots = nil
		e.childrenBuf = nil
		return
	}

	// Step 1: 构建 slotIndexMap（item 索引 → slot 索引），O(slots) 一次扫描。
	// 之前每个 item 循环扫描 slots 导致 O(slots × visibleRange)。现在对所有可见 item 的
	// lookup 是 O(1) 的数组访问 + 检查 slot 是否还持有该 item。
	if cap(e.slotIndexMap) < n {
		e.slotIndexMap = make([]int, n)
	}
	e.slotIndexMap = e.slotIndexMap[:n]
	for si := range e.slotIndexMap {
		e.slotIndexMap[si] = -1
	}
	for si, s := range e.slots {
		if s.index >= 0 && s.index < n {
			e.slotIndexMap[s.index] = si
		}
	}

	// Step 2: 释放离开可见区的 slot
	for _, s := range e.slots {
		if s.index >= 0 && (s.index < start || s.index >= end) {
			e.slotIndexMap[s.index] = -1
			s.index = -1
		}
	}

	// Step 3: 分配可见区间 slot
	// 预分配 visibleSlots 容量（避免多次 grow）
	if cap(e.visibleSlots) < end-start {
		e.visibleSlots = make([]int, 0, end-start+e.overscan*2)
	}
	e.visibleSlots = e.visibleSlots[:0]

	for i := start; i < end; i++ {
		// O(1) 查 slotIndexMap
		found := -1
		if i < len(e.slotIndexMap) {
			si := e.slotIndexMap[i]
			if si >= 0 && si < len(e.slots) && e.slots[si].index == i {
				found = si
			}
		}

		if found >= 0 {
			s := e.slots[found]
			shouldUpdate := true
			if e.vl.ItemVersion != nil {
				v := e.vl.ItemVersion(i)
				if s.version == v && v != 0 {
					shouldUpdate = false
				}
				s.version = v
			}
			if shouldUpdate {
				w := e.vl.RenderItem(i)
				if w != nil && s.el != nil {
					s.el.Update(w)
					s.layoutWidth = 0 // 强制 Layout 重测
					e.buildSubtree(s.el)
				}
			}
			e.visibleSlots = append(e.visibleSlots, found)
			continue
		}

		// 找空闲 slot（不用再去签 slotIndexMap，已标 -1 即为空闲）
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
		s := e.slots[free]
		s.index = i
		if i < len(e.slotIndexMap) {
			e.slotIndexMap[i] = free
		}

		w := e.vl.RenderItem(i)
		if w == nil {
			continue
		}
		shouldUpdate := true
		if e.vl.ItemVersion != nil {
			v := e.vl.ItemVersion(i)
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
		e.visibleSlots = append(e.visibleSlots, free)
	}

	// Step 4: 刷新 childrenBuf（预分配容量避免 grow）
	if cap(e.childrenBuf) < len(e.visibleSlots) {
		e.childrenBuf = make([]Element, 0, len(e.visibleSlots))
	}
	e.childrenBuf = e.childrenBuf[:0]
	for _, si := range e.visibleSlots {
		if s := e.slots[si]; s != nil && s.el != nil {
			e.childrenBuf = append(e.childrenBuf, s.el)
		}
	}
}

// furthestSlot 返回距当前可见区间最远的 slot 索引（池满时替换用）。
func (e *VirtualListElement) furthestSlot(start, end int) int {
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

// repositionAll 重新定位所有 slot（前缀和 O(1) per slot）。
func (e *VirtualListElement) repositionAll() {
	for _, s := range e.slots {
		if s.index < 0 || s.el == nil {
			continue
		}
		off := e.itemOffset(s.index)
		s.el.SetPosition(types.Point{X: 0, Y: off - e.scrollOffset})
	}
}

// rebuildVisible 外部触发（ScrollTo / wheel）时调用，重新分配+定位可见项。
func (e *VirtualListElement) rebuildVisible() {
	e.assignVisibleSlots()
	w := e.size.Width
	if w <= 0 {
		w = 300
	}
	e.slotLayoutCtx.Constraints = layout.BoxConstraints{
		MinWidth:  w,
		MaxWidth:  w,
		MinHeight: 0,
		MaxHeight: 1 << 20,
	}
	for _, s := range e.slots {
		if s.index < 0 || s.el == nil {
			continue
		}
		if s.layoutWidth == w && s.height > 0 {
			versionSame := true
			if e.vl.ItemVersion != nil {
				if e.vl.ItemVersion(s.index) != s.version {
					versionSame = false
				}
			}
			if versionSame {
				continue
			}
		}
		s.el.Layout(&e.slotLayoutCtx)
		s.layoutWidth = w
	}
	e.repositionAll()
}

// handleAutoScroll 受控滚到底部（带阻尼防高频拉回）。
func (e *VirtualListElement) handleAutoScroll() {
	if e.vl.AutoScrollToBottom && !e.autoScrolledToBottom && e.vl.ItemCount > 0 && e.maxScroll > 0 {
		e.autoScrolledToBottom = true
		e.lastScrollToBottom = e.vl.ScrollToBottomToken
		e.vl.AutoScrolledAlready = true
		e.vl.LastScrollToBottomToken = e.vl.ScrollToBottomToken
		maxScr := vlMaxF(0, e.maxScroll)
		if e.scrollOffset < maxScr {
			e.scrollOffset = maxScr
			e.repositionAll()
		}
		return
	}

	if e.vl.ScrollToBottomToken == e.lastScrollToBottom {
		return
	}
	e.lastScrollToBottom = e.vl.ScrollToBottomToken
	e.vl.LastScrollToBottomToken = e.vl.ScrollToBottomToken
	if !e.isNearBottom() {
		return
	}
	now := time.Now()
	minInt := e.vl.AutoScrollMinInterval
	if minInt <= 0 {
		minInt = time.Second
	}
	if now.Sub(e.lastUserScroll) > 3*time.Second && now.Sub(e.lastAutoScroll) >= minInt {
		maxScr := vlMaxF(0, e.maxScroll)
		if e.scrollOffset < maxScr {
			e.scrollOffset = maxScr
			e.lastAutoScroll = now
			e.repositionAll()
		}
	}
}

// isNearBottom 是否接近底部（8px）。
func (e *VirtualListElement) isNearBottom() bool {
	if e.maxScroll <= 0 {
		return true
	}
	return e.maxScroll-e.scrollOffset < 8
}

// fireScroll 触发滚动回调（仅 scrollOffset 实际变化时）。
func (e *VirtualListElement) fireScroll() {
	if e.vl.OnScroll != nil && e.scrollOffset != e.lastFiredOffset {
		e.lastFiredOffset = e.scrollOffset
		e.vl.OnScroll(e.scrollOffset)
	}
}

// ─── Paint ──────────────────────────────────────────
func (e *VirtualListElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()

	cvs.Save()
	cvs.ClipRect(pos.X, pos.Y, e.size.Width, e.size.Height)

	w := e.size.Width
	if w <= 0 {
		w = 300
	}

	for _, s := range e.slots {
		if s.index < 0 || s.el == nil || s.height <= 0 {
			continue
		}
		// 可见性检查（前缀和 O(1)）
		itemOff := e.itemOffset(s.index)
		if itemOff+s.height < e.scrollOffset || itemOff > e.scrollOffset+e.viewportH+100 {
			continue
		}
		screenY := pos.Y + (itemOff - e.scrollOffset)
		s.el.Paint(cvs, types.Point{X: pos.X, Y: screenY})
	}

	cvs.Restore()

	if e.maxScroll > 0 {
		e.paintScrollbar(cvs, pos)
	}
}

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

// ─── 事件处理 ──────────────────────────────────────
func (e *VirtualListElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseWheel:
		me := ev.(*event.MouseEvent)
		dy := me.DeltaY * wheelStep
		old := e.scrollOffset
		e.scrollOffset = vlClamp(e.scrollOffset-dy, 0, vlMaxF(0, e.maxScroll))
		e.externalScrollSet = false
		if e.scrollOffset != old {
			e.rebuildVisible()
			e.userScrolled = true
			e.lastUserScroll = time.Now()
			e.fireScroll()
			e.userScrolled = false
			if e.vl.OnReachEnd != nil && e.scrollOffset >= e.maxScroll-e.vl.ItemHeight*2 {
				e.vl.OnReachEnd()
			}
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

// ─── Update ──────────────────────────────────────────
func (e *VirtualListElement) Update(newWidget Widget) {
	if nv, ok := newWidget.(*VirtualList); ok {
		oldCount := e.vl.ItemCount
		e.vl = nv
		e.BaseElement.widget = newWidget
		e.overscan = nv.Overscan
		if e.overscan <= 0 {
			e.overscan = 5
		}

		// ★ 从 widget 同步滚动偏移（chat.go 通过 ScrollOffset 字段传回缓存的滚动位置，防鼠标移动/点击重建后跳回顶部）
		if nv.ScrollOffset >= 0 {
				e.scrollOffset = nv.ScrollOffset
				e.externalScrollSet = true
		}

		// ItemCount 变化 → 清空槽池 + 高度缓存
		if nv.ItemCount != oldCount {
			for _, s := range e.slots {
				if s.el != nil {
					s.el.Unmount()
				}
			}
			e.slots = nil
			e.visibleSlots = nil
			e.childrenBuf = nil
			e.itemHeightMap = make(map[int]float64)
			e.ensureSlotPool()
		}

		e.invalidatePrefix()

		// 同步跨重建持久化标记
		e.autoScrolledToBottom = nv.AutoScrolledAlready
		e.lastScrollToBottom = nv.LastScrollToBottomToken
	}
}

// ─── 滚动条拖拽 ──────────────────────────────────────
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
	e.scrollOffset = vlClamp(e.dragStartScrollY+delta, 0, vlMaxF(0, e.maxScroll))
	e.externalScrollSet = false
	e.lastUserScroll = time.Now()
	e.repositionAll()
	e.userScrolled = true
	e.fireScroll()
	e.userScrolled = false
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// buildSubtree 递归构建 Element 子树。
func (e *VirtualListElement) buildSubtree(el Element) {
	children := el.Build()
	for _, child := range children {
		e.buildSubtree(child)
	}
}

// ─── 工具函数 ──────────────────────────────────────
const wheelStep = 50.0

func vlMaxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func vlMinF(a, b float64) float64 {
	if a < b {
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
