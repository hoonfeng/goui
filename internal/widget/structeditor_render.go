package widget

import (
	"strings"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// StructEditor 的表格化渲染：左侧统一行号栏 + 程序集变量表 + 子程序(参数表/局部变量表/内嵌代码编辑器)。

// StructEditor 配色已移入 Theme.StructEditor（见 structeditor_theme.go 的 seXxx() 取色函数），随主题换肤。

func (e *StructEditorElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	w, h := e.size.Width, e.size.Height
	e.lastCanvas = cvs // Skia 测量（中文光标不漂移）

	bg := paint.DefaultPaint()
	bg.Color = seBackground()
	cvs.DrawRoundedRect(pos.X, pos.Y, w, h, 6, bg)
	border := elBorder()
	if e.focused {
		border = elPrimary()
	}
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, w-1, h-1, 6, bp)

	miniW := 0.0
	if e.showMinimap && w > 320 { // 太窄不显示缩略图
		miniW = seMiniW
	}

	cvs.Save()
	cvs.ClipRect(pos.X+1, pos.Y+1, w-2-miniW, h-2)
	e.cells = e.cells[:0]
	e.foldHits = e.foldHits[:0]
	e.miniSegs = e.miniSegs[:0]
	x := pos.X + sePad
	innerW := w - sePad*2 - miniW
	top := pos.Y + sePad - e.scrollY
	y := top

	gw := e.gutterWidth() // 统一行号栏宽度：变量表行号列与代码行号栏对齐成最左一条
	gl := 1               // 全局行号：贯穿 程序集变量→各子程序(参数+局部+代码)，整份是一个整体编辑器

	// 导入区（go 模式总显示，让代码结构完整且可从零加；单列可编辑/增删/末尾回车追加）
	if e.se.lang == "go" {
		n := len(e.program.Imports)
		if n == 0 {
			n = 1 // 空时 1 个占位行，点击即可添加第一个 import
		}
		importVars := make([]SEVar, n)
		for i := 0; i < len(e.program.Imports); i++ {
			importVars[i].Name = e.program.Imports[i]
		}
		iStart := y
		y = e.paintVarTable(cvs, x, y, innerW, gw, "imports", importCols, importVars, gl, false, false)
		e.miniSegs = append(e.miniSegs, seMiniSeg{iStart - top, y - iStart, segTable})
		gl += n
		y += 12
	}

	// 程序集变量表：与局部变量表同样式（无独立标题栏，首列即表头，可折叠）
	gStart := y
	y = e.paintVarTable(cvs, x, y, innerW, gw, "globals", e.curSchema().Globals, e.program.Globals, gl, true, e.globalsCollapsed)
	e.miniSegs = append(e.miniSegs, seMiniSeg{gStart - top, y - gStart, segTable})
	gl += len(e.program.Globals) // 行号始终累加（折叠也连续，与子程序折叠一致）
	y += 12

	for si := range e.program.Subs {
		sub := &e.program.Subs[si]
		cel := e.ensureBodyEd(si, innerW)
		collapsed := e.collapsed[si]
		// 函数声明 + 参数：合并成一张表（函数名/返回类型 是表内首行，蓝底 + 折叠三角）
		fStart := y
		y = e.paintFuncTable(cvs, x, y, innerW, gw, sub, si, collapsed, gl)
		e.miniSegs = append(e.miniSegs, seMiniSeg{fStart - top, y - fStart, segFunc})
		if collapsed { // 收起：跳过内容，但 gl 继续累加保持全局行号连续
			gl += 1 + len(sub.Params) + len(sub.Returns) + len(sub.Locals) + len(cel.lines)
			y += 8
			continue
		}
		gl += 1 + len(sub.Params) + len(sub.Returns) // 函数声明 1 行 + 参数 + 返回值各 1 行
		// 局部变量表（schema 驱动，首列即表头「局部变量」）+ 逻辑代码 合为一段缩略
		cStart := y
		y = e.paintVarTable(cvs, x, y, innerW, gw, "locals:"+itoaCE(si), e.curSchema().Locals, sub.Locals, gl, false, false)
		gl += len(sub.Locals)
		// 逻辑代码：内嵌 CodeEditor，行号接续全局计数，直接接在局部变量表后
		cel.ed.LineNumberOffset = gl - 1
		cel.ed.GutterOverride = gw
		bh := e.layoutBodyEd(cel, innerW)
		cel.SetPosition(types.Point{X: x - pos.X, Y: y - pos.Y})
		cel.Paint(cvs, types.Point{})
		gl += len(cel.lines)
		y += bh + 10
		e.miniSegs = append(e.miniSegs, seMiniSeg{cStart - top, y - cStart, segCode})
	}
	e.contentH = y - top
	if e.se.ScrollRef != nil { // 同步滚动位置，供切换视图后恢复
		*e.se.ScrollRef = e.scrollY
	}
	cvs.Restore()

	if miniW > 0 { // 缩略图（其视口框代替竖滚动条）
		e.paintSEMinimap(cvs, pos.X+w-miniW-1, pos.Y+2, h-4)
		return
	}
	e.miniRect = types.Rect{}
	// 竖滚动条
	viewH := h - 4
	if e.contentH > viewH {
		barW := 6.0
		bx := pos.X + w - barW - 3
		thumbH := viewH * viewH / e.contentH
		if thumbH < 24 {
			thumbH = 24
		}
		ratio := e.scrollY / (e.contentH - viewH)
		thumbY := pos.Y + 2 + ratio*(viewH-thumbH)
		th := paint.DefaultPaint()
		th.Color = types.ColorFromRGB(193, 193, 193)
		cvs.DrawRoundedRect(bx, thumbY, barW, thumbH, 3, th)
	}
}

const seMiniW = 72.0 // 缩略图宽

// seMiniSeg 缩略图的一个内容段（相对 content 顶的偏移 + 高度 + 类型）。
type seMiniSeg struct {
	off, h float64
	kind   int
}

const (
	segTable = iota // 变量表（灰）
	segFunc         // 函数声明行/参数（蓝）
	segCode         // 局部 + 逻辑代码（深灰）
)

// paintSEMinimap 画右侧缩略图：内容按 contentH 比例缩成色块段 + 当前视口框，点击/拖动跳转。
func (e *StructEditorElement) paintSEMinimap(cvs canvas.Canvas, miniX, areaTop, areaH float64) {
	e.miniRect = types.Rect{X: miniX, Y: areaTop, Width: seMiniW, Height: areaH}
	bg := paint.DefaultPaint()
	bg.Color = seMinimapBG()
	cvs.DrawRect(miniX, areaTop, seMiniW, areaH, bg)
	sep := paint.DefaultStrokePaint()
	sep.Color = seLineColor()
	cvs.DrawLine(miniX, areaTop, miniX, areaTop+areaH, sep)
	if e.contentH <= 0 {
		return
	}
	scale := areaH / e.contentH
	if scale > 1 {
		scale = 1 // 内容不足一屏：1:1，不放大
	}
	cvs.Save()
	cvs.ClipRect(miniX, areaTop, seMiniW, areaH)
	for _, s := range e.miniSegs {
		yy := areaTop + s.off*scale
		hh := s.h * scale
		if hh < 1.5 {
			hh = 1.5
		}
		p := paint.DefaultPaint()
		p.Color = segColor(s.kind)
		cvs.DrawRect(miniX+5, yy+0.5, seMiniW-10, hh-1, p)
	}
	cvs.Restore()
	// 当前视口框
	viewH := e.size.Height - 4
	if e.contentH > viewH {
		boxTop := areaTop + e.scrollY*scale
		boxH := viewH * scale
		vp := paint.DefaultPaint()
		vp.Color = withAlpha(elPrimary(), 36)
		cvs.DrawRect(miniX, boxTop, seMiniW, boxH, vp)
		vb := paint.DefaultStrokePaint()
		vb.Color = withAlpha(elPrimary(), 120)
		cvs.DrawRect(miniX+0.5, boxTop, seMiniW-1, boxH, vb)
	}
}

func segColor(kind int) types.Color {
	switch kind {
	case segFunc:
		return seFuncRowBG()
	case segCode:
		return seGutterNum() // 代码区用行号灰，区别于表格
	default:
		return seHeaderBG()
	}
}

// minimapJump 点击/拖动缩略图 my 处 → 滚到对应内容位置（居中）。
func (e *StructEditorElement) minimapJump(my float64) {
	if e.miniRect.Height <= 0 || e.contentH <= 0 {
		return
	}
	scale := e.miniRect.Height / e.contentH
	if scale > 1 {
		scale = 1
	}
	viewH := e.size.Height - 4
	target := (my-e.miniRect.Y)/scale - viewH/2
	maxScroll := e.contentH - viewH
	if maxScroll < 0 {
		maxScroll = 0
	}
	e.scrollY = clamp(target, 0, maxScroll)
	repaint()
}

// inMinimap 点是否落在缩略图区域。
func (e *StructEditorElement) inMinimap(x, y float64) bool {
	r := e.miniRect
	return r.Width > 0 && x >= r.X && x <= r.X+r.Width && y >= r.Y && y <= r.Y+r.Height
}

// paintSectionTitle 区段标题条（如「程序集变量表」「子程序 阶乘」）。
func (e *StructEditorElement) paintSectionTitle(cvs canvas.Canvas, x, y, w float64, text string) float64 {
	bar := paint.DefaultPaint()
	bar.Color = seFuncRowBG()
	cvs.DrawRoundedRect(x, y, w, seHeaderH, 3, bar)
	f := canvas.DefaultFont()
	f.Size = 13
	f.Weight = canvas.FontWeightBold
	canvas.DrawTextAligned(cvs, text, types.Rect{X: x + 8, Y: y, Width: w - 16, Height: seHeaderH}, f, elPrimary(), canvas.HAlignLeft, canvas.VAlignMiddle)
	return y + seHeaderH + 4
}

// gutterWidth 统一行号栏宽度：按全文档总行数的位数算（与内嵌代码编辑器同公式），
// 使变量表行号列与代码行号栏等宽、数字右对齐到同一位置，连成最左一条整体行号栏。
func (e *StructEditorElement) gutterWidth() float64 {
	total := len(e.program.Globals)
	nImp := len(e.program.Imports)
	if e.se.lang == "go" && nImp == 0 {
		nImp = 1 // 空导入占位行也占一个行号
	}
	total += nImp
	for i := range e.program.Subs {
		s := &e.program.Subs[i]
		total += len(s.Params) + len(s.Locals)
		total += strings.Count(expandTabs(s.Body), "\n") + 1
	}
	digits := len(itoaCE(total))
	if digits < 2 {
		digits = 2
	}
	charW := canvas.MeasureTextGlobal("0", canvas.Font{Family: "Consolas", Size: 14}).Width
	return float64(digits)*charW + ceGutterPad + ceFoldW
}

// paintTableRows 通用画表（schema 驱动）：行号列 + 表头(可折叠三角) + 数据行(文本/复选框单元格) + 列行网格盒。
// 列由 cols 定义，单元格值经 v.field(列.Field) 读、Check 列画复选框。不画行号栏右竖分隔线/最外圆角框（调用者按独立表/子表补）。
// foldable&collapsed 时只画表头。startLine 是本表首行的全局行号。返回表底部 y。
func (e *StructEditorElement) paintTableRows(cvs canvas.Canvas, x, y, w, gw float64, section string, cols []SECol, vars []SEVar, startLine int, foldable, collapsed bool) float64 {
	contentX := x + gw
	tw := w - gw
	numAreaW := gw - ceGutterPad - ceFoldW // 数字右对齐区，与代码行号栏一致
	cw := colWidths(cols, tw)
	heads := make([]string, len(cols))
	for i := range cols {
		heads[i] = cols[i].Title
	}
	gb := paint.DefaultPaint()
	gb.Color = seGutterBG()
	cvs.DrawRect(x, y, gw, seHeaderH, gb)
	if foldable { // 表头折叠三角 + 命中区（sub=-1 标识程序集变量表）
		e.paintFoldTri(cvs, x+gw-ceFoldW, y, seHeaderH, collapsed)
		e.foldHits = append(e.foldHits, seFoldHit{rect: types.Rect{X: x, Y: y, Width: gw, Height: seHeaderH}, sub: -1})
	}
	e.paintGridHeader(cvs, contentX, y, heads, cw)
	hy := y + seHeaderH
	if foldable && collapsed { // 收起：只画表头外框
		bp := paint.DefaultStrokePaint()
		bp.Color = seLineColor()
		bp.StrokeWidth = 1
		cvs.DrawRect(contentX+0.5, y+0.5, tw-1, seHeaderH-1, bp)
		return hy
	}
	nf := canvas.DefaultFont()
	nf.Size = 11
	for r := range vars {
		ry := hy + float64(r)*seRowH
		cvs.DrawRect(x, ry, gw, seRowH, gb) // 行号列灰底 + 右对齐全局行号
		canvas.DrawTextAligned(cvs, itoaCE(startLine+r), types.Rect{X: x, Y: ry, Width: numAreaW, Height: seRowH}, nf, seGutterNum(), canvas.HAlignRight, canvas.VAlignMiddle)
		col := contentX
		for ci := range cols {
			if cols[ci].Check { // 复选框列（如参考/传址）
				e.paintCheckCell(cvs, col, ry, cw[ci], vars[r].field(cols[ci].Field) == "是", section, r, ci)
			} else {
				e.paintCell(cvs, col, ry, cw[ci], vars[r].field(cols[ci].Field), section, r, ci, ci == 0)
			}
			col += cw[ci]
		}
	}
	bottom := hy + float64(len(vars))*seRowH
	e.paintGridBox(cvs, contentX, y, tw, bottom-y, colEdges(cw)) // 列分隔线用累计边界，与单元格严格一致
	return bottom
}

// paintVarTable 独立变量表（程序集变量/局部变量）：通用画表 + 行号栏右竖分隔线。foldable 仅程序集表用。
func (e *StructEditorElement) paintVarTable(cvs canvas.Canvas, x, y, w, gw float64, section string, cols []SECol, vars []SEVar, startLine int, foldable, collapsed bool) float64 {
	bottom := e.paintTableRows(cvs, x, y, w, gw, section, cols, vars, startLine, foldable, collapsed)
	contentX := x + gw
	sep := paint.DefaultStrokePaint() // 行号栏右分隔线（与代码编辑器 gutter 一致）
	sep.Color = elBorder()
	sep.StrokeWidth = 1
	cvs.DrawLine(contentX, y, contentX, bottom, sep)
	return bottom + 4 // 增行靠回车（末尾行回车追加），不再画「+ 增行」按钮
}

// paintFuncTable 函数声明 + 参数合并成一张表：
// 首行（蓝底）是函数声明——「函数名」标签+可编辑框 + 「返回值」标签+可编辑框（带折叠三角，占行号 declLine）；
// 其下是「参数 | 类型 | 数组 | 备注」子表（注明参数）。collapsed 时只画函数声明行。
func (e *StructEditorElement) paintFuncTable(cvs canvas.Canvas, x, y, w, gw float64, sub *SESub, si int, collapsed bool, declLine int) float64 {
	contentX := x + gw
	tw := w - gw
	numAreaW := gw - ceGutterPad - ceFoldW
	top := y
	nf := canvas.DefaultFont()
	nf.Size = 11
	lf := canvas.DefaultFont()
	lf.Size = 12
	sec := "func:" + itoaCE(si)

	// ── 函数声明行（蓝底）：行号 + 折叠三角 + 「函数名」标签+可编辑框 + 「返回值」标签+可编辑框 ──
	gb := paint.DefaultPaint()
	gb.Color = seGutterBG()
	cvs.DrawRect(x, y, gw, seRowH, gb)
	canvas.DrawTextAligned(cvs, itoaCE(declLine), types.Rect{X: x, Y: y, Width: numAreaW, Height: seRowH}, nf, seGutterNum(), canvas.HAlignRight, canvas.VAlignMiddle)
	e.paintFoldTri(cvs, x+gw-ceFoldW, y, seRowH, collapsed)
	e.foldHits = append(e.foldHits, seFoldHit{rect: types.Rect{X: x, Y: y, Width: gw, Height: seRowH}, sub: si})
	bar := paint.DefaultPaint()
	bar.Color = seFuncRowBG()
	cvs.DrawRect(contentX, y, tw, seRowH, bar)
	lx := contentX + 8
	canvas.DrawTextAligned(cvs, "函数名", types.Rect{X: lx, Y: y, Width: 40, Height: seRowH}, lf, seHeadText(), canvas.HAlignLeft, canvas.VAlignMiddle)
	lx += 40
	nameW := tw * 0.30
	e.paintEditCell(cvs, lx, y, nameW, sub.Name, sec, 0)
	lx += nameW + 12
	canvas.DrawTextAligned(cvs, "注释", types.Rect{X: lx, Y: y, Width: 30, Height: seRowH}, lf, seHeadText(), canvas.HAlignLeft, canvas.VAlignMiddle)
	lx += 30
	noteW := contentX + tw - lx - 8 // 注释框用剩余宽度
	if noteW < 80 {
		noteW = 80
	}
	e.paintEditCell(cvs, lx, y, noteW, sub.Note, sec, 1) // 返回值移到下方返回值子表（多返回值一行一个）
	y += seRowH

	// ── 参数子表 + 返回值子表（schema 驱动，复用通用画表 paintTableRows）──
	if !collapsed {
		sc := e.curSchema()
		y = e.paintTableRows(cvs, x, y, w, gw, "params:"+itoaCE(si), sc.Params, sub.Params, declLine+1, false, false)
		if len(sub.Returns) > 0 {
			y = e.paintTableRows(cvs, x, y, w, gw, "returns:"+itoaCE(si), sc.Returns, sub.Returns, declLine+1+len(sub.Params), false, false)
		}
	}
	// 外框 + 行号栏右分隔线
	bp := paint.DefaultStrokePaint()
	bp.Color = seLineColor()
	bp.StrokeWidth = 1
	cvs.DrawRect(contentX+0.5, top+0.5, tw-1, y-top-1, bp)
	sep := paint.DefaultStrokePaint()
	sep.Color = elBorder()
	sep.StrokeWidth = 1
	cvs.DrawLine(contentX, top, contentX, y, sep)
	return y + 4
}

// paintEditCell 画一个「看起来可编辑」的输入框单元格（白底+细框）+ 登记可编辑单元格（选中/光标/写回）。
func (e *StructEditorElement) paintEditCell(cvs canvas.Canvas, x, y, w float64, text, section string, col int) {
	wb := paint.DefaultPaint()
	wb.Color = seCellBg()
	cvs.DrawRoundedRect(x, y+3, w, seRowH-6, 2, wb)
	ob := paint.DefaultStrokePaint()
	ob.Color = seLineColor()
	ob.StrokeWidth = 1
	cvs.DrawRoundedRect(x+0.5, y+3.5, w-1, seRowH-7, 2, ob)
	e.paintCell(cvs, x, y, w, text, section, 0, col, false)
}

// paintCheckCell 画一个复选框单元格（参数「参考」列：勾选=传址 by-reference），并登记命中区（点击经 beginEdit 切换）。
func (e *StructEditorElement) paintCheckCell(cvs canvas.Canvas, x, y, w float64, checked bool, section string, row, col int) {
	if e.selSection == section && e.selRow == row && e.selCol == col { // 选中高亮
		sb := paint.DefaultPaint()
		sb.Color = seSelBG()
		cvs.DrawRect(x+1, y+1, w-2, seRowH-2, sb)
	}
	bs := 14.0 // 居中的小方框
	bx := x + (w-bs)/2
	by := y + (seRowH-bs)/2
	box := paint.DefaultPaint()
	if checked {
		box.Color = elPrimary()
		cvs.DrawRoundedRect(bx, by, bs, bs, 3, box)
		ck := paint.DefaultStrokePaint() // 白色对勾
		ck.Color = elSurface()
		ck.StrokeWidth = 1.6
		cvs.DrawLine(bx+3, by+7, bx+6, by+10, ck)
		cvs.DrawLine(bx+6, by+10, bx+11, by+4, ck)
	} else {
		box.Color = seCellBg()
		cvs.DrawRoundedRect(bx, by, bs, bs, 3, box)
		ob := paint.DefaultStrokePaint()
		ob.Color = types.ColorFromRGB(0xC0, 0xC4, 0xCC)
		ob.StrokeWidth = 1.2
		cvs.DrawRoundedRect(bx+0.5, by+0.5, bs-1, bs-1, 3, ob)
	}
	e.cells = append(e.cells, seCellHit{rect: types.Rect{X: x, Y: y, Width: w, Height: seRowH}, section: section, row: row, col: col})
}

// paintFoldTri 折叠三角（折叠▶ / 展开▼），用线段画，避免字体无字形的豆腐块。
func (e *StructEditorElement) paintFoldTri(cvs canvas.Canvas, gx, ly, rowH float64, folded bool) {
	cx := gx + ceFoldW/2
	cy := ly + rowH/2
	ap := paint.DefaultStrokePaint()
	ap.Color = elPrimary()
	ap.StrokeWidth = 1.5
	if folded { // ▶
		cvs.DrawLine(cx-2, cy-4, cx+3, cy, ap)
		cvs.DrawLine(cx+3, cy, cx-2, cy+4, ap)
	} else { // ▼
		cvs.DrawLine(cx-4, cy-2, cx, cy+3, ap)
		cvs.DrawLine(cx, cy+3, cx+4, cy-2, ap)
	}
}

func (e *StructEditorElement) paintGridHeader(cvs canvas.Canvas, x, y float64, heads []string, cw []float64) {
	hb := paint.DefaultPaint()
	hb.Color = seHeaderBG()
	tot := 0.0
	for _, c := range cw {
		tot += c
	}
	cvs.DrawRect(x, y, tot, seHeaderH, hb)
	f := canvas.DefaultFont()
	f.Size = 12
	cx := x
	for i, hd := range heads {
		if hd != "" {
			canvas.DrawTextAligned(cvs, hd, types.Rect{X: cx + 8, Y: y, Width: cw[i] - 10, Height: seHeaderH}, f, seHeadText(), canvas.HAlignLeft, canvas.VAlignMiddle)
		}
		cx += cw[i]
	}
}

// paintCell 画一个数据单元格 + 登记命中区；selected 高亮。
func (e *StructEditorElement) paintCell(cvs canvas.Canvas, x, y, w float64, text, section string, row, col int, leftPad bool) {
	e.paintCellColored(cvs, x, y, w, text, section, row, col, 0, seTextColor())
}

func (e *StructEditorElement) paintCellColored(cvs canvas.Canvas, x, y, w float64, text, section string, row, col int, indent float64, color types.Color) {
	if e.selSection == section && e.selRow == row && e.selCol == col {
		sb := paint.DefaultPaint()
		sb.Color = seSelBG()
		cvs.DrawRect(x+1, y+1, w-2, seRowH-2, sb)
		sp := paint.DefaultStrokePaint()
		sp.Color = elPrimary()
		sp.StrokeWidth = 1.4
		cvs.DrawRect(x+1, y+1, w-2, seRowH-2, sp)
	}
	if text != "" {
		f := canvas.DefaultFont()
		f.Size = 13
		canvas.DrawTextAligned(cvs, text, types.Rect{X: x + 8 + indent, Y: y, Width: w - 12 - indent, Height: seRowH}, f, color, canvas.HAlignLeft, canvas.VAlignMiddle)
	}
	// 编辑光标
	if e.editing && e.selSection == section && e.selRow == row && e.selCol == col {
		rr := []rune(text)
		ec := e.editCol
		if ec > len(rr) {
			ec = len(rr)
		}
		cx := x + 8 + indent + e.measure(string(rr[:ec]))
		e.cellCaretX, e.cellCaretY = cx, y+seRowH // 缓存供 IME 候选定位（不随闪烁变）
		if e.isCellCursorVisible() {
			cp := paint.DefaultStrokePaint()
			cp.Color = seTextColor()
			cp.StrokeWidth = 1.4
			cvs.DrawLine(cx, y+4, cx, y+seRowH-4, cp)
		}
	}
	e.cells = append(e.cells, seCellHit{rect: types.Rect{X: x, Y: y, Width: w, Height: seRowH}, section: section, row: row, col: col})
}

// paintGridBox 画整张表的外框 + 列分隔线 + 行分隔线。
func (e *StructEditorElement) paintGridBox(cvs canvas.Canvas, x, y, w, h float64, colEdges []float64) {
	lp := paint.DefaultStrokePaint()
	lp.Color = seLineColor()
	lp.StrokeWidth = 1
	cvs.DrawRect(x+0.5, y+0.5, w-1, h-1, lp)
	// 列分隔线
	cx := x
	for i := 0; i < len(colEdges); i++ {
		if i < len(colEdges) {
			cx = x + colEdges[i]
		}
		cvs.DrawLine(cx, y, cx, y+h, lp)
	}
	// 行分隔线（表头下 + 各行）
	rows := int((h - seHeaderH) / seRowH)
	for r := 0; r <= rows; r++ {
		ly := y + seHeaderH + float64(r)*seRowH
		cvs.DrawLine(x, ly, x+w, ly, lp)
	}
}
