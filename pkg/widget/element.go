package widget

import (
	"reflect"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/types"
)

// BaseElement 是 Element 的基类，提供默认实现
type BaseElement struct {
	widget    Widget
	parent    Element
	children  []Element
	size      types.Size
	position  types.Point
	offset    types.Point
	slotIndex int
	dirty     bool

	// widgetType 缓存 Widget 的动态类型，避免每帧 Build 时反复调用 reflect.TypeOf。
	// 在 CreateElementFor/Mount/Update 中设置。WidgetType() 返回缓存值，零值走 reflect.TypeOf 兜底。
	widgetType reflect.Type
}

// Widget 返回关联的 Widget
func (e *BaseElement) Widget() Widget { return e.widget }

// WidgetType 返回缓存的 Widget 动态类型（零值则按需反射）。
// 用于 Element 间类型比较，避免重复的 reflect.TypeOf 调用。
func (e *BaseElement) WidgetType() reflect.Type {
	if e.widgetType != nil {
		return e.widgetType
	}
	if e.widget != nil {
		e.widgetType = reflect.TypeOf(e.widget)
	}
	return e.widgetType
}

// updateWidgetType 更新 Widget 及其类型缓存。由 CreateElementFor 和 Update 调用。
func (e *BaseElement) updateWidgetType(w Widget) {
	e.widget = w
	e.widgetType = reflect.TypeOf(w)
}

// Parent 返回父 Element
func (e *BaseElement) Parent() Element { return e.parent }

// Children 返回子 Element 列表
func (e *BaseElement) Children() []Element { return e.children }

// Size 返回尺寸
func (e *BaseElement) Size() types.Size { return e.size }

// Position 返回位置
func (e *BaseElement) Position() types.Point { return e.position }

// SetPosition 设置位置
func (e *BaseElement) SetPosition(pos types.Point) { e.position = pos }

// Offset 返回全局偏移量
func (e *BaseElement) Offset() types.Point {
	if e.parent == nil {
		return e.position
	}
	parentOffset := e.parent.Offset()
	return types.Point{
		X: parentOffset.X + e.position.X,
		Y: parentOffset.Y + e.position.Y,
	}
}

// Mount 挂载 Element
func (e *BaseElement) Mount(parent Element, slotIndex int) {
	e.parent = parent
	e.slotIndex = slotIndex
	e.dirty = true
}

// Unmount 卸载 Element
func (e *BaseElement) Unmount() {
	e.parent = nil
	for _, child := range e.children {
		child.Unmount()
	}
	e.children = nil
}

// Update 更新 Widget 配置
func (e *BaseElement) Update(newWidget Widget) {
	e.updateWidgetType(newWidget)
	e.dirty = true
}

// Build 构建子 Element（由子类实现）
func (e *BaseElement) Build() []Element { return nil }

// Layout 执行布局
func (e *BaseElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(e.size)
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制（由子类实现）
func (e *BaseElement) Paint(c canvas.Canvas, offset types.Point) {}

// HandleEvent 处理事件
func (e *BaseElement) HandleEvent(ev event.Event) bool { return false }

// MarkNeedsLayout 标记需要重新布局
func (e *BaseElement) MarkNeedsLayout() {
	e.dirty = true
	if e.parent != nil {
		e.parent.MarkNeedsLayout()
	}
}

// MarkNeedsPaint 标记需要重绘
// 同时通知 Pipeline 需要重绘，确保视觉状态变化能反映到屏幕上
func (e *BaseElement) MarkNeedsPaint() {
	e.dirty = true
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// Focus 获取焦点（由子类重写以实现具体行为）
func (e *BaseElement) Focus() {
	// 默认向父 Element 冒泡焦点请求
	// 实际焦点管理由 App.RequestFocus 处理
}

// Blur 失去焦点（由子类重写）
func (e *BaseElement) Blur() {}

// IsFocused 检查是否拥有焦点
func (e *BaseElement) IsFocused() bool { return false }

// SubtreeStableChecker 由 buildTree 检查：Element 的子树自上次 Build 后无结构变化。
// 实现了此接口的 Element，buildTree 在对它调用 Build() 后检查 Stable() 返回值，
// 若为 true 则跳过其子树的递归遍历。
type SubtreeStableChecker interface {
	Stable() bool
}

// StatelessElement 无状态 Element
type StatelessElement struct {
	BaseElement
	built  bool
	stable bool // 上次 Build() 后子树结构无变化，buildTree 可跳过递归
}

// createStatelessElement 工厂函数
func createStatelessElement(w Widget) *StatelessElement {
	return &StatelessElement{BaseElement: BaseElement{widget: w}}
}

// Stable 返回子树结构是否稳定。
func (e *StatelessElement) Stable() bool { return e.stable }

// Update 更新 Widget 配置并标记需要重建
func (e *StatelessElement) Update(newWidget Widget) {
	e.updateWidgetType(newWidget)
	e.built = false
	e.stable = false
	e.dirty = true
}

// Build 构建子 Element
// 如果 Widget 实现了 Build(BuildContext) Widget 接口，调用它获取子控件。
// 当子控件类型匹配时复用已有的 Element 以保持状态。
func (e *StatelessElement) Build() []Element {
	if e.built {
		e.stable = true // 缓存命中 → 子树结构稳定，buildTree 可跳过递归
		return e.children
	}
	e.built = true
	e.stable = false

	// 检查是否实现了 Builder 接口
	if builder, ok := e.widget.(interface{ Build(BuildContext) Widget }); ok {
		w := builder.Build(BuildContext{Element: e})
		if w != nil {
			// 尝试复用已有的子 Element（类型匹配时）
			if len(e.children) > 0 && e.children[0].WidgetType() == reflect.TypeOf(w) {
				e.children[0].Update(w)
			} else {
				// 类型不匹配，卸载旧的创建新的
				for _, child := range e.children {
					child.Unmount()
				}
				child := CreateElementFor(w)
				child.Mount(e, 0)
				e.children = []Element{child}
			}
			return e.children
		}
	}

	// 没有子控件，清理旧的
	for _, child := range e.children {
		child.Unmount()
	}
	e.children = nil
	return nil
}

// Layout 委托给子 Element
func (e *StatelessElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if len(e.children) > 0 {
		result := e.children[0].Layout(ctx)
		e.size = result.Size
		return result
	}
	e.size = ctx.Constraints.Constrain(e.size)
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制子 Element
func (e *StatelessElement) Paint(cvs canvas.Canvas, offset types.Point) {
	for _, child := range e.children {
		child.Paint(cvs, offset)
	}
}

// globalRebuildSeq 在全局性变化（如语言切换）时自增，使所有 StatefulElement
// 在下一次 buildTree 强制重建一次（绕过脏检查缓存）。
var globalRebuildSeq int

// BumpRebuild 触发一次全局强制重建（语言切换等场景，配合 Pipeline 重新布局）。
func BumpRebuild() { globalRebuildSeq++ }

// StatefulElement 有状态 Element
type StatefulElement struct {
	BaseElement
	state          State
	child          Element
	buildDirty     bool // SetState/Update 后置位；仅脏时才重新 state.Build（减少每帧重建/GC）
	lastRebuildSeq int  // 上次重建时的 globalRebuildSeq，用于响应全局强制重建
	builtOnce      bool    // 首次 Build 已完成（防 Stable 误判跳过子树递归）
}

// createStatefulElement 工厂函数
func createStatefulElement(w Widget) *StatefulElement {
	return &StatefulElement{BaseElement: BaseElement{widget: w}}
}

func (e *StatefulElement) Stable() bool {
	return e.builtOnce && !e.buildDirty
}
// CreateState 创建并初始化状态
func (e *StatefulElement) CreateState() {
	if e.state != nil {
		return
	}
	// 通过接口断言查找 CreateState 方法
	if creator, ok := e.widget.(interface{ CreateState() State }); ok {
		e.state = creator.CreateState()
		if e.state != nil {
			if setter, ok := e.state.(interface {
				SetWidget(Widget)
				SetElement(*StatefulElement)
				SetContext(BuildContext)
			}); ok {
				setter.SetWidget(e.widget)
				setter.SetContext(BuildContext{Element: e})
				setter.SetElement(e)
			}
			e.state.InitState()
		}
	}
}

// Build 构建子 Element（调用 state.Build）
// 当 Widget 类型匹配时，复用已有的子 Element 以保留状态。
func (e *StatefulElement) Build() []Element {
	if e.state == nil {
		e.CreateState()
	}
	if e.state == nil {
		return nil
	}

	// 状态未脏、已构建过、且未发生全局强制重建：复用缓存子树，避免动画/高频
	// 重建时每帧重复 state.Build（显著减少分配与 GC）。
	if e.builtOnce && e.child != nil && !e.buildDirty && e.lastRebuildSeq == globalRebuildSeq {
		e.children = []Element{e.child}
		return e.children
	}
	e.buildDirty = false
	e.builtOnce = true
	e.lastRebuildSeq = globalRebuildSeq

	// 调用 state.Build 获取子控件
	w := e.state.Build(BuildContext{Element: e})
	if w == nil {
		e.child = nil
		e.children = nil
		return nil
	}

	// 尝试复用已有的子 Element（当 Widget 类型匹配时）
	if e.child != nil && e.child.WidgetType() == reflect.TypeOf(w) {
		e.child.Update(w)
		e.children = []Element{e.child}
		return e.children
	}

	// 类型不匹配或无现有子 Element，创建新的
	child := CreateElementFor(w)
	child.Mount(e, 0)
	e.child = child
	e.children = []Element{child}
	return e.children
}

// Layout 委托给子 Element
func (e *StatefulElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		result := e.child.Layout(ctx)
		e.size = result.Size
		return result
	}
	e.size = ctx.Constraints.Constrain(e.size)
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制子 Element
func (e *StatefulElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child != nil {
		e.child.Paint(cvs, offset)
	}
}

// Update 更新 StatefulElement
func (e *StatefulElement) Update(newWidget Widget) {
	old := e.widget
	e.BaseElement.Update(newWidget)
	if e.state != nil {
		// 同步 State 的 Widget 引用为最新配置（State.Widget() 返回最新）
		if setter, ok := e.state.(interface{ SetWidget(Widget) }); ok {
			setter.SetWidget(newWidget)
		}
		// 通知关心配置变化的 State（如隐式动画）以启动过渡
		if u, ok := e.state.(interface{ DidUpdateWidget(old Widget) }); ok {
			u.DidUpdateWidget(old)
		}
		// 仅当配置真的变化才标脏重建：与旧配置深度相等则复用缓存子树，避免父级重建时
		// 无谓地级联重建所有未变的子面板（拖动改尺寸/动画等高频 SetState 的卡顿根因——
		// 一次 shell.SetState 原本会连锁重建对话面板等整棵树、重渲全部 Markdown）。
		// 组件自身 SetState 走 Rebuild 直接置脏，状态变化照常重建，不受此影响。
		if old == nil || !configEqual(old, newWidget) {
			e.buildDirty = true
		}
	}
}

// configEqual 两个 Widget 配置是否深度相等。含非 nil 函数字段者必不等（reflect.DeepEqual
// 语义：函数仅 nil==nil）→ 带回调的组件照常重建；无字段/纯数据且未变者（如 companion 各
// 面板 struct{StatefulWidget}）才复用缓存、跳过级联重建。
func configEqual(a, b Widget) bool {
	return reflect.DeepEqual(a, b)
}

// Rebuild 重建子树（在 SetState 时调用）
func (e *StatefulElement) Rebuild() {
	if e.state == nil {
		return
	}
	// 标记需要重建；实际 state.Build 延迟到下一次 buildTree 的 Build()，
	// 避免 SetState 路径与 buildTree 路径重复构建（动画高频重建时尤其重要）。
	e.buildDirty = true

	// 
	// key fix: propagate buildDirty up parent chain so buildTree can reach all dirty nodes
	for p := e.parent; p != nil; p = p.Parent() {
		if se, ok := p.(*StatefulElement); ok {
			se.buildDirty = true
		}
	}

	// notify need layout and repaint
	e.MarkNeedsLayout()
	e.MarkNeedsPaint()
}

// IsFocusable 返回此元素是否可以获取焦点
func (e *StatefulElement) IsFocusable() bool {
	// 默认有状态的 StatefulWidget 不可聚焦，由具体子类重写
	return false
}
