package widget

import (
	"strings"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Mention 提及（el-mention）─────────────────────────────────
// 自绘 Input 子；输入到 "@" 后过滤 Users 弹建议(复用 acDropdown)，
// 点用户把 "@用户 " 插入到 @ 位置。

type Mention struct {
	StatelessWidget
	Users       []string
	Placeholder string
	OnChange    func(string)
	Width       float64
}

func NewMention(users ...string) *Mention {
	return &Mention{Users: users, Width: 320}
}
func (m *Mention) WithOnChange(fn func(string)) *Mention { m.OnChange = fn; return m }

func (m *Mention) CreateElement() Element {
	return &MentionElement{BaseElement: BaseElement{widget: m}, m: m, atPos: -1}
}

type MentionElement struct {
	BaseElement
	m         *Mention
	input     *Input
	inputEl   Element
	overlayID int
	open      bool
	selecting bool
	atPos     int
}

func (e *MentionElement) Build() []Element {
	if e.inputEl == nil {
		e.input = NewInput(orI18n(e.m.Placeholder, "el.mention.placeholder"), nil)
		e.input.OnTextChanged = func(text string) { e.onInput(text) }
		e.inputEl = CreateElementFor(e.input)
		e.inputEl.Mount(e, 0)
	}
	e.children = []Element{e.inputEl}
	return e.children
}

func (e *MentionElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := e.m.Width
	if w <= 0 {
		w = 320
	}
	e.inputEl.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: w, MaxWidth: w, MinHeight: 32, MaxHeight: 32}})
	e.inputEl.SetPosition(types.Point{})
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: 32})
	return layout.LayoutResult{Size: e.size}
}

func (e *MentionElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.inputEl != nil {
		e.inputEl.Paint(cvs, offset)
	}
}

func (e *MentionElement) onInput(text string) {
	if e.selecting {
		return
	}
	if e.m.OnChange != nil {
		e.m.OnChange(text)
	}
	at := strings.LastIndex(text, "@")
	if at < 0 {
		e.hide()
		e.atPos = -1
		return
	}
	query := strings.ToLower(text[at+1:])
	var matches []string
	for _, u := range e.m.Users {
		if strings.Contains(strings.ToLower(u), query) {
			matches = append(matches, u)
		}
	}
	if len(matches) > 0 {
		e.atPos = at
		e.showUsers(matches)
	} else {
		e.hide()
	}
}

func (e *MentionElement) showUsers(matches []string) {
	if e.open {
		HideOverlay(e.overlayID)
		e.open = false
	}
	off := e.Offset()
	dd := &acDropdown{matches: matches, width: e.size.Width, onSelect: func(user string) {
		e.selecting = true
		if ie, ok := e.inputEl.(*InputElement); ok {
			cur := ie.Text()
			pos := e.atPos
			if pos < 0 || pos > len(cur) {
				pos = len(cur)
			}
			ie.SetText(cur[:pos] + "@" + user + " ")
		}
		e.selecting = false
		e.hide()
		e.MarkNeedsPaint()
	}}
	e.overlayID = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: e.size.Width, Height: e.size.Height},
		Content:        dd,
		Placement:      PlacementBottomStart,
		Gap:            4,
		Transition:     "fade",
		OnClickOutside: func() { e.hide() },
	})
	e.open = true
}

func (e *MentionElement) hide() {
	if e.open {
		HideOverlay(e.overlayID)
		e.open = false
	}
}

func (e *MentionElement) HandleEvent(ev event.Event) bool { return false }

func (e *MentionElement) Update(newWidget Widget) {
	if nm, ok := newWidget.(*Mention); ok {
		e.m = nm
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
