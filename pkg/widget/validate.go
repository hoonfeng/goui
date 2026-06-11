package widget

import (
	"fmt"
	"math"
)

// ─────────────────────────────────────────────────────────────
// 全局验证辅助函数
// ─────────────────────────────────────────────────────────────

// errf 创建带格式的验证错误
func errf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// validateChildCount 验证子 Element 数量是否与预期一致
func validateChildCount(el Element, expected int) []error {
	var errs []error
	children := el.Children()
	if len(children) != expected {
		errs = append(errs, errf(
			"%T.Children() 数量为 %d，预期 %d",
			el, len(children), expected,
		))
	}
	return errs
}

// validateParentPtr 验证所有子 Element 的 Parent() 是否指向自身
func validateParentPtr(el Element) []error {
	var errs []error
	for _, child := range el.Children() {
		if child.Parent() != el {
			errs = append(errs, errf(
				"%T 的子 Element (%T) 的 Parent() 不是自身",
				el, child,
			))
		}
	}
	return errs
}

// validateSizeSane 验证尺寸是否合理（非负、非 INF、非 NaN）
func validateSizeSane(el Element) []error {
	var errs []error
	s := el.Size()
	if s.Width < 0 || s.Height < 0 {
		errs = append(errs, errf("%T 尺寸为负数: %v", el, s))
	}
	if math.IsInf(s.Width, 0) || math.IsInf(s.Height, 0) {
		errs = append(errs, errf("%T 尺寸为 INF: %v", el, s))
	}
	if math.IsNaN(s.Width) || math.IsNaN(s.Height) {
		errs = append(errs, errf("%T 尺寸为 NaN: %v", el, s))
	}
	return errs
}

// validateSingleChild 验证单子控件的子 Element 一致性
// 当 widgetChild != nil 时，子 Element 应恰好有 1 个且非 nil
// 当 widgetChild == nil 时，子 Element 应恰好为 0 个
func validateSingleChild(el Element, widgetChild Widget, runtimeChild Element) []error {
	var errs []error
	if widgetChild != nil {
		if runtimeChild == nil {
			errs = append(errs, errf("%T.Child 不为 nil，但 runtime child 为 nil", el))
		}
		children := el.Children()
		if len(children) != 1 {
			errs = append(errs, errf("%T.Child 不为 nil，但子 Element 数量为 %d", el, len(children)))
		} else if children[0] != runtimeChild {
			errs = append(errs, errf("%T.runtimeChild 不在 Children() 中", el))
		}
	} else {
		if runtimeChild != nil {
			errs = append(errs, errf("%T.Child 为 nil，但 runtime child 不为 nil", el))
		}
		if len(el.Children()) != 0 {
			errs = append(errs, errf("%T.Child 为 nil，但子 Element 数量为 %d", el, len(el.Children())))
		}
	}
	return errs
}

// validateMultiChild 验证多子控件的子 Element 数量一致性
func validateMultiChild(el Element, widgetChildren []Widget) []error {
	var errs []error
	children := el.Children()
	if len(children) != len(widgetChildren) {
		errs = append(errs, errf(
			"%T 子 Element 数量为 %d，但 Widget.Children 为 %d",
			el, len(children), len(widgetChildren),
		))
	}
	return errs
}

// ─────────────────────────────────────────────────────────────
// BaseElement 的 Validate（默认实现）
// ─────────────────────────────────────────────────────────────

func (e *BaseElement) Validate() []error {
	var errs []error
	// 验证 Widget 不为 nil
	if e.widget == nil {
		errs = append(errs, errf("BaseElement.widget 为 nil"))
	}
	// 验证尺寸合理性
	errs = append(errs, validateSizeSane(e)...)
	// 注意：parent 指针验证不放在 BaseLevel，因为 BaseElement.Validate()
	// 的 receiver 是 *BaseElement 而非具体的 Element 类型（如 *FlexElement），
	// 而子 Element 的 Parent() 存储的是具体 Element 类型。
	// Go 接口比较要求动态类型和值都相等，因此 *BaseElement 与具体类型不相等
	// （即使指向同一块内存）。parent 指针验证放在各具体 Element 的 Validate() 中。
	return errs
}

// ─────────────────────────────────────────────────────────────
// StatelessElement / StatefulElement
// ─────────────────────────────────────────────────────────────

func (e *StatelessElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	// 检查 parent 指针（使用具体类型 e）
	errs = append(errs, validateParentPtr(e)...)

	// 检查 StatelessElement 是否有 builder
	if builder, ok := e.widget.(interface{ Build(BuildContext) Widget }); ok {
		// 有 builder，理论上应该在 Build 后被调用
		_ = builder // 避免 unused warning
		// 验证 builder 的返回值：如果 built，children 应为 0 或 1
		if e.built && len(e.children) > 1 {
			errs = append(errs, errf("StatelessElement(builder) 有 %d 个子 Element，预期 0 或 1", len(e.children)))
		}
	}

	return errs
}

func (e *StatefulElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	// 检查 parent 指针（使用具体类型 e）
	errs = append(errs, validateParentPtr(e)...)
	if e.state == nil {
		errs = append(errs, errf("StatefulElement.state 为 nil"))
	} else {
		// 验证 state 的元素指针一致性
		if se, ok := e.state.(interface{ GetElement() *StatefulElement }); ok {
			if se.GetElement() != e {
				errs = append(errs, errf("StatefulElement.state.element 不指向自身"))
			}
		}
	}
	// 验证单子模式：StatefulElement 要么 0 个子 Element，要么 1 个
	if len(e.children) > 1 {
		errs = append(errs, errf("StatefulElement 有 %d 个子 Element，预期 0 或 1", len(e.children)))
	}
	if e.child != nil {
		if len(e.children) != 1 || e.children[0] != e.child {
			errs = append(errs, errf("StatefulElement.child 与 Children() 不一致"))
		}
	}
	return errs
}

// ─────────────────────────────────────────────────────────────
// ContainerElement
// ─────────────────────────────────────────────────────────────

func (e *ContainerElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.container == nil {
		errs = append(errs, errf("ContainerElement.container 为 nil"))
		return errs
	}
	errs = append(errs, validateSingleChild(e, e.container.Child, e.child)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// ButtonElement
// ─────────────────────────────────────────────────────────────

func (e *ButtonElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.button == nil {
		errs = append(errs, errf("ButtonElement.button 为 nil"))
		return errs
	}
	errs = append(errs, validateSingleChild(e, e.button.Child, e.child)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// CardElement
// ─────────────────────────────────────────────────────────────

func (e *CardElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.card == nil {
		errs = append(errs, errf("CardElement.card 为 nil"))
		return errs
	}
	errs = append(errs, validateSingleChild(e, e.card.Child, e.child)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// ScrollViewElement
// ─────────────────────────────────────────────────────────────

func (e *ScrollViewElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.scrollView == nil {
		errs = append(errs, errf("ScrollViewElement.scrollView 为 nil"))
		return errs
	}
	errs = append(errs, validateSingleChild(e, e.scrollView.Child, e.child)...)
	// 验证 maxScroll 合理性
	if e.maxScroll.X < 0 || e.maxScroll.Y < 0 {
		errs = append(errs, errf("ScrollViewElement.maxScroll 为负数: %v", e.maxScroll))
	}
	return errs
}

// ─────────────────────────────────────────────────────────────
// FlexElement
// ─────────────────────────────────────────────────────────────

func (e *FlexElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.flex == nil {
		errs = append(errs, errf("FlexElement.flex 为 nil"))
		return errs
	}
	errs = append(errs, validateMultiChild(e, e.flex.Children)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// TextElement
// ─────────────────────────────────────────────────────────────

func (e *TextElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.text == nil {
		errs = append(errs, errf("TextElement.text 为 nil"))
		return errs
	}
	// Text 无子控件
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// CheckboxElement
// ─────────────────────────────────────────────────────────────

func (e *CheckboxElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.checkbox == nil {
		errs = append(errs, errf("CheckboxElement.checkbox 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// SwitchElement
// ─────────────────────────────────────────────────────────────

func (e *SwitchElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.sw == nil {
		errs = append(errs, errf("SwitchElement.sw 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// RadioButtonElement
// ─────────────────────────────────────────────────────────────

func (e *RadioButtonElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.radio == nil {
		errs = append(errs, errf("RadioButtonElement.radio 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// SliderElement
// ─────────────────────────────────────────────────────────────

func (e *SliderElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.slider == nil {
		errs = append(errs, errf("SliderElement.slider 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	// 验证最小值 <= 最大值
	if e.slider.Min > e.slider.Max {
		errs = append(errs, errf("Slider.Min (%.2f) > Max (%.2f)", e.slider.Min, e.slider.Max))
	}
	// 验证 Value 在范围内
	if e.slider.Value < e.slider.Min || e.slider.Value > e.slider.Max {
		errs = append(errs, errf("Slider.Value (%.2f) 超出范围 [%.2f, %.2f]", e.slider.Value, e.slider.Min, e.slider.Max))
	}
	return errs
}

// ─────────────────────────────────────────────────────────────
// ProgressBarElement
// ─────────────────────────────────────────────────────────────

func (e *ProgressBarElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.bar == nil {
		errs = append(errs, errf("ProgressBarElement.bar 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	// 验证 Value 在 [0,1] 范围内
	if !e.bar.Indeterminate && (e.bar.Value < 0 || e.bar.Value > 1) {
		errs = append(errs, errf("ProgressBar.Value (%.2f) 超出范围 [0, 1]", e.bar.Value))
	}
	return errs
}

// ─────────────────────────────────────────────────────────────
// InputElement
// ─────────────────────────────────────────────────────────────

func (e *InputElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.input == nil {
		errs = append(errs, errf("InputElement.input 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// IconElement
// ─────────────────────────────────────────────────────────────

func (e *IconElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.icon == nil {
		errs = append(errs, errf("IconElement.icon 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// SpacerElement
// ─────────────────────────────────────────────────────────────

func (e *SpacerElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// DividerElement
// ─────────────────────────────────────────────────────────────

func (e *DividerElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.divider == nil {
		errs = append(errs, errf("DividerElement.divider 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// ImageElement
// ─────────────────────────────────────────────────────────────

func (e *ImageElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.image == nil {
		errs = append(errs, errf("ImageElement.image 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// TreeViewElement
// ─────────────────────────────────────────────────────────────

func (e *TreeViewElement) Validate() []error {
	var errs []error
	errs = append(errs, e.BaseElement.Validate()...)
	errs = append(errs, validateParentPtr(e)...)
	if e.treeView == nil {
		errs = append(errs, errf("TreeViewElement.treeView 为 nil"))
		return errs
	}
	errs = append(errs, validateChildCount(e, 0)...)
	return errs
}

// ─────────────────────────────────────────────────────────────
// 递归验证元素树
// ─────────────────────────────────────────────────────────────

// ValidateElementTree 递归遍历以 root 为根的 Element 子树，
// 对每个实现了 Verifiable 的 Element 调用 Validate() 并收集所有错误。
// 同时递归验证子 Element。
// 返回聚合后的错误列表，每个错误附带元素树路径信息。
func ValidateElementTree(root Element) []error {
	var allErrs []error
	validateElementTreeRecursive(root, "", &allErrs)
	return allErrs
}

func validateElementTreeRecursive(el Element, path string, allErrs *[]error) {
	// 构建路径
	currentPath := fmt.Sprintf("%T", el)
	if path != "" {
		currentPath = path + " > " + currentPath
	}

	// 如果实现了 Verifiable，调用 Validate()
	if v, ok := el.(Verifiable); ok {
		if errs := v.Validate(); len(errs) > 0 {
			for _, err := range errs {
				*allErrs = append(*allErrs, errf("[%s] %v", currentPath, err))
			}
		}
	}

	// 递归验证子 Element
	for _, child := range el.Children() {
		if child != nil {
			validateElementTreeRecursive(child, currentPath, allErrs)
		}
	}
}
