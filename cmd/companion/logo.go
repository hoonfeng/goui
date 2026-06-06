package main

import (
	"encoding/xml"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// 图标＝运行时**加载** assets/icon.svg（不 embed/硬编码），与 fonts/、libSkiaSharp.dll 一样属外部资源，
// 可换皮不重编。goui 自带 svg 库渲染器不解析 url(#gradient)（渐变描边会变黑），故这里解析原版 SVG 元素后
// 用 goui Skia 画布（支持渐变填充/描边）忠实渲染；仅辉光滤镜(feGaussianBlur)无法复刻（图标尺寸下不可见）。

// assetPath 解析外部资源路径：依次找 exe 目录、cwd 下的 assets/<name> 与 <name>。
func assetPath(name string) string {
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}
	for _, d := range dirs {
		for _, p := range []string{filepath.Join(d, "assets", name), filepath.Join(d, name)} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return filepath.Join("assets", name) // 兜底（让后续 ReadFile 自然报错）
}

type svgStop struct {
	Offset string `xml:"offset,attr"`
	Color  string `xml:"stop-color,attr"`
}
type svgGrad struct {
	ID    string    `xml:"id,attr"`
	X1    string    `xml:"x1,attr"`
	Y1    string    `xml:"y1,attr"`
	X2    string    `xml:"x2,attr"`
	Y2    string    `xml:"y2,attr"`
	Stops []svgStop `xml:"stop"`
}
type rectEl struct {
	X       string `xml:"x,attr"`
	Y       string `xml:"y,attr"`
	W       string `xml:"width,attr"`
	H       string `xml:"height,attr"`
	Rx      string `xml:"rx,attr"`
	Fill    string `xml:"fill,attr"`
	Stroke  string `xml:"stroke,attr"`
	StrokeW string `xml:"stroke-width,attr"`
}
type pathEl struct {
	D       string `xml:"d,attr"`
	Stroke  string `xml:"stroke,attr"`
	StrokeW string `xml:"stroke-width,attr"`
	Fill    string `xml:"fill,attr"`
}
type lineEl struct {
	X1      string `xml:"x1,attr"`
	Y1      string `xml:"y1,attr"`
	X2      string `xml:"x2,attr"`
	Y2      string `xml:"y2,attr"`
	Stroke  string `xml:"stroke,attr"`
	StrokeW string `xml:"stroke-width,attr"`
}
type circleEl struct {
	Cx      string `xml:"cx,attr"`
	Cy      string `xml:"cy,attr"`
	R       string `xml:"r,attr"`
	Fill    string `xml:"fill,attr"`
	Stroke  string `xml:"stroke,attr"`
	StrokeW string `xml:"stroke-width,attr"`
	Opacity string `xml:"opacity,attr"`
}
type iconDoc struct {
	ViewBox string    `xml:"viewBox,attr"`
	Grads   []svgGrad `xml:"defs>linearGradient"`
	Rects   []rectEl  `xml:"rect"`
	Paths   []pathEl  `xml:"path"`
	Lines   []lineEl  `xml:"line"`
	Circles []circleEl `xml:"circle"`
}

func atof(s string) float64 { f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64); return f }
func atofDef(s string, def float64) float64 {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return atof(s)
}
func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func absf(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}

// gradFor 解析 url(#id) 引用到渐变定义。
func gradFor(val string, grads map[string]svgGrad) (svgGrad, bool) {
	if strings.HasPrefix(val, "url(#") {
		id := strings.TrimSuffix(strings.TrimPrefix(val, "url(#"), ")")
		g, ok := grads[id]
		return g, ok
	}
	return svgGrad{}, false
}

// makeGrad 把渐变(objectBoundingBox 0~1 坐标)映射到元素包围盒 → paint.Gradient。
func makeGrad(g svgGrad, bx, by, bw, bh float64) *paint.Gradient {
	x1, y1, x2, y2 := atofDef(g.X1, 0), atofDef(g.Y1, 0), atofDef(g.X2, 1), atofDef(g.Y2, 0)
	gr := &paint.Gradient{
		Type:  paint.GradientLinear,
		Start: types.Point{X: bx + x1*bw, Y: by + y1*bh},
		End:   types.Point{X: bx + x2*bw, Y: by + y2*bh},
	}
	for _, st := range g.Stops {
		o := strings.TrimSpace(st.Offset)
		off := atof(strings.TrimSuffix(o, "%"))
		if strings.HasSuffix(o, "%") {
			off /= 100
		}
		gr.Stops = append(gr.Stops, paint.ColorStop{Offset: off, Color: types.ColorFromHex(st.Color)})
	}
	return gr
}

// fillFor 构造填充画笔（solid 或渐变；none/transparent 返回 false）。
func fillFor(fill string, grads map[string]svgGrad, bx, by, bw, bh float64) (paint.Paint, bool) {
	if fill == "" || fill == "none" || fill == "transparent" {
		return paint.Paint{}, false
	}
	p := paint.DefaultPaint()
	if g, ok := gradFor(fill, grads); ok {
		p.LinearGradient = makeGrad(g, bx, by, bw, bh)
	} else {
		p.Color = types.ColorFromHex(fill)
	}
	return p, true
}

// strokeFor 构造描边画笔（solid 或渐变；none 返回 false）。
func strokeFor(stroke, strokeW string, grads map[string]svgGrad, bx, by, bw, bh, s float64) (paint.Paint, bool) {
	if stroke == "" || stroke == "none" {
		return paint.Paint{}, false
	}
	p := paint.DefaultStrokePaint()
	if strokeW != "" {
		p.StrokeWidth = atof(strokeW) * s
	} else {
		p.StrokeWidth = s
	}
	if g, ok := gradFor(stroke, grads); ok {
		p.LinearGradient = makeGrad(g, bx, by, bw, bh)
	} else {
		p.Color = types.ColorFromHex(stroke)
	}
	return p, true
}

// parsePathML 解析仅含 M/L（绝对）命令的 path d，返回缩放后的点列。
func parsePathML(d string, s float64) []types.Point {
	toks := strings.Fields(strings.NewReplacer("M", " M ", "L", " L ", ",", " ").Replace(d))
	var pts []types.Point
	var nums []float64
	flush := func() {
		for i := 0; i+1 < len(nums); i += 2 {
			pts = append(pts, types.Point{X: nums[i] * s, Y: nums[i+1] * s})
		}
		nums = nil
	}
	for _, t := range toks {
		switch t {
		case "M", "L":
			flush()
		default:
			nums = append(nums, atof(t))
		}
	}
	flush()
	return pts
}

// renderPairIcon 解析 SVG 内容并用 goui Skia 画布(支持渐变)渲染为 size×size 图。
func renderPairIcon(svgContent string, size int) image.Image {
	var doc iconDoc
	if err := xml.Unmarshal([]byte(svgContent), &doc); err != nil {
		return nil
	}
	vbW := 512.0
	if f := strings.Fields(doc.ViewBox); len(f) == 4 {
		vbW = atof(f[2])
	}
	if vbW <= 0 {
		vbW = 512
	}
	s := float64(size) / vbW
	grads := map[string]svgGrad{}
	for _, g := range doc.Grads {
		grads[g.ID] = g
	}

	cv := canvas.NewSkiaCanvas(size, size)
	defer cv.Release()
	cv.Clear(types.Color{}) // 透明底（默认白底会让圆角徽标外露白角）

	for _, r := range doc.Rects { // 背景圆角矩形
		x, y, w, h, rx := atof(r.X)*s, atof(r.Y)*s, atof(r.W)*s, atof(r.H)*s, atof(r.Rx)*s
		if p, ok := fillFor(r.Fill, grads, x, y, w, h); ok {
			cv.DrawRoundedRect(x, y, w, h, rx, p)
		}
		if p, ok := strokeFor(r.Stroke, r.StrokeW, grads, x, y, w, h, s); ok {
			cv.DrawRoundedRect(x, y, w, h, rx, p)
		}
	}
	for _, pe := range doc.Paths { // 尖括号 < >
		pts := parsePathML(pe.D, s)
		if len(pts) < 2 {
			continue
		}
		minx, miny, maxx, maxy := pts[0].X, pts[0].Y, pts[0].X, pts[0].Y
		path := canvas.NewPath()
		path.MoveTo(pts[0].X, pts[0].Y)
		for _, pt := range pts[1:] {
			path.LineTo(pt.X, pt.Y)
			minx, miny = minf(minx, pt.X), minf(miny, pt.Y)
			maxx, maxy = -minf(-maxx, -pt.X), -minf(-maxy, -pt.Y)
		}
		if p, ok := strokeFor(pe.Stroke, pe.StrokeW, grads, minx, miny, maxx-minx, maxy-miny, s); ok {
			cv.DrawPath(path, p)
		}
	}
	for _, l := range doc.Lines { // 等号 =
		x1, y1, x2, y2 := atof(l.X1)*s, atof(l.Y1)*s, atof(l.X2)*s, atof(l.Y2)*s
		bx, by, bw, bh := minf(x1, x2), minf(y1, y2), absf(x2-x1), absf(y2-y1)
		if p, ok := strokeFor(l.Stroke, l.StrokeW, grads, bx, by, bw, bh, s); ok {
			cv.DrawLine(x1, y1, x2, y2, p)
		}
	}
	for _, c := range doc.Circles { // 中心光点
		cx, cy, r := atof(c.Cx)*s, atof(c.Cy)*s, atof(c.R)*s
		op := 1.0
		if c.Opacity != "" {
			op = atof(c.Opacity)
		}
		if c.Fill != "" && c.Fill != "none" && c.Fill != "transparent" {
			p := paint.DefaultPaint()
			p.Color = types.ColorFromHex(c.Fill)
			p.Opacity = op
			cv.DrawCircle(cx, cy, r, p)
		}
		if c.Stroke != "" && c.Stroke != "none" {
			p := paint.DefaultStrokePaint()
			p.Color = types.ColorFromHex(c.Stroke)
			p.Opacity = op
			if c.StrokeW != "" {
				p.StrokeWidth = atof(c.StrokeW) * s
			}
			cv.DrawCircle(cx, cy, r, p)
		}
	}

	cv.Flush()
	src := cv.Image()
	if src == nil {
		return nil
	}
	out := image.NewRGBA(src.Bounds()) // 拷出，免 Release 后失效
	copy(out.Pix, src.Pix)
	return out
}

var (
	pairLogoSrc    image.Image // 光栅化一次缓存（每次 build 复用）
	pairLogoLoaded bool        // 已尝试加载（避免每帧重试缺失文件）
)

// pairLogo 加载并渲染 assets/icon.svg 作标题栏 app 图标——运行时加载外部资源（非 embed），含渐变忠实渲染。
func pairLogo() widget.Widget {
	if !pairLogoLoaded {
		pairLogoLoaded = true
		if data, err := os.ReadFile(assetPath("icon.svg")); err == nil {
			pairLogoSrc = renderPairIcon(string(data), 44) // 2.2× 显示尺寸，下采样更清晰
		}
	}
	if pairLogoSrc == nil {
		return widget.Div(widget.Style{Width: 20, Height: 20}) // 资源缺失兜底空位
	}
	return widget.NewImage(pairLogoSrc).WithSize(20, 20).WithFit(widget.ImageFitContain)
}
