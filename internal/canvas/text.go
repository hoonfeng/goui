package canvas

// FontWeight 字体粗细
type FontWeight int

const (
	FontWeightNormal FontWeight = 400
	FontWeightMedium FontWeight = 500
	FontWeightBold   FontWeight = 700
)

// FontStyle 字体样式
type FontStyle int

const (
	FontStyleNormal FontStyle = iota
	FontStyleItalic
)

// Font 定义字体属性
type Font struct {
	Family  string
	Size    float64
	Weight  FontWeight
	Style   FontStyle
}

// DefaultFont 返回默认字体
func DefaultFont() Font {
	return Font{
		Family: "sans-serif",
		Size:   16,
		Weight: FontWeightNormal,
		Style:  FontStyleNormal,
	}
}

// TextAlign 文本对齐方式。
// 已统一为 TextHAlign 的别名(见 textdraw.go)——历史上有两套对齐枚举，现合并为一套，
// 老代码用 canvas.TextAlign / TextAlignCenter 仍可用(类型别名零破坏)。
type TextAlign = TextHAlign

const (
	TextAlignLeft   = HAlignLeft
	TextAlignCenter = HAlignCenter
	TextAlignRight  = HAlignRight
)

// TextMetrics 文本度量信息
type TextMetrics struct {
	Width   float64
	Height  float64
	Ascent  float64
	Descent float64
	Advance float64 // 水平前进宽度

	// 实际墨迹包围盒（相对基线，y 向下）：InkTop 为负=基线以上，InkBottom 为正=基线以下。
	// 来自 Skia sk_font_measure_text 的 tight bounds，用于「按实际字形」垂直居中
	// （中文方块字不占字体 descent，用 ascent/descent 居中会偏上，用墨迹盒才真居中）。
	InkTop    float64
	InkBottom float64
}
