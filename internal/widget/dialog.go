package widget

import (
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Dialog 对话框（L3 模态覆盖层）─────────────────────────────
// 1:1 复刻 el-dialog：全屏半透明遮罩 + 居中(或顶部偏移)的白色面板，
// 面板含 header(标题 + ✕) / body(内容) / footer(按钮组，右对齐)。
// 实现思路：「铺满视口 + 自绘遮罩 + 居中子面板」。面板本身用 Div/Flex 拼，
// 所以它天然继承 L1/L2 的全部 CSS 能力(圆角/阴影/内边距/弹性对齐)——
// 内容是真正渲染的子树，按钮是真正复用的 Button，不再像旧实现那样在 Paint 里手绘。
// 配合 ShowDialog/CloseDialog(dialog_overlay.go) 可挂到 OverlayHost 顶层真正弹出。

type Dialog struct {
	StatelessWidget
	Title     string
	Body      Widget      // 主体内容
	Footer    []Widget    // 底部按钮组(右对齐)
	Width     float64     // 面板宽(默认 420)
	Top       float64     // 距顶距离(默认 15% 视高；Center 时忽略)
	Center    bool        // 面板垂直居中
	ShowClose bool        // 右上角 ✕(经 NewDialog 创建时默认 true)
	MaskColor types.Color // 遮罩色(默认半透明黑 50%)
	OnClose   func()      // 关闭回调(点 ✕ 或点遮罩)
}

// NewDialog 创建对话框(默认显示右上角关闭 ✕)。
func NewDialog(title string, body Widget) *Dialog {
	return &Dialog{Title: title, Body: body, ShowClose: true}
}

// 对话框/抽屉深色主题（包级，零值=默认白底浅色）。深色应用统一对话框外观，思路同 SetMenuTheme。
var (
	dialogThemeBG    types.Color
	dialogThemeText  types.Color
	dialogThemeMuted types.Color
)

// SetDialogTheme 设置对话框/抽屉主体配色（bg 面板底、text 标题、muted 关闭符/次要）。零值恢复默认白底。
func SetDialogTheme(bg, text, muted types.Color) {
	dialogThemeBG, dialogThemeText, dialogThemeMuted = bg, text, muted
}

func (d *Dialog) WithFooter(btns ...Widget) *Dialog   { d.Footer = btns; return d }
func (d *Dialog) WithWidth(w float64) *Dialog         { d.Width = w; return d }
func (d *Dialog) WithTop(t float64) *Dialog           { d.Top = t; return d }
func (d *Dialog) Centered() *Dialog                   { d.Center = true; return d }
func (d *Dialog) WithMaskColor(c types.Color) *Dialog { d.MaskColor = c; return d }
func (d *Dialog) WithOnClose(fn func()) *Dialog       { d.OnClose = fn; return d }

func (d *Dialog) CreateElement() Element {
	return &DialogElement{BaseElement: BaseElement{widget: d}, dialog: d}
}

// modalRows 把标题/内容/按钮拼成 [header, body, footer] 三行，
// 供 Dialog 与 Drawer 共用(同构：header[标题+✕] / body / footer[按钮右对齐])。
// 用 divCore 直接吃 []Widget，省去 ...interface{} 展开。
func modalRows(title string, body Widget, footer []Widget, showClose, fillGap bool) []Widget {
	rows := make([]Widget, 0, 3)

	titleColor, closeColor := elTextPrimary(), elTextSecondary()
	if dialogThemeText != (types.Color{}) {
		titleColor = dialogThemeText
	}
	if dialogThemeMuted != (types.Color{}) {
		closeColor = dialogThemeMuted
	}
	// header：标题(左) + 弹簧 + ✕(右)
	headerKids := []Widget{
		&Text{
			Text:  title,
			Color: titleColor,
			Font:  canvas.Font{Family: "sans-serif", Size: 16, Weight: canvas.FontWeightBold},
		},
		SpacerDiv(),
	}
	if showClose {
		headerKids = append(headerKids, &Text{
			Text:  "✕",
			Color: closeColor,
			Font:  canvas.Font{Family: "sans-serif", Size: 16},
		})
	}
	rows = append(rows, divCore(
		Style{Padding: types.EdgeInsetsLTRB(20, 16, 16, 10), FlexDirection: "row", AlignItems: "center"},
		headerKids,
	))

	// body
	if body != nil {
		rows = append(rows, divCore(
			Style{Padding: types.EdgeInsetsLTRB(20, 6, 20, 10)},
			[]Widget{body},
		))
	}

	// fillGap：在 body 与 footer 之间插弹簧，把 footer 推到容器底部(Drawer 用；
	// 需配合容器撑满高度——见 DrawerElement.Layout 给 content 的 tight 高度约束)
	if fillGap {
		rows = append(rows, SpacerDiv())
	}

	// footer：弹簧把按钮推到右侧
	if len(footer) > 0 {
		footerKids := make([]Widget, 0, len(footer)+1)
		footerKids = append(footerKids, SpacerDiv())
		footerKids = append(footerKids, footer...)
		rows = append(rows, divCore(
			Style{Padding: types.EdgeInsetsLTRB(20, 4, 16, 16), FlexDirection: "row", Gap: 10},
			footerKids,
		))
	}

	return rows
}

// buildPanel 把内容拼成白色圆角面板(带阴影)。
func (d *Dialog) buildPanel() Widget {
	width := d.Width
	if width <= 0 {
		width = 420
	}
	st := Style{
		Width:           width,
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderRadius:    8,
		Shadow:          &paint.Shadow{Offset: types.Point{X: 0, Y: 12}, Blur: 32, Color: types.ColorFromRGBA(0, 0, 0, 38)},
	}
	if dialogThemeBG != (types.Color{}) { // 深色主题：换底 + 细边定形
		c := dialogThemeBG
		st.BackgroundColor = &c
		if dialogThemeMuted != (types.Color{}) {
			m := dialogThemeMuted
			st.BorderColor = &m
			st.BorderWidth = 1
		}
	}
	return divCore(st, modalRows(d.Title, d.Body, d.Footer, d.ShowClose, false))
}

type DialogElement struct {
	BaseElement
	dialog     *Dialog
	panel      Element
	panelSize  types.Size
	panelScale float64 // 面板进场缩放(0.92→1)
	enterInit  bool
	enterCtrl  *animation.Controller
}

// ensureEnter 首次绘制时启动面板进场缩放(配合 Overlay 的 fade，做出"蹦出来"的弹出感)。
func (e *DialogElement) ensureEnter() {
	if e.enterInit {
		return
	}
	e.enterInit = true
	e.panelScale = 0.85
	e.enterCtrl = animation.NewController(320*time.Millisecond, animation.EaseOutCubic)
	e.enterCtrl.OnUpdate = func(v float64) {
		e.panelScale = 0.85 + 0.15*v
		e.MarkNeedsPaint()
	}
	e.enterCtrl.Start()
}

func (e *DialogElement) Build() []Element {
	e.panel = CreateElementFor(e.dialog.buildPanel())
	e.panel.Mount(e, 0)
	e.children = []Element{e.panel}
	return e.children
}

func (e *DialogElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	// 铺满视口(模态层)
	maxW := ctx.Constraints.MaxWidth
	maxH := ctx.Constraints.MaxHeight
	if maxW >= float64(1<<30) {
		maxW = 600
	}
	if maxH >= float64(1<<30) {
		maxH = 400
	}
	e.size = types.Size{Width: maxW, Height: maxH}

	// 面板按内容测量(松约束，限制在视口内)
	res := e.panel.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
		MinWidth: 0, MaxWidth: maxW, MinHeight: 0, MaxHeight: maxH,
	}})
	e.panelSize = res.Size

	// 水平居中；垂直按 Center / Top / 默认 15% 视高
	px := (e.size.Width - e.panelSize.Width) / 2
	if px < 0 {
		px = 0
	}
	var py float64
	switch {
	case e.dialog.Center:
		py = (e.size.Height - e.panelSize.Height) / 2
	case e.dialog.Top > 0:
		py = e.dialog.Top
	default:
		py = e.size.Height * 0.15
	}
	if py < 0 {
		py = 0
	}
	e.panel.SetPosition(types.Point{X: px, Y: py})

	return layout.LayoutResult{Size: e.size}
}

func (e *DialogElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()

	// 半透明遮罩铺满
	maskColor := e.dialog.MaskColor
	if maskColor == (types.Color{}) {
		maskColor = types.ColorFromRGBA(0, 0, 0, 128) // 黑 50%
	}
	mp := paint.DefaultPaint()
	mp.Color = maskColor
	cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, mp)

	// 居中面板(进场缩放，以面板中心为基准)
	e.ensureEnter()
	if e.panelScale > 0 && e.panelScale < 1 {
		pp := e.panel.Offset()
		ps := e.panel.Size()
		cx := pp.X + ps.Width/2
		cy := pp.Y + ps.Height/2
		cvs.Save()
		cvs.Translate(cx, cy)
		cvs.Scale(e.panelScale, e.panelScale)
		cvs.Translate(-cx, -cy)
		e.panel.Paint(cvs, offset)
		cvs.Restore()
	} else {
		e.panel.Paint(cvs, offset)
	}
}

// HandleEvent 点击面板之外(遮罩区)触发关闭(对应 close-on-click-modal)。
func (e *DialogElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeMouseDown {
		if me, ok := ev.(*event.MouseEvent); ok && e.dialog.OnClose != nil {
			pp := e.panel.Offset()
			ps := e.panel.Size()
			inPanel := me.X >= pp.X && me.X <= pp.X+ps.Width &&
				me.Y >= pp.Y && me.Y <= pp.Y+ps.Height
			// ✕ 热区：面板右上角 44×44
			closeHot := e.dialog.ShowClose &&
				me.X >= pp.X+ps.Width-44 && me.X <= pp.X+ps.Width &&
				me.Y >= pp.Y && me.Y <= pp.Y+44
			if !inPanel || closeHot {
				e.dialog.OnClose()
				return true
			}
		}
	}
	return false
}

func (e *DialogElement) Update(newWidget Widget) {
	if nd, ok := newWidget.(*Dialog); ok {
		e.dialog = nd
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
