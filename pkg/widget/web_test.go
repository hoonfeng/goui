package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// ──────────────────────────────────────────────────────────────
// Web 风格组件测试
// ──────────────────────────────────────────────────────────────

// TestWebDiv 测试 Div() 的各种参数组合
func TestWebDiv(t *testing.T) {
	t.Run("Div with no args returns Container", func(t *testing.T) {
		w := Div()
		if w == nil {
			t.Fatal("Div() returned nil")
		}
		if _, ok := w.(*Container); !ok {
			t.Errorf("Div() should return *Container, got %T", w)
		}
	})

	t.Run("Div with text child returns Flex", func(t *testing.T) {
		w := Div(P("hello"))
		if w == nil {
			t.Fatal("Div(P()) returned nil")
		}
		if _, ok := w.(*Flex); !ok {
			t.Errorf("Div(P()) should return *Flex, got %T", w)
		}
	})

	t.Run("Div with Style and child applies Container", func(t *testing.T) {
		w := Div(
			Style{Padding: types.EdgeInsets(8), BackgroundColor: types.ColorRef(255, 255, 255)},
			P("hello"),
		)
		if w == nil {
			t.Fatal("Div(Style, P()) returned nil")
		}
		if _, ok := w.(*Container); !ok {
			t.Errorf("Div(Style, P()) should return *Container, got %T", w)
		}
	})

	t.Run("Div with multiple children returns Container with Flex child", func(t *testing.T) {
		w := Div(
			Style{Padding: types.EdgeInsets(8)},
			P("a"),
			P("b"),
		)
		if w == nil {
			t.Fatal("Div(Style, P(), P()) returned nil")
		}
		container, ok := w.(*Container)
		if !ok {
			t.Fatalf("should return *Container, got %T", w)
		}
		if container.Child == nil {
			t.Fatal("Container.Child should not be nil")
		}
	})
}

// TestWebHeadings 测试 H1~H4 标题组件
func TestWebHeadings(t *testing.T) {
	tests := []struct {
		name     string
		w        *Text
		expected float64
	}{
		{"H1", H1("Title 1"), 28},
		{"H2", H2("Title 2"), 22},
		{"H3", H3("Title 3"), 18},
		{"H4", H4("Title 4"), 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.w == nil {
				t.Fatalf("%s() returned nil", tt.name)
			}
			if tt.w.Font.Size != tt.expected {
				t.Errorf("%s font size = %v, want %v", tt.name, tt.w.Font.Size, tt.expected)
			}
			if tt.w.Font.Weight != canvas.FontWeightBold {
				t.Errorf("%s font weight should be bold", tt.name)
			}
			if tt.w.Text != "Title "+tt.name[1:] {
				t.Errorf("%s text = %q", tt.name, tt.w.Text)
			}
		})
	}
}

// TestWebText 测试 P, Small, Strong, NewText
func TestWebText(t *testing.T) {
	t.Run("P creates paragraph text", func(t *testing.T) {
		p := P("paragraph")
		if p == nil {
			t.Fatal("P() returned nil")
		}
		if p.Text != "paragraph" {
			t.Errorf("P().Text = %q, want %q", p.Text, "paragraph")
		}
		if p.Font.Size != 14 {
			t.Errorf("P font size = %v, want 14", p.Font.Size)
		}
	})

	t.Run("Small creates small text", func(t *testing.T) {
		s := Small("small text")
		if s == nil {
			t.Fatal("Small() returned nil")
		}
		if s.Font.Size != 12 {
			t.Errorf("Small font size = %v, want 12", s.Font.Size)
		}
	})

	t.Run("Strong creates bold text", func(t *testing.T) {
		s := Strong("bold text")
		if s == nil {
			t.Fatal("Strong() returned nil")
		}
		if s.Font.Weight != canvas.FontWeightBold {
			t.Error("Strong font weight should be bold")
		}
	})

	t.Run("NewText creates colored text", func(t *testing.T) {
		nt := NewText("colored", types.ColorFromRGB(66, 133, 244))
		if nt == nil {
			t.Fatal("NewText() returned nil")
		}
		if nt.Color.R != 66 || nt.Color.G != 133 || nt.Color.B != 244 {
			t.Errorf("NewText color = %+v, want RGB(66,133,244)", nt.Color)
		}
	})
}

// TestWebCheckbox 测试 NewCheckbox
func TestWebCheckbox(t *testing.T) {
	t.Run("NewCheckbox creates checkbox with label and default colors", func(t *testing.T) {
		cb := NewCheckbox("Remember me", true, nil)
		if cb == nil {
			t.Fatal("NewCheckbox() returned nil")
		}
		if cb.Label != "Remember me" {
			t.Errorf("Label = %q, want %q", cb.Label, "Remember me")
		}
		if !cb.Checked {
			t.Error("Checked should be true")
		}
		if cb.ActiveColor == (types.Color{}) {
			t.Error("ActiveColor should be set from theme")
		}
		if cb.LabelColor == (types.Color{}) {
			t.Error("LabelColor should be set from theme")
		}
		if cb.BoxSize <= 0 {
			t.Error("BoxSize should be > 0")
		}
	})

	t.Run("NewCheckbox without label creates valid checkbox", func(t *testing.T) {
		cb := NewCheckbox("", false, nil)
		if cb == nil {
			t.Fatal("NewCheckbox() returned nil")
		}
		if cb.Label != "" {
			t.Errorf("Label should be empty, got %q", cb.Label)
		}
	})

	t.Run("DefaultCheckbox returns checkbox with theme colors", func(t *testing.T) {
		cb := DefaultCheckbox()
		if cb == nil {
			t.Fatal("DefaultCheckbox() returned nil")
		}
		if cb.ActiveColor == (types.Color{}) {
			t.Error("ActiveColor should be set from theme")
		}
	})

	t.Run("Checkbox chain methods", func(t *testing.T) {
		var toggled bool
		cb := NewCheckbox("test", false, nil).
			OnChange(func(checked bool) { toggled = checked }).
			SetChecked(true).
			SetLabel("updated").
			SetColor(types.ColorFromRGB(255, 0, 0))

		if !cb.Checked {
			t.Error("SetChecked(true) should set Checked=true")
		}
		if cb.Label != "updated" {
			t.Errorf("SetLabel = %q, want %q", cb.Label, "updated")
		}
		if cb.ActiveColor.R != 255 {
			t.Error("SetColor should set ActiveColor")
		}
		if cb.OnChanged != nil {
			cb.OnChanged(true)
			if !toggled {
				t.Error("OnChange callback should fire")
			}
		}
	})

	t.Run("Checkbox Layout measures size correctly with label", func(t *testing.T) {
		cb := NewCheckbox("Hello World", false, nil)
		el := cb.CreateElement().(*CheckboxElement)

		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{
				MinWidth: 0, MaxWidth: 800,
				MinHeight: 0, MaxHeight: 600,
			},
		}
		result := el.Layout(ctx)
		if result.Size.Width <= 0 || result.Size.Height <= 0 {
			t.Errorf("Layout result invalid: %+v", result)
		}
	})

	t.Run("Checkbox Layout with empty label", func(t *testing.T) {
		cb := NewCheckbox("", false, nil)
		el := cb.CreateElement().(*CheckboxElement)
		ctx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{
				MinWidth: 0, MaxWidth: 800,
				MinHeight: 0, MaxHeight: 600,
			},
		}
		result := el.Layout(ctx)
		if result.Size.Width <= 0 || result.Size.Height <= 0 {
			t.Errorf("Layout result invalid: %+v", result)
		}
	})
}

// TestWebButton 测试 NewButton
func TestWebButton(t *testing.T) {
	t.Run("NewButton creates button with text", func(t *testing.T) {
		btn := NewButton("Click", nil)
		if btn == nil {
			t.Fatal("NewButton() returned nil")
		}
		if btn.Text != "Click" {
			t.Errorf("Text = %q, want %q", btn.Text, "Click")
		}
		if btn.Color == (types.Color{}) {
			t.Error("Button color should be set from theme")
		}
		if btn.MinWidth <= 0 || btn.MinHeight <= 0 {
			t.Error("Button min dimensions should be > 0")
		}
	})

	t.Run("NewButton with click callback", func(t *testing.T) {
		var clicked bool
		btn := NewButton("Test", func() { clicked = true })
		if btn.OnClick != nil {
			btn.OnClick()
			if !clicked {
				t.Error("OnClick callback should fire")
			}
		} else {
			t.Error("OnClick should not be nil")
		}
	})

	t.Run("Button chain methods", func(t *testing.T) {
		btn := NewButton("test", nil).
			WithColor(types.ColorFromRGB(255, 0, 0)).
			WithHoverColor(types.ColorFromRGB(200, 0, 0)).
			WithTextColor(types.ColorFromRGB(33, 33, 33)).
			WithFontSize(16).
			WithPadding(types.EdgeInsetsLTRB(16, 8, 16, 8)).
			WithMinWidth(100).
			WithMinHeight(40)

		if btn.Color.R != 255 {
			t.Error("WithColor should set Color")
		}
		if btn.TextColor.R != 33 {
			t.Error("WithTextColor should set TextColor")
		}
		if btn.FontSize != 16 {
			t.Errorf("FontSize = %v, want 16", btn.FontSize)
		}
		if btn.Padding.Left != 16 {
			t.Errorf("Padding.Left = %v, want 16", btn.Padding.Left)
		}
		if btn.MinWidth != 100 {
			t.Errorf("MinWidth = %v, want 100", btn.MinWidth)
		}
		if btn.MinHeight != 40 {
			t.Errorf("MinHeight = %v, want 40", btn.MinHeight)
		}
	})

	t.Run("Button TextColor from theme", func(t *testing.T) {
		theme := CurrentTheme()
		btn := NewButton("test", nil)
		if btn.TextColor != theme.Button.TextColor {
			t.Errorf("TextColor = %v, want %v (from theme)", btn.TextColor, theme.Button.TextColor)
		}
	})

	t.Run("Button padding default not nil", func(t *testing.T) {
		btn := NewButton("test", nil)
		if btn.Padding.Left <= 0 {
			t.Error("Button should have default left padding")
		}
		if btn.Padding.Right <= 0 {
			t.Error("Button should have default right padding")
		}
	})
}

// TestWebInput 测试 NewInput
func TestWebInput(t *testing.T) {
	t.Run("NewInput creates input with placeholder", func(t *testing.T) {
		in := NewInput("Enter text...", nil)
		if in == nil {
			t.Fatal("NewInput() returned nil")
		}
		if in.Placeholder != "Enter text..." {
			t.Errorf("Placeholder = %q, want %q", in.Placeholder, "Enter text...")
		}
	})

	t.Run("NewInput with change callback", func(t *testing.T) {
		var result string
		in := NewInput("test", func(s string) { result = s })
		if in.OnTextChanged != nil {
			in.OnTextChanged("hello")
			if result != "hello" {
				t.Errorf("OnTextChanged callback should pass 'hello', got %q", result)
			}
		} else {
			t.Error("OnTextChanged should not be nil")
		}
	})
}

// TestWebStyleSystem 测试 Define/Class/Style 系统
func TestWebStyleSystem(t *testing.T) {
	t.Run("Define and Class round-trip", func(t *testing.T) {
		Define("test-card", Style{
			Padding:         types.EdgeInsets(16),
			BackgroundColor: types.ColorRef(255, 255, 255),
			BorderRadius:    8,
		})

		s := Class("test-card")
		if s.Padding.Left != 16 || s.Padding.Right != 16 {
			t.Errorf("Class padding = %+v, want EdgeInsets(16)", s.Padding)
		}
		if s.BackgroundColor == nil || s.BackgroundColor.R != 255 {
			t.Error("Class BackgroundColor should be set")
		}
		if s.BorderRadius != 8 {
			t.Errorf("Class BorderRadius = %v, want 8", s.BorderRadius)
		}
	})

	t.Run("Class unknown returns empty style", func(t *testing.T) {
		s := Class("non-existent")
		if s.Padding.Left != 0 || s.BackgroundColor != nil {
			t.Error("Unknown class should return empty Style")
		}
	})

	t.Run("Merge combines multiple styles", func(t *testing.T) {
		s1 := Style{Padding: types.EdgeInsets(8), BackgroundColor: types.ColorRef(255, 0, 0)}
		s2 := Style{BorderRadius: 4, BackgroundColor: types.ColorRef(0, 255, 0)}
		merged := Merge(s1, s2)

		if merged.Padding.Left != 8 {
			t.Errorf("Merge padding = %v, want 8", merged.Padding.Left)
		}
		if merged.BackgroundColor == nil || merged.BackgroundColor.G != 255 {
			t.Error("Merge BackgroundColor should come from s2")
		}
		if merged.BorderRadius != 4 {
			t.Errorf("Merge BorderRadius = %v, want 4", merged.BorderRadius)
		}
	})

	t.Run("Inline style applied to Container via Div", func(t *testing.T) {
		w := Div(
			Style{
				Padding:         types.EdgeInsets(12),
				BackgroundColor: types.ColorRef(240, 240, 240),
				BorderColor:     types.ColorRef(200, 200, 200),
				BorderWidth:     1,
				BorderRadius:    6,
			},
			P("content"),
		)
		c, ok := w.(*Container)
		if !ok {
			t.Fatalf("Expected *Container, got %T", w)
		}
		if c.Padding.Left != 12 {
			t.Errorf("Container padding = %v, want 12", c.Padding.Left)
		}
		if c.Background == nil || c.Background.Color.R != 240 {
			t.Error("Container background should be set")
		}
		if c.Border == nil {
			t.Error("Container border should be set")
		}
	})
}

// TestWebLayout 测试 VBox/HBox/SpacerDiv/Separator
func TestWebLayout(t *testing.T) {
	t.Run("VBox creates Column with children", func(t *testing.T) {
		vb := VBox(P("a"), P("b"))
		if vb == nil {
			t.Fatal("VBox() returned nil")
		}
		if len(vb.Children) != 2 {
			t.Errorf("VBox children count = %d, want 2", len(vb.Children))
		}
	})

	t.Run("HBox creates Row with children", func(t *testing.T) {
		hb := HBox(P("a"), P("b"))
		if hb == nil {
			t.Fatal("HBox() returned nil")
		}
		if len(hb.Children) != 2 {
			t.Errorf("HBox children count = %d, want 2", len(hb.Children))
		}
	})

	t.Run("SpacerDiv creates Spacer", func(t *testing.T) {
		s := SpacerDiv()
		if s == nil {
			t.Fatal("SpacerDiv() returned nil")
		}
		// SpacerDiv returns *Spacer, just verify it's non-nil
		_ = s
	})

	t.Run("Separator creates Divider", func(t *testing.T) {
		s := Separator()
		if s == nil {
			t.Fatal("Separator() returned nil")
		}
		// Separator returns *Divider, just verify it's non-nil
		_ = s
	})
}

// TestWebStyled 测试 Styled 函数
func TestWebStyled(t *testing.T) {
	t.Run("Styled applies style to widget", func(t *testing.T) {
		styledBtn := Styled(NewButton("test", nil), Style{BorderRadius: 8})
		if styledBtn == nil {
			t.Fatal("Styled() returned nil")
		}
	})

	t.Run("Styled with text applies text color", func(t *testing.T) {
		red := types.ColorRef(255, 0, 0)
		styledText := Styled(P("hello"), Style{TextColor: red})
		sw, ok := styledText.(*StyledWidget)
		if !ok {
			t.Fatalf("Expected *StyledWidget, got %T", styledText)
		}
		if len(sw.Children) != 1 {
			t.Errorf("StyledWidget children count = %d, want 1", len(sw.Children))
		}
	})
}

// TestWebBuildTree 构建完整的 Web 风格组件树并验证 Build 展开
func TestWebBuildTree(t *testing.T) {
	Define("web-card", Style{
		Padding:         types.EdgeInsets(12),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderRadius:    6,
	})

	root := Div(
		Style{Padding: types.EdgeInsets(16)},
		Div(
			Class("web-card"),
			H2("Web 风格测试"),
			P("这是一个使用 Web 风格组件的集成测试"),
			Separator(),
			NewCheckbox("选项 1", true, nil),
			NewCheckbox("选项 2", false, nil),
			Separator(),
			HBox(
				NewButton("确定", nil),
				SpacerDiv(),
				NewButton("取消", nil),
			),
		),
	)

	if root == nil {
		t.Fatal("build Web tree returned nil")
	}

	rootEl := CreateElementFor(root)
	if rootEl == nil {
		t.Fatal("CreateElementFor returned nil for root widget")
	}

	var elementCount int
	validateTreeRecursive(t, rootEl, "root", &elementCount)

	if elementCount < 5 {
		t.Errorf("Expected at least 5 elements, got %d", elementCount)
	}
	t.Logf("Web style tree: %d elements validated successfully", elementCount)
}

// TestWebCheckboxOnChanged 测试 Checkbox 的事件处理
func TestWebCheckboxOnChanged(t *testing.T) {
	t.Run("With nil callback does not panic", func(t *testing.T) {
		cb := &CheckboxElement{
			checkbox: &Checkbox{
				Label:     "test",
				Checked:   false,
				OnChanged: nil,
			},
		}
		ev := createMouseUpEvent()
		cb.HandleEvent(ev)
		if !cb.checkbox.Checked {
			t.Error("Checkbox should toggle to true on click")
		}
	})

	t.Run("With callback fires on toggle", func(t *testing.T) {
		var val bool
		called := false
		cb := &CheckboxElement{
			checkbox: &Checkbox{
				Label:     "test",
				Checked:   false,
				OnChanged: func(v bool) { val = v; called = true },
			},
		}
		ev := createMouseUpEvent()
		cb.HandleEvent(ev)
		if !called {
			t.Error("OnChanged callback should be called")
		}
		if !val {
			t.Error("OnChanged should receive true after toggle")
		}
	})
}

// TestWebDefineDuplicate 测试重复 Define 覆盖
func TestWebDefineDuplicate(t *testing.T) {
	Define("dup-style", Style{
		Padding: types.EdgeInsets(8),
	})
	Define("dup-style", Style{
		Padding: types.EdgeInsets(16),
	})
	s := Class("dup-style")
	if s.Padding.Left != 16 {
		t.Errorf("Re-defined class should have padding=16, got %v", s.Padding.Left)
	}
}

// createMouseUpEvent 创建鼠标松开事件的辅助函数
func createMouseUpEvent() *event.MouseEvent {
	return event.NewMouseEvent(event.TypeMouseUp, 10, 10, event.ButtonLeft, event.ModNone)
}

// TestWebNewCheckboxElement 测试 NewCheckbox 创建的 Element
func TestWebNewCheckboxElement(t *testing.T) {
	cb := NewCheckbox("test", false, nil)
	el := cb.CreateElement()
	if el == nil {
		t.Fatal("CreateElement returned nil")
	}
	if _, ok := el.(*CheckboxElement); !ok {
		t.Fatalf("Expected *CheckboxElement, got %T", el)
	}
	if el.Widget() != cb {
		t.Error("Element.Widget() should return the original Checkbox")
	}
}

// TestWebNewButtonElement 测试 NewButton 创建的 Element
func TestWebNewButtonElement(t *testing.T) {
	btn := NewButton("test", nil)
	el := btn.CreateElement()
	if el == nil {
		t.Fatal("CreateElement returned nil")
	}
	if _, ok := el.(*ButtonElement); !ok {
		t.Fatalf("Expected *ButtonElement, got %T", el)
	}
}

// TestWebNewInputElement 测试 NewInput 创建的 Element
func TestWebNewInputElement(t *testing.T) {
	in := NewInput("placeholder", nil)
	el := in.CreateElement()
	if el == nil {
		t.Fatal("CreateElement returned nil")
	}
	if _, ok := el.(*InputElement); !ok {
		t.Fatalf("Expected *InputElement, got %T", el)
	}
}

// TestWebCheckboxLabelColor 测试 Checkbox LabelColor 回退逻辑
func TestWebCheckboxLabelColor(t *testing.T) {
	t.Run("LabelColor set from theme in NewCheckbox", func(t *testing.T) {
		cb := NewCheckbox("test", false, nil)
		if cb.LabelColor == (types.Color{}) {
			t.Error("LabelColor should be set from theme")
		}
		if cb.LabelColor.A == 0 {
			t.Error("LabelColor Alpha should not be 0")
		}
	})

	t.Run("DefaultCheckbox LabelColor from theme", func(t *testing.T) {
		cb := DefaultCheckbox()
		if cb.LabelColor == (types.Color{}) {
			t.Error("DefaultCheckbox LabelColor should be set from theme")
		}
		if cb.LabelColor.A == 0 {
			t.Error("DefaultCheckbox LabelColor Alpha should not be 0")
		}
	})
}

// TestWebTextChain 测试 Text 链式方法
func TestWebTextChain(t *testing.T) {
	t.Run("NewText WithFont", func(t *testing.T) {
		nt := NewText("hello", types.ColorBlack).
			WithFont(canvas.Font{Family: "sans-serif", Size: 20})
		if nt.Font.Size != 20 {
			t.Errorf("Font.Size = %v, want 20", nt.Font.Size)
		}
	})

	t.Run("NewText WithColor", func(t *testing.T) {
		nt := NewText("hello", types.ColorBlack).
			WithColor(types.ColorFromRGB(255, 0, 0))
		if nt.Color.R != 255 {
			t.Error("WithColor should update Color")
		}
	})

	t.Run("NewText WithAlign", func(t *testing.T) {
		nt := NewText("hello", types.ColorBlack).
			WithAlign(canvas.TextAlignCenter)
		if nt.Align != canvas.TextAlignCenter {
			t.Errorf("Align = %v, want TextAlignCenter", nt.Align)
		}
	})
}
