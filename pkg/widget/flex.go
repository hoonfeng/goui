package widget

import (
	"fmt"
	"reflect"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/i18n"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// Flex 实现 Flexbox 布局容器
type Flex struct {
	MultiChildWidget
	Direction          layout.FlexDirection
	MainAxisAlignment  layout.MainAxisAlignment
	CrossAxisAlignment layout.CrossAxisAlignment
	Gap                float64 // 子项之间的间距（CSS gap）
}

// FlexChild 是 Flex 子项的配置
type FlexChild struct {
	Widget Widget
	Flex   int // flex 值，0 表示不弹性
	Fit    layout.FlexFit
}

// FlexElement Flex 的运行时 Element
type FlexElement struct {
	BaseElement
	flex *Flex
}

// CreateElement 创建 FlexElement
func (f *Flex) CreateElement() Element {
	return &FlexElement{
		BaseElement: BaseElement{widget: f},
		flex:        f,
	}
}

// Update 更新 Widget 配置并同步内部 flex 引用。
// 关键：FlexElement.Build 使用 e.flex.Children，若不同步，复用时会沿用旧的子列表，
// 导致父级 SetState 后新的子组件配置无法传播到深层。
func (e *FlexElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.dirty = true
	switch w := newWidget.(type) {
	case *Flex:
		e.flex = w
	case *Column:
		e.flex = &w.Flex
	case *Row:
		e.flex = &w.Flex
	}
}

// Build 构建子 Element 列表
// 复用已有的子 Element（当 Widget 类型匹配时），避免丢失运行时状态
// （如 InputElement 的 focused/hovered/cursorPos/text 等）。
func (e *FlexElement) Build() []Element {
	children := e.flex.Children

	// 空子节点：卸载旧的并返回空的非 nil 切片。
	// Build() 的返回值约定始终非 nil，便于上层区分"无子节点"与"未构建"。
	if len(children) == 0 {
		for _, child := range e.children {
			child.Unmount()
		}
		e.children = []Element{}
		return e.children
	}

	oldChildren := e.children

	// 如果旧的 children 数量与新的不匹配，全部重建
	if len(oldChildren) != len(children) {
		for _, child := range oldChildren {
			child.Unmount()
		}
		e.children = make([]Element, 0, len(children))
		for _, child := range children {
			el := CreateElementFor(child)
			el.Mount(e, len(e.children))
			e.children = append(e.children, el)
		}
		return e.children
	}

	// 数量匹配时，逐个检查 Widget 类型，类型匹配则复用
	for i, child := range children {
		if i < len(oldChildren) && oldChildren[i].WidgetType() == reflect.TypeOf(child) {
			// 类型匹配：复用并更新
			oldChildren[i].Update(child)
		} else {
			// 类型不匹配：逐项替换而非全量重建——只 Unmount 当前旧 Element 并创建新的，
			// 其他位置的 Element（含深层 InputElement 等有状态组件）保留不受影响，
			// 避免因某个占位 Div↔Container 交替导致整棵树状态丢失、滚动复位、输入框失焦。
			if i < len(oldChildren) {
				oldChildren[i].Unmount()
			}
			newEl := CreateElementFor(child)
			newEl.Mount(e, i)
			if i < len(oldChildren) {
				oldChildren[i] = newEl
			} else {
				oldChildren = append(oldChildren, newEl)
			}
		}
	}
	e.children = oldChildren
	return e.children
}

// Layout 执行 Flexbox 布局（简化版本）
func (e *FlexElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	f := e.flex
	isRow := f.Direction == layout.FlexRow
	widgetType := fmt.Sprintf("%T", e.Widget())

	layout.LogLayoutDetailed("Flex(measure)", widgetType,
		ctx.Constraints, types.Size{}, fmt.Sprintf("children=%d dir=%v", len(e.children), f.Direction))

	// 主轴/交叉轴可用空间
	availMain := ctx.Constraints.MaxWidth
	availCross := ctx.Constraints.MaxHeight
	if !isRow {
		availMain = ctx.Constraints.MaxHeight
		availCross = ctx.Constraints.MaxWidth
	}
	// align-items: stretch —— 交叉轴拉伸子项填满 maxCross。
	stretch := f.CrossAxisAlignment == layout.CrossStretch

	// 第一遍：测量非弹性子；弹性子(Expanded/Spacer)记录 flex，待第二遍按比例分配。
	var usedMain, maxCross float64
	totalFlex := 0
	flexOf := make([]int, len(e.children))
	for i, child := range e.children {
		fl := childFlex(child)
		flexOf[i] = fl
		if fl > 0 {
			totalFlex += fl
			continue
		}
		cc := layout.Unbounded()
		if isRow {
			cc.MaxHeight = availCross
			if stretch && availCross < float64(1<<30) {
				cc.MinHeight = availCross
			}
		} else {
			cc.MaxWidth = availCross
			// 拉伸列：子直接按最终交叉宽测量，使多行文本据此算对行数/高度。
			// 否则第二遍只把宽拉到 maxCross、却锁死第一遍高度，行数变多就被裁。
			if stretch && availCross < float64(1<<30) {
				cc.MinWidth = availCross
			}
		}
		child.Layout(&layout.LayoutContext{Constraints: cc})
		sz := child.Size()
		if isRow {
			usedMain += sz.Width
			if sz.Height > maxCross {
				maxCross = sz.Height
			}
		} else {
			usedMain += sz.Height
			if sz.Width > maxCross {
				maxCross = sz.Width
			}
		}
	}

	// 计入子项间距（gap）：n 个子项之间有 n-1 道间隙
	if f.Gap > 0 && len(e.children) > 1 {
		usedMain += f.Gap * float64(len(e.children)-1)
	}

	// 第二遍：剩余主轴空间按 flex 比例分给弹性子（flex-grow）。
	remaining := 0.0
	if availMain < float64(1<<30) {
		if remaining = availMain - usedMain; remaining < 0 {
			remaining = 0
		}
	}
	for i, child := range e.children {
		if flexOf[i] == 0 {
			continue
		}
		mainSize := 0.0
		if totalFlex > 0 {
			mainSize = remaining * float64(flexOf[i]) / float64(totalFlex)
		}
		if mm := childMinMain(child); mm > mainSize { // 尊重弹性子的主轴最小尺寸，不挤到不可读（宁可溢出）
			mainSize = mm
		}
		cc := layout.BoxConstraints{}
		if isRow {
			cc.MinWidth, cc.MaxWidth = mainSize, mainSize
			cc.MaxHeight = availCross
		} else {
			cc.MinHeight, cc.MaxHeight = mainSize, mainSize
			cc.MaxWidth = availCross
		}
		child.Layout(&layout.LayoutContext{Constraints: cc})
		sz := child.Size()
		if isRow {
			usedMain += sz.Width
			if sz.Height > maxCross {
				maxCross = sz.Height
			}
		} else {
			usedMain += sz.Height
			if sz.Width > maxCross {
				maxCross = sz.Width
			}
		}
	}

	layout.LogLayout("  usedMain=%.0f maxCross=%.0f totalFlex=%d", usedMain, maxCross, totalFlex)

	// 分配位置 —— justify-content(主轴) 与 align-items(交叉轴)。
	// 有弹性子时剩余空间已被吃掉(freeSpace=0)；非弹性且非 start 时撑满主轴产生空闲。
	mainExtent := usedMain
	freeSpace := 0.0
	if availMain < float64(1<<30) && (f.MainAxisAlignment != layout.MainStart || totalFlex > 0) {
		mainExtent = availMain
		if totalFlex == 0 {
			if freeSpace = availMain - usedMain; freeSpace < 0 {
				freeSpace = 0
			}
		}
	}
	startOffset, betweenExtra := mainAxisDistribution(f.MainAxisAlignment, freeSpace, len(e.children))

	rtl := isRow && i18n.IsRTL()
	mainOffset := startOffset
	if rtl {
		mainOffset = mainExtent - startOffset
	}
	for i, child := range e.children {
		childSize := child.Size()
		// 动态更新 maxCross：弹性子在第二遍获得主轴分配后尺寸可能增大，
		// 空 Div(0 高)不应污染 maxCross。
		if isRow {
			if childSize.Height > maxCross {
				maxCross = childSize.Height
			}
		} else {
			if childSize.Width > maxCross {
				maxCross = childSize.Width
			}
		}
		if stretch {
			childCross := childSize.Height // 该子当前的交叉轴尺寸
			if !isRow {
				childCross = childSize.Width
			}
			if childCross < maxCross-0.5 { // 仅未达 maxCross 的子才需拉伸重排；已达的跳过这遍 layout 递归
				cc := layout.BoxConstraints{}
				if isRow {
					cc.MinWidth, cc.MaxWidth = childSize.Width, childSize.Width
					cc.MinHeight, cc.MaxHeight = maxCross, maxCross
				} else {
					cc.MinHeight, cc.MaxHeight = childSize.Height, childSize.Height
					cc.MinWidth, cc.MaxWidth = maxCross, maxCross
				}
				child.Layout(&layout.LayoutContext{Constraints: cc})
				childSize = child.Size()
			}
		}
		childCross := childSize.Height
		if !isRow {
			childCross = childSize.Width
		}
		cross := crossAxisOffset(f.CrossAxisAlignment, maxCross, childCross)
		extra := 0.0
		if i > 0 {
			extra = f.Gap + betweenExtra
		}
		if isRow {
			if rtl {
				mainOffset -= extra + childSize.Width
				child.SetPosition(types.Point{X: mainOffset, Y: cross})
			} else {
				mainOffset += extra
				child.SetPosition(types.Point{X: mainOffset, Y: cross})
				mainOffset += childSize.Width
			}
		} else {
			mainOffset += extra
			child.SetPosition(types.Point{X: cross, Y: mainOffset})
			mainOffset += childSize.Height
		}
	}

	// 修正 maxCross：内联拉伸（上方 for 循环）依赖「当前已累计」的 maxCross，
	// 排列靠前的子项可能在 maxCross 尚未更新时被跳过（如空 Div 高为 0 时
	// 排在内容子前面），导致前面子项高度不足。补偿拉伸强制收集最终交叉轴，
	// 对所有仍短于 finalMaxCross 的子补一次拉伸，确保 stretch 语义完全落地。
	// — 场景：agentMessageCard 的 Row(stretch) 中 shadowBar(高 0) 在
	//   cardBody(高 contentH) 之前 → 补偿拉伸把 shadowBar 拉到 contentH。
	if stretch {
		finalMaxCross := maxCross
		// 第一遍：收集最终最大交叉轴
		for _, child := range e.children {
			sz := child.Size()
			if isRow {
				if sz.Height > finalMaxCross {
					finalMaxCross = sz.Height
				}
			} else {
				if sz.Width > finalMaxCross {
					finalMaxCross = sz.Width
				}
			}
		}
		// 第二遍：对所有交叉轴 < finalMaxCross 的子项做补偿拉伸。
		// 注意：内联拉伸可能已把 maxCross 动态更新到 finalMaxCross，
		// 但排列在前面的子项仍可能未拉伸 → 此处强制遍历补拉。
		needsCompensation := false
		for _, child := range e.children {
			sz := child.Size()
			childCross := sz.Height
			if !isRow {
				childCross = sz.Width
			}
			if childCross < finalMaxCross-0.5 {
				needsCompensation = true
				break
			}
		}
		if needsCompensation || finalMaxCross > maxCross {
			maxCross = finalMaxCross
			for _, child := range e.children {
				sz := child.Size()
				childCross := sz.Height
				if !isRow {
					childCross = sz.Width
				}
				if childCross < maxCross-0.5 {
					cc := layout.BoxConstraints{}
					if isRow {
						cc.MinWidth, cc.MaxWidth = sz.Width, sz.Width
						cc.MinHeight, cc.MaxHeight = maxCross, maxCross
					} else {
						cc.MinHeight, cc.MaxHeight = sz.Height, sz.Height
						cc.MinWidth, cc.MaxWidth = maxCross, maxCross
					}
					child.Layout(&layout.LayoutContext{Constraints: cc})
					child.SetPosition(child.Position()) // 保持原定位
				}
			}
		}
	}

	// 计算 Flex 自身尺寸（主轴用 mainExtent：对齐撑满时为可用空间，否则为内容长度）
	if isRow {
		e.size = types.Size{
			Width:  ctx.Constraints.ClampWidth(mainExtent),
			Height: ctx.Constraints.ClampHeight(maxCross),
		}
	} else {
		e.size = types.Size{
			Width:  ctx.Constraints.ClampWidth(maxCross),
			Height: ctx.Constraints.ClampHeight(mainExtent),
		}
	}

	layout.LogLayoutDetailed("Flex(result)", widgetType, ctx.Constraints, e.size)
	return layout.LayoutResult{Size: e.size}
}

// mainAxisDistribution 按主轴对齐(justify-content)计算起始偏移与子项间额外间距。
func mainAxisDistribution(a layout.MainAxisAlignment, free float64, n int) (start, between float64) {
	if free <= 0 || n == 0 {
		return 0, 0
	}
	switch a {
	case layout.MainCenter:
		return free / 2, 0
	case layout.MainEnd:
		return free, 0
	case layout.MainSpaceBetween:
		if n > 1 {
			return 0, free / float64(n-1)
		}
		return 0, 0
	case layout.MainSpaceAround:
		g := free / float64(n)
		return g / 2, g
	default: // MainStart
		return 0, 0
	}
}

// crossAxisOffset 按交叉轴对齐(align-items)计算子项在交叉轴上的偏移。
func crossAxisOffset(a layout.CrossAxisAlignment, maxCross, childCross float64) float64 {
	switch a {
	case layout.CrossCenter:
		return (maxCross - childCross) / 2
	case layout.CrossEnd:
		return maxCross - childCross
	default: // CrossStart；CrossStretch 在定位阶段重新 layout 子，此处偏移 0
		return 0
	}
}

// childFlex 返回子项的弹性系数(Spacer/Expanded → flex 值；其他 → 0)，用于 flex-grow 分配。
func childFlex(child Element) int {
	switch w := child.Widget().(type) {
	case *Spacer:
		return 1
	case *Expanded:
		if w.Flex <= 0 {
			return 1
		}
		return w.Flex
	}
	return 0
}

// childMinMain 弹性子声明的主轴最小尺寸（Expanded.MinMain）；flex 分配不足时据此兜底，不挤子。
func childMinMain(child Element) float64 {
	if w, ok := child.Widget().(*Expanded); ok {
		return w.MinMain
	}
	return 0
}

// flexDir / parseJustify / parseAlign 把 CSS 风格字符串解析为布局枚举（供声明式 Style 使用）。
func flexDir(s string) layout.FlexDirection {
	if s == "row" {
		return layout.FlexRow
	}
	return layout.FlexColumn
}

func parseJustify(s string) layout.MainAxisAlignment {
	switch s {
	case "center":
		return layout.MainCenter
	case "end", "flex-end":
		return layout.MainEnd
	case "space-between":
		return layout.MainSpaceBetween
	case "space-around":
		return layout.MainSpaceAround
	default:
		return layout.MainStart
	}
}

func parseAlign(s string) layout.CrossAxisAlignment {
	switch s {
	case "center":
		return layout.CrossCenter
	case "end", "flex-end":
		return layout.CrossEnd
	case "stretch":
		return layout.CrossStretch
	default:
		return layout.CrossStart
	}
}

// Paint 绘制 Flex（递归绘制子控件）
func (e *FlexElement) Paint(cvs canvas.Canvas, offset types.Point) {
	for _, child := range e.children {
		child.Paint(cvs, offset)
	}
}

// Column 是垂直排列的 Flex 快捷方式
type Column struct {
	Flex
}

// NewColumn 创建垂直排列的列
func NewColumn(children []Widget) *Column {
	return &Column{
		Flex: Flex{
			MultiChildWidget: MultiChildWidget{Children: children},
			Direction:        layout.FlexColumn,
		},
	}
}

// Row 是水平排列的 Flex 快捷方式
type Row struct {
	Flex
}

// NewRow 创建水平排列的行
func NewRow(children []Widget) *Row {
	return &Row{
		Flex: Flex{
			MultiChildWidget: MultiChildWidget{Children: children},
			Direction:        layout.FlexRow,
		},
	}
}

// Spacer 是弹性空白控件，用于 Flex 中占据剩余空间
type Spacer struct {
	StatelessWidget
}

// CreateElement 创建 SpacerElement
func (s *Spacer) CreateElement() Element {
	return &SpacerElement{
		BaseElement: BaseElement{widget: s},
	}
}

// SpacerElement Spacer 的 Element
type SpacerElement struct {
	BaseElement
}

// Layout Spacer 在弹性布局中占据剩余空间，但在无界约束中不应无限膨胀
func (e *SpacerElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := ctx.Constraints.MaxWidth
	h := ctx.Constraints.MaxHeight
	// 在无界方向上不膨胀（例如 Column 中高度无界、Row 中宽度无界）
	if w >= float64(1<<30) {
		w = ctx.Constraints.MinWidth
	}
	if h >= float64(1<<30) {
		h = ctx.Constraints.MinHeight
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})

	layout.LogLayoutDetailed("Spacer", fmt.Sprintf("%T", e.Widget()),
		ctx.Constraints, e.size)
	return layout.LayoutResult{Size: e.size}
}

// ─── Web 风格便捷构建函数 ───────────────────────────────
// VBox 是 Vertical Box 的缩写，等同于 Column，提供 Web 风格的命名。
// 用法: widget.VBox(widget.Text("Hello"), widget.Button("OK"))
func VBox(children ...Widget) *Column {
	return NewColumn(children)
}

// HBox 是 Horizontal Box 的缩写，等同于 Row，提供 Web 风格的命名。
// 用法: widget.HBox(widget.Text("Name:"), widget.Input{...})
func HBox(children ...Widget) *Row {
	return NewRow(children)
}

// Expanded 让子控件填充 Flex 分配的弹性空间。
type Expanded struct {
	SingleChildWidget
	Flex int
	// MinMain 弹性子的主轴最小尺寸（行=宽、列=高）。flex 按比例分得的空间不足此值时，
	// 至少分配 MinMain（宁可整体溢出也不把子挤到不可读）。0=不限制。
	MinMain float64
}

// CreateElement 创建 ExpandedElement。
func (ex *Expanded) CreateElement() Element {
	return &ExpandedElement{BaseElement: BaseElement{widget: ex}, expanded: ex}
}

// ExpandedElement 把父(Flex)分配的主轴尺寸约束传给子，使子填满弹性空间。
type ExpandedElement struct {
	BaseElement
	expanded *Expanded
	child    Element
}

func (e *ExpandedElement) Build() []Element {
	if e.expanded.Child != nil {
		// 复用已有子 Element（类型匹配时），避免每帧重建导致丢失状态（如 Splitter 拖动、VirtualList 缓存等）。
		if e.child != nil && reflect.TypeOf(e.child.Widget()) == reflect.TypeOf(e.expanded.Child) {
			e.child.Update(e.expanded.Child)
		} else {
			if e.child != nil {
				e.child.Unmount()
			}
			e.child = CreateElementFor(e.expanded.Child)
			e.child.Mount(e, 0)
		}
		e.children = []Element{e.child}
		return e.children
	}
	if e.child != nil {
		e.child.Unmount()
		e.child = nil
	}
	e.children = nil
	return nil
}

func (e *ExpandedElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		e.child.SetPosition(types.Point{})
		res := e.child.Layout(ctx) // 子沿用父约束(主轴 Min=Max=分配尺寸 → 子填满主轴)
		e.size = ctx.Constraints.Constrain(res.Size)
	} else {
		w := ctx.Constraints.MaxWidth
		h := ctx.Constraints.MaxHeight
		if w >= float64(1<<30) {
			w = ctx.Constraints.MinWidth
		}
		if h >= float64(1<<30) {
			h = ctx.Constraints.MinHeight
		}
		e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *ExpandedElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child != nil {
		e.child.Paint(cvs, offset)
	}
}

func (e *ExpandedElement) Update(newWidget Widget) {
	if ne, ok := newWidget.(*Expanded); ok {
		e.expanded = ne
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
