// codeeditordemo — headless 渲染 StructEditor（易语言式表格化结构编辑器）到 PNG。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/codeeditordemo/
package main

import (
	"fmt"
	"os"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

const cw, ch = 760, 920

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	prog := widget.DemoFactorialProgram()
	prog.Subs[0].Params[0].Ref = "是" // 演示参数「参考」(传址)勾选
	se := widget.NewStructEditor(prog).WithSize(740, 900)
	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(10), BackgroundColor: types.ColorRef(245, 247, 250)},
		se,
	)
	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	widget.OnNeedsRepaint = func() { pipe.MarkNeedsRepaint() }
	pipe.Render() // 先布局（构建子树 + Paint 创建内嵌编辑器）
	ce := findSE(el).(*widget.StructEditorElement)
	ce.Focus()

	// ① 程序集变量表第 1 行「名称」单元格：演示单元格编辑（仍是表格）
	pos := ce.Offset()
	ce.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, pos.X+70, pos.Y+10+24+4+24+13, event.ButtonLeft, 0))
	typeStr(ce, "·改")

	// ② 阶乘「程序」区：点进去→末尾→回车换行→写一句赋值未声明变量「临时」→ 自动声明进局部变量表
	if bed := ce.BodyEditorAt(0); bed != nil {
		bo := bed.Offset()
		ce.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, bo.X+40, bo.Y+12, event.ButtonLeft, 0))
		ce.HandleEvent(event.NewKeyEvent(event.TypeKeyDown, 0, "End", 0))   // 行尾
		ce.HandleEvent(event.NewKeyEvent(event.TypeKeyDown, 0, "Enter", 0)) // 换行
		typeStr(ce, "临时 ＝ 积 × 2")                                         // → 整数型
		ce.HandleEvent(event.NewKeyEvent(event.TypeKeyDown, 0, "Enter", 0)) // 回车→自动声明 临时
		typeStr(ce, "sc")                                                  // 拼音首字母触发补全（sc→输出调试文本…）
	}

	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sk.SaveToPNG("codeeditor_output.png")
	fmt.Println("✅ codeeditor_output.png 已保存")

	// 文本↔表格双向：导出文本 + 解析回来校验无损 round-trip
	txt := prog.Serialize()
	fmt.Println("\n────── 序列化文本（表格→文本）──────")
	fmt.Print(txt)
	fmt.Println("──────────────────────────────────")
	if widget.ParseProgram(txt).Serialize() == txt {
		fmt.Println("round-trip（文本→表格→文本）无损 ✅")
	} else {
		fmt.Println("round-trip 不一致 ❌")
	}
}

func typeStr(el widget.Element, s string) {
	for _, r := range s {
		kc := event.NewKeyEvent(event.TypeKeyChar, uint32(r), string(r), 0)
		kc.Char = r
		el.HandleEvent(kc)
	}
}

func findSE(el widget.Element) widget.Element {
	if _, ok := el.(*widget.StructEditorElement); ok {
		return el
	}
	for _, c := range el.Children() {
		if r := findSE(c); r != nil {
			return r
		}
	}
	return nil
}
