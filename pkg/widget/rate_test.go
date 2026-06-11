package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
)

// TestRateValueAt 整星模式下局部 x → 分值映射(ss=20, gap=5, unit=25)。
func TestRateValueAt(t *testing.T) {
	el := DefaultRate().CreateElement().(*RateElement)
	if v := el.valueAt(20); v != 1 { // 第 1 颗星内 → 1
		t.Errorf("valueAt(20)=%v, want 1", v)
	}
	if v := el.valueAt(60); v != 3 { // idx=2, → 3
		t.Errorf("valueAt(60)=%v, want 3", v)
	}
	if v := el.valueAt(120); v != 5 { // 越界夹到最后一颗 → 5
		t.Errorf("valueAt(120)=%v, want 5", v)
	}
}

// TestRateAllowHalf 半星模式下星左半 → .5、右半 → 整数。
func TestRateAllowHalf(t *testing.T) {
	el := DefaultRate().WithAllowHalf(true).CreateElement().(*RateElement)
	if v := el.valueAt(5); v != 0.5 { // 第 1 颗星左半(within<10) → 0.5
		t.Errorf("valueAt(5) half=%v, want 0.5", v)
	}
	if v := el.valueAt(15); v != 1 { // 第 1 颗星右半 → 1
		t.Errorf("valueAt(15)=%v, want 1", v)
	}
}

// TestRateDisabled 只读评分不响应点击、值不变。
func TestRateDisabled(t *testing.T) {
	r := NewRate(3, nil).WithDisabled(true)
	el := r.CreateElement().(*RateElement)
	ev := event.NewMouseEvent(event.TypeMouseDown, 60, 5, event.ButtonLeft, event.ModNone)
	if el.HandleEvent(ev) {
		t.Error("只读 Rate 不应处理点击")
	}
	if r.Value != 3 {
		t.Errorf("只读 Rate 点击后值不应变，得 %v", r.Value)
	}
}

// TestRateClickSetsValue 点击设置分值并触发回调。
func TestRateClickSetsValue(t *testing.T) {
	got := -1.0
	r := NewRate(0, func(v float64) { got = v })
	el := r.CreateElement().(*RateElement)
	ev := event.NewMouseEvent(event.TypeMouseDown, 60, 5, event.ButtonLeft, event.ModNone)
	if !el.HandleEvent(ev) {
		t.Fatal("点击应被处理")
	}
	if r.Value != 3 || got != 3 {
		t.Errorf("点击第 3 颗星后 Value=%v got=%v, want 3", r.Value, got)
	}
}
