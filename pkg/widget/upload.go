package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Upload 上传（el-upload 拖拽区 UI）──────────────────────────
// 虚线拖拽区(点击/hover，自绘合理) + 已选文件列表(拼装：每行复用 Text + 删除 Link)。
// 点击拖拽区触发 OnUpload；点文件行 ✕ 触发 OnRemove(index)。
// (goui 暂无文件对话框，文件列表由调用方传入展示。)

type Upload struct {
	StatelessWidget
	Files    []string // 已选文件名（展示）
	Tip      string
	OnUpload func()
	OnRemove func(index int) // 删除某文件
}

func NewUpload() *Upload                            { return &Upload{} }
func (u *Upload) WithFiles(files ...string) *Upload { u.Files = files; return u }
func (u *Upload) WithOnUpload(fn func()) *Upload    { u.OnUpload = fn; return u }
func (u *Upload) WithOnRemove(fn func(int)) *Upload { u.OnRemove = fn; return u }

func (u *Upload) CreateElement() Element {
	return &UploadElement{BaseElement: BaseElement{widget: u}, upload: u}
}

const (
	uploadW     = 420.0
	uploadDropH = 120.0
	uploadFileH = 34.0
)

type UploadElement struct {
	BaseElement
	upload  *Upload
	hover   bool
	fileEls []Element
}

// uploadFileRow 拼装一行已选文件：📄 文件名 … ✓ 状态 + ✕ 删除。
func uploadFileRow(name string, onRemove func()) Widget {
	kids := []Widget{
		&Text{Text: "📄  " + name, Color: elTextRegular(), Font: canvas.Font{Family: "sans-serif", Size: 14}},
		SpacerDiv(),
		&Text{Text: "✓", Color: elSuccess(), Font: canvas.Font{Family: "sans-serif", Size: 14}},
	}
	if onRemove != nil {
		kids = append(kids, NewLink("✕", onRemove))
	}
	return divCore(
		Style{Padding: types.EdgeInsetsLTRB(8, 0, 10, 0), FlexDirection: "row", AlignItems: "center", Gap: 10, Height: uploadFileH},
		kids,
	)
}

func (e *UploadElement) Build() []Element {
	for _, el := range e.fileEls {
		el.Unmount()
	}
	e.fileEls = e.fileEls[:0]
	for i, name := range e.upload.Files {
		idx := i
		el := CreateElementFor(uploadFileRow(name, func() {
			if e.upload.OnRemove != nil {
				e.upload.OnRemove(idx)
			}
		}))
		el.Mount(e, i)
		e.fileEls = append(e.fileEls, el)
	}
	e.children = e.fileEls
	return e.children
}

func (e *UploadElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	totalH := uploadDropH
	for i, el := range e.fileEls {
		fy := uploadDropH + 10 + float64(i)*uploadFileH
		el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MinWidth: uploadW, MaxWidth: uploadW, MinHeight: 0, MaxHeight: uploadFileH,
		}})
		el.SetPosition(types.Point{X: 0, Y: fy})
		totalH = fy + uploadFileH
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: uploadW, Height: totalH})
	return layout.LayoutResult{Size: e.size}
}

func (e *UploadElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()

	// 拖拽区：虚线边框 + 圆角（hover 主色）—— 自绘合理
	border := elBorder()
	if e.hover {
		border = elPrimary()
	}
	bg := paint.DefaultPaint()
	bg.Color = types.ColorFromRGB(250, 251, 252)
	if e.hover {
		bg.Color = types.ColorFromRGB(236, 245, 255)
	}
	cvs.DrawRoundedRect(pos.X, pos.Y, uploadW, uploadDropH, 6, bg)
	dashRoundedRect(cvs, pos.X+0.5, pos.Y+0.5, uploadW-1, uploadDropH-1, border)

	icf := canvas.DefaultFont()
	icf.Size = 34
	canvas.DrawTextAligned(cvs, "⬆", types.Rect{X: pos.X, Y: pos.Y + 18, Width: uploadW, Height: 44},
		icf, elPlaceholder(), canvas.HAlignCenter, canvas.VAlignMiddle)
	tf := canvas.DefaultFont()
	tf.Size = 14
	canvas.DrawTextAligned(cvs, orI18n(e.upload.Tip, "el.upload.tip"), types.Rect{X: pos.X, Y: pos.Y + 70, Width: uploadW, Height: 30},
		tf, elTextRegular(), canvas.HAlignCenter, canvas.VAlignMiddle)

	// 文件列表（拼装的子行）
	for _, el := range e.fileEls {
		el.Paint(cvs, offset)
	}
}

// dashRoundedRect 画虚线圆角矩形边框（上下左右四边分段）。
func dashRoundedRect(cvs canvas.Canvas, x, y, w, h float64, color types.Color) {
	p := paint.DefaultStrokePaint()
	p.Color = color
	p.StrokeWidth = 1.2
	const dash, gap = 6.0, 4.0
	for dx := x + 8; dx < x+w-8; dx += dash + gap {
		end := dx + dash
		if end > x+w-8 {
			end = x + w - 8
		}
		cvs.DrawLine(dx, y, end, y, p)
		cvs.DrawLine(dx, y+h, end, y+h, p)
	}
	for dy := y + 8; dy < y+h-8; dy += dash + gap {
		end := dy + dash
		if end > y+h-8 {
			end = y + h - 8
		}
		cvs.DrawLine(x, dy, x, end, p)
		cvs.DrawLine(x+w, dy, x+w, end, p)
	}
}

func (e *UploadElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	inDrop := me.Y-e.Offset().Y < uploadDropH
	switch ev.Type() {
	case event.TypeMouseEnter, event.TypeMouseMove:
		if inDrop != e.hover {
			e.hover = inDrop
			e.MarkNeedsPaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hover = false
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if inDrop && e.upload.OnUpload != nil {
			e.upload.OnUpload()
			return true
		}
	}
	return false
}

func (e *UploadElement) Update(newWidget Widget) {
	if nu, ok := newWidget.(*Upload); ok {
		e.upload = nu
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
