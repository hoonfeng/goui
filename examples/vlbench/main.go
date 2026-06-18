package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// Generate a realistic large markdown message matching the scale of history data (141K chars).
func generateLargeMarkdown(contentScale int) string {
	var b strings.Builder
	b.WriteString("# Large Performance Test Document\n\n")
	b.WriteString("## Overview\n\n")
	b.WriteString("This document tests rendering performance with large markdown content.\n\n")
	b.WriteString("| Metric | Value | Threshold | Status |\n|--------|-------|-----------|--------|\n")
	for i := 0; i < 50; i++ {
		b.WriteString(fmt.Sprintf("| latency_%d | %dms | 100ms | OK |\n", i, i*3))
	}
	b.WriteString("\n")

	// Code blocks (matching real content pattern)
	for i := 0; i < 20; i++ {
		b.WriteString(fmt.Sprintf("## Code Section %d\n\n", i))
		b.WriteString("```go\n")
		b.WriteString("func ProcessData(items []string) (result []string, err error) {\n")
		b.WriteString("    for _, item := range items {\n")
		b.WriteString("        if item == \"\" { continue }\n")
		b.WriteString("        processed := strings.TrimSpace(item)\n")
		b.WriteString("        if strings.HasPrefix(processed, \"#\") { continue }\n")
		b.WriteString("        result = append(result, processed)\n")
		b.WriteString("    }\n")
		b.WriteString("    return result, nil\n")
		b.WriteString("}\n")
		b.WriteString("```\n\n")

		// Long paragraph
		b.WriteString(fmt.Sprintf("This is paragraph %d with **bold**, *italic*, and `inline code` elements. ", i))
		b.WriteString(strings.Repeat("More text here to simulate real content. ", 30))
		b.WriteString("\n\n")
	}

	// Block quotes
	b.WriteString("## Important Notes\n\n")
	for i := 0; i < 30; i++ {
		b.WriteString("> **Note " + fmt.Sprint(i+1) + "**: This is an important observation that ")
		b.WriteString("spans multiple lines of content for realistic testing purposes.\n")
	}
	b.WriteString("\n")

	// Nested lists
	b.WriteString("## Detailed Findings\n\n")
	for i := 0; i < 40; i++ {
		b.WriteString(fmt.Sprintf("1. **Finding %d**: Key observation with supporting evidence\n", i+1))
		b.WriteString("   - Supporting detail A for finding " + fmt.Sprint(i+1) + "\n")
		b.WriteString("   - Supporting detail B with `code reference`\n")
		b.WriteString("   - Supporting detail C with [link](http://example.com)\n\n")
	}

	return b.String()
}

func main() {
	fmt.Println("=== REALISTIC VirtualList Performance Benchmark ===")
	fmt.Println("Using large markdown content matching real history data scale")
	fmt.Println()

	cvs := canvas.NewSkiaCanvas(1000, 800)

	mdStyle := widget.MarkdownStyle{
		TextColor:    types.ColorFromRGBA(209, 213, 219, 255),
		TextSubtle:   types.ColorFromRGBA(156, 163, 175, 255),
		TextMuted:    types.ColorFromRGBA(107, 114, 128, 255),
		AccentColor:  types.ColorFromRGBA(59, 130, 246, 255),
		SuccessColor: types.ColorFromRGBA(34, 197, 94, 255),
		BorderColor:  types.ColorFromRGBA(55, 65, 81, 255),
		BgColor:      types.ColorFromRGBA(17, 24, 39, 255),
		BgSubtle:     types.ColorFromRGBA(31, 41, 55, 255),
		BgMuted:      types.ColorFromRGBA(55, 65, 81, 255),
		BodyFontSize: 13,
		MonoFont:     canvas.Font{Family: "Consolas", Size: 13},
	}

	// Generate large content
	largeContent := generateLargeMarkdown(1)
	fmt.Printf("Large content: %d chars\n", len(largeContent))

	// Phase 1: SINGLE item - detailed breakdown
	fmt.Println("\n=== Phase 1: Single Item Detailed Breakdown ===")

	start := time.Now()
	w := widget.NewMarkdownView(largeContent, mdStyle)
	fmt.Printf("  Widget creation (Markdown parse): %v\n", time.Since(start))

	start = time.Now()
	el := widget.CreateElementFor(w)
	fmt.Printf("  Element creation: %v\n", time.Since(start))

	start = time.Now()
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{MinWidth: 800, MaxWidth: 800, MinHeight: 0, MaxHeight: 1 << 20},
	}
	el.Layout(ctx)
	firstLayoutTime := time.Since(start)
	fmt.Printf("  First Layout (cold caches): %v\n", firstLayoutTime)

	// Second layout (warm caches)
	start = time.Now()
	el.Layout(ctx)
	secondLayoutTime := time.Since(start)
	layoutSpeedup := ""
	if secondLayoutTime < firstLayoutTime/10 {
		layoutSpeedup = fmt.Sprintf(" [%.0fx speedup!]", float64(firstLayoutTime)/float64(secondLayoutTime))
	}
	fmt.Printf("  Second Layout (warm caches): %v%s\n", secondLayoutTime, layoutSpeedup)

	// Mount and paint
	mount := &widget.BaseElement{}
	el.Mount(mount, 0)

	start = time.Now()
	el.Paint(cvs, types.Point{})
	fmt.Printf("  First Paint: %v\n", time.Since(start))

	// Repeated paint
	start = time.Now()
	for i := 0; i < 50; i++ {
		cvs.Clear(types.ColorWhite)
		el.Paint(cvs, types.Point{})
	}
	avgPaint := time.Since(start) / 50
	fmt.Printf("  Avg Paint (50x): %v (~%.0f FPS)\n", avgPaint, float64(time.Second)/float64(avgPaint))

	// Flush time
	start = time.Now()
	for i := 0; i < 50; i++ {
		cvs.Flush()
	}
	avgFlush := time.Since(start) / 50
	fmt.Printf("  Avg Flush (50x): %v (~%.0f FPS)\n", avgFlush, float64(time.Second)/float64(avgFlush))

	// Phase 2: Multiple items (simulate 15 visible messages)
	fmt.Println("\n=== Phase 2: 15 Visible Items (Chat Viewport) ===")

	// Create varied content for each "message"
	var items []widget.Widget
	totalWidgetCreate := time.Duration(0)
	totalElementCreate := time.Duration(0)
	totalFirstLayout := time.Duration(0)
	var elements []widget.Element

	for i := 0; i < 15; i++ {
		// Each "message" has different content scaled by index
		scale := 1 + i/3 // 0x, 2x, 4x, 8x content
		text := generateLargeMarkdown(scale)

		start = time.Now()
		w := widget.NewMarkdownView(text, mdStyle)
		totalWidgetCreate += time.Since(start)
		items = append(items, w)

		start = time.Now()
		el := widget.CreateElementFor(w)
		totalElementCreate += time.Since(start)
		elements = append(elements, el)

		start = time.Now()
		el.Layout(&layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 800, MaxWidth: 800, MinHeight: 0, MaxHeight: 1 << 20},
		})
		totalFirstLayout += time.Since(start)
	}

	fmt.Printf("  15 items Widget creation (total): %v (avg %v)\n", totalWidgetCreate, totalWidgetCreate/15)
	fmt.Printf("  15 items Element creation (total): %v (avg %v)\n", totalElementCreate, totalElementCreate/15)
	fmt.Printf("  15 items First Layout (total): %v (avg %v)\n", totalFirstLayout, totalFirstLayout/15)

	// Warm layout - measure all 15
	start = time.Now()
	for _, el := range elements {
		el.Layout(&layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 800, MaxWidth: 800, MinHeight: 0, MaxHeight: 1 << 20},
		})
	}
	warmLayoutAll := time.Since(start)
	fmt.Printf("  15 items Warm Layout (total): %v (avg %v)\n", warmLayoutAll, warmLayoutAll/15)

	// Paint all 15 items
	for i, el := range elements {
		el.Mount(mount, i)
		el.Paint(cvs, types.Point{})
	}

	start = time.Now()
	for i := 0; i < 100; i++ {
		cvs.Clear(types.ColorWhite)
		for _, el := range elements {
			el.Paint(cvs, types.Point{})
		}
	}
	avgAllPaint := time.Since(start) / 100
	fmt.Printf("  15 items Avg Paint (100x): %v (~%.0f FPS)\n", avgAllPaint, float64(time.Second)/float64(avgAllPaint))

	// Phase 3: Streaming simulation
	fmt.Println("\n=== Phase 3: Streaming (1 changing + 14 cached) ===")
	// 14 cached items already warm in elements[0:14]
	// Create 1 streaming item
	streamText := generateLargeMarkdown(1)
	streamW := widget.NewMarkdownView(streamText, mdStyle)
	streamEl := widget.CreateElementFor(streamW)
	streamEl.Layout(&layout.LayoutContext{
		Constraints: layout.BoxConstraints{MinWidth: 800, MaxWidth: 800, MinHeight: 0, MaxHeight: 1 << 20},
	})
	streamEl.Mount(mount, 15)

	allEls := append(elements, streamEl)

	// Warm paint all
	for i := 0; i < 5; i++ {
		cvs.Clear(types.ColorWhite)
		for _, el := range allEls {
			el.Paint(cvs, types.Point{})
		}
	}

	// Phase 3a: Just paint (no updates)
	start = time.Now()
	for i := 0; i < 100; i++ {
		cvs.Clear(types.ColorWhite)
		for _, el := range allEls {
			el.Paint(cvs, types.Point{})
		}
	}
	justPaint := time.Since(start) / 100
	fmt.Printf("  Just Paint (no updates): %v (~%.0f FPS)\n", justPaint, float64(time.Second)/float64(justPaint))

	// Phase 3b: Paint + streaming update (full rebuild)
	streamTimes := time.Duration(0)
	for i := 0; i < 10; i++ {
		newText := generateLargeMarkdown(1 + i)
		newW := widget.NewMarkdownView(newText, mdStyle)

		t1 := time.Now()
		streamEl.Update(newW)
		for _, c := range streamEl.Build() {
			_ = c
		}
		streamEl.Layout(&layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 800, MaxWidth: 800, MinHeight: 0, MaxHeight: 1 << 20},
		})
		streamTimes += time.Since(t1)

		cvs.Clear(types.ColorWhite)
		for _, el := range allEls {
			el.Paint(cvs, types.Point{})
		}
	}
	avgStream := streamTimes / 10
	fmt.Printf("  Streaming update (10x): avg %v per update\n", avgStream)

	// Phase 3c: Paint after streaming
	start = time.Now()
	for i := 0; i < 100; i++ {
		cvs.Clear(types.ColorWhite)
		for _, el := range allEls {
			el.Paint(cvs, types.Point{})
		}
	}
	paintAfterStream := time.Since(start) / 100
	fmt.Printf("  Paint after streaming: %v (~%.0f FPS)\n", paintAfterStream, float64(time.Second)/float64(paintAfterStream))

	// Phase 4: TextElement incremental measurement test
	fmt.Println("\n=== Phase 4: Streaming Text Append (Incremental Measurement) ===")
	baseText := "# Streaming Test\n\nThis is the base text."
	currentW := widget.NewMarkdownView(baseText, mdStyle)
	currentEl := widget.CreateElementFor(currentW)
	currentEl.Layout(&layout.LayoutContext{
		Constraints: layout.BoxConstraints{MinWidth: 800, MaxWidth: 800, MinHeight: 0, MaxHeight: 1 << 20},
	})

	for i := 0; i < 10; i++ {
		// Append to text each iteration (simulating streaming)
		newText := baseText + fmt.Sprintf("\n\n## Append %d\n\nNew content here **bold** `code` and more text. ", i)
		newText += strings.Repeat("padding ", 100) // Add significant new text
		newW := widget.NewMarkdownView(newText, mdStyle)

		start = time.Now()
		currentEl.Update(newW)
		for _, c := range currentEl.Build() {
			_ = c
		}
		currentEl.Layout(&layout.LayoutContext{
			Constraints: layout.BoxConstraints{MinWidth: 800, MaxWidth: 800, MinHeight: 0, MaxHeight: 1 << 20},
		})
		fmt.Printf("  Update+Layout #%d (total %d chars): %v\n", i+1, len(newText), time.Since(start))
	}

	// Final summary
	fmt.Println("\n=== FINAL SUMMARY ===")
	fmt.Printf("15 items Warm Layout: %v (per item: %v)\n", warmLayoutAll, warmLayoutAll/15)
	fmt.Printf("15 items Paint (100x): %v/frame (~%.0f FPS)\n", avgAllPaint, float64(time.Second)/float64(avgAllPaint))
	fmt.Printf("Streaming update (14 cached + 1 change): %v per frame\n", avgStream)
	fmt.Printf("Just Paint (14 cached + 1 cached): %v/frame (~%.0f FPS)\n", justPaint, float64(time.Second)/float64(justPaint))
	fmt.Printf("Single heavy item First Layout: %v\n", firstLayoutTime)
	fmt.Printf("Single heavy item Warm Layout: %v (%.0fx speedup)\n", secondLayoutTime, float64(firstLayoutTime)/float64(secondLayoutTime))
}
