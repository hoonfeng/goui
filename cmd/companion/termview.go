// 终端网格渲染：把 vterm 的 cell 网格自绘成 goui widget（PaintLayer，事件透传）。
// 等宽字体逐格画 背景块 + 字符，按 SGR 配色；块状半透明光标。
// 标准终端集成的「显示侧」——进程侧(pty/ConPTY) + 屏幕模型(vterm) 已就绪，这里把网格画出来。

//go:build windows

package main

import (
	"github.com/user/goui/cmd/companion/vterm"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// keyToVT 把按键事件转成写进 PTY 的 VT 字节：KeyChar→可打印字符；
// KeyDown→Ctrl 组合（控制字符）/ 回车/退格/Tab/Esc/方向键/Home/End/Page 等的 VT 序列。
func keyToVT(ev *event.KeyEvent) []byte {
	if ev.Type() == event.TypeKeyChar {
		if ev.Char >= 0x20 { // 可打印字符（含 Shift 后的）
			return []byte(string(ev.Char))
		}
		return nil
	}
	// KeyDown：Ctrl+字母 → 控制字符（Ctrl+C=0x03 等）
	if ev.Mods&event.ModCtrl != 0 && len(ev.Key) == 1 {
		c := ev.Key[0]
		switch {
		case c >= 'A' && c <= 'Z':
			return []byte{c - 'A' + 1}
		case c >= 'a' && c <= 'z':
			return []byte{c - 'a' + 1}
		}
	}
	switch ev.Key {
	case "Enter", "Return":
		return []byte("\r")
	case "Backspace":
		return []byte{0x7f}
	case "Tab":
		return []byte("\t")
	case "Escape":
		return []byte{0x1b}
	case "Delete":
		return []byte("\x1b[3~")
	case "ArrowUp":
		return []byte("\x1b[A")
	case "ArrowDown":
		return []byte("\x1b[B")
	case "ArrowRight":
		return []byte("\x1b[C")
	case "ArrowLeft":
		return []byte("\x1b[D")
	case "Home":
		return []byte("\x1b[H")
	case "End":
		return []byte("\x1b[F")
	case "PageUp":
		return []byte("\x1b[5~")
	case "PageDown":
		return []byte("\x1b[6~")
	}
	return nil
}

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

// paintVTGrid 在画布上画 vterm 网格：等宽逐格 背景块 + 字符（SGR 配色/粗体）+ 块状半透明光标。
// scrollOff 为回看偏移（0=贴底/实时，>0=向上看历史）：渲染「组合缓冲」(滚回历史+当前屏)的对应窗口。
func paintVTGrid(cvs canvas.Canvas, x, y, w, h float64, vt *vterm.Terminal, font canvas.Font, scrollOff int) {
	cw, ch := termCellSize(cvs, font)
	cols, rows := vt.Size()
	start := vt.ScrollbackLen() - scrollOff // 视窗顶行在组合缓冲中的下标
	for vr := 0; vr < rows; vr++ {
		cy := y + float64(vr)*ch
		if cy > y+h {
			break
		}
		row := vt.RowAt(start + vr) // 可能为 nil（越界）→ 该行空
		for c := 0; c < cols; c++ {
			cx := x + float64(c)*cw
			var cell vterm.Cell
			if c < len(row) {
				cell = row[c]
			} else {
				cell = vterm.Cell{Ch: ' ', FG: vterm.DefaultColor(), BG: vterm.DefaultColor()}
			}
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
	if scrollOff == 0 { // 仅贴底/实时时画光标
		ccx, ccy := vt.Cursor()
		cur := paint.DefaultPaint()
		cur.Color = types.ColorFromRGBA(cText.R, cText.G, cText.B, 150)
		cvs.DrawRect(x+float64(ccx)*cw, y+float64(ccy)*ch, cw, ch, cur)
	}
}

// termGridView 把 vterm 网格画成自绘 widget（铺满父约束；只显示不抓输入，预览/无输入场景用）。
func termGridView(vt *vterm.Terminal) widget.Widget {
	return &widget.PaintLayer{OnPaint: func(cvs canvas.Canvas, x, y, w, h float64) {
		paintVTGrid(cvs, x, y, w, h, vt, termGridFontNow(), 0)
	}}
}
