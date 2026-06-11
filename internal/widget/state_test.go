package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
)

// TestStateCheckbox 测试 Checkbox 的 Element 级状态管理。
// 覆盖 CreateElement、Build（返回 nil）、HandleEvent 驱动的状态变更、Update 更新。
func TestStateCheckbox(t *testing.T) {
	t.Run("CreateElement returns CheckboxElement", func(t *testing.T) {
		cb := DefaultCheckbox()
		el := cb.CreateElement()
		if _, ok := el.(*CheckboxElement); !ok {
			t.Fatalf("Expected *CheckboxElement, got %T", el)
		}
		if el.Widget() != cb {
			t.Error("Element.Widget() should return the original Checkbox")
		}
	})

	t.Run("Build returns nil (leaf widget)", func(t *testing.T) {
		cb := DefaultCheckbox()
		el := cb.CreateElement()
		result := el.Build()
		if result != nil {
			t.Errorf("CheckboxElement.Build() should return nil for leaf widget, got %v", result)
		}
	})

	t.Run("HandleEvent MouseUp toggles Checked from false to true", func(t *testing.T) {
		cb := DefaultCheckbox()
		el := cb.CreateElement().(*CheckboxElement)
		if cb.Checked {
			t.Fatal("DefaultCheckbox should have Checked=false initially")
		}
		// 模拟点击事件
		ev := event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
		handled := el.HandleEvent(ev)
		if !handled {
			t.Error("HandleEvent should return true for TypeMouseUp")
		}
		if !cb.Checked {
			t.Error("After MouseUp, Checkbox.Checked should be true")
		}
	})

	t.Run("HandleEvent MouseUp toggles Checked from true to false", func(t *testing.T) {
		cb := DefaultCheckbox()
		cb.Checked = true
		el := cb.CreateElement().(*CheckboxElement)
		ev := event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
		el.HandleEvent(ev)
		if cb.Checked {
			t.Error("After second MouseUp, Checkbox.Checked should be false (toggled)")
		}
	})

	t.Run("HandleEvent MouseEnter sets hovered", func(t *testing.T) {
		cb := DefaultCheckbox()
		el := cb.CreateElement().(*CheckboxElement)
		if el.hovered {
			t.Fatal("hovered should be false initially")
		}
		ev := event.NewMouseEvent(event.TypeMouseEnter, 0, 0, event.ButtonNone, event.ModNone)
		handled := el.HandleEvent(ev)
		if !handled {
			t.Error("HandleEvent should return true for TypeMouseEnter")
		}
		if !el.hovered {
			t.Error("hovered should be true after MouseEnter")
		}
	})

	t.Run("HandleEvent MouseLeave clears hovered", func(t *testing.T) {
		cb := DefaultCheckbox()
		el := cb.CreateElement().(*CheckboxElement)
		el.hovered = true
		ev := event.NewMouseEvent(event.TypeMouseLeave, 0, 0, event.ButtonNone, event.ModNone)
		handled := el.HandleEvent(ev)
		if !handled {
			t.Error("HandleEvent should return true for TypeMouseLeave")
		}
		if el.hovered {
			t.Error("hovered should be false after MouseLeave")
		}
	})

	t.Run("Update replaces widget reference", func(t *testing.T) {
		cb1 := DefaultCheckbox()
		cb1.Checked = false
		cb1.Label = "original"
		el := cb1.CreateElement().(*CheckboxElement)

		cb2 := DefaultCheckbox()
		cb2.Checked = true
		cb2.Label = "updated"
		el.Update(cb2)

		if el.checkbox != cb2 {
			t.Error("Element.checkbox should point to the new widget after Update")
		}
		if el.Widget() != cb2 {
			t.Error("Element.Widget() should return the new widget after Update")
		}
		if !el.dirty {
			t.Error("Element should be marked dirty after Update")
		}
	})

	t.Run("OnChanged callback fires on MouseUp", func(t *testing.T) {
		fired := false
		cb := DefaultCheckbox()
		cb.OnChanged = func(checked bool) {
			fired = true
			if !checked {
				t.Error("OnChanged should receive checked=true")
			}
		}
		el := cb.CreateElement().(*CheckboxElement)
		ev := event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
		el.HandleEvent(ev)
		if !fired {
			t.Error("OnChanged callback was not fired")
		}
	})
}

// TestStateSwitch 测试 Switch 的 Element 级状态管理。
func TestStateSwitch(t *testing.T) {
	t.Run("CreateElement returns SwitchElement", func(t *testing.T) {
		sw := DefaultSwitch()
		el := sw.CreateElement()
		if _, ok := el.(*SwitchElement); !ok {
			t.Fatalf("Expected *SwitchElement, got %T", el)
		}
		if el.Widget() != sw {
			t.Error("Element.Widget() should return the original Switch")
		}
	})

	t.Run("Build returns nil (leaf widget)", func(t *testing.T) {
		sw := DefaultSwitch()
		el := sw.CreateElement()
		result := el.Build()
		if result != nil {
			t.Errorf("SwitchElement.Build() should return nil for leaf widget, got %v", result)
		}
	})

	t.Run("HandleEvent MouseUp toggles Value", func(t *testing.T) {
		sw := DefaultSwitch()
		el := sw.CreateElement().(*SwitchElement)
		if sw.Value {
			t.Fatal("DefaultSwitch should have Value=false initially")
		}
		ev := event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
		handled := el.HandleEvent(ev)
		if !handled {
			t.Error("HandleEvent should return true for TypeMouseUp")
		}
		if !sw.Value {
			t.Error("After MouseUp, Switch.Value should be true")
		}
		// 第二次点击切换回 false
		el.HandleEvent(ev)
		if sw.Value {
			t.Error("After second MouseUp, Switch.Value should be false (toggled)")
		}
	})

	t.Run("HandleEvent MouseEnter sets hovered", func(t *testing.T) {
		sw := DefaultSwitch()
		el := sw.CreateElement().(*SwitchElement)
		ev := event.NewMouseEvent(event.TypeMouseEnter, 0, 0, event.ButtonNone, event.ModNone)
		el.HandleEvent(ev)
		if !el.hovered {
			t.Error("hovered should be true after MouseEnter")
		}
	})

	t.Run("HandleEvent MouseLeave clears hovered", func(t *testing.T) {
		sw := DefaultSwitch()
		el := sw.CreateElement().(*SwitchElement)
		el.hovered = true
		ev := event.NewMouseEvent(event.TypeMouseLeave, 0, 0, event.ButtonNone, event.ModNone)
		el.HandleEvent(ev)
		if el.hovered {
			t.Error("hovered should be false after MouseLeave")
		}
	})

	t.Run("Update replaces widget reference", func(t *testing.T) {
		sw1 := DefaultSwitch()
		sw1.Value = false
		el := sw1.CreateElement().(*SwitchElement)

		sw2 := DefaultSwitch()
		sw2.Value = true
		el.Update(sw2)

		if el.sw != sw2 {
			t.Error("Element.sw should point to the new widget after Update")
		}
		if el.Widget() != sw2 {
			t.Error("Element.Widget() should return the new widget after Update")
		}
		if !el.dirty {
			t.Error("Element should be marked dirty after Update")
		}
	})

	t.Run("OnChanged callback fires on MouseUp", func(t *testing.T) {
		fired := false
		sw := DefaultSwitch()
		sw.OnChanged = func(val bool) {
			fired = true
			if !val {
				t.Error("OnChanged should receive val=true")
			}
		}
		el := sw.CreateElement().(*SwitchElement)
		ev := event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
		el.HandleEvent(ev)
		if !fired {
			t.Error("OnChanged callback was not fired")
		}
	})
}

// TestStateRadioButton 测试 RadioButton 的 Element 级状态管理。
func TestStateRadioButton(t *testing.T) {
	t.Run("CreateElement returns RadioButtonElement", func(t *testing.T) {
		rb := DefaultRadioButton()
		el := rb.CreateElement()
		if _, ok := el.(*RadioButtonElement); !ok {
			t.Fatalf("Expected *RadioButtonElement, got %T", el)
		}
		if el.Widget() != rb {
			t.Error("Element.Widget() should return the original RadioButton")
		}
	})

	t.Run("Build returns nil (leaf widget)", func(t *testing.T) {
		rb := DefaultRadioButton()
		el := rb.CreateElement()
		result := el.Build()
		if result != nil {
			t.Errorf("RadioButtonElement.Build() should return nil for leaf widget, got %v", result)
		}
	})

	t.Run("HandleEvent MouseUp sets Selected=true", func(t *testing.T) {
		rb := DefaultRadioButton()
		el := rb.CreateElement().(*RadioButtonElement)
		if rb.Selected {
			t.Fatal("DefaultRadioButton should have Selected=false initially")
		}
		ev := event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
		handled := el.HandleEvent(ev)
		if !handled {
			t.Error("HandleEvent should return true for TypeMouseUp")
		}
		if !rb.Selected {
			t.Error("After MouseUp, RadioButton.Selected should be true")
		}
	})

	t.Run("HandleEvent MouseUp on already selected does nothing", func(t *testing.T) {
		rb := DefaultRadioButton()
		rb.Selected = true
		el := rb.CreateElement().(*RadioButtonElement)
		ev := event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
		el.HandleEvent(ev)
		if !rb.Selected {
			t.Error("RadioButton.Selected should remain true when clicked again")
		}
	})

	t.Run("Update replaces widget reference", func(t *testing.T) {
		rb1 := DefaultRadioButton()
		rb1.Selected = false
		el := rb1.CreateElement().(*RadioButtonElement)

		rb2 := DefaultRadioButton()
		rb2.Selected = true
		el.Update(rb2)

		if el.radio != rb2 {
			t.Error("Element.radio should point to the new widget after Update")
		}
		if !el.dirty {
			t.Error("Element should be marked dirty after Update")
		}
	})

	t.Run("OnChanged callback fires on MouseUp", func(t *testing.T) {
		fired := false
		rb := DefaultRadioButton()
		rb.OnChanged = func() {
			fired = true
		}
		el := rb.CreateElement().(*RadioButtonElement)
		ev := event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
		el.HandleEvent(ev)
		if !fired {
			t.Error("OnChanged callback was not fired")
		}
	})
}

// TestStateSlider 测试 Slider 的 Element 级状态管理。
func TestStateSlider(t *testing.T) {
	t.Run("CreateElement returns SliderElement", func(t *testing.T) {
		s := DefaultSlider()
		el := s.CreateElement()
		if _, ok := el.(*SliderElement); !ok {
			t.Fatalf("Expected *SliderElement, got %T", el)
		}
		if el.Widget() != s {
			t.Error("Element.Widget() should return the original Slider")
		}
	})

	t.Run("Build returns nil (leaf widget)", func(t *testing.T) {
		s := DefaultSlider()
		el := s.CreateElement()
		result := el.Build()
		if result != nil {
			t.Errorf("SliderElement.Build() should return nil for leaf widget, got %v", result)
		}
	})

	t.Run("HandleEvent MouseEnter sets hovered", func(t *testing.T) {
		s := DefaultSlider()
		el := s.CreateElement().(*SliderElement)
		ev := event.NewMouseEvent(event.TypeMouseEnter, 0, 0, event.ButtonNone, event.ModNone)
		el.HandleEvent(ev)
		if !el.hovered {
			t.Error("hovered should be true after MouseEnter")
		}
	})

	t.Run("HandleEvent MouseLeave clears hovered but keeps dragging", func(t *testing.T) {
		s := DefaultSlider()
		el := s.CreateElement().(*SliderElement)
		el.hovered = true
		el.dragging = true
		ev := event.NewMouseEvent(event.TypeMouseLeave, 0, 0, event.ButtonNone, event.ModNone)
		el.HandleEvent(ev)
		if el.hovered {
			t.Error("hovered should be false after MouseLeave")
		}
		// 拖拽中鼠标移出滑块不应停拖（有鼠标捕获，按住不放仍连续跟手）；由 MouseUp/DragEnd 才停。
		if !el.dragging {
			t.Error("dragging should stay true after MouseLeave (continuous drag)")
		}
	})

	t.Run("HandleEvent MouseDown sets dragging", func(t *testing.T) {
		s := DefaultSlider()
		el := s.CreateElement().(*SliderElement)
		ev := event.NewMouseEvent(event.TypeMouseDown, 50, 10, event.ButtonLeft, event.ModNone)
		handled := el.HandleEvent(ev)
		if !handled {
			t.Error("HandleEvent should return true for TypeMouseDown")
		}
		if !el.dragging {
			t.Error("dragging should be true after MouseDown")
		}
	})

	t.Run("HandleEvent MouseUp clears dragging", func(t *testing.T) {
		s := DefaultSlider()
		el := s.CreateElement().(*SliderElement)
		el.dragging = true
		ev := event.NewMouseEvent(event.TypeMouseUp, 0, 0, event.ButtonLeft, event.ModNone)
		handled := el.HandleEvent(ev)
		if !handled {
			t.Error("HandleEvent should return true for TypeMouseUp when dragging")
		}
		if el.dragging {
			t.Error("dragging should be false after MouseUp")
		}
	})

	t.Run("Update replaces widget reference", func(t *testing.T) {
		s1 := DefaultSlider()
		s1.Value = 25
		el := s1.CreateElement().(*SliderElement)

		s2 := DefaultSlider()
		s2.Value = 75
		el.Update(s2)

		if el.slider != s2 {
			t.Error("Element.slider should point to the new widget after Update")
		}
		if el.Widget() != s2 {
			t.Error("Element.Widget() should return the new widget after Update")
		}
		if !el.dirty {
			t.Error("Element should be marked dirty after Update")
		}
	})
}

// TestStateInput 测试 Input 的 Element 级状态管理。
func TestStateInput(t *testing.T) {
	t.Run("CreateElement returns InputElement", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement()
		if _, ok := el.(*InputElement); !ok {
			t.Fatalf("Expected *InputElement, got %T", el)
		}
		if el.Widget() != in {
			t.Error("Element.Widget() should return the original Input")
		}
	})

	t.Run("Build returns nil (leaf widget)", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement()
		result := el.Build()
		if result != nil {
			t.Errorf("InputElement.Build() should return nil for leaf widget, got %v", result)
		}
	})

	t.Run("Initial text is empty", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		if el.Text() != "" {
			t.Errorf("Initial text should be empty, got %q", el.Text())
		}
		if el.cursorPos != 0 {
			t.Errorf("Initial cursorPos should be 0, got %d", el.cursorPos)
		}
	})

	t.Run("SetText updates text content", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		el.SetText("hello")
		if el.Text() != "hello" {
			t.Errorf("Text should be 'hello', got %q", el.Text())
		}
	})

	t.Run("SetText adjusts cursor position when text shortens", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		el.SetText("hello world")
		el.cursorPos = 11
		el.SetText("hi")
		if el.cursorPos > 2 {
			t.Errorf("CursorPos should be clamped to text length (2), got %d", el.cursorPos)
		}
	})

	t.Run("Focus and cursor visible", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		el.Focus()
		if !el.IsFocused() {
			t.Error("el should be focused after Focus()")
		}
		if !el.isCursorVisible() {
			t.Error("cursor should be visible right after Focus()")
		}
		el.Blur()
		if el.IsFocused() {
			t.Error("el should not be focused after Blur()")
		}
		if el.isCursorVisible() {
			t.Error("cursor should not be visible after Blur()")
		}
	})

	t.Run("HandleEvent KeyChar appends character", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		el.Focus() // 设置为焦点

		keyEv := event.NewKeyEvent(event.TypeKeyChar, 0, "A", event.ModNone)
		keyEv.Char = 'A'
		el.HandleEvent(keyEv)
		if el.Text() != "A" {
			t.Errorf("Text should be 'A', got %q", el.Text())
		}
		if el.cursorPos != 1 {
			t.Errorf("cursorPos should be 1, got %d", el.cursorPos)
		}
	})

	t.Run("HandleEvent KeyChar respects MaxLength", func(t *testing.T) {
		in := DefaultInput()
		in.MaxLength = 3
		el := in.CreateElement().(*InputElement)
		el.Focus()

		for _, ch := range []rune{'a', 'b', 'c', 'd', 'e'} {
			keyEv := event.NewKeyEvent(event.TypeKeyChar, 0, string(ch), event.ModNone)
			keyEv.Char = ch
			el.HandleEvent(keyEv)
		}
		if el.Text() != "abc" {
			t.Errorf("Text should be truncated to 'abc' (MaxLength=3), got %q", el.Text())
		}
	})

	t.Run("HandleEvent Backspace deletes character before cursor", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		el.Focus()
		el.SetText("hello")
		el.cursorPos = 5

		backEv := event.NewKeyEvent(event.TypeKeyDown, 0, "Backspace", event.ModNone)
		el.HandleEvent(backEv)
		if el.Text() != "hell" {
			t.Errorf("Text should be 'hell' after backspace, got %q", el.Text())
		}
		if el.cursorPos != 4 {
			t.Errorf("cursorPos should be 4, got %d", el.cursorPos)
		}
	})

	t.Run("HandleEvent ArrowLeft/ArrowRight moves cursor", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		el.Focus()
		el.SetText("hello")
		el.cursorPos = 5

		leftEv := event.NewKeyEvent(event.TypeKeyDown, 0, "ArrowLeft", event.ModNone)
		el.HandleEvent(leftEv)
		if el.cursorPos != 4 {
			t.Errorf("cursorPos should be 4 after ArrowLeft, got %d", el.cursorPos)
		}

		rightEv := event.NewKeyEvent(event.TypeKeyDown, 0, "ArrowRight", event.ModNone)
		el.HandleEvent(rightEv)
		if el.cursorPos != 5 {
			t.Errorf("cursorPos should be 5 after ArrowRight, got %d", el.cursorPos)
		}
	})

	t.Run("HandleEvent Home/End moves cursor to start/end", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		el.Focus()
		el.SetText("hello")
		el.cursorPos = 2

		homeEv := event.NewKeyEvent(event.TypeKeyDown, 0, "Home", event.ModNone)
		el.HandleEvent(homeEv)
		if el.cursorPos != 0 {
			t.Errorf("cursorPos should be 0 after Home, got %d", el.cursorPos)
		}

		endEv := event.NewKeyEvent(event.TypeKeyDown, 0, "End", event.ModNone)
		el.HandleEvent(endEv)
		if el.cursorPos != 5 {
			t.Errorf("cursorPos should be 5 after End, got %d", el.cursorPos)
		}
	})

	t.Run("HandleEvent Delete removes character at cursor", func(t *testing.T) {
		in := DefaultInput()
		el := in.CreateElement().(*InputElement)
		el.Focus()
		el.SetText("hello")
		el.cursorPos = 0

		delEv := event.NewKeyEvent(event.TypeKeyDown, 0, "Delete", event.ModNone)
		el.HandleEvent(delEv)
		if el.Text() != "ello" {
			t.Errorf("Text should be 'ello' after Delete, got %q", el.Text())
		}
		if el.cursorPos != 0 {
			t.Errorf("cursorPos should remain 0, got %d", el.cursorPos)
		}
	})

	t.Run("HandleEvent Enter triggers OnSubmit and clears text", func(t *testing.T) {
		submitted := false
		var submittedText string
		in := DefaultInput()
		in.OnSubmit = func(text string) {
			submitted = true
			submittedText = text
		}
		el := in.CreateElement().(*InputElement)
		el.Focus()
		el.SetText("submit me")
		el.cursorPos = 9

		enterEv := event.NewKeyEvent(event.TypeKeyDown, 0, "Enter", event.ModNone)
		el.HandleEvent(enterEv)
		if !submitted {
			t.Error("OnSubmit callback was not fired")
		}
		if submittedText != "submit me" {
			t.Errorf("OnSubmit should receive 'submit me', got %q", submittedText)
		}
		if el.Text() != "" {
			t.Errorf("Text should be cleared after Enter, got %q", el.Text())
		}
		if el.cursorPos != 0 {
			t.Errorf("cursorPos should be 0 after Enter, got %d", el.cursorPos)
		}
	})

	t.Run("Update replaces widget reference and preserves text", func(t *testing.T) {
		in1 := DefaultInput()
		el := in1.CreateElement().(*InputElement)
		el.SetText("preserved")

		in2 := DefaultInput()
		in2.Placeholder = "new placeholder"
		el.Update(in2)

		if el.input != in2 {
			t.Error("Element.input should point to the new widget after Update")
		}
		if el.Widget() != in2 {
			t.Error("Element.Widget() should return the new widget after Update")
		}
		// 文本状态应该保留
		if el.Text() != "preserved" {
			t.Errorf("Text state should be preserved after Update, got %q", el.Text())
		}
		if !el.dirty {
			t.Error("Element should be marked dirty after Update")
		}
	})
}

// TestStateProgressBar 测试 ProgressBar 的 Element 级状态管理。
func TestStateProgressBar(t *testing.T) {
	t.Run("CreateElement returns ProgressBarElement", func(t *testing.T) {
		pb := DefaultProgressBar()
		el := pb.CreateElement()
		if _, ok := el.(*ProgressBarElement); !ok {
			t.Fatalf("Expected *ProgressBarElement, got %T", el)
		}
		if el.Widget() != pb {
			t.Error("Element.Widget() should return the original ProgressBar")
		}
	})

	t.Run("Build returns nil (leaf widget)", func(t *testing.T) {
		pb := DefaultProgressBar()
		el := pb.CreateElement()
		result := el.Build()
		if result != nil {
			t.Errorf("ProgressBarElement.Build() should return nil for leaf widget, got %v", result)
		}
	})

	t.Run("Default value is 0", func(t *testing.T) {
		pb := DefaultProgressBar()
		if pb.Value != 0 {
			t.Errorf("DefaultProgressBar.Value should be 0, got %f", pb.Value)
		}
	})

	t.Run("SetAnimOffset updates animation state", func(t *testing.T) {
		pb := DefaultProgressBar()
		pb.Indeterminate = true
		el := pb.CreateElement().(*ProgressBarElement)
		if el.animOffset != 0 {
			t.Fatal("Initial animOffset should be 0")
		}
		el.SetAnimOffset(0.5)
		if el.animOffset != 0.5 {
			t.Errorf("animOffset should be 0.5, got %f", el.animOffset)
		}
	})

	t.Run("Layout calculates size correctly", func(t *testing.T) {
		pb := DefaultProgressBar()
		pb.Height = 10
		el := pb.CreateElement().(*ProgressBarElement)
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{
				MinWidth:  0,
				MaxWidth:  200,
				MinHeight: 0,
				MaxHeight: 100,
			},
		}
		result := el.Layout(ctx)
		if result.Size.Width != 200 {
			t.Errorf("Width should be 200 (max constraint), got %f", result.Size.Width)
		}
		// ProgressBarElement.Layout 返回 Height + labelH + 4（底部内边距）
		// Height=10, labelH=0, 所以 totalH = 10 + 0 + 4 = 14
		expectedHeight := pb.Height + 4
		if result.Size.Height != expectedHeight {
			t.Errorf("Height should be %f (Height + 4 padding), got %f", expectedHeight, result.Size.Height)
		}
	})

	t.Run("Layout with ShowLabel includes label height", func(t *testing.T) {
		pb := DefaultProgressBar()
		pb.Height = 8
		pb.ShowLabel = true
		el := pb.CreateElement().(*ProgressBarElement)
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{
				MinWidth:  0,
				MaxWidth:  200,
				MinHeight: 0,
				MaxHeight: 100,
			},
		}
		el.Layout(ctx)
		// 有标签时高度应大于只有进度条高度（8 + labelH + 4）
		if el.Size().Height <= 8+4 {
			t.Errorf("Height with ShowLabel should be > 12, got %f", el.Size().Height)
		}
	})

	t.Run("Update replaces widget reference via BaseElement", func(t *testing.T) {
		pb1 := DefaultProgressBar()
		pb1.Value = 0.3
		el := pb1.CreateElement().(*ProgressBarElement)
		el.SetAnimOffset(0.7)

		pb2 := DefaultProgressBar()
		pb2.Value = 0.8
		// ProgressBarElement 没有自定义 Update 方法，使用 BaseElement.Update
		// 只更新 BaseElement.widget 和 dirty 标记
		el.Update(pb2)

		if el.Widget() != pb2 {
			t.Error("Element.Widget() should return the new widget after Update")
		}
		// animOffset 应该保留（存储在 Element 上的状态）
		if el.animOffset != 0.7 {
			t.Errorf("animOffset should be preserved (0.7), got %f", el.animOffset)
		}
		if !el.dirty {
			t.Error("Element should be marked dirty after Update")
		}
		// 注意：ProgressBarElement 没有自定义 Update，所以 el.bar 仍指向旧 widget
		if el.bar != pb1 {
			t.Log("Note: ProgressBarElement has no custom Update(), bar reference not updated")
		}
	})
}

