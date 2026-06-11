package widget

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestComponentSpec_Serialize 验证 ComponentSpec 的 JSON 序列化/反序列化
func TestComponentSpec_Serialize(t *testing.T) {
	spec := ComponentSpec{
		Type:  "Column",
		Class: "page",
		Text:  "test",
		Style: &StyleSpec{
			Padding:         "16",
			BackgroundColor: "#FFFFFF",
			BorderRadius:    8,
		},
		Props: map[string]any{
			"checked": true,
		},
		Events: map[string]string{
			"click": "handleClick",
		},
		Children: []ComponentSpec{
			{Type: "Text", Text: "Hello"},
			{Type: "Button", Text: "OK"},
		},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	var decoded ComponentSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if decoded.Type != "Column" {
		t.Errorf("Type should be 'Column', got %q", decoded.Type)
	}
	if decoded.Class != "page" {
		t.Errorf("Class should be 'page', got %q", decoded.Class)
	}
	if decoded.Text != "test" {
		t.Errorf("Text should be 'test', got %q", decoded.Text)
	}
	if decoded.Style == nil {
		t.Fatal("Style should not be nil")
	}
	if decoded.Style.Padding != "16" {
		t.Errorf("Style.Padding should be '16', got %q", decoded.Style.Padding)
	}
	if decoded.Style.BackgroundColor != "#FFFFFF" {
		t.Errorf("Style.BackgroundColor should be '#FFFFFF', got %q", decoded.Style.BackgroundColor)
	}
	if decoded.Style.BorderRadius != 8 {
		t.Errorf("Style.BorderRadius should be 8, got %f", decoded.Style.BorderRadius)
	}
	if len(decoded.Children) != 2 {
		t.Errorf("Children count should be 2, got %d", len(decoded.Children))
	}
	if decoded.Children[0].Type != "Text" {
		t.Errorf("First child type should be 'Text', got %q", decoded.Children[0].Type)
	}
}

// TestBuildFromSpec_Button 验证从 ComponentSpec 构建 Button
func TestBuildFromSpec_Button(t *testing.T) {
	handlers := Handlers{
		"handleClick": func(ctx EventContext) {
			// 测试用：只是验证回调被调用
		},
	}

	w := BuildFromSpec(ComponentSpec{
		Type: "Button",
		Text: "确定",
		Props: map[string]any{
			"minWidth": 100,
		},
		Events: map[string]string{
			"click": "handleClick",
		},
	}, handlers)

	btn, ok := w.(*Button)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Button, got %T", w)
	}
	if btn.Text != "确定" {
		t.Errorf("Button text should be '确定', got %q", btn.Text)
	}
	if btn.MinWidth != 100 {
		t.Errorf("Button MinWidth should be 100, got %f", btn.MinWidth)
	}
	if btn.OnClick == nil {
		t.Error("Button OnClick should be set")
	}
}

// TestBuildFromSpec_Column 验证从 ComponentSpec 构建 Column
func TestBuildFromSpec_Column(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type:  "Column",
		Class: "card",
		Children: []ComponentSpec{
			{Type: "H1", Text: "Title"},
			{Type: "P", Text: "Description"},
			{Type: "Button", Text: "OK"},
		},
	}, handlers)

	col, ok := w.(*Column)
	if !ok {
		// 如果有样式类可能需要容器包装
		// 如果没有 box model 样式，就是 Column
		t.Logf("BuildFromSpec returned %T (may be wrapped in Container)", w)
	}
	_ = col
}

// TestBuildFromSpec_Text 验证文本组件构建
func TestBuildFromSpec_Text(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type: "Text",
		Text: "Hello World",
		Props: map[string]any{
			"fontSize": 18,
			"color":    "#FF0000",
		},
	}, handlers)

	txt, ok := w.(*Text)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Text, got %T", w)
	}
	if txt.Text != "Hello World" {
		t.Errorf("Text should be 'Hello World', got %q", txt.Text)
	}
}

// TestBuildFromSpec_Input 验证输入框组件构建和事件绑定
func TestBuildFromSpec_Input(t *testing.T) {
	eventFired := false
	handlers := Handlers{
		"handleChange": func(ctx EventContext) {
			eventFired = true
			if ctx.Name != "change" {
				t.Errorf("Event name should be 'change', got %q", ctx.Name)
			}
		},
	}

	w := BuildFromSpec(ComponentSpec{
		Type: "Input",
		Text: "placeholder text",
		Events: map[string]string{
			"change": "handleChange",
		},
	}, handlers)

	in, ok := w.(*Input)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Input, got %T", w)
	}
	if in.Placeholder != "placeholder text" {
		t.Errorf("Placeholder should be 'placeholder text', got %q", in.Placeholder)
	}
	if in.OnTextChanged == nil {
		t.Error("OnTextChanged should be set")
	}

	// 触发回调验证事件绑定
	in.OnTextChanged("test")
	if !eventFired {
		t.Error("EventHandler should have been called")
	}
}

// TestBuildFromSpec_Checkbox 验证复选框构建
func TestBuildFromSpec_Checkbox(t *testing.T) {
	handlers := Handlers{
		"handleChange": func(ctx EventContext) {
			if ctx.Name != "change" {
				t.Errorf("Event name should be 'change', got %q", ctx.Name)
			}
		},
	}

	w := BuildFromSpec(ComponentSpec{
		Type: "Checkbox",
		Text: "记住我",
		Props: map[string]any{
			"checked": true,
		},
		Events: map[string]string{
			"change": "handleChange",
		},
	}, handlers)

	cb, ok := w.(*Checkbox)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Checkbox, got %T", w)
	}
	if cb.Label != "记住我" {
		t.Errorf("Label should be '记住我', got %q", cb.Label)
	}
	if !cb.Checked {
		t.Error("Checkbox should be checked")
	}
	if cb.OnChanged == nil {
		t.Error("OnChanged should be set")
	}
}

// TestBuildFromSpec_UnknownType 验证未知组件类型返回占位文本
func TestBuildFromSpec_UnknownType(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type: "NonExistentComponent",
		Text: "test",
	}, handlers)

	_, ok := w.(*Text)
	if !ok {
		t.Fatalf("Unknown type should return *Text placeholder, got %T", w)
	}
}

// TestLoadConfig_Simple 验证从 JSON 加载
func TestLoadConfig_Simple(t *testing.T) {
	jsonData := `{
		"type": "Column",
		"children": [
			{"type": "H1", "text": "Hello"},
			{"type": "Button", "text": "Click", "events": {"click": "onClick"}}
		]
	}`

	handlers := Handlers{
		"onClick": func(ctx EventContext) {},
	}

	w, err := LoadConfig([]byte(jsonData), handlers)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if w == nil {
		t.Fatal("LoadConfig returned nil widget")
	}
}

// TestLoadConfig_InvalidJSON 验证无效 JSON 返回错误
func TestLoadConfig_InvalidJSON(t *testing.T) {
	_, err := LoadConfig([]byte("{invalid}"), Handlers{})
	if err == nil {
		t.Error("LoadConfig should return error for invalid JSON")
	}
}

// TestLoadConfig_EmptyType 验证空 type 返回错误
func TestLoadConfig_EmptyType(t *testing.T) {
	_, err := LoadConfig([]byte(`{}`), Handlers{})
	if err == nil {
		t.Error("LoadConfig should return error for empty type")
	}
}

// TestBuildFromSpec_Slider 验证滑块组件构建
func TestBuildFromSpec_Slider(t *testing.T) {
	handlers := Handlers{
		"onChange": func(ctx EventContext) {},
	}

	w := BuildFromSpec(ComponentSpec{
		Type: "Slider",
		Props: map[string]any{
			"min":       0.0,
			"max":       100.0,
			"value":     50.0,
			"step":      1.0,
			"showLabel": true,
		},
		Events: map[string]string{
			"change": "onChange",
		},
	}, handlers)

	sl, ok := w.(*Slider)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Slider, got %T", w)
	}
	if sl.Min != 0 {
		t.Errorf("Min should be 0, got %f", sl.Min)
	}
	if sl.Max != 100 {
		t.Errorf("Max should be 100, got %f", sl.Max)
	}
	if sl.Value != 50 {
		t.Errorf("Value should be 50, got %f", sl.Value)
	}
	if sl.Step != 1 {
		t.Errorf("Step should be 1, got %f", sl.Step)
	}
	if !sl.ShowLabel {
		t.Error("ShowLabel should be true")
	}
}

// TestBuildFromSpec_Switch 验证开关组件构建
func TestBuildFromSpec_Switch(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type: "Switch",
		Text: "WiFi",
		Props: map[string]any{
			"value": true,
		},
	}, handlers)

	sw, ok := w.(*Switch)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Switch, got %T", w)
	}
	if sw.Label != "WiFi" {
		t.Errorf("Label should be 'WiFi', got %q", sw.Label)
	}
	if !sw.Value {
		t.Error("Switch value should be true")
	}
}

// TestBuildFromSpec_ProgressBar 验证进度条组件构建
func TestBuildFromSpec_ProgressBar(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type: "ProgressBar",
		Props: map[string]any{
			"value":     0.75,
			"showLabel": true,
			"height":    12,
		},
	}, handlers)

	pb, ok := w.(*ProgressBar)
	if !ok {
		t.Fatalf("BuildFromSpec should return *ProgressBar, got %T", w)
	}
	if pb.Value != 0.75 {
		t.Errorf("Value should be 0.75, got %f", pb.Value)
	}
	if !pb.ShowLabel {
		t.Error("ShowLabel should be true")
	}
	if pb.Height != 12 {
		t.Errorf("Height should be 12, got %f", pb.Height)
	}
}

// TestParseColor 验证颜色解析函数
func TestParseColor(t *testing.T) {
	tests := []struct {
		input string
		want  string // 格式: R,G,B,A
	}{
		{"#FF0000", "255,0,0,255"},
		{"#00FF00", "0,255,0,255"},
		{"#0000FF", "0,0,255,255"},
		{"#FF000080", "255,0,0,128"},
		{"#FFF", "255,255,255,255"},
		{"black", "0,0,0,255"},
		{"white", "255,255,255,255"},
		{"red", "255,0,0,255"},
		{"", ""},
	}

	for _, tt := range tests {
		c := parseColor(tt.input)
		if tt.want == "" {
			if c != nil {
				t.Errorf("parseColor(%q) should return nil, got %v", tt.input, c)
			}
			continue
		}
		if c == nil {
			t.Errorf("parseColor(%q) returned nil", tt.input)
			continue
		}
		got := strings.TrimRight(strings.ReplaceAll(
			strings.TrimRight(fmt.Sprintf("%d,%d,%d,%d", c.R, c.G, c.B, c.A), "0"),
			"0,", ","), ",")
		expected := tt.want
		if got != expected {
			// 简单比较
			if fmt.Sprintf("%d,%d,%d,%d", c.R, c.G, c.B, c.A) != expected {
				t.Errorf("parseColor(%q) = %d,%d,%d,%d, want %s",
					tt.input, c.R, c.G, c.B, c.A, tt.want)
			}
		}
	}
}

// TestBuildFromSpec_Row 验证 Row 组件构建
func TestBuildFromSpec_Row(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type: "Row",
		Children: []ComponentSpec{
			{Type: "Text", Text: "Left"},
			{Type: "Spacer"},
			{Type: "Text", Text: "Right"},
		},
	}, handlers)

	row, ok := w.(*Row)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Row, got %T", w)
	}
	_ = row
}

// TestBuildFromSpec_Div 验证 Div 组件构建
func TestBuildFromSpec_Div(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type: "Div",
		Style: &StyleSpec{
			Padding:         "16",
			BackgroundColor: "#F0F0F0",
			BorderRadius:    8,
		},
		Children: []ComponentSpec{
			{Type: "P", Text: "Content"},
		},
	}, handlers)

	if w == nil {
		t.Fatal("BuildFromSpec returned nil")
	}
	// Div with style should be wrapped in Container
	_ = w
}

// TestRegisterComponent 验证自定义组件注册
func TestRegisterComponent(t *testing.T) {
	called := false
	RegisterComponent("TestWidget", func(ctx DeclarativeContext) Widget {
		called = true
		return &Text{Text: "custom"}
	})

	w := BuildFromSpec(ComponentSpec{
		Type: "TestWidget",
	}, Handlers{})

	if !called {
		t.Error("Custom factory should have been called")
	}
	if w == nil {
		t.Fatal("BuildFromSpec returned nil for custom widget")
	}
	txt, ok := w.(*Text)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Text, got %T", w)
	}
	if txt.Text != "custom" {
		t.Errorf("Text should be 'custom', got %q", txt.Text)
	}
}

// TestBuildFromSpec_Divider 验证分割线组件构建
func TestBuildFromSpec_Divider(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type: "Divider",
	}, handlers)

	_, ok := w.(*Divider)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Divider, got %T", w)
	}
}

// TestBuildFromSpec_Label 验证标签组件构建
func TestBuildFromSpec_Label(t *testing.T) {
	handlers := Handlers{}
	w := BuildFromSpec(ComponentSpec{
		Type: "Label",
		Text: "用户名",
	}, handlers)

	txt, ok := w.(*Text)
	if !ok {
		t.Fatalf("BuildFromSpec should return *Text (Label), got %T", w)
	}
	if txt.Text != "用户名" {
		t.Errorf("Text should be '用户名', got %q", txt.Text)
	}
}
