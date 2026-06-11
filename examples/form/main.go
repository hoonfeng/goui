// form — 验证 batch④ 收官组件：Form 表单校验(el-form 核心)。
// 实时校验/提交需窗口实测；此处 headless 触发 Validate() 展示失败红边框+红字。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/form/
package main

import (
	"fmt"
	"os"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

const (
	cw = 440
	ch = 300
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	form := widget.NewForm(
		&widget.FormItem{Label: "用户名", Placeholder: "请输入用户名", Required: true},
		&widget.FormItem{Label: "邮箱", Placeholder: "请输入邮箱", Rules: []widget.FormRule{
			{Required: true, Pattern: `^[\w.\-]+@[\w.\-]+\.\w+$`, Message: "邮箱格式不正确"},
		}},
		&widget.FormItem{Label: "密码", Placeholder: "请输入密码", Rules: []widget.FormRule{
			{Required: true, MinLen: 6, Message: "密码至少 6 位"},
		}},
	).WithLabelWidth(70)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 12},
		widget.H3("Form 表单校验（提交触发·失败红边框+红字）"),
		form,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil { // 初始渲染(建树，无错)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	form.Validate()         // 空值提交 → 全报必填
	pipe.MarkNeedsLayout()  // 错误行增高，需重排
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("form_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ form_demo.png 已保存")
}
