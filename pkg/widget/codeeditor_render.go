package widget

import (
	"strings"
	"time"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// CodeEditor 的渲染（行号栏 + 语法高亮 + 当前行 + 选区 + 光标 + 滚动条）与事件处理。

// ceTokenColor token 种类 → 颜色，取自 Theme.CodeEditor（SetTheme 换肤即生效）。
func ceTokenColor(k ceTokKind) types.Color {
	t := CurrentTheme().CodeEditor
	switch k {
	case tkKeyword:
		return t.Keyword
	case tkType:
		return t.Type
	case tkString:
		return t.String
	case tkComment:
		return t.Comment
	case tkNumber:
		return t.Number
	case tkFunc:
		return t.Func
	default:
		return t.Text
	}
}

// ── 坐标换算 ──

func (e *CodeEditorElement) colToX(line, col int) float64 {
	r := e.lineRunes(line)
	if col > len(r) {
		col = len(r)
	}
	if col <= 0 {
		return 0
	}
	return e.measure(string(r[:col]))
}

// xToCol 把行内局部 x（已扣除滚动/起点）换算为列。
func (e *CodeEditorElement) xToCol(line int, lx float64) int {
	r := e.lineRunes(line)
	col := 0
	for col < len(r) {
		w := e.measure(string(r[:col+1]))
		if w > lx {
			// 取更近的一侧
			prev := e.measure(string(r[:col]))
			if lx-prev > w-lx {
				col++
			}
			return col
		}
		col++
	}
	return len(r)
}

// lineTopY 返回实际行号在屏幕上的顶部 Y（经折叠可见索引换算）。
func (e *CodeEditorElement) lineTopY(line int, top float64) float64 {
	return top + float64(e.visIndexOf(line))*ceLineH
}

// isStringOrComment 判断 (行,列) 处的字符是否在字符串/注释 token 内（括号匹配时跳过）。
func (e *CodeEditorElement) isStringOrComment(li, ci int) bool {
	if li >= len(e.hl) {
		return false
	}
	for _, tk := range e.hl[li] {
		if ci >= tk.start && ci < tk.end {
			return tk.kind == tkString || tk.kind == tkComment
		}
	}
	return false
}

// enclosingBracket 返回包裹光标的最内层「跨行」括号对 {}/()/[] 的开/闭位置（按行包含光标行）；无则 ok=false。
// 结果只随「内容(bracketRev) + 光标行」变化，故缓存命中时直接返回，免每帧全文扫描。
func (e *CodeEditorElement) enclosingBracket() (ol, oc, cl, cc int, ok bool) {
	if e.brkCacheValid && e.brkCacheRev == e.bracketRev && e.brkCacheLine == e.cursor.line {
		return e.brkOL, e.brkOC, e.brkCL, e.brkCC, e.brkOK
	}
	defer func() { // 计算后写回缓存
		e.brkCacheValid, e.brkCacheRev, e.brkCacheLine = true, e.bracketRev, e.cursor.line
		e.brkOL, e.brkOC, e.brkCL, e.brkCC, e.brkOK = ol, oc, cl, cc, ok
	}()
	match := map[rune]rune{')': '(', ']': '[', '}': '{'}
	opens := map[rune]bool{'(': true, '[': true, '{': true}
	type bp struct {
		line, col int
		ch        rune
	}
	var stack []bp
	curL := e.cursor.line
	best := -1
	for li := 0; li < len(e.lines); li++ {
		runes := []rune(e.lines[li])
		for ci := 0; ci < len(runes); ci++ {
			r := runes[ci]
			isOpen := opens[r]
			open, isClose := match[r]
			if (isOpen || isClose) && e.isStringOrComment(li, ci) {
				continue // 忽略字符串/注释里的括号
			}
			if isOpen {
				stack = append(stack, bp{li, ci, r})
			} else if isClose {
				n := len(stack)
				if n == 0 {
					continue
				}
				o := stack[n-1]
				stack = stack[:n-1]
				if o.ch != open || li <= o.line { // 仅跨行括号对
					continue
				}
				if o.line <= curL && curL <= li { // 光标行落在括号对的行范围内
					if area := li - o.line; best < 0 || area < best { // 取最内层（行跨度最小）
						best = area
						ol, oc, cl, cc, ok = o.line, o.col, li, ci, true
					}
				}
			}
		}
	}
	return
}

func (e *CodeEditorElement) maxLineWidth() float64 {
	// 虚拟加载：用字符数×charW 估算最大行宽，避免对超长行逐行 Skia MeasureText。
	// 水平滚动条不需要像素级精确，估算值足够。
	maxChars := 0
	for _, ln := range e.lines {
		n := len(ln) // 对 ASCII 文本 bytes≈chars，CJK 会高估但安全（滚动条偏大不影响功能）
		if n > maxChars {
			maxChars = n
		}
	}
	return float64(maxChars) * e.measure("0")
}

// posFromXY 屏幕坐标 → (行,列)。换行时按视觉段定位：视觉行→段→段内列。
func (e *CodeEditorElement) posFromXY(sx, sy float64) cePos {
	pos := e.Offset()
	contentTop := pos.Y + 4
	vi := int((sy - contentTop + e.scrollY) / ceLineH) // 视觉行索引
	if vi < 0 {
		vi = 0
	}
	if len(e.wrapSegs) == 0 {
		return cePos{0, 0}
	}
	if vi >= len(e.wrapSegs) {
		vi = len(e.wrapSegs) - 1
	}
	s := e.wrapSegs[vi]
	editorTextX := pos.X + e.gutterW + ceTextPad
	lx := sx - (editorTextX - e.scrollX)
	if lx < 0 {
		lx = 0
	}
	return cePos{s.line, e.segColAtX(s, lx)}
}

// ── Paint ──

func (e *CodeEditorElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	w, h := e.size.Width, e.size.Height
	e.lastCanvas = cvs     // 缓存画布，供点击定位/IME 候选用与渲染一致的 Skia 测量
	e.drainLSPCompletion() // 消费异步到达的 LSP 补全结果

	if e.ed == nil || !e.ed.Embedded {
		// 独立模式：白底圆角 + 边框（聚焦主色）。嵌入模式(StructEditor)完全无边框，无缝融入整份文档。
		bg := paint.DefaultPaint()
		bg.Color = CurrentTheme().CodeEditor.Background
		cvs.DrawRoundedRect(pos.X, pos.Y, w, h, 6, bg)
		border := elBorder()
		if e.focused {
			border = elPrimary()
		}
		bp := paint.DefaultStrokePaint()
		bp.Color = border
		bp.StrokeWidth = 1
		cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, w-1, h-1, 6, bp)
	}

	contentTop := pos.Y + 4
	editorX := pos.X + e.gutterW
	editorTextX := editorX + ceTextPad

	// 滚动范围 + 滚动条占位。换行开启时高度按视觉段数算、横向不滚（maxLineW 视作 ≤ viewW）。
	miniW := 0.0
	if e.showMinimap && w > 260 { // 太窄不显示缩略图
		miniW = ceMiniW
	}
	// 先估一遍 editorViewW 以构建视觉段（v 条按内容是否溢出预留；换行下 h 条恒不占位）。
	estViewW := w - e.gutterW - ceTextPad - miniW
	if miniW == 0 {
		estViewW -= sbThick // 长文一般溢出，预留竖条宽，避免段宽来回抖动
	}
	e.ensureWrapSegs(estViewW)

	contentH := float64(len(e.wrapSegs)) * ceLineH
	maxLineW := e.maxLineWidth()
	vBar, hBar := 0.0, 0.0
	if miniW == 0 && contentH > h-8 { // 有缩略图时用其视口框代替竖滚动条
		vBar = sbThick
	}
	editorViewW := w - e.gutterW - ceTextPad - vBar - miniW
	if !e.wrap && maxLineW > editorViewW { // 换行时不出横条
		hBar = sbThick
	}
	viewH := h - 8 - hBar
	if miniW == 0 && vBar == 0 && contentH > viewH {
		vBar = sbThick
		editorViewW = w - e.gutterW - ceTextPad - vBar - miniW
	}
	// editorViewW 定下后再确保视觉段按它构建（宽度变才会真重建）。
	e.ensureWrapSegs(editorViewW)
	contentH = float64(len(e.wrapSegs)) * ceLineH
	maxScrollY := contentH - viewH
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	maxScrollX := maxLineW - editorViewW + 4
	if e.wrap || maxScrollX < 0 {
		maxScrollX = 0 // 换行：无横向滚动
	}

	// 光标跟随（按视觉行 + 段内 x）
	if e.focused && e.cursorMoved {
		curTop := float64(e.segRowOf(e.cursor.line, e.cursor.col)) * ceLineH
		if curTop-e.scrollY < 0 {
			e.scrollY = curTop
		}
		if curTop+ceLineH-e.scrollY > viewH {
			e.scrollY = curTop + ceLineH - viewH
		}
		if !e.wrap {
			curX := e.colToX(e.cursor.line, e.cursor.col)
			if curX-e.scrollX < 0 {
				e.scrollX = curX
			}
			if curX-e.scrollX > editorViewW-6 {
				e.scrollX = curX - editorViewW + 6
			}
		}
		e.cursorMoved = false
	}
	if e.wrap {
		e.scrollX = 0
	}
	e.scrollY = clamp(e.scrollY, 0, maxScrollY)
	e.scrollX = clamp(e.scrollX, 0, maxScrollX)

	top := contentTop - e.scrollY
	left := editorTextX - e.scrollX

	// 行号栏背景
	gb := paint.DefaultPaint()
	gb.Color = CurrentTheme().CodeEditor.GutterBg
	cvs.DrawRect(pos.X+1, pos.Y+1, e.gutterW, h-2, gb)

	// firstVis/lastVis 现为 wrapSegs（视觉行）下标范围。
	firstVis := int(e.scrollY/ceLineH) - 1
	if firstVis < 0 {
		firstVis = 0
	}
	lastVis := int((e.scrollY+viewH)/ceLineH) + 1
	if lastVis >= len(e.wrapSegs) {
		lastVis = len(e.wrapSegs) - 1
	}

	// 当前行高亮（编辑区 + 行号栏）：换行时高亮光标逻辑行覆盖的【所有可见】视觉行（限可视范围，免大文件每帧全扫）。
	if e.focused && !e.hasSel() {
		for i := firstVis; i <= lastVis; i++ {
			if i < 0 || i >= len(e.wrapSegs) || e.wrapSegs[i].line != e.cursor.line {
				continue
			}
			cy := top + float64(i)*ceLineH
			cl := paint.DefaultPaint()
			cl.Color = CurrentTheme().CodeEditor.CurrentLineBg
			cvs.DrawRect(editorX+1, cy, w-e.gutterW-2, ceLineH, cl)
			gl := paint.DefaultPaint()
			gl.Color = CurrentTheme().CodeEditor.GutterActiveBg
			cvs.DrawRect(pos.X+1, cy, e.gutterW, ceLineH, gl)
		}
	}

	// 编辑区裁剪：选区/文本/光标（右侧给缩略图/滚动条留位）
	cvs.Save()
	cvs.ClipRect(editorX+1, pos.Y+1, w-e.gutterW-2-vBar-miniW, h-2-hBar)

	// 选区高亮（主 + 多光标额外选区）
	if e.hasSel() {
		e.paintSelection(cvs, left, top)
	}
	for _, c := range e.extraCarets {
		if !cePosEq(c.cursor, c.anchor) {
			lo, hi := c.anchor, c.cursor
			if cePosLess(c.cursor, c.anchor) {
				lo, hi = c.cursor, c.anchor
			}
			e.paintSelRange(cvs, left, top, lo, hi)
		}
	}
	// 查找匹配高亮（所有匹配黄、当前橙）
	if e.findActive {
		e.paintMatchHighlights(cvs, left, top)
	}

	// 缩进连线（indent guides）：各级缩进画虚线；光标所在最内层「跨行括号对」画实线折线——
	// 折线从开括号位置横折引出、竖到闭括号位置（真正的括号范围，非缩进推断）。
	if e.ed.IndentGuides {
		dash := paint.DefaultStrokePaint()
		dash.Color = types.ColorFromRGB(0xDD, 0xE2, 0xE8)
		dash.StrokeWidth = 1
		for vi := firstVis; vi <= lastVis; vi++ {
			if vi < 0 || vi >= len(e.wrapSegs) {
				continue
			}
			s := e.wrapSegs[vi]
			if s.start != 0 { // 缩进连线只画在逻辑行首段（缩进在那）
				continue
			}
			i := s.line
			ly := top + float64(vi)*ceLineH
			for k := 0; (k+1)*ceIndentSize <= leadingSpaces(e.lines[i]); k++ {
				gx := left + e.colToX(i, k*ceIndentSize) + 0.5
				for yy := ly + 1; yy < ly+ceLineH-1; yy += 4 {
					ye := yy + 2
					if ye > ly+ceLineH-1 {
						ye = ly + ceLineH - 1
					}
					cvs.DrawLine(gx, yy, gx, ye, dash)
				}
			}
		}
		if e.focused { // 活动折线：包裹光标的最内层跨行括号对，从括号位置引出（走行间空隙，不穿过文字）
			if ol, oc, cl, cc, ok := e.enclosingBracket(); ok {
				solid := paint.DefaultStrokePaint()
				solid.Color = types.ColorFromRGB(0x6F, 0x8F, 0xD0)
				solid.StrokeWidth = 1.3
				gx := e.posX(ol, leadingSpaces(e.lines[ol]), left) + 0.5 // 竖线在开括号行缩进列
				ocx := e.posX(ol, oc, left) + 0.5                        // 开括号 { 的 x（行末，文字右侧）
				ccx := e.posX(cl, cc, left) + 0.5                        // 闭括号 } 的 x
				oyTop := e.posTopY(ol, oc, top)
				oyBot := oyTop + ceLineH
				cyTop := e.posTopY(cl, cc, top)
				cvs.DrawLine(ocx, oyTop+ceLineH/2, ocx, oyBot, solid) // 开括号 → 本行底（沿括号右侧下行）
				cvs.DrawLine(ocx, oyBot, gx, oyBot, solid)            // 行底横折到竖线列（在文字下方空隙）
				cvs.DrawLine(gx, oyBot, gx, cyTop, solid)             // 块体竖线
				cvs.DrawLine(gx, cyTop, ccx, cyTop, solid)            // 行顶横折到闭括号列（在文字上方空隙）
				cvs.DrawLine(ccx, cyTop, ccx, cyTop+ceLineH/2, solid) // → 闭括号
			}
		}
	}

	// 文本（遍历视觉段，逐段按 token 着色；token 按列裁剪到段区间并平移到段起点）
	// 虚拟加载：对所有 DrawText 调用，按可见右边界 visRightX 裁剪字符串，
	// 只对视口内可见的字符进行实际的 GPU 绘制。超长行（5万字符）仅首次可见的 ~44 字符被 DrawText，
	// 其余在水平滚动时才按需绘制，首帧 textDrawing 从 42 秒降至毫秒级。
	visRightX := editorTextX + editorViewW
	charW := e.measure("0") // 等宽单字符宽，用于可见性快速估算
	for vi := firstVis; vi <= lastVis; vi++ {
		if vi < 0 || vi >= len(e.wrapSegs) {
			continue
		}
		sg := e.wrapSegs[vi]
		i := sg.line // 实际行号
		ly := top + float64(vi)*ceLineH
		baseY := canvas.BaselineFor(ly, ceLineH, e.font.Size, canvas.VAlignMiddle)
		runes := []rune(e.lines[i])
		toks := e.hl[i]
		drawn := sg.start
		currentX := left // 增量 x 代替 segX（避免对大文件每 token 调 e.measure 从行首重测，O(n²)）
		// drawClipped 在 currentX 处绘制 runes[lo:hi]，超出 visRightX 则裁剪并终止本段后续绘制。
		// 【普通虚加载】先检查可见性再测量：对已过可见右界的 token 直接跳过，不调用 Skia measure，
		// 避免每帧对不可见 token 做 CGO 测量（单文字虚加载每 token 先 measure 再检查可见性，帧率反降）。
		drawClipped := func(lo, hi int, color types.Color) bool {
			if currentX >= visRightX {
				return false // 起点已在可见区右侧，整段跳过（不调用 measure）
			}
			w := e.measure(string(runes[lo:hi]))
			if currentX+w > visRightX { // 部分超出 → 裁剪字符串到可见宽度
				availW := visRightX - currentX
				if availW > 0 {
					clipChars := int(availW / charW) // 等宽估算可见字符数
					if clipChars > hi-lo {
						clipChars = hi - lo
					}
					if clipChars > 0 {
						cvs.DrawText(string(runes[lo:lo+clipChars]), currentX, baseY, e.font, mkPaint(color))
					}
				}
				return false // 后续 token 都在可见区右侧
			}
			cvs.DrawText(string(runes[lo:hi]), currentX, baseY, e.font, mkPaint(color))
			currentX += w
			return true
		}
		for _, tk := range toks {
			s, en := tk.start, tk.end // token 裁剪到本段 [sg.start,sg.end)
			if en <= sg.start || s >= sg.end {
				continue
			}
			if s < sg.start {
				s = sg.start
			}
			if en > sg.end {
				en = sg.end
			}
			if s > drawn { // token 之前的未着色段
				if !drawClipped(drawn, s, ceTokenColor(tkText)) {
					drawn = sg.end
					break
				}
			}
			if !drawClipped(s, en, ceTokenColor(tk.kind)) {
				drawn = sg.end
				break
			}
			drawn = en
		}
		if drawn < sg.end {
			drawClipped(drawn, sg.end, ceTokenColor(tkText))
		}
		// 折叠提示：该行被折叠 → 行尾画 ⋯ 块（画在末段尾部）
		isLastSeg := vi+1 >= len(e.wrapSegs) || e.wrapSegs[vi+1].line != i
		if isLastSeg && e.folded[i] && e.isFoldStart(i) {
			lineEndX := currentX
			fp := paint.DefaultPaint()
			fp.Color = types.ColorFromRGB(0xE8, 0xEC, 0xF0)
			cvs.DrawRoundedRect(lineEndX+6, ly+4, 22, ceLineH-8, 3, fp)
			canvas.DrawTextAligned(cvs, "...", types.Rect{X: lineEndX + 6, Y: ly - 2, Width: 22, Height: ceLineH}, e.font, types.ColorFromRGB(0x6A, 0x73, 0x7D), canvas.HAlignCenter, canvas.VAlignMiddle)
		}
	}

	// 诊断波浪线（LSP 错误/警告）
	e.paintDiagnostics(cvs, left, top)

	// 括号匹配高亮（光标紧邻括号时，两端淡蓝底）
	if e.focused {
		if a, b, ok := e.findMatchingBracket(); ok {
			e.paintBracketBox(cvs, a, left, top)
			e.paintBracketBox(cvs, b, left, top)
		}
	}

	// IME 组合预览 + 光标
	baseCx := e.posX(e.cursor.line, e.cursor.col, left)
	cyTop := e.posTopY(e.cursor.line, e.cursor.col, top)
	cBase := canvas.BaselineFor(cyTop, ceLineH, e.font.Size, canvas.VAlignMiddle)
	caretX := baseCx
	if e.composition != "" && !e.findActive { // 组合预览：白底盖后方文字 + 文本 + 蓝下划线（查找激活时改在查找框显示）
		compW := canvas.MeasureTextGlobal(e.composition, e.font).Width
		wbg := paint.DefaultPaint()
		wbg.Color = CurrentTheme().CodeEditor.Background
		cvs.DrawRect(baseCx, cyTop+1, compW, ceLineH-2, wbg)
		cvs.DrawText(e.composition, baseCx, cBase, e.font, mkPaint(types.ColorFromRGB(0x24, 0x29, 0x2E)))
		ul := paint.DefaultStrokePaint()
		ul.Color = types.ColorFromRGB(0x40, 0x9E, 0xFF)
		ul.StrokeWidth = 1
		cvs.DrawLine(baseCx, cBase+2, baseCx+compW, cBase+2, ul)
		caretX = baseCx + compW
	}
	if e.focused && (e.isCursorVisible() || e.composition != "") {
		cp := paint.DefaultStrokePaint()
		cp.Color = CurrentTheme().CodeEditor.Text
		cp.StrokeWidth = 2.0
		cvs.DrawLine(caretX, cyTop+1, caretX, cyTop+ceLineH-1, cp)
	}
	// 缓存光标客户区位置（供 IME 候选窗口定位）。Y 用光标【顶部】——与 InputElement
	// 一致：多数输入法把候选放到锚点下方，用顶部锚点候选恰好紧贴文字下方；用底部会偏下一行。
	e.cursorClientX = caretX
	e.cursorClientY = cyTop
	if e.ed.CursorRef != nil { // 同步光标/滚动，供切换视图后恢复
		e.ed.CursorRef.Line, e.ed.CursorRef.Col = e.cursor.line, e.cursor.col
		e.ed.CursorRef.ScrollX, e.ed.CursorRef.ScrollY = e.scrollX, e.scrollY
	}

	// 额外光标（多光标）
	if e.focused && e.isCursorVisible() {
		for _, c := range e.extraCarets {
			ecx := e.posX(c.cursor.line, c.cursor.col, left)
			ecTopY := e.posTopY(c.cursor.line, c.cursor.col, top)
			cp := paint.DefaultStrokePaint()
			cp.Color = CurrentTheme().CodeEditor.Text
			cp.StrokeWidth = 2.0
			cvs.DrawLine(ecx, ecTopY+1, ecx, ecTopY+ceLineH-1, cp)
		}
	}
	cvs.Restore()

	// 行号 + 折叠箭头（裁剪到行号栏）
	cvs.Save()
	cvs.ClipRect(pos.X+1, pos.Y+1, e.gutterW, h-2-hBar)
	for vi := firstVis; vi <= lastVis; vi++ {
		if vi < 0 || vi >= len(e.wrapSegs) {
			continue
		}
		sg := e.wrapSegs[vi]
		if sg.start != 0 { // 续接段不画行号/折叠箭头（行号只在逻辑行首段显示）
			continue
		}
		i := sg.line
		ly := top + float64(vi)*ceLineH
		numColor := CurrentTheme().CodeEditor.GutterText
		if i == e.cursor.line && e.focused {
			numColor = CurrentTheme().CodeEditor.Text // 当前行号加深
		}
		canvas.DrawTextAligned(cvs, itoaCE(i+1+e.ed.LineNumberOffset),
			types.Rect{X: pos.X, Y: ly, Width: e.gutterW - ceGutterPad - ceFoldW, Height: ceLineH},
			e.font, numColor, canvas.HAlignRight, canvas.VAlignMiddle)
		if e.isFoldStart(i) { // 折叠箭头
			e.paintFoldArrow(cvs, pos.X+e.gutterW-ceFoldW, ly, e.folded[i])
		}
	}
	// 行号栏诊断标记（红/橙圆点）
	e.paintGutterDiagnostics(cvs, top)
	cvs.Restore()

	// 行号栏右分隔线
	sep := paint.DefaultStrokePaint()
	sep.Color = elBorder()
	cvs.DrawLine(editorX, pos.Y+1, editorX, pos.Y+h-1, sep)

	e.paintScrollbars(cvs, pos, w, h, contentH, maxLineW, viewH, editorViewW, maxScrollY, maxScrollX, hBar, vBar, editorTextX)

	// 缩略图（右侧）
	e.miniRect = types.Rect{}
	if miniW > 0 {
		e.paintMinimap(cvs, pos.X+w-miniW-1, pos.Y+1, h-2, viewH, e.scrollY, maxScrollY)
	}

	// 补全弹窗（编辑器内容之上，可超出编辑区）
	if e.completing {
		e.paintCompletion(cvs, left, top)
	}
	e.paintHover(cvs, left, top) // 悬停信息浮层（showHover 触发）
	e.paintDiagHover(cvs)        // 诊断悬停提示浮层

	// 查找栏（最上层，右上角）
	if e.findActive {
		e.paintFindBar(cvs, pos, w)
	}
}

// paintBracketBox 在括号字符位置画一个蓝色边框（匹配高亮）。
func (e *CodeEditorElement) paintBracketBox(cvs canvas.Canvas, p cePos, left, top float64) {
	x0 := e.posX(p.line, p.col, left)
	x1 := e.posX(p.line, p.col+1, left)
	ly := e.posTopY(p.line, p.col, top)
	bp := paint.DefaultStrokePaint()
	bp.Color = types.ColorFromRGB(0x40, 0x9E, 0xFF)
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(x0-1, ly+2, x1-x0+2, ceLineH-4, 2, bp)
}

// paintSelection 逐行画主选区高亮。
func (e *CodeEditorElement) paintSelection(cvs canvas.Canvas, left, top float64) {
	lo, hi := e.sortedSel()
	e.paintSelRange(cvs, left, top, lo, hi)
}

// paintSelRange 逐【视觉行】画 [lo,hi) 选区高亮（主选区 + 多光标额外选区共用）。
// 换行时一个逻辑行的列区间会被 forSegSpans 拆到其各视觉段上。
func (e *CodeEditorElement) paintSelRange(cvs canvas.Canvas, left, top float64, lo, hi cePos) {
	sel := paint.DefaultPaint()
	sel.Color = CurrentTheme().CodeEditor.Selection
	for ln := lo.line; ln <= hi.line; ln++ {
		r := e.lineRunes(ln)
		startCol := 0
		endCol := len(r)
		if ln == lo.line {
			startCol = lo.col
		}
		if ln == hi.line {
			endCol = hi.col
		}
		crossNL := ln < hi.line // 跨行：行尾补一小段表示选中换行
		e.forSegSpans(ln, startCol, endCol, left, top, crossNL, func(rowTopY, x0, x1 float64) {
			if x1 > x0 {
				cvs.DrawRect(x0, rowTopY, x1-x0, ceLineH, sel)
			}
		})
	}
}

func (e *CodeEditorElement) paintScrollbars(cvs canvas.Canvas, pos types.Point, w, h, contentH, maxLineW, viewH, editorViewW, maxScrollY, maxScrollX, hBar, vBar, editorTextX float64) {
	e.vbarThumb, e.vbarFactor = types.Rect{}, 0
	if maxScrollY > 0 && vBar > 0 { // 有缩略图时 vBar=0，用缩略图视口框代替竖滚动条
		barW := 6.0
		bx := pos.X + w - barW - 3
		trackH := h - 6 - hBar
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
	e.hbarThumb, e.hbarFactor = types.Rect{}, 0
	if maxScrollX > 0 {
		barH := 6.0
		by := pos.Y + h - barH - 3
		trackW := editorViewW
		thumbW := editorViewW / maxLineW * trackW
		if thumbW < 20 {
			thumbW = 20
		}
		thumbX := editorTextX + (e.scrollX/maxScrollX)*(trackW-thumbW)
		e.hbarThumb = types.Rect{X: thumbX, Y: by, Width: thumbW, Height: barH}
		if trackW-thumbW > 0 {
			e.hbarFactor = maxScrollX / (trackW - thumbW)
		}
		th := paint.DefaultPaint()
		th.Color = types.ColorFromRGB(193, 193, 193)
		cvs.DrawRoundedRect(thumbX, by, thumbW, barH, 3, th)
	}
}

// ── 光标移动 ──

func (e *CodeEditorElement) moveCursor(dLine, dCol int, extend bool) {
	c := e.cursor
	if dCol != 0 {
		c.col += dCol
		if c.col < 0 { // 跨到上一【可见】行末
			if vi := e.visIndexOf(c.line) - 1; vi >= 0 {
				c.line = e.visRows[vi]
				c.col = len(e.lineRunes(c.line))
			} else {
				c.col = 0
			}
		} else if c.col > len(e.lineRunes(c.line)) { // 跨到下一【可见】行首
			if vi := e.visIndexOf(c.line) + 1; vi < len(e.visRows) {
				c.line = e.visRows[vi]
				c.col = 0
			} else {
				c.col = len(e.lineRunes(c.line))
			}
		}
	}
	if dLine != 0 {
		if e.wrap { // 换行：按【视觉段】上/下移，保持目标 x 像素尽量不变
			row := e.segRowOf(c.line, c.col)
			cur := e.segAt(row)
			r := e.lineRunes(c.line)
			cc := c.col
			if cc > len(r) {
				cc = len(r)
			}
			if cc < cur.start {
				cc = cur.start
			}
			wantX := e.measure(string(r[cur.start:cc])) // 段内目标横向像素
			nrow := row + dLine
			if nrow < 0 {
				nrow = 0
			}
			if nrow >= len(e.wrapSegs) {
				nrow = len(e.wrapSegs) - 1
			}
			tgt := e.segAt(nrow)
			c.line = tgt.line
			c.col = e.segColAtX(tgt, wantX) // 目标段内找最近列
		} else {
			vi := e.visIndexOf(c.line) + dLine // 上/下移按可见行
			if vi < 0 {
				vi = 0
			}
			if vi >= len(e.visRows) {
				vi = len(e.visRows) - 1
			}
			if vi >= 0 && vi < len(e.visRows) {
				c.line = e.visRows[vi]
			}
			c = e.clampPos(c) // 保持列，clamp 到目标行长度
		}
	}
	e.cursor = c
	if !extend {
		e.anchor = c
	}
	e.breakUndo()
	e.cursorMoved = true
	e.resetBlink()
	if e.ed.OnCursorMove != nil {
		e.ed.OnCursorMove(e.cursor.line, e.cursor.col)
	}
	repaint()
}

// indentSelection 给选区涉及的每一行加/减一级缩进（Tab/Shift+Tab）。
func (e *CodeEditorElement) indentSelection(out bool) {
	lo, hi := e.sortedSel()
	for ln := lo.line; ln <= hi.line; ln++ {
		if out {
			s := e.lines[ln]
			n := 0
			for n < 4 && n < len(s) && s[n] == ' ' {
				n++
			}
			e.lines[ln] = s[n:]
		} else {
			e.lines[ln] = "    " + e.lines[ln]
		}
	}
	e.rehighlight()
	if e.ed.OnChange != nil {
		e.ed.OnChange(e.text())
	}
	repaint()
}

// ── 事件 ──

func (e *CodeEditorElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeIMEComposition:
		ce, ok := ev.(*event.IMECompositionEvent)
		if !ok || !e.focused {
			break
		}
		e.composition = ce.Composition // 组合预览（提交文本走 KeyChar→handleCharInput）
		e.compositionCursor = ce.CursorPos
		repaint()
		return true

	case event.TypeContextMenu:
		if SuppressEditorContextMenu { // 宿主（如 companion）自带编辑器右键菜单 → 放行冒泡给上层 ContextArea
			return false
		}
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		ShowContextMenu(me.X, me.Y, e.contextItems())
		return true

	case event.TypeMouseWheel:
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		if me.DeltaX != 0 || me.Mods&(event.ModShift|event.ModCtrl) != 0 {
			d := me.DeltaX
			if d == 0 {
				d = me.DeltaY
			}
			e.scrollX -= d * 40
			if e.scrollX < 0 {
				e.scrollX = 0
			}
		} else {
			e.scrollY -= me.DeltaY * 40
			if e.scrollY < 0 {
				e.scrollY = 0
			}
		}
		ev.StopPropagation()
		repaint()
		return true

	case event.TypeMouseEnter:
		e.hovered = true
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.diagHoverMsg = "" // 离开编辑器区域 → 关闭诊断悬停
		e.hoverText = ""    // 离开编辑器区域 → 关闭 LSP 悬停
		if e.hoverTimerCancel != nil {
			e.hoverTimerCancel()
			e.hoverTimerCancel = nil
		}
		return true

	case event.TypeMouseDown:
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		if me.Button == event.ButtonRight {
			e.Focus() // 右键也聚焦本编辑器：使宿主右键菜单的编辑命令(撤销/格式化/自动换行…)经 focusedCodeEditor 派发到它；随后放行冒泡给宿主菜单
			return false
		}
		if e.findActive && e.findBarHit(me.X, me.Y) { // 查找栏按钮/输入框
			return true
		}
		if line, ok := e.foldArrowAt(me.X, me.Y); ok { // 折叠箭头
			e.focused = true
			e.toggleFold(line)
			return true
		}
		if e.minimapHit(me.X, me.Y) { // 缩略图点击/拖动跳转
			e.miniDragging = true
			e.minimapJump(me.Y)
			if RequestPointerCapture != nil {
				RequestPointerCapture(e)
			}
			ev.StopPropagation()
			return true
		}
		if thumbHit(e.vbarThumb, me.X, me.Y) {
			e.draggingVBar = true
			e.dragStartMouse, e.dragStartScroll = me.Y, e.scrollY
			if RequestPointerCapture != nil {
				RequestPointerCapture(e)
			}
			ev.StopPropagation()
			return true
		}
		if thumbHit(e.hbarThumb, me.X, me.Y) {
			e.draggingHBar = true
			e.dragStartMouse, e.dragStartScroll = me.X, e.scrollX
			if RequestPointerCapture != nil {
				RequestPointerCapture(e)
			}
			ev.StopPropagation()
			return true
		}
		e.focused = true
		p := e.posFromXY(me.X, me.Y)
		if me.Mods&event.ModCtrl != 0 { // Ctrl+点击：在点击处加光标
			e.addCaretAt(p)
			return true
		}
		e.clearExtraCarets() // 普通点击：取消多光标
		e.completing = false // 取消补全
		e.cursor = p
		e.anchor = p
		e.selecting = true
		e.breakUndo()
		e.cursorMoved = true
		e.resetBlink()
		if e.ed.OnCursorMove != nil {
			e.ed.OnCursorMove(e.cursor.line, e.cursor.col)
		}
		repaint()
		return true

	case event.TypeMouseMove:
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		e.mouseX, e.mouseY = me.X, me.Y
		if e.draggingVBar {
			e.scrollY = clamp(e.dragStartScroll+(me.Y-e.dragStartMouse)*e.vbarFactor, 0, 1e9)
			ev.StopPropagation()
			repaint()
			return true
		}
		if e.draggingHBar {
			e.scrollX = clamp(e.dragStartScroll+(me.X-e.dragStartMouse)*e.hbarFactor, 0, 1e9)
			ev.StopPropagation()
			repaint()
			return true
		}
		if e.miniDragging {
			e.minimapJump(me.Y)
			ev.StopPropagation()
			return true
		}
		if e.selecting {
			e.cursor = e.posFromXY(me.X, me.Y)
			e.cursorMoved = true
			if e.ed.OnCursorMove != nil {
				e.ed.OnCursorMove(e.cursor.line, e.cursor.col)
			}
			repaint()
			return true
		}
		// 诊断悬停检测 + LSP 悬停检测：鼠标不在拖拽/选择时
		if e.ed != nil {
			p := e.posFromXY(me.X, me.Y)
			// 诊断悬停（波浪线）
			msg := e.diagnosticAtPos(p.line, p.col)
			if msg != "" {
				e.diagHoverMsg = cleanText(msg)
				e.diagHoverLine = p.line
				e.diagHoverCol = p.col
			} else {
				e.diagHoverMsg = ""
			}
			// LSP 语义悬停：鼠标 200ms 不动 → 触发请求
			if e.lspReady && !e.selecting && !e.draggingVBar && !e.draggingHBar && !e.miniDragging {
				// 判断：位置变了（字符级）或鼠标像素移动超过 4px？
				// 任一条件为真 → 视为「移开」，取消旧悬停/旧定时器，重新计时。
				// 像素阈值可避免同字符内微动反复关闭/重触发。
				dx := me.X - e.hoverMouseX
				dy := me.Y - e.hoverMouseY
				moved := dx*dx+dy*dy > 16 // 4px（平方比较避免 sqrt）
				posChanged := !cePosEq(p, e.mouseHoverPos)
				if moved || posChanged {
					e.mouseHoverPos = p
					// 已有悬停且鼠标移开 → 关闭
					if e.hoverText != "" && e.hoverByMouse {
						e.hoverText = ""
					}
					// 取消旧定时器
					if e.hoverTimerCancel != nil {
						e.hoverTimerCancel()
					}
					// 启动新 200ms 延迟
					ctx := make(chan struct{}, 1)
					e.hoverTimerCancel = func() {
						select {
						case ctx <- struct{}{}:
						default:
						}
					}
					go func(pos cePos, mx, my float64, cancel <-chan struct{}) {
						select {
						case <-time.After(200 * time.Millisecond):
							e.lspMu.Lock()
							if cePosEq(e.mouseHoverPos, pos) && e.lspReady {
								e.hoverByMouse = true
								e.hoverAnchor = pos
								e.hoverMouseX = mx
								e.hoverMouseY = my
							}
							e.lspMu.Unlock()
							if e.lspReady {
								e.requestHoverAt(pos)
							}
						case <-cancel:
						}
					}(p, me.X, me.Y, ctx)
				}
			}
		}
		return false

	case event.TypeMouseUp:
		e.selecting = false
		e.draggingVBar = false
		e.draggingHBar = false
		e.miniDragging = false
		return true

	case event.TypeMouseDoubleClick:
		// 双击选中光标所在单词
		e.selectWord()
		return true

	case event.TypeKeyChar:
		keyEv, ok := ev.(*event.KeyEvent)
		if !ok {
			break
		}
		if e.findActive { // 查找栏激活：输入进查找/替换框
			if keyEv.Char >= 32 {
				e.handleFindChar(keyEv.Char)
			}
			return true
		}
		if !e.focused {
			break
		}
		if keyEv.Char >= 32 {
			e.handleCharInput(keyEv.Char) // 含自动配对/包裹/over-type/撤销记录
			if !e.hasMultiCaret() {
				e.updateCompletion() // 输入后刷新补全候选
			}
			return true
		}

	case event.TypeKeyDown:
		keyEv, ok := ev.(*event.KeyEvent)
		if !ok {
			break
		}
		if e.findActive {
			return e.handleFindKey(keyEv)
		}
		if !e.focused {
			break
		}
		return e.handleKeyDown(keyEv)
	}
	return false
}

func (e *CodeEditorElement) handleKeyDown(k *event.KeyEvent) bool {
	if k.Mods&event.ModCtrl != 0 {
		switch k.Key {
		case "A":
			e.anchor = cePos{0, 0}
			last := len(e.lines) - 1
			e.cursor = cePos{last, len(e.lineRunes(last))}
			repaint()
		case "C":
			if e.hasSel() && ClipboardWrite != nil {
				ClipboardWrite(e.selText())
			}
		case "X":
			if e.hasSel() && ClipboardWrite != nil {
				ClipboardWrite(e.selText())
				e.recordUndo("cut")
				e.deleteSel()
				e.afterEdit()
			}
		case "V":
			if ClipboardRead != nil {
				e.recordUndo("paste")
				e.insertStr(ClipboardRead())
			}
		case "Z":
			if k.Mods&event.ModShift != 0 {
				e.redo()
			} else {
				e.undo()
			}
		case "Y":
			e.redo()
		case "F":
			e.openFind(false)
		case "H":
			e.openFind(true)
		case "Space", " ": // Ctrl+Space 手动触发补全
			e.triggerCompletion()
		}
		return true
	}
	if k.Mods&event.ModAlt != 0 && k.Mods&event.ModShift != 0 && k.Key == "F" {
		e.formatDocument()
		return true
	}
	if e.completing { // 补全弹窗激活：拦截导航/接受/取消键
		switch k.Key {
		case "ArrowDown":
			e.compMove(1)
			return true
		case "ArrowUp":
			e.compMove(-1)
			return true
		case "Enter", "Tab":
			e.acceptCompletion()
			return true
		case "Escape":
			e.cancelCompletion()
			repaint()
			return true
		case "ArrowLeft", "ArrowRight", "Home", "End":
			e.cancelCompletion() // 光标离开词→关闭，继续走下面正常移动
		}
	}
	extend := k.Mods&event.ModShift != 0
	if e.hasMultiCaret() { // 多光标编辑/移动
		switch k.Key {
		case "Escape":
			e.clearExtraCarets()
			e.hoverText = "" // Esc 也关闭 LSP 悬停
			return true
		case "Backspace":
			e.editEachCaret("delete", e.backspace)
			return true
		case "Delete":
			e.editEachCaret("delete", e.deleteForward)
			return true
		case "Enter":
			e.editEachCaret("newline", e.insertNewline)
			return true
		case "Tab":
			e.editEachCaret("indent", func() { e.insertStr("    ") })
			return true
		case "ArrowLeft":
			e.forEachCaret(func() { e.moveCursor(0, -1, extend) })
			return true
		case "ArrowRight":
			e.forEachCaret(func() { e.moveCursor(0, 1, extend) })
			return true
		case "ArrowUp":
			e.forEachCaret(func() { e.moveCursor(-1, 0, extend) })
			return true
		case "ArrowDown":
			e.forEachCaret(func() { e.moveCursor(1, 0, extend) })
			return true
		}
	}
	switch k.Key {
	case "Escape":
		e.clearExtraCarets()
		e.hoverText = "" // Esc 也关闭 LSP 悬停
	case "Backspace":
		e.recordUndo("delete")
		e.backspace()
		if e.completing {
			e.updateCompletion() // 退格后重新过滤
		}
	case "Delete":
		e.recordUndo("delete")
		e.deleteForward()
	case "Enter":
		e.recordUndo("newline")
		e.insertNewline()
	case "Tab":
		e.recordUndo("indent")
		if e.hasSel() && e.anchorDiffLine() {
			e.indentSelection(false)
		} else {
			e.insertStr("    ")
		}
	case "ArrowLeft":
		e.moveCursor(0, -1, extend)
	case "ArrowRight":
		e.moveCursor(0, 1, extend)
	case "ArrowUp":
		e.moveCursor(-1, 0, extend)
	case "ArrowDown":
		e.moveCursor(1, 0, extend)
	case "Home":
		// 换行时 Home/End 落到【当前视觉段】首/尾（贴合所见行，非整逻辑行）；不换行同旧=逻辑行首/尾。
		if e.wrap {
			e.cursor.col = e.segAt(e.segRowOf(e.cursor.line, e.cursor.col)).start
		} else {
			e.cursor.col = 0
		}
		if !extend {
			e.anchor = e.cursor
		}
		e.cursorMoved = true
		e.resetBlink()
		repaint()
	case "End":
		// 设计选择：换行时 End 落到当前视觉段末列 seg.end（贴合所见行宽，而非整逻辑行尾）。
		// 末段时 seg.end==逻辑行尾；中间段（硬断）时 seg.end 是断点列——其插入符会渲染在下一视觉行行首
		// （断点处无 affinity 双向状态，取“归属后续字符”的统一约定，与 segRowOf 一致）。
		if e.wrap {
			e.cursor.col = e.segAt(e.segRowOf(e.cursor.line, e.cursor.col)).end
		} else {
			e.cursor.col = len(e.lineRunes(e.cursor.line))
		}
		if !extend {
			e.anchor = e.cursor
		}
		e.cursorMoved = true
		e.resetBlink()
		repaint()
	}
	return true
}

func (e *CodeEditorElement) anchorDiffLine() bool { return e.cursor.line != e.anchor.line }

// selectWord 选中光标所在的标识符单词。
func (e *CodeEditorElement) selectWord() {
	r := e.lineRunes(e.cursor.line)
	c := e.cursor.col
	s := c
	for s > 0 && isIdentPart(r[s-1]) {
		s--
	}
	en := c
	for en < len(r) && isIdentPart(r[en]) {
		en++
	}
	if en > s {
		e.anchor = cePos{e.cursor.line, s}
		e.cursor = cePos{e.cursor.line, en}
		repaint()
	}
}

func (e *CodeEditorElement) contextItems() []MenuItem {
	hasSel := e.hasSel()
	return []MenuItem{
		{Label: "撤销", Enabled: e.canUndo(), Shortcut: "Ctrl+Z", OnClick: func() {
			e.undo()
		}},
		{Label: "重做", Enabled: e.canRedo(), Shortcut: "Ctrl+Shift+Z", OnClick: func() {
			e.redo()
		}},
		{Separator: true},
		{Label: "剪切", Enabled: hasSel && ClipboardWrite != nil, Shortcut: "Ctrl+X", OnClick: func() {
			if e.hasSel() && ClipboardWrite != nil {
				ClipboardWrite(e.selText())
				e.recordUndo("cut")
				e.deleteSel()
				e.afterEdit()
			}
		}},
		{Label: "复制", Enabled: hasSel && ClipboardWrite != nil, Shortcut: "Ctrl+C", OnClick: func() {
			if e.hasSel() && ClipboardWrite != nil {
				ClipboardWrite(e.selText())
			}
		}},
		{Label: "粘贴", Enabled: ClipboardRead != nil, Shortcut: "Ctrl+V", OnClick: func() {
			if ClipboardRead != nil {
				e.recordUndo("paste")
				e.insertStr(ClipboardRead())
			}
		}},
		{Separator: true},
		{Label: "全选", Enabled: true, Shortcut: "Ctrl+A", OnClick: func() {
			e.anchor = cePos{0, 0}
			last := len(e.lines) - 1
			e.cursor = cePos{last, len(e.lineRunes(last))}
			repaint()
		}},
		{Separator: true},
		{Label: "格式化文档", Enabled: e.lspReady, Shortcut: "Alt+Shift+F", OnClick: func() {
			e.formatDocument()
		}},
	}
}

func (e *CodeEditorElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*CodeEditor); ok {
		e.ed = nc
		e.lang = ceLangFor(nc.Language)
		e.BaseElement.widget = newWidget
		e.dirty = true
		// 受控重载：ReloadToken 变了 → 把运行时内容重置为新文件（切换打开的文件）。
		if nc.ReloadToken != e.lastReload {
			e.lastReload = nc.ReloadToken
			e.lines = strings.Split(expandTabs(nc.initial), "\n")
			if len(e.lines) == 0 {
				e.lines = []string{""}
			}
			e.cursor = cePos{0, 0}
			e.anchor = e.cursor
			e.scrollX, e.scrollY = 0, 0
			e.rehighlight()
			e.computeVisible()
			// 标签编辑器复用单 Element 跨文件：① 语言服务器变了（切到别的语言）→ 重启 LSP；
			// ② 同服务器换文件 → 关旧开新（未就绪则仅更新目标，待 startLSP 打开当前文件）。
			if nc.LSPServer != e.lspServer {
				e.restartLSP(nc)
			} else if nc.LSPFile != "" && nc.LSPFile != e.lspURI {
				if e.lspReady {
					e.switchLSPFile(nc.LSPFile)
				} else {
					e.lspURI = nc.LSPFile
				}
			}
		}
		// 跳转到行：RevealToken 变了 → 移光标到 RevealLine 并滚动到可见（在重载之后，行已就绪）。
		if nc.RevealToken != e.lastReveal {
			e.lastReveal = nc.RevealToken
			if nc.RevealLine > 0 {
				e.revealLine(nc.RevealLine)
			}
		}
	}
}

