package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
)

// TestSwitchDisabled 禁用的开关不响应点击、状态不变。
func TestSwitchDisabled(t *testing.T) {
	sw := NewSwitch(false, nil).WithDisabled(true)
	el := sw.CreateElement().(*SwitchElement)
	ev := event.NewMouseEvent(event.TypeMouseUp, 5, 5, event.ButtonLeft, event.ModNone)
	if el.HandleEvent(ev) {
		t.Error("禁用的 Switch 不应处理 MouseUp 事件")
	}
	if sw.Value {
		t.Error("禁用的 Switch 点击后 Value 不应改变")
	}
}

// TestSwitchSize 尺寸变体返回正确的轨道宽高。
func TestSwitchSize(t *testing.T) {
	cases := []struct {
		size       string
		wantW      float64
		wantH      float64
	}{
		{"large", 50, 26},
		{"", 44, 24},
		{"small", 36, 18},
	}
	for _, c := range cases {
		sw := NewSwitch(true, nil).WithSize(c.size)
		w, h := sw.dims()
		if w != c.wantW || h != c.wantH {
			t.Errorf("size=%q dims=(%.0f,%.0f), want (%.0f,%.0f)", c.size, w, h, c.wantW, c.wantH)
		}
	}
}

// TestSwitchChainSetters 链式 setter 正确赋值。
func TestSwitchChainSetters(t *testing.T) {
	sw := NewSwitch(false, nil).
		WithLabel("启用").
		WithTexts("开", "关").
		WithDisabled(true).
		WithSize("large")
	if sw.Label != "启用" || sw.ActiveText != "开" || sw.InactiveText != "关" || !sw.Disabled || sw.Size != "large" {
		t.Errorf("链式 setter 赋值不正确: %+v", sw)
	}
}
