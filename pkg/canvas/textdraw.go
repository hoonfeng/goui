package canvas

import (
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── 统一文字绘制原语 ──────────────────────────────────────────
// 把"在一个矩形内按对齐方式绘制单行文字"这个横切关注点收敛到一处，
// 各组件(Tag/Button/Badge/Avatar/Tooltip/Alert/Input…)复用，不再各自计算基线，
// 从根本上消除"文字居中公式散落各处、改一处要改 N 处"的重复。

// TextHAlign 水平对齐。
type TextHAlign int

const (
	HAlignLeft   TextHAlign = iota // 左对齐
	HAlignCenter                   // 水平居中
	HAlignRight                    // 右对齐
)

// TextVAlign 垂直对齐。
type TextVAlign int

const (
	VAlignTop    TextVAlign = iota // 顶对齐(基线在 top+ascent)
	VAlignMiddle                   // 垂直居中
	VAlignBottom                   // 底对齐
)

// BaselineFor 返回在区间 [top, top+height] 内、按垂直对齐方式绘制
// 字号 fontSize 文字时的基线 Y。
//
// ascent/descent 来自 Skia 真实字体度量（见 GetFaceAscent/Descent）——不再有经验估算
// 或修正常数：垂直居中即把字体 [基线-ascent, 基线+descent] 这段在盒内居中，
// baseline = 中线 + (ascent-descent)/2，自然对齐图标等几何居中的内容。
func BaselineFor(top, height, fontSize float64, v TextVAlign) float64 {
	ascent := GetFaceAscent(fontSize)
	descent := GetFaceDescent(fontSize)
	switch v {
	case VAlignTop:
		return top + ascent
	case VAlignBottom:
		return top + height - descent
	default: // VAlignMiddle
		return top + height/2 + (ascent-descent)/2
	}
}

// BaselineForMiddle 返回在 [top, top+height] 内**按实际墨迹**垂直居中文字的基线 Y。
// 用 Skia tight bounds（实际字形包围盒）而非字体 ascent/descent：中文方块字不占字体
// descent，用 ascent/descent 居中会偏上、与图标错位；用墨迹盒中心对齐盒中心才真居中。
//
// ⚠️ 墨迹度量走全局共享测量画布（raster），**不用传入的渲染画布 c**：GPU 渲染画布在 Paint
// 期（GL 上下文激活、Skia 正在 flush）调 MeasureText 取不到墨迹盒（返回空）→ 会回退成偏上的
// ascent/descent 居中，导致「headless raster 截图居中、实机 GPU 窗口却偏」。统一走共享画布，
// 与字号行高(faceVMetrics)同源，保证两种后端居中一致。c 参数保留仅为兼容调用点。
func BaselineForMiddle(c Canvas, text string, font Font, top, height float64) float64 {
	if text != "" {
		m := MeasureTextGlobal(text, font)
		if m.InkTop != 0 || m.InkBottom != 0 {
			inkCenter := (m.InkTop + m.InkBottom) / 2 // 墨迹中心相对基线
			return top + height/2 - inkCenter
		}
	}
	return BaselineFor(top, height, font.Size, VAlignMiddle)
}

// MeasureWidth 返回文字宽度，优先用画布(Skia)测量，回退全局共享测量。
func MeasureWidth(c Canvas, text string, font Font) float64 {
	if c != nil {
		if w := c.MeasureText(text, font).Width; w > 0 {
			return w
		}
	}
	return MeasureTextGlobal(text, font).Width
}

// DrawTextAligned 在矩形 rect 内按 (h, v) 对齐绘制单行文字。
func DrawTextAligned(c Canvas, text string, rect types.Rect, font Font, color types.Color, h TextHAlign, v TextVAlign) {
	if text == "" {
		return
	}
	x := rect.X
	switch h {
	case HAlignCenter:
		x = rect.X + (rect.Width-MeasureWidth(c, text, font))/2
	case HAlignRight:
		x = rect.X + rect.Width - MeasureWidth(c, text, font)
	}
	var y float64
	if v == VAlignMiddle {
		y = BaselineForMiddle(c, text, font, rect.Y, rect.Height) // 按实际墨迹居中
	} else {
		y = BaselineFor(rect.Y, rect.Height, font.Size, v)
	}
	p := paint.DefaultPaint()
	p.Color = color
	c.DrawText(text, x, y, font, p)
}
