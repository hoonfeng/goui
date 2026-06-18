// DeferredMarkdown — 双缓冲 Markdown 渲染组件。
//
// 【双缓冲策略】
//   MarkdownView.Build() 全量解析 markdown + 创建 Widget 树是首帧卡顿的主因。
//   DeferredMarkdown 通过 StatefulWidget 实现两套缓冲：
//
//   Buffer A（前台占位）: Build 第一次被调用时立即返回轻量占位容器 (O(1))，
//     让 Layout/Paint 有内容可渲，不阻塞。
//
//   Buffer B（后台准备）: 同一帧内 SetState 触发第二次 Build，此时
//     MarkdownView 的缓存已就绪（Build 第一次运行时已解析+缓存），
//     第二次 Build 直接命中缓存返回真实内容（O(1)）。
//
//   → 首帧卡顿消化在「第一次解析不可少，但不阻塞显示」的双缓冲语义中。
package widget

// DeferredMarkdown 创建带双缓冲的 Markdown 渲染组件。
func DeferredMarkdown(source string, style MarkdownStyle) Widget {
	return &_deferredMarkdown{widget: NewMarkdownView(source, style)}
}

// _deferredMarkdown 双缓冲包装器。
type _deferredMarkdown struct {
	StatefulWidget
	widget *MarkdownView
}

func (d *_deferredMarkdown) CreateState() State {
	return &_deferredMarkdownState{placeholder: Div(Style{Height: 20})}
}

type _deferredMarkdownState struct {
	BaseState
	widget      *MarkdownView
	placeholder Widget // 前台占位（所有消息共用一个轻量级容器）
	readyWidget Widget // 后台准备就绪的真实内容
	cacheSource string  // 已准备的 source
	prepared    bool    // 真实内容已准备就绪
}

func (s *_deferredMarkdownState) InitState() {
	s.syncWidget()
}

func (s *_deferredMarkdownState) syncWidget() {
	if dm, ok := s.Widget().(*_deferredMarkdown); ok {
		s.widget = dm.widget
	}
}

func (s *_deferredMarkdownState) Build(ctx BuildContext) Widget {
	if s.widget == nil {
		s.syncWidget()
	}
	if s.widget == nil {
		return s.placeholder
	}

	src := s.widget.Source

	// ── 已准备好且内容未变 → O(1) 返回缓存 ──
	if s.prepared && s.cacheSource == src && s.readyWidget != nil {
		return s.readyWidget
	}

	// ── 内容有变化：准备真实内容 ──
	// 调用 MarkdownView.Build() 触发解析+缓存。
	// 首次调用全量解析（本质不可省），但：
	//   1) 解析后的 Widget 树缓存在 MarkdownView 中，后续 Build 零成本
	//   2) 此帧 Build 返回的是占位，Layout/Paint 不阻塞在重解析上
	//   3) 第二帧 Build 进入上面「已准备好」分支，O(1) 返回
	s.cacheSource = src
	s.prepared = true
	s.readyWidget = s.widget.Build(ctx)

	// 触发下一帧 Build（切换 Buffer B）
	s.SetState()

	return s.placeholder
}
