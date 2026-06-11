// 动画特性演示画廊：点击按钮，观察多种动画同时过渡——
// 尺寸/颜色、淡入淡出、缩放(回弹)、旋转，以及按钮 hover 平滑反馈。
//
// 运行方式（需要 CGO + libSkiaSharp.dll + fonts/ 在工作目录）:
//
//	$env:CGO_ENABLED='1'; go run ./examples/anim/
package main

import (
	"runtime"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/app"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
	_ "github.com/hoonfeng/goui/internal/window/win32"
)

// AnimDemo 是演示用的有状态根组件。
type AnimDemo struct {
	widget.StatefulWidget
}

func (d *AnimDemo) CreateState() widget.State { return &animDemoState{} }

type animDemoState struct {
	widget.BaseState
	on bool
}

func box(col types.Color, w, h float64) widget.Widget {
	c := col
	return &widget.Container{Width: w, Height: h, BorderRadius: 8, Background: &widget.PaintWidget{Color: &c}}
}

func row(label string, child widget.Widget) widget.Widget {
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 6, 0, 6)},
		widget.HBox(
			func() widget.Widget {
				t := widget.NewText(label, types.ColorFromRGB(90, 90, 90))
				return t
			}(),
			widget.SpacerDiv(),
			child,
		),
	)
}

func (s *animDemoState) Build(ctx widget.BuildContext) widget.Widget {
	on := s.on

	blue := types.ColorFromRGB(66, 133, 244)
	red := types.ColorFromRGB(234, 67, 53)

	// 各动画的目标值随 on 切换
	cw, ch, ccol, radius := 80.0, 56.0, blue, 8.0
	opacity, scale, rot := 1.0, 1.0, 0.0
	if on {
		cw, ch, ccol, radius = 220.0, 56.0, red, 28.0
		opacity, scale, rot = 0.25, 1.6, 180.0
	}

	return widget.Div(
		widget.Style{Padding: types.EdgeInsets(32)},
		widget.H2("goui 动画特性演示"),
		widget.P("点击按钮，下面的元素会同时以不同动画过渡；按钮自身的 hover 也是平滑的。"),

		row("尺寸/颜色/圆角", &widget.AnimatedContainer{
			Width: cw, Height: ch, Radius: radius, Color: &ccol,
			Duration: 400 * time.Millisecond, Curve: animation.EaseInOutCubic,
		}),
		row("淡入淡出", &widget.AnimatedOpacity{
			Opacity: opacity, Duration: 400 * time.Millisecond,
			Child: box(types.ColorFromRGB(52, 168, 83), 56, 56),
		}),
		row("缩放(回弹)", &widget.AnimatedScale{
			Scale: scale, Duration: 450 * time.Millisecond, Curve: animation.EaseOutBack,
			Child: box(types.ColorFromRGB(251, 188, 4), 56, 56),
		}),
		row("旋转", &widget.AnimatedRotation{
			Rotation: rot, Duration: 500 * time.Millisecond, Curve: animation.EaseInOutCubic,
			Child: box(types.ColorFromRGB(156, 39, 176), 56, 56),
		}),

		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 16, 0, 0)},
			widget.NewButton("切换 / Toggle", func() { s.on = !s.on; s.SetState() }),
		),
	)
}

func main() {
	runtime.LockOSThread()
	application := app.NewApplication()
	application.SetRootWidget(&AnimDemo{})
	cfg := app.DefaultConfig()
	cfg.Title = "goui — Animation Gallery"
	application.Run(cfg)
}
