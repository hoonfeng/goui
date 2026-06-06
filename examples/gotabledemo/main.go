// gotabledemo — 接 Go：把真实 Go 源码解析进表格化编辑器(ParseGo)渲染，再导出回 Go(ToGo)。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/gotabledemo/
package main

import (
	"fmt"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const goSrc = `package main

var 全局计数 int

// 阶乘 递归求 n 的阶乘
func 阶乘(n int) int {
	积 := 1
	for i := 1; i <= n; i++ {
		积 *= i
	}
	return 积
}

// 除法 返回商与余数
func 除法(
	a int, // 被除数
	b int, // 除数
) (
	商 int, // 整数商
	余 int, // 余数
) {
	return a / b, a % b
}
`

const cw, ch = 760, 640

func main() {
	p, err := widget.ParseGo(goSrc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ParseGo:", err)
		os.Exit(1)
	}
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()
	se := widget.NewStructEditor(p).WithLang("go").WithSize(740, 620) // Go 模式：逻辑区按 Go 高亮
	root := widget.Div(widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(10), BackgroundColor: types.ColorRef(245, 247, 250)}, se)
	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	widget.OnNeedsRepaint = func() { pipe.MarkNeedsRepaint() }
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sk.SaveToPNG("gotable_output.png")
	fmt.Println("✅ gotable_output.png 已保存")

	fmt.Println("\n────── 表格 → Go 源码（ToGo）──────")
	fmt.Print(p.ToGo())
	fmt.Println("──────────────────────────────────")
	if p2, err := widget.ParseGo(p.ToGo()); err == nil && p2.ToGo() == p.ToGo() {
		fmt.Println("round-trip（Go→表格→Go）idempotent 稳定 ✅")
	}
}
