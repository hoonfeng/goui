package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
)

// 把 InputNumber 元素布局到固定尺寸（HandleEvent 依赖 e.size 与按钮区划分）。
func layoutInputNumber(n *InputNumber) *InputNumberElement {
	el := n.CreateElement().(*InputNumberElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 180, MaxHeight: 32}})
	return el
}

// TestInputNumberStep 点右 [+] / 左 [−] 区按 Step 增减。
func TestInputNumberStep(t *testing.T) {
	n := NewInputNumber(5, nil).WithStep(2)
	el := layoutInputNumber(n)
	// 右 [+] 区：x > 180-32=148
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 160, 16, event.ButtonLeft, event.ModNone))
	if n.Value != 7 {
		t.Errorf("点 [+] 后 value=%v, want 7", n.Value)
	}
	// 左 [−] 区：x < 32
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 10, 16, event.ButtonLeft, event.ModNone))
	if n.Value != 5 {
		t.Errorf("点 [−] 后 value=%v, want 5", n.Value)
	}
}

// TestInputNumberClamp 到边界不越界。
func TestInputNumberClamp(t *testing.T) {
	n := NewInputNumber(10, nil).WithRange(0, 10)
	el := layoutInputNumber(n)
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 160, 16, event.ButtonLeft, event.ModNone))
	if n.Value != 10 {
		t.Errorf("已到上界，点 [+] 后 value=%v, want 10", n.Value)
	}
}

// TestInputNumberDisabled 禁用不响应。
func TestInputNumberDisabled(t *testing.T) {
	n := NewInputNumber(5, nil).WithDisabled(true)
	el := layoutInputNumber(n)
	if el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 160, 16, event.ButtonLeft, event.ModNone)) {
		t.Error("禁用的 InputNumber 不应处理点击")
	}
	if n.Value != 5 {
		t.Errorf("禁用点击后 value=%v, want 5", n.Value)
	}
}

// TestInputNumberPrecision 精度格式化。
func TestInputNumberPrecision(t *testing.T) {
	n := NewInputNumber(3.14159, nil).WithPrecision(2)
	if s := n.formatValue(); s != "3.14" {
		t.Errorf("precision=2 formatValue=%q, want \"3.14\"", s)
	}
}
