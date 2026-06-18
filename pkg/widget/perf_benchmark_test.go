package widget

import (
	"fmt"
	"testing"
	"time"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// ──────────────────────────────────────────────────────────────
// 性能基准测试：VirtualList / 增量 Build / SetState
// ──────────────────────────────────────────────────────────────

// Container 使用 SingleChildWidget 嵌入，通过该方式传入子控件。
func containerWithChild(w Widget) *Container {
	return &Container{SingleChildWidget: SingleChildWidget{Child: w}}
}

func benchVL(n int, itemH float64) *VirtualList {
	return &VirtualList{
		ItemCount:  n,
		ItemHeight: itemH,
		Height:     600,
		Width:      400,
		Overscan:   5,
		RenderItem: func(i int) Widget {
			return containerWithChild(
				NewText(fmt.Sprintf("Item %d / %d", i, n), types.ColorBlack))
		},
	}
}

func benchVLWithRenderFunc(n int, itemH float64, render func(i int) Widget) *VirtualList {
	return &VirtualList{
		ItemCount:  n,
		ItemHeight: itemH,
		Height:     600,
		Width:      400,
		Overscan:   5,
		RenderItem: render,
	}
}

// benchBuildTree 递归构建 Element 树（模拟 Pipeline.buildTree），与 buildTreeRec 不同名避免冲突。
func benchBuildTree(el Element) {
	children := el.Build()
	for _, child := range children {
		benchBuildTree(child)
	}
}

// ──────────────────────────────────────────────────────────────
// Benchmark 1: buildTree 扩展性（不同数量 VirtualList 项）
// ──────────────────────────────────────────────────────────────

func BenchmarkVirtualList_BuildTree(b *testing.B) {
	sizes := []int{100, 1000, 5000, 10000, 50000}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			vl := benchVL(n, 32)
			el := vl.CreateElement()
			el.Mount(nil, 0)
			benchBuildTree(el) // 首帧

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchBuildTree(el)
			}
			b.StopTimer()
		})
	}
}

// ──────────────────────────────────────────────────────────────
// Benchmark 2: Layout 扩展性（不同数量的 VirtualList 项）
// ──────────────────────────────────────────────────────────────

func BenchmarkVirtualList_Layout(b *testing.B) {
	sizes := []int{100, 1000, 5000, 10000, 50000}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			vl := benchVL(n, 32)
			el := vl.CreateElement()
			el.Mount(nil, 0)
			benchBuildTree(el)

			ctx := &layout.LayoutContext{
				Constraints: layout.BoxConstraints{MinWidth: 0, MaxWidth: 400, MinHeight: 0, MaxHeight: 600},
			}
			el.Layout(ctx)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				el.Layout(ctx)
			}
			b.StopTimer()
		})
	}
}

// ──────────────────────────────────────────────────────────────
// Benchmark 3: 纯 Layout（跳过 Build）
// ──────────────────────────────────────────────────────────────

func BenchmarkVirtualList_LayoutOnly(b *testing.B) {
	n := 10000
	vl := benchVL(n, 32)
	el := vl.CreateElement()
	el.Mount(nil, 0)
	benchBuildTree(el)
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{MinWidth: 0, MaxWidth: 400, MinHeight: 0, MaxHeight: 600},
	}
	el.Layout(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		el.Layout(ctx)
	}
	b.StopTimer()
}

// ──────────────────────────────────────────────────────────────
// Benchmark 4: 增量 BuildTree vs 全量（稳定子树跳过）
// ──────────────────────────────────────────────────────────────

func BenchmarkVirtualList_IncrementalBuild(b *testing.B) {
	n := 5000
	b.Run("FullRebuild", func(b *testing.B) {
		vl := benchVL(n, 32)
		el := vl.CreateElement()
		el.Mount(nil, 0)
		benchBuildTree(el)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			el2 := vl.CreateElement()
			el2.Mount(nil, 0)
			benchBuildTree(el2)
		}
	})

	b.Run("IncrementalUpdate", func(b *testing.B) {
		vl := benchVL(n, 32)
		el := vl.CreateElement()
		el.Mount(nil, 0)
		benchBuildTree(el)

		vl2 := benchVL(n, 32)
		vl2.RenderItem = func(i int) Widget {
			if i == n/2 {
				return containerWithChild(
					NewText(fmt.Sprintf("UPDATED Item %d", i), types.ColorBlack))
			}
			return containerWithChild(
				NewText(fmt.Sprintf("Item %d / %d", i, n), types.ColorBlack))
		}

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			el.Update(vl2)
			benchBuildTree(el)
		}
		b.StopTimer()
	})
}

// ──────────────────────────────────────────────────────────────
// Benchmark 5: 流式追加（SetState 模拟 Agent）
// ──────────────────────────────────────────────────────────────

type streamBenchState struct {
	BaseState
	items []Widget
}

func (s *streamBenchState) Build(ctx BuildContext) Widget {
	return &Column{Flex: Flex{
		MultiChildWidget: MultiChildWidget{Children: s.items},
	}}
}

func BenchmarkSetState_Streaming(b *testing.B) {
	b.ReportAllocs()
	for iter := 0; iter < b.N; iter++ {
		state := &streamBenchState{}
		w := &struct{ StatefulWidget }{}
		el := &StatefulElement{
			BaseElement: BaseElement{widget: w},
			state:       state,
			buildDirty:  true,
		}
		// 手动设置内部字段
		if setter, ok := interface{}(state).(interface {
			SetWidget(Widget)
			SetElement(*StatefulElement)
			SetContext(BuildContext)
		}); ok {
			setter.SetWidget(w)
			setter.SetElement(el)
			setter.SetContext(BuildContext{Element: el})
		}
		el.Mount(nil, 0)
		benchBuildTree(el)

		for i := 0; i < 100; i++ {
			state.items = append(state.items,
				containerWithChild(
					NewText(fmt.Sprintf("Msg %d: content for streaming test", i), types.ColorBlack)))
			state.SetState()
			benchBuildTree(el)
			ctx := &layout.LayoutContext{
				Constraints: layout.BoxConstraints{MinWidth: 0, MaxWidth: 400, MinHeight: 0, MaxHeight: 800},
			}
			el.Layout(ctx)
		}
	}
}

// ──────────────────────────────────────────────────────────────
// Benchmark 6: AppendColumn 增量追加（与 Column 对比）
// ──────────────────────────────────────────────────────────────

type appendBenchState struct {
	BaseState
	items []Widget
}

func (s *appendBenchState) Build(ctx BuildContext) Widget {
	return &AppendColumn{Children: s.items}
}

func BenchmarkSetState_AppendColumn(b *testing.B) {
	b.ReportAllocs()
	for iter := 0; iter < b.N; iter++ {
		state := &appendBenchState{}
		w := &struct{ StatefulWidget }{}
		el := &StatefulElement{
			BaseElement: BaseElement{widget: w},
			state:       state,
			buildDirty:  true,
		}
		if setter, ok := interface{}(state).(interface {
			SetWidget(Widget)
			SetElement(*StatefulElement)
			SetContext(BuildContext)
		}); ok {
			setter.SetWidget(w)
			setter.SetElement(el)
			setter.SetContext(BuildContext{Element: el})
		}
		el.Mount(nil, 0)
		benchBuildTree(el)

		for i := 0; i < 100; i++ {
			state.items = append(state.items,
				containerWithChild(
					NewText(fmt.Sprintf("Msg %d: content for streaming test", i), types.ColorBlack)))
			state.SetState()
			benchBuildTree(el)
			ctx := &layout.LayoutContext{
				Constraints: layout.BoxConstraints{MinWidth: 0, MaxWidth: 400, MinHeight: 0, MaxHeight: 800},
			}
			el.Layout(ctx)
		}
	}
}

func BenchmarkSetState_ColumnFull(b *testing.B) {
	b.ReportAllocs()
	for iter := 0; iter < b.N; iter++ {
		state := &streamBenchState{}
		w := &struct{ StatefulWidget }{}
		el := &StatefulElement{
			BaseElement: BaseElement{widget: w},
			state:       state,
			buildDirty:  true,
		}
		if setter, ok := interface{}(state).(interface {
			SetWidget(Widget)
			SetElement(*StatefulElement)
			SetContext(BuildContext)
		}); ok {
			setter.SetWidget(w)
			setter.SetElement(el)
			setter.SetContext(BuildContext{Element: el})
		}
		el.Mount(nil, 0)
		benchBuildTree(el)

		for i := 0; i < 100; i++ {
			state.items = append(state.items,
				containerWithChild(
					NewText(fmt.Sprintf("Msg %d: content for streaming test", i), types.ColorBlack)))
			state.SetState()
			benchBuildTree(el)
			ctx := &layout.LayoutContext{
				Constraints: layout.BoxConstraints{MinWidth: 0, MaxWidth: 400, MinHeight: 0, MaxHeight: 800},
			}
			el.Layout(ctx)
		}
	}
}

// ──────────────────────────────────────────────────────────────
// Benchmark 7: 模拟 Agent 综合场景
// ──────────────────────────────────────────────────────────────

func BenchmarkAgentSimulation(b *testing.B) {
	itemH := 40.0
	totalItems := 100

	b.ReportAllocs()
	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		vl := benchVL(totalItems, itemH)
		el := vl.CreateElement()
		el.Mount(nil, 0)
		benchBuildTree(el)
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 0, MaxWidth: 400, MinHeight: 0, MaxHeight: 600},
		}
		el.Layout(ctx)

		for msg := 0; msg < 50; msg++ {
			vl2 := benchVL(totalItems, itemH)
			el.Update(vl2)
			benchBuildTree(el)
			el.Layout(ctx)
		}

		vle := el.(*VirtualListElement)
		if vle.maxScroll > 0 {
			vle.ScrollTo(vle.maxScroll)
			el.Layout(ctx)
		}
		if vle.maxScroll > 0 {
			vle.ScrollTo(vle.maxScroll * 0.3)
			el.Layout(ctx)
		}
		if vle.maxScroll > 0 {
			vle.ScrollTo(vle.maxScroll)
			el.Layout(ctx)
		}
	}
	b.StopTimer()
}

// ──────────────────────────────────────────────────────────────
// Benchmark 7: 可变高度 VirtualList
// ──────────────────────────────────────────────────────────────

func BenchmarkVirtualList_VariableHeight(b *testing.B) {
	n := 5000
	heights := make([]float64, n)
	for i := range heights {
		heights[i] = 30 + float64(i%20)*25
	}

	vl := benchVLWithRenderFunc(n, 32, func(i int) Widget {
		h := heights[i]
		return &Container{
			Width:  400,
			Height: h,
			SingleChildWidget: SingleChildWidget{
				Child: NewText(fmt.Sprintf("VarH Item %d h=%.0f", i, h), types.ColorBlack),
			},
		}
	})
	el := vl.CreateElement()
	el.Mount(nil, 0)
	benchBuildTree(el)
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{MinWidth: 0, MaxWidth: 400, MinHeight: 0, MaxHeight: 600},
	}
	el.Layout(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		el.Layout(ctx)
	}
	b.StopTimer()
}

// ──────────────────────────────────────────────────────────────
// 验证测试：稳定子树缓存
// ──────────────────────────────────────────────────────────────

func TestStableSubtreeCache(t *testing.T) {
	el := &StatelessElement{
		BaseElement: BaseElement{widget: &Container{Width: 100, Height: 100}},
	}
	el.Mount(nil, 0)

	el.Build()
	if el.Stable() {
		t.Error("first Build should set stable=false")
	}

	el.Build()
	if !el.Stable() {
		t.Error("second Build should set stable=true (cache hit)")
	}

	el.Update(&Container{Width: 200, Height: 200})
	if el.Stable() {
		t.Error("after Update, Stable() should be false")
	}

	el.Build()
	if el.Stable() {
		t.Error("first Build after Update should set stable=false")
	}

	el.Build()
	if !el.Stable() {
		t.Error("second Build after Update should set stable=true")
	}
}

func TestStatefulElementStable(t *testing.T) {
	el := &StatefulElement{
		BaseElement: BaseElement{widget: &struct{ StatefulWidget }{}},
		buildDirty:  false,
	}
	if !el.Stable() {
		t.Error("buildDirty=false should be Stable()=true")
	}
	el.buildDirty = true
	if el.Stable() {
		t.Error("buildDirty=true should be Stable()=false")
	}
}

// ──────────────────────────────────────────────────────────────
// 详细性能分析（go test -v 使用）
// ──────────────────────────────────────────────────────────────

func testBuildTime(el Element) time.Duration {
	t0 := time.Now()
	benchBuildTree(el)
	return time.Since(t0)
}

func testLayoutTime(el Element, w, h float64) time.Duration {
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{MinWidth: 0, MaxWidth: w, MinHeight: 0, MaxHeight: h},
	}
	t0 := time.Now()
	el.Layout(ctx)
	return time.Since(t0)
}

func TestDetailedPerfAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in short mode")
	}

	sizes := []int{100, 1000, 5000, 10000}
	t.Logf("\n=== VirtualList 性能详细分析 ===\n")
	t.Logf("%-12s %-12s %-12s %-12s", "Items", "Build(μs)", "Layout(μs)", "Total(μs)")
	t.Log("--------------------------------------------------------------")

	for _, n := range sizes {
		vl := benchVL(n, 32)
		el := vl.CreateElement()
		el.Mount(nil, 0)

		dBuild := testBuildTime(el)
		dLayout := testLayoutTime(el, 400, 600)
		total := dBuild + dLayout

		t.Logf("%-12d %-12.1f %-12.1f %-12.1f",
			n,
			float64(dBuild)/float64(time.Microsecond),
			float64(dLayout)/float64(time.Microsecond),
			float64(total)/float64(time.Microsecond))
	}

	t.Logf("\n=== 增量 Build vs 全量 Build（5000 项）===\n")
	n := 5000
	vl := benchVL(n, 32)
	el := vl.CreateElement()
	el.Mount(nil, 0)
	testBuildTime(el)

	dFull := testBuildTime(el)

	vl2 := benchVL(n, 32)
	el.Update(vl2)
	dIncremental := testBuildTime(el)

	t.Logf("全量 Build:    %.1f μs", float64(dFull)/float64(time.Microsecond))
	t.Logf("增量 Build:    %.1f μs (加速 %.1fx)",
		float64(dIncremental)/float64(time.Microsecond),
		float64(dFull)/float64(dIncremental))
}

// ──────────────────────────────────────────────────────────────
// Markdown 首帧 + 双缓冲基准测试
// ──────────────────────────────────────────────────────────────

// genMD 生成长度为 n 字符的 markdown 文本。
func genMD(n int) string {
	text := "# Performance Test\n\n"
	for len(text) < n {
		text += "## Section\n\nThis is **bold** and `code` text with [link](http://example.com).\n\n"
		text += "- list item 1\n- list item 2\n- list item 3\n\n"
		text += "```go\nfunc main() { fmt.Println(\"hello\") }\n```\n\n"
	}
	return text[:n]
}

// BenchmarkMarkdownView_Build 测量 MarkdownView.Build 的首次/二次/流式增量性能。
func BenchmarkMarkdownView_Build(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, n := range sizes {
		md := genMD(n)
		style := DefaultMarkdownStyle()

		// 首次 Build（全量解析）
		b.Run(fmt.Sprintf("N=%d/First", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				mv := NewMarkdownView(md, style)
				_ = mv.Build(BuildContext{})
			}
		})

		// 二次 Build（命中缓存）
		b.Run(fmt.Sprintf("N=%d/Cached", n), func(b *testing.B) {
			mv := NewMarkdownView(md, style)
			mv.Build(BuildContext{}) // 首次，填充缓存
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = mv.Build(BuildContext{})
			}
		})

		// 流式追加增量（从空逐步追加到 n）
		b.Run(fmt.Sprintf("N=%d/Incremental", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				mv := NewMarkdownView("", style)
				chunkSize := n / 10
				for pos := 0; pos < n; pos += chunkSize {
					end := pos + chunkSize
					if end > n {
						end = n
					}
					mv.Source = md[:end]
					_ = mv.Build(BuildContext{})
				}
			}
		})
	}
}

// BenchmarkDeferredMarkdown 测量双缓冲效果。
func BenchmarkDeferredMarkdown(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, n := range sizes {
		md := genMD(n)
		style := DefaultMarkdownStyle()

		b.Run(fmt.Sprintf("N=%d/DeferredBuild", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				d := DeferredMarkdown(md, style)
				el := CreateElementFor(d).(*StatefulElement)
				el.Mount(nil, 0)

				// 第一次 Build → 返回占位
				el.Build()

				// 第二次 Build → 返回真实内容（缓存命中）
				el.Build()
			}
		})
	}
}

// TestMarkdownCache 验证 MarkdownView 缓存生效。
func TestMarkdownCache(t *testing.T) {
	md := genMD(5000)
	mv := NewMarkdownView(md, DefaultMarkdownStyle())

	// 首次 Build：返回的 Widget 不为 nil
	w1 := mv.Build(BuildContext{})
	if w1 == nil {
		t.Fatal("first Build returned nil")
	}
	// 验证缓存已填充
	if mv.cachedSource != md {
		t.Fatal("cache not populated after first Build")
	}
	if len(mv.cachedWidgets) == 0 {
		t.Fatal("no cached widgets after first Build")
	}

	// 二次 Build：返回相同内容
	w2 := mv.Build(BuildContext{})
	if w2 == nil {
		t.Fatal("second Build returned nil")
	}

	// 验证增量解析
	appended := md + "\n\n## Appended Section\nNew content here."
	mv.Source = appended
	w3 := mv.Build(BuildContext{})
	if w3 == nil {
		t.Fatal("incremental Build returned nil")
	}
	// 缓存应更新
	if mv.cachedSource != appended {
		t.Fatal("cache not updated after incremental append")
	}
}

// TestDeferredMarkdownDoubleBuffer 验证双缓冲首帧占位+次帧真实内容。
func TestDeferredMarkdownDoubleBuffer(t *testing.T) {
	md := genMD(10000)
	d := DeferredMarkdown(md, DefaultMarkdownStyle())
	el := CreateElementFor(d).(*StatefulElement)
	el.Mount(nil, 0)

	// 首帧 Build：应返回占位（Div{Height:20}）
	children1 := el.Build()
	if len(children1) == 0 {
		t.Fatal("first Build returned no children (expected placeholder)")
	}
	// 验证 state 中已标记准备就绪
	st := el.state.(*_deferredMarkdownState)
	if !st.prepared {
		t.Fatal("state.prepared should be true after first Build")
	}
	if st.readyWidget == nil {
		t.Fatal("readyWidget should be populated after first Build")
	}

	// 触发 rebuild（SetState 已调用）
	el.buildDirty = true

	// 次帧 Build：应返回真实内容（缓存命中）
	children2 := el.Build()
	if len(children2) == 0 {
		t.Fatal("second Build returned no children (expected real content)")
	}
	// 验证第二次返回的是 readyWidget
	if len(el.Children()) == 0 {
		t.Fatal("Children() should be non-empty after second Build")
	}
}
