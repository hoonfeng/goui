package widget

import (
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Tabs 标签页（1:1 复刻 el-tabs line 类型）─────────────────────
// 顶部标签栏 + 底部激活滑块(平滑移动) + 下方切换内容区。
// 点标签切换：activeIndex 变 → 重建当前 panel 内容(同 panel 复用保状态) + relayout。

// TabPane 一个标签页(标签 + 内容)。
type TabPane struct {
	Label   string
	Content Widget
}

// Tabs 标签页容器。
type Tabs struct {
	StatelessWidget
	Panes       []TabPane
	ActiveIndex int
	OnChange    func(int)
}

func NewTabs(panes ...TabPane) *Tabs        { return &Tabs{Panes: panes} }
func (t *Tabs) WithActive(i int) *Tabs      { t.ActiveIndex = i; return t }
func (t *Tabs) WithOnChange(fn func(int)) *Tabs { t.OnChange = fn; return t }

const tabsBarH = 42.0

func (t *Tabs) CreateElement() Element {
	return &TabsElement{
		BaseElement: BaseElement{widget: t},
		tabs:        t,
		activeIndex: t.ActiveIndex,
		builtIndex:  -1,
	}
}

type TabsElement struct {
	BaseElement
	tabs        *Tabs
	contentEl   Element
	activeIndex int
	builtIndex  int          // 当前已构建内容对应的 index(用于复用判断)
	labelRects  []types.Rect // 每个标签矩形(相对本元素左上)
	sliderX     float64      // 当前滑块 x(动画插值)
	sliderW     float64      // 当前滑块宽
	sliderInit  bool
	sliderCtrl  *animation.Controller
}

// Build 构建当前 panel 内容(同 panel 复用以保留状态，切换则重建)。
func (e *TabsElement) Build() []Element {
	if len(e.tabs.Panes) == 0 || e.activeIndex < 0 || e.activeIndex >= len(e.tabs.Panes) {
		if e.contentEl != nil {
			e.contentEl.Unmount()
			e.contentEl = nil
		}
		e.children = nil
		return nil
	}
	content := e.tabs.Panes[e.activeIndex].Content
	if e.contentEl != nil && e.builtIndex == e.activeIndex {
		e.contentEl.Update(content)
	} else {
		if e.contentEl != nil {
			e.contentEl.Unmount()
		}
		e.contentEl = CreateElementFor(content)
		e.contentEl.Mount(e, 0)
		e.builtIndex = e.activeIndex
	}
	e.children = []Element{e.contentEl}
	return e.children
}

func (e *TabsElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	font := canvas.DefaultFont()
	font.Size = 14

	// 标签矩形(横排，每个左右 16 内边距)
	x := 0.0
	e.labelRects = e.labelRects[:0]
	for _, p := range e.tabs.Panes {
		w := canvas.MeasureTextGlobal(p.Label, font).Width + 32
		e.labelRects = append(e.labelRects, types.Rect{X: x, Y: 0, Width: w, Height: tabsBarH})
		x += w
	}

	availW := ctx.Constraints.MaxWidth
	contentH := 0.0
	if e.contentEl != nil {
		res := e.contentEl.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: availW, MinHeight: 0, MaxHeight: ctx.Constraints.MaxHeight,
		}})
		e.contentEl.SetPosition(types.Point{X: 0, Y: tabsBarH + 16})
		contentH = res.Size.Height
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: availW, Height: tabsBarH + 16 + contentH})

	// 滑块首帧定位到激活标签
	if !e.sliderInit && e.activeIndex < len(e.labelRects) {
		r := e.labelRects[e.activeIndex]
		e.sliderX = r.X + 16
		e.sliderW = r.Width - 32
		e.sliderInit = true
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *TabsElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	font := canvas.DefaultFont()
	font.Size = 14

	// 标签文字
	for i, r := range e.labelRects {
		color := elTextRegular()
		if i == e.activeIndex {
			color = elPrimary()
		}
		tr := types.Rect{X: pos.X + r.X, Y: pos.Y, Width: r.Width, Height: tabsBarH}
		canvas.DrawTextAligned(cvs, e.tabs.Panes[i].Label, tr, font, color, canvas.HAlignCenter, canvas.VAlignMiddle)
	}

	// 底部分隔线
	lp := paint.DefaultStrokePaint()
	lp.Color = elBorderLight()
	lp.StrokeWidth = 1
	cvs.DrawLine(pos.X, pos.Y+tabsBarH, pos.X+e.size.Width, pos.Y+tabsBarH, lp)

	// 激活滑块(蓝条，压在分隔线上)
	sp := paint.DefaultPaint()
	sp.Color = elPrimary()
	cvs.DrawRect(pos.X+e.sliderX, pos.Y+tabsBarH-2, e.sliderW, 2, sp)

	// 内容
	if e.contentEl != nil {
		e.contentEl.Paint(cvs, offset)
	}
}

// animateSlider 把滑块平滑移动到目标标签下方。
func (e *TabsElement) animateSlider(target int) {
	if target < 0 || target >= len(e.labelRects) {
		return
	}
	r := e.labelRects[target]
	toX, toW := r.X+16, r.Width-32
	fromX, fromW := e.sliderX, e.sliderW
	if e.sliderCtrl != nil {
		e.sliderCtrl.Stop()
	}
	e.sliderCtrl = animation.NewController(220*time.Millisecond, animation.EaseOutCubic)
	e.sliderCtrl.OnUpdate = func(v float64) {
		e.sliderX = animation.LerpFloat(fromX, toX, v)
		e.sliderW = animation.LerpFloat(fromW, toW, v)
		e.MarkNeedsPaint()
	}
	e.sliderCtrl.Start()
}

func (e *TabsElement) HandleEvent(ev event.Event) bool {
	if ev.Type() != event.TypeMouseDown {
		return false
	}
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	localX, localY := me.X-pos.X, me.Y-pos.Y
	if localY > tabsBarH {
		return false
	}
	for i, r := range e.labelRects {
		if localX >= r.X && localX < r.X+r.Width {
			if i != e.activeIndex {
				e.activeIndex = i
				if e.tabs.OnChange != nil {
					e.tabs.OnChange(i)
				}
				e.animateSlider(i)
				if OnNeedsLayout != nil {
					OnNeedsLayout() // 切换内容 → 重建 + relayout
				}
			}
			return true
		}
	}
	return false
}

func (e *TabsElement) Update(newWidget Widget) {
	if nt, ok := newWidget.(*Tabs); ok {
		e.tabs = nt
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
