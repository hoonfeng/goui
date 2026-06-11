package widget

import (
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Tabs 标签页（1:1 复刻 el-tabs line 类型）─────────────────────
// 顶部标签栏 + 底部激活滑块(平滑移动) + 下方切换内容区。
// 点标签切换：activeIndex 变 → 重建当前 panel 内容(同 panel 复用保状态) + relayout。

// TabPane 一个标签页(标签 + 可选图标 + 内容)。
type TabPane struct {
	Label   string
	Icon    string // 可选 lucide 图标（标签左侧）
	Content Widget
}

// Tabs 标签页容器。零值颜色=el 亮色默认；设置后可深色化。Compact=IDE 侧栏紧凑模式（矮条、无内容间隙、内容填满剩余）。
type Tabs struct {
	StatelessWidget
	Panes         []TabPane
	ActiveIndex   int
	OnChange      func(int)
	Compact       bool        // 紧凑模式：矮标签条 + 内容紧贴下方并填满
	ActiveColor   types.Color // 激活标签文字/滑块色（零=elPrimary）
	InactiveColor types.Color // 非激活标签文字色（零=elTextRegular）
	BarColor      types.Color // 标签条背景（零=透明）
	LineColor     types.Color // 底部分隔线色（零=elBorderLight）
}

func NewTabs(panes ...TabPane) *Tabs            { return &Tabs{Panes: panes} }
func (t *Tabs) WithActive(i int) *Tabs          { t.ActiveIndex = i; return t }
func (t *Tabs) WithOnChange(fn func(int)) *Tabs { t.OnChange = fn; return t }

const tabsBarH = 42.0
const tabsCompactBarH = 34.0

// barHeight 据 Compact 返回标签条高。
func (t *Tabs) barHeight() float64 {
	if t.Compact {
		return tabsCompactBarH
	}
	return tabsBarH
}

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

func (e *TabsElement) tabFont() canvas.Font {
	f := canvas.DefaultFont()
	if e.tabs.Compact {
		f.Size = 13
	} else {
		f.Size = 14
	}
	return f
}

func (e *TabsElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	font := e.tabFont()
	barH := e.tabs.barHeight()

	// 标签矩形(横排，左右 12 内边距；有图标再 +19=图标 14+间隙 5)
	x := 0.0
	e.labelRects = e.labelRects[:0]
	for _, p := range e.tabs.Panes {
		w := canvas.MeasureTextGlobal(p.Label, font).Width + 24
		if p.Icon != "" {
			w += 19
		}
		e.labelRects = append(e.labelRects, types.Rect{X: x, Y: 0, Width: w, Height: barH})
		x += w
	}

	availW := ctx.Constraints.MaxWidth
	gap := 16.0
	if e.tabs.Compact { // IDE 紧凑：内容紧贴标签条
		gap = 0
	}
	contentTop := barH + gap
	maxH := ctx.Constraints.MaxHeight
	bounded := maxH < float64(1<<30)
	contentH := 0.0
	if e.contentEl != nil {
		remain := maxH - contentTop
		if remain < 0 {
			remain = 0
		}
		cc := layout.BoxConstraints{MinWidth: 0, MaxWidth: availW, MinHeight: 0, MaxHeight: remain}
		if !bounded {
			cc.MaxHeight = float64(1 << 30) // 无界：内容取自然高
		} else if e.tabs.Compact {
			cc.MinHeight = remain // 紧凑+有界：内容填满剩余高（IDE 面板撑满）
		}
		res := e.contentEl.Layout(&layout.LayoutContext{Constraints: cc})
		e.contentEl.SetPosition(types.Point{X: 0, Y: contentTop})
		contentH = res.Size.Height
	}
	h := contentTop + contentH // el-tabs：收缩到内容（保持既有用户行为）
	if e.tabs.Compact && bounded {
		h = maxH // 紧凑：填满整个区域（IDE 侧栏）
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: availW, Height: h})

	// 滑块首帧定位到激活标签（紧凑=全标签下划线）
	if !e.sliderInit && e.activeIndex < len(e.labelRects) {
		e.sliderX, e.sliderW = e.sliderTarget(e.activeIndex)
		e.sliderInit = true
	}
	return layout.LayoutResult{Size: e.size}
}

// sliderTarget 激活滑块的目标 x/宽：紧凑=整标签宽；el 模式=标签内缩 16。
func (e *TabsElement) sliderTarget(i int) (float64, float64) {
	r := e.labelRects[i]
	if e.tabs.Compact {
		return r.X, r.Width
	}
	return r.X + 16, r.Width - 32
}

func (e *TabsElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	font := e.tabFont()
	barH := e.tabs.barHeight()

	active := e.tabs.ActiveColor
	if active.A == 0 {
		active = elPrimary()
	}
	inactive := e.tabs.InactiveColor
	if inactive.A == 0 {
		inactive = elTextRegular()
	}
	line := e.tabs.LineColor
	if line.A == 0 {
		line = elBorderLight()
	}

	if e.tabs.BarColor.A > 0 { // 标签条背景（深色侧栏用）
		bp := paint.DefaultPaint()
		bp.Color = e.tabs.BarColor
		cvs.DrawRect(pos.X, pos.Y, e.size.Width, barH, bp)
	}

	// 标签：图标 + 文字（居中成组）
	for i, r := range e.labelRects {
		color := inactive
		if i == e.activeIndex {
			color = active
		}
		p := e.tabs.Panes[i]
		labelW := canvas.MeasureTextGlobal(p.Label, font).Width
		iconW := 0.0
		if p.Icon != "" {
			iconW = 14 + 5
		}
		startX := pos.X + r.X + (r.Width-(iconW+labelW))/2
		if p.Icon != "" {
			PaintLucide(cvs, p.Icon, startX, pos.Y+(barH-14)/2, 14, 2, color)
		}
		tr := types.Rect{X: startX + iconW, Y: pos.Y, Width: labelW + 2, Height: barH}
		canvas.DrawTextAligned(cvs, p.Label, tr, font, color, canvas.HAlignLeft, canvas.VAlignMiddle)
	}

	lp := paint.DefaultStrokePaint() // 底部分隔线
	lp.Color = line
	lp.StrokeWidth = 1
	cvs.DrawLine(pos.X, pos.Y+barH, pos.X+e.size.Width, pos.Y+barH, lp)

	sp := paint.DefaultPaint() // 激活滑块（压在分隔线上）
	sp.Color = active
	cvs.DrawRect(pos.X+e.sliderX, pos.Y+barH-2, e.sliderW, 2, sp)

	if e.contentEl != nil {
		e.contentEl.Paint(cvs, offset)
	}
}

// animateSlider 把滑块平滑移动到目标标签下方。
func (e *TabsElement) animateSlider(target int) {
	if target < 0 || target >= len(e.labelRects) {
		return
	}
	toX, toW := e.sliderTarget(target)
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
	if localY > e.tabs.barHeight() {
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
		// 受控：外部 ActiveIndex 变化（如菜单切视图）→ 同步并动画滑块。
		if nt.ActiveIndex != e.activeIndex && nt.ActiveIndex >= 0 && nt.ActiveIndex < len(nt.Panes) {
			e.activeIndex = nt.ActiveIndex
			if e.sliderInit {
				e.animateSlider(e.activeIndex)
			}
		}
		e.dirty = true
	}
}
