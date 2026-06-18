// perftest — goui 性能压力测试
//
// 模拟 Agent 运行场景：
//   1. 持续输入事件流（鼠标滚轮 10,000 次滚动）
//   2. 持续追加数据到 50,000 条
//   3. 测量每帧的 Build / Layout / Paint / Flush 耗时
//
// 使用 SoftCanvas（纯 Go 渲染，无需 CGO/goskia）。
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
	"strconv"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// ─── 配置 ─────────────────────────────────────────
type PerfConfig struct {
	ItemCount     int     // 总消息数
	ItemHeight    float64 // 固定行高
	ViewportW     float64
	ViewportH     float64
	Overscan      int
	EnableVirtual bool // true = VirtualList, false = Column(全部渲染)
	EnableLog     bool
}

// ─── 统计 ─────────────────────────────────────────
type PerfStats struct {
	buildTimes   []float64 // μs
	layoutTimes  []float64 // μs
	paintTimes   []float64 // μs
	flushTimes   []float64 // μs
	totalTimes   []float64 // μs
	eventTimes   []float64 // μs
	buildCount   int
	layoutCount  int
	paintCount   int
}

func (s *PerfStats) record(build, layout, paint, flush, total, evt time.Duration) {
	if s.buildCount%100 == 0 { // 每 100 帧采样一次
		s.buildTimes = append(s.buildTimes, float64(build)/float64(time.Microsecond))
		s.layoutTimes = append(s.layoutTimes, float64(layout)/float64(time.Microsecond))
		s.paintTimes = append(s.paintTimes, float64(paint)/float64(time.Microsecond))
		s.flushTimes = append(s.flushTimes, float64(flush)/float64(time.Microsecond))
		s.totalTimes = append(s.totalTimes, float64(total)/float64(time.Microsecond))
		s.eventTimes = append(s.eventTimes, float64(evt)/float64(time.Microsecond))
	}
	s.buildCount++
}

func (s *PerfStats) report(title string) {
	fmt.Printf("\n═══ %s ═══\n", title)
	if len(s.totalTimes) == 0 {
		fmt.Println("  (no data)")
		return
	}

	avg := func(v []float64) float64 {
		var sum float64
		for _, x := range v {
			sum += x
		}
		return sum / float64(len(v))
	}
	p99 := func(v []float64) float64 {
		if len(v) < 2 {
			return v[0]
		}
		// 简单排序取 p99
		cp := make([]float64, len(v))
		copy(cp, v)
		for i := 0; i < len(cp); i++ {
			for j := i + 1; j < len(cp); j++ {
				if cp[j] < cp[i] {
					cp[i], cp[j] = cp[j], cp[i]
				}
			}
		}
		idx := int(float64(len(cp)) * 0.99)
		if idx >= len(cp) {
			idx = len(cp) - 1
		}
		return cp[idx]
	}

	fmt.Printf("  样本数: %d\n", len(s.totalTimes))
	fmt.Printf("  总耗时  | avg=%.1fμs  p99=%.1fμs\n", avg(s.totalTimes), p99(s.totalTimes))
	fmt.Printf("  Event   | avg=%.1fμs  p99=%.1fμs\n", avg(s.eventTimes), p99(s.eventTimes))
	fmt.Printf("  Build   | avg=%.1fμs  p99=%.1fμs\n", avg(s.buildTimes), p99(s.buildTimes))
	fmt.Printf("  Layout  | avg=%.1fμs  p99=%.1fμs\n", avg(s.layoutTimes), p99(s.layoutTimes))
	fmt.Printf("  Paint   | avg=%.1fμs  p99=%.1fμs\n", avg(s.paintTimes), p99(s.paintTimes))
	fmt.Printf("  Flush   | avg=%.1fμs  p99=%.1fμs\n", avg(s.flushTimes), p99(s.flushTimes))

	fps := 1e6 / avg(s.totalTimes)
	fmt.Printf("  等效 FPS: %.1f\n", fps)
}

// ─── 构建 VirtualList Widget ──────────────────────

func buildVirtualList(cfg PerfConfig) *widget.VirtualList {
	return &widget.VirtualList{
		ItemCount:  cfg.ItemCount,
		ItemHeight: cfg.ItemHeight,
		Height:     cfg.ViewportH,
		Width:      cfg.ViewportW,
		Overscan:   cfg.Overscan,
		RenderItem: func(i int) widget.Widget {
			if i%10 == 0 {
				// 模拟"大消息"—多行文本
				return widget.NewContainer(
					widget.NewText(fmt.Sprintf("[%d] ═══ 重要消息 %d ═══ 包含多行内容用于测试渲染性能 "+
						"这是额外填充文本确保每行有足够长度以测试换行逻辑", i, i),
						types.ColorFromRGBA(209, 213, 219, 255)),
				).WithWidth(cfg.ViewportW).WithPadding(types.EdgeInsetsLTRB(12, 8, 12, 8)).
					WithBackground(types.ColorRef(31, 41, 55)).
					WithBorderRadius(6)
			}
			return widget.NewContainer(
				widget.NewText(fmt.Sprintf("[%d] 测试消息 %d — 普通单行内容", i, i),
					types.ColorFromRGBA(156, 163, 175, 255)),
			).WithWidth(cfg.ViewportW).WithPadding(types.EdgeInsetsLTRB(12, 4, 12, 4))
		},
	}
}

// 构建全量 Column（不虚拟化，用于对比）
func buildFullColumn(cfg PerfConfig) widget.Widget {
	children := make([]widget.Widget, cfg.ItemCount)
	for i := range children {
		children[i] = widget.NewContainer(
			widget.NewText(fmt.Sprintf("[%d] 测试消息", i),
				types.ColorFromRGBA(156, 163, 175, 255)),
		).WithWidth(cfg.ViewportW)
	}
	return widget.NewColumn(children...)
}

// ─── 主测试函数 ───────────────────────────────────

func runTest(cfg PerfConfig, title string) *PerfStats {
	fmt.Printf("\n━━━ %s ━━━\n", title)
	fmt.Printf("  项目数: %d, 视口: %.0fx%.0f, 虚拟化: %v\n",
		cfg.ItemCount, cfg.ViewportW, cfg.ViewportH, cfg.EnableVirtual)

	// 创建 Canvas 和 Pipeline
	cvs := canvas.NewSoftCanvas(int(cfg.ViewportW), int(cfg.ViewportH))
	pipeline := render.NewPipeline(int(cfg.ViewportW), int(cfg.ViewportH), cvs)

	// 构建 Widget 树
	var rootWidget widget.Widget
	if cfg.EnableVirtual {
		rootWidget = buildVirtualList(cfg)
	} else {
		rootWidget = buildFullColumn(cfg)
	}

	// 创建 Element
	rootEl := widget.CreateElementFor(widget.NewOverlayHost(rootWidget))
	rootEl.Mount(nil, 0)
	pipeline.SetRootElement(rootEl)

	// 全局回调设置
	needsLayout := false
	needsRepaint := false
	widget.OnNeedsRepaint = func() { needsRepaint = true }
	widget.OnNeedsLayout = func() { needsLayout = true; needsRepaint = true }

	// 首次完整渲染
	pipeline.Render()

	stats := &PerfStats{}

	// ─── Phase 1: 模拟 Agent 流式追加 ──────────
	fmt.Println("  Phase 1: 模拟 Agent 流式追加...")
	phase1Start := time.Now()

	vle, _ := rootEl.(*widget.OverlayHostElement)
	var child widget.Element
	if vle != nil {
		for _, c := range rootEl.Children() {
			child = c
			break
		}
	}

	for step := 0; step < 50; step++ {
		// 每次追加 ~200 条 = 总计 50*200 = 10000 条
		for batch := 0; batch < 200; batch++ {
			cfg.ItemCount++
			if cfg.EnableVirtual {
				nvl := buildVirtualList(cfg)
				if child != nil {
					child.Update(nvl)
				}
			}
		}

		// 记录时间
		totalStart := time.Now()

		// 标记需要布局
		pipeline.MarkNeedsLayout()

		// Render
		t0 := time.Now()
		needsLayout = false
		needsRepaint = false
		dEvent := time.Since(t0)

		// 执行 Render
		t1 := time.Now()
		pipeline.Render()
		dTotal := time.Since(t1)

		stats.record(0, 0, 0, 0, dTotal, dEvent)
	}

	phase1Dur := time.Since(phase1Start)
	fmt.Printf("  Phase 1 完成: %v (最终 %d 项)\n", phase1Dur, cfg.ItemCount)

	// ─── Phase 2: 模拟鼠标滚轮事件流 ──────────
	fmt.Println("  Phase 2: 模拟鼠标滚轮事件流 (10,000 次滚动)...")
	phase2Start := time.Now()

	scrollPos := 0.0
	for i := 0; i < 10000; i++ {
		// 模拟滚轮事件
		scrollPos += float64(i%3+1) * 20 // 随机滚动量
		if cfg.EnableVirtual && child != nil {
			if vle, ok := child.(*widget.VirtualListElement); ok {
				vle.ScrollTo(scrollPos)
			}
		}

		totalStart := time.Now()

		// 模拟 Pipeline.ConsumePendingUpdates
		pipeline.MarkNeedsLayout()
		dEvent := time.Since(totalStart)

		t1 := time.Now()
		pipeline.Render()
		dTotal := time.Since(t1)

		stats.record(0, 0, 0, 0, dTotal, dEvent)
	}

	phase2Dur := time.Since(phase2Start)
	fmt.Printf("  Phase 2 完成: %v (%.0f 次采样)\n", phase2Dur, float64(len(stats.totalTimes)))

	// ─── Phase 3: 精细时序分析 ────────────────
	fmt.Println("  Phase 3: 精确帧时序分析 (100 帧)...")
	phase3Start := time.Now()

	var accBuild, accLayout, accPaint, accFlush, accTotal time.Duration
	frameCount := 100

	for i := 0; i < frameCount; i++ {
		// 模拟更新 + 渲染
		needsLayout = true
		needsRepaint = true

		frameStart := time.Now()

		// 模拟 Pipeline.ConsumePendingUpdates
		pipeline.MarkNeedsLayout()

		// 模拟主循环的 processEvents → animation.Tick → ConsumePendingUpdates
		// 直接 Render（包含 PerformLayout）。
		// 为了更精细的测量，手动分开测量各阶段
		pipeline.MarkNeedsLayout()

		// 分析各阶段
		t0 := time.Now()
		pipeline.EnsureLayout()
		durLayout := time.Since(t0)

		t1 := time.Now()
		pipeline.Render()
		durTotal := time.Since(t1)

		// 近似计算：Paint 和 Flush 的时间
		durBuild := time.Duration(0)
		durPaint := durTotal - durLayout
		durFlush := time.Duration(0)
		if durPaint < 0 {
			durPaint = 0
		}

		accBuild += durBuild
		accLayout += durLayout
		accPaint += durPaint
		accFlush += durFlush
		accTotal += durTotal

		stats.record(durBuild, durLayout, durPaint, durFlush, durTotal, 0)
	}

	phase3Dur := time.Since(phase3Start)
	avgFrame := accTotal / time.Duration(frameCount)
	fmt.Printf("  Phase 3 完成: %v\n", phase3Dur)
	fmt.Printf("  ─── 平均帧耗时 ───\n")
	fmt.Printf("  Total:   %v (%.0f FPS)\n", avgFrame, float64(time.Second)/float64(avgFrame))
	fmt.Printf("  Layout:  %v (%.1f%%)\n", accLayout/time.Duration(frameCount),
		float64(accLayout)*100/float64(accTotal))
	fmt.Printf("  Paint:   %v (%.1f%%)\n", accPaint/time.Duration(frameCount),
		float64(accPaint)*100/float64(accTotal))
	fmt.Printf("  Flush:   %v (%.1f%%)\n", accFlush/time.Duration(frameCount),
		float64(accFlush)*100/float64(accTotal))

	return stats
}

// ─── 入口 ─────────────────────────────────────────

func main() {
	// 环境变量配置
	itemCount := 50000
	viewportW := 800.0
	viewportH := 600.0

	if v := os.Getenv("PERF_ITEMS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			itemCount = n
		}
	}
	if v := os.Getenv("PERF_WIDTH"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			viewportW = n
		}
	}
	if v := os.Getenv("PERF_HEIGHT"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			viewportH = n
		}
	}

	commonCfg := PerfConfig{
		ItemHeight:    40,
		ViewportW:     viewportW,
		ViewportH:     viewportH,
		Overscan:      5,
	}

	fmt.Println(strings.Repeat("═", 60))
	fmt.Println("  goui 性能压力测试")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("  最终项目数: %d\n", itemCount)

	// 测试 1: VirtualList + 大量数据
	cfg1 := commonCfg
	cfg1.ItemCount = itemCount
	cfg1.EnableVirtual = true
	stats1 := runTest(cfg1, "Test 1: VirtualList 虚拟滚动")
	stats1.report("VirtualList 统计")

	// 测试 2: Column 全量渲染（对比组，少量数据）
	cfg2 := commonCfg
	cfg2.ItemCount = 100 // 全量渲染只能少量，否则 OOM
	cfg2.EnableVirtual = false
	stats2 := runTest(cfg2, "Test 2: Column 全量渲染 (N=100, 对比组)")
	stats2.report("Column 全量渲染统计")

	fmt.Println(strings.Repeat("═", 60))
	fmt.Println("\n结论:")
	fmt.Printf("  VirtualList 在 %d 项时的帧性能已记录。\n", itemCount)
	fmt.Printf("  对比 Column 全量渲染(100 项)的帧性能已记录。\n")
	fmt.Println(strings.Repeat("═", 60))
}
