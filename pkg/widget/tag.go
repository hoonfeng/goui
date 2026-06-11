package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Tag 标签控件 ──────────────────────────────────────────────
// 1:1 复刻 Element Plus 的 el-tag：
//   props : type / effect / size / closable / round / hit / color / disable-transitions
//   events: click / close
//   slot  : default（即 Text 内容）

// TagType 标签的语义类型，对应 el-tag 的 type 属性。
type TagType int

const (
	TagPrimary TagType = iota // primary（默认）
	TagSuccess                // success
	TagInfo                   // info
	TagWarning                // warning
	TagDanger                 // danger
)

// TagEffect 标签的主题风格，对应 el-tag 的 effect 属性。
type TagEffect int

const (
	TagLight TagEffect = iota // light（默认）：浅色背景 + 主色文字 + 浅边框
	TagDark                   // dark：实色背景 + 白色文字
	TagPlain                  // plain：白色背景 + 主色文字 + 主色描边
)

// TagSize 标签尺寸，对应 el-tag 的 size 属性。
type TagSize int

const (
	TagSizeDefault TagSize = iota // default（高 24）
	TagSizeLarge                  // large（高 32）
	TagSizeSmall                  // small（高 20）
)

// Tag 标签控件。
type Tag struct {
	StatelessWidget
	Text               string
	Type               TagType
	Effect             TagEffect
	Size               TagSize
	Closable           bool        // 是否显示关闭按钮（×）
	Round              bool        // 是否圆角胶囊（border-radius = 高度/2）
	Hit                bool        // 是否描边（让 dark 也带主色边框）
	Color              types.Color // 自定义背景色（覆盖 type 配色），零值表示不启用
	DisableTransitions bool        // 是否禁用过渡动画（保留以对齐 API）
	OnClick            func()      // 点击标签主体触发
	OnClose            func()      // 点击关闭按钮触发
}

// NewTag 创建默认（primary / light / default）标签。
func NewTag(text string) *Tag {
	return &Tag{Text: text, Type: TagPrimary, Effect: TagLight, Size: TagSizeDefault}
}

// 链式 setter ──────────────────────────────────────────────────
func (t *Tag) WithType(tp TagType) *Tag    { t.Type = tp; return t }
func (t *Tag) WithEffect(e TagEffect) *Tag  { t.Effect = e; return t }
func (t *Tag) WithSize(s TagSize) *Tag      { t.Size = s; return t }
func (t *Tag) WithClosable(b bool) *Tag     { t.Closable = b; return t }
func (t *Tag) WithRound(b bool) *Tag        { t.Round = b; return t }
func (t *Tag) WithHit(b bool) *Tag          { t.Hit = b; return t }
func (t *Tag) WithColor(c types.Color) *Tag { t.Color = c; return t }
func (t *Tag) OnClickFn(fn func()) *Tag     { t.OnClick = fn; return t }
func (t *Tag) OnCloseFn(fn func()) *Tag     { t.OnClose = fn; return t }

// hasCustomColor 报告用户是否设置了自定义背景色。
func (t *Tag) hasCustomColor() bool { return t.Color != (types.Color{}) }

// baseColor 返回当前 type 的主色（自定义 Color 优先）。
func (t *Tag) baseColor() types.Color {
	if t.hasCustomColor() {
		return t.Color
	}
	tg := CurrentTheme().Tag
	switch t.Type {
	case TagSuccess:
		return tg.SuccessColor
	case TagInfo:
		return tg.InfoColor
	case TagWarning:
		return tg.WarningColor
	case TagDanger:
		return tg.DangerColor
	default:
		return tg.PrimaryColor
	}
}

// CreateElement 创建 TagElement。
func (t *Tag) CreateElement() Element {
	return &TagElement{BaseElement: BaseElement{widget: t}, tag: t}
}

// TagElement 标签运行时 Element。
type TagElement struct {
	BaseElement
	tag          *Tag
	hovered      bool
	closeHovered bool
	pressed      bool
}

// tagSizeMetrics 按尺寸返回 (高度, 水平内边距, 字号, 关闭图标尺寸, 文字与关闭按钮间距)。
func tagSizeMetrics(s TagSize) (h, padH, fontSize, closeSize, gap float64) {
	switch s {
	case TagSizeLarge:
		return 32, 11, 14, 16, 6
	case TagSizeSmall:
		return 20, 7, 12, 12, 4
	default:
		return 24, 9, 12, 14, 5
	}
}

// tagMix 把 base 与当前主题背景按 base 占比 t 混合（t=1 纯 base，t=0 纯背景）。
// 混背景(elSurface)而非硬编码白，使 light-tint(Tag light/Alert/Message…) 深浅主题自适应。
func tagMix(base types.Color, t float64) types.Color {
	bgC := elSurface()
	return types.Color{
		R: uint8(float64(base.R)*t + float64(bgC.R)*(1-t)),
		G: uint8(float64(base.G)*t + float64(bgC.G)*(1-t)),
		B: uint8(float64(base.B)*t + float64(bgC.B)*(1-t)),
		A: 255,
	}
}

// resolveColors 按 effect 返回 (背景, 边框, 文字) 三色。
func (e *TagElement) resolveColors() (bg, border, text types.Color) {
	t := e.tag
	base := t.baseColor()
	switch t.Effect {
	case TagDark:
		return base, base, types.ColorWhite
	case TagPlain:
		return types.ColorWhite, tagMix(base, 0.5), base
	default: // light
		if t.hasCustomColor() {
			return tagMix(base, 0.2), tagMix(base, 0.4), base
		}
		return tagMix(base, 0.1), tagMix(base, 0.2), base
	}
}

// Build 叶子节点，无子元素。
func (e *TagElement) Build() []Element { return nil }

// Layout 测量标签尺寸：水平 = padH + 文字宽 + (关闭按钮) + padH，高度由尺寸决定。
func (e *TagElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	t := e.tag
	h, padH, fontSize, closeSize, gap := tagSizeMetrics(t.Size)

	font := canvas.DefaultFont()
	font.Size = fontSize
	textW := canvas.MeasureTextGlobal(t.Text, font).Width

	w := padH + textW + padH
	if t.Closable {
		w += gap + closeSize
	}

	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

// closeCenter 返回关闭按钮的圆心坐标与半径（全局坐标）。
func (e *TagElement) closeCenter() (cx, cy, cr float64) {
	pos := e.Offset()
	_, padH, _, closeSize, _ := tagSizeMetrics(e.tag.Size)
	cr = closeSize / 2
	cx = pos.X + e.size.Width - padH - cr
	cy = pos.Y + e.size.Height/2
	return
}

// inCloseArea 判断全局坐标 (x,y) 是否落在关闭按钮命中区（略放宽便于点击）。
func (e *TagElement) inCloseArea(x, y float64) bool {
	cx, cy, cr := e.closeCenter()
	dx, dy := x-cx, y-cy
	r := cr + 3
	return dx*dx+dy*dy <= r*r
}

// Paint 绘制标签：圆角背景 → 边框 → 文字 → 关闭按钮。
func (e *TagElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	t := e.tag
	_, padH, fontSize, _, _ := tagSizeMetrics(t.Size)
	bg, border, textColor := e.resolveColors()
	base := t.baseColor()

	radius := CurrentTheme().Tag.BorderRadius
	if t.Round {
		radius = e.size.Height / 2
	}

	// 背景
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = bg
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, radius, bgPaint)

	// 边框：plain / light 始终描边；dark 仅 hit 时描边。hit 用主色加深。
	if t.Effect == TagPlain || t.Effect == TagLight || t.Hit {
		bp := paint.DefaultStrokePaint()
		if t.Hit {
			bp.Color = base
		} else {
			bp.Color = border
		}
		bp.StrokeWidth = 1
		cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, e.size.Width-1, e.size.Height-1, radius, bp)
	}

	// 文字（左对齐于内边距 + 垂直居中）—— 复用统一对齐绘制原语
	font := canvas.DefaultFont()
	font.Size = fontSize
	textRect := types.Rect{X: pos.X + padH, Y: pos.Y, Width: e.size.Width - 2*padH, Height: e.size.Height}
	canvas.DrawTextAligned(cvs, t.Text, textRect, font, textColor, canvas.HAlignLeft, canvas.VAlignMiddle)

	// 关闭按钮 ×
	if t.Closable {
		cx, cy, cr := e.closeCenter()
		if e.closeHovered {
			// 悬停：主色圆底 + 反白 ×
			circle := paint.DefaultPaint()
			circle.Color = textColor
			cvs.DrawCircle(cx, cy, cr, circle)
		}
		xp := paint.DefaultStrokePaint()
		if e.closeHovered {
			xp.Color = bg
		} else {
			xp.Color = textColor
		}
		xp.StrokeWidth = 1.2
		d := cr * 0.45
		cvs.DrawLine(cx-d, cy-d, cx+d, cy+d, xp)
		cvs.DrawLine(cx-d, cy+d, cx+d, cy-d, xp)
	}
}

// HandleEvent 处理鼠标交互：hover / 关闭按钮命中 / click·close 分发。
func (e *TagElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		tagRepaint()
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.closeHovered = false
		e.pressed = false
		tagRepaint()
		return true
	case event.TypeMouseMove:
		if me, ok := ev.(*event.MouseEvent); ok && e.tag.Closable {
			was := e.closeHovered
			e.closeHovered = e.inCloseArea(me.X, me.Y)
			if was != e.closeHovered {
				tagRepaint()
			}
		}
		return true
	case event.TypeMouseDown:
		e.pressed = true
		return true
	case event.TypeMouseUp:
		if e.pressed {
			if me, ok := ev.(*event.MouseEvent); ok && e.tag.Closable && e.inCloseArea(me.X, me.Y) {
				if e.tag.OnClose != nil {
					e.tag.OnClose()
				}
			} else if e.tag.OnClick != nil {
				e.tag.OnClick()
			}
		}
		e.pressed = false
		tagRepaint()
		return true
	}
	return false
}

// Update 同步新的 Widget 配置。
func (e *TagElement) Update(newWidget Widget) {
	if nt, ok := newWidget.(*Tag); ok {
		e.tag = nt
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// tagRepaint 触发一次重绘（hover/关闭态变化只需重绘，不重建/布局）。
func tagRepaint() {
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}
