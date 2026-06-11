package widget

import (
	"testing"

	"github.com/hoonfeng/goui/internal/layout"
)

// TestMessageStyle 各类型返回不同图标字符与配色。
func TestMessageStyle(t *testing.T) {
	cases := []struct {
		mt   MessageType
		icon string
	}{
		{MsgSuccess, "✓"},
		{MsgWarning, "!"},
		{MsgError, "✕"},
		{MsgInfo, "i"},
	}
	for _, c := range cases {
		if _, _, _, ch := messageStyle(c.mt); ch != c.icon {
			t.Errorf("messageStyle(%d) icon=%q, want %q", c.mt, ch, c.icon)
		}
	}
}

// TestMessageBarLayout 消息条高度固定、宽度随文字增长。
func TestMessageBarLayout(t *testing.T) {
	short := (&messageBar{text: "短", mtype: MsgInfo}).CreateElement().(*messageBarElement)
	short.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 500, MaxHeight: 100}})
	if short.size.Height != messageBarH {
		t.Errorf("高度=%v, want %v", short.size.Height, messageBarH)
	}

	long := (&messageBar{text: "这是一条很长很长的消息内容", mtype: MsgInfo}).CreateElement().(*messageBarElement)
	long.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 500, MaxHeight: 100}})
	if long.size.Width <= short.size.Width {
		t.Errorf("长文字宽度(%.0f)应大于短文字(%.0f)", long.size.Width, short.size.Width)
	}
}
