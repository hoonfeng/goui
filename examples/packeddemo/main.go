// packeddemo — 验证快速拼装组件：Statistic/Descriptions/Timeline/Skeleton/Result。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/packeddemo/
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
	cw = 460
	ch = 720
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	green := types.ColorFromRGB(103, 194, 58)
	blue := types.ColorFromRGB(64, 158, 255)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 18},
		widget.H3("快速拼装：Statistic / Descriptions / Timeline / Skeleton / Result"),
		widget.NewSpace(48,
			widget.NewStatistic("总销售额", "126,560").WithSuffix("元"),
			widget.NewStatistic("订单数", "1,280"),
		),
		widget.NewDescriptions(
			widget.DescItem{Label: "用户名", Value: "张三"},
			widget.DescItem{Label: "手机号", Value: "138****8888"},
			widget.DescItem{Label: "地址", Value: "北京市朝阳区建国路"},
		).WithTitle("用户信息"),
		widget.NewTimeline(
			widget.TimelineItem{Time: "01-01 创建", Content: "订单已创建", Color: green},
			widget.TimelineItem{Time: "01-02 发货", Content: "包裹已发出", Color: blue},
			widget.TimelineItem{Time: "01-03 签收", Content: "用户已签收"},
		),
		widget.NewResult("success", "操作成功").WithSubTitle("数据已保存到云端"),
		widget.NewSkeleton(3),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("packed_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ packed_demo.png 已保存")
}
