package widget

import (
	"reflect"
	"sync"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Overlay 浮层定位系统(地基) ──────────────────────────────────
// 为 Tooltip / Popover / Select / Dropdown / DatePicker 等"挂在锚点旁边的浮层"
// 提供统一的定位与渲染机制：
//   - Placement   12 方位定位(对齐 Element Plus 的 placement)
//   - 边缘翻转     空间不足时主方向自动翻转(top↔bottom / left↔right)
//   - 视口约束     次方向越界时平移回视口内(shift)
//   - OverlayHost  把浮层作为真正的 Element 渲染在主内容之上(正确布局/绘制/命中)
//   - OverlayManager 全局浮层栈，Show/Hide 触发宿主重新布局

// Placement 浮层相对锚点的方位。
type Placement int

const (
	PlacementTop Placement = iota
	PlacementTopStart
	PlacementTopEnd
	PlacementBottom
	PlacementBottomStart
	PlacementBottomEnd
	PlacementLeft
	PlacementLeftStart
	PlacementLeftEnd
	PlacementRight
	PlacementRightStart
	PlacementRightEnd
)

// mainDir 主方向；crossAlign 次方向对齐。
type mainDir int

const (
	mTop mainDir = iota
	mBottom
	mLeft
	mRight
)

type crossAlign int

const (
	cStart crossAlign = iota
	cCenter
	cEnd
)

func splitPlacement(p Placement) (mainDir, crossAlign) {
	switch p {
	case PlacementTop:
		return mTop, cCenter
	case PlacementTopStart:
		return mTop, cStart
	case PlacementTopEnd:
		return mTop, cEnd
	case PlacementBottom:
		return mBottom, cCenter
	case PlacementBottomStart:
		return mBottom, cStart
	case PlacementBottomEnd:
		return mBottom, cEnd
	case PlacementLeft:
		return mLeft, cCenter
	case PlacementLeftStart:
		return mLeft, cStart
	case PlacementLeftEnd:
		return mLeft, cEnd
	case PlacementRight:
		return mRight, cCenter
	case PlacementRightStart:
		return mRight, cStart
	case PlacementRightEnd:
		return mRight, cEnd
	default:
		return mBottom, cCenter
	}
}

func joinPlacement(m mainDir, c crossAlign) Placement {
	switch m {
	case mTop:
		return []Placement{PlacementTopStart, PlacementTop, PlacementTopEnd}[c]
	case mBottom:
		return []Placement{PlacementBottomStart, PlacementBottom, PlacementBottomEnd}[c]
	case mLeft:
		return []Placement{PlacementLeftStart, PlacementLeft, PlacementLeftEnd}[c]
	default:
		return []Placement{PlacementRightStart, PlacementRight, PlacementRightEnd}[c]
	}
}

// crossX 水平次方向对齐(用于 top/bottom)。
func crossX(c crossAlign, a types.Rect, sz types.Size) float64 {
	switch c {
	case cStart:
		return a.X
	case cEnd:
		return a.X + a.Width - sz.Width
	default:
		return a.X + (a.Width-sz.Width)/2
	}
}

// crossY 垂直次方向对齐(用于 left/right)。
func crossY(c crossAlign, a types.Rect, sz types.Size) float64 {
	switch c {
	case cStart:
		return a.Y
	case cEnd:
		return a.Y + a.Height - sz.Height
	default:
		return a.Y + (a.Height-sz.Height)/2
	}
}

// placeAt 按主方向 + 次方向算出浮层左上角(全局坐标)。
func placeAt(m mainDir, c crossAlign, a types.Rect, sz types.Size, gap float64) types.Point {
	switch m {
	case mTop:
		return types.Point{X: crossX(c, a, sz), Y: a.Y - sz.Height - gap}
	case mBottom:
		return types.Point{X: crossX(c, a, sz), Y: a.Y + a.Height + gap}
	case mLeft:
		return types.Point{X: a.X - sz.Width - gap, Y: crossY(c, a, sz)}
	default: // mRight
		return types.Point{X: a.X + a.Width + gap, Y: crossY(c, a, sz)}
	}
}

// flipMain 主方向空间不足且对面够用时翻转。
func flipMain(m mainDir, a types.Rect, sz types.Size, vp types.Size, gap float64) mainDir {
	switch m {
	case mTop:
		if a.Y-sz.Height-gap < 0 && a.Y+a.Height+gap+sz.Height <= vp.Height {
			return mBottom
		}
	case mBottom:
		if a.Y+a.Height+gap+sz.Height > vp.Height && a.Y-sz.Height-gap >= 0 {
			return mTop
		}
	case mLeft:
		if a.X-sz.Width-gap < 0 && a.X+a.Width+gap+sz.Width <= vp.Width {
			return mRight
		}
	case mRight:
		if a.X+a.Width+gap+sz.Width > vp.Width && a.X-sz.Width-gap >= 0 {
			return mLeft
		}
	}
	return m
}

// shiftIntoViewport 把浮层平移回视口内(次方向越界纠正)。
func shiftIntoViewport(p types.Point, sz types.Size, vp types.Size) types.Point {
	if p.X+sz.Width > vp.Width {
		p.X = vp.Width - sz.Width
	}
	if p.Y+sz.Height > vp.Height {
		p.Y = vp.Height - sz.Height
	}
	if p.X < 0 {
		p.X = 0
	}
	if p.Y < 0 {
		p.Y = 0
	}
	return p
}

// computeOverlayPosition 给定锚点矩形/浮层尺寸/期望方位/视口，
// 返回浮层左上角(全局坐标)与实际生效的方位(可能因翻转改变)。
func computeOverlayPosition(anchor types.Rect, content types.Size, placement Placement, viewport types.Size, gap float64) (types.Point, Placement) {
	m, c := splitPlacement(placement)
	m = flipMain(m, anchor, content, viewport, gap)
	pos := placeAt(m, c, anchor, content, gap)
	pos = shiftIntoViewport(pos, content, viewport)
	return pos, joinPlacement(m, c)
}

// parsePlacement 把 Element Plus 风格的字符串解析为 Placement(用于声明式)。
func parsePlacement(s string) Placement {
	switch s {
	case "top":
		return PlacementTop
	case "top-start":
		return PlacementTopStart
	case "top-end":
		return PlacementTopEnd
	case "bottom":
		return PlacementBottom
	case "bottom-start":
		return PlacementBottomStart
	case "bottom-end":
		return PlacementBottomEnd
	case "left":
		return PlacementLeft
	case "left-start":
		return PlacementLeftStart
	case "left-end":
		return PlacementLeftEnd
	case "right":
		return PlacementRight
	case "right-start":
		return PlacementRightStart
	case "right-end":
		return PlacementRightEnd
	default:
		return PlacementBottom
	}
}

// ─── OverlayManager 全局浮层栈 ──────────────────────────────────

// OverlayEntry 一个活跃浮层。
type OverlayEntry struct {
	AnchorRect     types.Rect // 锚点的全局矩形
	Content        Widget     // 浮层内容
	Placement      Placement  // 期望方位
	Gap            float64    // 与锚点的间距(0→默认 8)
	Fullscreen     bool       // 全屏模态(铺满视口、不走 anchor 定位；Dialog/Drawer/Loading 用)
	Toast          bool       // toast 模式(顶部居中/右上角堆叠，非锚点；Message/Notification 用)
	ToastPos       string     // "top"(默认) / "top-right"
	Transition     string     // 进出场过渡: ""(无) / "fade" / "zoom" / "slide-top" / "slide-right" / "slide-bottom"
	OnClickOutside func()     // 点击浮层与锚点之外时回调(用于关闭)

	id                int
	pos               types.Point // 解析后的全局左上角
	size              types.Size  // 浮层实际尺寸
	resolvedPlacement Placement   // 翻转后的实际方位
	enterProgress     float64     // 进出场进度(0=隐藏，1=完全显示)
	exiting           bool        // 是否正在退场动画中
}

// ResolvedPlacement 返回翻转后实际生效的方位(供箭头朝向等使用)。
func (en *OverlayEntry) ResolvedPlacement() Placement { return en.resolvedPlacement }

var (
	overlayMu        sync.Mutex
	overlayEntries   []*OverlayEntry
	overlayNextID    int
	OnOverlayChanged func() // 由 app 接到 Pipeline.MarkNeedsLayout
)

// ShowOverlay 压入一个浮层，返回其 id；若设了 Transition 则播放进场动画。
func ShowOverlay(e *OverlayEntry) int {
	overlayMu.Lock()
	overlayNextID++
	e.id = overlayNextID
	if e.Gap == 0 {
		e.Gap = 8
	}
	if e.Transition == "" {
		e.enterProgress = 1 // 无过渡：立即完全显示
	} else {
		e.enterProgress = 0
	}
	overlayEntries = append(overlayEntries, e)
	overlayMu.Unlock()
	notifyOverlayChanged()
	if e.Transition != "" {
		startOverlayEnter(e)
	}
	return e.id
}

// HideOverlay 按 id 关闭浮层；若设了 Transition 则先播退场动画再移除。
func HideOverlay(id int) {
	overlayMu.Lock()
	var target *OverlayEntry
	for _, e := range overlayEntries {
		if e.id == id {
			target = e
			break
		}
	}
	overlayMu.Unlock()
	if target == nil {
		return
	}
	if target.Transition != "" && !target.exiting {
		target.exiting = true
		startOverlayExit(target, id)
		return
	}
	removeOverlay(id)
}

// removeOverlay 立即从栈移除(退场动画结束 / 无过渡时)。
func removeOverlay(id int) {
	overlayMu.Lock()
	for i, e := range overlayEntries {
		if e.id == id {
			overlayEntries = append(overlayEntries[:i], overlayEntries[i+1:]...)
			break
		}
	}
	overlayMu.Unlock()
	notifyOverlayChanged()
}

// startOverlayEnter 播放进场动画(enterProgress 0→1)。
func startOverlayEnter(e *OverlayEntry) {
	ctrl := animation.NewController(240*time.Millisecond, animation.EaseOutCubic)
	ctrl.OnUpdate = func(v float64) {
		e.enterProgress = v
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
	}
	ctrl.Start()
}

// startOverlayExit 播放退场动画(enterProgress→0)，结束后移除。
func startOverlayExit(e *OverlayEntry, id int) {
	from := e.enterProgress
	ctrl := animation.NewController(180*time.Millisecond, animation.EaseInQuad)
	ctrl.OnUpdate = func(v float64) {
		e.enterProgress = from * (1 - v)
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
	}
	ctrl.OnDone = func() { removeOverlay(id) }
	ctrl.Start()
}

// transitionAt 按过渡类型与进度 p(0→1) 返回 (opacity, translateX, translateY, scale)。
func transitionAt(t string, p float64) (op, tx, ty, scale float64) {
	op, tx, ty, scale = p, 0, 0, 1
	switch t {
	case "zoom":
		scale = 0.9 + 0.1*p
	case "slide-top":
		ty = -18 * (1 - p)
	case "slide-right":
		tx = 28 * (1 - p)
	case "slide-bottom":
		ty = 18 * (1 - p)
	}
	return
}

// UpdateOverlayAnchor 更新某浮层的锚点矩形(锚点移动/滚动时)。
func UpdateOverlayAnchor(id int, rect types.Rect) {
	overlayMu.Lock()
	for _, e := range overlayEntries {
		if e.id == id {
			e.AnchorRect = rect
			break
		}
	}
	overlayMu.Unlock()
	notifyOverlayChanged()
}

// activeOverlays 返回逻辑活跃(非退场中)的浮层快照——供测试/逻辑判断"显示数"。
func activeOverlays() []*OverlayEntry {
	overlayMu.Lock()
	defer overlayMu.Unlock()
	out := make([]*OverlayEntry, 0, len(overlayEntries))
	for _, e := range overlayEntries {
		if !e.exiting {
			out = append(out, e)
		}
	}
	return out
}

// CloseAnchoredOverlays 关闭所有锚点型浮层(下拉/日历/级联/Tooltip/Popover)。
// 供 ScrollView 滚动时调用——锚点随内容滚走、浮层应收起(主流交互)；
// 全屏模态(Dialog/Drawer/Loading)与 toast(Message/Notification)不受影响。
// 通过各浮层的 OnClickOutside 回调关闭，使触发器同步 open 状态(锁外执行避免重入)。
func CloseAnchoredOverlays() {
	overlayMu.Lock()
	var cbs []func()
	for _, e := range overlayEntries {
		if e.Fullscreen || e.Toast || e.exiting {
			continue
		}
		if cb := e.OnClickOutside; cb != nil {
			cbs = append(cbs, cb)
		} else {
			id := e.id
			cbs = append(cbs, func() { HideOverlay(id) })
		}
	}
	overlayMu.Unlock()
	for _, cb := range cbs {
		cb()
	}
}

// renderOverlays 返回全部浮层快照(含退场中)——供渲染：退场动画期间浮层仍需绘制。
func renderOverlays() []*OverlayEntry {
	overlayMu.Lock()
	defer overlayMu.Unlock()
	out := make([]*OverlayEntry, len(overlayEntries))
	copy(out, overlayEntries)
	return out
}

func notifyOverlayChanged() {
	if OnOverlayChanged != nil {
		OnOverlayChanged()
	}
}

// ─── OverlayHost 浮层宿主 ──────────────────────────────────────

// OverlayHost 包裹应用内容，并在其上渲染全局浮层栈。
// 通常作为顶层 Widget 包裹根内容(位于 0,0)。
type OverlayHost struct {
	StatelessWidget
	Child Widget
}

// NewOverlayHost 用给定子内容创建浮层宿主。
func NewOverlayHost(child Widget) *OverlayHost { return &OverlayHost{Child: child} }

// CreateElement 创建 OverlayHostElement。
func (h *OverlayHost) CreateElement() Element {
	return &OverlayHostElement{BaseElement: BaseElement{widget: h}, host: h}
}

type overlayInstance struct {
	entry *OverlayEntry
	el    Element
}

// OverlayHostElement 浮层宿主运行时 Element。
type OverlayHostElement struct {
	BaseElement
	host     *OverlayHost
	childEl  Element
	overlays []overlayInstance
}

// Build 构建主内容子树 + 各浮层子树。
func (e *OverlayHostElement) Build() []Element {
	kids := make([]Element, 0, 4)
	if e.host.Child != nil {
		// reconcile：复用主内容子树，保留其运行时状态（如 ScrollView 滚动位置）
		if e.childEl != nil && reflect.TypeOf(e.childEl.Widget()) == reflect.TypeOf(e.host.Child) {
			e.childEl.Update(e.host.Child)
		} else {
			if e.childEl != nil {
				e.childEl.Unmount()
			}
			e.childEl = CreateElementFor(e.host.Child)
			e.childEl.Mount(e, 0)
		}
		kids = append(kids, e.childEl)
	}
	// 浮层是动态的，每次重建
	e.overlays = e.overlays[:0]
	for i, entry := range renderOverlays() {
		if entry.Content == nil {
			continue
		}
		oel := CreateElementFor(entry.Content)
		oel.Mount(e, i+1)
		e.overlays = append(e.overlays, overlayInstance{entry: entry, el: oel})
		kids = append(kids, oel)
	}
	e.children = kids
	return kids
}

// Layout 主内容铺满；浮层先测尺寸再按方位定位(翻转/平移)。
func (e *OverlayHostElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  ctx.Constraints.MaxWidth,
		Height: ctx.Constraints.MaxHeight,
	})

	if e.childEl != nil {
		e.childEl.Layout(&layout.LayoutContext{Constraints: ctx.Constraints})
		e.childEl.SetPosition(types.Point{})
	}

	vp := types.Size{Width: e.size.Width, Height: e.size.Height}
	host := e.Offset() // 宿主全局偏移(顶层时为 0,0)
	toastStack := map[string]float64{} // 各 toast 位置的累计堆叠高度
	for _, inst := range e.overlays {
		// 全屏模态：铺满视口(tight 约束)，不走 anchor 定位
		if inst.entry.Fullscreen {
			inst.el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
				MinWidth: e.size.Width, MaxWidth: e.size.Width,
				MinHeight: e.size.Height, MaxHeight: e.size.Height,
			}})
			inst.entry.pos = host // 全局左上(顶层宿主为 0,0)
			inst.entry.size = e.size
			inst.el.SetPosition(types.Point{X: 0, Y: 0})
			continue
		}
		// Toast：五种锚位(top/top-left/top-right/bottom-left/bottom-right)，同位置堆叠，非锚点定位
		if inst.entry.Toast {
			res := inst.el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
				MinWidth: 0, MaxWidth: e.size.Width, MinHeight: 0, MaxHeight: e.size.Height,
			}})
			sz := res.Size
			const tMargin = 16.0
			posKey := inst.entry.ToastPos
			stacked := toastStack[posKey]
			var x, y float64
			switch posKey {
			case "top-left", "bottom-left":
				x = tMargin
			case "top-right", "bottom-right":
				x = vp.Width - sz.Width - tMargin
			default: // top(居中)
				x = (vp.Width - sz.Width) / 2
			}
			switch posKey {
			case "bottom", "bottom-left", "bottom-right":
				y = vp.Height - tMargin - stacked - sz.Height // 底部向上堆叠
			default: // top* 顶部向下堆叠
				y = tMargin + stacked
			}
			inst.entry.pos = types.Point{X: x, Y: y}
			inst.entry.size = sz
			inst.el.SetPosition(types.Point{X: x - host.X, Y: y - host.Y})
			toastStack[posKey] += sz.Height + 12
			continue
		}
		res := inst.el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: e.size.Width, MinHeight: 0, MaxHeight: e.size.Height,
		}})
		gap := inst.entry.Gap
		if gap == 0 {
			gap = 8
		}
		pos, actual := computeOverlayPosition(inst.entry.AnchorRect, res.Size, inst.entry.Placement, vp, gap)
		inst.entry.pos = pos
		inst.entry.size = res.Size
		inst.entry.resolvedPlacement = actual
		// 浮层是宿主的子：设局部坐标，Offset() 会自动累加宿主偏移得到全局
		inst.el.SetPosition(types.Point{X: pos.X - host.X, Y: pos.Y - host.Y})
	}

	return layout.LayoutResult{Size: e.size}
}

// Paint 先画主内容，再叠加浮层。
func (e *OverlayHostElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.childEl != nil {
		e.childEl.Paint(cvs, offset)
	}
	for _, inst := range e.overlays {
		en := inst.entry
		p := en.enterProgress
		if en.Transition == "" {
			p = 1
		}
		op, tx, ty, sc := transitionAt(en.Transition, p)
		cvs.Save()
		// 缩放(zoom)：以浮层中心为基准
		if sc != 1 {
			cx := en.pos.X + en.size.Width/2
			cy := en.pos.Y + en.size.Height/2
			cvs.Translate(cx, cy)
			cvs.Scale(sc, sc)
			cvs.Translate(-cx, -cy)
		}
		// 平移(slide)
		if tx != 0 || ty != 0 {
			cvs.Translate(tx, ty)
		}
		// 透明度(fade)：图层 alpha 包裹
		if op < 1 {
			cvs.SaveLayerAlpha(op)
			inst.el.Paint(cvs, offset)
			cvs.Restore()
		} else {
			inst.el.Paint(cvs, offset)
		}
		cvs.Restore()
	}
}

// HandleEvent 点击浮层与锚点之外时触发各浮层的 OnClickOutside(用于关闭)。
func (e *OverlayHostElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeMouseDown {
		if me, ok := ev.(*event.MouseEvent); ok {
			for _, inst := range e.overlays {
				en := inst.entry
				inOverlay := me.X >= en.pos.X && me.X <= en.pos.X+en.size.Width &&
					me.Y >= en.pos.Y && me.Y <= en.pos.Y+en.size.Height
				inAnchor := me.X >= en.AnchorRect.X && me.X <= en.AnchorRect.X+en.AnchorRect.Width &&
					me.Y >= en.AnchorRect.Y && me.Y <= en.AnchorRect.Y+en.AnchorRect.Height
				if !inOverlay && !inAnchor && en.OnClickOutside != nil {
					en.OnClickOutside()
				}
			}
		}
	}
	return false
}
