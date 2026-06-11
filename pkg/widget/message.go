package widget

import (
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Message 全局消息（1:1 复刻 el-message）──────────────────────
// 命令式 API：顶部居中浮出一条消息条(success/warning/info/error)，
// 默认 3s 后自动消失；多条向下堆叠。复用 Overlay 的 toast 模式。
//
// 用法:
//   widget.MessageSuccess("保存成功")
//   widget.MessageError("网络错误")
//   id := widget.ShowMessage("自定义", widget.MsgWarning, 5*time.Second)

type MessageType int

const (
	MsgInfo MessageType = iota
	MsgSuccess
	MsgWarning
	MsgError
)

// ToastConfig 是 Message/Notification 的可选配置(零值字段=用默认)。
type ToastConfig struct {
	Duration    time.Duration // 0 → 默认(Message 3s / Notification 4.5s)
	Position    string        // "" → 默认；top / top-left / top-right / bottom-left / bottom-right
	BgColor     *types.Color  // nil → 跟类型(el 标准色)
	BorderColor *types.Color
	IconColor   *types.Color
	TextColor   *types.Color
}

// toastColors 自定义配色(nil 字段回退到类型默认色)。
type toastColors struct {
	bg, border, icon, text *types.Color
}

func (c *ToastConfig) colors() *toastColors {
	if c.BgColor == nil && c.BorderColor == nil && c.IconColor == nil && c.TextColor == nil {
		return nil
	}
	return &toastColors{bg: c.BgColor, border: c.BorderColor, icon: c.IconColor, text: c.TextColor}
}

// toastTransitionFor 按锚位选进场滑入方向。
func toastTransitionFor(pos string) string {
	switch pos {
	case "top-right":
		return "slide-right"
	case "bottom", "bottom-left", "bottom-right":
		return "slide-bottom"
	default: // top, top-left
		return "slide-top"
	}
}

// ShowMessage 顶部居中弹出一条消息，duration(<=0 取默认 3s)后自动消失，返回浮层 id。
func ShowMessage(text string, mtype MessageType, duration time.Duration) int {
	return ShowMessageWith(text, mtype, ToastConfig{Duration: duration})
}

// ShowMessageWith 带配置(位置/配色/时长)弹出消息。
func ShowMessageWith(text string, mtype MessageType, cfg ToastConfig) int {
	pos := cfg.Position
	if pos == "" {
		pos = "top"
	}
	bar := &messageBar{text: text, mtype: mtype, custom: cfg.colors()}
	id := ShowOverlay(&OverlayEntry{Content: bar, Toast: true, ToastPos: pos, Transition: toastTransitionFor(pos)})
	dur := cfg.Duration
	if dur <= 0 {
		dur = 3 * time.Second
	}
	startToastTimer(id, dur)
	return id
}

// startToastTimer 启动 toast 计时：duration 后调 HideOverlay
// (退场的淡出/滑出由 Overlay 的 Transition 统一播放)。Message/Notification 共用。
func startToastTimer(id int, duration time.Duration) {
	ctrl := animation.NewController(duration, nil)
	ctrl.OnDone = func() { HideOverlay(id) }
	ctrl.Start()
}

// 命令式快捷方法（默认 3s 自动消失）。
func MessageSuccess(text string) int { return ShowMessage(text, MsgSuccess, 0) }
func MessageError(text string) int   { return ShowMessage(text, MsgError, 0) }
func MessageWarning(text string) int { return ShowMessage(text, MsgWarning, 0) }
func MessageInfo(text string) int    { return ShowMessage(text, MsgInfo, 0) }

// MessageBarPreview 返回独立消息条 widget，仅供 demo/测试预览外观
// （正常使用经 ShowMessage 弹到 Overlay 顶层）。
func MessageBarPreview(text string, mtype MessageType) Widget {
	return &messageBar{text: text, mtype: mtype}
}

// messageStyle 返回各类型的背景/边框/图标色/图标字符(对齐 el 配色)。
func messageStyle(t MessageType) (bg, border, icon types.Color, ch string) {
	switch t {
	case MsgSuccess:
		return types.ColorFromRGB(240, 249, 235), types.ColorFromRGB(225, 243, 216), elSuccess(), "✓"
	case MsgWarning:
		return types.ColorFromRGB(253, 246, 236), types.ColorFromRGB(250, 236, 216), elWarning(), "!"
	case MsgError:
		return types.ColorFromRGB(254, 240, 240), types.ColorFromRGB(253, 226, 226), elDanger(), "✕"
	default: // info
		return types.ColorFromRGB(244, 244, 245), types.ColorFromRGB(233, 233, 235), elTextSecondary(), "i"
	}
}

// ─── messageBar 消息条（toast 内容）────────────────────────────

type messageBar struct {
	StatelessWidget
	text   string
	mtype  MessageType
	custom *toastColors // nil → 跟类型默认色
}

func (m *messageBar) CreateElement() Element {
	return &messageBarElement{BaseElement: BaseElement{widget: m}, bar: m}
}

type messageBarElement struct {
	BaseElement
	bar *messageBar
}

func (e *messageBarElement) Build() []Element { return nil }

const messageBarH = 42.0

func (e *messageBarElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	font := canvas.DefaultFont()
	font.Size = 14
	textW := canvas.MeasureTextGlobal(e.bar.text, font).Width
	w := 16 + 18 + 8 + textW + 16 // 左padding + 图标 + gap + 文字 + 右padding
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: messageBarH})
	return layout.LayoutResult{Size: e.size}
}

func (e *messageBarElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height
	bg, border, iconColor, iconCh := messageStyle(e.bar.mtype)
	textColor := elTextRegular()
	if c := e.bar.custom; c != nil {
		if c.bg != nil {
			bg = *c.bg
		}
		if c.border != nil {
			border = *c.border
		}
		if c.icon != nil {
			iconColor = *c.icon
		}
		if c.text != nil {
			textColor = *c.text
		}
	}

	// 阴影
	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 16)
	cvs.DrawRoundedRect(pos.X, pos.Y+2, W, H, 4, shadow)
	// 背景 + 边框
	bgp := paint.DefaultPaint()
	bgp.Color = bg
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bgp)
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	// 图标（类型色实心圆 + 白符号）
	icx := pos.X + 16 + 9
	icy := pos.Y + H/2
	ip := paint.DefaultPaint()
	ip.Color = iconColor
	cvs.DrawCircle(icx, icy, 9, ip)
	ifont := canvas.DefaultFont()
	ifont.Size = 12
	ifont.Weight = canvas.FontWeightBold
	canvas.DrawTextAligned(cvs, iconCh, types.Rect{X: icx - 9, Y: pos.Y, Width: 18, Height: H}, ifont, types.ColorWhite, canvas.HAlignCenter, canvas.VAlignMiddle)

	// 文字
	font := canvas.DefaultFont()
	font.Size = 14
	tr := types.Rect{X: pos.X + 16 + 18 + 8, Y: pos.Y, Width: W - (16 + 18 + 8) - 12, Height: H}
	canvas.DrawTextAligned(cvs, e.bar.text, tr, font, textColor, canvas.HAlignLeft, canvas.VAlignMiddle)
}

func (e *messageBarElement) Update(newWidget Widget) {
	if nb, ok := newWidget.(*messageBar); ok {
		e.bar = nb
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
