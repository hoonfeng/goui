// Package canvas 定义了 2D 绘制的核心类型和接口。
package canvas

import "github.com/user/goui/internal/types"

// PathOp 路径操作类型
type PathOp int

const (
	PathOpMoveTo       PathOp = iota // 移动到指定点
	PathOpLineTo                     // 画直线到指定点
	PathOpQuadTo                     // 二次贝塞尔曲线
	PathOpCubicTo                    // 三次贝塞尔曲线
	PathOpArc                        // 圆弧
	PathOpClose                      // 闭合路径
)

// PathItem 路径中的一个操作
type PathItem struct {
	Op    PathOp
	Points [6]float64 // 根据操作类型使用不同数量的坐标
}

// Path 代表一个绘制路径
type Path struct {
	Items   []PathItem
	Current types.Point
	Start   types.Point
}

// NewPath 创建新路径
func NewPath() *Path {
	return &Path{
		Items: make([]PathItem, 0),
	}
}

// MoveTo 将当前点移动到指定位置（不画线）
func (p *Path) MoveTo(x, y float64) {
	p.Items = append(p.Items, PathItem{
		Op:     PathOpMoveTo,
		Points: [6]float64{x, y, 0, 0, 0, 0},
	})
	p.Current = types.Point{X: x, Y: y}
	p.Start = types.Point{X: x, Y: y}
}

// LineTo 从当前点画直线到指定点
func (p *Path) LineTo(x, y float64) {
	p.Items = append(p.Items, PathItem{
		Op:     PathOpLineTo,
		Points: [6]float64{x, y, 0, 0, 0, 0},
	})
	p.Current = types.Point{X: x, Y: y}
}

// QuadTo 二次贝塞尔曲线
func (p *Path) QuadTo(cx, cy, x, y float64) {
	p.Items = append(p.Items, PathItem{
		Op:     PathOpQuadTo,
		Points: [6]float64{cx, cy, x, y, 0, 0},
	})
	p.Current = types.Point{X: x, Y: y}
}

// CubicTo 三次贝塞尔曲线
func (p *Path) CubicTo(cx1, cy1, cx2, cy2, x, y float64) {
	p.Items = append(p.Items, PathItem{
		Op:     PathOpCubicTo,
		Points: [6]float64{cx1, cy1, cx2, cy2, x, y},
	})
	p.Current = types.Point{X: x, Y: y}
}

// Arc 添加圆弧
func (p *Path) Arc(x, y, radius, startAngle, endAngle float64, clockwise bool) {
	p.Items = append(p.Items, PathItem{
		Op:     PathOpArc,
		Points: [6]float64{x, y, radius, startAngle, endAngle, boolToFloat(clockwise)},
	})
}

// Close 闭合路径（回到起点）
func (p *Path) Close() {
	p.Items = append(p.Items, PathItem{Op: PathOpClose})
	p.Current = p.Start
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// Rect 添加矩形路径
func (p *Path) Rect(x, y, w, h float64) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
}

// RoundedRect 添加圆角矩形路径
func (p *Path) RoundedRect(x, y, w, h, r float64) {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	// 上边 + 右上角
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.CubicTo(x+w-r*0.5, y, x+w, y+r*0.5, x+w, y+r)
	// 右边 + 右下角
	p.LineTo(x+w, y+h-r)
	p.CubicTo(x+w, y+h-r*0.5, x+w-r*0.5, y+h, x+w-r, y+h)
	// 下边 + 左下角
	p.LineTo(x+r, y+h)
	p.CubicTo(x+r*0.5, y+h, x, y+h-r*0.5, x, y+h-r)
	// 左边 + 左上角
	p.LineTo(x, y+r)
	p.CubicTo(x, y+r*0.5, x+r*0.5, y, x+r, y)
	p.Close()
}

// Circle 添加圆形路径
func (p *Path) Circle(cx, cy, radius float64) {
	// 使用四段三次贝塞尔曲线近似圆
	const kappa = 0.5522847498
	r := radius
	p.MoveTo(cx-r, cy)
	p.CubicTo(cx-r, cy-r*kappa, cx-r*kappa, cy-r, cx, cy-r)
	p.CubicTo(cx+r*kappa, cy-r, cx+r, cy-r*kappa, cx+r, cy)
	p.CubicTo(cx+r, cy+r*kappa, cx+r*kappa, cy+r, cx, cy+r)
	p.CubicTo(cx-r*kappa, cy+r, cx-r, cy+r*kappa, cx-r, cy)
	p.Close()
}
