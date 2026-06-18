//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

var (
	Bg       = *types.ColorRef(30, 30, 30)
	BgSubtle = *types.ColorRef(37, 37, 38)
	BgMuted  = *types.ColorRef(45, 45, 46)
	Fg       = *types.ColorRef(204, 204, 204)
	FgMuted  = *types.ColorRef(140, 140, 140)
	Border   = *types.ColorRef(45, 45, 45)
	Accent   = *types.ColorRef(88, 166, 255)
	UserBg   = *types.ColorRef(40, 45, 55)
	UserBrd  = *types.ColorRef(55, 65, 80)
)

type historyFile struct {
	Version int       `json:"version"`
	Seq     int       `json:"seq"`
	Threads []*Thread `json:"threads"`
}

type Role int
const (
	User Role = iota
	Assistant
)

type TimelineEntry struct {
	Kind    string `json:"kind,omitempty"`
	Content string `json:"content,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Args    string `json:"args,omitempty"`
	CallID  string `json:"callID,omitempty"`
	Result  string `json:"result,omitempty"`
	Done    bool   `json:"done,omitempty"`
}

type Activity struct {
	CallID string `json:"callID,omitempty"`
	Tool   string `json:"tool,omitempty"`
	Args   string `json:"args,omitempty"`
	Result string `json:"result,omitempty"`
	Done   bool   `json:"done,omitempty"`
}

type Message struct {
	Role              Role             `json:"role"`
	Text              string           `json:"text"`
	Thinking          string           `json:"thinking,omitempty"`
	Activities        []Activity       `json:"activities,omitempty"`
	Streaming         bool             `json:"streaming,omitempty"`
	Collapsed         bool             `json:"collapsed,omitempty"`
	ThinkingExpanded  bool             `json:"thinkingExpanded,omitempty"`
	Timeline          []TimelineEntry  `json:"timeline,omitempty"`
}

type Thread struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Messages []Message `json:"messages"`
}

func main() {
	// 加载真实历史（从 gou-ide repo 根读取）
	path := filepath.Join("F:\\syproject\\gou-ide", ".pair", "conversations", "history.json")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("无法读取历史:", err)
		os.Exit(1)
	}
	var hf historyFile
	if err := json.Unmarshal(data, &hf); err != nil {
		fmt.Println("JSON 解析失败:", err)
		os.Exit(1)
	}
	fmt.Printf("已加载 %d 个会话\n", len(hf.Threads))

	var allMsgs []Message
	for _, t := range hf.Threads {
		allMsgs = append(allMsgs, t.Messages...)
	}
	fmt.Printf("总消息数: %d\n", len(allMsgs))
	
	// 只取前 20 条来渲染
	if len(allMsgs) > 20 {
		allMsgs = allMsgs[:20]
	}
	fmt.Printf("渲染前 %d 条消息\n", len(allMsgs))

	// 构建 VirtualList
	vlW, vlH := 380.0, 500.0
	vl := &widget.VirtualList{
		ItemCount:  len(allMsgs),
		ItemHeight: 80,
		Height:     vlH,
		Width:      vlW,
		Overscan:   5,
		RenderItem: func(i int) widget.Widget {
			m := allMsgs[i]
			return widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(10, 0, 10, 0), FlexDirection: "column"},
				renderMsg(m),
				widget.Div(widget.Style{Height: 8}),
			)
		},
	}
	// 使用已知的 Canvas 实例
	sk := canvas.NewSkiaCanvas(int(vlW), int(vlH))
	pipe := render.NewPipeline(int(vlW), int(vlH), sk)
	pipe.SetRootElement(widget.CreateElementFor(vl))

	if err := pipe.Render(); err != nil {
		fmt.Println("渲染错误:", err)
		os.Exit(1)
	}

	// 调试输出
	root := pipe.RootElement()
	dumpTree(root, 0)

	// 保存 PNG
	if err := sk.SaveToPNG("history_render_test.png"); err != nil {
		fmt.Println("保存 PNG 失败:", err)
		os.Exit(1)
	}
	fmt.Println("✅ history_render_test.png 已保存")
}

func renderMsg(m Message) widget.Widget {
	if m.Role == User {
		return widget.Div(
			widget.Style{BackgroundColor: &UserBg, BorderColor: &UserBrd, BorderWidth: 1, BorderRadius: 6,
				Padding: types.EdgeInsetsLTRB(12, 8, 12, 8), FlexDirection: "column"},
			widget.NewText(m.Text, Fg),
		)
	}
	kids := []widget.Widget{}
	if m.Thinking != "" && !m.Collapsed {
		kids = append(kids,
			widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(0, 0, 0, 6), FlexDirection: "row", AlignItems: "center"},
				widget.Lucide("brain", widget.IconSize(12), widget.IconColor(FgMuted)),
				widget.Div(widget.Style{Width: 5}), widget.NewText("思考", FgMuted)),
		)
	}
	if !m.Collapsed {
		if m.Text != "" {
			kids = append(kids, widget.NewText(m.Text, Fg))
		}
		for _, a := range m.Activities {
			kids = append(kids,
				widget.Div(widget.Style{Height: 4}),
				widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(6, 4, 6, 4),
					BackgroundColor: &BgMuted, BorderRadius: 4},
					widget.Lucide("search", widget.IconSize(11), widget.IconColor(Accent)),
					widget.Div(widget.Style{Width: 5}), widget.NewText(a.Tool, Fg)),
			)
		}
	} else {
		kids = append(kids, widget.NewText("折叠态: "+trunc(m.Text, 30), FgMuted))
	}
	style := widget.Style{BorderColor: &Border, BorderWidth: 1, BorderRadius: 6,
		Padding: types.EdgeInsetsLTRB(12, 8, 12, 8), FlexDirection: "column"}
	if m.Collapsed {
		style.BackgroundColor = &BgSubtle
	}
	return widget.Div(style, kids)
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func dumpTree(el widget.Element, depth int) {
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "  "
	}
	off := el.Offset()
	sz := el.Size()
	fmt.Printf("%s%T pos=(%.0f,%.0f) size=(%.0f,%.0f) children=%d\n",
		prefix, el, off.X, off.Y, sz.Width, sz.Height, len(el.Children()))
	for _, c := range el.Children() {
		dumpTree(c, depth+1)
	}
}
