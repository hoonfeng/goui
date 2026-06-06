package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── ResizeHandle 可拖动分隔条（IDE 停靠面板调宽/调高）──────────
//
// 通用拖动手柄：自身既是一条分隔线，又是拖动靶子。拖动时持续回调"相对上次的
// 位移增量"，由上层据此调整相邻面板尺寸并 SetState——因此它对左栏/右栏/底栏
// 完全通用，加还是减由上层符号决定。一行即可装一根：
//
//	widget.VResize(lineColor, activeColor, func(dx float64) { p.LeftW += dx; s.SetState() })
//
// 设计要点：视觉只占一条细线，但整个 Thickness 区域都可命中（好点中）；拖动中
// 走 OnNeedsLayout，buildTree 类型匹配复用本 Element，故 dragging/last 不丢失。

type ResizeHandle struct {
	StatelessWidget
	Vertical  bool               // true=竖条（左右拖、回调 dx，用于调宽）；false=横条（上下拖、回调 dy，用于调高）
	Thickness float64            // 命中+占位厚度（像素，默认 6）
	Line      types.Color        // 平时分隔线颜色
	Active    types.Color        // 拖动时整条高亮颜色
	OnDrag    func(delta float64) // 每次移动回调相对上次的位移增量（像素）
}

// VResize 竖直分隔条（左右拖动调宽）。Thickness 默认 6（命中够宽）；粗细/颜色可链式覆盖。
func VResize(line, active types.Color, onDrag func(delta float64)) *ResizeHandle {
	return &ResizeHandle{Vertical: true, Thickness: 6, Line: line, Active: active, OnDrag: onDrag}
}

// HResize 水平分隔条（上下拖动调高）。
func HResize(line, active types.Color, onDrag func(delta float64)) *ResizeHandle {
	return &ResizeHandle{Vertical: false, Thickness: 6, Line: line, Active: active, OnDrag: onDrag}
}

// WithThickness 覆盖分隔条粗细（像素），链式：widget.VResize(...).WithThickness(4)。
func (h *ResizeHandle) WithThickness(t float64) *ResizeHandle { h.Thickness = t; return h }

// WithColors 覆盖分隔色 / 高亮色（hover、拖动时用高亮色），链式可选。
func (h *ResizeHandle) WithColors(line, active types.Color) *ResizeHandle {
	h.Line, h.Active = line, active
	return h
}

func (h *ResizeHandle) CreateElement() Element {
	return &resizeHandleElement{BaseElement: BaseElement{widget: h}, h: h}
}

type resizeHandleElement struct {
	BaseElement
	h        *ResizeHandle
	dragging bool
	hover    bool
	last     float64 // 上次鼠标在主轴上的坐标
}

func (e *resizeHandleElement) Build() []Element { return nil }

func (e *resizeHandleElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	t := e.h.Thickness
	if t <= 0 {
		t = 6
	}
	var w, hh float64
	if e.h.Vertical { // 竖条：固定宽、撑满父高
		w = t
		hh = ctx.Constraints.MaxHeight
		if hh >= float64(1<<30) {
			hh = ctx.Constraints.MinHeight
		}
	} else { // 横条：固定高、撑满父宽
		hh = t
		w = ctx.Constraints.MaxWidth
		if w >= float64(1<<30) {
			w = ctx.Constraints.MinWidth
		}
	}
	e.size = types.Size{Width: w, Height: hh}
	return layout.LayoutResult{Size: e.size}
}

func (e *resizeHandleElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	// 整条填充：平时分隔色、拖动高亮色。必须填满整个命中宽度，否则其余像素透出根
	// 背景（白），出现“白条夹灰线”。颜色全部取自 Line/Active 参数（非硬编码）。
	col := e.h.Line
	if e.dragging || e.hover {
		col = e.h.Active
	}
	p := paint.DefaultPaint()
	p.Color = col
	cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, p)
}

func (e *resizeHandleElement) axis(me *event.MouseEvent) float64 {
	if e.h.Vertical {
		return me.X
	}
	return me.Y
}

func (e *resizeHandleElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseDown:
		e.dragging = true
		e.last = e.axis(me)
		if RequestPointerCapture != nil {
			RequestPointerCapture(e) // 抢捕获：拖到面板上方也持续收到 move
		}
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	case event.TypeMouseMove:
		if e.dragging {
			cur := e.axis(me)
			d := cur - e.last
			e.last = cur
			if d != 0 && e.h.OnDrag != nil {
				e.h.OnDrag(d)
			}
			return true
		}
	case event.TypeMouseUp:
		// 仅 MouseUp 结束拖动。绝不能在 MouseLeave 时结束——拖动中鼠标常移出这条窄条，
		// app 的 hover 检测随即给本元素派发 MouseLeave；若据此清 dragging，拖动会在鼠标
		// 稍离分隔条时就中断（”拖几像素就停、鼠标脱离分隔条”的真正原因）。已 SetCapture，
		// move/up 仍会持续送达本元素，无需 MouseLeave 兜底。
		if e.dragging {
			e.dragging = false
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
	case event.TypeMouseEnter:
		if !e.hover {
			e.hover = true
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		return true
	case event.TypeMouseLeave:
		// 只更新 hover 视觉，绝不在此碰 dragging（拖动只由 MouseUp 结束）。
		if e.hover {
			e.hover = false
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		return true
	}
	return false
}

func (e *resizeHandleElement) Update(nw Widget) {
	if h, ok := nw.(*ResizeHandle); ok {
		e.h = h // 同步新闭包（OnDrag 捕获最新状态指针）；不重置 dragging/last，拖动跨重建不丢
	}
	e.widget = nw
	e.dirty = true
}

// Cursor 让鼠标悬停/拖动本分隔条时显示对应的 resize 光标（竖条 ↔ / 横条 ↕）。
func (e *resizeHandleElement) Cursor() int {
	if e.h.Vertical {
		return CursorResizeH
	}
	return CursorResizeV
}
