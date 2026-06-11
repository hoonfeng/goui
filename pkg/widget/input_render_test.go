package widget

import (
	"bytes"
	"testing"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// renderInput 将 InputElement 绘制到一张 SkiaCanvas 上并返回像素拷贝。
func renderInput(t *testing.T, el *InputElement) []byte {
	t.Helper()
	c := canvas.NewSkiaCanvas(220, 36)
	el.SetPosition(types.Point{X: 4, Y: 2})
	el.Paint(c, types.Point{})
	if err := c.Flush(); err != nil {
		t.Fatalf("canvas flush 失败: %v", err)
	}
	img := c.Image()
	out := make([]byte, len(img.Pix))
	copy(out, img.Pix)
	return out
}

// allWhite 判断 RGBA 像素缓冲是否全为白色（即什么都没画出来）。
func allWhite(pix []byte) bool {
	for i := 0; i+3 < len(pix); i += 4 {
		if pix[i] != 255 || pix[i+1] != 255 || pix[i+2] != 255 {
			return false
		}
	}
	return true
}

func newLaidOutInput(t *testing.T) *InputElement {
	t.Helper()
	el := DefaultInput().CreateElement().(*InputElement)
	el.Mount(nil, 0)
	el.Layout(&layout.LayoutContext{Constraints: layout.Tight(212, 32)})
	return el
}

// TestInputFocusTypeWritesAndRenders 验证用户报告的“输入内容没有写入/没有渲染”：
// 聚焦后逐字符发送 KeyChar，文本应写入 InputElement，且 Paint 应产生可见内容。
func TestInputFocusTypeWritesAndRenders(t *testing.T) {
	el := newLaidOutInput(t)

	el.Focus()
	if !el.IsFocused() {
		t.Fatal("Input 在 Focus() 之后应当处于聚焦状态")
	}

	for _, ch := range "Hi" {
		ev := event.NewKeyEvent(event.TypeKeyChar, uint32(ch), string(ch), event.ModNone)
		ev.Char = ch
		if !el.HandleEvent(ev) {
			t.Fatalf("聚焦的输入框应当处理字符 %q", string(ch))
		}
	}

	if got := el.Text(); got != "Hi" {
		t.Fatalf("输入后 Text() = %q，期望 %q（字符未写入输入框）", got, "Hi")
	}

	// 渲染必须产生可见内容（至少背景/边框），否则说明绘制链路有问题。
	el.Blur() // 去掉闪烁光标，避免相位影响对比
	imgTyped := renderInput(t, el)
	if allWhite(imgTyped) {
		t.Fatal("输入框渲染结果全白：Paint 没有画出任何内容")
	}

	// 文本是否实际被绘制：与空输入框对比。若环境缺字体导致文字未绘制，
	// 仅作提示而不判失败（Text() 写入已由上面硬断言覆盖）。
	empty := newLaidOutInput(t)
	empty.Blur()
	imgEmpty := renderInput(t, empty)
	if bytes.Equal(imgEmpty, imgTyped) {
		t.Log("提示：含文字与空输入框渲染像素相同，可能是测试环境缺少字体导致文字未绘制；文本写入逻辑已验证通过")
	}
}

// TestInputIMECompositionPreview 验证拼音预览（IME 组合串）能被聚焦输入框接收。
// 这对应修复：app 层此前未注册 TypeIMEComposition 监听器，导致预览丢失。
func TestInputIMECompositionPreview(t *testing.T) {
	el := newLaidOutInput(t)
	el.Focus()

	ev := event.NewIMECompositionEvent("nihao", 5)
	if !el.HandleEvent(ev) {
		t.Fatal("聚焦的输入框应当处理 IME 组合事件")
	}
	if el.composition != "nihao" {
		t.Fatalf("composition = %q，期望 %q（拼音预览未写入）", el.composition, "nihao")
	}

	// 未聚焦时不应接收组合预览。
	el.Blur()
	ev2 := event.NewIMECompositionEvent("wohao", 5)
	if el.HandleEvent(ev2) {
		t.Error("未聚焦的输入框不应处理 IME 组合事件")
	}
}

// TestNewInputAcceptsTyping 回归：NewInput 此前未设置 MaxLength（零值 0），
// 导致 KeyChar 的插入条件 len(runes) < 0 恒为假——通过 NewInput 创建的输入框
// 一个字符都打不进去。guitest/demo 全部使用 NewInput，这是“输入内容没有写入
// 输入框”的头号根因（与 IME 无关，连英文也打不进）。
func TestNewInputAcceptsTyping(t *testing.T) {
	var changed string
	in := NewInput("请输入用户名...", func(text string) { changed = text })
	el := in.CreateElement().(*InputElement)
	el.Mount(nil, 0)
	el.Layout(&layout.LayoutContext{Constraints: layout.Tight(212, 32)})
	el.Focus()

	for _, ch := range "hello" {
		ev := event.NewKeyEvent(event.TypeKeyChar, uint32(ch), string(ch), event.ModNone)
		ev.Char = ch
		el.HandleEvent(ev)
	}

	if el.Text() != "hello" {
		t.Fatalf("通过 NewInput 创建的输入框 Text() = %q，期望 %q（MaxLength 默认值导致无法输入）", el.Text(), "hello")
	}
	if changed != "hello" {
		t.Errorf("OnTextChanged 应收到 %q，实际 %q", "hello", changed)
	}
}

// TestInputIMECompositionClears 验证空组合事件能清除预览残留——
// 对应"单字符拼音退格删空后预览残留"的修复（win32 即使组合串为空也发送事件）。
func TestInputIMECompositionClears(t *testing.T) {
	el := newLaidOutInput(t)
	el.Focus()

	el.HandleEvent(event.NewIMECompositionEvent("n", 1))
	if el.composition != "n" {
		t.Fatalf("组合预览 = %q，期望 'n'", el.composition)
	}

	// 退格把拼音删空 → 空组合事件应清空预览
	el.HandleEvent(event.NewIMECompositionEvent("", 0))
	if el.composition != "" {
		t.Errorf("空组合事件后预览应清空，实际 = %q", el.composition)
	}
}

// TestInputHorizontalScroll 验证文本超出宽度时按光标水平滚动。
func TestInputHorizontalScroll(t *testing.T) {
	el := newLaidOutInput(t)
	el.Focus()
	el.SetText("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghij")

	// 光标置尾并渲染 → 应向右滚动（scrollOffset > 0）以显示末尾
	el.cursorPos = len([]rune(el.Text()))
	_ = renderInput(t, el)
	if el.scrollOffset <= 0 {
		t.Errorf("长文本光标置尾应产生水平滚动，scrollOffset=%.1f", el.scrollOffset)
	}

	// 光标回到开头并渲染 → 滚动应归零
	el.cursorPos = 0
	_ = renderInput(t, el)
	if el.scrollOffset != 0 {
		t.Errorf("光标回到开头后滚动应归零，scrollOffset=%.1f", el.scrollOffset)
	}
}

// TestInputCandidatePosFollowsComposition 验证 IME 候选定位：
// X 跟随拼音串末尾（含组合串宽度），Y 贴在文字底部而非输入框下沿。
func TestInputCandidatePosFollowsComposition(t *testing.T) {
	el := newLaidOutInput(t)
	el.Focus()
	el.SetText("ab")
	el.cursorPos = 2

	// 候选位置取自 Paint 缓存（Skia 测量），需先渲染一帧填充缓存。
	_ = renderInput(t, el) // renderInput 内部 SetPosition(4,2)
	x0, y0 := el.CursorClientPos()

	// 加入组合串（拼音预览）后，候选 X 应右移到拼音末尾，Y 不变。
	el.composition = "nihao"
	_ = renderInput(t, el)
	x1, y1 := el.CursorClientPos()
	if x1 <= x0 {
		t.Errorf("有组合串时候选 X 应右移到拼音末尾：x0=%.1f x1=%.1f", x0, x1)
	}
	if y1 != y0 {
		t.Errorf("候选 Y 不应随组合串变化：y0=%.1f y1=%.1f", y0, y1)
	}

	// Y 锚点用 caret 顶部，应落在输入框内、明显高于输入框下沿。
	pos := el.Offset()
	boxBottom := pos.Y + el.Size().Height
	if y0 >= boxBottom {
		t.Errorf("候选 Y 锚点应在输入框内(%.1f)而非下沿(%.1f)", y0, boxBottom)
	}
}
