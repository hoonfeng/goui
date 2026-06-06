package widget

import (
	"strings"
	"sync"
	"time"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Autocomplete 自动完成（el-autocomplete）────────────────────
// 自绘：Input 子 + 输入时弹建议浮层；点建议回填 Input。
// 两种数据源：
//   • 本地：Suggestions 静态列表，输入即 Contains 过滤。
//   • 异步：Fetch(query, done) 远程拉取——防抖触发、loading 态、过期结果丢弃。
//     异步安全：overlay 仅主线程开关；fetch 结果经 mutex 写入，再 OnNeedsLayout
//     设脏，主线程下一帧(Input 聚焦时 16ms tick)读最新状态重绘。

type Autocomplete struct {
	StatelessWidget
	Suggestions []string
	Placeholder string
	OnChange    func(string)
	Fetch       func(query string, done func([]string)) // 异步远程搜索；设了则忽略 Suggestions
	DebounceMs  int                                      // 防抖毫秒，默认 300
	Width       float64
}

func NewAutocomplete(suggestions ...string) *Autocomplete {
	return &Autocomplete{Suggestions: suggestions, Width: 240}
}
func (a *Autocomplete) WithOnChange(fn func(string)) *Autocomplete { a.OnChange = fn; return a }
func (a *Autocomplete) WithFetch(fn func(query string, done func([]string))) *Autocomplete {
	a.Fetch = fn
	return a
}
func (a *Autocomplete) WithDebounce(ms int) *Autocomplete { a.DebounceMs = ms; return a }

func (a *Autocomplete) CreateElement() Element {
	return &AutocompleteElement{BaseElement: BaseElement{widget: a}, ac: a}
}

type AutocompleteElement struct {
	BaseElement
	ac        *Autocomplete
	input     *Input
	inputEl   Element
	overlayID int
	open      bool
	selecting bool

	mu       sync.Mutex
	matches  []string
	loading  bool
	queryID  int
	debounce *time.Timer
}

func (e *AutocompleteElement) Build() []Element {
	if e.inputEl == nil {
		e.input = NewInput(orI18n(e.ac.Placeholder, "el.autocomplete.placeholder"), nil)
		e.input.OnTextChanged = func(text string) { e.onInput(text) }
		e.inputEl = CreateElementFor(e.input)
		e.inputEl.Mount(e, 0)
	}
	e.children = []Element{e.inputEl}
	return e.children
}

func (e *AutocompleteElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := e.ac.Width
	if w <= 0 {
		w = 240
	}
	e.inputEl.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: w, MaxWidth: w, MinHeight: 32, MaxHeight: 32}})
	e.inputEl.SetPosition(types.Point{})
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: 32})
	return layout.LayoutResult{Size: e.size}
}

func (e *AutocompleteElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.inputEl != nil {
		e.inputEl.Paint(cvs, offset)
	}
}

func (e *AutocompleteElement) onInput(text string) {
	if e.selecting {
		return // 忽略点选建议时 SetText 触发的回调，防回环
	}
	if e.ac.OnChange != nil {
		e.ac.OnChange(text)
	}
	if e.ac.Fetch != nil {
		e.asyncInput(text)
		return
	}
	// 本地过滤
	var matches []string
	if text != "" {
		low := strings.ToLower(text)
		for _, s := range e.ac.Suggestions {
			if strings.Contains(strings.ToLower(s), low) {
				matches = append(matches, s)
			}
		}
	}
	if len(matches) > 0 {
		e.showLocal(matches)
	} else {
		e.hide()
	}
}

// ─── 异步分支 ───────────────────────────────────────────────────

func (e *AutocompleteElement) asyncInput(text string) {
	if text == "" {
		if e.debounce != nil {
			e.debounce.Stop()
		}
		e.hide()
		return
	}
	// 立即进入 loading（主线程）+ 弹出浮层（内容动态读共享状态）
	e.mu.Lock()
	e.queryID++
	id := e.queryID
	e.loading = true
	e.matches = nil
	e.mu.Unlock()
	e.ensureAsyncDropdown()
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	}
	// 防抖：停顿后才真正 fetch
	if e.debounce != nil {
		e.debounce.Stop()
	}
	ms := e.ac.DebounceMs
	if ms <= 0 {
		ms = 300
	}
	e.debounce = time.AfterFunc(time.Duration(ms)*time.Millisecond, func() {
		e.ac.Fetch(text, func(results []string) {
			e.mu.Lock()
			if id != e.queryID { // 过期结果丢弃
				e.mu.Unlock()
				return
			}
			e.matches = results
			e.loading = false
			e.mu.Unlock()
			if OnNeedsLayout != nil {
				OnNeedsLayout() // 设脏 → 主线程下一帧重绘下拉
			}
		})
	})
}

func (e *AutocompleteElement) ensureAsyncDropdown() {
	if e.open {
		return
	}
	off := e.Offset()
	dd := &acDropdown{
		width: e.size.Width,
		getState: func() (bool, []string) {
			e.mu.Lock()
			defer e.mu.Unlock()
			return e.loading, append([]string(nil), e.matches...)
		},
		onSelect: e.pick,
	}
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

// ─── 本地分支 ───────────────────────────────────────────────────

func (e *AutocompleteElement) showLocal(matches []string) {
	if e.open {
		HideOverlay(e.overlayID)
		e.open = false
	}
	off := e.Offset()
	dd := &acDropdown{matches: matches, width: e.size.Width, onSelect: e.pick}
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

func (e *AutocompleteElement) pick(s string) {
	e.selecting = true
	if ie, ok := e.inputEl.(*InputElement); ok {
		ie.SetText(s)
	}
	e.selecting = false
	e.hide()
	if e.ac.OnChange != nil {
		e.ac.OnChange(s)
	}
	e.MarkNeedsPaint()
}

func (e *AutocompleteElement) hide() {
	if e.open {
		HideOverlay(e.overlayID)
		e.open = false
	}
}

func (e *AutocompleteElement) Update(newWidget Widget) {
	if na, ok := newWidget.(*Autocomplete); ok {
		e.ac = na
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── acDropdown 建议浮层（本地 matches 或异步 getState）──────────

type acDropdown struct {
	StatelessWidget
	matches  []string
	loading  bool
	width    float64
	getState func() (bool, []string) // 异步：返回 (loading, matches)；nil=用 matches/loading 字段
	onSelect func(string)
}

func (d *acDropdown) CreateElement() Element {
	return &acDropdownElement{BaseElement: BaseElement{widget: d}, dd: d, hoverIdx: -1}
}

const acItemH = 32.0

type acDropdownElement struct {
	BaseElement
	dd       *acDropdown
	hoverIdx int
}

func (e *acDropdownElement) state() (bool, []string) {
	if e.dd.getState != nil {
		return e.dd.getState()
	}
	return e.dd.loading, e.dd.matches
}

func (e *acDropdownElement) Build() []Element { return nil }

func (e *acDropdownElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	loading, matches := e.state()
	n := len(matches)
	if loading || n == 0 {
		n = 1 // loading / 空状态占一行
	}
	h := float64(n)*acItemH + 8
	e.size = ctx.Constraints.Constrain(types.Size{Width: e.dd.width, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *acDropdownElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height
	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 22)
	cvs.DrawRoundedRect(pos.X, pos.Y+2, W, H, 4, shadow)
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = elBorderLight()
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	font := canvas.DefaultFont()
	font.Size = 14
	loading, matches := e.state()

	if loading {
		e.paintSpinner(cvs, pos.X+18, pos.Y+acItemH/2+4)
		canvas.DrawTextAligned(cvs, "加载中…", types.Rect{X: pos.X + 32, Y: pos.Y + 4, Width: W - 40, Height: acItemH},
			font, elTextSecondary(), canvas.HAlignLeft, canvas.VAlignMiddle)
		return
	}
	if len(matches) == 0 {
		canvas.DrawTextAligned(cvs, orI18n("", "el.select.noMatch"), types.Rect{X: pos.X, Y: pos.Y + 4, Width: W, Height: acItemH},
			font, elPlaceholder(), canvas.HAlignCenter, canvas.VAlignMiddle)
		return
	}
	for i, s := range matches {
		iy := pos.Y + 4 + float64(i)*acItemH
		if i == e.hoverIdx {
			hp := paint.DefaultPaint()
			hp.Color = elFill()
			cvs.DrawRect(pos.X+1, iy, W-2, acItemH, hp)
		}
		canvas.DrawTextAligned(cvs, s, types.Rect{X: pos.X + 14, Y: iy, Width: W - 20, Height: acItemH},
			font, elTextRegular(), canvas.HAlignLeft, canvas.VAlignMiddle)
	}
}

// paintSpinner 画一个简易加载转圈（四段渐隐弧点）。
func (e *acDropdownElement) paintSpinner(cvs canvas.Canvas, cx, cy float64) {
	pts := [4][2]float64{{0, -5}, {5, 0}, {0, 5}, {-5, 0}}
	alphas := [4]uint8{220, 150, 90, 40}
	for i, p := range pts {
		sp := paint.DefaultPaint()
		sp.Color = types.ColorFromRGBA(64, 158, 255, alphas[i])
		cvs.DrawCircle(cx+p[0], cy+p[1], 1.6, sp)
	}
}

func (e *acDropdownElement) itemAt(localY float64) int {
	_, matches := e.state()
	idx := int((localY - 4) / acItemH)
	if idx < 0 || idx >= len(matches) {
		return -1
	}
	return idx
}

func (e *acDropdownElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		e.hoverIdx = e.itemAt(me.Y - e.Offset().Y)
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hoverIdx = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		_, matches := e.state()
		if idx := e.itemAt(me.Y - e.Offset().Y); idx >= 0 && idx < len(matches) && e.dd.onSelect != nil {
			e.dd.onSelect(matches[idx])
		}
		return true
	}
	return false
}

func (e *acDropdownElement) Update(newWidget Widget) {
	if nd, ok := newWidget.(*acDropdown); ok {
		e.dd = nd
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// AutocompleteDropdownPreview 预览建议浮层外观（loading 或结果态），供 demo/验证。
func AutocompleteDropdownPreview(matches []string, loading bool, width float64) Widget {
	return &acDropdown{matches: matches, loading: loading, width: width}
}
