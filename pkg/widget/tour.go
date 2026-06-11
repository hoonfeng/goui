package widget

import (
	"fmt"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Tour 漫游引导（el-tour，步骤卡片版）──────────────────────
// 命令式 ShowTour(steps...)：全屏遮罩 + 居中步骤卡片。
// 卡片仿 Dialog 拼装：标题/内容/进度用 Text，上一步/下一步复用真 Button，
// ✕ 用右上角热区(同 Dialog 范式)。current 步进经 OnNeedsLayout 重建卡片。
// 目标元素高亮为进阶；当前为分步引导卡片。

type TourStep struct {
	Title   string
	Content string
}

// ShowTour 弹出引导，返回浮层 id。
func ShowTour(steps ...TourStep) int {
	if len(steps) == 0 {
		return 0
	}
	card := &tourCard{steps: steps}
	var id int
	card.close = func() { HideOverlay(id) }
	id = ShowOverlay(&OverlayEntry{Content: card, Fullscreen: true, Transition: "fade"})
	return id
}

// TourPreview 返回引导卡片 widget（全屏遮罩 + 居中卡），供 demo/headless 验证。
func TourPreview(steps ...TourStep) Widget {
	return &tourCard{steps: steps}
}

type tourCard struct {
	StatelessWidget
	steps []TourStep
	close func()
}

func (c *tourCard) CreateElement() Element {
	return &tourCardElement{BaseElement: BaseElement{widget: c}, card: c}
}

const tourW = 380.0

type tourCardElement struct {
	BaseElement
	card    *tourCard
	current int
	panel   Element
}

// buildTourPanel 把一步引导拼成白色圆角卡片：header[标题+✕] / body[内容] / footer[进度+按钮]。
func buildTourPanel(steps []TourStep, current int, onPrev, onNext func()) Widget {
	st := steps[current]
	nextLabel := "下一步"
	if current == len(steps)-1 {
		nextLabel = "完成"
	}

	header := divCore(
		Style{Padding: types.EdgeInsetsLTRB(22, 18, 16, 8), FlexDirection: "row", AlignItems: "center"},
		[]Widget{
			&Text{Text: st.Title, Color: elTextPrimary(), Font: canvas.Font{Family: "sans-serif", Size: 17, Weight: canvas.FontWeightBold}},
			SpacerDiv(),
			&Text{Text: "✕", Color: elPlaceholder(), Font: canvas.Font{Family: "sans-serif", Size: 15}},
		},
	)
	body := divCore(
		Style{Padding: types.EdgeInsetsLTRB(22, 4, 22, 12)},
		[]Widget{&Text{Text: st.Content, Color: elTextRegular(), Font: canvas.Font{Family: "sans-serif", Size: 14}}},
	)
	footerKids := []Widget{
		&Text{Text: fmt.Sprintf("%d / %d", current+1, len(steps)), Color: elTextSecondary(), Font: canvas.Font{Family: "sans-serif", Size: 13}},
		SpacerDiv(),
	}
	if current > 0 {
		footerKids = append(footerKids, NewButton("上一步", onPrev))
	}
	footerKids = append(footerKids, NewButton(nextLabel, onNext))
	footer := divCore(
		Style{Padding: types.EdgeInsetsLTRB(22, 4, 16, 16), FlexDirection: "row", AlignItems: "center", Gap: 10},
		footerKids,
	)

	return divCore(
		Style{
			Width: tourW, BackgroundColor: types.ColorRef(255, 255, 255), BorderRadius: 8,
			Shadow: &paint.Shadow{Offset: types.Point{X: 0, Y: 4}, Blur: 16, Color: types.ColorFromRGBA(0, 0, 0, 28)},
		},
		[]Widget{header, body, footer},
	)
}

func (e *tourCardElement) Build() []Element {
	c := e.card
	relayout := func() {
		if OnNeedsLayout != nil {
			OnNeedsLayout()
		}
	}
	e.panel = CreateElementFor(buildTourPanel(c.steps, e.current,
		func() { // 上一步
			if e.current > 0 {
				e.current--
				relayout()
			}
		},
		func() { // 下一步 / 完成
			if e.current == len(c.steps)-1 {
				if c.close != nil {
					c.close()
				}
			} else {
				e.current++
				relayout()
			}
		},
	))
	e.panel.Mount(e, 0)
	e.children = []Element{e.panel}
	return e.children
}

func (e *tourCardElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	maxW := ctx.Constraints.MaxWidth
	maxH := ctx.Constraints.MaxHeight
	if maxW >= float64(1<<30) {
		maxW = 600
	}
	if maxH >= float64(1<<30) {
		maxH = 400
	}
	e.size = types.Size{Width: maxW, Height: maxH}

	res := e.panel.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
		MinWidth: 0, MaxWidth: maxW, MinHeight: 0, MaxHeight: maxH,
	}})
	ps := res.Size
	px := (maxW - ps.Width) / 2
	py := (maxH - ps.Height) / 2
	if px < 0 {
		px = 0
	}
	if py < 0 {
		py = 0
	}
	e.panel.SetPosition(types.Point{X: px, Y: py})
	return layout.LayoutResult{Size: e.size}
}

func (e *tourCardElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	mask := paint.DefaultPaint()
	mask.Color = types.ColorFromRGBA(0, 0, 0, 90)
	cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, mask)
	if e.panel != nil {
		e.panel.Paint(cvs, offset)
	}
}

func (e *tourCardElement) HandleEvent(ev event.Event) bool {
	// 上一步/下一步是真 Button 子元素，事件自动路由给它们；
	// 这里只兜底 ✕ 右上角热区(同 Dialog 范式)。
	if ev.Type() != event.TypeMouseDown {
		return false
	}
	me, ok := ev.(*event.MouseEvent)
	if !ok || e.panel == nil {
		return false
	}
	pp := e.panel.Offset()
	ps := e.panel.Size()
	if me.X >= pp.X+ps.Width-44 && me.X <= pp.X+ps.Width && me.Y >= pp.Y && me.Y <= pp.Y+44 {
		if e.card.close != nil {
			e.card.close()
		}
		return true
	}
	return false
}

func (e *tourCardElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*tourCard); ok {
		e.card = nc
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
