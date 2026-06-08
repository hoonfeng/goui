// 终端网格渲染：把 vterm 的 cell 网格自绘成 goui widget（PaintLayer，事件透传）。
// 等宽字体逐格画 背景块 + 字符，按 SGR 配色；块状半透明光标。
// 标准终端集成的「显示侧」——进程侧(pty/ConPTY) + 屏幕模型(vterm) 已就绪，这里把网格画出来。

//go:build windows

package main

import (
	"github.com/user/goui/cmd/companion/vterm"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

var termGridFont = canvas.Font{Family: "Consolas", Size: 13}

// vtColor 把 vterm 颜色转 goui 颜色；默认色用传入的 def（终端前景/背景）。
func vtColor(c vterm.Color, def types.Color) types.Color {
	if c.Default {
		return def
	}
	return types.ColorFromRGB(c.R, c.G, c.B)
}

// termCellSize 当前终端字体下的格宽/行高（等宽）。cvs 可测量则用真实字宽，否则估算。
func termCellSize(cvs canvas.Canvas, font canvas.Font) (cw, ch float64) {
	cw = font.Size * 0.6
	if cvs != nil {
		if w := cvs.MeasureText("M", font).Width; w > 0 {
			cw = w
		}
	}
	return cw, font.Size * 1.45
}

// termGridFontNow 据终端字号设置取等宽字体。
func termGridFontNow() canvas.Font {
	f := termGridFont
	if theSettings.TermFontSize > 0 {
		f.Size = float64(theSettings.TermFontSize)
	}
	return f
}

// termGridView 把 vterm 网格画成自绘 widget（铺满父约束）。
func termGridView(vt *vterm.Terminal) widget.Widget {
	return &widget.PaintLayer{OnPaint: func(cvs canvas.Canvas, x, y, w, h float64) {
		font := termGridFontNow()
		cw, ch := termCellSize(cvs, font)
		cols, rows := vt.Size()
		for r := 0; r < rows; r++ {
			cy := y + float64(r)*ch
			if cy > y+h {
				break
			}
			for c := 0; c < cols; c++ {
				cell := vt.Cell(r, c)
				cx := x + float64(c)*cw
				if !cell.BG.Default { // 背景块
					bp := paint.DefaultPaint()
					bp.Color = vtColor(cell.BG, *cEditor)
					cvs.DrawRect(cx, cy, cw+0.5, ch, bp)
				}
				if cell.Ch != ' ' && cell.Ch != 0 { // 字符
					tp := paint.DefaultPaint()
					tp.Color = vtColor(cell.FG, cText)
					f := font
					if cell.Bold {
						f.Weight = canvas.FontWeightBold
					}
					cvs.DrawText(string(cell.Ch), cx, canvas.BaselineFor(cy, ch, f.Size, canvas.VAlignMiddle), f, tp)
				}
			}
		}
		// 块状半透明光标
		ccx, ccy := vt.Cursor()
		cur := paint.DefaultPaint()
		cur.Color = types.ColorFromRGBA(cText.R, cText.G, cText.B, 150)
		cvs.DrawRect(x+float64(ccx)*cw, y+float64(ccy)*ch, cw, ch, cur)
	}}
}
