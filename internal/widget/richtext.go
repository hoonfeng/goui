package widget

import (
	"image"
	"strconv"
	"strings"
	"time"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── RichText 富文本编辑器（WYSIWYG，一体化：内嵌工具条 + 编辑区）──────────
// 真·所见即所得，整体是一个组件：顶部紧凑工具栏(自绘小按钮 + 激活态跟随光标样式
// + hover + 分组分隔) + 下方无缝编辑区(带样式文档模型→混排文本流布局→跨 span
// 光标/选区→选区格式应用)，外层统一边框圆角。光标按文字基线对齐 + 闪烁。
//
// 已实现：粗体/斜体/下划线、H1-H3、有序/无序列表、引用、选中应用格式、工具栏激活态、
//   输入继承样式、退格/回车换段/方向键、点击拖选、竖向滚动、光标对齐+闪烁。
// 进阶：字号/颜色、对齐、链接/图片、撤销重做、IME 组合预览。

// ─── 数据模型 ───────────────────────────────────────────────────

type RichSpan struct {
	Text      string
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool         // 删除线
	Color     *types.Color // nil=默认文字色
	Size      float64      // 字号（0=继承块默认字号；>0 覆盖，可选中部分文字单独设）
}

type RichBlock struct {
	Type  string // "p"/"h1"/"h2"/"h3"/"ul"/"ol"/"quote"/"image"
	Spans []RichSpan
	Img   image.Image // Type=="image" 时的已解码图片（nil 则显示占位框）
	Path  string      // 图片来源路径（仅记录，便于序列化/重载）
	Align string      // 段落水平对齐：""/"left"（左）/"center"（中）/"right"（右）
}

type richPos struct {
	block  int
	offset int
}

func richPosLess(a, b richPos) bool {
	if a.block != b.block {
		return a.block < b.block
	}
	return a.offset < b.offset
}
func richPosEq(a, b richPos) bool { return a.block == b.block && a.offset == b.offset }

type spanStyle struct {
	bold, italic, underline, strike bool
	color                           *types.Color
	size                            float64
}

// ─── 工具栏定义 ─────────────────────────────────────────────────

type rtTool struct {
	label string
	kind  string // "style" / "block" / "sep"
	arg   string // bold/italic/underline 或 h1/h2/h3/p/ul/ol/quote
}

var rtTools = []rtTool{
	{"B", "style", "bold"},
	{"I", "style", "italic"},
	{"U", "style", "underline"},
	{"S", "style", "strike"},
	{"", "sep", ""},
	{"字号", "fontsize", ""},
	{"", "sep", ""},
	{"H1", "block", "h1"},
	{"H2", "block", "h2"},
	{"H3", "block", "h3"},
	{"正文", "block", "p"},
	{"", "sep", ""},
	{"列表", "block", "ul"},
	{"编号", "block", "ol"},
	{"引用", "block", "quote"},
	{"", "sep", ""},
	{"左", "align", "left"},
	{"中", "align", "center"},
	{"右", "align", "right"},
	{"", "sep", ""},
	{"A ▾", "colorpick", ""}, // 分体按钮：A=应用当前色，▾=弹拾色器
	{"图片", "image", ""},
	{"", "sep", ""},
	{"清除", "clear", ""}, // 清除选区内联样式
}

// ─── RichText widget ────────────────────────────────────────────

type RichText struct {
	StatelessWidget
	Width   float64
	Height  float64
	initial []RichBlock

	ContextMenuItems    []MenuItem // 自定义右键菜单项（nil=默认剪切/复制/粘贴/全选）
	ContextMenuDisabled bool       // 禁用右键菜单
}

// WithContextMenu 自定义右键菜单项（替换默认菜单）。
func (r *RichText) WithContextMenu(items ...MenuItem) *RichText {
	r.ContextMenuItems = items
	r.ContextMenuDisabled = false
	return r
}

// WithoutContextMenu 禁用右键菜单（右键不弹出）。
func (r *RichText) WithoutContextMenu() *RichText {
	r.ContextMenuDisabled = true
	return r
}

func NewRichText(initialText string) *RichText {
	return &RichText{Width: 640, Height: 300,
		initial: []RichBlock{{Type: "p", Spans: []RichSpan{{Text: initialText}}}}}
}
func (r *RichText) WithSize(w, h float64) *RichText { r.Width = w; r.Height = h; return r }
func (r *RichText) WithBlocks(blocks []RichBlock) *RichText { r.initial = blocks; return r }

func (r *RichText) CreateElement() Element {
	blocks := make([]RichBlock, len(r.initial))
	copy(blocks, r.initial)
	if len(blocks) == 0 {
		blocks = []RichBlock{{Type: "p", Spans: []RichSpan{{Text: ""}}}}
	}
	return &RichTextElement{BaseElement: BaseElement{widget: r}, rt: r, blocks: blocks, hoverTool: -1}
}

const (
	rtToolbarH = 38.0
	rtPadding  = 12.0
)

type RichTextElement struct {
	BaseElement
	rt      *RichText
	blocks  []RichBlock
	cursor   richPos
	anchor   richPos
	focused  bool
	pending   *spanStyle   // 待应用样式（无选区设颜色/字号/粗斜等时记此，影响下次输入；移动光标即重置）
	lastColor *types.Color // 最近用拾色器选定的颜色：颜色按钮“主体”点击直接应用此色（仿 Word/wangEditor）

	focusTime time.Time
	hoverTool int
	toolRects []types.Rect // 与 rtTools 等长（sep 也占位）
	toolbarH  float64      // 工具栏实际高度（按钮多时自动换行，可为多行高度）

	vlines   []richVLine
	contentH float64
	scrollY  float64
	dragging bool

	textDragArmed bool    // 在选区内按下，待判定拖动文本/单击
	textDragging  bool    // 正在拖动选中文本
	dragArmPos    richPos // 按下位置（"只是点击选区"时用）
	dropPos       richPos // 拖放目标
	dragDownX     float64 // 按下点（拖动阈值）
	dragDownY     float64

	undoStack    []richSnapshot // 撤销栈（编辑前快照）
	redoStack    []richSnapshot // 重做栈
	lastUndoKind string         // 上次记录的编辑类型（连续 insert/delete 合并为一个撤销单元）
}

// richSnapshot 富文本的一份可撤销快照：blocks 深拷贝 + 光标/锚点。
type richSnapshot struct {
	blocks         []RichBlock
	cursor, anchor richPos
}

func cloneRichBlocks(bs []RichBlock) []RichBlock {
	out := make([]RichBlock, len(bs))
	for i, b := range bs {
		out[i] = b
		out[i].Spans = append([]RichSpan(nil), b.Spans...) // RichSpan 值拷贝（Color 指针不可变，共享安全）
	}
	return out
}

func (e *RichTextElement) snapshot() richSnapshot {
	return richSnapshot{cloneRichBlocks(e.blocks), e.cursor, e.anchor}
}

// recordUndo 在编辑前记快照；连续 insert/delete 合并为一个撤销单元。
func (e *RichTextElement) recordUndo(kind string) {
	if (kind == "insert" || kind == "delete") && kind == e.lastUndoKind {
		return
	}
	e.undoStack = append(e.undoStack, e.snapshot())
	if len(e.undoStack) > 200 {
		e.undoStack = e.undoStack[1:]
	}
	e.redoStack = nil
	e.lastUndoKind = kind
}

func (e *RichTextElement) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	e.redoStack = append(e.redoStack, e.snapshot())
	s := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.blocks, e.cursor, e.anchor = cloneRichBlocks(s.blocks), s.cursor, s.anchor
	e.lastUndoKind = ""
	e.afterEdit()
}

func (e *RichTextElement) redo() {
	if len(e.redoStack) == 0 {
		return
	}
	e.undoStack = append(e.undoStack, e.snapshot())
	s := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.blocks, e.cursor, e.anchor = cloneRichBlocks(s.blocks), s.cursor, s.anchor
	e.lastUndoKind = ""
	e.afterEdit()
}

type richRun struct {
	text   string
	font   canvas.Font
	under  bool
	strike bool
	color  *types.Color
	x, w   float64
}

type richVLine struct {
	runs        []richRun
	y           float64
	height      float64
	blockIdx    int
	startOffset int
	caretX      []float64
	prefix      string
	indent      float64
	isImage     bool
	imgW        float64 // isImage 时的显示宽度（高度用 height）
	alignOffset float64 // 按段落对齐算出的整行/整图水平偏移
	maxSize     float64 // 行内最大字号（决定行高与基线）
}

// ─── 块/样式 ─────────────────────────────────────────────────────

func blockBaseSize(t string) float64 {
	switch t {
	case "h1":
		return 25
	case "h2":
		return 20
	case "h3":
		return 17
	default:
		return 15
	}
}
func blockBold(t string) bool { return t == "h1" || t == "h2" || t == "h3" }

func spanFont(sp RichSpan, base float64, blockType string) canvas.Font {
	f := canvas.DefaultFont()
	f.Size = base
	if sp.Size > 0 { // span 级字号覆盖块默认（标题里也能对部分文字单独设字号）
		f.Size = sp.Size
	}
	if sp.Bold || blockBold(blockType) {
		f.Weight = canvas.FontWeightBold
	}
	if sp.Italic {
		f.Style = canvas.FontStyleItalic
	}
	return f
}

func blockText(b RichBlock) []rune {
	var rs []rune
	for _, sp := range b.Spans {
		rs = append(rs, []rune(sp.Text)...)
	}
	return rs
}
func blockLen(b RichBlock) int {
	if b.Type == "image" {
		return 1 // 图片块视作一个原子字符（offset 0=图片前，1=图片后）
	}
	return len(blockText(b))
}

// alignOffsetFor 按段落对齐算整行/整图相对左边距的水平偏移。
func alignOffsetFor(align string, lineWidth, contentWidth float64) float64 {
	free := contentWidth - lineWidth
	if free <= 0 {
		return 0
	}
	switch align {
	case "center":
		return free / 2
	case "right":
		return free
	}
	return 0 // 默认左对齐
}

func styleAt(b RichBlock, offset int) spanStyle {
	pos := 0
	for _, sp := range b.Spans {
		n := len([]rune(sp.Text))
		if offset < pos+n || (offset == pos+n && offset > 0) {
			return spanStyle{sp.Bold, sp.Italic, sp.Underline, sp.Strike, sp.Color, sp.Size}
		}
		pos += n
	}
	if len(b.Spans) > 0 {
		last := b.Spans[len(b.Spans)-1]
		return spanStyle{last.Bold, last.Italic, last.Underline, last.Strike, last.Color, last.Size}
	}
	return spanStyle{}
}

// ─── Build / Layout ──────────────────────────────────────────────

func (e *RichTextElement) Build() []Element { e.children = nil; return nil }

func (e *RichTextElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := e.rt.Width
	if w <= 0 {
		w = 640
	}
	h := e.rt.Height
	if h <= 0 {
		h = 300
	}
	// 工具栏按钮 rect（自绘，超宽自动换行→多行）
	e.toolRects = e.toolRects[:0]
	x, row := 6.0, 0
	tf := canvas.DefaultFont()
	tf.Size = 14
	for _, t := range rtTools {
		rowTop := float64(row) * rtToolbarH
		if t.kind == "sep" {
			e.toolRects = append(e.toolRects, types.Rect{X: x + 3, Y: rowTop + 8, Width: 1, Height: rtToolbarH - 16})
			x += 9
			continue
		}
		measureLabel := t.label
		if t.kind == "fontsize" {
			measureLabel = "字号 88" // 给“字号 NN”动态文本预留宽度
		}
		bw := canvas.MeasureTextGlobal(measureLabel, tf).Width + 16
		if bw < 28 {
			bw = 28
		}
		if x+bw > w-6 && x > 6 { // 当前行放不下→换行
			row++
			x = 6
			rowTop = float64(row) * rtToolbarH
		}
		e.toolRects = append(e.toolRects, types.Rect{X: x, Y: rowTop + 5, Width: bw, Height: rtToolbarH - 10})
		x += bw + 3
	}
	e.toolbarH = float64(row+1) * rtToolbarH // 总高度（行数 × 行高）
	e.layoutDoc(w - rtPadding*2)
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *RichTextElement) layoutDoc(width float64) {
	e.vlines = e.vlines[:0]
	y := 0.0
	for bi, block := range e.blocks {
		if block.Type == "image" {
			imgW, imgH := 220.0, 130.0 // 无图时的占位框尺寸
			if block.Img != nil {
				b := block.Img.Bounds()
				nw, nh := float64(b.Dx()), float64(b.Dy())
				if nw > 0 && nh > 0 {
					imgW = nw
					maxW := width // 不超过内容宽度
					if maxW > 0 && imgW > maxW {
						imgW = maxW
					}
					imgH = imgW * nh / nw // 按原始宽高比缩放
					if imgH > 460 {        // 过高时再按高度回收一次
						imgH = 460
						imgW = imgH * nw / nh
					}
				}
			}
			e.vlines = append(e.vlines, richVLine{y: y, height: imgH, imgW: imgW, blockIdx: bi, isImage: true,
				caretX: []float64{0, imgW}, alignOffset: alignOffsetFor(block.Align, imgW, width)})
			y += imgH + 6
			continue
		}
		base := blockBaseSize(block.Type)
		lineMax := base // 当前行最大字号（行高/基线随之自适应，支持 span 级字号）
		indent := 0.0
		prefix := ""
		switch block.Type {
		case "ul":
			indent = 22
			prefix = "•"
		case "ol":
			indent = 26
			prefix = "1."
		case "quote":
			indent = 14
		}

		type gch struct {
			r   rune
			st  RichSpan
			w   float64
			fnt canvas.Font
		}
		var chars []gch
		for _, sp := range block.Spans {
			f := spanFont(sp, base, block.Type)
			for _, r := range sp.Text {
				chars = append(chars, gch{r, sp, canvas.MeasureTextGlobal(string(r), f).Width, f})
			}
		}

		x := indent
		caretX := []float64{indent}
		var runs []richRun
		startOffset := 0
		offset := 0
		curText := ""
		var curFont canvas.Font
		curUnder := false
		curStrike := false
		var curColor *types.Color
		curRunX := indent
		flushRun := func() {
			if curText != "" {
				runs = append(runs, richRun{text: curText, font: curFont, under: curUnder, strike: curStrike, color: curColor, x: curRunX, w: x - curRunX})
				curText = ""
			}
		}
		newLine := func(first bool) {
			flushRun()
			px := prefix
			if !first {
				px = ""
			}
			lineH := lineMax * 1.7 // 行高随该行最大字号自适应
			e.vlines = append(e.vlines, richVLine{runs: runs, y: y, height: lineH, maxSize: lineMax, blockIdx: bi,
				startOffset: startOffset, caretX: caretX, prefix: px, indent: indent,
				alignOffset: alignOffsetFor(block.Align, x, width)})
			y += lineH
			runs = nil
			x = indent
			caretX = []float64{indent}
			startOffset = offset
			curRunX = indent
			lineMax = base
		}

		first := true
		for _, c := range chars {
			if x+c.w > width && x > indent {
				newLine(first)
				first = false
			}
			if curText != "" && (curFont != c.fnt || curUnder != c.st.Underline || curStrike != c.st.Strike || !colorEq(curColor, c.st.Color)) {
				flushRun()
				curRunX = x
			}
			if curText == "" {
				curRunX = x
				curFont = c.fnt
				curUnder = c.st.Underline
				curStrike = c.st.Strike
				curColor = c.st.Color
			}
			if c.fnt.Size > lineMax { // 跟踪行内最大字号
				lineMax = c.fnt.Size
			}
			curText += string(c.r)
			x += c.w
			offset++
			caretX = append(caretX, x)
		}
		newLine(first)
	}
	e.contentH = y
}

// ─── Paint ───────────────────────────────────────────────────────

func (e *RichTextElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	w, h := e.size.Width, e.size.Height

	// 外层一体容器：白底 + 边框圆角（聚焦时主色边框）
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, w, h, 6, bg)
	border := elBorder()
	if e.focused {
		border = elPrimary()
	}
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, w-1, h-1, 6, bp)

	// 工具栏（自绘，浅灰底 + 底分隔线）
	e.paintToolbar(cvs, pos, w)

	// 编辑区（无缝，工具栏下方）
	editTop := pos.Y + e.toolbarH
	clipTop := editTop + 6
	clipH := h - e.toolbarH - 12
	maxScroll := e.contentH - clipH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.scrollY > maxScroll {
		e.scrollY = maxScroll
	}
	if e.scrollY < 0 {
		e.scrollY = 0
	}

	cvs.Save()
	cvs.ClipRect(pos.X+1, clipTop, w-2, clipH)
	originX := pos.X + rtPadding
	originY := clipTop - e.scrollY
	lo, hi, hasSel := e.selRange()

	for _, vl := range e.vlines {
		ly := originY + vl.y
		if ly+vl.height < clipTop || ly > clipTop+clipH {
			continue
		}
		if vl.isImage {
			imgW := vl.imgW
			if imgW <= 0 {
				imgW = 220
			}
			imgX := originX + vl.alignOffset // 按段落对齐摆放（左/中/右）
			blk := e.blocks[vl.blockIdx]
			if blk.Img != nil {
				// 真实图片：等比绘入计算好的矩形
				cvs.DrawImage(blk.Img, types.Rect{X: imgX, Y: ly + 4, Width: imgW, Height: vl.height - 8})
			} else {
				// 占位框（无图或加载失败）
				fr := paint.DefaultPaint()
				fr.Color = elFill()
				cvs.DrawRoundedRect(imgX, ly+4, imgW, vl.height-8, 6, fr)
				f := canvas.DefaultFont()
				f.Size = 14
				canvas.DrawTextAligned(cvs, "🖼  图片", types.Rect{X: imgX, Y: ly, Width: imgW, Height: vl.height}, f, elTextSecondary(), canvas.HAlignCenter, canvas.VAlignMiddle)
			}
			// 始终画边框；选中该图片块时用主色粗边高亮
			fb := paint.DefaultStrokePaint()
			fb.Color = elBorder()
			fb.StrokeWidth = 1
			if e.cursor.block == vl.blockIdx {
				fb.Color = elPrimary()
				fb.StrokeWidth = 2
			}
			cvs.DrawRoundedRect(imgX+0.5, ly+4.5, imgW-1, vl.height-9, 6, fb)
			continue
		}
		bt := e.blocks[vl.blockIdx].Type
		size := vl.maxSize // 用行内最大字号定基线（一行多字号时按最大者对齐基线）
		if size <= 0 {
			size = blockBaseSize(bt)
		}
		asc := canvas.GetFaceAscent(size)
		desc := canvas.GetFaceDescent(size)
		baseY := ly + vl.height/2 + (asc-desc)/2 // 文字基线（垂直居中行）
		lineX := originX + vl.alignOffset        // 按段落对齐摆放整行

		if bt == "quote" {
			qp := paint.DefaultPaint()
			qp.Color = elBorder()
			cvs.DrawRect(originX, ly+3, 3, vl.height-6, qp)
		}
		if vl.prefix != "" {
			pf := canvas.DefaultFont()
			pf.Size = size
			cvs.DrawText(vl.prefix, lineX, baseY, pf, mkPaint(elTextRegular()))
		}
		if hasSel {
			e.paintLineSelection(cvs, vl, lineX, ly, lo, hi)
		}
		for _, run := range vl.runs {
			rc := elTextPrimary()
			if run.color != nil {
				rc = *run.color
			}
			cvs.DrawText(run.text, lineX+run.x, baseY, run.font, mkPaint(rc))
			if run.under {
				up := paint.DefaultStrokePaint()
				up.Color = rc
				up.StrokeWidth = 1
				cvs.DrawLine(lineX+run.x, baseY+2, lineX+run.x+run.w, baseY+2, up)
			}
			if run.strike { // 删除线：约在文字中线高度
				sp := paint.DefaultStrokePaint()
				sp.Color = rc
				sp.StrokeWidth = 1
				sy := baseY - run.font.Size*0.3
				cvs.DrawLine(lineX+run.x, sy, lineX+run.x+run.w, sy, sp)
			}
		}
	}

	// 光标（按文字基线对齐 + 闪烁）
	if e.focused && e.isCursorVisible() {
		cx, top, bot := e.caretScreen(originX, originY)
		if bot >= clipTop && top <= clipTop+clipH {
			cp := paint.DefaultStrokePaint()
			cp.Color = elTextPrimary()
			cp.StrokeWidth = 1.4
			cvs.DrawLine(cx, top, cx, bot, cp)
		}
	}

	// 文本拖放：插入点指示（橙色竖线）
	if e.textDragging {
		dx, dtop, dbot := e.caretScreenAt(e.dropPos, originX, originY)
		dp := paint.DefaultStrokePaint()
		dp.Color = types.ColorFromRGB(245, 130, 32)
		dp.StrokeWidth = 2
		cvs.DrawLine(dx, dtop, dx, dbot, dp)
	}
	cvs.Restore()

	// 竖滚动条
	if maxScroll > 0 {
		barW := 6.0
		bx := pos.X + w - barW - 3
		thumbH := clipH * clipH / e.contentH
		if thumbH < 20 {
			thumbH = 20
		}
		ratio := e.scrollY / maxScroll
		thumbY := clipTop + ratio*(clipH-thumbH)
		th := paint.DefaultPaint()
		th.Color = types.ColorFromRGB(193, 193, 193)
		cvs.DrawRoundedRect(bx, thumbY, barW, thumbH, 3, th)
	}
}

func mkPaint(c types.Color) paint.Paint {
	p := paint.DefaultPaint()
	p.Color = c
	return p
}

func (e *RichTextElement) paintToolbar(cvs canvas.Canvas, pos types.Point, w float64) {
	tb := paint.DefaultPaint()
	tb.Color = elFill()
	cvs.DrawRect(pos.X+1, pos.Y+1, w-2, e.toolbarH-1, tb)
	ln := paint.DefaultStrokePaint()
	ln.Color = elBorder()
	cvs.DrawLine(pos.X+1, pos.Y+e.toolbarH, pos.X+w-1, pos.Y+e.toolbarH, ln)

	tf := canvas.DefaultFont()
	tf.Size = 14
	for i, t := range rtTools {
		r := e.toolRects[i]
		rx := types.Rect{X: pos.X + r.X, Y: pos.Y + r.Y, Width: r.Width, Height: r.Height}
		if t.kind == "sep" {
			sp := paint.DefaultPaint()
			sp.Color = elBorder()
			cvs.DrawRect(rx.X, rx.Y, rx.Width, rx.Height, sp)
			continue
		}
		active := e.toolActive(t)
		txtColor := elTextRegular()
		if active {
			hl := paint.DefaultPaint()
			hl.Color = types.ColorFromRGB(217, 236, 255)
			cvs.DrawRoundedRect(rx.X, rx.Y, rx.Width, rx.Height, 4, hl)
			txtColor = elPrimary()
		} else if e.hoverTool == i {
			hv := paint.DefaultPaint()
			hv.Color = types.ColorFromRGB(236, 238, 241)
			cvs.DrawRoundedRect(rx.X, rx.Y, rx.Width, rx.Height, 4, hv)
		}
		bf := tf
		if t.arg == "bold" || active {
			bf.Weight = canvas.FontWeightBold
		}
		if t.arg == "italic" {
			bf.Style = canvas.FontStyleItalic
		}
		if t.kind == "colorpick" {
			e.paintColorButton(cvs, rx, tf)
			continue
		}
		label := t.label
		if t.kind == "fontsize" {
			label = "字号 " + strconv.FormatFloat(e.curSize(), 'g', -1, 64) // 显示当前字号
		}
		canvas.DrawTextAligned(cvs, label, rx, bf, txtColor, canvas.HAlignCenter, canvas.VAlignMiddle)
	}
}

// paintColorButton 绘制颜色分体按钮：左“A”(显示当前色) + 当前色下划条 ┊ 右“▾”(弹拾色器)。
func (e *RichTextElement) paintColorButton(cvs canvas.Canvas, rx types.Rect, tf canvas.Font) {
	cur := elTextPrimary()
	if e.lastColor != nil {
		cur = *e.lastColor
	}
	split := rx.X + rx.Width*0.62
	aRect := types.Rect{X: rx.X, Y: rx.Y, Width: split - rx.X, Height: rx.Height}
	bf := tf
	bf.Weight = canvas.FontWeightBold
	canvas.DrawTextAligned(cvs, "A", aRect, bf, cur, canvas.HAlignCenter, canvas.VAlignMiddle)
	// 当前色下划条（A 下方）
	bar := paint.DefaultPaint()
	bar.Color = cur
	cvs.DrawRoundedRect(aRect.X+4, aRect.Y+aRect.Height-6, aRect.Width-8, 3, 1.5, bar)
	// 分隔竖线
	dv := paint.DefaultPaint()
	dv.Color = elBorder()
	cvs.DrawRect(split, rx.Y+5, 1, rx.Height-10, dv)
	// 右侧 ▾ 自绘小三角（不依赖字体字形）
	cx := split + (rx.X+rx.Width-split)/2
	cy := rx.Y + rx.Height/2
	tp := paint.DefaultStrokePaint()
	tp.Color = elTextRegular()
	tp.StrokeWidth = 1.3
	cvs.DrawLine(cx-4, cy-2, cx, cy+2, tp)
	cvs.DrawLine(cx, cy+2, cx+4, cy-2, tp)
}

func (e *RichTextElement) toolActive(t rtTool) bool {
	switch t.kind {
	case "style":
		st := e.currentStyle()
		switch t.arg {
		case "bold":
			return st.bold
		case "italic":
			return st.italic
		case "underline":
			return st.underline
		case "strike":
			return st.strike
		}
	case "block":
		if e.cursor.block < len(e.blocks) {
			return e.blocks[e.cursor.block].Type == t.arg
		}
	case "align":
		if e.cursor.block < len(e.blocks) {
			a := e.blocks[e.cursor.block].Align
			if a == "" {
				a = "left" // 默认左对齐
			}
			return a == t.arg
		}
	}
	return false
}

func (e *RichTextElement) paintLineSelection(cvs canvas.Canvas, vl richVLine, originX, ly float64, lo, hi richPos) {
	nChars := len(vl.caretX) - 1
	for i := 0; i < nChars; i++ {
		p := richPos{vl.blockIdx, vl.startOffset + i}
		if !richPosLess(p, lo) && richPosLess(p, hi) {
			sp := paint.DefaultPaint()
			sp.Color = types.ColorFromRGBA(64, 158, 255, 70)
			cvs.DrawRect(originX+vl.caretX[i], ly+2, vl.caretX[i+1]-vl.caretX[i], vl.height-4, sp)
		}
	}
}

// caretScreen 返回光标屏幕 x、顶部 y、底部 y（按文字基线 ascent/descent 对齐）。
func (e *RichTextElement) caretScreen(originX, originY float64) (x, top, bot float64) {
	return e.caretScreenAt(e.cursor, originX, originY)
}

// caretScreenAt 返回任意 richPos 的光标屏幕位置（供光标 + 拖放插入点指示复用）。
func (e *RichTextElement) caretScreenAt(c richPos, originX, originY float64) (x, top, bot float64) {
	for _, vl := range e.vlines {
		if vl.blockIdx != c.block {
			continue
		}
		lineX := originX + vl.alignOffset
		ly := originY + vl.y
		if vl.isImage { // 图片块：光标贴图片左/右边，竖跨整图高
			idx := c.offset
			if idx >= len(vl.caretX) {
				idx = len(vl.caretX) - 1
			}
			return lineX + vl.caretX[idx], ly + 4, ly + vl.height - 4
		}
		nChars := len(vl.caretX) - 1
		if c.offset >= vl.startOffset && c.offset <= vl.startOffset+nChars {
			idx := c.offset - vl.startOffset
			size := vl.maxSize
			if size <= 0 {
				size = blockBaseSize(e.blocks[vl.blockIdx].Type)
			}
			asc := canvas.GetFaceAscent(size)
			desc := canvas.GetFaceDescent(size)
			baseY := ly + vl.height/2 + (asc-desc)/2
			return lineX + vl.caretX[idx], baseY - asc, baseY + desc
		}
	}
	for _, vl := range e.vlines {
		if vl.blockIdx == c.block {
			return originX + vl.alignOffset + vl.indent, originY + vl.y + 4, originY + vl.y + vl.height - 4
		}
	}
	return originX, originY, originY + 20
}

// ─── 闪烁/焦点 ───────────────────────────────────────────────────

func (e *RichTextElement) isCursorVisible() bool {
	if !e.focused {
		return false
	}
	return time.Since(e.focusTime).Milliseconds()%1000 < 530
}
func (e *RichTextElement) Focus() {
	e.focused = true
	e.focusTime = time.Now()
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}
func (e *RichTextElement) Blur() {
	e.focused = false
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}
func (e *RichTextElement) IsFocused() bool { return e.focused }

// resetBlink 光标移动后立即可见。
func (e *RichTextElement) resetBlink() { e.focusTime = time.Now() }

// ─── 选区 ────────────────────────────────────────────────────────

func (e *RichTextElement) hasSelection() bool { return !richPosEq(e.cursor, e.anchor) }
func (e *RichTextElement) selRange() (lo, hi richPos, has bool) {
	if richPosEq(e.cursor, e.anchor) {
		return e.cursor, e.cursor, false
	}
	if richPosLess(e.cursor, e.anchor) {
		return e.cursor, e.anchor, true
	}
	return e.anchor, e.cursor, true
}

func (e *RichTextElement) posFromXY(sx, sy float64) richPos {
	pos := e.Offset()
	originX := pos.X + rtPadding
	originY := pos.Y + e.toolbarH + 6 - e.scrollY
	var best richVLine
	found := false
	for _, vl := range e.vlines {
		top := originY + vl.y
		if sy >= top && sy < top+vl.height {
			best = vl
			found = true
			break
		}
	}
	if !found && len(e.vlines) > 0 {
		if sy < originY {
			best = e.vlines[0]
		} else {
			best = e.vlines[len(e.vlines)-1]
		}
	}
	lx := sx - originX - best.alignOffset
	idx := 0
	for i := 0; i < len(best.caretX); i++ {
		if best.caretX[i] <= lx {
			idx = i
		}
	}
	if idx < len(best.caretX)-1 && lx-best.caretX[idx] > best.caretX[idx+1]-lx {
		idx++
	}
	return richPos{best.blockIdx, best.startOffset + idx}
}

// ─── 编辑（span 操作）────────────────────────────────────────────

func splitSpansAt(spans []RichSpan, offset int) (left, right []RichSpan) {
	pos := 0
	for _, sp := range spans {
		n := len([]rune(sp.Text))
		if offset >= pos+n {
			left = append(left, sp)
			pos += n
			continue
		}
		if offset <= pos {
			right = append(right, sp)
			pos += n
			continue
		}
		rs := []rune(sp.Text)
		cut := offset - pos
		l := sp
		l.Text = string(rs[:cut])
		r := sp
		r.Text = string(rs[cut:])
		left = append(left, l)
		right = append(right, r)
		pos += n
	}
	return
}

func normalizeSpans(spans []RichSpan) []RichSpan {
	var out []RichSpan
	for _, sp := range spans {
		if sp.Text == "" {
			continue
		}
		if len(out) > 0 {
			last := &out[len(out)-1]
			if last.Bold == sp.Bold && last.Italic == sp.Italic && last.Underline == sp.Underline &&
				last.Strike == sp.Strike && colorEq(last.Color, sp.Color) && last.Size == sp.Size {
				last.Text += sp.Text
				continue
			}
		}
		out = append(out, sp)
	}
	if len(out) == 0 {
		out = []RichSpan{{Text: ""}}
	}
	return out
}

func (e *RichTextElement) insertText(s string) {
	e.recordUndo("insert")
	if e.hasSelection() {
		e.deleteSelection()
	}
	// 图片块不可输入文字：先把光标挪到相邻文本块
	if e.blocks[e.cursor.block].Type == "image" {
		e.moveCursorOutOfImage()
	}
	// 新输入继承光标前一个字符的格式（含颜色/字号）；无选区设色/设字号时由 pending 提供。
	st := e.currentStyle()
	b := &e.blocks[e.cursor.block]
	left, right := splitSpansAt(b.Spans, e.cursor.offset)
	mid := RichSpan{Text: s, Bold: st.bold, Italic: st.italic, Underline: st.underline,
		Strike: st.strike, Color: st.color, Size: st.size}
	b.Spans = normalizeSpans(append(append(left, mid), right...))
	e.cursor.offset += len([]rune(s))
	e.anchor = e.cursor
	e.pending = nil
	e.afterEdit()
}

// moveCursorOutOfImage 当光标停在图片块时，按在图片前(offset 0)/后(offset>=1)
// 跳到相邻文本块；若相邻没有文本块则新建一个空段落容纳输入。
func (e *RichTextElement) moveCursorOutOfImage() {
	bi := e.cursor.block
	if e.cursor.offset == 0 {
		if bi > 0 && e.blocks[bi-1].Type != "image" {
			e.cursor = richPos{bi - 1, blockLen(e.blocks[bi-1])}
		} else {
			e.blocks = insertBlockAt(e.blocks, bi, RichBlock{Type: "p", Spans: []RichSpan{{Text: ""}}})
			e.cursor = richPos{bi, 0}
		}
	} else {
		if bi < len(e.blocks)-1 && e.blocks[bi+1].Type != "image" {
			e.cursor = richPos{bi + 1, 0}
		} else {
			e.blocks = insertBlockAt(e.blocks, bi+1, RichBlock{Type: "p", Spans: []RichSpan{{Text: ""}}})
			e.cursor = richPos{bi + 1, 0}
		}
	}
	e.anchor = e.cursor
}

// insertBlockAt 在 idx 处插入块。
func insertBlockAt(blocks []RichBlock, idx int, b RichBlock) []RichBlock {
	nb := append([]RichBlock{}, blocks[:idx]...)
	nb = append(nb, b)
	nb = append(nb, blocks[idx:]...)
	return nb
}

func (e *RichTextElement) deleteSelection() {
	lo, hi, has := e.selRange()
	if !has {
		return
	}
	if lo.block == hi.block {
		b := &e.blocks[lo.block]
		left, _ := splitSpansAt(b.Spans, lo.offset)
		_, right := splitSpansAt(b.Spans, hi.offset)
		b.Spans = normalizeSpans(append(left, right...))
	} else {
		lb := e.blocks[lo.block]
		hb := e.blocks[hi.block]
		left, _ := splitSpansAt(lb.Spans, lo.offset)
		_, right := splitSpansAt(hb.Spans, hi.offset)
		merged := RichBlock{Type: lb.Type, Spans: normalizeSpans(append(left, right...))}
		nb := append([]RichBlock{}, e.blocks[:lo.block]...)
		nb = append(nb, merged)
		nb = append(nb, e.blocks[hi.block+1:]...)
		e.blocks = nb
	}
	e.cursor = lo
	e.anchor = lo
	e.afterEdit()
}

func (e *RichTextElement) deleteBackward() {
	e.recordUndo("delete")
	if e.hasSelection() {
		e.deleteSelection()
		return
	}
	if e.cursor.offset > 0 {
		b := &e.blocks[e.cursor.block]
		left, _ := splitSpansAt(b.Spans, e.cursor.offset-1)
		_, right := splitSpansAt(b.Spans, e.cursor.offset)
		b.Spans = normalizeSpans(append(left, right...))
		e.cursor.offset--
		e.anchor = e.cursor
		e.afterEdit()
	} else if e.cursor.block > 0 {
		prev := e.cursor.block - 1
		prevLen := blockLen(e.blocks[prev])
		e.blocks[prev].Spans = normalizeSpans(append(e.blocks[prev].Spans, e.blocks[e.cursor.block].Spans...))
		e.blocks = append(e.blocks[:e.cursor.block], e.blocks[e.cursor.block+1:]...)
		e.cursor = richPos{prev, prevLen}
		e.anchor = e.cursor
		e.afterEdit()
	}
}

func (e *RichTextElement) splitBlock() {
	e.recordUndo("split")
	if e.hasSelection() {
		e.deleteSelection()
	}
	b := e.blocks[e.cursor.block]
	left, right := splitSpansAt(b.Spans, e.cursor.offset)
	rightType := b.Type
	if rightType == "h1" || rightType == "h2" || rightType == "h3" || rightType == "quote" {
		rightType = "p"
	}
	nb := append([]RichBlock{}, e.blocks[:e.cursor.block]...)
	nb = append(nb, RichBlock{Type: b.Type, Spans: normalizeSpans(left)}, RichBlock{Type: rightType, Spans: normalizeSpans(right)})
	nb = append(nb, e.blocks[e.cursor.block+1:]...)
	e.blocks = nb
	e.cursor = richPos{e.cursor.block + 1, 0}
	e.anchor = e.cursor
	e.afterEdit()
}

// ─── 格式 ────────────────────────────────────────────────────────

func (e *RichTextElement) currentStyle() spanStyle {
	if e.pending != nil {
		return *e.pending
	}
	if e.cursor.block >= len(e.blocks) {
		return spanStyle{}
	}
	b := e.blocks[e.cursor.block]
	off := e.cursor.offset
	if off > 0 {
		off--
	}
	return styleAt(b, off)
}

// curSize 返回光标处/选区处的当前字号（span 未设则取块默认字号），供工具栏「字号」显示。
func (e *RichTextElement) curSize() float64 {
	sz := e.currentStyle().size
	if sz <= 0 {
		if e.cursor.block < len(e.blocks) {
			sz = blockBaseSize(e.blocks[e.cursor.block].Type)
		} else {
			sz = 15
		}
	}
	return sz
}

func (e *RichTextElement) toggleStyle(kind string) {
	e.recordUndo("style")
	if !e.hasSelection() {
		st := e.currentStyle()
		switch kind {
		case "bold":
			st.bold = !st.bold
		case "italic":
			st.italic = !st.italic
		case "underline":
			st.underline = !st.underline
		case "strike":
			st.strike = !st.strike
		}
		e.pending = &st
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return
	}
	lo, hi, _ := e.selRange()
	allOn := e.selAllHas(kind, lo, hi)
	e.applyStyleRange(lo, hi, kind, !allOn)
	e.afterEditKeepSel()
}

func (e *RichTextElement) selAllHas(kind string, lo, hi richPos) bool {
	for bi := lo.block; bi <= hi.block; bi++ {
		b := e.blocks[bi]
		start, end := 0, blockLen(b)
		if bi == lo.block {
			start = lo.offset
		}
		if bi == hi.block {
			end = hi.offset
		}
		pos := 0
		for _, sp := range b.Spans {
			n := len([]rune(sp.Text))
			s0, s1 := pos, pos+n
			pos += n
			if s1 <= start || s0 >= end {
				continue
			}
			on := false
			switch kind {
			case "bold":
				on = sp.Bold
			case "italic":
				on = sp.Italic
			case "underline":
				on = sp.Underline
			case "strike":
				on = sp.Strike
			}
			if !on {
				return false
			}
		}
	}
	return true
}

func (e *RichTextElement) applyStyleRange(lo, hi richPos, kind string, on bool) {
	for bi := lo.block; bi <= hi.block; bi++ {
		b := &e.blocks[bi]
		start, end := 0, blockLen(*b)
		if bi == lo.block {
			start = lo.offset
		}
		if bi == hi.block {
			end = hi.offset
		}
		if start >= end {
			continue
		}
		l, midRight := splitSpansAt(b.Spans, start)
		mid, r := splitSpansAt(midRight, end-start)
		for i := range mid {
			switch kind {
			case "bold":
				mid[i].Bold = on
			case "italic":
				mid[i].Italic = on
			case "underline":
				mid[i].Underline = on
			case "strike":
				mid[i].Strike = on
			}
		}
		b.Spans = normalizeSpans(append(append(l, mid...), r...))
	}
}

// clearFormat 清除选区文字的所有内联样式（粗斜下划删除线/颜色/字号），只保留文字。
func (e *RichTextElement) clearFormat() {
	lo, hi, ok := e.selRange()
	if !ok {
		e.pending = &spanStyle{} // 无选区：清待应用样式
		e.lastColor = nil
		return
	}
	e.recordUndo("clear")
	for bi := lo.block; bi <= hi.block; bi++ {
		b := &e.blocks[bi]
		start, end := 0, blockLen(*b)
		if bi == lo.block {
			start = lo.offset
		}
		if bi == hi.block {
			end = hi.offset
		}
		if start >= end {
			continue
		}
		l, midRight := splitSpansAt(b.Spans, start)
		mid, r := splitSpansAt(midRight, end-start)
		for i := range mid {
			mid[i] = RichSpan{Text: mid[i].Text} // 只留文字，清所有样式
		}
		b.Spans = normalizeSpans(append(append(l, mid...), r...))
	}
	e.afterEditKeepSel()
}

func (e *RichTextElement) setBlockType(t string) {
	e.recordUndo("block")
	lo, hi, _ := e.selRange()
	for bi := lo.block; bi <= hi.block; bi++ {
		e.blocks[bi].Type = t
	}
	e.afterEditKeepSel()
}

// setAlign 设置选区涉及段落（含图片块）的水平对齐。
func (e *RichTextElement) setAlign(a string) {
	e.recordUndo("align")
	lo, hi, _ := e.selRange()
	for bi := lo.block; bi <= hi.block; bi++ {
		e.blocks[bi].Align = a
	}
	e.afterEditKeepSel()
}

// ── 右键菜单动作 ──

// richRangeText 取选区范围内的纯文本（跨块以 \n 连接）。
func (e *RichTextElement) richRangeText(lo, hi richPos) string {
	var sb strings.Builder
	for bi := lo.block; bi <= hi.block && bi < len(e.blocks); bi++ {
		if bi > lo.block {
			sb.WriteString("\n")
		}
		rs := blockText(e.blocks[bi])
		start, end := 0, len(rs)
		if bi == lo.block {
			start = lo.offset
		}
		if bi == hi.block {
			end = hi.offset
		}
		if start < 0 {
			start = 0
		}
		if end > len(rs) {
			end = len(rs)
		}
		if start < end {
			sb.WriteString(string(rs[start:end]))
		}
	}
	return sb.String()
}

func (e *RichTextElement) ctxCopy() {
	if !e.hasSelection() || ClipboardWrite == nil {
		return
	}
	lo, hi, _ := e.selRange()
	ClipboardWrite(e.richRangeText(lo, hi))
}

// ── 文本拖放（把选中文本拖到别处移动）──

// globalOffOf 把 richPos 换算为整文档扁平 rune 偏移（每块计 blockLen+1，+1 为块分隔）。
func (e *RichTextElement) globalOffOf(p richPos) int {
	g := 0
	for b := 0; b < p.block && b < len(e.blocks); b++ {
		g += blockLen(e.blocks[b]) + 1
	}
	return g + p.offset
}

// posFromGlobal 把扁平偏移换算回 richPos。
func (e *RichTextElement) posFromGlobal(g int) richPos {
	if g < 0 {
		g = 0
	}
	for b := 0; b < len(e.blocks); b++ {
		bl := blockLen(e.blocks[b])
		if g <= bl {
			return richPos{b, g}
		}
		g -= bl + 1
	}
	last := len(e.blocks) - 1
	if last < 0 {
		return richPos{0, 0}
	}
	return richPos{last, blockLen(e.blocks[last])}
}

// posInSel 判断位置是否落在选区内（含边界）。
func (e *RichTextElement) posInSel(p richPos) bool {
	if !e.hasSelection() {
		return false
	}
	lo, hi, _ := e.selRange()
	return !richPosLess(p, lo) && !richPosLess(hi, p)
}

// finishTextDragRich 把选中文本（纯文本）移动到 dropPos：删原选区→扁平偏移校正目标→插入。
func (e *RichTextElement) finishTextDragRich() {
	if !e.hasSelection() {
		return
	}
	lo, hi, _ := e.selRange()
	drop := e.dropPos
	if !richPosLess(drop, lo) && !richPosLess(hi, drop) {
		return // 落在选区内→取消
	}
	text := e.richRangeText(lo, hi)
	gLo, gHi, gDrop := e.globalOffOf(lo), e.globalOffOf(hi), e.globalOffOf(drop)
	e.deleteSelection()
	if gDrop > gLo {
		gDrop -= gHi - gLo // 删除选区后目标前移
	}
	e.cursor = e.posFromGlobal(gDrop)
	e.anchor = e.cursor
	for i, part := range strings.Split(text, "\n") {
		if i > 0 {
			e.splitBlock()
		}
		if part != "" {
			e.insertText(part)
		}
	}
	e.afterEdit()
}

func (e *RichTextElement) ctxCut() {
	if !e.hasSelection() || ClipboardWrite == nil {
		return
	}
	lo, hi, _ := e.selRange()
	ClipboardWrite(e.richRangeText(lo, hi))
	e.deleteSelection()
}

func (e *RichTextElement) ctxPaste() {
	if ClipboardRead == nil {
		return
	}
	for i, part := range strings.Split(ClipboardRead(), "\n") {
		if i > 0 {
			e.splitBlock() // 换行→分段
		}
		if part != "" {
			e.insertText(part)
		}
	}
}

func (e *RichTextElement) selectAllRich() {
	e.anchor = richPos{0, 0}
	last := len(e.blocks) - 1
	e.cursor = richPos{last, blockLen(e.blocks[last])}
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// defaultContextItems 富文本默认右键菜单：剪切/复制/粘贴/全选。
func (e *RichTextElement) defaultContextItems() []MenuItem {
	hasSel := e.hasSelection()
	return []MenuItem{
		{Label: "剪切", Enabled: hasSel && ClipboardWrite != nil, OnClick: e.ctxCut},
		{Label: "复制", Enabled: hasSel && ClipboardWrite != nil, OnClick: e.ctxCopy},
		{Label: "粘贴", Enabled: ClipboardRead != nil, OnClick: e.ctxPaste},
		{Label: "全选", Enabled: true, OnClick: e.selectAllRich},
	}
}

func colorEq(a, b *types.Color) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func hexColor(hex string) types.Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 6 {
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return types.ColorFromRGB(uint8(r), uint8(g), uint8(b))
	}
	return elTextPrimary()
}

// applyColor 给选区文字应用颜色（空 hex=恢复默认色）；颜色仅作用于选区。
func (e *RichTextElement) applyColor(hex string) {
	if !e.hasSelection() {
		return
	}
	var c *types.Color
	if hex != "" {
		col := hexColor(hex)
		c = &col
	}
	lo, hi, _ := e.selRange()
	for bi := lo.block; bi <= hi.block; bi++ {
		b := &e.blocks[bi]
		start, end := 0, blockLen(*b)
		if bi == lo.block {
			start = lo.offset
		}
		if bi == hi.block {
			end = hi.offset
		}
		if start >= end {
			continue
		}
		l, midRight := splitSpansAt(b.Spans, start)
		mid, r := splitSpansAt(midRight, end-start)
		for i := range mid {
			mid[i].Color = c
		}
		b.Spans = normalizeSpans(append(append(l, mid...), r...))
	}
	e.afterEditKeepSel()
}

// ─── 公共 ────────────────────────────────────────────────────────

func (e *RichTextElement) clampCursor() {
	if e.cursor.block < 0 {
		e.cursor.block = 0
	}
	if e.cursor.block >= len(e.blocks) {
		e.cursor.block = len(e.blocks) - 1
	}
	bl := blockLen(e.blocks[e.cursor.block])
	if e.cursor.offset < 0 {
		e.cursor.offset = 0
	}
	if e.cursor.offset > bl {
		e.cursor.offset = bl
	}
}

func (e *RichTextElement) afterEdit() {
	e.clampCursor()
	e.pending = nil
	e.resetBlink()
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	} else if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// afterEditKeepSel 改样式/块类型后保留选区（不折叠光标）。
func (e *RichTextElement) afterEditKeepSel() {
	e.resetBlink()
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	} else if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// ─── 事件 ────────────────────────────────────────────────────────

func (e *RichTextElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeContextMenu:
		if e.rt.ContextMenuDisabled {
			return false // 禁用：右键不弹出
		}
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		items := e.rt.ContextMenuItems
		if len(items) == 0 {
			items = e.defaultContextItems()
		}
		ShowContextMenu(me.X, me.Y, items)
		return true
	case event.TypeMouseWheel:
		if me, ok := ev.(*event.MouseEvent); ok && me.DeltaY != 0 {
			pos := e.Offset()
			if me.Y-pos.Y < e.toolbarH {
				return false
			}
			e.scrollY -= me.DeltaY * 40
			if e.scrollY < 0 {
				e.scrollY = 0
			}
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
	case event.TypeMouseMove:
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		pos := e.Offset()
		ly := me.Y - pos.Y
		// 文本拖放：在选区内按下后移动超阈值→拖动，实时更新插入点
		if e.textDragArmed {
			if !e.textDragging {
				dx, dy := me.X-e.dragDownX, me.Y-e.dragDownY
				if dx*dx+dy*dy > 16 {
					e.textDragging = true
					if RequestPointerCapture != nil {
						RequestPointerCapture(e)
					}
				}
			}
			if e.textDragging {
				e.dropPos = e.posFromXY(me.X, me.Y)
				if OnNeedsRepaint != nil {
					OnNeedsRepaint()
				}
			}
			return true
		}
		if e.dragging {
			e.cursor = e.posFromXY(me.X, me.Y)
			e.resetBlink()
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		// 工具栏 hover
		nh := -1
		if ly < e.toolbarH {
			lx := me.X - pos.X
			for i, t := range rtTools {
				if t.kind == "sep" {
					continue
				}
				r := e.toolRects[i]
				if lx >= r.X && lx <= r.X+r.Width && ly >= r.Y && ly <= r.Y+r.Height {
					nh = i
					break
				}
			}
		}
		if nh != e.hoverTool {
			e.hoverTool = nh
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		return true
	case event.TypeMouseLeave:
		if e.hoverTool != -1 {
			e.hoverTool = -1
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		return true
	case event.TypeMouseDown:
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		if me.Button == event.ButtonRight {
			e.focused = true // 右键也聚焦，但不移动光标/不清选区（保留选区供复制），菜单交给 ContextMenu
			return false
		}
		pos := e.Offset()
		lx, ly := me.X-pos.X, me.Y-pos.Y
		if ly < e.toolbarH {
			// 工具栏点击
			for i, t := range rtTools {
				if t.kind == "sep" {
					continue
				}
				r := e.toolRects[i]
				if lx >= r.X && lx <= r.X+r.Width && ly >= r.Y && ly <= r.Y+r.Height {
					e.focused = true
					switch t.kind {
					case "style":
						e.toggleStyle(t.arg)
					case "block":
						e.setBlockType(t.arg)
					case "align":
						e.setAlign(t.arg)
					case "colorpick":
						ax, ay := pos.X+r.X, pos.Y+r.Y+r.Height
						if lx > r.X+r.Width*0.62 {
							e.openColorPalette(ax, ay) // 右侧 ▾：弹拾色器选新色
						} else {
							e.applyCurrentColor(ax, ay) // 左侧 A：应用最近选定的颜色
						}
					case "fontsize":
						e.openFontSizeMenu(pos.X+r.X, pos.Y+r.Y+r.Height)
					case "image":
						e.insertImage()
					case "clear":
						e.clearFormat()
					}
					return true
				}
			}
			return true
		}
		cp := e.posFromXY(me.X, me.Y)
		if e.posInSel(cp) {
			// 在选中文本上按下：待判定拖动文本/单击
			e.focused = true
			e.textDragArmed = true
			e.dragArmPos = cp
			e.dragDownX, e.dragDownY = me.X, me.Y
			return true
		}
		e.focused = true
		e.resetBlink()
		e.cursor = cp
		e.anchor = cp
		e.dragging = true
		e.pending = nil
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	case event.TypeMouseUp:
		if e.textDragging {
			e.finishTextDragRich()
			e.textDragging = false
			e.textDragArmed = false
			return true
		}
		if e.textDragArmed { // 只点了一下选区（没拖）→ 清选区、光标落到点击处
			e.cursor = e.dragArmPos
			e.anchor = e.cursor
			e.textDragArmed = false
			e.resetBlink()
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		e.dragging = false
		return true
	case event.TypeKeyChar:
		keyEv, ok := ev.(*event.KeyEvent)
		if !ok || !e.focused {
			break
		}
		if keyEv.Char >= 32 {
			e.insertText(string(keyEv.Char))
			return true
		}
	case event.TypeKeyDown:
		keyEv, ok := ev.(*event.KeyEvent)
		if !ok || !e.focused {
			break
		}
		if keyEv.Mods&event.ModCtrl != 0 {
			switch keyEv.Key {
			case "B":
				e.toggleStyle("bold")
			case "I":
				e.toggleStyle("italic")
			case "U":
				e.toggleStyle("underline")
			case "Z":
				if keyEv.Mods&event.ModShift != 0 {
					e.redo()
				} else {
					e.undo()
				}
			case "Y":
				e.redo()
			}
			return true
		}
		switch keyEv.Key {
		case "Backspace":
			e.deleteBackward()
		case "Enter":
			e.splitBlock()
		case "ArrowLeft":
			e.moveCursor(-1)
		case "ArrowRight":
			e.moveCursor(1)
		case "ArrowUp":
			e.moveCursorLine(-1)
		case "ArrowDown":
			e.moveCursorLine(1)
		case "Home":
			e.cursor.offset = 0
			e.anchor = e.cursor
			e.resetBlink()
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		case "End":
			e.cursor.offset = blockLen(e.blocks[e.cursor.block])
			e.anchor = e.cursor
			e.resetBlink()
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		return true
	}
	return false
}

func (e *RichTextElement) moveCursor(dir int) {
	e.cursor.offset += dir
	if e.cursor.offset < 0 {
		if e.cursor.block > 0 {
			e.cursor.block--
			e.cursor.offset = blockLen(e.blocks[e.cursor.block])
		} else {
			e.cursor.offset = 0
		}
	} else if e.cursor.offset > blockLen(e.blocks[e.cursor.block]) {
		if e.cursor.block < len(e.blocks)-1 {
			e.cursor.block++
			e.cursor.offset = 0
		} else {
			e.cursor.offset = blockLen(e.blocks[e.cursor.block])
		}
	}
	e.anchor = e.cursor
	e.pending = nil
	e.resetBlink()
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

func (e *RichTextElement) moveCursorLine(dir int) {
	pos := e.Offset()
	originX := pos.X + rtPadding
	cx, _, _ := e.caretScreen(originX, 0)
	curLine := -1
	for i, vl := range e.vlines {
		nChars := len(vl.caretX) - 1
		if vl.blockIdx == e.cursor.block && e.cursor.offset >= vl.startOffset && e.cursor.offset <= vl.startOffset+nChars {
			curLine = i
			break
		}
	}
	if curLine < 0 {
		return
	}
	target := curLine + dir
	if target < 0 || target >= len(e.vlines) {
		return
	}
	vl := e.vlines[target]
	idx := 0
	lx := cx - originX
	for i := 0; i < len(vl.caretX); i++ {
		if vl.caretX[i] <= lx {
			idx = i
		}
	}
	e.cursor = richPos{vl.blockIdx, vl.startOffset + idx}
	e.anchor = e.cursor
	e.resetBlink()
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

func (e *RichTextElement) Update(newWidget Widget) {
	if nr, ok := newWidget.(*RichText); ok {
		e.rt = nr
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── 颜色选择（弹出色板）──────────────────────────────────────────

func (e *RichTextElement) applyColorVal(c *types.Color) {
	e.recordUndo("color")
	if !e.hasSelection() {
		return
	}
	lo, hi, _ := e.selRange()
	for bi := lo.block; bi <= hi.block; bi++ {
		b := &e.blocks[bi]
		start, end := 0, blockLen(*b)
		if bi == lo.block {
			start = lo.offset
		}
		if bi == hi.block {
			end = hi.offset
		}
		if start >= end {
			continue
		}
		l, midRight := splitSpansAt(b.Spans, start)
		mid, r := splitSpansAt(midRight, end-start)
		for i := range mid {
			mid[i].Color = c
		}
		b.Spans = normalizeSpans(append(append(l, mid...), r...))
	}
	e.afterEditKeepSel()
}

// setPendingColor 无选区时记下颜色，影响下次输入（输入后靠继承光标前字符延续同色）。
func (e *RichTextElement) setPendingColor(c *types.Color) {
	st := e.currentStyle()
	st.color = c
	e.pending = &st
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// applyColorPick 应用一个具体颜色：有选区染选区、无选区设待应用色。两种情形都记为 lastColor。
func (e *RichTextElement) applyColorPick(c *types.Color) {
	e.lastColor = c
	if e.hasSelection() {
		e.applyColorVal(c)
	} else {
		e.setPendingColor(c)
	}
}

// applyCurrentColor 颜色按钮“主体”行为：直接套用最近选定的颜色（没有则等同弹选择器）。
func (e *RichTextElement) applyCurrentColor(ax, ay float64) {
	if e.lastColor == nil {
		e.openColorPalette(ax, ay)
		return
	}
	c := *e.lastColor
	e.applyColorPick(&c)
}

// openColorPalette 颜色按钮“▾”行为：弹出拾色器选新色。
func (e *RichTextElement) openColorPalette(ax, ay float64) {
	var id int
	cp := NewColorPicker().WithOnChange(func(hex string) {
		col := hexColor(hex)
		e.applyColorPick(&col)
	})
	id = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: ax, Y: ay, Width: 1, Height: 1},
		Content:        cp,
		Placement:      PlacementBottomStart,
		Gap:            2,
		Transition:     "fade",
		OnClickOutside: func() { HideOverlay(id) },
	})
}

// ─── 字号 ────────────────────────────────────────────────────────

var rtFontSizes = []float64{12, 14, 16, 18, 20, 24, 28, 36}

// setPendingSize 无选区时记下字号，影响下次输入（之后靠继承延续）。
func (e *RichTextElement) setPendingSize(size float64) {
	st := e.currentStyle()
	st.size = size
	e.pending = &st
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// applySizeRange 给选区文字设字号。
func (e *RichTextElement) applySizeRange(size float64) {
	lo, hi, _ := e.selRange()
	for bi := lo.block; bi <= hi.block; bi++ {
		b := &e.blocks[bi]
		start, end := 0, blockLen(*b)
		if bi == lo.block {
			start = lo.offset
		}
		if bi == hi.block {
			end = hi.offset
		}
		if start >= end {
			continue
		}
		l, midRight := splitSpansAt(b.Spans, start)
		mid, r := splitSpansAt(midRight, end-start)
		for i := range mid {
			mid[i].Size = size
		}
		b.Spans = normalizeSpans(append(append(l, mid...), r...))
	}
	e.afterEditKeepSel()
}

// applyFontSize 字号选择：有选区设选区、无选区设待应用。
func (e *RichTextElement) applyFontSize(size float64) {
	if e.hasSelection() {
		e.applySizeRange(size)
	} else {
		e.setPendingSize(size)
	}
}

func (e *RichTextElement) openFontSizeMenu(ax, ay float64) {
	var id int
	menu := &rtSizeMenu{onPick: func(sz float64) {
		e.applyFontSize(sz)
		HideOverlay(id)
	}}
	id = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: ax, Y: ay, Width: 1, Height: 1},
		Content:        menu,
		Placement:      PlacementBottomStart,
		Gap:            2,
		Transition:     "fade",
		OnClickOutside: func() { HideOverlay(id) },
	})
}

// rtSizeMenu 字号下拉菜单（浮层内容，自绘竖列；与 ColorPicker 同为浮层内容组件）。
type rtSizeMenu struct {
	StatelessWidget
	onPick func(float64)
}

func (m *rtSizeMenu) CreateElement() Element {
	return &rtSizeMenuElement{BaseElement: BaseElement{widget: m}, menu: m, hover: -1}
}

const (
	rtSizeRowH  = 30.0
	rtSizeMenuW = 84.0
)

type rtSizeMenuElement struct {
	BaseElement
	menu  *rtSizeMenu
	hover int
}

func (e *rtSizeMenuElement) Build() []Element { return nil }

func (e *rtSizeMenuElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	h := rtSizeRowH*float64(len(rtFontSizes)) + 8
	e.size = ctx.Constraints.Constrain(types.Size{Width: rtSizeMenuW, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *rtSizeMenuElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	sh := paint.DefaultPaint()
	sh.Color = types.ColorFromRGBA(0, 0, 0, 24)
	cvs.DrawRoundedRect(pos.X, pos.Y+2, e.size.Width, e.size.Height, 6, sh)
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, 6, bg)
	bd := paint.DefaultStrokePaint()
	bd.Color = elBorder()
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, e.size.Width-1, e.size.Height-1, 6, bd)
	for i, sz := range rtFontSizes {
		ry := pos.Y + 4 + float64(i)*rtSizeRowH
		rect := types.Rect{X: pos.X + 4, Y: ry, Width: e.size.Width - 8, Height: rtSizeRowH}
		if e.hover == i {
			hv := paint.DefaultPaint()
			hv.Color = types.ColorFromRGB(236, 245, 255)
			cvs.DrawRoundedRect(rect.X, rect.Y, rect.Width, rect.Height, 4, hv)
		}
		f := canvas.DefaultFont()
		f.Size = 14
		canvas.DrawTextAligned(cvs, strconv.FormatFloat(sz, 'g', -1, 64), rect, f, elTextPrimary(), canvas.HAlignCenter, canvas.VAlignMiddle)
	}
}

func (e *rtSizeMenuElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	idx := int((me.Y - pos.Y - 4) / rtSizeRowH)
	switch ev.Type() {
	case event.TypeMouseMove:
		if idx >= 0 && idx < len(rtFontSizes) && idx != e.hover {
			e.hover = idx
			e.MarkNeedsPaint()
		}
		return true
	case event.TypeMouseDown:
		if idx >= 0 && idx < len(rtFontSizes) && e.menu.onPick != nil {
			e.menu.onPick(rtFontSizes[idx])
		}
		return true
	}
	return false
}

func (e *rtSizeMenuElement) Update(newWidget Widget) {
	if nm, ok := newWidget.(*rtSizeMenu); ok {
		e.menu = nm
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── 插入图片 ────────────────────────────────────────────────────

// insertImage 弹出系统文件对话框选图，解码后在光标所在块后插入图片块。
// 取消选择或加载失败则不插入。未桥接文件对话框的平台插入占位图片块。
func (e *RichTextElement) insertImage() {
	e.recordUndo("image")
	block := RichBlock{Type: "image"}
	if OpenFileDialog != nil {
		path := OpenFileDialog("插入图片", "图片文件|*.png;*.jpg;*.jpeg;*.bmp;*.gif\n所有文件|*.*")
		if path == "" {
			return // 用户取消
		}
		img, err := NewImageFromFile(path)
		if err != nil || img == nil || img.Source == nil {
			return // 解码失败，放弃插入
		}
		block.Img = img.Source
		block.Path = path
	}
	cur := e.blocks[e.cursor.block]
	if cur.Type == "image" {
		// 已在图片块上：其后插入新图片 + 空段落（落点在新段落，可继续打字）
		nb := insertBlockAt(e.blocks, e.cursor.block+1, block)
		nb = insertBlockAt(nb, e.cursor.block+2, RichBlock{Type: "p", Spans: []RichSpan{{Text: ""}}})
		e.blocks = nb
		e.cursor = richPos{e.cursor.block + 2, 0}
		e.anchor = e.cursor
		e.afterEdit()
		return
	}
	// 在光标处把当前段落一分为二，图片夹在中间：[左半][图片][右半]
	// 这样图片前(左半)、图片后(右半)都有可编辑段落，可把图片摆到文字之前/之间。
	left, right := splitSpansAt(cur.Spans, e.cursor.offset)
	leftBlock := RichBlock{Type: cur.Type, Spans: normalizeSpans(left), Align: cur.Align}
	rightBlock := RichBlock{Type: cur.Type, Spans: normalizeSpans(right), Align: cur.Align}
	nb := append([]RichBlock{}, e.blocks[:e.cursor.block]...)
	nb = append(nb, leftBlock, block, rightBlock)
	nb = append(nb, e.blocks[e.cursor.block+1:]...)
	e.blocks = nb
	e.cursor = richPos{e.cursor.block + 2, 0} // 落到图片后的段落开头
	e.anchor = e.cursor
	e.afterEdit()
}
