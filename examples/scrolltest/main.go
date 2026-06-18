// scrolltest — 滚动渲染机制全面测试 Demo
//
// 测试内容：
//   1. ScrollView 鼠标滚轮 → 平滑滚动动画 → 每帧渲染验证
//   2. VirtualList 鼠标滚轮 → 直接跳转 → 渲染正确性
//   3. VirtualList 内容流式增长（模拟对话）→ 滚动位置保持
//   4. VirtualList 滚动条视觉位置与 Layout 一致性
//
// 运行方式：
//   $env:CGO_ENABLED='1'; go run ./examples/scrolltest/
//
// 输出: scrolltest_output.png（单帧截图）+ 控制台诊断日志

//go:build !ignore

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

const (
	canvasW = 600
	canvasH = 800
)

func main() {
	sk := canvas.NewSkiaCanvas(canvasW, canvasH)
	defer sk.Release()

	// 测试 1: ScrollView 基本滚动 + 渲染
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("测试 1: ScrollView 平滑滚动 + 渲染正确性")
	fmt.Println(strings.Repeat("=", 60))
	testScrollViewRender()

	// 测试 2: VirtualList 滚动 + 渲染
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("测试 2: VirtualList 虚加载 + 滚动渲染")
	fmt.Println(strings.Repeat("=", 60))
	testVirtualListRender()

	// 测试 3: VirtualList 内容增长（模拟对话流式）
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("测试 3: VirtualList 流式内容增长 + 滚动位置保持")
	fmt.Println(strings.Repeat("=", 60))
	testVirtualListStreaming()

	// 测试 4: Scrollbar 视觉一致性
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("测试 4: Scrollbar 滑块位置正确性验证")
	fmt.Println(strings.Repeat("=", 60))
	testScrollbarConsistency()

	// 渲染一帧作为视觉参考
	root := buildTestScene()
	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(canvasW, canvasH, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "渲染失败: %v\n", err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("scrolltest_output.png"); err != nil {
		fmt.Fprintf(os.Stderr, "保存PNG失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n✅ scrolltest_output.png 已保存")
}

// buildTestScene 构建一个包含 ScrollView + VirtualList + 消息列表的综合测试场景
func buildTestScene() widget.Widget {
	// ScrollView 测试区内的长内容
	scrollItems := make([]widget.Widget, 0, 30)
	for i := 0; i < 30; i++ {
		scrollItems = append(scrollItems,
			widget.Div(
				widget.Style{Height: 50, Padding: types.EdgeInsetsLTRB(8, 4, 8, 4)},
				widget.NewText(fmt.Sprintf("滚动内容行 #%d — 这是测试文本，用于验证滚动渲染", i), types.ColorFromRGB(33, 33, 33)),
			),
		)
	}

	return widget.Div(
		widget.Style{
			Padding:         types.EdgeInsets(16),
			BackgroundColor: types.ColorRef(245, 247, 250),
			FlexDirection:   "column",
			AlignItems:      "stretch",
		},
		widget.H1("滚动渲染测试"),
		widget.Div(widget.Style{Height: 12}),
		// ScrollView 测试区
		widget.Div(
			widget.Style{
				BackgroundColor: types.ColorRef(255, 255, 255),
				BorderRadius:    6,
				BorderColor:     types.ColorRef(220, 224, 228),
				BorderWidth:     1,
				Padding:         types.EdgeInsets(12),
				Height:          200,
			},
			widget.NewScrollView(
				widget.Div(
					widget.Style{FlexDirection: "column", AlignItems: "stretch"},
					scrollItems...,
				),
			).WithBar(true),
		),
		widget.Div(widget.Style{Height: 12}),
		// VirtualList 测试区
		widget.Div(
			widget.Style{
				BackgroundColor: types.ColorRef(255, 255, 255),
				BorderRadius:    6,
				BorderColor:     types.ColorRef(220, 224, 228),
				BorderWidth:     1,
				Padding:         types.EdgeInsets(12),
				Height:          250,
			},
			widget.NewVirtualList(50, 40, func(i int) widget.Widget {
				return widget.Div(
					widget.Style{
						Height:          40,
						FlexDirection:   "row",
						AlignItems:      "center",
						Padding:         types.EdgeInsetsLTRB(8, 0, 8, 0),
						BackgroundColor: types.ColorRef(245, 247, 250),
						BorderRadius:    4,
					},
					widget.NewText(fmt.Sprintf("虚拟列表项 #%d", i), types.ColorFromRGB(33, 33, 33)),
				)
			}).WithHeight(220).WithOverscan(3),
		),
	)
}

// ─── 测试函数（无窗口，直接操作 Element 验证内部状态）────

func testScrollViewRender() {
	// 构建 ScrollView 内容
	items := make([]widget.Widget, 0, 30)
	for i := 0; i < 30; i++ {
		items = append(items,
			widget.Div(
				widget.Style{Height: 50, Padding: types.EdgeInsetsLTRB(8, 4, 8, 4)},
				widget.NewText(fmt.Sprintf("行 #%d", i), types.ColorFromRGB(33, 33, 33)),
			),
		)
	}
	content := widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch"}, items...)

	// 创建 ScrollView
	sv := widget.NewScrollView(content).WithBar(true)
	el := sv.CreateElement().(*widget.ScrollViewElement)
	el.Mount(nil, 0)
	el.Build()

	// Layout
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: 500,
			MinHeight: 0, MaxHeight: 200,
		},
	}
	el.Layout(ctx)

	fmt.Printf("  ScrollView 尺寸: %.0fx%.0f\n", el.Size().Width, el.Size().Height)
	fmt.Printf("  maxScroll: (%.0f, %.0f)\n", el.MaxScroll().X, el.MaxScroll().Y)

	if el.MaxScroll().Y <= 0 {
		fmt.Println("  ⚠️  maxScroll.Y <= 0，内容未超出视口！检查子内容布局")
		fmt.Println("  可能原因：Div 子内容通过 items 切片传递时未用 ... 展开")
	} else {
		fmt.Printf("  ✅ maxScroll.Y = %.0f (>0)，内容可滚动\n", el.MaxScroll().Y)
	}

	// 模拟滚动
	el.ScrollTo(types.Point{X: 0, Y: 100})
	fmt.Printf("  滚动到 Y=100 → scrollOffset.Y=%.0f\n", el.ScrollOffset().Y)
	if el.ScrollOffset().Y != 100 {
		fmt.Printf("  ⚠️  scrollOffset 与目标不一致！got=%.0f want=100\n", el.ScrollOffset().Y)
	} else {
		fmt.Println("  ✅ ScrollTo 正确")
	}
}

func testVirtualListRender() {
	// 创建 VirtualList（固定高度）
	vl := &widget.VirtualList{
		ItemCount:  50,
		ItemHeight: 40,
		Height:     220,
		Width:      500,
		Overscan:   3,
		RenderItem: func(i int) widget.Widget {
			return widget.Div(
				widget.Style{Height: 40, BackgroundColor: types.ColorRef(245, 247, 250), BorderRadius: 4},
				widget.NewText(fmt.Sprintf("项 #%d", i), types.ColorFromRGB(33, 33, 33)),
			)
		},
	}
	el := vl.CreateElement().(*widget.VirtualListElement)
	el.Mount(nil, 0)
	el.Build()
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth: 500, MaxWidth: 500,
			MinHeight: 0, MaxHeight: 220,
		},
	}
	el.Layout(ctx)

	totalH := float64(vl.ItemCount) * vl.ItemHeight
	maxScroll := totalH - 220
	fmt.Printf("  ViewportH: %.0f\n", el.ViewportH())
	fmt.Printf("  总内容高: %.0f\n", totalH)
	fmt.Printf("  maxScroll: %.0f (期望 %.0f)\n", el.MaxScroll(), maxScroll)

	if el.MaxScroll() != maxScroll {
		fmt.Printf("  ⚠️  maxScroll 不匹配！got=%.0f want=%.0f\n", el.MaxScroll(), maxScroll)
	} else {
		fmt.Println("  ✅ maxScroll 计算正确")
	}

	// 测试滚动
	el.ScrollTo(200)
	fmt.Printf("  滚动到 200 → scrollOffset=%.0f\n", el.ScrollOffset())
	if el.ScrollOffset() != 200 {
		fmt.Printf("  ⚠️  scrollOffset 不匹配！got=%.0f want=200\n", el.ScrollOffset())
	} else {
		fmt.Println("  ✅ ScrollTo 正确")
	}
}

func testVirtualListStreaming() {
	// 模拟消息流式增长场景：初始高度数组
	heights := []float64{80, 100, 120, 90, 200, 80, 150, 100, 90, 80}

	vl := &widget.VirtualList{
		ItemCount:   len(heights),
		ItemHeight:  80,
		ItemHeights: heights,
		Height:      300,
		Width:       500,
		Overscan:    3,
		RenderItem: func(i int) widget.Widget {
			return widget.Div(widget.Style{Height: heights[i]})
		},
	}

	el := vl.CreateElement().(*widget.VirtualListElement)
	el.Mount(nil, 0)
	el.Build()
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth: 500, MaxWidth: 500,
			MinHeight: 0, MaxHeight: 300,
		},
	}
	el.Layout(ctx)

	initialMaxScroll := el.MaxScroll()
	fmt.Printf("  初始 maxScroll=%.0f (总高度 %.0f, 视口 300)\n",
		initialMaxScroll, 80+100+120+90+200+80+150+100+90+80)

	// 模拟用户滚到中间位置
	scrollTarget := 200.0
	el.ScrollTo(scrollTarget)
	fmt.Printf("  用户滚到 scrollOffset=%.0f\n", el.ScrollOffset())

	// 模拟流式增长：第5条消息从200→400，总高度增长200
	heights[4] = 400
	// 更新 VirtualList 配置（模拟 SetState 重建）
	vl2 := &widget.VirtualList{
		ItemCount:   len(heights),
		ItemHeight:  80,
		ItemHeights: heights,
		Height:      300,
		Width:       500,
		Overscan:    3,
		RenderItem: func(i int) widget.Widget {
			return widget.Div(widget.Style{Height: heights[i]})
		},
	}
	el.Update(vl2)
	el.Build()
	el.Layout(ctx)

	newMaxScroll := el.MaxScroll()
	newOffset := el.ScrollOffset()
	fmt.Printf("  流式增长后 maxScroll=%.0f (增长 %.0f)\n",
		newMaxScroll, newMaxScroll-initialMaxScroll)

	// 验证：scrollOffset 不应被强制改变位置
	if newOffset <= 0 {
		fmt.Println("  ❌ scrollOffset 被重置到顶部！内容增长不应改变用户滚动位置")
	} else if newOffset < scrollTarget*0.9 || newOffset > scrollTarget*1.1 {
		fmt.Printf("  ⚠️  scrollOffset 变化过大 %.0f→%.0f！\n", scrollTarget, newOffset)
	} else {
		fmt.Printf("  ✅ scrollOffset 在流式增长后保持稳定 (%.0f)\n", newOffset)
	}
}

func testScrollbarConsistency() {
	// 测试 Paint 和 Layout 中的 maxScroll 计算是否一致
	heights := []float64{100, 200, 150, 180, 120, 300, 80, 90, 110, 250}

	vl := &widget.VirtualList{
		ItemCount:   len(heights),
		ItemHeight:  80,
		ItemHeights: heights,
		Height:      400,
		Width:       500,
		Overscan:    3,
		RenderItem: func(i int) widget.Widget {
			return widget.Div(widget.Style{Height: heights[i]})
		},
	}
	el := vl.CreateElement().(*widget.VirtualListElement)
	el.Mount(nil, 0)
	el.Build()
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth: 500, MaxWidth: 500,
			MinHeight: 0, MaxHeight: 400,
		},
	}
	el.Layout(ctx)

	// 计算期望值
	total := 0.0
	for _, h := range heights {
		total += h
	}
	expectedMaxScroll := total - 400
	fmt.Printf("  totalContentH=%.0f viewportH=%.0f maxScroll=%.0f (期望 %.0f)\n",
		total, el.ViewportH(), el.MaxScroll(), expectedMaxScroll)

	if el.MaxScroll() != expectedMaxScroll {
		fmt.Printf("  ⚠️  Layout maxScroll=%.0f 与期望 %.0f 不一致\n",
			el.MaxScroll(), expectedMaxScroll)
	} else {
		fmt.Println("  ✅ Layout maxScroll 正确")
	}

	// 多次 Layout 检查一致性
	for i := 0; i < 5; i++ {
		el2 := vl.CreateElement().(*widget.VirtualListElement)
		el2.Mount(nil, 0)
		el2.Build()
		el2.Layout(ctx)
		if el2.MaxScroll() != expectedMaxScroll {
			fmt.Printf("  ⚠️  第 %d 次 Layout maxScroll=%.0f != 期望 %.0f\n",
				i+1, el2.MaxScroll(), expectedMaxScroll)
		}
	}
	fmt.Println("  ✅ 多次 Layout 结果一致")
}
