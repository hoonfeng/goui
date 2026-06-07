package widget

import (
	"math"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
	svgpath "github.com/hoonfeng/svg/path"
)

// ─── Lucide 矢量图标 ──────────────────────────────────────────
//
// 全项目统一的真实图标方案：用开源 Lucide 图标（24×24 viewBox、线性 stroke 风格），
// 把每个图标的 SVG path d 通过 svg/path 解析为命令，直接发射成 canvas.Path 的原生
// 曲线（CubicTo/QuadTo，弧线按标准算法分解为三次贝塞尔），缩放平移后用 Canvas.DrawPath
// 描边渲染——精确曲线交给 Skia、可任意缩放无锯齿、可动态着色、走 GPU，无光栅化/无折线展平。
//
// 严禁用 emoji / Unicode 字符当图标（见 cmd/companion/AGENTS.md 铁律）。

// lucideIcons 是图标名 → 该图标的一组 SVG path d（Lucide 原始数据，含 circle 等已折算为 path）。
// 一个图标可由多条子路径组成（如 close = 两笔）。按需从 Lucide 官方（MIT）增补。
var lucideIcons = map[string][]string{
	"x":             {"M18 6 6 18", "m6 6 12 12"},
	"minus":         {"M5 12h14"},
	"plus":          {"M5 12h14", "M12 5v14"},
	"check":         {"M20 6 9 17l-5-5"},
	"chevron-down":  {"m6 9 6 6 6-6"},
	"chevron-up":    {"m18 15-6-6-6 6"},
	"chevron-left":  {"m15 18-6-6 6-6"},
	"chevron-right": {"m9 18 6-6-6-6"},
	"menu":          {"M4 6h16", "M4 12h16", "M4 18h16"},
	"search":        {"m21 21-4.34-4.34", "M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16Z"},
	"settings":      {"M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2Z", "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"},
	"folder":        {"M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"},
	"file":          {"M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z", "M14 2v4a2 2 0 0 0 2 2h4"},
	"folder-open":    {"m6 14 1.5-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.54 6a2 2 0 0 1-1.95 1.5H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3.9a2 2 0 0 1 1.69.9l.81 1.2a2 2 0 0 0 1.67.9H18a2 2 0 0 1 2 2v2"},
	"folder-plus":    {"M12 10v6", "M9 13h6", "M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"},
	"file-code":      {"M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z", "M14 2v5a1 1 0 0 0 1 1h5", "M10 12.5 8 15l2 2.5", "m14 12.5 2 2.5-2 2.5"},
	"file-text":      {"M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z", "M14 2v5a1 1 0 0 0 1 1h5", "M10 9H8", "M16 13H8", "M16 17H8"},
	"braces":         {"M8 3H7a2 2 0 0 0-2 2v5a2 2 0 0 1-2 2 2 2 0 0 1 2 2v5c0 1.1.9 2 2 2h1", "M16 21h1a2 2 0 0 0 2-2v-5c0-1.1.9-2 2-2a2 2 0 0 1-2-2V5a2 2 0 0 0-2-2h-1"},
	"message-square": {"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2Z"},
	"terminal":      {"m4 17 6-6-6-6", "M12 19h8"},
	"send":          {"m22 2-7 20-4-9-9-4Z", "M22 2 11 13"},
	"trash-2":       {"M3 6h18", "M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2", "M10 11v6", "M14 11v6"},
	"paperclip":     {"m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"},
	"user":          {"M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2", "M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0Z"},
	"bot":           {"M12 8V4H8", "M9 8h6a2 2 0 0 1 2 2v6a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2v-6a2 2 0 0 1 2-2Z", "M2 14h2", "M20 14h2", "M15 13v2", "M9 13v2"},
	"square-pen":      {"M12 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7", "M18.375 2.625a1 1 0 0 1 3 3l-9.013 9.014a2 2 0 0 1-.853.505l-2.873.84a.5.5 0 0 1-.62-.62l.84-2.873a2 2 0 0 1 .506-.852z"},
	"list":            {"M3 12h.01", "M3 18h.01", "M3 6h.01", "M8 6h13", "M8 12h13", "M8 18h13"},
	"arrow-left-right": {"M8 3 4 7l4 4", "M4 7h16", "m16 21 4-4-4-4", "M20 17H4"},
	"download":        {"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4", "M7 10l5 5 5-5", "M12 15V3"},
	"shield":          {"M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"},
	"shield-off":      {"M19.7 14a6.9 6.9 0 0 0 .3-2V5l-8-3-3.18 1.19", "m2 2 20 20", "M4.73 4.73 4 5v7c0 6 8 10 8 10a20.29 20.29 0 0 0 5.62-4.38"},
	"shield-alert":    {"M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z", "M12 8v4", "M12 16h.01"},
	"refresh-cw":      {"M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8", "M21 3v5h-5", "M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16", "M8 16H3v5"},
	"lightbulb":       {"M15 14c.2-1 .7-1.7 1.5-2.5 1-.9 1.5-2.2 1.5-3.5A6 6 0 0 0 6 8c0 1 .2 2.2 1.5 3.5.7.7 1.3 1.5 1.5 2.5", "M9 18h6", "M10 22h4"},
	"git-branch":    {"M6 3v12", "M18 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z", "M6 21a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z", "M15 6a9 9 0 0 0-9 9"},
	"arrow-down-to-line": {"M12 17V3", "m6 11 6 6 6-6", "M19 21H5"}, // 拉取
	"arrow-up-from-line":  {"m18 9-6-6-6 6", "M12 3v14", "M5 21h14"},  // 推送
	"circle-check":        {"M21.801 10A10 10 0 1 1 17 3.335", "m9 11 3 3L22 4"}, // 工作区干净
	"globe":               {"M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z", "M2 12h20", "M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"}, // web_fetch
	"circle":              {"M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z"},      // 计划 pending
	"circle-x":            {"M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z", "m15 9-6 6", "m9 9 6 6"}, // kill_process
	"circle-help":         {"M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z", "M9.09 9a3 3 0 0 1 5.82 1c0 2-3 3-3 3", "M12 17h.01"}, // ask_user 问答卡
	"loader-circle":       {"M21 12a9 9 0 1 1-6.219-8.56"},                   // 计划 in_progress
	"list-checks":         {"m3 17 2 2 4-4", "m3 7 2 2 4-4", "M13 6h8", "M13 12h8", "M13 18h8"}, // 计划标题
	"panel-left":    {"M3 5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z", "M9 3v18"},
	"panel-right":   {"M3 5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z", "M15 3v18"},
	"panel-bottom":  {"M3 5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z", "M3 15h18"},
	"square":        {"M3 5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"},
	"minimize":      {"M8 3v3a2 2 0 0 1-2 2H3", "M21 8h-3a2 2 0 0 1-2-2V3", "M3 16h3a2 2 0 0 1 2 2v3", "M16 21v-3a2 2 0 0 1 2-2h3"}, // 还原（角内收）
	"copy":          {"M10 8h10a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H10a2 2 0 0 1-2-2V10a2 2 0 0 1 2-2z", "M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"},
	"clipboard":     {"M9 2h6a1 1 0 0 1 1 1v2a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1z", "M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"},
	"eraser":        {"m7 21-4.3-4.3c-1-1-1-2.5 0-3.4l9.6-9.6c1-1 2.5-1 3.4 0l5.6 5.6c1 1 1 2.5 0 3.4L13 21", "M22 21H7", "m5 11 9 9"},
}

// IconOpt 配置 Lucide 图标（链式可选项）。
type IconOpt func(*lucideIcon)

// IconSize 设置图标边长（像素，默认 20）。
func IconSize(px float64) IconOpt { return func(i *lucideIcon) { i.size = px } }

// IconColor 设置图标描边颜色（默认跟随主题文字色）。
func IconColor(c types.Color) IconOpt { return func(i *lucideIcon) { i.color = &c } }

// IconStroke 设置描边宽度（24 viewBox 下的值，默认 2，与 Lucide 一致）。
func IconStroke(w float64) IconOpt { return func(i *lucideIcon) { i.stroke = w } }

// Lucide 创建一个 Lucide 矢量图标。用法极简：widget.Lucide("search")，
// 或 widget.Lucide("x", widget.IconSize(16), widget.IconColor(red))。
func Lucide(name string, opts ...IconOpt) Widget {
	ic := &lucideIcon{name: name, size: 20, stroke: 2}
	for _, o := range opts {
		o(ic)
	}
	return ic
}

type lucideIcon struct {
	StatelessWidget
	name   string
	size   float64
	stroke float64
	color  *types.Color
}

func (l *lucideIcon) CreateElement() Element {
	return &lucideIconElement{BaseElement: BaseElement{widget: l}, icon: l}
}

func (e *lucideIconElement) Update(nw Widget) {
	e.widget = nw
	e.dirty = true
	if l, ok := nw.(*lucideIcon); ok {
		e.icon = l
	}
}

type lucideIconElement struct {
	BaseElement
	icon *lucideIcon
}

func (e *lucideIconElement) Build() []Element { return nil }

// IgnoreHitTest 图标是装饰：命中测试穿透它，让点击落到可交互的祖先（如 Button）。
func (e *lucideIconElement) IgnoreHitTest() bool { return true }

func (e *lucideIconElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	s := e.icon.size
	e.size = types.Size{Width: s, Height: s}
	return layout.LayoutResult{Size: e.size}
}

func (e *lucideIconElement) Paint(cvs canvas.Canvas, offset types.Point) {
	col := elTextPrimary()
	if e.icon.color != nil {
		col = *e.icon.color
	}
	pos := e.Offset()
	PaintLucide(cvs, e.icon.name, pos.X, pos.Y, e.icon.size, e.icon.stroke, col)
}

// PaintLucide 在画布 (x,y) 处画一个 size×size 的 Lucide 图标（描边宽 stroke、颜色 color）。
// 供需要在自绘 Paint 里嵌矢量图标的组件复用（如 Menu 项左图标），免去字符图标（铁律①）。
func PaintLucide(cvs canvas.Canvas, name string, x, y, size, stroke float64, color types.Color) {
	prims := lucideIcons[name]
	if len(prims) == 0 {
		return // 未知图标：留空（缺图标去补 path，不用字符替代）
	}
	scale := size / 24.0 // Lucide viewBox 24×24
	sp := paint.DefaultStrokePaint()
	sp.Color = color
	if stroke <= 0 {
		stroke = 2
	}
	sp.StrokeWidth = stroke * scale
	origin := types.Point{X: x, Y: y}
	for _, d := range prims {
		for _, gp := range svgPathToGouiPaths(d, scale, origin) {
			cvs.DrawPath(gp, sp)
		}
	}
}

// svgPathToGouiPaths 把一条 SVG path d 解析为若干 goui Path（每个子路径一条），直接发射
// canvas.Path 的原生曲线：C/S→CubicTo、Q/T→QuadTo、A→标准 arc→cubic 分解、L/H/V→LineTo。
// 全程在 SVG 用户坐标下计算，仅发射顶点时经 at() 缩放 scale、平移到 origin——曲线精度交给
// Skia，任意尺寸都平滑。不再走 svgpath 的折线展平（那是先前图标畸形的根因）。
func svgPathToGouiPaths(d string, scale float64, origin types.Point) []*canvas.Path {
	sp, err := svgpath.ParsePath(d)
	if err != nil || sp == nil {
		return nil
	}
	at := func(x, y float64) (float64, float64) {
		return origin.X + x*scale, origin.Y + y*scale
	}
	var paths []*canvas.Path
	var cur *canvas.Path
	// 状态全部维护在 SVG 用户坐标系：当前点、子路径起点、上一条曲线末控制点（供 S/T 反射）。
	var curX, curY, startX, startY float64
	var ctrlX, ctrlY float64
	var prevKind byte // 0=非曲线, 'C'=三次曲线, 'Q'=二次曲线

	flush := func() {
		if cur != nil {
			paths = append(paths, cur)
			cur = nil
		}
	}
	ensure := func() { // 没有显式 MoveTo 时（极少），用当前点起一条子路径
		if cur == nil {
			cur = canvas.NewPath()
			dx, dy := at(curX, curY)
			cur.MoveTo(dx, dy)
		}
	}
	line := func(x, y float64) {
		ensure()
		dx, dy := at(x, y)
		cur.LineTo(dx, dy)
		curX, curY, prevKind = x, y, 0
	}
	cubic := func(x1, y1, x2, y2, ex, ey float64) {
		ensure()
		c1x, c1y := at(x1, y1)
		c2x, c2y := at(x2, y2)
		dx, dy := at(ex, ey)
		cur.CubicTo(c1x, c1y, c2x, c2y, dx, dy)
		curX, curY, ctrlX, ctrlY, prevKind = ex, ey, x2, y2, 'C'
	}
	quad := func(x1, y1, ex, ey float64) {
		ensure()
		c1x, c1y := at(x1, y1)
		dx, dy := at(ex, ey)
		cur.QuadTo(c1x, c1y, dx, dy)
		curX, curY, ctrlX, ctrlY, prevKind = ex, ey, x1, y1, 'Q'
	}

	for _, cmd := range sp.Commands {
		switch c := cmd.(type) {
		case *svgpath.MoveToCommand:
			if c.Relative {
				curX, curY = curX+c.X, curY+c.Y
			} else {
				curX, curY = c.X, c.Y
			}
			flush()
			cur = canvas.NewPath()
			dx, dy := at(curX, curY)
			cur.MoveTo(dx, dy)
			startX, startY, prevKind = curX, curY, 0
		case *svgpath.LineToCommand:
			if c.Relative {
				line(curX+c.X, curY+c.Y)
			} else {
				line(c.X, c.Y)
			}
		case *svgpath.HorizontalLineToCommand:
			if c.Relative {
				line(curX+c.X, curY)
			} else {
				line(c.X, curY)
			}
		case *svgpath.VerticalLineToCommand:
			if c.Relative {
				line(curX, curY+c.Y)
			} else {
				line(curX, c.Y)
			}
		case *svgpath.CubicCurveToCommand:
			if c.Relative {
				cubic(curX+c.X1, curY+c.Y1, curX+c.X2, curY+c.Y2, curX+c.X, curY+c.Y)
			} else {
				cubic(c.X1, c.Y1, c.X2, c.Y2, c.X, c.Y)
			}
		case *svgpath.SmoothCubicCurveToCommand:
			x1, y1 := curX, curY // 默认第一控制点=当前点
			if prevKind == 'C' { // 反射上一条三次曲线的末控制点
				x1, y1 = 2*curX-ctrlX, 2*curY-ctrlY
			}
			if c.Relative {
				cubic(x1, y1, curX+c.X2, curY+c.Y2, curX+c.X, curY+c.Y)
			} else {
				cubic(x1, y1, c.X2, c.Y2, c.X, c.Y)
			}
		case *svgpath.QuadraticCurveToCommand:
			if c.Relative {
				quad(curX+c.X1, curY+c.Y1, curX+c.X, curY+c.Y)
			} else {
				quad(c.X1, c.Y1, c.X, c.Y)
			}
		case *svgpath.SmoothQuadraticCurveToCommand:
			x1, y1 := curX, curY
			if prevKind == 'Q' {
				x1, y1 = 2*curX-ctrlX, 2*curY-ctrlY
			}
			if c.Relative {
				quad(x1, y1, curX+c.X, curY+c.Y)
			} else {
				quad(x1, y1, c.X, c.Y)
			}
		case *svgpath.ArcToCommand:
			ex, ey := c.X, c.Y
			if c.Relative {
				ex, ey = curX+c.X, curY+c.Y
			}
			ensure()
			for _, s := range arcToCubics(curX, curY, c.RX, c.RY, c.XAxisRotation, c.LargeArc, c.Sweep, ex, ey) {
				cubic(s[0], s[1], s[2], s[3], s[4], s[5])
			}
			prevKind = 0 // 弧线之后不参与 S/T 反射
		case *svgpath.ClosePathCommand:
			if cur != nil {
				cur.Close()
			}
			curX, curY, prevKind = startX, startY, 0
		}
	}
	flush()
	return paths
}

// arcToCubics 把 SVG 椭圆弧（端点参数化）转为一串三次贝塞尔段（SVG 用户坐标，每段 6 个控制坐标）。
// 标准 W3C 实现（SVG 1.1 附录 F.6）：端点→中心参数化，按 ≤90° 切段，每段单条 cubic 近似（误差极小）。
func arcToCubics(x1, y1, rx, ry, phiDeg float64, largeArc, sweep bool, x2, y2 float64) [][6]float64 {
	// 退化：半径为 0 或起终点重合 → 退化为直线（两控制点取终点）
	if rx == 0 || ry == 0 || (x1 == x2 && y1 == y2) {
		return [][6]float64{{x2, y2, x2, y2, x2, y2}}
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	phi := phiDeg * math.Pi / 180
	cosP, sinP := math.Cos(phi), math.Sin(phi)
	// 1) 旋转到椭圆主轴坐标系下的中点偏移
	dx2, dy2 := (x1-x2)/2, (y1-y2)/2
	x1p := cosP*dx2 + sinP*dy2
	y1p := -sinP*dx2 + cosP*dy2
	// 2) 半径过小则等比放大
	if l := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry); l > 1 {
		s := math.Sqrt(l)
		rx, ry = rx*s, ry*s
	}
	// 3) 撇坐标系下的圆心
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	co := 0.0
	if den != 0 {
		co = math.Sqrt(math.Max(0, num/den))
	}
	if largeArc == sweep {
		co = -co
	}
	cxp := co * rx * y1p / ry
	cyp := -co * ry * x1p / rx
	// 4) 还原到原坐标系的圆心
	cx := cosP*cxp - sinP*cyp + (x1+x2)/2
	cy := sinP*cxp + cosP*cyp + (y1+y2)/2
	// 5) 起始角与扫掠角
	ux, uy := (x1p-cxp)/rx, (y1p-cyp)/ry
	vx, vy := (-x1p-cxp)/rx, (-y1p-cyp)/ry
	theta1 := svgAngle(1, 0, ux, uy)
	dTheta := svgAngle(ux, uy, vx, vy)
	if !sweep && dTheta > 0 {
		dTheta -= 2 * math.Pi
	} else if sweep && dTheta < 0 {
		dTheta += 2 * math.Pi
	}
	// 6) ≤90° 切段，每段一条 cubic
	segs := int(math.Ceil(math.Abs(dTheta)/(math.Pi/2) - 1e-9))
	if segs == 0 {
		segs = 1
	}
	delta := dTheta / float64(segs)
	t := 4.0 / 3.0 * math.Tan(delta/4)
	point := func(a float64) (float64, float64) {
		ca, sa := math.Cos(a), math.Sin(a)
		return cx + rx*ca*cosP - ry*sa*sinP, cy + rx*ca*sinP + ry*sa*cosP
	}
	deriv := func(a float64) (float64, float64) {
		ca, sa := math.Cos(a), math.Sin(a)
		return -rx*sa*cosP - ry*ca*sinP, -rx*sa*sinP + ry*ca*cosP
	}
	out := make([][6]float64, 0, segs)
	ang := theta1
	px, py := point(ang)
	for i := 0; i < segs; i++ {
		a2 := ang + delta
		ex, ey := point(a2)
		d1x, d1y := deriv(ang)
		d2x, d2y := deriv(a2)
		out = append(out, [6]float64{px + t*d1x, py + t*d1y, ex - t*d2x, ey - t*d2y, ex, ey})
		ang, px, py = a2, ex, ey
	}
	return out
}

// svgAngle 返回向量 (ux,uy)→(vx,vy) 的有符号夹角（弧度），用于弧线起始角/扫掠角。
func svgAngle(ux, uy, vx, vy float64) float64 {
	l := math.Sqrt((ux*ux + uy*uy) * (vx*vx + vy*vy))
	if l == 0 {
		return 0
	}
	c := (ux*vx + uy*vy) / l
	if c > 1 {
		c = 1
	} else if c < -1 {
		c = -1
	}
	a := math.Acos(c)
	if ux*vy-uy*vx < 0 {
		a = -a
	}
	return a
}
