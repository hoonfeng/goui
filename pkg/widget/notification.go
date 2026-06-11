package widget

import (
	"time"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Notification 通知（1:1 复刻 el-notification）────────────────
// 命令式 API：右上角浮出通知卡片(图标 + 标题 + 内容 + ✕)，默认 4.5s 自动消失，多条堆叠。
// 复用 Overlay toast 模式(ToastPos: top-right) + MessageType/messageStyle 配色。
//
// 用法:
//   widget.NotifySuccess("成功", "数据已保存到云端")
//   widget.NotifyError("错误", "连接服务器失败")

// ShowNotification 右上角弹出通知卡片，duration(<=0 取默认 4.5s)后自动消失，返回浮层 id。
func ShowNotification(title, message string, ntype MessageType, duration time.Duration) int {
	return ShowNotificationWith(title, message, ntype, ToastConfig{Duration: duration})
}

// ShowNotificationWith 带配置(位置/配色/时长)弹出通知卡片。
func ShowNotificationWith(title, message string, ntype MessageType, cfg ToastConfig) int {
	pos := cfg.Position
	if pos == "" {
		pos = "top-right"
	}
	card := &notificationCard{title: title, message: message, ntype: ntype, custom: cfg.colors()}
	var id int
	card.onClose = func() { HideOverlay(id) }
	id = ShowOverlay(&OverlayEntry{Content: card, Toast: true, ToastPos: pos, Transition: toastTransitionFor(pos)})
	dur := cfg.Duration
	if dur <= 0 {
		dur = 4500 * time.Millisecond
	}
	startToastTimer(id, dur)
	return id
}

func NotifySuccess(title, message string) int { return ShowNotification(title, message, MsgSuccess, 0) }
func NotifyError(title, message string) int   { return ShowNotification(title, message, MsgError, 0) }
func NotifyWarning(title, message string) int { return ShowNotification(title, message, MsgWarning, 0) }
func NotifyInfo(title, message string) int    { return ShowNotification(title, message, MsgInfo, 0) }

// NotificationPreview 返回独立通知卡片 widget，仅供 demo/测试预览外观。
func NotificationPreview(title, message string, ntype MessageType) Widget {
	return &notificationCard{title: title, message: message, ntype: ntype}
}

// ─── notificationCard 通知卡片（toast 内容）─────────────────────

type notificationCard struct {
	StatelessWidget
	title   string
	message string
	ntype   MessageType
	onClose func()
	custom  *toastColors // nil → 跟类型默认色
}

func (c *notificationCard) CreateElement() Element {
	return &notificationCardElement{BaseElement: BaseElement{widget: c}, card: c}
}

const (
	notifyW = 330.0
	notifyH = 78.0
)

type notificationCardElement struct {
	BaseElement
	card *notificationCard
}

func (e *notificationCardElement) Build() []Element { return nil }

func (e *notificationCardElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(types.Size{Width: notifyW, Height: notifyH})
	return layout.LayoutResult{Size: e.size}
}

func (e *notificationCardElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height
	_, _, iconColor, iconCh := messageStyle(e.card.ntype)
	cardBg := types.ColorFromRGB(255, 255, 255)
	titleColor := elTextPrimary()
	msgColor := types.ColorFromRGB(108, 110, 115)
	if c := e.card.custom; c != nil {
		if c.bg != nil {
			cardBg = *c.bg
		}
		if c.icon != nil {
			iconColor = *c.icon
		}
		if c.text != nil {
			titleColor = *c.text
			msgColor = *c.text
		}
	}

	// 阴影 + 白底 + 边框
	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 22)
	cvs.DrawRoundedRect(pos.X, pos.Y+3, W, H, 8, shadow)
	bg := paint.DefaultPaint()
	bg.Color = cardBg
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 8, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = elBorderLighter()
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 8, bp)

	// 左图标（类型色圆 + 白符号）
	icx := pos.X + 16 + 11
	icy := pos.Y + 22
	ip := paint.DefaultPaint()
	ip.Color = iconColor
	cvs.DrawCircle(icx, icy, 11, ip)
	ifont := canvas.DefaultFont()
	ifont.Size = 13
	ifont.Weight = canvas.FontWeightBold
	canvas.DrawTextAligned(cvs, iconCh, types.Rect{X: icx - 11, Y: icy - 11, Width: 22, Height: 22}, ifont, types.ColorWhite, canvas.HAlignCenter, canvas.VAlignMiddle)

	textX := pos.X + 16 + 22 + 12
	// 标题（加粗）
	tf := canvas.DefaultFont()
	tf.Size = 15
	tf.Weight = canvas.FontWeightBold
	canvas.DrawTextAligned(cvs, e.card.title, types.Rect{X: textX, Y: pos.Y + 12, Width: W - (textX - pos.X) - 30, Height: 22}, tf, titleColor, canvas.HAlignLeft, canvas.VAlignMiddle)
	// 内容（灰）
	mf := canvas.DefaultFont()
	mf.Size = 14
	canvas.DrawTextAligned(cvs, e.card.message, types.Rect{X: textX, Y: pos.Y + 36, Width: W - (textX - pos.X) - 16, Height: 30}, mf, msgColor, canvas.HAlignLeft, canvas.VAlignMiddle)

	// 右上角 ✕
	xp := paint.DefaultStrokePaint()
	xp.Color = elPlaceholder()
	xp.StrokeWidth = 1.3
	xcx := pos.X + W - 16
	xcy := pos.Y + 16
	cvs.DrawLine(xcx-4, xcy-4, xcx+4, xcy+4, xp)
	cvs.DrawLine(xcx-4, xcy+4, xcx+4, xcy-4, xp)
}

// HandleEvent 点击右上角 ✕ 热区关闭。
func (e *notificationCardElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeMouseDown {
		if me, ok := ev.(*event.MouseEvent); ok {
			pos := e.Offset()
			if me.X >= pos.X+e.size.Width-32 && me.Y <= pos.Y+32 {
				if e.card.onClose != nil {
					e.card.onClose()
				}
				return true
			}
		}
	}
	return false
}

func (e *notificationCardElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*notificationCard); ok {
		e.card = nc
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
