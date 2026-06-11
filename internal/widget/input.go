package widget

import (
	"strings"
	"time"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Web 风格便捷构造函数 ──────────────────────────

// NewInput 创建输入框控件。
// 用法: widget.NewInput("请输入...", fn)
func NewInput(placeholder string, onChanged func(string)) *Input {
	t := CurrentTheme()
	return &Input{
		Placeholder:      placeholder,
		OnTextChanged:    onChanged,
		MaxLength:        1024, // 必须设置：MaxLength 为 0 时插入条件 len<0 恒为假，将完全无法输入
		Font:             canvas.DefaultFont(),
		Color:            t.Input.TextColor,
		BGColor:          t.Input.BGColor,
		BorderColor:      t.Input.BorderColor,
		FocusBorderColor: t.Input.FocusBorderColor,
		PlaceholderColor: t.Input.PlaceholderColor,
		CursorColor:      t.Input.CursorColor,
		CursorWidth:      t.Input.CursorWidth,
	}
}

// WithOnSubmit 设置提交回调（链式调用）
func (in *Input) WithOnSubmit(fn func(string)) *Input {
	in.OnSubmit = fn
	return in
}

// WithPlaceholder 设置占位文本（链式调用）
func (in *Input) WithPlaceholder(p string) *Input {
	in.Placeholder = p
	return in
}

// WithOnChanged 设置文本变化回调（链式调用）
func (in *Input) WithOnChanged(fn func(string)) *Input {
	in.OnTextChanged = fn
	return in
}

// WithFocusColor 设置聚焦时的边框色（CSS :focus，链式）。
func (in *Input) WithFocusColor(c types.Color) *Input {
	in.FocusBorderColor = c
	return in
}

// WithHoverColor 设置悬停时的边框色（CSS :hover，链式）。
func (in *Input) WithHoverColor(c types.Color) *Input {
	in.HoverBorderColor = c
	return in
}

// Input 文本输入框控件
type Input struct {
	StatelessWidget
	Placeholder      string
	OnTextChanged    func(text string)
	OnSubmit         func(text string)
	// 单行输入按上/下键回调（终端命令历史用）：返回 (要回填的文本, 是否处理)；处理则替换输入内容。
	OnArrowUp   func() (string, bool)
	OnArrowDown func() (string, bool)
	MaxLength        int
	Font             canvas.Font
	Color            types.Color
	BGColor          types.Color
	BorderColor      types.Color
	FocusBorderColor types.Color
	HoverBorderColor types.Color // 悬停边框色（CSS :hover；零值=用 el #c0c4cc 调色板）
	PlaceholderColor types.Color
	CursorColor      types.Color
	CursorWidth      float64
	Multiline        bool   // 多行文本框(el-input type=textarea)
	Wrap             bool   // 多行自动换行：长行按宽度折行显示（无横向滚动）；零值=不换行(横向滚动)
	Rows             int    // 多行显示行数(默认 3)
	Text             string // 初始文本(预填)
	ResetToken       int    // 受控清空/重置：改变此值(+SetState)会把运行时文本重置为 Text（发送后清空等）

	ContextMenuItems    []MenuItem // 自定义右键菜单项（nil=用默认剪切/复制/粘贴/全选）
	ContextMenuDisabled bool       // 禁用右键菜单（右键不弹出）
}

// WithContextMenu 自定义右键菜单项（替换默认菜单）。传空切片等同默认。
func (in *Input) WithContextMenu(items ...MenuItem) *Input {
	in.ContextMenuItems = items
	in.ContextMenuDisabled = false
	return in
}

// WithoutContextMenu 禁用右键菜单（右键不弹出任何菜单）。
func (in *Input) WithoutContextMenu() *Input {
	in.ContextMenuDisabled = true
	return in
}

// NewTextarea 创建多行文本框(el-input textarea)：回车插入换行、支持上下键、自适应行高。
func NewTextarea(placeholder string, rows int, onChanged func(string)) *Input {
	in := NewInput(placeholder, onChanged)
	in.Multiline = true
	in.MaxLength = 10000
	if rows <= 0 {
		rows = 3
	}
	in.Rows = rows
	return in
}

// DefaultInput 返回默认配置的 Input
func DefaultInput() *Input {
	return &Input{
		Font:             canvas.DefaultFont(),
		Color:            elTextPrimary(),
		BGColor:          types.ColorFromRGB(255, 255, 255),
		BorderColor:      types.ColorFromRGB(180, 180, 180),
		FocusBorderColor: types.ColorFromRGB(66, 133, 244),
		PlaceholderColor: types.ColorFromRGB(180, 180, 180),
		CursorColor:      elTextPrimary(),
		CursorWidth:      1.5,
		MaxLength:        1024,
	}
}

// CreateElement 创建 InputElement
func (in *Input) CreateElement() Element {
	return &InputElement{
		BaseElement: BaseElement{widget: in},
		input:       in,
		text:        in.Text,
		cursorPos:   len([]rune(in.Text)),
		cursorMoved: true,
		selAnchor:   -1, // 初始无选区
	}
}

// InputElement 文本输入框运行时 Element
type InputElement struct {
	BaseElement
	input             *Input
	text              string
	lastReset         int     // 上次见到的 input.ResetToken（受控清空用）
	cursorPos         int     // 光标位置（rune 索引）
	scrollOffset      float64 // 文本水平滚动偏移（像素，>=0），保证光标始终可见
	scrollY           float64 // 多行垂直滚动偏移（像素，>=0）
	scrollX           float64 // 多行水平滚动偏移（像素，>=0）——长行不换行时横向滚动
	cursorMoved       bool    // 光标刚移动(打字/方向键/点击)→ 下一帧让光标可见；滚轮不设此标志
	vbarThumb         types.Rect // 多行竖滚动条滑块矩形（Paint 缓存，供拖动命中）
	hbarThumb         types.Rect // 多行横滚动条滑块矩形
	vbarFactor        float64    // 竖滚动条：每像素拖动对应的 scrollY 增量
	hbarFactor        float64    // 横滚动条：每像素拖动对应的 scrollX 增量
	draggingVBar      bool       // 正在拖竖滚动条
	draggingHBar      bool       // 正在拖横滚动条
	dragStartMouse    float64    // 拖动起点鼠标坐标
	dragStartScroll   float64    // 拖动起点滚动偏移
	textDragArmed     bool       // 在选区内按下，待判定拖动文本/单击
	textDragging      bool       // 正在拖动选中文本到别处
	dragArmPos        int        // 按下时的光标位置（"只是点击选区"时用）
	dropPos           int        // 拖放目标插入位置
	dragDownX         float64    // 按下点（拖动阈值判定）
	dragDownY         float64
	cursorClientX     float64       // Paint 缓存：光标在客户区中的 X（Skia 测量），供 IME 候选定位
	cursorCaretTop    float64       // Paint 缓存：光标/组合串顶部的客户区 Y，作为 IME 候选锚点
	lastCanvas        canvas.Canvas // Paint 缓存：供点击定位用与渲染一致的 Skia 文本测量
	selAnchor         int     // 选区锚点（rune 索引）；-1 表示无选区，选区为 [min(anchor,cursor), max(anchor,cursor)]
	selecting         bool    // 是否正在用鼠标拖动选择
	focused           bool
	hovered           bool      // 鼠标是否悬停在输入框上
	focusTime         time.Time // 获得焦点的时间戳（用于计算光标闪烁）
	composition       string    // IME 组合预览字符串（如输入法未确认的拼音）
	compositionCursor int       // 组合字符串中的光标位置
}

// Build 返回 nil（Input 没有子控件）
func (e *InputElement) Build() []Element { return nil }

// Layout 测量并布局 Input
func (e *InputElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	font := e.input.Font
	if font.Size <= 0 {
		font = canvas.DefaultFont()
	}

	// 测量占位文本或当前文本，确定合适的高度
	displayText := e.text
	if displayText == "" {
		displayText = e.input.Placeholder
	}

	// 使用 Canvas 测量文本（通过 Pipeline 的 Canvas）
	// 在 Layout 阶段，Canvas 可能不可用，使用近似值
	lineHeight := font.Size * 1.4
	if lineHeight < 24 {
		lineHeight = 24
	}

	boxH := lineHeight + 8 // 单行：上下各 4px padding
	if e.input.Multiline {
		rows := e.input.Rows
		if rows <= 0 {
			rows = 3
		}
		boxH = float64(rows)*lineHeight + 10
	}
	height := ctx.Constraints.Constrain(types.Size{
		Width:  ctx.Constraints.MaxWidth,
		Height: boxH,
	})
	e.size = height
	return layout.LayoutResult{Size: height}
}

// Paint 绘制 Input
func (e *InputElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	in := e.input
	e.lastCanvas = cvs // 缓存画布，供点击定位用与渲染一致的 Skia 测量

	// 获取字体
	font := in.Font
	if font.Size <= 0 {
		font = canvas.DefaultFont()
	}

	// 计算内边距
	padLeft := 11.0
	padRight := 11.0
	// 文本基线垂直居中。
	// 用经验度量（约 0.8/0.2 字号）而非字体 face 的 ascent/descent：某些字体
	// （如阿里普惠体）face ascent 远大于字号（16 字号 face ascent≈23），会把基线
	// 压低，导致文字偏下、上方留白过多。经验值更贴近 Skia 实际渲染的字形高度。
	ascent := canvas.GetFaceAscent(font.Size)
	descent := canvas.GetFaceDescent(font.Size)
	textY := canvas.BaselineFor(pos.Y, e.size.Height, font.Size, canvas.VAlignMiddle)
	textX := pos.X + padLeft
	textWidth := e.size.Width - padLeft - padRight

	// 背景（el 风格圆角，白底不随 hover 加深）
	const inputRadius = 4.0
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = in.BGColor
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, inputRadius, bgPaint)

	// 边框（聚焦→主色，悬停→el 加深灰 #c0c4cc，否则常态边框；el 风格 1px 圆角）
	borderColor := in.BorderColor
	if e.IsFocused() {
		borderColor = in.FocusBorderColor
	} else if e.hovered {
		borderColor = in.HoverBorderColor
		if borderColor == (types.Color{}) {
			borderColor = elPlaceholder() // 默认 el hover #c0c4cc
		}
	}
	borderPaint := paint.DefaultStrokePaint()
	borderPaint.Color = borderColor
	borderPaint.StrokeWidth = 1.0
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, e.size.Width-1, e.size.Height-1, inputRadius, borderPaint)

	// 多行(textarea)：走独立多行渲染，复用上面的背景/边框
	if in.Multiline {
		e.paintMultiline(cvs, font, textX, textWidth, ascent)
		return
	}

	// ── 文本内容区：在裁剪 + 水平滚动下统一绘制选区/文本/组合/光标 ──
	runes := []rune(e.text)
	cp := e.cursorPos
	if cp > len(runes) {
		cp = len(runes)
	}

	// 光标在“文本坐标系”（未滚动）中的 X 偏移
	cursorTextX := 0.0
	if cp > 0 {
		cursorTextX = cvs.MeasureText(string(runes[:cp]), font).Width
	}
	// 组合串接在光标后，光标实际落点在组合串末尾
	compStartTextX := cursorTextX
	if e.composition != "" {
		cursorTextX += cvs.MeasureText(e.composition, font).Width
	}

	// 是否显示占位文本（空文本、无 IME 组合、有占位串）
	showPlaceholder := e.text == "" && e.composition == "" && in.Placeholder != ""

	// 根据光标位置调整水平滚动，保证光标始终可见
	if showPlaceholder {
		e.scrollOffset = 0
	} else {
		if cursorTextX-e.scrollOffset > textWidth {
			e.scrollOffset = cursorTextX - textWidth
		}
		if cursorTextX < e.scrollOffset {
			e.scrollOffset = cursorTextX
		}
		// 文本变短时收紧滚动，避免右侧出现空白
		totalW := 0.0
		if e.text != "" {
			totalW = cvs.MeasureText(e.text, font).Width
		}
		if e.composition != "" {
			totalW += cvs.MeasureText(e.composition, font).Width
		}
		if maxScroll := totalW - textWidth; maxScroll > 0 && e.scrollOffset > maxScroll {
			e.scrollOffset = maxScroll
		}
		if e.scrollOffset < 0 {
			e.scrollOffset = 0
		}
	}

	drawBaseX := textX - e.scrollOffset

	// 裁剪到文本可见区，滚出框外的文字不绘制
	cvs.Save()
	cvs.ClipRect(textX, pos.Y, textWidth, e.size.Height)

	// 选区高亮（作为文字下方的背景）
	if e.IsFocused() && e.hasSelection() {
		lo, hi := e.selectionRange()
		selLoX := drawBaseX + cvs.MeasureText(string(runes[:lo]), font).Width
		selHiX := drawBaseX + cvs.MeasureText(string(runes[:hi]), font).Width
		if selHiX > selLoX {
			selPaint := paint.DefaultPaint()
			selPaint.Color = types.ColorFromRGBA(66, 133, 244, 80) // 半透明蓝
			cvs.DrawRect(selLoX, textY-ascent, selHiX-selLoX, ascent+descent, selPaint)
		}
	}

	// 文本或占位文本
	if showPlaceholder {
		phPaint := paint.DefaultPaint()
		phPaint.Color = in.PlaceholderColor
		if phPaint.Color.A == 0 {
			phPaint.Color = types.ColorFromRGB(180, 180, 180)
		}
		cvs.DrawText(in.Placeholder, textX, textY, font, phPaint)
	} else if e.text != "" {
		txtPaint := paint.DefaultPaint()
		txtPaint.Color = in.Color
		if txtPaint.Color.A == 0 {
			txtPaint.Color = elTextPrimary()
		}
		cvs.DrawText(e.text, drawBaseX, textY, font, txtPaint)
	}

	// IME 组合预览（接在光标处，带下划线）
	if e.IsFocused() && e.composition != "" {
		compX := drawBaseX + compStartTextX
		compPaint := paint.DefaultPaint()
		compPaint.Color = in.Color
		if compPaint.Color.A == 0 {
			compPaint.Color = elTextPrimary()
		}
		cvs.DrawText(e.composition, compX, textY, font, compPaint)

		compW := cvs.MeasureText(e.composition, font).Width
		linePaint := paint.DefaultStrokePaint()
		linePaint.Color = types.ColorFromRGB(66, 133, 244) // 蓝色下划线
		linePaint.StrokeWidth = 1.0
		cvs.DrawLine(compX, textY+2, compX+compW, textY+2, linePaint)
	}

	// 光标（聚焦且闪烁可见时）。光标高度匹配文字（基线上 ascent、基线下 descent）。
	if e.IsFocused() && e.isCursorVisible() {
		cursorPaint := paint.DefaultStrokePaint()
		cursorPaint.Color = in.CursorColor
		cursorPaint.StrokeWidth = in.CursorWidth
		cursorX := drawBaseX + cursorTextX
		cvs.DrawLine(cursorX, textY-ascent, cursorX, textY+descent, cursorPaint)
	}

	// 文本拖放：插入点指示（橙色竖线）
	if e.textDragging {
		dp := e.dropPos
		if dp < 0 {
			dp = 0
		}
		if dp > len(runes) {
			dp = len(runes)
		}
		ddx := drawBaseX + cvs.MeasureText(string(runes[:dp]), font).Width
		dpaint := paint.DefaultStrokePaint()
		dpaint.Color = types.ColorFromRGB(245, 130, 32)
		dpaint.StrokeWidth = 2
		cvs.DrawLine(ddx, textY-ascent, ddx, textY+descent, dpaint)
	}

	cvs.Restore()

	// 缓存光标客户区位置（与渲染同一套 Skia 测量），供 IME 候选窗口精确定位。
	// 关键：候选定位不能改用 gofont 的 MeasureTextGlobal —— 它与 Skia 渲染的字宽
	// （尤其中文）差异大且逐字累积，会让候选随输入越漂越远。
	e.cursorClientX = drawBaseX + cursorTextX
	e.cursorCaretTop = textY - ascent // caret 顶部，IME 据此把候选放到文字下方
}

// ─── 多行(textarea)辅助 ──────────────────────────────

const sbThick = 9.0 // 滚动条占位厚度（滑块 6 + 间隙）

// vline 一条「可视行」：text=可视行文本；start=其首字符在整段文本中的全局 rune 偏移。
type vline struct {
	text  string
	start int
}

// wrapRunes 把一段（不含 \n）按宽度贪心折行，优先在空格后断；返回各段的 [起始 rune 下标]。
// 性能：先一次性测每个字的宽度再累加判断（O(n)），避免每步重测增长前缀（O(n²)）——
// 否则长文每次重绘都 O(n²)，长按退格会明显卡顿。
func wrapRunes(runes []rune, measure func(string) float64, viewW float64) []int {
	starts := []int{0}
	n := len(runes)
	if viewW <= 0 || n == 0 {
		return starts
	}
	cw := make([]float64, n)
	for i := range runes {
		cw[i] = measure(string(runes[i]))
	}
	lineStart, lastSpace, w := 0, -1, 0.0
	for i := 0; i < n; i++ {
		if runes[i] == ' ' {
			lastSpace = i
		}
		if i > lineStart && w+cw[i] > viewW {
			brk := i
			if lastSpace > lineStart {
				brk = lastSpace + 1 // 空格后断行
			}
			starts = append(starts, brk)
			w = 0
			for j := brk; j <= i; j++ { // 移到新行的字（brk..i）重算行宽
				w += cw[j]
			}
			lineStart, lastSpace = brk, -1
		} else {
			w += cw[i]
		}
	}
	return starts
}

// visualLines 把文本切成可视行：Wrap 时按 viewW 折长行，否则一逻辑行=一可视行。
func (e *InputElement) visualLines(measure func(string) float64, viewW float64) []vline {
	var out []vline
	off := 0
	for _, lg := range strings.Split(e.textWithComposition(), "\n") {
		runes := []rune(lg)
		if !e.input.Wrap {
			out = append(out, vline{text: lg, start: off})
			off += len(runes) + 1
			continue
		}
		starts := wrapRunes(runes, measure, viewW)
		for si, s := range starts {
			end := len(runes)
			if si+1 < len(starts) {
				end = starts[si+1]
			}
			out = append(out, vline{text: string(runes[s:end]), start: off + s})
		}
		off += len(runes) + 1
	}
	return out
}

// wrapPlain 折一段纯文本（占位符用）为可视行文本。
func wrapPlain(s string, measure func(string) float64, viewW float64) []string {
	var out []string
	for _, lg := range strings.Split(s, "\n") {
		runes := []rune(lg)
		starts := wrapRunes(runes, measure, viewW)
		for si, st := range starts {
			end := len(runes)
			if si+1 < len(starts) {
				end = starts[si+1]
			}
			out = append(out, string(runes[st:end]))
		}
	}
	return out
}

// vlineAtPos 把 rune 位置换算为 (可视行号, 行内列)：取 start<=cp 的最后一行（折行/换行边界都归到下一可视行起点）。
func vlineAtPos(vls []vline, cp int) (row, col int) {
	for i := range vls {
		if vls[i].start <= cp {
			row = i
		} else {
			break
		}
	}
	col = cp - vls[row].start
	if n := len([]rune(vls[row].text)); col > n {
		col = n
	}
	return
}

// paintMultiline 绘制多行文本 + 选区高亮 + 行列光标 + 纵/横滚动(光标跟随、滚动条留白)。
func (e *InputElement) paintMultiline(cvs canvas.Canvas, font canvas.Font, textX, textWidth, ascent float64) {
	in := e.input
	if in.Wrap { // 自动换行走独立可视行渲染（无横向滚动）
		e.paintMultilineWrap(cvs, font, textX, textWidth)
		return
	}
	_ = ascent // 多行改用 BaselineFor 居中，不再用传入的 face ascent
	pos := e.Offset()
	lineH := font.Size * 1.4
	if lineH < 24 {
		lineH = 24
	}
	lines := strings.Split(e.textWithComposition(), "\n")
	cl, cc, lineStr := e.cursorLineCol()

	// 内容尺寸
	contentH := float64(len(lines)) * lineH
	maxLineW := 0.0
	for _, line := range lines {
		if lw := canvas.MeasureTextGlobal(line, font).Width; lw > maxLineW {
			maxLineW = lw
		}
	}
	// 滚动条占位：竖条占右、横条占底。先判竖条→缩窄可视宽→再判横条→缩矮可视高，
	// 再二次校正（横条出现后变矮可能反过来需要竖条）。给文字留出滚动条的边距。
	vBar, hBar := 0.0, 0.0
	if contentH > e.size.Height-10 {
		vBar = sbThick
	}
	viewW := textWidth - vBar
	if maxLineW > viewW {
		hBar = sbThick
	}
	viewH := e.size.Height - 10 - hBar
	if vBar == 0 && contentH > viewH {
		vBar = sbThick
		viewW = textWidth - vBar
		if hBar == 0 && maxLineW > viewW {
			hBar = sbThick
			viewH = e.size.Height - 10 - hBar
		}
	}
	maxScrollY := contentH - viewH
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	maxScrollX := maxLineW - viewW + 4
	if maxScrollX < 0 {
		maxScrollX = 0
	}

	// 光标跟随（仅光标刚移动时；滚轮浏览不触发，避免被拉回）——同时管纵向与横向
	if e.IsFocused() && e.cursorMoved {
		curTop := float64(cl) * lineH
		if curTop-e.scrollY < 0 {
			e.scrollY = curTop
		}
		if curTop+lineH-e.scrollY > viewH {
			e.scrollY = curTop + lineH - viewH
		}
		lr := []rune(lineStr)
		ccc := cc
		if ccc > len(lr) {
			ccc = len(lr)
		}
		curX := canvas.MeasureTextGlobal(string(lr[:ccc]), font).Width
		if curX-e.scrollX < 0 {
			e.scrollX = curX
		}
		if curX-e.scrollX > viewW-6 {
			e.scrollX = curX - viewW + 6
		}
		e.cursorMoved = false
	}
	e.scrollY = clamp(e.scrollY, 0, maxScrollY)
	e.scrollX = clamp(e.scrollX, 0, maxScrollX)

	top := pos.Y + 5 - e.scrollY
	left := textX - e.scrollX // 文本水平起点（含横向滚动偏移）

	cvs.Save()
	cvs.ClipRect(textX, pos.Y+1, viewW, e.size.Height-2-hBar) // 裁掉竖条/横条占用区

	// 选区高亮（聚焦 + 有选区）——逐行求交集画半透明蓝
	if e.IsFocused() && e.hasSelection() {
		lo, hi := e.selectionRange()
		e.paintMultilineSel(cvs, lines, font, left, top, lineH, lo, hi)
	}

	if e.text == "" && e.composition == "" && in.Placeholder != "" {
		ph := paint.DefaultPaint()
		ph.Color = in.PlaceholderColor
		if ph.Color.A == 0 {
			ph.Color = types.ColorFromRGB(180, 180, 180)
		}
		cvs.DrawText(in.Placeholder, textX, canvas.BaselineFor(top, lineH, font.Size, canvas.VAlignMiddle), font, ph)
	} else {
		txt := paint.DefaultPaint()
		txt.Color = in.Color
		if txt.Color.A == 0 {
			txt.Color = elTextPrimary()
		}
		for i, line := range lines {
			ly := top + float64(i)*lineH
			if ly+lineH < pos.Y || ly > pos.Y+e.size.Height {
				continue // 视口外的行跳过
			}
			// 基线按行格垂直居中（face ascent 偏大会让文字偏下/溢出，与满格选区错位）
			cvs.DrawText(line, left, canvas.BaselineFor(ly, lineH, font.Size, canvas.VAlignMiddle), font, txt)
		}
	}

	if e.IsFocused() && e.isCursorVisible() {
		lr := []rune(lineStr)
		if cc > len(lr) {
			cc = len(lr)
		}
		cursorX := left
		if cc > 0 {
			cursorX += cvs.MeasureText(string(lr[:cc]), font).Width
		}
		// 光标按行格垂直居中（贴合文字字形高度），与文本/选区对齐
		cursorLy := top + float64(cl)*lineH
		cBase := canvas.BaselineFor(cursorLy, lineH, font.Size, canvas.VAlignMiddle)
		emA := font.Size * 0.82
		emD := font.Size * 0.22
		cp := paint.DefaultStrokePaint()
		cp.Color = in.CursorColor
		cp.StrokeWidth = in.CursorWidth
		cvs.DrawLine(cursorX, cBase-emA, cursorX, cBase+emD, cp)
		e.cursorClientX = cursorX
		e.cursorCaretTop = cBase - emA
	}

	// 文本拖放：插入点指示（橙色竖线，标示松手后文本将落到的位置）
	if e.textDragging {
		dl, dcc, dls := e.lineColAt(e.dropPos)
		dlr := []rune(dls)
		if dcc > len(dlr) {
			dcc = len(dlr)
		}
		dx := left
		if dcc > 0 {
			dx += cvs.MeasureText(string(dlr[:dcc]), font).Width
		}
		dBase := canvas.BaselineFor(top+float64(dl)*lineH, lineH, font.Size, canvas.VAlignMiddle)
		dp := paint.DefaultStrokePaint()
		dp.Color = types.ColorFromRGB(245, 130, 32)
		dp.StrokeWidth = 2
		cvs.DrawLine(dx, dBase-font.Size*0.82, dx, dBase+font.Size*0.22, dp)
	}
	cvs.Restore()

	// 竖滚动条（缓存滑块矩形 + 拖动换算系数）
	e.vbarThumb, e.vbarFactor = types.Rect{}, 0
	if maxScrollY > 0 {
		barW := 6.0
		bx := pos.X + e.size.Width - barW - 3
		trackH := e.size.Height - 6 - hBar
		thumbH := viewH / contentH * trackH
		if thumbH < 20 {
			thumbH = 20
		}
		thumbY := pos.Y + 3 + (e.scrollY/maxScrollY)*(trackH-thumbH)
		e.vbarThumb = types.Rect{X: bx, Y: thumbY, Width: barW, Height: thumbH}
		if trackH-thumbH > 0 {
			e.vbarFactor = maxScrollY / (trackH - thumbH)
		}
		th := paint.DefaultPaint()
		th.Color = types.ColorFromRGB(193, 193, 193)
		cvs.DrawRoundedRect(bx, thumbY, barW, thumbH, 3, th)
	}

	// 横滚动条（底部）
	e.hbarThumb, e.hbarFactor = types.Rect{}, 0
	if maxScrollX > 0 {
		barH := 6.0
		by := pos.Y + e.size.Height - barH - 3
		trackW := viewW
		thumbW := viewW / maxLineW * trackW
		if thumbW < 20 {
			thumbW = 20
		}
		thumbX := textX + (e.scrollX/maxScrollX)*(trackW-thumbW)
		e.hbarThumb = types.Rect{X: thumbX, Y: by, Width: thumbW, Height: barH}
		if trackW-thumbW > 0 {
			e.hbarFactor = maxScrollX / (trackW - thumbW)
		}
		th := paint.DefaultPaint()
		th.Color = types.ColorFromRGB(193, 193, 193)
		cvs.DrawRoundedRect(thumbX, by, thumbW, barH, 3, th)
	}
}

// paintMultilineWrap 自动换行模式渲染：按 viewW 折行成可视行，竖向滚动 + 竖滚动条（无横向）。
func (e *InputElement) paintMultilineWrap(cvs canvas.Canvas, font canvas.Font, textX, textWidth float64) {
	in := e.input
	pos := e.Offset()
	lineH := font.Size * 1.4
	if lineH < 24 {
		lineH = 24
	}
	measure := func(s string) float64 { return cvs.MeasureText(s, font).Width }
	viewW := textWidth - sbThick // 预留竖条槽（稳定折行宽，避免与滚动条互相影响）
	if viewW < 10 {
		viewW = textWidth
	}
	vls := e.visualLines(measure, viewW)
	contentH := float64(len(vls)) * lineH
	viewH := e.size.Height - 10
	maxScrollY := contentH - viewH
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	crow, ccol := vlineAtPos(vls, e.cursorPos)
	if e.IsFocused() && e.cursorMoved {
		curTop := float64(crow) * lineH
		if curTop-e.scrollY < 0 {
			e.scrollY = curTop
		}
		if curTop+lineH-e.scrollY > viewH {
			e.scrollY = curTop + lineH - viewH
		}
		e.cursorMoved = false
	}
	e.scrollY = clamp(e.scrollY, 0, maxScrollY)
	e.scrollX = 0
	top := pos.Y + 5 - e.scrollY

	cvs.Save()
	cvs.ClipRect(textX, pos.Y+1, viewW, e.size.Height-2)

	if e.IsFocused() && e.hasSelection() {
		lo, hi := e.selectionRange()
		sel := paint.DefaultPaint()
		sel.Color = types.ColorFromRGBA(66, 133, 244, 80)
		for i, vl := range vls {
			rs := []rune(vl.text)
			ls, le := vl.start, vl.start+len(rs)
			a, b := lo, hi
			if a < ls {
				a = ls
			}
			if b > le {
				b = le
			}
			if a < b {
				x0 := textX + measure(string(rs[:a-ls]))
				x1 := textX + measure(string(rs[:b-ls]))
				cvs.DrawRect(x0, top+float64(i)*lineH, x1-x0, lineH, sel)
			}
		}
	}

	if e.text == "" && e.composition == "" && in.Placeholder != "" {
		ph := paint.DefaultPaint()
		ph.Color = in.PlaceholderColor
		if ph.Color.A == 0 {
			ph.Color = types.ColorFromRGB(180, 180, 180)
		}
		for i, line := range wrapPlain(in.Placeholder, measure, viewW) {
			ly := top + float64(i)*lineH
			cvs.DrawText(line, textX, canvas.BaselineFor(ly, lineH, font.Size, canvas.VAlignMiddle), font, ph)
		}
	} else {
		txt := paint.DefaultPaint()
		txt.Color = in.Color
		if txt.Color.A == 0 {
			txt.Color = elTextPrimary()
		}
		for i, vl := range vls {
			ly := top + float64(i)*lineH
			if ly+lineH < pos.Y || ly > pos.Y+e.size.Height {
				continue
			}
			cvs.DrawText(vl.text, textX, canvas.BaselineFor(ly, lineH, font.Size, canvas.VAlignMiddle), font, txt)
		}
	}

	if e.IsFocused() && e.isCursorVisible() {
		rs := []rune(vls[crow].text)
		if ccol > len(rs) {
			ccol = len(rs)
		}
		cursorX := textX + measure(string(rs[:ccol]))
		cBase := canvas.BaselineFor(top+float64(crow)*lineH, lineH, font.Size, canvas.VAlignMiddle)
		cp := paint.DefaultStrokePaint()
		cp.Color = in.CursorColor
		cp.StrokeWidth = in.CursorWidth
		cvs.DrawLine(cursorX, cBase-font.Size*0.82, cursorX, cBase+font.Size*0.22, cp)
		e.cursorClientX = cursorX
		e.cursorCaretTop = cBase - font.Size*0.82
	}
	cvs.Restore()

	e.vbarThumb, e.vbarFactor = types.Rect{}, 0
	e.hbarThumb, e.hbarFactor = types.Rect{}, 0
	if maxScrollY > 0 {
		barW := 6.0
		bx := pos.X + e.size.Width - barW - 3
		trackH := e.size.Height - 6
		thumbH := viewH / contentH * trackH
		if thumbH < 20 {
			thumbH = 20
		}
		thumbY := pos.Y + 3 + (e.scrollY/maxScrollY)*(trackH-thumbH)
		e.vbarThumb = types.Rect{X: bx, Y: thumbY, Width: barW, Height: thumbH}
		if trackH-thumbH > 0 {
			e.vbarFactor = maxScrollY / (trackH - thumbH)
		}
		th := paint.DefaultPaint()
		th.Color = types.ColorFromRGB(193, 193, 193)
		cvs.DrawRoundedRect(bx, thumbY, barW, thumbH, 3, th)
	}
}

// paintMultilineSel 逐行绘制选区高亮（lo,hi 为整文本的 rune 索引）。
func (e *InputElement) paintMultilineSel(cvs canvas.Canvas, lines []string, font canvas.Font, left, top, lineH float64, lo, hi int) {
	sel := paint.DefaultPaint()
	sel.Color = types.ColorFromRGBA(66, 133, 244, 80)
	off := 0
	for i, line := range lines {
		lr := []rune(line)
		ls, le := off, off+len(lr)
		off = le + 1 // +1：跳过换行符
		extendNL := hi > le && lo <= le // 选区跨过本行末尾换行→行尾补一小段表示选中换行
		a, b := lo, hi
		if a < ls {
			a = ls
		}
		if b > le {
			b = le
		}
		if a > b {
			if !extendNL {
				continue
			}
			a, b = le, le
		}
		x0 := left + canvas.MeasureTextGlobal(string(lr[:a-ls]), font).Width
		x1 := left + canvas.MeasureTextGlobal(string(lr[:b-ls]), font).Width
		if extendNL {
			x1 += 6
		}
		if x1 <= x0 {
			continue
		}
		ly := top + float64(i)*lineH
		cvs.DrawRect(x0, ly, x1-x0, lineH, sel)
	}
}

// thumbHit 判断点 (x,y) 是否落在滚动条滑块矩形上（四周略放宽，便于点中）。
func thumbHit(r types.Rect, x, y float64) bool {
	if r.Width <= 0 || r.Height <= 0 {
		return false
	}
	return x >= r.X-4 && x <= r.X+r.Width+4 && y >= r.Y-4 && y <= r.Y+r.Height+4
}

// textWithComposition 在光标处插入 IME 组合串（用于显示）。
func (e *InputElement) textWithComposition() string {
	if e.composition == "" {
		return e.text
	}
	runes := []rune(e.text)
	cp := e.cursorPos
	if cp > len(runes) {
		cp = len(runes)
	}
	return string(runes[:cp]) + e.composition + string(runes[cp:])
}

// cursorLineCol 返回光标所在行号、列号(rune)、该行字符串。
func (e *InputElement) cursorLineCol() (line, col int, lineStr string) {
	return e.lineColAt(e.cursorPos)
}

// lineColAt 把任意 rune 位置换算为 (行, 列, 该行文本)。
func (e *InputElement) lineColAt(cp int) (line, col int, lineStr string) {
	runes := []rune(e.text)
	if cp > len(runes) {
		cp = len(runes)
	}
	if cp < 0 {
		cp = 0
	}
	lineStart := 0
	for i := 0; i < cp; i++ {
		if runes[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}
	col = cp - lineStart
	end := len(runes)
	for i := cp; i < len(runes); i++ {
		if runes[i] == '\n' {
			end = i
			break
		}
	}
	lineStr = string(runes[lineStart:end])
	return
}

// moveCursorVertical 多行光标上/下移一行（尽量保持列）。
func (e *InputElement) moveCursorVertical(dir int) {
	if e.input.Wrap { // 折行模式：按可视行上下移
		font := e.input.Font
		if font.Size <= 0 {
			font = canvas.DefaultFont()
		}
		measure := func(s string) float64 {
			if e.lastCanvas != nil {
				return e.lastCanvas.MeasureText(s, font).Width
			}
			return canvas.MeasureTextGlobal(s, font).Width
		}
		viewW := e.size.Width - 22 - sbThick
		if viewW < 10 {
			viewW = e.size.Width - 22
		}
		vls := e.visualLines(measure, viewW)
		row, col := vlineAtPos(vls, e.cursorPos)
		target := row + dir
		if target < 0 || target >= len(vls) {
			return
		}
		if n := len([]rune(vls[target].text)); col > n {
			col = n
		}
		e.cursorPos = vls[target].start + col
		e.clearSelection()
		e.cursorMoved = true
		return
	}
	cl, cc, _ := e.cursorLineCol()
	lines := strings.Split(e.text, "\n")
	target := cl + dir
	if target < 0 || target >= len(lines) {
		return
	}
	if tlen := len([]rune(lines[target])); cc > tlen {
		cc = tlen
	}
	pos := 0
	for i := 0; i < target; i++ {
		pos += len([]rune(lines[i])) + 1
	}
	e.cursorPos = pos + cc
	e.clearSelection()
}

// cursorPosFromLocalMulti 多行点击定位：局部 (lx,ly) → cursorPos。
func (e *InputElement) cursorPosFromLocalMulti(lx, ly float64) int {
	font := e.input.Font
	if font.Size <= 0 {
		font = canvas.DefaultFont()
	}
	lineH := font.Size * 1.4
	if lineH < 24 {
		lineH = 24
	}
	measure := func(s string) float64 {
		if e.lastCanvas != nil {
			return e.lastCanvas.MeasureText(s, font).Width
		}
		return canvas.MeasureTextGlobal(s, font).Width
	}
	if e.input.Wrap { // 折行模式：按可视行定位（无横向滚动）
		viewW := e.size.Width - 22 - sbThick
		if viewW < 10 {
			viewW = e.size.Width - 22
		}
		vls := e.visualLines(measure, viewW)
		row := int((ly - 5 + e.scrollY) / lineH)
		if row < 0 {
			row = 0
		}
		if row >= len(vls) {
			row = len(vls) - 1
		}
		rs := []rune(vls[row].text)
		col := 0
		target := lx - 11
		for col < len(rs) {
			if measure(string(rs[:col+1])) > target {
				break
			}
			col++
		}
		return vls[row].start + col
	}
	lines := strings.Split(e.text, "\n")
	line := int((ly - 5 + e.scrollY) / lineH) // 计入垂直滚动
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		line = len(lines) - 1
	}
	target := lx - 11 + e.scrollX // 计入水平滚动
	col := 0
	runes := []rune(lines[line])
	if e.lastCanvas != nil {
		for col < len(runes) {
			if e.lastCanvas.MeasureText(string(runes[:col+1]), font).Width > target {
				break
			}
			col++
		}
	}
	pos := 0
	for i := 0; i < line; i++ {
		pos += len([]rune(lines[i])) + 1
	}
	return pos + col
}

// ─── 选区与编辑辅助 ──────────────────────────────

// hasSelection 是否存在非空选区。
func (e *InputElement) hasSelection() bool {
	return e.selAnchor >= 0 && e.selAnchor != e.cursorPos
}

// selectionRange 返回规范化的选区 [lo, hi)（rune 索引）。
func (e *InputElement) selectionRange() (int, int) {
	lo, hi := e.selAnchor, e.cursorPos
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi
}

// clearSelection 清除当前选区。
func (e *InputElement) clearSelection() { e.selAnchor = -1 }

// selectedText 返回选区内的文本（无选区时为空串）。
func (e *InputElement) selectedText() string {
	if !e.hasSelection() {
		return ""
	}
	lo, hi := e.selectionRange()
	return string([]rune(e.text)[lo:hi])
}

// deleteSelection 删除选区文本，光标置于选区起点，返回是否删除了内容。
func (e *InputElement) deleteSelection() bool {
	if !e.hasSelection() {
		return false
	}
	lo, hi := e.selectionRange()
	runes := []rune(e.text)
	e.text = string(runes[:lo]) + string(runes[hi:])
	e.cursorPos = lo
	e.clearSelection()
	return true
}

// cursorPosFromLocalX 根据相对文本起点的 X 偏移返回最近的光标 rune 索引。
func (e *InputElement) cursorPosFromLocalX(localX float64) int {
	if localX <= 0 {
		return 0
	}
	font := e.input.Font
	if font.Size <= 0 {
		font = canvas.DefaultFont()
	}
	// 与渲染一致的文本测量：优先用 Paint 缓存的画布（Skia），回退到全局测量。
	// 否则点击/拖动定位会因 gofont 与 Skia 字宽不一致（尤其中文）而落到错误字符。
	measure := func(s string) float64 {
		if e.lastCanvas != nil {
			return e.lastCanvas.MeasureText(s, font).Width
		}
		return canvas.MeasureTextGlobal(s, font).Width
	}
	runes := []rune(e.text)
	for i := 1; i <= len(runes); i++ {
		w := measure(string(runes[:i]))
		if w >= localX {
			prevW := 0.0
			if i > 1 {
				prevW = measure(string(runes[:i-1]))
			}
			if localX-prevW < w-localX {
				return i - 1
			}
			return i
		}
	}
	return len(runes)
}

// insertText 在光标处插入文本（先删除选区），过滤换行，受 MaxLength 约束，
// 并触发 OnTextChanged。用于字符输入与粘贴。
func (e *InputElement) insertText(s string) {
	s = strings.ReplaceAll(s, "\r", "")
	if !e.input.Multiline {
		s = strings.ReplaceAll(s, "\n", "") // 单行：剔除换行；多行保留以支持回车/多行粘贴
	}
	if s == "" {
		return
	}
	e.deleteSelection()
	// 清空选区锚点：点击聚焦时 selAnchor 被置成光标位（空选区），插入后光标右移会把它变成
	// 覆盖首字的真选区，导致「聚焦后第一个字被选中、下一字覆盖它」。插入后本就不应有选区。
	e.selAnchor = -1
	runes := []rune(e.text)
	ins := []rune(s)
	if e.input.MaxLength > 0 {
		room := e.input.MaxLength - len(runes)
		if room <= 0 {
			return
		}
		if len(ins) > room {
			ins = ins[:room]
		}
	}
	e.text = string(runes[:e.cursorPos]) + string(ins) + string(runes[e.cursorPos:])
	e.cursorPos += len(ins)
	e.cursorMoved = true
	if e.input.OnTextChanged != nil {
		e.input.OnTextChanged(e.text)
	}
}

// Text 返回当前文本
func (e *InputElement) Text() string { return e.text }

// SetText 设置文本并调整光标位置
func (e *InputElement) SetText(text string) {
	e.text = text
	if e.cursorPos > len([]rune(text)) {
		e.cursorPos = len([]rune(text))
	}
	e.clearSelection() // 外部设置文本时重置选区，避免选区索引越界
}

// CursorClientPos 返回光标在窗口客户区中的 X 坐标。
// 用于 app 层设置 IME 候选窗口位置。
func (e *InputElement) CursorClientPos() (x, y float64) {
	pos := e.Offset()
	padLeft := 11.0
	padRight := 11.0
	textX := pos.X + padLeft
	textWidth := e.size.Width - padLeft - padRight

	// 直接使用 Paint 缓存的光标位置（与渲染同一套 Skia 测量）。
	// 不能在此用 gofont 的 MeasureTextGlobal 重新计算：它与 Skia 渲染的字宽
	// （尤其中文）差异大且逐字累积，会让候选窗口随输入越漂越远。
	x = e.cursorClientX
	if x < textX {
		x = textX
	}
	if x > textX+textWidth {
		x = textX + textWidth
	}
	// 候选锚点用 caret 顶部：多数输入法（微软拼音/搜狗）会把候选放到锚点下方
	// 一行，用顶部锚点能让候选恰好落在文字下方紧贴；用底部会再下移一行而偏下。
	y = e.cursorCaretTop
	return
}

// HandleEvent 处理鼠标和键盘事件
// ── 文本拖放（在选中文本上按下→拖到别处移动）──

// clickPos 把鼠标事件换算为文本中的 rune 位置（单行/多行各自映射）。
func (e *InputElement) clickPos(me *event.MouseEvent) int {
	pos := e.Offset()
	if e.input.Multiline {
		return e.cursorPosFromLocalMulti(me.X-pos.X, me.Y-pos.Y)
	}
	return e.cursorPosFromLocalX(me.X - (pos.X + 8.0) + e.scrollOffset)
}

// posInSelection 判断位置是否落在当前选区内（含边界）。
func (e *InputElement) posInSelection(pos int) bool {
	if !e.hasSelection() {
		return false
	}
	lo, hi := e.selectionRange()
	return pos >= lo && pos <= hi
}

// finishTextDrag 把选中文本移动到 dropPos：删原选区→在目标处插入→选中移动后的文本。
func (e *InputElement) finishTextDrag() {
	if !e.hasSelection() {
		return
	}
	lo, hi := e.selectionRange()
	drop := e.dropPos
	if drop >= lo && drop <= hi {
		return // 拖回选区内：视为取消，不动
	}
	runes := []rune(e.text)
	if lo < 0 {
		lo = 0
	}
	if hi > len(runes) {
		hi = len(runes)
	}
	moved := append([]rune{}, runes[lo:hi]...)
	rest := append(append([]rune{}, runes[:lo]...), runes[hi:]...)
	if drop > hi { // 删除选区后，目标若在选区之后需前移
		drop -= hi - lo
	}
	if drop < 0 {
		drop = 0
	}
	if drop > len(rest) {
		drop = len(rest)
	}
	final := append([]rune{}, rest[:drop]...)
	final = append(final, moved...)
	final = append(final, rest[drop:]...)
	e.text = string(final)
	e.selAnchor = drop // 选中移动后的文本
	e.cursorPos = drop + len(moved)
	e.cursorMoved = true
	if e.input.OnTextChanged != nil {
		e.input.OnTextChanged(e.text)
	}
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// ── 右键菜单动作（与 Ctrl 快捷键共用一套语义）──
func (e *InputElement) ctxCopy() {
	if e.hasSelection() && ClipboardWrite != nil {
		ClipboardWrite(e.selectedText())
	}
}
func (e *InputElement) ctxCut() {
	if e.hasSelection() && ClipboardWrite != nil {
		ClipboardWrite(e.selectedText())
		e.deleteSelection()
		if e.input.OnTextChanged != nil {
			e.input.OnTextChanged(e.text)
		}
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
	}
}
func (e *InputElement) ctxPaste() {
	if ClipboardRead != nil {
		e.insertText(ClipboardRead())
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
	}
}
func (e *InputElement) selectAll() {
	if len(e.text) > 0 {
		e.selAnchor = 0
		e.cursorPos = len([]rune(e.text))
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
	}
}

// defaultContextItems 默认右键菜单：剪切/复制/粘贴/全选，按选区与剪贴板可用性置灰。
func (e *InputElement) defaultContextItems() []MenuItem {
	hasSel := e.hasSelection()
	return []MenuItem{
		{Label: "剪切", Enabled: hasSel && ClipboardWrite != nil, OnClick: e.ctxCut},
		{Label: "复制", Enabled: hasSel && ClipboardWrite != nil, OnClick: e.ctxCopy},
		{Label: "粘贴", Enabled: ClipboardRead != nil, OnClick: e.ctxPaste},
		{Label: "全选", Enabled: len(e.text) > 0, OnClick: e.selectAll},
	}
}

func (e *InputElement) HandleEvent(ev event.Event) bool {
	in := e.input

	switch ev.Type() {
	case event.TypeContextMenu:
		if in.ContextMenuDisabled {
			return false // 禁用：右键不弹出菜单
		}
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		items := in.ContextMenuItems
		if len(items) == 0 {
			items = e.defaultContextItems()
		}
		ShowContextMenu(me.X, me.Y, items)
		return true
	case event.TypeMouseWheel:
		if in.Multiline {
			if me, ok := ev.(*event.MouseEvent); ok {
				// Ctrl/Shift+滚轮 或 触摸板横向手势 → 水平滚动；否则垂直。
				// 无论横竖都 StopPropagation：自己消费后阻断冒泡，外层 ScrollView/窗口不再跟着滚。
				if me.DeltaX != 0 || me.Mods&(event.ModShift|event.ModCtrl) != 0 {
					d := me.DeltaX
					if d == 0 {
						d = me.DeltaY
					}
					e.scrollX -= d * 40
					if e.scrollX < 0 {
						e.scrollX = 0
					}
					ev.StopPropagation()
					if OnNeedsRepaint != nil {
						OnNeedsRepaint()
					}
					return true
				}
				if me.DeltaY != 0 {
					e.scrollY -= me.DeltaY * 40
					if e.scrollY < 0 {
						e.scrollY = 0
					}
					ev.StopPropagation()
					if OnNeedsRepaint != nil {
						OnNeedsRepaint()
					}
					return true
				}
			}
		}
		return false
	case event.TypeMouseEnter:
		e.hovered = true
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true

	case event.TypeMouseLeave:
		e.hovered = false
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true

	case event.TypeMouseDown:
		// 点击定位光标，并开始一次可能的拖动选择
		mouseEv, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		if mouseEv.Button == event.ButtonRight {
			return false // 右键留给 ContextMenu：不移动光标、不清选区（保留选区供复制/剪切）
		}
		if in.Multiline {
			// 先判滚动条滑块：命中则进入拖动（抢指针捕获 + 阻断冒泡），不定位光标
			if thumbHit(e.vbarThumb, mouseEv.X, mouseEv.Y) {
				e.draggingVBar = true
				e.dragStartMouse = mouseEv.Y
				e.dragStartScroll = e.scrollY
				if RequestPointerCapture != nil {
					RequestPointerCapture(e)
				}
				ev.StopPropagation()
				return true
			}
			if thumbHit(e.hbarThumb, mouseEv.X, mouseEv.Y) {
				e.draggingHBar = true
				e.dragStartMouse = mouseEv.X
				e.dragStartScroll = e.scrollX
				if RequestPointerCapture != nil {
					RequestPointerCapture(e)
				}
				ev.StopPropagation()
				return true
			}
		}
		cp := e.clickPos(mouseEv)
		if e.posInSelection(cp) {
			// 在选中文本上按下：待判定是拖动文本还是单击（先不动选区/光标）
			e.textDragArmed = true
			e.dragArmPos = cp
			e.dragDownX, e.dragDownY = mouseEv.X, mouseEv.Y
			return true
		}
		e.cursorPos = cp
		e.selAnchor = cp // 锚点：拖动时与光标形成选区；单击未移动则为空选区
		e.selecting = true
		e.cursorMoved = true
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true

	case event.TypeMouseMove:
		mouseEv, ok := ev.(*event.MouseEvent)
		if !ok {
			return false
		}
		// 拖动滚动条：按鼠标位移 × 换算系数更新滚动偏移
		if e.draggingVBar {
			e.scrollY = clamp(e.dragStartScroll+(mouseEv.Y-e.dragStartMouse)*e.vbarFactor, 0, 1e9)
			ev.StopPropagation()
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		if e.draggingHBar {
			e.scrollX = clamp(e.dragStartScroll+(mouseEv.X-e.dragStartMouse)*e.hbarFactor, 0, 1e9)
			ev.StopPropagation()
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		// 文本拖放：在选区内按下后移动超阈值→进入拖动，实时更新插入点 dropPos
		if e.textDragArmed {
			if !e.textDragging {
				dx, dy := mouseEv.X-e.dragDownX, mouseEv.Y-e.dragDownY
				if dx*dx+dy*dy > 16 { // 超过 ~4px 判为拖动
					e.textDragging = true
					if RequestPointerCapture != nil {
						RequestPointerCapture(e)
					}
				}
			}
			if e.textDragging {
				e.dropPos = e.clickPos(mouseEv)
				if OnNeedsRepaint != nil {
					OnNeedsRepaint()
				}
			}
			return true
		}
		// 鼠标按下拖动时，更新选区终点（光标）
		if !e.selecting {
			return false
		}
		pos := e.Offset()
		if in.Multiline {
			e.cursorPos = e.cursorPosFromLocalMulti(mouseEv.X-pos.X, mouseEv.Y-pos.Y)
		} else {
			padLeft := 8.0
			e.cursorPos = e.cursorPosFromLocalX(mouseEv.X - (pos.X + padLeft) + e.scrollOffset)
		}
		e.cursorMoved = true // 让视图跟随拖动光标：拖到视口外即自动滚动
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true

	case event.TypeMouseUp:
		if e.textDragging { // 完成文本移动
			e.finishTextDrag()
			e.textDragging = false
			e.textDragArmed = false
			return true
		}
		if e.textDragArmed { // 在选区内只是点了一下（没拖）→ 清选区、光标落到点击处
			e.clearSelection()
			e.cursorPos = e.dragArmPos
			e.cursorMoved = true
			e.textDragArmed = false
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		// 结束拖动选择 / 滚动条拖动
		e.selecting = false
		e.draggingVBar = false
		e.draggingHBar = false
		return true

	case event.TypeMouseDoubleClick:
		// 双击选中全部文本
		e.selAnchor = 0
		e.cursorPos = len([]rune(e.text))
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true

	case event.TypeKeyChar:
		keyEv, ok := ev.(*event.KeyEvent)
		if !ok || !e.IsFocused() {
			break
		}
		// 可打印字符：插入（若有选区则先替换之）。insertText 内部处理选区、
		// MaxLength（<=0 视为不限制）与 OnTextChanged。
		// 回车（\r/\n < 32）不在此处理：提交逻辑统一由 KeyDown 的 "Enter" 分支处理，
		// 避免 WM_KEYDOWN 与 WM_CHAR 同时到达导致 OnSubmit 被触发两次。
		if keyEv.Char != 0 && keyEv.Char >= 32 {
			e.insertText(string(keyEv.Char))
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		return true

	case event.TypeKeyDown:
		keyEv, ok := ev.(*event.KeyEvent)
		if !ok || !e.IsFocused() {
			break
		}
		// ── Ctrl 快捷键：全选/复制/剪切/粘贴 ──
		if keyEv.Mods&event.ModCtrl != 0 {
			switch keyEv.Key {
			case "A": // 全选
				if len(e.text) > 0 {
					e.selAnchor = 0
					e.cursorPos = len([]rune(e.text))
				}
			case "C": // 复制
				if e.hasSelection() && ClipboardWrite != nil {
					ClipboardWrite(e.selectedText())
				}
			case "X": // 剪切
				if e.hasSelection() && ClipboardWrite != nil {
					ClipboardWrite(e.selectedText())
					e.deleteSelection()
					if in.OnTextChanged != nil {
						in.OnTextChanged(e.text)
					}
				}
			case "V": // 粘贴（替换选区）
				if ClipboardRead != nil {
					e.insertText(ClipboardRead())
				}
			}
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}

		key := keyEv.Key
		changed := false
		switch key {
		case "Backspace":
			if e.deleteSelection() {
				changed = true
				if in.OnTextChanged != nil {
					in.OnTextChanged(e.text)
				}
			} else if e.cursorPos > 0 {
				runes := []rune(e.text)
				e.text = string(runes[:e.cursorPos-1]) + string(runes[e.cursorPos:])
				e.cursorPos--
				changed = true
				if in.OnTextChanged != nil {
					in.OnTextChanged(e.text)
				}
			}
		case "Delete":
			if e.deleteSelection() {
				changed = true
				if in.OnTextChanged != nil {
					in.OnTextChanged(e.text)
				}
			} else {
				runes := []rune(e.text)
				if e.cursorPos < len(runes) {
					e.text = string(runes[:e.cursorPos]) + string(runes[e.cursorPos+1:])
					changed = true
					if in.OnTextChanged != nil {
						in.OnTextChanged(e.text)
					}
				}
			}
		case "ArrowLeft":
			if e.hasSelection() {
				// 有选区时，左移折叠到选区左端
				lo, _ := e.selectionRange()
				e.cursorPos = lo
				e.clearSelection()
				changed = true
			} else if e.cursorPos > 0 {
				e.cursorPos--
				changed = true
			}
		case "ArrowRight":
			if e.hasSelection() {
				// 有选区时，右移折叠到选区右端
				_, hi := e.selectionRange()
				e.cursorPos = hi
				e.clearSelection()
				changed = true
			} else {
				runes := []rune(e.text)
				if e.cursorPos < len(runes) {
					e.cursorPos++
					changed = true
				}
			}
		case "Home":
			e.clearSelection()
			if e.cursorPos != 0 {
				e.cursorPos = 0
				changed = true
			}
		case "End":
			e.clearSelection()
			runes := []rune(e.text)
			if e.cursorPos != len(runes) {
				e.cursorPos = len(runes)
				changed = true
			}
		case "Enter":
			if in.Multiline {
				e.insertText("\n") // 多行：插入换行
			} else if in.OnSubmit != nil {
				// 仅在配置了 OnSubmit（提交型输入框）时，回车提交当前文本并清空
				in.OnSubmit(e.text)
				e.text = ""
				e.cursorPos = 0
				e.clearSelection()
				changed = true
				if in.OnTextChanged != nil {
					in.OnTextChanged(e.text)
				}
			}
		case "ArrowUp":
			if in.Multiline {
				e.moveCursorVertical(-1)
				changed = true
			} else if in.OnArrowUp != nil {
				if txt, ok := in.OnArrowUp(); ok {
					e.SetText(txt)
					e.cursorPos = len([]rune(txt))
					if in.OnTextChanged != nil {
						in.OnTextChanged(txt)
					}
					changed = true
				}
			}
		case "ArrowDown":
			if in.Multiline {
				e.moveCursorVertical(1)
				changed = true
			} else if in.OnArrowDown != nil {
				if txt, ok := in.OnArrowDown(); ok {
					e.SetText(txt)
					e.cursorPos = len([]rune(txt))
					if in.OnTextChanged != nil {
						in.OnTextChanged(txt)
					}
					changed = true
				}
			}
		}
		if changed {
			e.cursorMoved = true
		}
		if changed && OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true

	case event.TypeIMEComposition:
		// IME 组合字符串更新
		compEv, ok := ev.(*event.IMECompositionEvent)
		if !ok || !e.IsFocused() {
			break
		}
		e.composition = compEv.Composition
		e.compositionCursor = compEv.CursorPos
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	}
	return false
}

// isCursorVisible 基于时间戳计算光标是否可见。
// 光标以约 500ms 为周期闪烁（250ms 可见，250ms 隐藏）。
func (e *InputElement) isCursorVisible() bool {
	if !e.focused || e.focusTime.IsZero() {
		return false
	}
	elapsed := time.Since(e.focusTime)
	// 每 500ms 切换一次可见性（前 250ms 可见，后 250ms 隐藏）
	const cursorPeriod = 500 * time.Millisecond
	const cursorOnDuration = 250 * time.Millisecond
	mod := elapsed % cursorPeriod
	return mod < cursorOnDuration
}

// Focus 获取焦点
func (e *InputElement) Focus() {
	if e.focused {
		return
	}
	e.focused = true
	e.focusTime = time.Now()
	e.composition = "" // 清除之前的 IME 组合状态
	e.selecting = false

	// 立即触发重绘，显示光标
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// Blur 失去焦点
func (e *InputElement) Blur() {
	e.focused = false
	e.cursorPos = len([]rune(e.text))
	e.focusTime = time.Time{}
	e.composition = "" // 清除 IME 组合状态
	e.clearSelection() // 失焦清除选区
	e.selecting = false

	// 触发重绘，清除光标
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// IsFocused 是否拥有焦点
func (e *InputElement) IsFocused() bool { return e.focused }

// Update 更新 Input 配置
func (e *InputElement) Update(newWidget Widget) {
	if newInput, ok := newWidget.(*Input); ok {
		// 受控清空/重置：ResetToken 变化时把运行时文本重置为 Text（如发送后清空输入框）。
		if newInput.ResetToken != e.lastReset {
			e.text = newInput.Text
			e.cursorPos = len([]rune(e.text))
			e.scrollOffset, e.scrollX, e.scrollY = 0, 0, 0
			e.lastReset = newInput.ResetToken
		}
		e.input = newInput
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
