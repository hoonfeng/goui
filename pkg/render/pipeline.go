package render

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

var DebugPerf bool

type Pipeline struct {
	rootLayer      *Layer
	rootElement    widget.Element
	finalCanvas    canvas.Canvas
	width          int
	height         int
	needsRepaint   atomic.Bool
	needsLayout    atomic.Bool
	didRender      atomic.Bool
	pendingUpdates atomic.Int64
	needsRebuild   atomic.Bool
	dirtyRect      types.Rect
	perfFrames     int
	perfTotal      time.Duration
	perfBuild      time.Duration
	perfLayout     time.Duration
	perfPaint      time.Duration
	perfFlush      time.Duration
	perfLastLog    time.Time
	perfTmpBuild   time.Duration
	BackgroundColor types.Color
}

func NewPipeline(width, height int, finalCanvas canvas.Canvas) *Pipeline {
	if os.Getenv("GOUIPERF") == "1" {
		DebugPerf = true
	}
	p := &Pipeline{
		rootLayer: NewLayer(types.Rect{X: 0, Y: 0, Width: float64(width), Height: float64(height)}),
		finalCanvas: finalCanvas,
		width:       width,
		height:      height,
		dirtyRect:   types.Rect{X: 0, Y: 0, Width: float64(width), Height: float64(height)},
		BackgroundColor:  types.ColorWhite,
	}
	p.needsRepaint.Store(true)
	p.needsLayout.Store(true)
	p.needsRebuild.Store(true)
	return p
}

func (p *Pipeline) ConsumePendingUpdates() bool {
	if p.pendingUpdates.Load() == 0 {
		return false
	}
	p.pendingUpdates.Store(0)
	p.markFullDirty()
	p.needsRebuild.Store(true)
	p.needsLayout.Store(true)
	p.needsRepaint.Store(true)
	return true
}

func (p *Pipeline) EnqueueUpdate()  { p.pendingUpdates.Add(1) }
func (p *Pipeline) EnqueueRepaint() { p.needsRepaint.Store(true); p.markFullDirty() }

func (p *Pipeline) markFullDirty() {
	p.dirtyRect = types.Rect{X: 0, Y: 0, Width: float64(p.width), Height: float64(p.height)}
}

func (p *Pipeline) MarkRectDirty(r types.Rect) {
	if p.dirtyRect.Width == 0 && p.dirtyRect.Height == 0 {
		p.dirtyRect = r
	} else {
		p.dirtyRect = mergeRects(p.dirtyRect, r)
	}
	p.needsRepaint.Store(true)
}

func mergeRects(a, b types.Rect) types.Rect {
	minX := min(a.X, b.X)
	minY := min(a.Y, b.Y)
	maxX := max(a.X+a.Width, b.X+b.Width)
	maxY := max(a.Y+a.Height, b.Y+b.Height)
	return types.Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

func (p *Pipeline) SetRootElement(el widget.Element) {
	p.rootElement = el
	p.markFullDirty()
	p.needsRebuild.Store(true)
	p.needsLayout.Store(true)
	p.needsRepaint.Store(true)
}

func (p *Pipeline) SetRootLayer(layer *Layer) {
	p.rootLayer = layer
	p.markFullDirty()
	p.needsRepaint.Store(true)
}

func (p *Pipeline) MarkNeedsRepaint()                        { p.needsRepaint.Store(true); p.markFullDirty() }
func (p *Pipeline) MarkNeedsLayout()                          { p.markFullDirty(); p.needsLayout.Store(true); p.needsRepaint.Store(true); p.needsRebuild.Store(true) }
func (p *Pipeline) NeedsRepaint() bool                        { return p.needsRepaint.Load() }
func (p *Pipeline) NeedsLayout() bool                         { return p.needsLayout.Load() }

func (p *Pipeline) Resize(width, height int) {
	p.width = width
	p.height = height
	p.rootLayer.Bounds = types.Rect{X: 0, Y: 0, Width: float64(width), Height: float64(height)}
	p.markFullDirty()
	p.needsLayout.Store(true)
	p.needsRepaint.Store(true)
	if resizer, ok := p.finalCanvas.(interface{ Resize(int, int) }); ok {
		resizer.Resize(width, height)
	}
}

func (p *Pipeline) PerformLayout() {
	if p.rootElement == nil {
		return
	}
	if p.needsRebuild.Load() {
		p.buildTree(p.rootElement)
		p.needsRebuild.Store(false)
	}
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: float64(p.width),
			MinHeight: 0, MaxHeight: float64(p.height),
		},
	}
	p.rootElement.Layout(ctx)
	p.needsLayout.Store(false)
}

func (p *Pipeline) EnsureLayout() {
	if p.needsLayout.Load() && p.rootElement != nil {
		p.PerformLayout()
	}
}

func (p *Pipeline) buildTree(el widget.Element) {
	// 先检查子树稳定性：首次 Build 时 builtOnce=false, Stable=false,
	// 确保完整递归构建子树；SetState 后 buildDirty=true, Stable=false 触发重建递归
	if checker, ok := el.(widget.SubtreeStableChecker); ok && checker.Stable() {
		return
	}
	children := el.Build()
	for _, child := range children {
		p.buildTree(child)
	}
}

func (p *Pipeline) Render() error {
	p.didRender.Store(false)
	needsWork := p.needsRepaint.Load() || p.needsLayout.Load()
	if !needsWork {
		return nil
	}
	frameStart := time.Now()
	var tLayout, tPaint, tFlush time.Duration
	if p.needsLayout.Load() {
		t0 := time.Now()
		p.PerformLayout()
		tLayout = time.Since(t0)
	}
	p.clearCanvas()
	if p.rootElement != nil {
		t0 := time.Now()
		p.finalCanvas.Save()
		dr := p.dirtyRect
		p.finalCanvas.ClipRect(dr.X, dr.Y, dr.Width, dr.Height)
		p.rootElement.Paint(p.finalCanvas, types.Point{})
		p.finalCanvas.Restore()
		tPaint = time.Since(t0)
	}
	p.dirtyRect = types.Rect{}
	t0 := time.Now()
	if err := p.finalCanvas.Flush(); err != nil {
		return err
	}
	tFlush = time.Since(t0)
	frameTotal := time.Since(frameStart)
	p.needsRepaint.Store(false)
	p.didRender.Store(true)
	if DebugPerf {
		p.perfFrames++
		p.perfTotal += frameTotal
		p.perfBuild += p.perfTmpBuild
		p.perfTmpBuild = 0
		p.perfLayout += tLayout
		p.perfPaint += tPaint
		p.perfFlush += tFlush
		if time.Since(p.perfLastLog) >= 5*time.Second {
			f := p.perfFrames
			if f > 0 {
				avgTotal := p.perfTotal / time.Duration(f)
				avgBuild := p.perfBuild / time.Duration(f)
				avgLayout := p.perfLayout / time.Duration(f)
				avgPaint := p.perfPaint / time.Duration(f)
				avgFlush := p.perfFlush / time.Duration(f)
				fmt.Fprintf(os.Stderr, "[perf] fps=%d frames=%d build=%v layout=%v paint=%v flush=%v total=%v\n",
					int(float64(f)/5.0), f, avgBuild, avgLayout, avgPaint, avgFlush, avgTotal)
			}
			p.perfFrames = 0
			p.perfTotal = 0
			p.perfBuild = 0
			p.perfLayout = 0
			p.perfPaint = 0
			p.perfFlush = 0
			p.perfLastLog = time.Now()
		}
	}
	return nil
}

func (p *Pipeline) DidRender() bool          { return p.didRender.Load() }

func (p *Pipeline) clearCanvas() {
	dr := p.dirtyRect
	if dr.Width <= 0 || dr.Height <= 0 {
		return
	}
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = p.BackgroundColor
	p.finalCanvas.DrawRect(dr.X, dr.Y, dr.Width, dr.Height, bgPaint)
}

func (p *Pipeline) HitTest(x, y float64) widget.Element {
	if p.rootElement == nil {
		return nil
	}
	return hitTestRecursive(p.rootElement, x, y)
}

func hitTestRecursive(el widget.Element, x, y float64) widget.Element {
	childX, childY := x, y
	if tr, ok := el.(widget.HitTestTransformer); ok {
		childX, childY = tr.TransformHitPoint(x, y)
	}
	children := el.Children()
	for i := len(children) - 1; i >= 0; i-- {
		if result := hitTestRecursive(children[i], childX, childY); result != nil {
			return result
		}
	}
	if ig, ok := el.(widget.HitTestIgnorer); ok && ig.IgnoreHitTest() {
		return nil
	}
	offset := el.Offset()
	size := el.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return nil
	}
	if x >= offset.X && x <= offset.X+size.Width &&
		y >= offset.Y && y <= offset.Y+size.Height {
		return el
	}
	return nil
}

func (p *Pipeline) Canvas() canvas.Canvas            { return p.finalCanvas }
func (p *Pipeline) Width() int                         { return p.width }
func (p *Pipeline) Height() int                        { return p.height }
func (p *Pipeline) RootElement() widget.Element        { return p.rootElement }
