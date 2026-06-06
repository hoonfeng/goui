package widget

import (
	"reflect"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Popconfirm 气泡确认（el-popconfirm）────────────────────────
// 复刻 Dropdown 浮层骨架(触发器 + 点击弹浮层)；浮层内容复用 Button 拼装
// 「图标 + 标题 + 取消/确定」确认框，确定/取消执行回调并关闭。

type Popconfirm struct {
	StatelessWidget
	Trigger     Widget
	Title       string
	ConfirmText string
	CancelText  string
	OnConfirm   func()
	OnCancel    func()
	Placement   Placement
}

func NewPopconfirm(trigger Widget, title string) *Popconfirm {
	return &Popconfirm{Trigger: trigger, Title: title, Placement: PlacementTop} // 按钮文案空 → 渲染时取 i18n 默认
}
func (p *Popconfirm) WithOnConfirm(fn func()) *Popconfirm { p.OnConfirm = fn; return p }
func (p *Popconfirm) WithOnCancel(fn func()) *Popconfirm  { p.OnCancel = fn; return p }
func (p *Popconfirm) WithButtons(confirm, cancel string) *Popconfirm {
	p.ConfirmText, p.CancelText = confirm, cancel
	return p
}
func (p *Popconfirm) WithPlacement(pl Placement) *Popconfirm { p.Placement = pl; return p }

func (p *Popconfirm) CreateElement() Element {
	return &PopconfirmElement{BaseElement: BaseElement{widget: p}, pc: p}
}

type PopconfirmElement struct {
	BaseElement
	pc        *Popconfirm
	triggerEl Element
	overlayID int
	shown     bool
}

func (e *PopconfirmElement) Build() []Element {
	c := e.pc.Trigger
	if c == nil {
		e.children = nil
		return nil
	}
	if e.triggerEl != nil && reflect.TypeOf(e.triggerEl.Widget()) == reflect.TypeOf(c) {
		e.triggerEl.Update(c)
	} else {
		if e.triggerEl != nil {
			e.triggerEl.Unmount()
		}
		e.triggerEl = CreateElementFor(c)
		e.triggerEl.Mount(e, 0)
	}
	e.children = []Element{e.triggerEl}
	return e.children
}

func (e *PopconfirmElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.triggerEl != nil {
		res := e.triggerEl.Layout(ctx)
		e.triggerEl.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *PopconfirmElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.triggerEl != nil {
		e.triggerEl.Paint(cvs, offset)
	}
}

func (e *PopconfirmElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeMouseDown {
		if e.shown {
			e.hide()
		} else {
			e.show()
		}
	}
	return false
}

func (e *PopconfirmElement) show() {
	if e.shown || e.triggerEl == nil {
		return
	}
	off := e.triggerEl.Offset()
	sz := e.triggerEl.Size()
	e.overlayID = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: sz.Width, Height: sz.Height},
		Content:        e.buildBody(),
		Placement:      e.pc.Placement,
		Gap:            10,
		Transition:     "fade",
		OnClickOutside: func() { e.hide() },
	})
	e.shown = true
}

func (e *PopconfirmElement) hide() {
	if !e.shown {
		return
	}
	HideOverlay(e.overlayID)
	e.shown = false
}

// buildBody 拼装确认气泡（复用 Button）。
func (e *PopconfirmElement) buildBody() Widget {
	pc := e.pc
	return Div(Style{
		FlexDirection: "column", Gap: 12, Padding: types.EdgeInsets(14), Width: 250,
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderColor:     types.ColorRef(228, 231, 237), BorderWidth: 1, BorderRadius: 6,
		Shadow: &paint.Shadow{Offset: types.Point{X: 0, Y: 2}, Blur: 14, Color: types.ColorFromRGBA(0, 0, 0, 26)},
	},
		Div(Style{FlexDirection: "row", Gap: 8, AlignItems: "center"},
			NewText("⚠", elWarning()),
			NewText(pc.Title, elTextPrimary()),
		),
		Div(Style{FlexDirection: "row", Gap: 8, JustifyContent: "end"},
			NewButton(orI18n(pc.CancelText, "el.popconfirm.cancelButtonText"), func() {
				if pc.OnCancel != nil {
					pc.OnCancel()
				}
				e.hide()
			}).WithColor(types.ColorFromRGB(244, 244, 245)).WithTextColor(elTextRegular()),
			NewButton(orI18n(pc.ConfirmText, "el.popconfirm.confirmButtonText"), func() {
				if pc.OnConfirm != nil {
					pc.OnConfirm()
				}
				e.hide()
			}).WithColor(elPrimary()),
		),
	)
}

func (e *PopconfirmElement) Update(newWidget Widget) {
	if np, ok := newWidget.(*Popconfirm); ok {
		e.pc = np
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
