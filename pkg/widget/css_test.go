package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/canvas"
)

// TestStyleSpecToStyle 验证 JSON StyleSpec → Style 的新属性解析(含阴影字符串)。
func TestStyleSpecToStyle(t *testing.T) {
	spec := &StyleSpec{
		MaxWidth:   300,
		Opacity:    0.5,
		FontWeight: "bold",
		TextAlign:  "center",
		LineHeight: 24,
		Shadow:     "0 4 10 #00000040",
	}
	s := spec.ToStyle()

	if s.MaxWidth != 300 {
		t.Errorf("MaxWidth=%v, want 300", s.MaxWidth)
	}
	if s.Opacity == nil || *s.Opacity != 0.5 {
		t.Errorf("Opacity=%v, want 0.5", s.Opacity)
	}
	if s.FontWeight != canvas.FontWeightBold {
		t.Errorf("FontWeight=%v, want Bold", s.FontWeight)
	}
	if s.TextAlign != canvas.HAlignCenter {
		t.Errorf("TextAlign=%v, want Center", s.TextAlign)
	}
	if s.LineHeight != 24 {
		t.Errorf("LineHeight=%v, want 24", s.LineHeight)
	}
	if s.Shadow == nil {
		t.Fatal("Shadow 应从字符串解析成功")
	}
	if s.Shadow.Offset.Y != 4 || s.Shadow.Blur != 10 {
		t.Errorf("Shadow=%+v, want offsetY=4 blur=10", s.Shadow)
	}
}

// TestStyleSpecPseudo 验证 JSON StyleSpec 的嵌套伪类 → Style.Hover/Focus/Active 递归解析。
func TestStyleSpecPseudo(t *testing.T) {
	spec := &StyleSpec{
		BackgroundColor: "#ebeef5",
		Hover:           &StyleSpec{BackgroundColor: "#409eff", BorderColor: "#3078c8", BorderWidth: 2},
		Focus:           &StyleSpec{BackgroundColor: "#67c23a"},
		Active:          &StyleSpec{BackgroundColor: "#f56c6c"},
	}
	s := spec.ToStyle()
	if s.Hover == nil || s.Hover.BackgroundColor == nil {
		t.Fatal(":hover 应递归解析出嵌套样式")
	}
	if s.Hover.BorderColor == nil || s.Hover.BorderWidth != 2 {
		t.Errorf(":hover 边框应解析，得 color=%v width=%v", s.Hover.BorderColor, s.Hover.BorderWidth)
	}
	if s.Focus == nil || s.Focus.BackgroundColor == nil {
		t.Error(":focus 应递归解析")
	}
	if s.Active == nil || s.Active.BackgroundColor == nil {
		t.Error(":active 应递归解析")
	}
	// 端到端：经 applyToContainer 挂到 Container，ContainerElement 按状态解析
	c := &Container{}
	s.applyToContainer(c)
	e := &ContainerElement{container: c}
	e.hovered = true
	if got := e.effBg(); got == nil || got.B < 200 {
		t.Errorf("JSON :hover 应使背景偏蓝，得 %v", got)
	}
}

// TestStyleReflectMerge 验证反射式 mergeStyle：非零覆盖、零值保留、指针属性合并。
func TestStyleReflectMerge(t *testing.T) {
	op := 0.7
	base := Style{Width: 100, FontSize: 14}
	over := Style{Width: 200, Opacity: &op}
	m := Merge(base, over)

	if m.Width != 200 {
		t.Errorf("Width=%v, want 200(被覆盖)", m.Width)
	}
	if m.FontSize != 14 {
		t.Errorf("FontSize=%v, want 14(未被零值覆盖，保留)", m.FontSize)
	}
	if m.Opacity == nil || *m.Opacity != 0.7 {
		t.Errorf("Opacity=%v, want 0.7(指针属性合并)", m.Opacity)
	}
}
