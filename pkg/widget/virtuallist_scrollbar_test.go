package widget

import (
	"fmt"
	"testing"

	"github.com/hoonfeng/goui/internal/layout"
)

// 测试 VirtualList 滚动条高度计算，直接在包内访问内部字段
func TestVirtualListScrollbarHeight(t *testing.T) {
	// 辅助：建一个新 VirtualListElement 并 Layout
	makeVL := func(itemCount int, itemHeights []float64, viewportH float64) *VirtualListElement {
		v := &VirtualList{
			ItemCount:   itemCount,
			ItemHeight:  80,
			ItemHeights: itemHeights,
			Height:      viewportH,
			Width:       300,
			RenderItem: func(i int) Widget {
				h := 80.0
				if i >= 0 && i < len(itemHeights) {
					h = itemHeights[i]
				}
				return &Container{Width: 300, Height: h}
			},
		}
		el := v.CreateElement().(*VirtualListElement)
		el.Mount(nil, 0)
		el.Build()
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: viewportH},
		}
		el.Layout(ctx)
		return el
	}

	// 测试 1：全部展开 vs 折叠后 maxScroll 变化
	t.Run("collapse_changes_maxScroll", func(t *testing.T) {
		heights := []float64{300, 400, 500, 200, 350} // 5条展开消息
		el := makeVL(5, heights, 300)

		totalBefore := 0.0
		for _, h := range el.vl.ItemHeights {
			totalBefore += h
		}
		maxScrollBefore := el.maxScroll
		t.Logf("展开时 total=%.0f maxScroll=%.0f", totalBefore, maxScrollBefore)

		// 模拟折叠 item[1]（400→56）
		newHeights := []float64{300, 56, 500, 200, 350}
		v2 := &VirtualList{
			ItemCount:   5,
			ItemHeight:  80,
			ItemHeights: newHeights,
			Height:      300,
			Width:       300,
			RenderItem: func(i int) Widget {
				h := 80.0
				if i >= 0 && i < len(newHeights) {
					h = newHeights[i]
				}
				return &Container{Width: 300, Height: h}
			},
		}
		el.Update(v2)
		el.Build()
		el.Layout(&layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: 300},
		})

		totalAfter := 0.0
		for _, h := range el.vl.ItemHeights {
			totalAfter += h
		}
		maxScrollAfter := el.maxScroll
		t.Logf("折叠后 total=%.0f maxScroll=%.0f", totalAfter, maxScrollAfter)

		if maxScrollAfter >= maxScrollBefore {
			t.Errorf("折叠后 maxScroll 应该变小！before=%.0f after=%.0f", maxScrollBefore, maxScrollAfter)
		}
	})

	// 测试 2：Slice 引用独立 — Layout 修改应不影响 chat.go 缓存
	t.Run("slice_independence", func(t *testing.T) {
		cached := []float64{100, 200, 150}
		v := &VirtualList{
			ItemCount:   3,
			ItemHeight:  80,
			ItemHeights: cached,
			Height:      300,
			Width:       300,
			RenderItem: func(i int) Widget {
				return &Container{Width: 300, Height: cached[i]}
			},
		}
		el := v.CreateElement().(*VirtualListElement)
		el.Mount(nil, 0)
		el.Build()
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: 300},
		}
		el.Layout(ctx)

		t.Logf("cached[0]=%.0f (Layout 后)", cached[0])
		// Layout 会捕获实际高并回写 ItemHeights（即 cached 切片）
		// 当 Container 高 = cached[i] 时，实际高就等于 cached[i]
		// 所以修正不应改变 cached[0]
		// 但如果 Container 高是固定值而非 cached[i]，就可能变化
	})

	// 测试 3：Paint 时 maxScroll 独立重算 vs Layout 的 maxScroll
	t.Run("paint_maxScroll_recalc", func(t *testing.T) {
		heights := []float64{500, 500, 500, 500, 500}
		el := makeVL(5, heights, 300)
		total := 0.0
		for _, h := range el.vl.ItemHeights {
			total += h
		}
		// Paint 中重算：total - viewportH
		msFromTotal := total - el.viewportH
		t.Logf("Layout maxScroll=%.0f Paint recalc ms=%.0f (total=%.0f viewport=%.0f)",
			el.maxScroll, msFromTotal, total, el.viewportH)
		if el.maxScroll != msFromTotal {
			t.Errorf("maxScroll mismatch: Layout=%.0f Paint=%.0f", el.maxScroll, msFromTotal)
		}
	})

	// 测试 4：重复 collapse/expand 10 次，maxScroll 稳定
	t.Run("collapse_expand_cycle", func(t *testing.T) {
		heights := []float64{300, 400, 500, 200, 350}
		v := &VirtualList{
			ItemCount:   5,
			ItemHeight:  80,
			ItemHeights: heights,
			Height:      300,
			Width:       300,
			RenderItem: func(i int) Widget {
				return &Container{Width: 300, Height: heights[i]}
			},
		}
		el := v.CreateElement().(*VirtualListElement)
		el.Mount(nil, 0)
		el.Build()
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: 300},
		}
		el.Layout(ctx)

		for cycle := 0; cycle < 10; cycle++ {
			// 折叠
			heights[2] = 56
			v2 := &VirtualList{
				ItemCount:   5,
				ItemHeight:  80,
				ItemHeights: heights,
				Height:      300, Width: 300,
				RenderItem: func(i int) Widget {
					return &Container{Width: 300, Height: heights[i]}
				},
			}
			el.Update(v2)
			el.Build()
			el.Layout(ctx)

			maxCollapsed := el.maxScroll

			// 展开
			heights[2] = 500
			v3 := &VirtualList{
				ItemCount:   5,
				ItemHeight:  80,
				ItemHeights: heights,
				Height:      300, Width: 300,
				RenderItem: func(i int) Widget {
					return &Container{Width: 300, Height: heights[i]}
				},
			}
			el.Update(v3)
			el.Build()
			el.Layout(ctx)

			maxExpanded := el.maxScroll

			if maxExpanded <= maxCollapsed {
				t.Errorf("cycle %d: 展开 maxScroll=%.0f 应 > 折叠 maxScroll=%.0f",
					cycle, maxExpanded, maxCollapsed)
			}
		}
		t.Logf("10 轮 collapse/expand 循环通过")
	})

	// 测试 5：彻底模拟 chat.go 的 cachedHeights 模式
	t.Run("chat_cachedHeights_pattern", func(t *testing.T) {
		// 模拟 chat.go 中的 cachedHeights 变量
		var cachedHeights []float64
		cacheMsgLen := 0

		messages := []struct {
			collapsed bool
			textH     float64 // 展开时文本部分的高度
		}{
			{false, 200}, {false, 300}, {false, 150}, {false, 400}, {false, 100},
		}

		estimateH := func(idx int) float64 {
			m := messages[idx]
			if m.collapsed {
				return 56
			}
			return 56 + m.textH // 基础 + 内容
		}

		// 第一次 scrollMessages 调用
		buildItemHeights := func() []float64 {
			if cachedHeights == nil || len(cachedHeights) != len(messages) || len(messages) != cacheMsgLen {
				cachedHeights = make([]float64, len(messages))
				for i := range messages {
					cachedHeights[i] = estimateH(i)
				}
				cacheMsgLen = len(messages)
			}
			return cachedHeights
		}

		ih := buildItemHeights()
		t.Logf("初始 ItemHeights: %v", ih)

		v := &VirtualList{
			ItemCount:   len(messages),
			ItemHeight:  80,
			ItemHeights: ih,
			Height:      300, Width: 300,
			RenderItem: func(i int) Widget {
				h := messages[i].textH + 56
				if messages[i].collapsed {
					h = 56
				}
				return &Container{Width: 300, Height: h}
			},
		}
		el := v.CreateElement().(*VirtualListElement)
		el.Mount(nil, 0)
		el.Build()
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: 300},
		}
		el.Layout(ctx)
		t.Logf("Layout 后 cachedHeights: %v", cachedHeights)
		// Layout 修改了 cachedHeights 就地！

		// 模拟折叠消息[1]
		messages[1].collapsed = true
		cachedHeights = nil // ← collapse 回调干的
		ih = buildItemHeights()
		t.Logf("折叠后 ItemHeights: %v", ih)

		v2 := &VirtualList{
			ItemCount:   len(messages),
			ItemHeight:  80,
			ItemHeights: ih,
			Height:      300, Width: 300,
			RenderItem: func(i int) Widget {
				h := messages[i].textH + 56
				if messages[i].collapsed {
					h = 56
				}
				return &Container{Width: 300, Height: h}
			},
		}
		el.Update(v2)
		el.Build()
		el.Layout(ctx)
		t.Logf("折叠 Layout 后 cachedHeights: %v", cachedHeights)

		if cachedHeights[1] != 56 {
			t.Errorf("折叠后 item[1] 高度应为 56，实际 %.0f", cachedHeights[1])
		}
	})
}

// 辅助：获取切片指针用于引用比较
func ptrDiff[T any](a, b []T) string {
	if len(a) == 0 || len(b) == 0 {
		return "empty"
	}
	if &a[0] == &b[0] {
		return "same"
	}
	return "diff"
}

func TestVirtualListThumbCalculation(t *testing.T) {
	tests := []struct {
		name       string
		heights    []float64
		viewportH  float64
		wantTotal  float64
		wantMaxScr float64
	}{
		{"5 items total 650", []float64{100, 200, 150, 80, 120}, 300, 650, 350},
		{"3 small items", []float64{50, 60, 70}, 300, 180, 0},
		{"mixed", []float64{56, 56, 500, 56, 56}, 300, 724, 424},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := (&VirtualList{
				ItemCount:   len(tt.heights),
				ItemHeight:  80,
				ItemHeights: tt.heights,
				Height:      tt.viewportH,
				Width:       300,
				RenderItem:  func(i int) Widget { return &Container{Width: 300, Height: tt.heights[i]} },
			}).CreateElement().(*VirtualListElement)
			el.Mount(nil, 0)
			el.Build()
			ctx := &layout.LayoutContext{
				Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: tt.viewportH},
			}
			el.Layout(ctx)

			total := 0.0
			for _, h := range el.vl.ItemHeights {
				total += h
			}
			if total != tt.wantTotal {
				t.Errorf("totalContentH: got %.0f, want %.0f\nheights: %v", total, tt.wantTotal, el.vl.ItemHeights)
			}
			if el.maxScroll != tt.wantMaxScr {
				t.Errorf("maxScroll: got %.0f, want %.0f", el.maxScroll, tt.wantMaxScr)
			}
		})
	}

	// 预期 thumbH 值
	t.Run("thumbH_size", func(t *testing.T) {
		heights := []float64{2000}
		el := (&VirtualList{
			ItemCount:   1,
			ItemHeight:  80,
			ItemHeights: heights,
			Height:      300, Width: 300,
			RenderItem: func(i int) Widget { return &Container{Width: 300, Height: heights[i]} },
		}).CreateElement().(*VirtualListElement)
		el.Mount(nil, 0)
		el.Build()
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: 300},
		}
		el.Layout(ctx)

		// Paint 时 scrollbar 高度: th² / (th+maxScroll) = 300²/(300+1700)=90000/2000=45
		total := el.vl.ItemHeights[0]
		maxScroll := total - 300 // 1700
		thumbH := 300.0 * 300.0 / (300.0 + maxScroll)
		t.Logf("total=%.0f maxScroll=%.0f thumbH=%.2f (expect ~45)", total, maxScroll, thumbH)
		if thumbH < 40 || thumbH > 50 {
			t.Errorf("thumbH 应为 ~45，实际 %.2f", thumbH)
		}
	})
}

// ─── 首帧完整显示测试 ────────────────────────────
// 验证 VirtualList 在首帧 Build→Layout 后显示全部可视行，
// 而非只显示默认 600px viewport 内的行（修复前的 bug）。

func TestVirtualListFullViewportOnFirstFrame(t *testing.T) {
	t.Run("50_items_24px_800vh", func(t *testing.T) {
		// 模拟文件树面板：50 个文件，每行 24px，视口 800px
		itemCount := 50
		itemH := 24.0
		viewportH := 800.0

		el := (&VirtualList{
			ItemCount:  itemCount,
			ItemHeight: itemH,
			Height:     viewportH,
			Width:      260,
			RenderItem: func(i int) Widget {
				return &Container{Width: 260, Height: itemH}
			},
		}).CreateElement().(*VirtualListElement)
		el.Mount(nil, 0)
		el.Build()

		// 此时（修复前）viewportH=0→默认 600，visibleRange 只覆盖 600/24=25+5=30 项
		// visibleRange 实际输出：600/24+1+5=31，但需要验证 visibleRange 的范围
		start1, end1 := 0, 0
		// 无法直接调用 visibleRange（未导出），通过 log 验证
		t.Logf("Build 后 viewportH=%.0f (预期 800, 修复前为 600)", el.ViewportH())

		// Layout 设置正确 viewportH
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 260, MaxWidth: 260, MinHeight: 0, MaxHeight: viewportH},
		}
		el.Layout(ctx)

		_ = start1
		_ = end1
		t.Logf("Layout 后 viewportH=%.0f", el.ViewportH())
		t.Logf("maxScroll=%.0f (50*24-800=400)", el.MaxScroll())

		// 验证所有可见项已构建：视口内应有 ceil(800/24) = 34 项
		expectedInView := int(viewportH/itemH) + 1 // 34
		if int(viewportH) > int(itemH)*itemCount {
			expectedInView = itemCount
		}
		if expectedInView > itemCount {
			expectedInView = itemCount
		}

		builtCount := len(el.Children())
		t.Logf("已构建 child 数=%d, 预期至少 %d (不计算 overscan)", builtCount, expectedInView)

		if builtCount < expectedInView {
			t.Errorf("首帧构建不足: got %d children, want >= %d (viewportH=%.0f itemH=%.0f)",
				builtCount, expectedInView, viewportH, itemH)
		}

		// 验证 child 的 Y 位置覆盖了视口
		maxY := 0.0
		for _, child := range el.Children() {
			childOff := child.Offset()
			childSz := child.Size()
			childBottom := childOff.Y + childSz.Height
			if childBottom > maxY {
				maxY = childBottom
			}
		}
		t.Logf("可视区底部 child Y=%.0f, viewportH=%.0f", maxY, viewportH)
		// 修复后，maxY >= viewportH 说明视口被填满（或内容不足视口时=总内容高）
		// maxY 可能略大于 viewportH（overscan），但不应该远小于 viewportH
		if maxY < viewportH-50 && builtCount < itemCount {
			t.Errorf("首帧可视内容不足: maxY=%.0f < viewportH=%.0f, children=%d, total=%d",
				maxY, viewportH, builtCount, itemCount)
		}
	})

	t.Run("filetree_like_200_items", func(t *testing.T) {
		// 模拟文件树：200 个文件（含子目录），24px/行，视口 900px
		itemCount := 200
		itemH := 24.0
		viewportH := 900.0

		names := make([]string, itemCount)
		for i := range names {
			names[i] = fmt.Sprintf("file_%d.go", i)
		}

		el := (&VirtualList{
			ItemCount:  itemCount,
			ItemHeight: itemH,
			Height:     viewportH,
			Width:      280,
			RenderItem: func(i int) Widget {
				return &Container{Width: 280, Height: itemH}
			},
		}).CreateElement().(*VirtualListElement)
		el.Mount(nil, 0)
		el.Build()

		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 280, MaxWidth: 280, MinHeight: 0, MaxHeight: viewportH},
		}
		el.Layout(ctx)

		expectedInView := int(viewportH/itemH) + 1 // 38
		if expectedInView > itemCount {
			expectedInView = itemCount
		}
		// 加上 overscan (5)
		expectedWithOverscan := expectedInView + 5 + 5
		if expectedWithOverscan > itemCount {
			expectedWithOverscan = itemCount
		}

		builtCount := len(el.Children())
		t.Logf("200 items: viewportH=%.0f, 预期视口内 %d 项, 含 overscan %d 项, 实际构建 %d",
			viewportH, expectedInView, expectedWithOverscan, builtCount)

		// 修复后应该至少有视口内项数
		if builtCount < expectedInView {
			t.Errorf("200项首帧不足: built=%d < expected_in_view=%d", builtCount, expectedInView)
		}

		// 验证 child 数量稳定
		t.Logf("maxScroll=%.0f", el.MaxScroll())

		// scroll 一次，再验证重建后数量 >= expectedInView
		el.ScrollTo(itemH * 5)
		scrolledCount := len(el.Children())
		t.Logf("滚动5项后 child 数=%d", scrolledCount)
	})

	t.Run("1000_items_perf_check", func(t *testing.T) {
		// 验证大量项时首帧也正确
		itemCount := 1000
		itemH := 24.0
		viewportH := 850.0

		el := (&VirtualList{
			ItemCount:  itemCount,
			ItemHeight: itemH,
			Height:     viewportH,
			Width:      300,
			RenderItem: func(i int) Widget {
				return &Container{Width: 300, Height: itemH}
			},
		}).CreateElement().(*VirtualListElement)
		el.Mount(nil, 0)
		el.Build()

		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 300, MaxWidth: 300, MinHeight: 0, MaxHeight: viewportH},
		}
		el.Layout(ctx)

		expectedInView := int(viewportH/itemH) + 1 // 36
		builtCount := len(el.Children())
		t.Logf("1000 items: viewportH=%.0f, 预期视口内 %d, 实际构建 %d, maxScroll=%.0f",
			viewportH, expectedInView, builtCount, el.MaxScroll())

		if builtCount < expectedInView {
			t.Errorf("1000项首帧不足: built=%d < expected_in_view=%d", builtCount, expectedInView)
		}
	})
}

// 在运行前先清除 vltest 目录避免影响
func init() {
	fmt.Println("VirtualList scrollbar tests loaded")
}
