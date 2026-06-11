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

// ─── Carousel 走马灯（el-carousel）─────────────────────────────
// 轮播任意 Widget 子项：显示当前项 + 底部指示条 + 左右箭头 + 可选自动播放。
// 子项都布局在 (0,0)，只绘制 current；切换由箭头/指示/定时器驱动。

type Carousel struct {
	StatelessWidget
	Items    []Widget
	Height   float64
	Interval time.Duration // 0=不自动播放
}

func NewCarousel(items ...Widget) *Carousel        { return &Carousel{Items: items, Height: 180} }
func (c *Carousel) WithHeight(h float64) *Carousel  { c.Height = h; return c }
func (c *Carousel) WithInterval(d time.Duration) *Carousel { c.Interval = d; return c }

func (c *Carousel) CreateElement() Element {
	return &CarouselElement{BaseElement: BaseElement{widget: c}, carousel: c}
}

type CarouselElement struct {
	BaseElement
	carousel *Carousel
	itemEls  []Element
	current  int
	timer    *animation.Controller
	ticking  bool
}

func (e *CarouselElement) Build() []Element {
	if e.itemEls == nil {
		e.itemEls = make([]Element, len(e.carousel.Items))
		for i, it := range e.carousel.Items {
			e.itemEls[i] = CreateElementFor(it)
			e.itemEls[i].Mount(e, i)
		}
	} else {
		for i, it := range e.carousel.Items {
			if i < len(e.itemEls) {
				e.itemEls[i].Update(it)
			}
		}
	}
	e.children = e.itemEls
	return e.children
}

func (e *CarouselElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := ctx.Constraints.MaxWidth
	if w >= float64(1<<30) {
		w = 400
	}
	h := e.carousel.Height
	for _, el := range e.itemEls {
		el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: w, MaxWidth: w, MinHeight: h, MaxHeight: h}})
		el.SetPosition(types.Point{})
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *CarouselElement) ensureTimer() {
	if e.ticking || e.carousel.Interval <= 0 || len(e.itemEls) <= 1 {
		return
	}
	e.ticking = true
	e.startTick()
}

func (e *CarouselElement) startTick() {
	e.timer = animation.NewController(e.carousel.Interval, nil)
	e.timer.OnDone = func() {
		e.current = (e.current + 1) % len(e.itemEls)
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		e.startTick()
	}
	e.timer.Start()
}

func (e *CarouselElement) Paint(cvs canvas.Canvas, offset types.Point) {
	e.ensureTimer()
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height

	// 裁剪 + 当前项
	cvs.Save()
	cvs.ClipRect(pos.X, pos.Y, W, H)
	if e.current >= 0 && e.current < len(e.itemEls) {
		e.itemEls[e.current].Paint(cvs, offset)
	}
	cvs.Restore()

	// 底部指示条
	n := len(e.itemEls)
	if n > 1 {
		dotY := pos.Y + H - 16
		totalW := float64(n)*22 - 12
		startX := pos.X + (W-totalW)/2
		for i := 0; i < n; i++ {
			dp := paint.DefaultPaint()
			if i == e.current {
				dp.Color = elSurface()
			} else {
				dp.Color = types.ColorFromRGBA(255, 255, 255, 110)
			}
			cvs.DrawRoundedRect(startX+float64(i)*22, dotY, 16, 4, 2, dp)
		}
		// 左右箭头
		paintCarouselArrow(cvs, pos.X+20, pos.Y+H/2, "‹")
		paintCarouselArrow(cvs, pos.X+W-20, pos.Y+H/2, "›")
	}
}

func paintCarouselArrow(cvs canvas.Canvas, cx, cy float64, ch string) {
	bg := paint.DefaultPaint()
	bg.Color = types.ColorFromRGBA(0, 0, 0, 110)
	cvs.DrawCircle(cx, cy, 14, bg)
	f := canvas.DefaultFont()
	f.Size = 18
	canvas.DrawTextAligned(cvs, ch, types.Rect{X: cx - 14, Y: cy - 14, Width: 28, Height: 28},
		f, types.ColorFromRGB(255, 255, 255), canvas.HAlignCenter, canvas.VAlignMiddle)
}

func (e *CarouselElement) goTo(i int) {
	n := len(e.itemEls)
	if n == 0 {
		return
	}
	e.current = (i%n + n) % n
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

func (e *CarouselElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	if ev.Type() == event.TypeMouseDown {
		lx := me.X - e.Offset().X
		if lx < 44 {
			e.goTo(e.current - 1)
			return true
		}
		if lx > e.size.Width-44 {
			e.goTo(e.current + 1)
			return true
		}
	}
	return false
}

func (e *CarouselElement) Unmount() {
	if e.timer != nil {
		e.timer.Stop()
		e.timer = nil
	}
	e.BaseElement.Unmount()
}

func (e *CarouselElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*Carousel); ok {
		e.carousel = nc
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
