package widget

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Form 表单 + 校验（1:1 复刻 el-form 核心）────────────────────
// batch④ 唯一非浮层组件：FormItem(标签 + 复用 Input 字段 + 错误提示) + 校验规则。
// 值变实时校验 / Form.Validate() 提交时全量校验；失败→输入框红边框 + 下方红字。

// FormRule 一条校验规则。
type FormRule struct {
	Required  bool                // 必填
	Pattern   string              // 正则(非空时校验)
	MinLen    int                 // 最小长度(0=不限)
	MaxLen    int                 // 最大长度(0=不限)
	Message   string              // 自定义错误信息(覆盖默认)
	Validator func(string) string // 自定义校验，返回错误信息(空=通过)
}

// FormItem 表单项。
type FormItem struct {
	StatelessWidget
	Label       string
	Placeholder string
	Prop        string   // 字段标识(联动/取值/单字段校验用)
	Rules       []FormRule
	Required    bool     // 快捷必填(标签前显示红 *)
	Triggers    []string // 本字段变化时，额外重新校验这些 Prop 的字段(规则联动)
	Field       Widget   // 可选自定义控件(替代默认 Input)；设了则纯布局、校验由用户自管
	labelWidth  float64  // 由 Form 填充
}

// Form 表单容器。
type Form struct {
	StatelessWidget
	Items      []*FormItem
	LabelWidth float64
	element    *FormElement // 运行时回填，供 Validate() 调用
}

// NewForm 创建表单。
func NewForm(items ...*FormItem) *Form {
	return &Form{Items: items, LabelWidth: 90}
}

// WithLabelWidth 设置标签列宽。
func (f *Form) WithLabelWidth(w float64) *Form { f.LabelWidth = w; return f }

// Validate 全量校验所有项，返回是否全部通过(供提交按钮调用)。
func (f *Form) Validate() bool {
	if f.element != nil {
		return f.element.validateAll()
	}
	return false
}

// ValidateField 校验单个字段(按 Prop)，返回是否通过。
func (f *Form) ValidateField(prop string) bool {
	if f.element != nil {
		return f.element.validateField(prop)
	}
	return false
}

// FieldValue 返回某字段(按 Prop)的当前值——供联动 Validator 读取其他字段。
func (f *Form) FieldValue(prop string) string {
	if f.element != nil {
		return f.element.fieldValue(prop)
	}
	return ""
}

// AddItem 动态追加一个表单项并重新布局。
func (f *Form) AddItem(item *FormItem) {
	f.Items = append(f.Items, item)
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	}
}

// RemoveItem 动态移除某字段(按 Prop)并重新布局。
func (f *Form) RemoveItem(prop string) {
	out := f.Items[:0]
	for _, it := range f.Items {
		if it.Prop != prop {
			out = append(out, it)
		}
	}
	f.Items = out
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	}
}

func (f *Form) CreateElement() Element {
	return &FormElement{BaseElement: BaseElement{widget: f}, form: f}
}

// FormElement 表单运行时 Element。
type FormElement struct {
	BaseElement
	form    *Form
	itemEls []Element
}

func (e *FormElement) Build() []Element {
	e.form.element = e
	// 按 Prop 复用旧 element(支持动态增删 + 保留输入状态)
	oldByKey := map[string]Element{}
	for i, el := range e.itemEls {
		if el == nil {
			continue
		}
		key := fmt.Sprintf("#%d", i)
		if fi, ok := el.(*FormItemElement); ok && fi.item.Prop != "" {
			key = fi.item.Prop
		}
		oldByKey[key] = el
	}
	used := map[string]bool{}
	newEls := make([]Element, len(e.form.Items))
	for i, item := range e.form.Items {
		item.labelWidth = e.form.LabelWidth
		key := item.Prop
		if key == "" {
			key = fmt.Sprintf("#%d", i)
		}
		if old, ok := oldByKey[key]; ok && !used[key] {
			old.Update(item)
			newEls[i] = old
			used[key] = true
		} else {
			el := CreateElementFor(item)
			el.Mount(e, i)
			if fi, ok := el.(*FormItemElement); ok {
				fi.formEl = e
			}
			newEls[i] = el
		}
	}
	// 卸载未复用的旧 element
	for key, el := range oldByKey {
		if !used[key] {
			el.Unmount()
		}
	}
	e.itemEls = newEls
	e.children = newEls
	return e.children
}

func (e *FormElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := ctx.Constraints.MaxWidth
	if w >= float64(1<<30) {
		w = 400
	}
	y := 0.0
	for i, it := range e.itemEls {
		if i > 0 {
			y += 16 // 项间距
		}
		res := it.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: w, MaxHeight: float64(1 << 30)}})
		it.SetPosition(types.Point{X: 0, Y: y})
		y += res.Size.Height
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: y})
	return layout.LayoutResult{Size: e.size}
}

func (e *FormElement) Paint(cvs canvas.Canvas, offset types.Point) {
	for _, it := range e.itemEls {
		it.Paint(cvs, offset)
	}
}

func (e *FormElement) HandleEvent(ev event.Event) bool { return false }

// validateAll 全量校验(提交)；标记所有项已触碰，relayout 让错误行增减生效。
func (e *FormElement) validateAll() bool {
	ok := true
	for _, it := range e.itemEls {
		if fi, is := it.(*FormItemElement); is {
			fi.touched = true
			if !fi.validate() {
				ok = false
			}
		}
	}
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	}
	return ok
}

// validateField 校验单个字段(按 Prop)。
func (e *FormElement) validateField(prop string) bool {
	for _, el := range e.itemEls {
		if fi, ok := el.(*FormItemElement); ok && fi.item.Prop == prop {
			fi.touched = true
			res := fi.validate()
			if OnNeedsLayout != nil {
				OnNeedsLayout()
			}
			return res
		}
	}
	return true
}

// revalidate 重新校验某字段(联动用)，返回错误显隐是否变化。
func (e *FormElement) revalidate(prop string) bool {
	for _, el := range e.itemEls {
		if fi, ok := el.(*FormItemElement); ok && fi.item.Prop == prop {
			had := fi.errorMsg != ""
			fi.touched = true
			fi.validate()
			return (fi.errorMsg != "") != had
		}
	}
	return false
}

// fieldValue 返回某字段(按 Prop)的当前值。
func (e *FormElement) fieldValue(prop string) string {
	for _, el := range e.itemEls {
		if fi, ok := el.(*FormItemElement); ok && fi.item.Prop == prop {
			return fi.value
		}
	}
	return ""
}

func (e *FormElement) Update(newWidget Widget) {
	if nf, ok := newWidget.(*Form); ok {
		e.form = nf
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── FormItemElement ───────────────────────────────────────────

func (item *FormItem) CreateElement() Element {
	return &FormItemElement{BaseElement: BaseElement{widget: item}, item: item}
}

type FormItemElement struct {
	BaseElement
	item     *FormItem
	formEl   *FormElement // 回指，供规则联动
	inputEl  Element
	input    *Input
	value    string
	errorMsg string
	touched  bool
	inputH   float64
}

func (e *FormItemElement) Build() []Element {
	if e.inputEl == nil {
		if e.item.Field != nil {
			// 自定义控件(Select/Switch/任意)：纯布局，校验由用户自管
			e.inputEl = CreateElementFor(e.item.Field)
		} else {
			e.input = NewInput(e.item.Placeholder, nil)
			e.input.OnTextChanged = func(v string) {
				e.value = v
				e.touched = true
				hadErr := e.errorMsg != ""
				e.validate()
				changed := (e.errorMsg != "") != hadErr
				// 规则联动：本字段变化 → 重新校验依赖它的字段(如改密码触发确认密码重校验)
				if e.formEl != nil {
					for _, prop := range e.item.Triggers {
						if e.formEl.revalidate(prop) {
							changed = true
						}
					}
				}
				// 错误行增减→高度变→relayout；否则只重绘
				if changed {
					if OnNeedsLayout != nil {
						OnNeedsLayout()
					}
				} else if OnNeedsRepaint != nil {
					OnNeedsRepaint()
				}
			}
			e.inputEl = CreateElementFor(e.input)
		}
		e.inputEl.Mount(e, 0)
	}
	e.children = []Element{e.inputEl}
	return e.children
}

func (e *FormItemElement) isRequired() bool {
	if e.item.Required {
		return true
	}
	for _, r := range e.item.Rules {
		if r.Required {
			return true
		}
	}
	return false
}

// validate 计算错误信息 + 设输入框边框色，返回是否通过。
func (e *FormItemElement) validate() bool {
	if e.input == nil {
		return true // 自定义控件：校验由用户自管
	}
	rules := e.item.Rules
	if e.item.Required {
		rules = append([]FormRule{{Required: true}}, rules...)
	}
	e.errorMsg = validateValue(e.value, e.item.Label, rules)
	if e.input != nil {
		if e.errorMsg != "" {
			red := elDanger()
			e.input.BorderColor = red
			e.input.FocusBorderColor = red
		} else {
			t := CurrentTheme()
			e.input.BorderColor = t.Input.BorderColor
			e.input.FocusBorderColor = t.Input.FocusBorderColor
		}
	}
	return e.errorMsg == ""
}

func (e *FormItemElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := ctx.Constraints.MaxWidth
	lw := e.item.labelWidth
	if lw <= 0 {
		lw = 80
	}
	inputW := w - lw
	if inputW < 60 {
		inputW = 60
	}
	res := e.inputEl.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: inputW, MaxHeight: 100}})
	e.inputH = res.Size.Height
	e.inputEl.SetPosition(types.Point{X: lw, Y: 0})
	h := e.inputH
	if e.errorMsg != "" {
		h += 18 // 错误行
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *FormItemElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	lw := e.item.labelWidth
	if lw <= 0 {
		lw = 80
	}
	lf := canvas.DefaultFont()
	lf.Size = 14
	label := e.item.Label

	// 标签(右对齐到字段左侧)
	labelRight := lw - 12
	canvas.DrawTextAligned(cvs, label,
		types.Rect{X: pos.X, Y: pos.Y, Width: labelRight, Height: e.inputH},
		lf, elTextRegular(), canvas.HAlignRight, canvas.VAlignMiddle)

	// 必填红 *(标签文字左侧)
	if e.isRequired() {
		labelW := canvas.MeasureTextGlobal(label, lf).Width
		starX := pos.X + labelRight - labelW - 9
		canvas.DrawTextAligned(cvs, "*",
			types.Rect{X: starX, Y: pos.Y, Width: 8, Height: e.inputH},
			lf, elDanger(), canvas.HAlignCenter, canvas.VAlignMiddle)
	}

	// 字段(Input 子元素)
	if e.inputEl != nil {
		e.inputEl.Paint(cvs, offset)
	}

	// 错误提示(字段下方红字)
	if e.errorMsg != "" {
		ef := canvas.DefaultFont()
		ef.Size = 12
		canvas.DrawTextAligned(cvs, e.errorMsg,
			types.Rect{X: pos.X + lw, Y: pos.Y + e.inputH + 1, Width: e.size.Width - lw, Height: 16},
			ef, elDanger(), canvas.HAlignLeft, canvas.VAlignMiddle)
	}
}

func (e *FormItemElement) HandleEvent(ev event.Event) bool { return false }

func (e *FormItemElement) Update(newWidget Widget) {
	if ni, ok := newWidget.(*FormItem); ok {
		e.item = ni
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// validateValue 按规则校验值，返回首条错误信息(空=通过)。
func validateValue(value, label string, rules []FormRule) string {
	v := strings.TrimSpace(value)
	// 必填先判(纯 required 规则用其 Message；混合规则的 Message 留给 pattern/长度，required 空报默认)
	for _, r := range rules {
		if r.Required && v == "" {
			if r.Message != "" && r.Pattern == "" && r.MinLen == 0 && r.MaxLen == 0 && r.Validator == nil {
				return r.Message
			}
			return label + "不能为空"
		}
	}
	if v == "" {
		return "" // 非必填且空 → 通过
	}
	runes := []rune(v)
	for _, r := range rules {
		if r.MinLen > 0 && len(runes) < r.MinLen {
			if r.Message != "" {
				return r.Message
			}
			return fmt.Sprintf("%s至少 %d 个字符", label, r.MinLen)
		}
		if r.MaxLen > 0 && len(runes) > r.MaxLen {
			if r.Message != "" {
				return r.Message
			}
			return fmt.Sprintf("%s最多 %d 个字符", label, r.MaxLen)
		}
		if r.Pattern != "" {
			if ok, _ := regexp.MatchString(r.Pattern, v); !ok {
				if r.Message != "" {
					return r.Message
				}
				return label + "格式不正确"
			}
		}
		if r.Validator != nil {
			if msg := r.Validator(v); msg != "" {
				return msg
			}
		}
	}
	return ""
}
