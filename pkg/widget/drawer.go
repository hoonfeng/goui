package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Drawer 抽屉（L3 模态边缘面板）──────────────────────────────
// 1:1 复刻 el-drawer：全屏遮罩 + 从某一边缘贴边滑出的面板。
// 与 Dialog 同构(共用 modalRows 拼 header/body/footer)，差别在于面板贴边、占满该边。
// 实现：「铺满视口 + 自绘遮罩 + 自绘贴边白底 + 内容顶部对齐」。
// 配合 ShowDialog/ShowDrawer(交互层) 可挂到 OverlayHost 顶层真正弹出。

type Drawer struct {
	StatelessWidget
	Title      string
	Body       Widget
	Footer     []Widget
	Direction  string      // "right"(默认)/"left"/"top"/"bottom"
	Size       float64     // 抽屉厚度(left/right 为宽，top/bottom 为高)，默认 320
	ShowClose  bool        // 右上角 ✕(经 NewDrawer 创建时默认 true)
	PanelColor types.Color // 抽屉背景(默认白)
	MaskColor  types.Color // 遮罩色(默认半透明黑 50%)
	OnClose    func()      // 关闭回调(点遮罩区)
}

func NewDrawer(title string, body Widget) *Drawer {
	return &Drawer{Title: title, Body: body, Direction: "right", ShowClose: true}
}

func (d *Drawer) WithFooter(btns ...Widget) *Drawer { d.Footer = btns; return d }
func (d *Drawer) WithDirection(dir string) *Drawer  { d.Direction = dir; return d }
func (d *Drawer) WithSize(s float64) *Drawer        { d.Size = s; return d }
func (d *Drawer) WithOnClose(fn func()) *Drawer     { d.OnClose = fn; return d }

func (d *Drawer) CreateElement() Element {
	return &DrawerElement{BaseElement: BaseElement{widget: d}, drawer: d}
}

// buildContent 抽屉面板内容(header/body/footer 竖排，无背景；白底由 Element 自绘)。
func (d *Drawer) buildContent() Widget {
	return divCore(Style{}, modalRows(d.Title, d.Body, d.Footer, d.ShowClose, true))
}

type DrawerElement struct {
	BaseElement
	drawer    *Drawer
	content   Element
	panelRect types.Rect // 贴边面板矩形(相对本元素左上)
}

func (e *DrawerElement) Build() []Element {
	e.content = CreateElementFor(e.drawer.buildContent())
	e.content.Mount(e, 0)
	e.children = []Element{e.content}
	return e.children
}

func (e *DrawerElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	maxW := ctx.Constraints.MaxWidth
	maxH := ctx.Constraints.MaxHeight
	if maxW >= float64(1<<30) {
		maxW = 600
	}
	if maxH >= float64(1<<30) {
		maxH = 400
	}
	e.size = types.Size{Width: maxW, Height: maxH}

	size := e.drawer.Size
	if size <= 0 {
		size = 320
	}
	dir := e.drawer.Direction
	if dir == "" {
		dir = "right"
	}

	// 贴边面板矩形
	var r types.Rect
	switch dir {
	case "left":
		r = types.Rect{X: 0, Y: 0, Width: size, Height: maxH}
	case "top":
		r = types.Rect{X: 0, Y: 0, Width: maxW, Height: size}
	case "bottom":
		r = types.Rect{X: 0, Y: maxH - size, Width: maxW, Height: size}
	default: // right
		r = types.Rect{X: maxW - size, Y: 0, Width: size, Height: maxH}
	}
	e.panelRect = r

	// 内容撑满面板矩形(tight 高度)——使 modalRows 的弹簧能把 footer 压到底
	e.content.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
		MinWidth: 0, MaxWidth: r.Width, MinHeight: r.Height, MaxHeight: r.Height,
	}})
	e.content.SetPosition(types.Point{X: r.X, Y: r.Y})

	return layout.LayoutResult{Size: e.size}
}

func (e *DrawerElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()

	// 遮罩铺满
	maskColor := e.drawer.MaskColor
	if maskColor == (types.Color{}) {
		maskColor = types.ColorFromRGBA(0, 0, 0, 128)
	}
	mp := paint.DefaultPaint()
	mp.Color = maskColor
	cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, mp)

	// 贴边白底
	panelColor := e.drawer.PanelColor
	if panelColor == (types.Color{}) {
		panelColor = types.ColorFromRGB(255, 255, 255)
	}
	pp := paint.DefaultPaint()
	pp.Color = panelColor
	cvs.DrawRect(pos.X+e.panelRect.X, pos.Y+e.panelRect.Y, e.panelRect.Width, e.panelRect.Height, pp)

	// 内容
	e.content.Paint(cvs, offset)
}

// HandleEvent 点击面板之外(遮罩区)触发关闭。
func (e *DrawerElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeMouseDown {
		if me, ok := ev.(*event.MouseEvent); ok && e.drawer.OnClose != nil {
			pos := e.Offset()
			rx := pos.X + e.panelRect.X
			ry := pos.Y + e.panelRect.Y
			inPanel := me.X >= rx && me.X <= rx+e.panelRect.Width &&
				me.Y >= ry && me.Y <= ry+e.panelRect.Height
			// ✕ 热区：面板右上角 44×44
			closeHot := e.drawer.ShowClose &&
				me.X >= rx+e.panelRect.Width-44 && me.X <= rx+e.panelRect.Width &&
				me.Y >= ry && me.Y <= ry+44
			if !inPanel || closeHot {
				e.drawer.OnClose()
				return true
			}
		}
	}
	return false
}

func (e *DrawerElement) Update(newWidget Widget) {
	if nd, ok := newWidget.(*Drawer); ok {
		e.drawer = nd
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
