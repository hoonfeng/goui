package widget

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── ColorPicker 取色器（el-color-picker 内联面板）──────────────
// 自绘 HSV：SV 饱和度-明度方块(逐像素生成 image) + Hue 彩虹条 + 预览/hex。
// 拖动选 SV/Hue；Hue 变化时重新生成 SV 方块。

type ColorPicker struct {
	StatelessWidget
	OnChange func(hex string)
}

func NewColorPicker() *ColorPicker                            { return &ColorPicker{} }
func (c *ColorPicker) WithOnChange(fn func(string)) *ColorPicker { c.OnChange = fn; return c }

func (c *ColorPicker) CreateElement() Element {
	return &ColorPickerElement{BaseElement: BaseElement{widget: c}, cp: c, hue: 210, sat: 0.75, val: 0.9}
}

const (
	cpSVW  = 240.0
	cpSVH  = 160.0
	cpHueH = 16.0
	cpGap  = 12.0
	cpPrev = 28.0
)

type ColorPickerElement struct {
	BaseElement
	cp                *ColorPicker
	hue, sat, val     float64
	svImg             *image.RGBA
	svImgHue          float64
	hueImg            *image.RGBA
	dragSV, dragHue   bool
}

func (e *ColorPickerElement) Build() []Element { return nil }

func (e *ColorPickerElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := cpSVW
	h := cpSVH + cpGap + cpHueH + cpGap + cpPrev
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

// hsvToRGB h∈[0,360) s,v∈[0,1]。
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}

func (e *ColorPickerElement) ensureImages() {
	w, svh, hh := int(cpSVW), int(cpSVH), int(cpHueH)
	if e.hueImg == nil {
		e.hueImg = image.NewRGBA(image.Rect(0, 0, w, hh))
		for x := 0; x < w; x++ {
			r, g, b := hsvToRGB(float64(x)/cpSVW*360, 1, 1)
			for y := 0; y < hh; y++ {
				e.hueImg.SetRGBA(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}
	if e.svImg == nil || e.svImgHue != e.hue {
		e.svImgHue = e.hue
		e.svImg = image.NewRGBA(image.Rect(0, 0, w, svh))
		for x := 0; x < w; x++ {
			s := float64(x) / cpSVW
			for y := 0; y < svh; y++ {
				v := 1 - float64(y)/cpSVH
				r, g, b := hsvToRGB(e.hue, s, v)
				e.svImg.SetRGBA(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}
}

func (e *ColorPickerElement) Paint(cvs canvas.Canvas, offset types.Point) {
	e.ensureImages()
	pos := e.Offset()

	// SV 方块
	cvs.DrawImage(e.svImg, types.Rect{X: pos.X, Y: pos.Y, Width: cpSVW, Height: cpSVH})
	// SV 选点（白圈 + 黑描边）
	sx := pos.X + e.sat*cpSVW
	sy := pos.Y + (1-e.val)*cpSVH
	o := paint.DefaultStrokePaint()
	o.Color = types.ColorFromRGBA(0, 0, 0, 120)
	o.StrokeWidth = 3
	cvs.DrawCircle(sx, sy, 6, o)
	ring := paint.DefaultStrokePaint()
	ring.Color = elSurface()
	ring.StrokeWidth = 1.6
	cvs.DrawCircle(sx, sy, 6, ring)

	// Hue 条
	hueY := pos.Y + cpSVH + cpGap
	cvs.DrawImage(e.hueImg, types.Rect{X: pos.X, Y: hueY, Width: cpSVW, Height: cpHueH})
	hx := pos.X + e.hue/360*cpSVW
	hr := paint.DefaultStrokePaint()
	hr.Color = elSurface()
	hr.StrokeWidth = 2
	cvs.DrawRoundedRect(hx-3, hueY-2, 6, cpHueH+4, 2, hr)

	// 预览 + hex
	prevY := hueY + cpHueH + cpGap
	r, g, b := hsvToRGB(e.hue, e.sat, e.val)
	pv := paint.DefaultPaint()
	pv.Color = types.ColorFromRGB(r, g, b)
	cvs.DrawRoundedRect(pos.X, prevY, cpPrev, cpPrev, 4, pv)
	pb := paint.DefaultStrokePaint()
	pb.Color = elBorder()
	pb.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, prevY+0.5, cpPrev-1, cpPrev-1, 4, pb)
	font := canvas.DefaultFont()
	font.Size = 15
	canvas.DrawTextAligned(cvs, fmt.Sprintf("#%02X%02X%02X", r, g, b),
		types.Rect{X: pos.X + cpPrev + 12, Y: prevY, Width: 140, Height: cpPrev},
		font, elTextPrimary(), canvas.HAlignLeft, canvas.VAlignMiddle)
}

func (e *ColorPickerElement) updateSV(lx, ly float64) {
	e.sat = clamp(lx/cpSVW, 0, 1)
	e.val = clamp(1-ly/cpSVH, 0, 1)
	e.fireChange()
	e.MarkNeedsPaint()
}

func (e *ColorPickerElement) updateHue(lx float64) {
	e.hue = clamp(lx/cpSVW*360, 0, 359.9)
	e.fireChange()
	e.MarkNeedsPaint()
}

func (e *ColorPickerElement) fireChange() {
	if e.cp.OnChange != nil {
		r, g, b := hsvToRGB(e.hue, e.sat, e.val)
		e.cp.OnChange(fmt.Sprintf("#%02X%02X%02X", r, g, b))
	}
}

func (e *ColorPickerElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	lx, ly := me.X-pos.X, me.Y-pos.Y
	hueY := cpSVH + cpGap
	switch ev.Type() {
	case event.TypeMouseDown:
		if ly >= 0 && ly < cpSVH {
			e.dragSV = true
			e.updateSV(lx, ly)
			return true
		}
		if ly >= hueY && ly < hueY+cpHueH {
			e.dragHue = true
			e.updateHue(lx)
			return true
		}
	case event.TypeMouseMove:
		if e.dragSV {
			e.updateSV(lx, ly)
			return true
		}
		if e.dragHue {
			e.updateHue(lx)
			return true
		}
	case event.TypeMouseUp, event.TypeMouseLeave:
		e.dragSV, e.dragHue = false, false
		return true
	}
	return false
}

func (e *ColorPickerElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*ColorPicker); ok {
		e.cp = nc
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
