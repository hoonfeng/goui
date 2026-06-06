package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Menu 菜单（对应 Element Plus el-menu，上下文菜单形态）──────────
// 响应 TypeContextMenu 弹出；支持多级子菜单：MenuItem.Children 非空的项画 ▶，
// hover 或点击时在父项右侧(PlacementRightStart)经 Overlay 级联弹出，递归任意层级。

// MenuItem 表示一个菜单项。
type MenuItem struct {
	Label     string
	OnClick   func()
	Enabled   bool
	Children  []MenuItem // 子菜单（hover/点击右侧级联弹出）
	Icon      string     // 可选左侧 Lucide 图标名（向量渲染，非字符）
	Shortcut  string     // 可选右侧快捷键提示（如 "Ctrl+N"，灰显）
	Separator bool       // 分隔行：画一条分组线，不可点/不高亮
	Danger    bool       // 危险项：红字、hover 红底（删除等）
}

// NewMenuItem 创建菜单项。
func NewMenuItem(label string, onClick func()) *MenuItem {
	return &MenuItem{
		Label:   label,
		OnClick: onClick,
		Enabled: true,
	}
}

// Menu 上下文菜单组件，响应 TypeContextMenu 事件弹出。
type Menu struct {
	StatelessWidget
	Items       []MenuItem  // 菜单项列表
	TextColor   types.Color // 文字颜色
	BGColor     types.Color // 背景色
	HoverColor  types.Color // 悬停项背景色
	BorderColor types.Color // 边框 + 分隔线色（零值=默认 el 调色板；深色主题菜单设此）
	MutedColor  types.Color // 快捷键提示等次要文字色（零值=默认）
	Font        canvas.Font // 字体
	onClose     func()      // 关闭回调（由创建者设置，用于通知应用层移除菜单）
}

// 全局菜单配色（零值=默认 el 浅色）。由 SetMenuTheme 设置，统一上下文菜单(Menu / CodeEditor 右键)
// 与下拉(Dropdown) 的外观——深色应用调一次即可，免逐处传色。
var (
	menuThemeBG     types.Color
	menuThemeText   types.Color
	menuThemeHover  types.Color
	menuThemeBorder types.Color
	menuThemeMuted  types.Color
)

// SetMenuTheme 设置全局菜单配色（深色应用启动时调一次）。任一为零值则该项用默认。
func SetMenuTheme(bg, text, hover, border, muted types.Color) {
	menuThemeBG, menuThemeText, menuThemeHover, menuThemeBorder, menuThemeMuted = bg, text, hover, border, muted
}

// NewMenu 创建上下文菜单。
func NewMenu(items ...MenuItem) *Menu {
	f := canvas.DefaultFont()
	f.Size = 13 // 紧凑菜单字号（上下文菜单/下拉，贴近参考 12px）
	m := &Menu{
		Items:      items,
		TextColor:  elTextRegular(),
		BGColor:    types.ColorFromRGB(255, 255, 255),
		HoverColor: elFill(),
		Font:       f,
	}
	if menuThemeBG != (types.Color{}) {
		m.BGColor = menuThemeBG
	}
	if menuThemeText != (types.Color{}) {
		m.TextColor = menuThemeText
	}
	if menuThemeHover != (types.Color{}) {
		m.HoverColor = menuThemeHover
	}
	if menuThemeBorder != (types.Color{}) {
		m.BorderColor = menuThemeBorder
	}
	if menuThemeMuted != (types.Color{}) {
		m.MutedColor = menuThemeMuted
	}
	return m
}

// menuDanger 危险项红色（#f85149，复刻参考 --danger）。
var menuDanger = types.ColorFromRGB(248, 81, 73)

// 菜单尺寸常量（紧凑：贴近 el / 参考上下文菜单观感）。
const (
	menuPadV     = 4.0  // 上下内边距
	menuItemPadH = 12.0 // 项左右内边距
	menuIconCol  = 20.0 // 图标列宽（有图标时）
	menuSepH     = 8.0  // 分隔行高
	menuIconSize = 13.0
)

// CreateElement 创建 MenuElement
func (m *Menu) CreateElement() Element {
	return &MenuElement{
		BaseElement: BaseElement{widget: m},
		menu:        m,
		hoverIndex:  -1,
		submenuOpen: -1,
	}
}

// MenuElement 菜单的运行时 Element
type MenuElement struct {
	BaseElement
	menu        *Menu
	hoverIndex  int // -1 表示无悬停
	submenuID   int // 已展开子菜单的 overlay id
	submenuOpen int // 已展开子菜单的项索引，-1 表示无
}

// Build 返回空（无子控件；子菜单经 Overlay 弹出）
func (e *MenuElement) Build() []Element { return nil }

func (e *MenuElement) font() canvas.Font {
	f := e.menu.Font
	if f.Size <= 0 {
		f = canvas.DefaultFont()
	}
	return f
}

// rowHeight 普通项行高（紧凑）。
func (e *MenuElement) rowHeight() float64 {
	if ih := e.font().Size*1.45 + 5; ih > 26 {
		return ih
	}
	return 26
}

// itemH 第 i 项高度（分隔行更矮）。
func (e *MenuElement) itemH(i int) float64 {
	if i >= 0 && i < len(e.menu.Items) && e.menu.Items[i].Separator {
		return menuSepH
	}
	return e.rowHeight()
}

// offsetOf 第 i 项顶相对内容区（不含上 padding）的 Y。
func (e *MenuElement) offsetOf(i int) float64 {
	y := 0.0
	for k := 0; k < i && k < len(e.menu.Items); k++ {
		y += e.itemH(k)
	}
	return y
}

// hasIcons 是否有任何项带图标（决定是否预留图标列对齐标签）。
func (e *MenuElement) hasIcons() bool {
	for _, it := range e.menu.Items {
		if !it.Separator && it.Icon != "" {
			return true
		}
	}
	return false
}

func (e *MenuElement) mutedColor() types.Color {
	if e.menu.MutedColor != (types.Color{}) {
		return e.menu.MutedColor
	}
	return elPlaceholder()
}

// Layout 测量菜单尺寸
func (e *MenuElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	m := e.menu
	font := e.font()
	iconCol := 0.0
	if e.hasIcons() {
		iconCol = menuIconCol
	}
	maxLabel, maxShort := 0.0, 0.0
	hasChild := false
	for _, it := range m.Items {
		if it.Separator {
			continue
		}
		if w := canvas.MeasureTextGlobal(it.Label, font).Width; w > maxLabel {
			maxLabel = w
		}
		if it.Shortcut != "" {
			if w := canvas.MeasureTextGlobal(it.Shortcut, font).Width; w > maxShort {
				maxShort = w
			}
		}
		if len(it.Children) > 0 {
			hasChild = true
		}
	}
	w := menuItemPadH + iconCol + maxLabel + menuItemPadH
	if maxShort > 0 {
		w += maxShort + 24 // 快捷键列 + 间距
	}
	if hasChild {
		w += 14 // ▶ 箭头
	}
	if w < 180 {
		w = 180
	}
	h := 2*menuPadV + e.offsetOf(len(m.Items))
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制菜单
func (e *MenuElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	m := e.menu
	font := e.font()
	iconCol := 0.0
	if e.hasIcons() {
		iconCol = menuIconCol
	}
	muted := e.mutedColor()
	ascent := canvas.GetFaceAscent(font.Size)

	// 背景 + 边框（圆角 6）
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = m.BGColor
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, 6, bgPaint)
	borderCol := elBorder()
	if m.BorderColor != (types.Color{}) {
		borderCol = m.BorderColor
	}
	borderPaint := paint.DefaultStrokePaint()
	borderPaint.Color = borderCol
	borderPaint.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, 6, borderPaint)

	y := pos.Y + menuPadV
	for i, item := range m.Items {
		ih := e.itemH(i)
		if item.Separator { // 分组分隔线
			lp := paint.DefaultStrokePaint()
			lp.Color = borderCol
			lp.StrokeWidth = 1
			ly := y + ih/2
			cvs.DrawLine(pos.X+8, ly, pos.X+e.size.Width-8, ly, lp)
			y += ih
			continue
		}
		hovered := (i == e.hoverIndex || i == e.submenuOpen) && item.Enabled
		textCol := m.TextColor
		if item.Danger {
			textCol = menuDanger
		}
		if !item.Enabled {
			textCol = muted
		}
		if hovered { // hover：高亮底 + 白字（危险项红底）
			hb := paint.DefaultPaint()
			hb.Color = m.HoverColor
			if item.Danger {
				hb.Color = menuDanger
			}
			cvs.DrawRoundedRect(pos.X+4, y+1, e.size.Width-8, ih-2, 4, hb)
			textCol = types.ColorWhite
		}
		// 左图标（Lucide 向量）
		if iconCol > 0 && item.Icon != "" {
			PaintLucide(cvs, item.Icon, pos.X+menuItemPadH, y+(ih-menuIconSize)/2, menuIconSize, 2, textCol)
		}
		// 标签
		tp := paint.DefaultPaint()
		tp.Color = textCol
		textY := y + ih/2 + ascent/2
		cvs.DrawText(item.Label, pos.X+menuItemPadH+iconCol, textY, font, tp)
		// 右侧快捷键
		if item.Shortcut != "" {
			scCol := muted
			if hovered {
				scCol = types.ColorWhite
			}
			sp := paint.DefaultPaint()
			sp.Color = scCol
			sw := canvas.MeasureTextGlobal(item.Shortcut, font).Width
			cvs.DrawText(item.Shortcut, pos.X+e.size.Width-menuItemPadH-sw, textY, font, sp)
		}
		// 子菜单 ▶ 箭头
		if len(item.Children) > 0 {
			ap := paint.DefaultStrokePaint()
			ap.Color = textCol
			ap.StrokeWidth = 1.3
			ax := pos.X + e.size.Width - 12
			ay := y + ih/2
			cvs.DrawLine(ax-2, ay-4, ax+2, ay, ap)
			cvs.DrawLine(ax+2, ay, ax-2, ay+4, ap)
		}
		y += ih
	}
}

// openSubmenu 在第 index 项右侧弹出子菜单（经 Overlay，递归支持多级）。
func (e *MenuElement) openSubmenu(index int) {
	if e.submenuOpen == index {
		return
	}
	e.closeSubmenu()
	if index < 0 || index >= len(e.menu.Items) {
		return
	}
	item := e.menu.Items[index]
	if len(item.Children) == 0 {
		return
	}
	pos := e.Offset()
	itemY := pos.Y + menuPadV + e.offsetOf(index)
	sub := NewMenu(item.Children...)
	sub.BGColor, sub.TextColor, sub.HoverColor = e.menu.BGColor, e.menu.TextColor, e.menu.HoverColor
	sub.BorderColor, sub.MutedColor, sub.Font = e.menu.BorderColor, e.menu.MutedColor, e.menu.Font
	sub.onClose = e.menu.onClose // 子项点击关闭整条菜单链
	e.submenuID = ShowOverlay(&OverlayEntry{
		AnchorRect: types.Rect{X: pos.X, Y: itemY, Width: e.size.Width, Height: e.itemH(index)},
		Content:    sub,
		Placement:  PlacementRightStart,
		Gap:        -2,
		Transition: "fade",
	})
	e.submenuOpen = index
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

func (e *MenuElement) closeSubmenu() {
	if e.submenuOpen >= 0 {
		HideOverlay(e.submenuID)
		e.submenuOpen = -1
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
	}
}

// HandleEvent 处理鼠标事件（悬停高亮、级联子菜单、点击触发）
func (e *MenuElement) HandleEvent(ev event.Event) bool {
	m := e.menu

	// indexAt 按累计高度把 y 映射到项索引；分隔行返回 -1（不可点/不高亮）。
	indexAt := func(y float64) int {
		rel := y - (e.Offset().Y + menuPadV)
		if rel < 0 {
			return -1
		}
		acc := 0.0
		for i := range m.Items {
			h := e.itemH(i)
			if rel >= acc && rel < acc+h {
				if m.Items[i].Separator {
					return -1
				}
				return i
			}
			acc += h
		}
		return -1
	}

	switch ev.Type() {
	case event.TypeMouseMove:
		mouseEv, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		index := indexAt(mouseEv.Y)
		if index != e.hoverIndex {
			e.hoverIndex = index
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		// 级联：hover 到有子菜单的项 → 展开；hover 到其它项 → 收起
		if index >= 0 && len(m.Items[index].Children) > 0 {
			e.openSubmenu(index)
		} else if index >= 0 {
			e.closeSubmenu()
		}
		return true

	case event.TypeMouseDown:
		mouseEv, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		index := indexAt(mouseEv.Y)
		if index >= 0 && m.Items[index].Enabled {
			if len(m.Items[index].Children) > 0 {
				e.openSubmenu(index) // 有子菜单：展开，不关闭
				return true
			}
			if m.Items[index].OnClick != nil {
				m.Items[index].OnClick()
			}
			if m.onClose != nil {
				m.onClose()
			}
		}
		return true

	case event.TypeMouseLeave:
		// 不在此收子菜单（让鼠标能移进子菜单）；仅清除本级 hover。
		if e.hoverIndex != -1 {
			e.hoverIndex = -1
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		return true

	case event.TypeContextMenu:
		if m.onClose != nil {
			m.onClose()
		}
		return true
	}
	return false
}

// SetOnClose 设置关闭回调（由应用层调用）
func (m *Menu) SetOnClose(fn func()) {
	m.onClose = fn
}

// ShowContextMenu 在屏幕坐标 (x,y) 弹出上下文菜单（点击项或点击外部即关闭）。
// 供可编辑组件右键、或任意需要右键菜单的场景命令式调用；返回 overlay id，可手动 HideOverlay。
func ShowContextMenu(x, y float64, items []MenuItem) int {
	return ShowContextMenuStyled(x, y, items, types.Color{}, types.Color{}, types.Color{}, types.Color{})
}

// ShowContextMenuStyled 同 ShowContextMenu，但可指定配色（深色主题菜单用）。任一颜色为零值则用默认。
func ShowContextMenuStyled(x, y float64, items []MenuItem, bg, text, hover, border types.Color) int {
	if len(items) == 0 {
		return -1
	}
	var id int
	m := NewMenu(items...)
	if bg != (types.Color{}) {
		m.BGColor = bg
	}
	if text != (types.Color{}) {
		m.TextColor = text
	}
	if hover != (types.Color{}) {
		m.HoverColor = hover
	}
	if border != (types.Color{}) {
		m.BorderColor = border
	}
	m.SetOnClose(func() { HideOverlay(id) })
	id = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: x, Y: y, Width: 1, Height: 1},
		Content:        m,
		Placement:      PlacementBottomStart,
		Gap:            0,
		Transition:     "fade",
		OnClickOutside: func() { HideOverlay(id) },
	})
	return id
}
