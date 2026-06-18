package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"strings"

	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

type HistoryData struct {
	Version int      `json:"version"`
	Seq     int      `json:"seq"`
	Threads []Thread `json:"threads"`
}
type Thread struct {
	ID       string    `json:"ID"`
	Title    string    `json:"Title"`
	Messages []Message `json:"Messages"`
}
type Message struct {
	Role             int        `json:"Role"`
	Text             string     `json:"Text"`
	Thinking         string     `json:"Thinking"`
	Tools            []ToolData `json:"Tools"`
	Collapsed        bool       `json:"Collapsed"`
	Streaming        bool       `json:"Streaming"`
	ThinkingExpanded bool       `json:"ThinkingExpanded"`
}
type ToolData struct {
	Name   string `json:"Name"`
	Input  string `json:"Input"`
	Result string `json:"Result"`
}

var (
	bgMsg    = types.ColorFromRGBA(20, 22, 26, 255)
	bgThink  = types.ColorFromRGBA(40, 44, 52, 255)
	bgTool   = types.ColorFromRGBA(30, 30, 30, 255)
	fg       = types.ColorFromRGBA(200, 200, 200, 255)
	fgDim    = types.ColorFromRGBA(150, 150, 150, 255)
)

func main() {
	var h HistoryData
	data, err := os.ReadFile("F:\\syproject\\gou-ide\\.Pair\\conversations\\history.json")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if err := json.Unmarshal(data, &h); err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return
	}

	var allMsgs []Message
	for _, t := range h.Threads {
		for _, m := range t.Messages {
			mm := m
			mm.Collapsed = false
			mm.ThinkingExpanded = true
			allMsgs = append(allMsgs, mm)
		}
	}
	fmt.Printf("Real messages loaded: %d\n", len(allMsgs))
	if len(allMsgs) == 0 {
		return
	}

	totalChars := 0
	for _, m := range allMsgs {
		totalChars += len(m.Text) + len(m.Thinking)
		for _, t := range m.Tools {
			totalChars += len(t.Result)
		}
	}
	fmt.Printf("Total chars: %d (%.0f KB)\n", totalChars, float64(totalChars)/1024)

	// ========== BENCHMARK ==========
	testScales := []int{10, 25, 50, 100, 200, 500, 1000}

	fmt.Println()
	fmt.Println(" Msgs | Build(us) | Layout(us) | Steady(us) | Update(us) | scroll(us) ")
	fmt.Println("------+-----------+------------+------------+------------+------------")

	for _, totalMsg := range testScales {
		var testMsgs []Message
		for i := 0; i < totalMsg; i++ {
			testMsgs = append(testMsgs, allMsgs[i%len(allMsgs)])
		}

		itemHeights := make([]float64, totalMsg)
		for i, m := range testMsgs {
			itemHeights[i] = messageHeight(m)
		}

		viewH, width := 800.0, 400.0

		vl := &widget.VirtualList{
			ItemCount:          totalMsg,
			ItemHeight:         80,
			ItemHeights:        itemHeights,
			Height:             viewH,
			Width:              width,
			Overscan:           5,
			AutoScrollToBottom: true,
			RenderItem: func(i int) widget.Widget {
				return wrapCard(&testMsgs[i])
			},
		}

		el := vl.CreateElement()
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{
				MinWidth: width, MaxWidth: width,
				MinHeight: 0, MaxHeight: viewH,
			},
		}

		t0 := time.Now()
		for i := 0; i < 100; i++ {
			el.Build()
		}
		buildTime := time.Since(t0) / 100

		t0 = time.Now()
		for i := 0; i < 20; i++ {
			el.Build()
			el.Layout(ctx)
		}
		initLayout := time.Since(t0) / 20

		t0 = time.Now()
		for i := 0; i < 1000; i++ {
			el.Layout(ctx)
		}
		steadyTime := time.Since(t0) / 1000

		streamIter := 0
		vl2 := &widget.VirtualList{
			ItemCount:          totalMsg,
			ItemHeight:         80,
			ItemHeights:        itemHeights,
			Height:             viewH,
			Width:              width,
			Overscan:           5,
			AutoScrollToBottom: true,
			ScrollToBottomToken: 2,
			RenderItem: func(i int) widget.Widget {
				return wrapCard(&testMsgs[i])
			},
			ItemVersion: func(i int) int {
				if streamIter%5 == 0 && i == 0 {
					return 999
				}
				return 1
			},
		}
		t0 = time.Now()
		for i := 0; i < 100; i++ {
			streamIter++
			el.Update(vl2)
			el.Layout(ctx)
		}
		updateTime := time.Since(t0) / 100

		t0 = time.Now()
		for iter := 0; iter < 500; iter++ {
			filtered := make([]int, 0, totalMsg)
			for i := 0; i < totalMsg; i++ {
				filtered = append(filtered, i)
			}
			heights := make([]float64, len(filtered))
			for i, idx := range filtered {
				heights[i] = messageHeight(testMsgs[idx])
			}
			_ = heights
		}
		scrollTime := time.Since(t0) / 500

		fmt.Printf(" %4d | %9d | %10d | %10d | %10d | %10d \n",
			totalMsg,
			buildTime.Microseconds(),
			initLayout.Microseconds(),
			steadyTime.Microseconds(),
			updateTime.Microseconds(),
			scrollTime.Microseconds())
	}
}

func wrapCard(m *Message) widget.Widget {
	text := m.Text
	if text == "" {
		text = "(empty)"
	}
	var kids []widget.Widget

	if m.Thinking != "" {
		thought := m.Thinking
		if len(thought) > 800 {
			thought = thought[:800] + "..."
		}
		kids = append(kids,
			widget.Div(
				widget.Style{
					BackgroundColor: &bgThink,
					BorderRadius:    4,
					Padding:         types.EdgeInsets(8),
					FlexDirection:   "column",
				},
				widget.NewText("thinking: "+thought, fg),
			),
			widget.Div(widget.Style{Height: 4}),
		)
	}
	for _, tool := range m.Tools {
		result := tool.Result
		if len(result) > 500 {
			result = result[:500] + "..."
		}
		kids = append(kids,
			widget.Div(
				widget.Style{
					BackgroundColor: &bgTool,
					BorderRadius:    3,
					Padding:         types.EdgeInsets(6),
					FlexDirection:   "column",
				},
				widget.NewText(">> "+tool.Name, fgDim),
				widget.Div(widget.Style{Height: 4}),
			),
		)
	}
	kids = append(kids, widget.NewText(text, fg))

	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(10, 0, 10, 0), FlexDirection: "column"},
		widget.Div(
			widget.Style{
				BackgroundColor: &bgMsg,
				BorderRadius:    6,
				Padding:         types.EdgeInsets(12),
				FlexDirection:   "column",
				AlignItems:      "stretch",
			},
			kids...,
		),
		widget.Div(widget.Style{Height: 8}),
	)
}

func messageHeight(m Message) float64 {
	h := 56.0
	if strings.TrimSpace(m.Thinking) != "" {
		h += 60
	}
	for range m.Tools {
		h += 30
	}
	if txt := strings.TrimSpace(m.Text); txt != "" {
		lines := strings.Count(txt, "\n") + 1
		h += float64(lines)*19.0 + 4
		if h > 8000 {
			h = 8000
		}
	}
	return h
}
