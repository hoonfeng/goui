package widget

// OnNeedsRepaint 是全局回调函数，当控件需要重绘时被调用。
// 由 Application 在初始化时设置，用于通知 Pipeline 需要重绘。
var OnNeedsRepaint func()

// OnNeedsLayout 是全局回调函数，当状态改变需要重新构建并布局子树时调用。
// 由 Application 设置，连接到 Pipeline.MarkNeedsLayout。
// buildTree 在类型匹配时复用 Element（不会丢失 Input 等运行时状态），
// 因此 SetState 走 relayout 是安全的，且深层组件的配置变化才能传播。
var OnNeedsLayout func()

// RequestPointerCapture 请求把鼠标指针捕获到指定 Element，使后续 MouseMove/MouseUp
// 即便落在其他元素上也直接送达它。由 Application 设置。
// 用于"拖动靶子不是独立 Element、HitTest 命中其下方内容"的场景(自绘滚动条等)：
// 组件在 MouseDown 命中拖动区时调用，把捕获从内容子元素抢到自己。
var RequestPointerCapture func(Element)

// BaseState 是 State 的基类，提供默认实现
type BaseState struct {
	stateWidget Widget
	context     BuildContext
	dirty       bool
	element     *StatefulElement // 关联的 StatefulElement
}

// InitState 状态初始化
func (s *BaseState) InitState() {}

// Build 构建控件树（由子类实现）
func (s *BaseState) Build(ctx BuildContext) Widget { return nil }

// Widget 返回关联的 StatefulWidget
func (s *BaseState) Widget() Widget { return s.stateWidget }

// Context 返回构建上下文
func (s *BaseState) Context() BuildContext { return s.context }

// SetState 标记需要重建
func (s *BaseState) SetState() {
	s.dirty = true
	if s.element != nil {
		s.element.Rebuild()
	}
	// 状态改变需要重新构建并布局子树，深层组件的配置变化才能传播。
	// buildTree 在类型匹配时复用 Element（不会丢失运行时状态），走 relayout 安全；
	// relayout 同时会触发重绘。无 layout 回调时退化为仅重绘。
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	} else if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// IsDirty 返回是否需要重建
func (s *BaseState) IsDirty() bool { return s.dirty }

// SetWidget 设置关联的 Widget
func (s *BaseState) SetWidget(w Widget) { s.stateWidget = w }

// SetContext 设置构建上下文
func (s *BaseState) SetContext(ctx BuildContext) { s.context = ctx }

// SetElement 设置关联的 Element（由 StatefulElement 调用）
func (s *BaseState) SetElement(el *StatefulElement) { s.element = el }

// GetElement 返回关联的 Element（供 Validate 使用）
func (s *BaseState) GetElement() *StatefulElement { return s.element }
