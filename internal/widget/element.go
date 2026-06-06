package widget

import (
	"reflect"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
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
}

// Widget 返回关联的 Widget
func (e *BaseElement) Widget() Widget { return e.widget }

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
	e.widget = newWidget
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

// StatelessElement 无状态 Element
type StatelessElement struct {
	BaseElement
	built bool
}

// createStatelessElement 工厂函数
func createStatelessElement(w Widget) *StatelessElement {
	return &StatelessElement{BaseElement: BaseElement{widget: w}}
}

// Update 更新 Widget 配置并标记需要重建
func (e *StatelessElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.built = false // 重置，下次 Build 会重新构建
	e.dirty = true
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

// Build 构建子 Element
// 如果 Widget 实现了 Build(BuildContext) Widget 接口，调用它获取子控件。
// 当子控件类型匹配时复用已有的 Element 以保持状态。
func (e *StatelessElement) Build() []Element {
	if e.built {
		return e.children
	}
	e.built = true

	// 检查是否实现了 Builder 接口
	if builder, ok := e.widget.(interface{ Build(BuildContext) Widget }); ok {
		w := builder.Build(BuildContext{Element: e})
		if w != nil {
			// 尝试复用已有的子 Element（类型匹配时）
			if len(e.children) > 0 && reflect.TypeOf(e.children[0].Widget()) == reflect.TypeOf(w) {
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
}

// createStatefulElement 工厂函数
func createStatefulElement(w Widget) *StatefulElement {
	return &StatefulElement{BaseElement: BaseElement{widget: w}}
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
	if e.child != nil && !e.buildDirty && e.lastRebuildSeq == globalRebuildSeq {
		e.children = []Element{e.child}
		return e.children
	}
	e.buildDirty = false
	e.lastRebuildSeq = globalRebuildSeq

	// 调用 state.Build 获取子控件
	w := e.state.Build(BuildContext{Element: e})
	if w == nil {
		e.child = nil
		e.children = nil
		return nil
	}

	// 尝试复用已有的子 Element（当 Widget 类型匹配时）
	if e.child != nil && reflect.TypeOf(e.child.Widget()) == reflect.TypeOf(w) {
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
		// 配置可能变化，标记重建；Update 发生在 buildTree 中，重建由后续对本元素的
		// Build() 递归调用完成，无需再触发 SetState（避免双重 state.Build）。
		e.buildDirty = true
	}
}

// Rebuild 重建子树（在 SetState 时调用）
func (e *StatefulElement) Rebuild() {
	if e.state == nil {
		return
	}
	// 标记需要重建；实际 state.Build 延迟到下一次 buildTree 的 Build()，
	// 避免 SetState 路径与 buildTree 路径重复构建（动画高频重建时尤其重要）。
	e.buildDirty = true

	// 标记需要布局和重绘
	e.MarkNeedsLayout()
	e.MarkNeedsPaint()

	// 通知 Pipeline 需要重绘（UI 状态已改变）
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// RenderObjectElement 直接管理渲染的 Element
type RenderObjectElement struct {
	BaseElement
}
