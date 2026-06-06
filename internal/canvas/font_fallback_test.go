package canvas

import (
	"testing"

	"github.com/user/goui/internal/paint"
)

func TestFallbackFamily(t *testing.T) {
	cases := map[rune]string{
		'ا':  "Tahoma",         // 阿拉伯
		'ש':  "Tahoma",         // 希伯来
		'😀': "Segoe UI Emoji",  // emoji
		'▾':  "Segoe UI Symbol", // 几何图形(用户实测方块→已回退)
		'◀':  "Segoe UI Symbol",
		'⎘':  "Segoe UI Symbol", // 技术符号
		'A':  "",                // 拉丁 → 默认
	}
	for r, want := range cases {
		if got := FallbackFamily(r); got != want {
			t.Errorf("FallbackFamily(%q)=%q, want %q", r, got, want)
		}
	}
}

// TestMixedScriptRenderNoPanic 多脚本混排（中文/阿拉伯/emoji/拉丁）分段绘制与测量不应 panic。
func TestMixedScriptRenderNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("mixed-script render panicked: %v", r)
		}
	}()
	c := NewSkiaCanvas(300, 60)
	c.DrawText("中文 English العربية 😀 123", 5, 30, DefaultFont(), paint.DefaultPaint())
	_ = c.Flush()
	if m := c.MeasureText("中文 العربية 😀", DefaultFont()); m.Width <= 0 {
		t.Errorf("mixed-script MeasureText width=%.1f, want >0", m.Width)
	}
}
