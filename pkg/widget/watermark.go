package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Watermark 水印（L3 覆盖层）────────────────────────────────
// 1:1 复刻 el-watermark 核心：在子内容之上铺满重复、旋转的半透明水印文字。
// 用绝对覆盖 + 平铺绘制实现。

type Watermark struct {
	StatelessWidget
	Child    Widget
	Content  string
	Color    types.Color // 水印颜色(默认半透明灰)
	FontSize float64     // 默认 16
	Rotate   float64     // 旋转角度，度(默认 -22)
	GapX     float64     // 水平间距(默认 140)
	GapY     float64     // 垂直间距(默认 90)
}

func NewWatermark(content string, child Widget) *Watermark {
	return &Watermark{Child: child, Content: content}
}
func (w *Watermark) WithColor(c types.Color) *Watermark { w.Color = c; return w }
func (w *Watermark) WithRotate(d float64) *Watermark    { w.Rotate = d; return w }
func (w *Watermark) WithGap(x, y float64) *Watermark    { w.GapX = x; w.GapY = y; return w }
func (w *Watermark) WithFontSize(s float64) *Watermark  { w.FontSize = s; return w }

func (w *Watermark) CreateElement() Element {
	return &WatermarkElement{BaseElement: BaseElement{widget: w}, watermark: w}
}

type WatermarkElement struct {
	BaseElement
	watermark *Watermark
	child     Element
}

func (e *WatermarkElement) Build() []Element {
	if e.watermark.Child != nil {
		e.child = CreateElementFor(e.watermark.Child)
		e.child.Mount(e, 0)
		e.children = []Element{e.child}
		return e.children
	}
	e.children = nil
	return nil
}

func (e *WatermarkElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		res := e.child.Layout(ctx)
		e.child.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *WatermarkElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child != nil {
		e.child.Paint(cvs, offset)
	}
	w := e.watermark
	if w.Content == "" {
		return
	}
	pos := e.Offset()

	fontSize := w.FontSize
	if fontSize <= 0 {
		fontSize = 16
	}
	rotate := w.Rotate
	if rotate == 0 {
		rotate = -22
	}
	gapX := w.GapX
	if gapX <= 0 {
		gapX = 140
	}
	gapY := w.GapY
	if gapY <= 0 {
		gapY = 90
	}
	color := w.Color
	if color == (types.Color{}) {
		color = types.ColorFromRGBA(0, 0, 0, 28) // 半透明灰
	}

	font := canvas.DefaultFont()
	font.Size = fontSize
	wp := paint.DefaultPaint()
	wp.Color = color

	// 裁剪到内容区，水印不溢出
	cvs.Save()
	cvs.ClipRect(pos.X, pos.Y, e.size.Width, e.size.Height)
	for y := gapY / 2; y < e.size.Height+gapY; y += gapY {
		for x := gapX / 2; x < e.size.Width+gapX; x += gapX {
			cvs.Save()
			cvs.Translate(pos.X+x, pos.Y+y)
			cvs.Rotate(rotate)
			cvs.DrawText(w.Content, 0, 0, font, wp)
			cvs.Restore()
		}
	}
	cvs.Restore()
}

func (e *WatermarkElement) Update(newWidget Widget) {
	if nw, ok := newWidget.(*Watermark); ok {
		e.watermark = nw
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
