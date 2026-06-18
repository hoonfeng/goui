package main

import (
	"fmt"
	"os"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

func runCheck(name string, got, want float64) int {
	if int(got) != int(want) {
		fmt.Printf("  ❌ %s: got %.0f want %.0f\n", name, got, want)
		return 1
	}
	fmt.Printf("  ✅ %s = %.0f\n", name, got)
	return 0
}

func main() {
	fmt.Println("=== VirtualList Scrollbar 诊断测试 ===\n")
	fails := 0

	//
	// 测试 1：基本高度累计（ItemHeights 源）
	//
	fmt.Println("--- 测试 1: 基本高度累计 ---")
	v1 := &widget.VirtualList{
		ItemCount:   5,
		ItemHeight:  80,
		ItemHeights: []float64{100, 200, 150, 80, 120},
		Height:      300,
		Width:       300,
		RenderItem: func(i int) widget.Widget {
			return &widget.Container{Width: 300, Height: 100}
		},
	}
	el1 := v1.CreateElement().(*widget.VirtualListElement)
	el1.Mount(nil, 0)
	_ = el1.Build()
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: 300},
	}
	_ = el1.Layout(ctx)

	total1 := 0.0
	for _, h := range el1.VL().ItemHeights {
		total1 += h
	}
	fmt.Printf("  ItemHeights[0..4] = %.0f,%.0f,%.0f,%.0f,%.0f\n",
		el1.VL().ItemHeights[0], el1.VL().ItemHeights[1], el1.VL().ItemHeights[2],
		el1.VL().ItemHeights[3], el1.VL().ItemHeights[4])
	fails += runCheck("totalContentH", total1, 500)
	fails += runCheck("maxScroll", el1.MaxScroll(), 200)
	fmt.Println()

	//
	// 测试 2：Update 新 ItemHeights（模拟折叠后重建 VirtualList）
	// chat.go 流程：SetState → Build → scrollMessages 建新 VirtualList → Update
	//
	fmt.Println("--- 测试 2: Update 新 ItemHeights（模拟折叠 item[1] 200→56）---")
	newHeights := []float64{100, 56, 150, 80, 120} // 折叠 item[1]
	v2 := &widget.VirtualList{
		ItemCount:   5,
		ItemHeight:  80,
		ItemHeights: newHeights,
		Height:      300,
		Width:       300,
		RenderItem: v1.RenderItem,
	}
	el1.Update(v2)
	_ = el1.Build()
	_ = el1.Layout(ctx)

	total2 := 0.0
	for _, h := range el1.VL().ItemHeights {
		total2 += h
	}
	fails += runCheck("折叠后 totalContentH", total2, 506)
	fails += runCheck("折叠后 maxScroll", el1.MaxScroll(), 206)
	fmt.Println()

	//
	// 测试 3：Streaming 追加消息（ItemCount+ItemHeights 同时变）
	//
	fmt.Println("--- 测试 3: Streaming 追加 2 条 ---")
	streamHeights := []float64{100, 56, 150, 80, 120, 180, 120}
	v3 := &widget.VirtualList{
		ItemCount:   7,
		ItemHeight:  80,
		ItemHeights: streamHeights,
		Height:      300,
		Width:       300,
		RenderItem: func(i int) widget.Widget {
			return &widget.Container{Width: 300, Height: 100}
		},
	}
	el1.Update(v3)
	_ = el1.Build()
	_ = el1.Layout(ctx)

	total3 := 0.0
	for _, h := range el1.VL().ItemHeights {
		total3 += h
	}
	fails += runCheck("追加后 totalContentH", total3, 806)
	fails += runCheck("追加后 maxScroll", el1.MaxScroll(), 506)
	fmt.Println()

	//
	// 测试 4：Slice 引用 — 验证 chat.go cachedHeights 模式
	//
	fmt.Println("--- 测试 4: Slice 引用（chat.go cachedHeights 模式）---")
	cachedA := []float64{100, 200, 150, 80, 120}
	v4a := &widget.VirtualList{
		ItemCount:   5,
		ItemHeight:  80,
		ItemHeights: cachedA,
		Height:      300, Width: 300,
		RenderItem: func(i int) widget.Widget {
			return &widget.Container{Width: 300, Height: cachedA[i]}
		},
	}
	el4 := v4a.CreateElement().(*widget.VirtualListElement)
	el4.Mount(nil, 0)
	_ = el4.Build()
	_ = el4.Layout(ctx)
	fmt.Printf("  cachedA[0] after Layout: %.0f (Container实际高 = %.0f)\n", cachedA[0], 100.0)
	fmt.Printf("  与 e.vl.ItemHeights 同一指针? %v\n", &cachedA[0] == &el4.VL().ItemHeights[0])
	if cachedA[0] != 100 {
		fmt.Printf("  ⚠️ Layout 把 cachedA[0] 从 %.0f 改为 %.0f！chat.go 的缓存被就地修改！\n", 100.0, cachedA[0])
	}

	// chat.go scrollMessages 重建：新建 cachedB
	cachedB := []float64{100, 200, 150, 80, 120}
	v4b := &widget.VirtualList{
		ItemCount:   5,
		ItemHeight:  80,
		ItemHeights: cachedB,
		Height:      300, Width: 300,
		RenderItem: func(i int) widget.Widget {
			return &widget.Container{Width: 300, Height: cachedB[i]}
		},
	}
	el4.Update(v4b)
	_ = el4.Build()
	_ = el4.Layout(ctx)
	fmt.Printf("  cachedB[0] after 2nd Layout: %.0f\n", cachedB[0])
	// cachedB[0] = 100, Container 实际高 = cachedB[0] = 100
	fmt.Println()

	//
	// 测试 5：关键情景 — 不可见项的折叠高度不会被 Layout 修正！
	//
	fmt.Println("--- 测试 5: 不可见项折叠 → Layout 不修正 → maxScroll 变化 ---")
	bigHeights := make([]float64, 30)
	for i := range bigHeights {
		bigHeights[i] = 200
	}
	// 折叠 items 10~20（不可见，视口只能看约 1.5 项）
	for i := 10; i < 20; i++ {
		bigHeights[i] = 56
	}
	v5 := &widget.VirtualList{
		ItemCount:   30,
		ItemHeight:  80,
		ItemHeights: bigHeights,
		Height:      300,
		Width:       300,
		RenderItem: func(i int) widget.Widget {
			return &widget.Container{Width: 300, Height: 200}
		},
	}
	el5 := v5.CreateElement().(*widget.VirtualListElement)
	el5.Mount(nil, 0)
	_ = el5.Build()
	_ = el5.Layout(ctx)

	total5 := 0.0
	for _, h := range el5.VL().ItemHeights {
		total5 += h
	}
	// 10~19 是 56，其余是 200
	expTotal5 := 200.0*20 + 56.0*10 // 4000 + 560 = 4560
	fails += runCheck("不可见折叠 total", total5, expTotal5)
	fails += runCheck("不可见折叠 maxScroll", el5.MaxScroll(), expTotal5-300)

	// 验证 items 10~19 的 Layout 捕获值
	for i := 10; i < 20; i++ {
		if el5.VL().ItemHeights[i] != 56 {
			fmt.Printf("  ⚠️ 不可见 item[%d] 被 Layout 从 56 修正为 %.0f！\n", i, el5.VL().ItemHeights[i])
			fails++
		}
	}
	fmt.Printf("  不可见 items[10..19] 保持折叠高 56? ")
	all56 := true
	for i := 10; i < 20; i++ {
		if el5.VL().ItemHeights[i] != 56 {
			all56 = false
		}
	}
	if all56 {
		fmt.Println("✅ 是")
	} else {
		fmt.Println("❌ 否（部分被 Layout 修正）")
	}
	fmt.Println()

	//
	// 测试 6：Paint 时独立重算 maxScroll（当 Layout 未更新时）
	//
	fmt.Println("--- 测试 6: Paint 安全网（独立重算 maxScroll vs e.maxScroll）---")
	fmt.Println("  (可视化检查，无断言)")
	fmt.Println()

	if fails > 0 {
		fmt.Printf("\n❌ 失败 %d 个测试\n", fails)
		os.Exit(1)
	}
	fmt.Println("✅ 所有测试通过！")
}
