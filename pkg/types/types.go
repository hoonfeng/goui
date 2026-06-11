// Package types 定义了 goui 框架的基础数据类型。
package types

// Point 表示二维坐标点
type Point struct {
	X float64
	Y float64
}

// Size 表示宽高尺寸
type Size struct {
	Width  float64
	Height float64
}

// Rect 表示矩形区域（位置 + 尺寸）
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// Inset 表示四边内边距
type Inset struct {
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

// EdgeInsets 创建统一的边距
func EdgeInsets(all float64) Inset {
	return Inset{Top: all, Right: all, Bottom: all, Left: all}
}

// EdgeInsetsLTRB 创建指定四边的边距
func EdgeInsetsLTRB(left, top, right, bottom float64) Inset {
	return Inset{Top: top, Right: right, Bottom: bottom, Left: left}
}

// Matrix4 是 4x4 变换矩阵（列主序）
type Matrix4 [16]float64

// IdentityMatrix4 返回单位矩阵
func IdentityMatrix4() Matrix4 {
	return Matrix4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Color 表示 RGBA 颜色，分量范围 0-255
type Color struct {
	R, G, B, A uint8
}

// Predefined colors
var (
	ColorBlack   = Color{0, 0, 0, 255}
	ColorWhite   = Color{255, 255, 255, 255}
	ColorRed     = Color{255, 0, 0, 255}
	ColorGreen   = Color{0, 255, 0, 255}
	ColorBlue    = Color{0, 0, 255, 255}
	ColorGray    = Color{128, 128, 128, 255}
	ColorLightGray = Color{211, 211, 211, 255}
	ColorDarkGray  = Color{64, 64, 64, 255}
	ColorTransparent = Color{0, 0, 0, 0}
)

// ColorFromRGB 从 RGB 创建颜色（alpha=255）
func ColorFromRGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b, A: 255}
}

// ColorRef 从 RGB 创建颜色指针，用于需要 *Color 的字段
func ColorRef(r, g, b uint8) *Color {
	return &Color{R: r, G: g, B: b, A: 255}
}

// ColorFromRGBA 从 RGBA 创建颜色
func ColorFromRGBA(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// ColorFromHex 从十六进制颜色代码创建颜色
// 支持格式: #RGB, #RRGGBB, #RRGGBBAA
func ColorFromHex(hex string) Color {
	if len(hex) == 0 || hex[0] != '#' {
		return ColorBlack
	}
	hex = hex[1:]
	var r, g, b, a uint8 = 0, 0, 0, 255
	switch len(hex) {
	case 3:
		r = hexPair(hex[0]) * 17
		g = hexPair(hex[1]) * 17
		b = hexPair(hex[2]) * 17
	case 6:
		r = hexPair(hex[0])*16 + hexPair(hex[1])
		g = hexPair(hex[2])*16 + hexPair(hex[3])
		b = hexPair(hex[4])*16 + hexPair(hex[5])
	case 8:
		r = hexPair(hex[0])*16 + hexPair(hex[1])
		g = hexPair(hex[2])*16 + hexPair(hex[3])
		b = hexPair(hex[4])*16 + hexPair(hex[5])
		a = hexPair(hex[6])*16 + hexPair(hex[7])
	}
	return Color{R: r, G: g, B: b, A: a}
}

func hexPair(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

// WithAlpha 返回带指定透明度的颜色副本
func (c Color) WithAlpha(a uint8) Color {
	return Color{R: c.R, G: c.G, B: c.B, A: a}
}

// ToFloat32 返回 [r, g, b, a] 的 float32 数组 (0.0-1.0)
func (c Color) ToFloat32() [4]float32 {
	return [4]float32{
		float32(c.R) / 255.0,
		float32(c.G) / 255.0,
		float32(c.B) / 255.0,
		float32(c.A) / 255.0,
	}
}
