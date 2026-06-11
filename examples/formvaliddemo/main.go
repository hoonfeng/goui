// formvaliddemo — 验证 Form 校验：必填红 * + 错误红字 + 红边框。
// 联动(改密码触发确认密码重校验)/动态增删/单字段 validateField 需窗口实测；
// headless 这里 render 后调 Validate() 触发全量校验来展示错误显示。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/formvaliddemo/
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
	ch = 380
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	var form *widget.Form
	form = widget.NewForm(
		&widget.FormItem{Label: "用户名", Prop: "username", Placeholder: "请输入用户名", Required: true, Rules: []widget.FormRule{{MinLen: 3}}},
		&widget.FormItem{Label: "邮箱", Prop: "email", Placeholder: "name@example.com", Rules: []widget.FormRule{{Required: true, Pattern: `^[^@]+@[^@]+\.[^@]+$`, Message: "邮箱格式不正确"}}},
		&widget.FormItem{Label: "密码", Prop: "password", Placeholder: "请输入密码", Required: true, Triggers: []string{"confirm"}},
		&widget.FormItem{Label: "确认密码", Prop: "confirm", Placeholder: "再次输入密码", Required: true, Rules: []widget.FormRule{{Validator: func(v string) string {
			if v != form.FieldValue("password") {
				return "两次输入不一致"
			}
			return ""
		}}}},
	).WithLabelWidth(80)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 10},
		widget.H3("Form 校验：单字段 / 动态增删 / 规则联动"),
		form,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	pipe.Render()          // 建树（form.element 回填）
	form.Validate()        // 全量校验：空必填 → 各字段报错
	pipe.MarkNeedsLayout() // 错误行使高度变化
	pipe.Render()          // 重绘显示错误

	if err := sk.SaveToPNG("formvalid_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ formvalid_demo.png 已保存")
}
