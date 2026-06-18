package widget

import (
	"sync/atomic"
	"time"
)

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

	// 【轮询模式】数据版本号指针 + 上次检查值。
	// UI 线程每帧调用 PollCheckVersion 检测外部数据（Agent 写）是否变化，
	// 变化时自动触发 MarkNeedsLayout。适用于"Agent 只写数据、UI 自己读"的架构。
	// 由外部通过 BindPollVersion 设置。
	pollVersion *atomic.Int64
	lastPollVer int64
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

// SetStateRepaint 标记需要重绘但**不触发重新布局**。
// 用于仅视觉变化（光标位置、悬停态、动画帧等）不需要重建控件树的场景。
// 与 SetState 的区别：
//   - SetState → MarkNeedsLayout → buildTree（全树重建，可能导致滚动复位、输入框失焦）
//   - SetStateRepaint → MarkNeedsRepaint → 仅触发 Paint（保留所有运行时状态）
// 注意：子树的 Widget 配置变化（如文本内容变化）仍需要 SetState 确保重建传播；
// 仅当显示依赖外部可变全局（如 cursorPosText 从全局 editorpanel.Editor 读取）时可用此法。
func (s *BaseState) SetStateRepaint() {
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// SetStateStreaming 流式更新专用节流版 SetState：两次重建间隔 >= 100ms。
// Agent 流式输出每秒可能产生数十个事件，将重建频率限制到 ≤10 次/秒。
// 文本修改直接作用于 state.Message 对象，100ms 间隔内累积的内容在下一次
// 重建时自动拾取，用户看到的仍是实时追加的流式文本。
//
// 注意：这会触发全树重建（buildTree）。如果仅文本追加（thinking/content）、
// 没有新增工具调用等结构变化，应使用 SetStateStreamingRepaint。
func (s *BaseState) SetStateStreaming(lastTime *time.Time) {
	if lastTime == nil {
		return
	}
	now := time.Now()
	if now.Sub(*lastTime) < 250*time.Millisecond {
		return
	}
	*lastTime = now
	s.SetState()
}

// SetStateNoRebuild 纯数据更新——仅标记需要重绘，不触发重建、不触发布局。
// 
// 这是「UI线程轮询渲染」模式的核心：Agent/后台 goroutine 通过 drain 把数据写到 state，
// 调用 SetStateNoRebuild 只告诉管线「有数据更新」，管线在下一次 Render 时自行从 state 读取最新数据。
// 组件（如 FusedContent）在 Paint 阶段直接访问 state 指针读取最新内容。
//
// 完全不调用 element.Rebuild()、不触发 EnqueueUpdate、不调用 OnNeedsLayout。
// 相比 SetStateRepaint（仅重绘，仍依赖 OnNeedsRepaint 全局回调）：
//   SetStateNoRebuild 调用 OnNeedsRepaint 让管线知道需要下一帧渲染。
func (s *BaseState) SetStateNoRebuild() {
	s.dirty = true
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// SetStateStreamingRepaint 流式更新专用重绘版：两次调用间隔 >= 250ms，
// 只触发重绘+布局，**不触发全树重建（buildTree）**。
//
// Agent 流式输出（thinking/text 追加）只改消息内容，不改消息结构（不增删消息、
// 不切换折叠态、不添加工具调用），不需要 buildTree。
// FusedContent 在 Layout 中检查 SegmentVersion 后自行更新 slot 内容。
//
// 原理：
//   - 只调用 OnNeedsLayout → MarkNeedsLayout → needsLayout=true, needsRepaint=true
//   - 不调用 s.element.Rebuild() → 不 EnqueueUpdate → pendingUpdates 不变
//   - ConsumePendingUpdates 无消费 → needsRebuild=false
//   - PerformLayout 看到 needsRebuild=false → 跳过 buildTree → 只跑 Layout 瀑布
//   - FusedContent.Layout 检查 SegmentVersion → 更新变化段的内容 → Paint 渲染
//
// 结构变化（新增工具调用、增删消息、完成/停止）仍应使用 SetStateStreaming / SetState。
//
// 即将废弃：新代码应使用 SetStateNoRebuild + FusedContent.Paint 从 state 读取最新数据。
func (s *BaseState) SetStateStreamingRepaint(lastTime *time.Time) {
	if lastTime == nil {
		return
	}
	now := time.Now()
	if now.Sub(*lastTime) < 250*time.Millisecond {
		return
	}
	*lastTime = now
	// 只触发布局+重绘，不触发全树重建
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	}
}

// BindPollVersion 绑定外部数据版本号指针。
// 【轮询模式】Agent/后台 goroutine 只写数据、递增版本号，不碰任何 UI API。
// UI 线程每帧调用 PollCheckVersion 检测版本号变化后自动触发重建。
func (s *BaseState) BindPollVersion(ver *atomic.Int64) {
	s.pollVersion = ver
	if ver != nil {
		s.lastPollVer = ver.Load()
	}
}

// PollCheckVersion 检查绑定的数据版本号是否变化。
// 返回 true 表示数据已更新，UI 需要重建。
// 由 app.go mainLoop 每帧调用（RootElement 递归遍历可见 StatefulElement）。
func (s *BaseState) PollCheckVersion() bool {
	if s.pollVersion == nil {
		return false
	}
	v := s.pollVersion.Load()
	if v != s.lastPollVer {
		s.lastPollVer = v
		return true
	}
	return false
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
