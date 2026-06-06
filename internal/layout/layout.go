package layout

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/user/goui/internal/types"
)

func init() {
	if v, err := strconv.ParseBool(os.Getenv("LAYOUT_DEBUG")); err == nil && v {
		DebugLayout = true
	}
}

// LayoutContext 提供布局上下文信息
type LayoutContext struct {
	Constraints BoxConstraints
}

// LayoutResult 包含布局后的尺寸和位置信息
type LayoutResult struct {
	Size types.Size
}

// DebugLayout 控制是否输出布局调试日志（默认为关闭，调试时设为 true）
var DebugLayout = false

// LogLayout 输出布局调试日志
func LogLayout(format string, args ...interface{}) {
	if DebugLayout {
		log.Printf("[LAYOUT] "+format, args...)
	}
}

// LogLayoutDetailed 输出详细布局调试日志（包含组件层级信息）
func LogLayoutDetailed(prefix, widgetType string, constraints BoxConstraints, resultSize types.Size, extra ...string) {
	if !DebugLayout {
		return
	}
	extraStr := ""
	if len(extra) > 0 && extra[0] != "" {
		extraStr = " | " + extra[0]
	}
	log.Printf("[LAYOUT] %s %s: in=(MinW=%.0f,MaxW=%.0f,MinH=%.0f,MaxH=%.0f) out=(%.0f x %.0f)%s",
		prefix, widgetType,
		constraints.MinWidth, constraints.MaxWidth,
		constraints.MinHeight, constraints.MaxHeight,
		resultSize.Width, resultSize.Height,
		extraStr,
	)
}

// ConstraintString 返回约束的简短字符串表示
func ConstraintString(c BoxConstraints) string {
	minW, maxW, minH, maxH := c.MinWidth, c.MaxWidth, c.MinHeight, c.MaxHeight
	if maxW >= float64(1<<30) {
		return fmt.Sprintf("(W:0..INF, H:%.0f..%.0f)", minH, maxH)
	}
	if maxH >= float64(1<<30) {
		return fmt.Sprintf("(W:%.0f..%.0f, H:0..INF)", minW, maxW)
	}
	return fmt.Sprintf("(W:%.0f..%.0f, H:%.0f..%.0f)", minW, maxW, minH, maxH)
}

// Layouter 是所有布局算法的接口
type Layouter interface {
	// Layout 执行布局计算
	Layout(ctx *LayoutContext) LayoutResult
}

// Alignment 元素相对于父容器的对齐方式
type Alignment int

const (
	AlignTopLeft     Alignment = iota
	AlignTopCenter
	AlignTopRight
	AlignCenterLeft
	AlignCenter
	AlignCenterRight
	AlignBottomLeft
	AlignBottomCenter
	AlignBottomRight
)

// FlexDirection Flexbox 方向
type FlexDirection int

const (
	FlexRow    FlexDirection = iota // 水平排列
	FlexColumn                      // 垂直排列
)

// MainAxisAlignment 主轴对齐方式
type MainAxisAlignment int

const (
	MainStart        MainAxisAlignment = iota // 起点对齐
	MainCenter                                // 居中对齐
	MainEnd                                   // 终点对齐
	MainSpaceBetween                          // 两端对齐
	MainSpaceAround                           // 均匀分布
)

// CrossAxisAlignment 交叉轴对齐方式
type CrossAxisAlignment int

const (
	CrossStart CrossAxisAlignment = iota
	CrossCenter
	CrossEnd
	CrossStretch // 拉伸填满
)

// FlexFit Flex 子项适应方式
type FlexFit int

const (
	FitLoose FlexFit = iota // 子项自行决定尺寸
	FitTight                // 强制填满可用空间
)
