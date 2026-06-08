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

	stickyH := seRowH // 顶部保留一行给「当前区段」常驻标签，内容裁剪到其下方（不再遮盖内容）
	cvs.Save()
	cvs.ClipRect(pos.X+1, pos.Y+1+stickyH, w-2-miniW, h-2-stickyH)
	e.cells = e.cells[:0]
	e.foldHits = e.foldHits[:0]
	e.miniSegs = e.miniSegs[:0]
	e.secRanges = e.secRanges[:0]
	x := pos.X + sePad
	innerW := w - sePad*2 - miniW
	top := pos.Y + stickyH + sePad - e.scrollY
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
		e.secRanges = append(e.secRanges, secRange{"导入", iStart, y})
		gl += n
		y += 12
	}

	// 常量表（go 模式显示，可折叠）
	if e.se.lang == "go" && len(e.program.Consts) > 0 {
		cStart := y
		y = e.paintVarTable(cvs, x, y, innerW, gw, "consts", constCols, e.program.Consts, gl, true, e.constsCollapsed)
		e.miniSegs = append(e.miniSegs, seMiniSeg{cStart - top, y - cStart, segTable})
		e.secRanges = append(e.secRanges, secRange{"常量", cStart, y})
		gl += len(e.program.Consts)
		y += 12
	}

	// 类型定义（go 模式）：每类型一行头(名|种类|成员摘要|备注)+折叠三角；展开→字段/方法成可编辑子表。
	if e.se.lang == "go" && len(e.program.Types) > 0 {
		for i := range e.program.Types {
			td := &e.program.Types[i]
			tStart := y
			hdrY := y
			hv := []SEVar{{Name: td.Name, Type: string(td.Kind), Array: typeMembersSummary(td), Note: td.Note}}
			y = e.paintVarTable(cvs, x, y, innerW, gw, "type:"+itoaCE(i), typeDefCols, hv, gl, i == 0, false)
			if m := e.typeMembers(i); m != nil && len(*m) > 0 { // 有成员→▸ 提示：点开浮窗看字段表（不再内联展开）
				rowY := hdrY
				if i == 0 {
					rowY += seRowH // 首个类型前多一行「类型定义」表头
				}
				triX := x + innerW - 18
				paintFoldTriangle(cvs, triX, rowY+6, 9, true, seGutterNum()) // ▸ 可点开
				e.foldHits = append(e.foldHits, seFoldHit{rect: types.Rect{X: triX - 3, Y: rowY + 3, Width: 16, Height: seRowH - 6}, section: "type:" + itoaCE(i)})
			}
			gl++
			e.miniSegs = append(e.miniSegs, seMiniSeg{tStart - top, y - tStart, segTable})
			e.secRanges = append(e.secRanges, secRange{"类型定义 › " + td.Name, tStart, y})
		}
		y += 12
	}

	// 程序集变量表：与局部变量表同样式（无独立标题栏，首列即表头，可折叠）
	gStart := y
	y = e.paintVarTable(cvs, x, y, innerW, gw, "globals", e.curSchema().Globals, e.program.Globals, gl, true, e.globalsCollapsed)
	e.miniSegs = append(e.miniSegs, seMiniSeg{gStart - top, y - gStart, segTable})
	e.secRanges = append(e.secRanges, secRange{"程序集变量", gStart, y})
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
			e.secRanges = append(e.secRanges, secRange{"函数 " + sub.Name, fStart, y})
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
		e.secRanges = append(e.secRanges, secRange{"函数 " + sub.Name, fStart, y})
	}
	e.contentH = y - top
	if e.se.ScrollRef != nil { // 同步滚动位置，供切换视图后恢复
		*e.se.ScrollRef = e.scrollY
	}
	cvs.Restore()
	e.paintStickyHeader(cvs, pos, w, miniW, stickyH) // 顶部保留区画「当前区段」常驻标签（不裁剪、不遮内容）

	if miniW > 0 { // 缩略图（其视口框代替竖滚动条）
		e.paintSEMinimap(cvs, pos.X+w-miniW-1, pos.Y+2, h-4)
	} else {
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
	if e.popupType >= 0 { // 类型字段表浮窗：画在最上层（含背景遮罩、可编辑字段表）
		e.paintTypePopup(cvs, pos, w, h)
	} else {
		e.popupRect = types.Rect{}
	}
}

// paintStickyHeader 在顶部保留区画「当前区段」常驻标签（随滚动更新），让用户始终知道在看哪段内容。
// 画在保留的 stickyH 行内（内容已裁剪到其下方），故不遮盖内容。
func (e *StructEditorElement) paintStickyHeader(cvs canvas.Canvas, pos types.Point, w, miniW, stickyH float64) {
	viewTop := pos.Y + stickyH + sePad + 2 // 内容逻辑顶端
	label := ""
	for _, sr := range e.secRanges {
		if sr.y0 <= viewTop { // 起点已滚到视口顶之上 → 当前所在（取最后一个）区段
			label = sr.label
		}
	}
	if label == "" {
		return
	}
	barW := w - 2 - miniW
	bp := paint.DefaultPaint()
	bp.Color = seHeaderBG()
	cvs.DrawRect(pos.X+1, pos.Y+1, barW, stickyH, bp)
	ln := paint.DefaultStrokePaint()
	ln.Color = seLineColor()
	cvs.DrawLine(pos.X+1, pos.Y+1+stickyH, pos.X+1+barW, pos.Y+1+stickyH, ln)
	f, _, _ := canvas.FontWithStyle("monospace", 12, canvas.FontRegular)
	canvas.DrawTextAligned(cvs, "当前："+label, types.Rect{X: pos.X + 10, Y: pos.Y + 1, Width: barW - 16, Height: stickyH},
		f, seHeaderText(), canvas.HAlignLeft, canvas.VAlignMiddle)
}

// paintTypePopup 类型字段表浮窗：点类型行▸/成员列弹出，居中面板显示该类型的字段/方法可编辑表，点外部/关闭收起。
func (e *StructEditorElement) paintTypePopup(cvs canvas.Canvas, pos types.Point, w, h float64) {
	i := e.popupType
	if i < 0 || i >= len(e.program.Types) {
		e.popupType = -1
		return
	}
	td := &e.program.Types[i]
	m := e.typeMembers(i)
	n := 0
	if m != nil {
		n = len(*m)
	}
	// 背景遮罩（点它=点面板外→关闭）
	bd := paint.DefaultPaint()
	bd.Color = types.Color{R: 20, G: 20, B: 25, A: 120}
	cvs.DrawRect(pos.X+1, pos.Y+1, w-2, h-2, bd)
	// 面板尺寸/居中
	pw := w * 0.82
	if pw > 560 {
		pw = 560
	}
	ph := float64(n+2)*seRowH + 18 // 标题 + 列头 + n 字段 + 留白
	if ph > h-24 {
		ph = h - 24
	}
	px := pos.X + (w-pw)/2
	py := pos.Y + (h-ph)/2
	e.popupRect = types.Rect{X: px, Y: py, Width: pw, Height: ph}
	// 面板底 + 边框
	pb := paint.DefaultPaint()
	pb.Color = seBackground()
	cvs.DrawRoundedRect(px, py, pw, ph, 8, pb)
	brd := paint.DefaultStrokePaint()
	brd.Color = elPrimary()
	brd.StrokeWidth = 1.5
	cvs.DrawRoundedRect(px+0.5, py+0.5, pw-1, ph-1, 8, brd)
	// 标题条（类型名 + 关闭）
	hb := paint.DefaultPaint()
	hb.Color = seFuncRowBG()
	cvs.DrawRect(px+1, py+1, pw-2, seRowH, hb)
	f, _, _ := canvas.FontWithStyle("monospace", 13, canvas.FontRegular)
	canvas.DrawTextAligned(cvs, "type "+td.Name+" "+string(td.Kind), types.Rect{X: px + 12, Y: py + 1, Width: pw - 92, Height: seRowH}, f, seFuncRowText(), canvas.HAlignLeft, canvas.VAlignMiddle)
	canvas.DrawTextAligned(cvs, "关闭 ✕", types.Rect{X: px + pw - 80, Y: py + 1, Width: 72, Height: seRowH}, f, seText(), canvas.HAlignRight, canvas.VAlignMiddle)
	// 字段/方法可编辑表（gw=0 无行号栏；showHeader→列标题；注册命中格供点击编辑）
	if m != nil && n > 0 {
		e.paintVarTable(cvs, px+8, py+seRowH+6, pw-16, 0, "typefields:"+itoaCE(i), typeFieldCols, *m, 0, true, false)
	} else {
		canvas.DrawTextAligned(cvs, "（无字段）", types.Rect{X: px + 12, Y: py + seRowH + 10, Width: pw - 24, Height: seRowH}, f, types.ColorFromRGB(150, 150, 150), canvas.HAlignLeft, canvas.VAlignMiddle)
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
	ratio := (my - e.miniRect.Y) / e.miniRect.Height
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	e.scrollY = ratio * (e.contentH - e.size.Height + 4)
	repaint()
}

// ── 行号栏 ──

// gutterWidth 行号栏宽度：按程序最大行号位数算（拼接 import/var/type/const/所有函数体），
// 保证整份文档行号栏统一宽；无内容→0（仅表格视图允许隐藏行号）。
func (e *StructEditorElement) gutterWidth() float64 {
	maxLine := 0
	// imports
	for range e.program.Imports {
		maxLine++
	}
	if len(e.program.Imports) == 0 {
		maxLine++ // 占位行
	}
	// consts
	maxLine += len(e.program.Consts)
	// types
	maxLine += len(e.program.Types)
	// globals
	maxLine += len(e.program.Globals)
	// subs
	for si := range e.program.Subs {
		cel := e.ensureBodyEd(si, 200)
		maxLine += 1 + len(e.program.Subs[si].Params) + len(e.program.Subs[si].Returns) + len(e.program.Subs[si].Locals) + len(cel.lines)
	}
	if maxLine < 10 {
		return 32 // 短文件固定窄行号栏
	}
	digits := 0
	for maxLine > 0 {
		digits++
		maxLine /= 10
	}
	e.font, _, _ = canvas.FontWithStyle("monospace", 13, canvas.FontRegular)
	charW := e.font.Measure("8")
	return float64(digits)*charW + ceGutterPad + ceFoldW
}

// ── 通用变量表 ──

// paintVarTable 画一张变量表（字段名|类型|数组|参考|备注），含行号栏 + 分隔线。
// section 为区段标识（"globals"/"locals:N"/"params:N"/"returns:N"）；caption 为列首名称。
// showHeader=true 在首行画表头并高亮；collapsed=true 只画首行(表头) + 折叠三角。
// 返回下一行 y 坐标（按行高步进）。
func (e *StructEditorElement) paintVarTable(cvs canvas.Canvas, x, y, innerW, gw float64, section string, cols []SECol, vars []SEVar, gl int, showHeader bool, collapsed bool) float64 {
	rows := len(vars)
	if collapsed && rows > 0 {
		rows = 0 // 折叠时只有表头
	}
	colW := colWidths(cols, innerW-gw)
	colE := colEdges(colW)
	x0 := x + gw
	cw := innerW - gw // 内容区宽（不含行号栏）
	// 累计所有线（行号栏竖线以外）到 e.cells，供 hit-test 编辑用
	if showHeader && len(vars) > 0 { // 画表头行（蓝底+加粗+首列名即标题）
		hdrY := y
		titles := make([]string, len(cols)) // 表头：首列=区段名，其余列=各自列标题（不再只有首列）
		titles[0] = sectionTitle(section, cols)
		for ci := 1; ci < len(cols); ci++ {
			titles[ci] = cols[ci].Title
		}
		e.paintRow(cvs, x, hdrY, gw, colW, colE, section, -1, titles, nil, gl, true)
		// 折叠三角（程序集变量/常量整段折叠）：画三角 + 命中区（带 section 供 applyFold 区分）
		if section == "globals" || section == "consts" {
			triX := x0 + cw - 18
			paintFoldTriangle(cvs, triX, hdrY+6, 9, collapsed, seGutterNum())
			e.foldHits = append(e.foldHits, seFoldHit{rect: types.Rect{X: triX - 3, Y: hdrY + 3, Width: 16, Height: seRowH - 6}, sub: -1, section: section})
		}
		y = hdrY + seRowH
	}
	if collapsed {
		return y
	}
	for ri := range vars {
		rowY := y
		vals := make([]string, len(cols))
		for ci, col := range cols {
			vals[ci] = vars[ri].field(col.Field)
		}
		eachW := make([]float64, len(cols))
		for ci := range cols {
			eachW[ci] = colW[ci]
		}
		e.paintRow(cvs, x, rowY, gw, colW, colE, section, ri, vals, eachW, gl, false)
		y = rowY + seRowH
	}
	return y
}

// paintRow 画一行：行号 + 各单元格文本。
func (e *StructEditorElement) paintRow(cvs canvas.Canvas, x, y, gw float64, colW, colE []float64, section string, row int, vals []string, eachW []float64, gl int, isHeader bool) {
	x0 := x + gw
	cw := 0.0
	for _, w := range colW {
		cw += w
	}
	// 行号
	e.drawLineNum(cvs, x, y, gw, gl, isHeader)
	// 行底色
	bgCol := seRowBG()
	if isHeader {
		bgCol = seHeaderBG()
	} else if section == "globals" && row >= 0 && row < len(e.program.Globals) {
		bgCol = seRowBG()
	}
	bp := paint.DefaultPaint()
	bp.Color = bgCol
	cvs.DrawRect(x0, y, cw, seRowH, bp)
	// 单元格
	cellX := x0
	for ci, val := range vals {
		w := colW[ci]
		if w < 1 {
			continue
		}
		// 命中区
		e.cells = append(e.cells, seCellHit{rect: types.Rect{X: cellX, Y: y, Width: w, Height: seRowH}, section: section, row: row, col: ci})
		// 选中/编辑高亮
		if !isHeader && e.selSection == section && e.selRow == row && e.selCol == ci {
			sel := paint.DefaultPaint()
			sel.Color = elPrimary()
			cvs.DrawRect(cellX+1, y+1, w-2, seRowH-2, sel)
		}
		// 文本
		txtCol := seText()
		if isHeader {
			txtCol = seHeaderText()
		}
		e.font, _, _ = canvas.FontWithStyle("monospace", 13, canvas.FontRegular)
		padL := 4.0
		if ci == 0 {
			padL = 8 // 首列左间距大些
		}
		tx := cellX + padL
		// 长度限制
		maxW := w - padL - 4
		display := val
		for e.font.Measure(display) > maxW {
			display = display[:len(display)-1]
		}
		canvas.DrawTextAligned(cvs, display, types.Rect{X: tx, Y: y, Width: w - padL - 4, Height: seRowH}, e.font, txtCol, canvas.HAlignLeft, canvas.VAlignMiddle)
		cellX += w
	}
	// 行分隔线
	border := seLineColor()
	for _, ex := range colE {
		ln := paint.DefaultStrokePaint()
		ln.Color = border
		cvs.DrawLine(x0+ex, y, x0+ex, y+seRowH, ln)
	}
	// 底部分隔线
	ln := paint.DefaultStrokePaint()
	ln.Color = border
	cvs.DrawLine(x0, y+seRowH, x0+cw, y+seRowH, ln)
}

// drawLineNum 画行号（单行）。
func (e *StructEditorElement) drawLineNum(cvs canvas.Canvas, x, y, gw float64, n int, isHeader bool) {
	col := seGutterNum()
	if isHeader {
		col = seGutterNum()
	}
	if n >= 0 && !isHeader {
		e.font, _, _ = canvas.FontWithStyle("monospace", 13, canvas.FontRegular)
		canvas.DrawTextAligned(cvs, itoaCE(n), types.Rect{X: x, Y: y, Width: gw - ceGutterPad - ceFoldW, Height: seRowH},
			e.font, col, canvas.HAlignRight, canvas.VAlignMiddle)
	}
}

// ── 折叠三角 ──

// paintFoldTriangle 画折叠三角（▶ 或 ▼）。（待渲染到 canvas）
func paintFoldTriangle(cvs canvas.Canvas, x, y, size float64, collapsed bool, col types.Color) {
	if collapsed {
		// ▶
		cvs.DrawLine(x+size*0.3, y+size*0.2, x+size*0.7, y+size*0.5, paint.StrokePaint(types.Color{}, size*0.12))
		cvs.DrawLine(x+size*0.7, y+size*0.5, x+size*0.3, y+size*0.8, paint.StrokePaint(types.Color{}, size*0.12))
	} else {
		// ▼
		cvs.DrawLine(x+size*0.2, y+size*0.3, x+size*0.5, y+size*0.7, paint.StrokePaint(types.Color{}, size*0.12))
		cvs.DrawLine(x+size*0.5, y+size*0.7, x+size*0.8, y+size*0.3, paint.StrokePaint(types.Color{}, size*0.12))
	}
}

// ── 函数表 ──

// paintFuncTable 画一个子程序（函数声明 1 行 + 参数表 + 返回值表 + 局部变量表）。
// 折叠时只画声明行（蓝底+折叠三角）+ 折叠虚线 // ...，函数体不展开。
func (e *StructEditorElement) paintFuncTable(cvs canvas.Canvas, x, y, innerW, gw float64, sub *SESub, si int, collapsed bool, gl int) float64 {
	x0 := x + gw
	cw := innerW - gw

	// 函数声明行（蓝底高亮，与易语言一致）
	declY := y
	// 函数名 + 接收器 + 类型参数
	namePart := sub.Name
	if sub.Recv != "" {
		namePart = sub.Recv + " " + namePart
	}
	if len(sub.TypeParams) > 0 {
		tp := typeParamsGo(sub.TypeParams)
		namePart += tp
	}
	// 行号
	e.drawLineNum(cvs, x, declY, gw, gl, false)
	// 行底色
	bg := paint.DefaultPaint()
	bg.Color = seFuncRowBG()
	cvs.DrawRect(x0, declY, cw, seRowH, bg)
	// 选中/编辑函数名 → 高亮（之前无任何反馈，点了像「不可编辑」）
	editingName := e.selSection == "func:"+itoaCE(si) && e.selRow == 0 && e.selCol == 0
	nameColor := seFuncRowText()
	if editingName {
		sel := paint.DefaultPaint()
		sel.Color = elPrimary()
		cvs.DrawRect(x0+1, declY+1, cw-2, seRowH-2, sel)
		nameColor = types.ColorFromRGB(255, 255, 255)
	}
	// 单元格
	e.font, _, _ = canvas.FontWithStyle("monospace", 13, canvas.FontRegular)
	padL := 8.0
	tx := x0 + padL
	canvas.DrawTextAligned(cvs, namePart, types.Rect{X: tx, Y: declY, Width: cw - padL, Height: seRowH}, e.font, nameColor, canvas.HAlignLeft, canvas.VAlignMiddle)
	// 命中区（用于双击编辑函数名等）
	e.cells = append(e.cells, seCellHit{rect: types.Rect{X: x0, Y: declY, Width: cw, Height: seRowH}, section: "func:" + itoaCE(si), row: 0, col: 0})
	// 折叠三角
	e.foldHits = append(e.foldHits, seFoldHit{rect: types.Rect{X: x0 + cw - 18, Y: declY + 3, Width: 16, Height: seRowH - 6}, sub: si})
	// 分隔线
	ln := paint.DefaultStrokePaint()
	ln.Color = seLineColor()
	cvs.DrawLine(x0, declY+seRowH, x0+cw, declY+seRowH, ln)
	y = declY + seRowH

	if collapsed {
		// ... 折叠虚线
		e.font, _, _ = canvas.FontWithStyle("monospace", 13, canvas.FontRegular)
		canvas.DrawTextAligned(cvs, "// ...", types.Rect{X: x0 + 8, Y: y, Width: 60, Height: seRowH}, e.font, types.ColorFromRGB(140, 140, 140), canvas.HAlignLeft, canvas.VAlignMiddle)
		y += seRowH
		return y
	}

	// 参数表
	if e.curSchema().Params != nil && len(e.curSchema().Params) > 0 {
		// 参数用 params schema 显示
		parmCols := e.curSchema().Params
		if len(sub.Params) > 0 {
			y = e.paintVarTable(cvs, x, y, innerW, gw, "params:"+itoaCE(si), parmCols, sub.Params, gl, true, false)
		}
	}

	// 返回值表
	if e.curSchema().Returns != nil && len(e.curSchema().Returns) > 0 && len(sub.Returns) > 0 {
		// 返回值用 returns schema
		y = e.paintVarTable(cvs, x, y, innerW, gw, "returns:"+itoaCE(si), e.curSchema().Returns, sub.Returns, gl, true, false)
	}

	return y
}

// ── Schema ──

// constCols 常量表的列定义。
var constCols = []SECol{
	{Title: "常量", Field: SEFieldName, Weight: 0.24},
	{Title: "类型", Field: SEFieldType, Weight: 0.18},
	{Title: "常量值", Field: SEFieldArray, Weight: 0.33}, // 值单独一列（存 Array 字段）
	{Title: "注释", Field: SEFieldNote, Weight: 0.25},   // 注释单独一列
}

// typeDefCols 类型定义表的列定义。
var typeDefCols = []SECol{
	{Title: "类型名", Field: SEFieldName, Weight: 0.22},
	{Title: "种类", Field: SEFieldType, Weight: 0.13},
	{Title: "成员/底层类型", Field: SEFieldArray, Weight: 0.45},
	{Title: "备注", Field: SEFieldNote, Weight: 0.20},
}

// typeFieldCols 类型展开后的字段/方法表列定义。
// 列边界对齐类型头(typeDefCols)：字段名↔类型名(0.22)、类型/签名跨 种类+成员(0.58)、注释↔备注(0.20)。
var typeFieldCols = []SECol{
	{Title: "字段/方法", Field: SEFieldName, Weight: 0.22},
	{Title: "类型/签名", Field: SEFieldType, Weight: 0.58},
	{Title: "备注", Field: SEFieldNote, Weight: 0.20},
}

// sectionTitle 取区段的标题文字。
func sectionTitle(section string, cols []SECol) string {
	switch {
	case section == "imports":
		return "导入"
	case section == "globals":
		if len(cols) > 0 {
			return cols[0].Title
		}
		return "程序集变量"
	case section == "consts":
		return "常量"
	case section == "types" || strings.HasPrefix(section, "type:"):
		return "类型定义"
	case strings.HasPrefix(section, "typefields:"):
		if len(cols) > 0 {
			return cols[0].Title // 「字段/方法」
		}
		return "字段"
	case strings.HasPrefix(section, "params:"):
		if len(cols) > 0 {
			return cols[0].Title
		}
		return "参数"
	case strings.HasPrefix(section, "returns:"):
		if len(cols) > 0 {
			return cols[0].Title
		}
		return "返回值"
	case strings.HasPrefix(section, "locals:"):
		if len(cols) > 0 {
			return cols[0].Title
		}
		return "局部变量"
	default:
		return ""
	}
}
