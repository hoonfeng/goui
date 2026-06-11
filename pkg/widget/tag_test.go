package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

func newTagElement(tg *Tag) *TagElement { return tg.CreateElement().(*TagElement) }

func layoutTag(e *TagElement) float64 {
	r := e.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
		MinWidth: 0, MaxWidth: 1000, MinHeight: 0, MaxHeight: 1000,
	}})
	return r.Size.Width
}

func TestTagBaseColor(t *testing.T) {
	tg := CurrentTheme().Tag
	cases := []struct {
		tp   TagType
		want types.Color
	}{
		{TagPrimary, tg.PrimaryColor},
		{TagSuccess, tg.SuccessColor},
		{TagInfo, tg.InfoColor},
		{TagWarning, tg.WarningColor},
		{TagDanger, tg.DangerColor},
	}
	for _, c := range cases {
		if got := (&Tag{Type: c.tp}).baseColor(); got != c.want {
			t.Errorf("baseColor(type=%d)=%v, want %v", c.tp, got, c.want)
		}
	}
	custom := types.ColorFromRGB(1, 2, 3)
	if got := (&Tag{Type: TagPrimary, Color: custom}).baseColor(); got != custom {
		t.Errorf("自定义色应优先：baseColor=%v, want %v", got, custom)
	}
}

func TestTagResolveColors(t *testing.T) {
	base := CurrentTheme().Tag.PrimaryColor

	// dark：实色底 + 白字
	bg, _, text := newTagElement(NewTag("x").WithEffect(TagDark)).resolveColors()
	if bg != base || text != types.ColorWhite {
		t.Errorf("dark bg=%v text=%v, want bg=base text=white", bg, text)
	}

	// plain：白底 + 主色字
	bg, _, text = newTagElement(NewTag("x").WithEffect(TagPlain)).resolveColors()
	if bg != types.ColorWhite || text != base {
		t.Errorf("plain bg=%v text=%v, want bg=white text=base", bg, text)
	}

	// light：主色字 + 介于主色与白之间的浅底
	bg, _, text = newTagElement(NewTag("x").WithEffect(TagLight)).resolveColors()
	if text != base {
		t.Errorf("light text=%v, want base %v", text, base)
	}
	if bg == base || bg == types.ColorWhite {
		t.Errorf("light bg=%v, want 介于主色与白之间的浅色", bg)
	}
}

func TestTagSizeMetrics(t *testing.T) {
	hl, _, fl, _, _ := tagSizeMetrics(TagSizeLarge)
	hd, _, _, _, _ := tagSizeMetrics(TagSizeDefault)
	hs, _, _, _, _ := tagSizeMetrics(TagSizeSmall)
	if !(hl > hd && hd > hs) {
		t.Errorf("高度未递减：large=%.0f default=%.0f small=%.0f", hl, hd, hs)
	}
	if fl != 14 {
		t.Errorf("large 字号=%.0f, want 14", fl)
	}
}

func TestTagClosableWider(t *testing.T) {
	wp := layoutTag(newTagElement(NewTag("标签")))
	wc := layoutTag(newTagElement(NewTag("标签").WithClosable(true)))
	if wc <= wp {
		t.Errorf("可关闭标签宽度=%.1f 应大于普通标签=%.1f", wc, wp)
	}
}

func TestTagRoundRadius(t *testing.T) {
	e := newTagElement(NewTag("圆").WithRound(true))
	layoutTag(e)
	if e.size.Height <= 0 {
		t.Fatal("布局后高度应 > 0")
	}
}

func TestTagEventDispatch(t *testing.T) {
	clicked, closed := false, false
	tg := NewTag("可关闭").WithClosable(true)
	tg.OnClick = func() { clicked = true }
	tg.OnClose = func() { closed = true }
	e := newTagElement(tg)
	layoutTag(e) // 确定尺寸（offset 默认 0,0）

	down := func(x, y float64) {
		e.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, x, y, event.ButtonLeft, event.ModNone))
		e.HandleEvent(event.NewMouseEvent(event.TypeMouseUp, x, y, event.ButtonLeft, event.ModNone))
	}

	// 点击关闭区域 → 仅 close
	cx, cy, _ := e.closeCenter()
	down(cx, cy)
	if !closed || clicked {
		t.Errorf("点 × 应只触发 close：closed=%v clicked=%v", closed, clicked)
	}

	// 点击主体 → 仅 click
	clicked, closed = false, false
	down(2, cy)
	if !clicked || closed {
		t.Errorf("点主体应只触发 click：clicked=%v closed=%v", clicked, closed)
	}
}

func TestTagDeclarative(t *testing.T) {
	factory, ok := componentRegistry["Tag"]
	if !ok {
		t.Fatal("Tag 未注册到 componentRegistry")
	}
	spec := ComponentSpec{
		Type: "Tag",
		Text: "测试",
		Props: map[string]any{
			"type":     "success",
			"effect":   "dark",
			"size":     "large",
			"closable": true,
			"round":    true,
		},
	}
	w := factory(DeclarativeContext{Spec: spec})
	tg, ok := w.(*Tag)
	if !ok {
		t.Fatalf("工厂返回 %T, want *Tag", w)
	}
	if tg.Text != "测试" || tg.Type != TagSuccess || tg.Effect != TagDark ||
		tg.Size != TagSizeLarge || !tg.Closable || !tg.Round {
		t.Errorf("声明式属性映射有误：%+v", tg)
	}
}
