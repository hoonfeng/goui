// Package widget 定义了 goui 的控件系统。
// 采用 Widget/Element 分离模式，类似 Flutter 架构。
package widget

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/types"
)

// Widget 是所有控件的接口。
// Widget 是不可变的配置描述，不包含运行时状态。
type Widget interface {
	// CreateElement 创建对应的运行时 Element
	CreateElement() Element
}

// StatelessWidget 是无状态控件的基类。
// 这类控件没有可变状态，完全由其父控件和配置决定。
type StatelessWidget struct {
	// 嵌入此结构体表示是一个无状态控件
}

// CreateElement 创建 StatelessElement
func (w *StatelessWidget) CreateElement() Element {
	return createStatelessElement(w)
}

// Build 由子类实现，构建控件树
type Build func(ctx BuildContext) Widget

// StatefulWidget 是有状态控件的基类。
type StatefulWidget struct {
	// 嵌入此结构体表示是一个有状态控件
}

// CreateElement 创建 StatefulElement
func (w *StatefulWidget) CreateElement() Element {
	return createStatefulElement(w)
}

// CreateState 由 StatefulWidget 的子类实现，创建状态对象
type CreateState func() State

// State 是有状态控件的可变状态。
type State interface {
	// InitState 状态初始化时调用
	InitState()

	// Build 构建控件树
	Build(ctx BuildContext) Widget

	// SetState 标记控件需要重建
	SetState()

	// Widget 返回关联的 StatefulWidget（具体类型通过断言获取）
	Widget() Widget

	// Context 返回构建上下文
	Context() BuildContext
}

// CreateElementFor 为 Widget 创建正确的 Element。
// 相比 Widget.CreateElement()，此函数能正确处理 StatefulWidget 嵌入的场景，
// 保留外层具体类型的类型信息，确保 CreateState() 的类型断言能够成功。
//
// 背景：当具体类型（如 *TestApp）嵌入 StatefulWidget 时，提升调用的
// StatefulWidget.CreateElement() 方法的 receiver 是 *StatefulWidget（嵌入字段），
// 而非 *TestApp，导致 StatefulElement 存储的 widget 类型错误，CreateState() 失败。
// 此函数通过类型断言检测 CreateState() 方法的存在性来区分有状态/无状态控件。
func CreateElementFor(w Widget) Element {
	// 检查是否为有状态控件（有 CreateState 方法）
	if creator, ok := w.(interface{ CreateState() State }); ok {
		el := &StatefulElement{
			BaseElement: BaseElement{widget: w},
			buildDirty: true,
		}
		// 立即创建状态
		state := creator.CreateState()
		if state != nil {
			el.state = state
			// 使用接口断言设置 State 内部字段，兼容嵌入 BaseState 的任意具体类型
			if setter, ok := state.(interface {
				SetWidget(Widget)
				SetElement(*StatefulElement)
				SetContext(BuildContext)
			}); ok {
				setter.SetWidget(w)
				setter.SetElement(el)
				setter.SetContext(BuildContext{Element: el})
			}
			state.InitState()
		}
		return el
	}
	// 无状态控件：使用标准创建路径
	el := w.CreateElement()
	// 修复嵌入（embedding）导致的类型丢失：
	// 当具体类型（如 *ConfigProvider）嵌入 StatelessWidget 但没有自己的 CreateElement 时，
	// 通过方法提升调用的 StatelessWidget.CreateElement() 的 receiver 是 *StatelessWidget，
	// 导致 StatelessElement.widget 存储的是 *StatelessWidget 而非 *ConfigProvider。
	// 类似地，*Column/*Row 嵌入 *Flex 时，FlexElement.widget 存储的是 *Flex 而非外层类型。
	// 这里检测到 widget 类型不匹配时，修复为正确的类型。
	if se, ok := el.(*StatelessElement); ok {
		// 仅当 widget 类型不一致时才修复（避免不必要的操作）
		if se.widget != w {
			se.widget = w
			se.widgetType = reflect.TypeOf(w)
		}
	}
	if fe, ok := el.(*FlexElement); ok {
		if fe.widget != w {
			fe.widget = w
			fe.widgetType = reflect.TypeOf(w)
		}
	}
	// 为新创建的 Element 设置 widgetType 缓存（若尚未设置）
	if be, ok := el.(*BaseElement); ok && be.widgetType == nil && be.widget != nil {
		be.widgetType = reflect.TypeOf(be.widget)
	}
	return el
}

// BuildContext 提供构建时的上下文信息
type BuildContext struct {
	// InheritedWidgets 继承控件映射
	InheritedWidgets map[string]Widget
	// Element 关联的 Element
	Element Element
}

// Element 是控件的运行时实例，管理生命周期。
type Element interface {
	// Widget 返回关联的 Widget
	Widget() Widget

	// WidgetType 返回 Widget 的动态类型（反射缓存）。用于 Build 中类型匹配判断，
	// 替代 reflect.TypeOf(el.Widget()) == reflect.TypeOf(w) 的重复反射开销。
	WidgetType() reflect.Type

	// Mount 挂载 Element 到树中
	Mount(parent Element, slotIndex int)

	// Unmount 从树中卸载
	Unmount()

	// Update 更新 Widget 配置
	Update(newWidget Widget)

	// Build 构建并返回子 Element 列表
	Build() []Element

	// Layout 执行布局
	Layout(ctx *layout.LayoutContext) layout.LayoutResult

	// Paint 绘制控件内容
	Paint(canvas canvas.Canvas, offset types.Point)

	// Size 返回 Element 的尺寸
	Size() types.Size

	// Position 返回 Element 的位置（相对于父控件）
	Position() types.Point

	// SetPosition 设置位置
	SetPosition(pos types.Point)

	// HandleEvent 处理事件
	HandleEvent(e event.Event) bool

	// Parent 返回父 Element
	Parent() Element

	// Children 返回子 Element 列表
	Children() []Element

	// Offset 返回 Element 的全局偏移量
	Offset() types.Point

	// MarkNeedsLayout 标记需要重新布局
	MarkNeedsLayout()

	// MarkNeedsPaint 标记需要重绘
	MarkNeedsPaint()

	// Focus/Blur 焦点管理
	Focus()
	Blur()
	IsFocused() bool
}

// MultiChildWidget 是多子控件容器的基类。
// 本身不实现 Widget 接口，由具体控件（Flex 等）组合使用。
type MultiChildWidget struct {
	Children []Widget
}

// SingleChildWidget 是单子控件容器的基类。
// 本身不实现 Widget 接口，由具体控件（Container 等）组合使用。
type SingleChildWidget struct {
	Child Widget
}

// Verifiable 接口表示可自我验证的 Element。
// Validate 检查 Element 的状态一致性并返回所有发现的错误。
type Verifiable interface {
	Validate() []error
}

// Draggable 表示 Element 支持拖拽操作。
// 实现了此接口的 Element 在鼠标按下并移动超过拖拽阈值后，
// 将收到 DragStart/DragMove/DragEnd 事件（替代原始的 MouseMove/MouseUp）。
type Draggable interface {
	// Draggable 返回 true 表示此 Element 需要拖拽事件。
	// App 层在 MouseDown 后检测到显著移动时，若此方法返回 true，
	// 后续的 MouseMove/MouseUp 将转换为 DragEvent 投递。
	Draggable() bool
}

// 鼠标光标形状常量（与 window.Cursor* 对齐；放此处免 widget→window 依赖）。
const (
	CursorDefault = 0
	CursorResizeH = 1 // ↔ 水平双向（拖动调宽）
	CursorResizeV = 2 // ↕ 垂直双向（拖动调高）
	CursorHand    = 3 // 手型（可点击）
	CursorText    = 4 // 文本 I 形
)

// CursorProvider 由想自定义鼠标光标的 Element 实现：当鼠标悬停于其上（或拖动捕获于其上）时，
// App 据返回的形状设置窗口光标。
type CursorProvider interface {
	Cursor() int // 返回 Cursor* 形状常量
}

// SetCursorShape 请求把窗口光标设为给定形状（Cursor* 常量），由 App 注入桥接到平台层。
var SetCursorShape func(shape int)

// HitTestIgnorer 由"装饰型" Element 实现（图标、纯展示文本等）：命中测试穿透它，
// 使点击落到可交互的祖先（如 Button）。否则按钮内的图标会截走点击——HitTest 命中最内层
// 的图标、捕获也落在图标上，MouseUp 直接发给图标不再冒泡，Button 永远收不到 up、OnClick 不触发。
type HitTestIgnorer interface {
	IgnoreHitTest() bool
}

// DropTarget 表示 Element 可以接收拖放操作。
// 实现了此接口的 Element 可以在拖拽进行中接收 DragOver/DragEnter/DragLeave/Drop 事件。
type DropTarget interface {
	// CanAcceptDrop 询问 DropTarget 是否可以接受指定数据源的拖放操作。
	CanAcceptDrop(source Element) bool
}

// Walkable 接口表示可以被遍历的控件
type Walkable interface {
	WalkChildren(fn func(Widget) bool)
}

// HitTestResult 命中测试结果
type HitTestResult struct {
	Element Element
	Point   types.Point
}

// WalkWidget 递归遍历控件树
func WalkWidget(w Widget, fn func(Widget) bool) {
	if !fn(w) {
		return
	}
	if walkable, ok := w.(Walkable); ok {
		walkable.WalkChildren(fn)
	}
}

// DebugPrintElementTree 递归遍历 Element 树并返回结构化的层级信息字符串。
// 每个节点显示 Widget 类型和 Element 类型，缩进反映树深度。
// indent 参数用于控制缩进（首次调用传入 "" 即可）。
func DebugPrintElementTree(root Element, indent string) string {
	var sb strings.Builder
	totalNodes := 0
	debugPrintElementTreeRecursive(root, indent, true, &sb, &totalNodes)
	sb.WriteString(fmt.Sprintf("\nTotal elements: %d\n", totalNodes))
	return sb.String()
}

// debugPrintElementTreeRecursive 递归辅助函数
func debugPrintElementTreeRecursive(el Element, indent string, isLast bool, sb *strings.Builder, total *int) {
	*total++
	prefix := "├── "
	if isLast {
		prefix = "└── "
	}

	widgetTypeStr := fmt.Sprintf("%T", el.Widget())
	elemTypeStr := fmt.Sprintf("%T", el)

	sb.WriteString(indent)
	sb.WriteString(prefix)
	sb.WriteString(fmt.Sprintf("[%s] (%s)", elemTypeStr, widgetTypeStr))

	children := el.Children()
	if len(children) == 0 {
		sb.WriteString(" [leaf]")
	}
	sb.WriteString("\n")

	childIndent := indent + "│   "
	if isLast {
		childIndent = indent + "    "
	}

	for i, child := range children {
		debugPrintElementTreeRecursive(child, childIndent, i == len(children)-1, sb, total)
	}
}
