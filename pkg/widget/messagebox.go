package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/i18n"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── MessageBox 消息弹框（1:1 复刻 el-messagebox）──────────────
// 命令式 API：居中模态确认框。复用 Dialog 模态体系(ShowDialog) + MessageType 类型图标。
//   ShowAlert   单「确定」按钮(告知)
//   ShowConfirm 「取消 / 确定」双按钮(确认)
//
// 用法:
//   widget.ShowConfirm("提示", "确定删除吗？", widget.MsgWarning,
//       func(){ /* 确定 */ }, func(){ /* 取消 */ })

// ShowAlert 弹出告知框(单「确定」按钮)，点击后回调 onOk。
func ShowAlert(title, message string, mtype MessageType, onOk func()) int {
	var id int
	dialog := NewDialog(title, messageBoxBody(message, mtype)).WithWidth(400).WithFooter(
		NewButton(i18n.T("el.messagebox.confirm"), func() {
			HideOverlay(id)
			if onOk != nil {
				onOk()
			}
		}).WithColor(elPrimary()),
	)
	id = ShowDialog(dialog)
	return id
}

// ShowConfirm 弹出确认框(「取消 / 确定」)，分别回调 onCancel / onConfirm。
func ShowConfirm(title, message string, mtype MessageType, onConfirm, onCancel func()) int {
	var id int
	dialog := NewDialog(title, messageBoxBody(message, mtype)).WithWidth(400).WithFooter(
		NewButton(i18n.T("el.messagebox.cancel"), func() {
			HideOverlay(id)
			if onCancel != nil {
				onCancel()
			}
		}).WithColor(types.ColorFromRGB(244, 244, 245)).WithTextColor(elTextRegular()),
		NewButton(i18n.T("el.messagebox.confirm"), func() {
			HideOverlay(id)
			if onConfirm != nil {
				onConfirm()
			}
		}).WithColor(elPrimary()),
	)
	id = ShowDialog(dialog)
	return id
}

// MessageBoxBodyPreview 返回 MessageBox 内容区 widget(图标 + 文字)，仅供 demo/测试。
func MessageBoxBodyPreview(message string, mtype MessageType) Widget {
	return messageBoxBody(message, mtype)
}

// messageBoxBody 内容区：类型图标 + 文字横排。
func messageBoxBody(message string, mtype MessageType) Widget {
	_, _, iconColor, iconCh := messageStyle(mtype)
	return divCore(
		Style{FlexDirection: "row", AlignItems: "center", Gap: 12},
		[]Widget{
			&messageBoxIcon{color: iconColor, ch: iconCh},
			&Text{
				Text:  message,
				Color: elTextRegular(),
				Font:  canvas.Font{Family: "sans-serif", Size: 14},
			},
		},
	)
}

// ─── messageBoxIcon 类型图标（圆 + 符号）────────────────────────

type messageBoxIcon struct {
	StatelessWidget
	color types.Color
	ch    string
}

func (i *messageBoxIcon) CreateElement() Element {
	return &messageBoxIconElement{BaseElement: BaseElement{widget: i}, icon: i}
}

type messageBoxIconElement struct {
	BaseElement
	icon *messageBoxIcon
}

func (e *messageBoxIconElement) Build() []Element { return nil }

func (e *messageBoxIconElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(types.Size{Width: 28, Height: 28})
	return layout.LayoutResult{Size: e.size}
}

func (e *messageBoxIconElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	cx := pos.X + 14
	cy := pos.Y + 14
	ip := paint.DefaultPaint()
	ip.Color = e.icon.color
	cvs.DrawCircle(cx, cy, 13, ip)
	f := canvas.DefaultFont()
	f.Size = 15
	f.Weight = canvas.FontWeightBold
	canvas.DrawTextAligned(cvs, e.icon.ch, types.Rect{X: pos.X, Y: pos.Y, Width: 28, Height: 28}, f, types.ColorWhite, canvas.HAlignCenter, canvas.VAlignMiddle)
}

func (e *messageBoxIconElement) Update(newWidget Widget) {
	if ni, ok := newWidget.(*messageBoxIcon); ok {
		e.icon = ni
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
