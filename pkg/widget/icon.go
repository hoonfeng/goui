package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// IconType 图标类型枚举
type IconType int

const (
	// 导航与操作
	IconArrowUp      IconType = iota // ▲
	IconArrowDown                    // ▼
	IconArrowLeft                    // ◀
	IconArrowRight                   // ▶
	IconChevronUp                    // ⌃
	IconChevronDown                  // ⌄
	IconChevronLeft                  // ‹
	IconChevronRight                 // ›
	IconClose                        // ✕
	IconCheck                        // ✓
	IconPlus                         // +
	IconMinus                        // −
	IconMenu                         // ☰
	IconSearch                       // 🔍 使用文本 "⌕"
	IconSettings                     // ⚙
	IconHome                         // ⌂
	IconStar                         // ★
	IconStarOutline                  // ☆
	IconHeart                        // ♥
	IconInfo                         // ⓘ
	IconWarning                      // ⚠
	IconError                        // ✖
	IconFolder                       // ▸
	IconFolderOpen                   // ▾
	IconFile                         // 📄 → 用文本
	IconEdit                         // ✎
	IconTrash                        // 🗑 → 用文本
	IconMail                         // ✉
	IconUser                         // 👤 → 用文本
	IconClock                        // 🕐 → 用文本
	IconCalendar                     // 📅 → 用文本
)

// iconChars 图标类型到 Unicode 字符的映射
var iconChars = map[IconType]string{
	IconArrowUp:      "\u25B2", // ▲
	IconArrowDown:    "\u25BC", // ▼
	IconArrowLeft:    "\u25C0", // ◀
	IconArrowRight:   "\u25B6", // ▶
	IconChevronUp:    "\u2303", // ⌃
	IconChevronDown:  "\u2304", // ⌄
	IconChevronLeft:  "\u2039", // ‹
	IconChevronRight: "\u203A", // ›
	IconClose:        "\u2715", // ✕
	IconCheck:        "\u2713", // ✓
	IconPlus:         "+",
	IconMinus:        "\u2212", // −
	IconMenu:         "\u2630", // ☰
	IconSearch:       "\u2315", // ⌕
	IconSettings:     "\u2699", // ⚙
	IconHome:         "\u2302", // ⌂
	IconStar:         "\u2605", // ★
	IconStarOutline:  "\u2606", // ☆
	IconHeart:        "\u2665", // ♥
	IconInfo:         "\u24D8", // ⓘ
	IconWarning:      "\u26A0", // ⚠
	IconError:        "\u2716", // ✖
	IconFolder:       "\u25B8", // ▸
	IconFolderOpen:   "\u25BE", // ▾
	IconFile:         "\u2756", // ❖
	IconEdit:         "\u270E", // ✎
	IconTrash:        "\u2717", // ✗
	IconMail:         "\u2709", // ✉
	IconUser:         "\u263A", // ☺
	IconClock:        "\u23F0", // ⏰ (some systems)
	IconCalendar:     "\u23F3", // ⏳ (some systems)
}

// Icon 图标控件，使用 Unicode 符号渲染常见图标。
// 不依赖外部图标字体，所有图标使用系统字体中的 Unicode 字符。
type Icon struct {
	StatelessWidget
	Icon  IconType    // 图标类型
	Size  float64     // 图标尺寸（字体大小）
	Color types.Color // 图标颜色
	Font  canvas.Font // 字体（可选）
}

// DefaultIcon 返回默认配置的图标
func DefaultIcon() *Icon {
	return &Icon{
		Icon:  IconStar,
		Size:  24,
		Color: elTextPrimary(),
	}
}

// CreateElement 创建 IconElement
func (ic *Icon) CreateElement() Element {
	return &IconElement{
		BaseElement: BaseElement{widget: ic},
		icon:        ic,
	}
}

// IconElement 图标的运行时 Element
type IconElement struct {
	BaseElement
	icon *Icon
}

// Build 返回空（无子控件）
func (e *IconElement) Build() []Element { return nil }

// Layout 测量图标尺寸
func (e *IconElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	ic := e.icon
	size := ic.Size
	if size <= 0 {
		size = 24
	}

	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  size * 1.2,
		Height: size * 1.2,
	})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制图标
func (e *IconElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	ic := e.icon

	char, ok := iconChars[ic.Icon]
	if !ok {
		return
	}

	font := ic.Font
	if font.Size <= 0 {
		font = canvas.DefaultFont()
	}
	font.Size = ic.Size

	textPaint := paint.DefaultPaint()
	textPaint.Color = ic.Color

	// 居中对齐
	metrics := cvs.MeasureText(char, font)
	textX := pos.X + (e.size.Width-metrics.Width)/2
	textY := pos.Y + (e.size.Height-metrics.Height)/2 + metrics.Ascent

	cvs.DrawText(char, textX, textY, font, textPaint)
}

// GetIconChar 返回指定图标类型的 Unicode 字符
func GetIconChar(typ IconType) string {
	if ch, ok := iconChars[typ]; ok {
		return ch
	}
	return "?"
}
