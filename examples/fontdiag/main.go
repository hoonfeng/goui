// fontdiag — 排查字形缺失(方块/豆腐块)：打印各符号的主字体覆盖与 fallback，并渲染成图。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/fontdiag/
package main

import (
	"fmt"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

func main() {
	chars := []rune{'▾', '▼', '▲', '◀', '▶', '‹', '›', '★', '☆', '✓', '✗', '✎', '⎘', '📄', '🖼', '🎵'}

	for _, r := range chars {
		fmt.Printf("U+%04X %c  覆盖=%-5v  fallback=%q\n",
			r, r, canvas.RuneCoveredByPrimary(r), canvas.FallbackFamily(r))
	}

	sk := canvas.NewSkiaCanvas(560, 90)
	defer sk.Release()
	bg := paint.DefaultPaint()
	bg.Color = types.ColorFromRGB(255, 255, 255)
	sk.DrawRect(0, 0, 560, 90, bg)
	font := canvas.DefaultFont()
	font.Size = 30
	canvas.DrawTextAligned(sk, string(chars), types.Rect{X: 12, Y: 0, Width: 540, Height: 90},
		font, types.ColorFromRGB(0, 0, 0), canvas.HAlignLeft, canvas.VAlignMiddle)
	sk.Flush()
	if err := sk.SaveToPNG("fontdiag.png"); err != nil {
		fmt.Println("save err:", err)
	}
	fmt.Println("✅ fontdiag.png 已保存")
}
