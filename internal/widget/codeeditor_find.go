package widget

import (
	"strings"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// CodeEditor 查找/替换：Ctrl+F 查找、Ctrl+H 替换；高亮所有匹配 + 上一个/下一个 + 计数。

const ceFindBarH = 34.0

// openFind 打开查找栏（replace=true 同时显示替换行）。
func (e *CodeEditorElement) openFind(replace bool) {
	e.findActive = true
	if replace {
		e.replaceShown = true
	}
	e.findField = 0
	e.findOrigin = e.cursor // 记住原始光标，定位首个匹配时不随输入漂移
	// 选区单行非空→作为初始查找串
	if e.hasSel() {
		lo, hi := e.sortedSel()
		if lo.line == hi.line {
			r := e.lineRunes(lo.line)
			e.findQuery = string(r[lo.col:hi.col])
			e.findCol = len([]rune(e.findQuery))
		}
	}
	e.recomputeMatches()
	e.selectMatchNearCursor()
	repaint()
}

func (e *CodeEditorElement) closeFind() {
	e.findActive = false
	e.replaceShown = false
	e.matches = nil
	e.focused = true
	repaint()
}

// recomputeMatches 重新扫描所有匹配（按文档顺序）。
func (e *CodeEditorElement) recomputeMatches() {
	e.matches = nil
	if e.findQuery == "" {
		return
	}
	needle := e.findQuery
	for li, line := range e.lines {
		hay := line
		nd := needle
		if !e.findCase {
			hay = strings.ToLower(hay)
			nd = strings.ToLower(needle)
		}
		from := 0
		for from <= len(hay) {
			idx := strings.Index(hay[from:], nd)
			if idx < 0 {
				break
			}
			bytePos := from + idx
			startCol := len([]rune(line[:bytePos]))
			endCol := startCol + len([]rune(e.findQuery))
			e.matches = append(e.matches, ceMatch{cePos{li, startCol}, cePos{li, endCol}})
			from = bytePos + len(nd)
			if len(nd) == 0 {
				break
			}
		}
	}
	if e.curMatch >= len(e.matches) {
		e.curMatch = 0
	}
}

// selectMatchNearCursor 把当前匹配设为光标处或之后的第一个匹配。
func (e *CodeEditorElement) selectMatchNearCursor() {
	if len(e.matches) == 0 {
		return
	}
	e.curMatch = 0
	for i, m := range e.matches {
		if !cePosLess(m.start, e.findOrigin) { // 用原始光标定位，输入时不漂移
			e.curMatch = i
			break
		}
	}
	e.gotoMatch(e.curMatch)
}

// gotoMatch 跳到第 idx 个匹配（选中它 + 滚入视野）。
func (e *CodeEditorElement) gotoMatch(idx int) {
	if len(e.matches) == 0 {
		return
	}
	idx = (idx%len(e.matches) + len(e.matches)) % len(e.matches)
	e.curMatch = idx
	m := e.matches[idx]
	e.anchor = m.start
	e.cursor = m.end
	e.cursorMoved = true
	repaint()
}

func (e *CodeEditorElement) nextMatch() { e.gotoMatch(e.curMatch + 1) }
func (e *CodeEditorElement) prevMatch() { e.gotoMatch(e.curMatch - 1) }

// replaceCurrent 替换当前匹配为替换串，再跳下一个。
func (e *CodeEditorElement) replaceCurrent() {
	if e.curMatch >= len(e.matches) {
		return
	}
	m := e.matches[e.curMatch]
	e.recordUndo("replace")
	r := e.lineRunes(m.start.line)
	e.lines[m.start.line] = string(r[:m.start.col]) + e.replaceQuery + string(r[m.end.col:])
	e.cursor = cePos{m.start.line, m.start.col + len([]rune(e.replaceQuery))}
	e.anchor = e.cursor
	e.afterEdit()
	e.recomputeMatches()
	if len(e.matches) > 0 {
		e.gotoMatch(e.curMatch)
	}
}

// replaceAll 替换所有匹配（逐行从后往前，避免列偏移失效）。
func (e *CodeEditorElement) replaceAll() {
	if len(e.matches) == 0 {
		return
	}
	e.recordUndo("replaceAll")
	// 按行分组，每行内从右往左替换
	for li := len(e.lines) - 1; li >= 0; li-- {
		var inLine []ceMatch
		for _, m := range e.matches {
			if m.start.line == li {
				inLine = append(inLine, m)
			}
		}
		for k := len(inLine) - 1; k >= 0; k-- {
			m := inLine[k]
			r := e.lineRunes(li)
			e.lines[li] = string(r[:m.start.col]) + e.replaceQuery + string(r[m.end.col:])
		}
	}
	e.cursor = e.clampPos(e.cursor)
	e.anchor = e.cursor
	e.afterEdit()
	e.recomputeMatches()
}

// ── 查找栏输入（在查找/替换框内编辑）──

func (e *CodeEditorElement) curFindText() (string, int) {
	if e.findField == 1 {
		return e.replaceQuery, e.replaceCol
	}
	return e.findQuery, e.findCol
}
func (e *CodeEditorElement) setFindText(s string, col int) {
	if e.findField == 1 {
		e.replaceQuery, e.replaceCol = s, col
	} else {
		e.findQuery, e.findCol = s, col
		e.recomputeMatches()
		e.selectMatchNearCursor()
	}
}

// handleFindKey 处理查找栏激活时的键。返回是否已消费。
func (e *CodeEditorElement) handleFindKey(k *event.KeyEvent) bool {
	if k.Mods&event.ModCtrl != 0 {
		switch k.Key {
		case "F":
			e.findField = 0
			repaint()
			return true
		case "H":
			e.replaceShown = true
			e.findField = 1
			repaint()
			return true
		}
		return true
	}
	s, col := e.curFindText()
	r := []rune(s)
	switch k.Key {
	case "Escape":
		e.closeFind()
	case "Enter":
		if e.findField == 1 {
			e.replaceCurrent()
		} else if k.Mods&event.ModShift != 0 {
			e.prevMatch()
		} else {
			e.nextMatch()
		}
	case "Tab":
		if e.replaceShown {
			e.findField = 1 - e.findField
			repaint()
		}
	case "Backspace":
		if col > 0 {
			e.setFindText(string(r[:col-1])+string(r[col:]), col-1)
			repaint()
		}
	case "Delete":
		if col < len(r) {
			e.setFindText(string(r[:col])+string(r[col+1:]), col)
			repaint()
		}
	case "ArrowLeft":
		if col > 0 {
			e.findColSet(col - 1)
		}
	case "ArrowRight":
		if col < len(r) {
			e.findColSet(col + 1)
		}
	case "Home":
		e.findColSet(0)
	case "End":
		e.findColSet(len(r))
	}
	return true
}

func (e *CodeEditorElement) findColSet(col int) {
	if e.findField == 1 {
		e.replaceCol = col
	} else {
		e.findCol = col
	}
	repaint()
}

// handleFindChar 在查找/替换框插入一个字符。
func (e *CodeEditorElement) handleFindChar(ch rune) {
	s, col := e.curFindText()
	r := []rune(s)
	ns := string(r[:col]) + string(ch) + string(r[col:])
	e.setFindText(ns, col+1)
	repaint()
}

// ── 渲染 ──

// paintMatchHighlights 高亮所有匹配（当前匹配更醒目）。在编辑区裁剪内调用。
func (e *CodeEditorElement) paintMatchHighlights(cvs canvas.Canvas, left, top float64) {
	for i, m := range e.matches {
		x0 := left + e.colToX(m.start.line, m.start.col)
		x1 := left + e.colToX(m.end.line, m.end.col)
		ly := e.lineTopY(m.start.line, top)
		hp := paint.DefaultPaint()
		if i == e.curMatch {
			hp.Color = types.ColorFromRGBA(0xFF, 0x96, 0x00, 130) // 当前匹配：橙
		} else {
			hp.Color = types.ColorFromRGBA(0xFF, 0xE0, 0x66, 130) // 其它匹配：黄
		}
		cvs.DrawRect(x0, ly+1, x1-x0, ceLineH-2, hp)
	}
}

// paintFindBar 绘制右上角查找栏，并填充按钮命中区。
func (e *CodeEditorElement) paintFindBar(cvs canvas.Canvas, pos types.Point, w float64) {
	e.findBtns = e.findBtns[:0]
	barW := 320.0
	if barW > w-20 {
		barW = w - 20
	}
	rows := 1.0
	if e.replaceShown {
		rows = 2
	}
	bx := pos.X + w - barW - 8
	by := pos.Y + 6
	bh := ceFindBarH*rows + 6

	// 背景卡片
	sh := paint.DefaultPaint()
	sh.Color = types.ColorFromRGBA(0, 0, 0, 28)
	cvs.DrawRoundedRect(bx, by+2, barW, bh, 6, sh)
	bg := paint.DefaultPaint()
	bg.Color = types.ColorFromRGB(0xFA, 0xFA, 0xFA)
	cvs.DrawRoundedRect(bx, by, barW, bh, 6, bg)
	bd := paint.DefaultStrokePaint()
	bd.Color = elBorder()
	cvs.DrawRoundedRect(bx+0.5, by+0.5, barW-1, bh-1, 6, bd)

	font := canvas.Font{Family: "Consolas", Size: 13}
	// ── 查找行 ──
	e.paintFindField(cvs, bx+8, by+5, barW-150, ceFindBarH-8, e.findQuery, e.findCol, e.findField == 0 && e.findActive, "查找", "focusFind", font)
	// 计数
	count := "无结果"
	if len(e.matches) > 0 {
		count = itoaCE(e.curMatch+1) + "/" + itoaCE(len(e.matches))
	}
	canvas.DrawTextAligned(cvs, count, types.Rect{X: bx + barW - 142, Y: by + 5, Width: 56, Height: ceFindBarH - 8}, font, elTextSecondary(), canvas.HAlignRight, canvas.VAlignMiddle)
	// 按钮：Aa(大小写) ◀ ▶ ✕
	e.findButton(cvs, bx+barW-82, by+5, "Aa", "case", e.findCase, font)
	e.findButton(cvs, bx+barW-58, by+5, "‹", "prev", false, font)
	e.findButton(cvs, bx+barW-36, by+5, "›", "next", false, font)
	e.findButton(cvs, bx+barW-16-2, by+5, "✕", "close", false, font)

	// ── 替换行 ──
	if e.replaceShown {
		ry := by + ceFindBarH
		e.paintFindField(cvs, bx+8, ry+1, barW-150, ceFindBarH-8, e.replaceQuery, e.replaceCol, e.findField == 1 && e.findActive, "替换", "focusReplace", font)
		e.findButton2(cvs, bx+barW-138, ry+1, 60, "替换", "replace", font)
		e.findButton2(cvs, bx+barW-74, ry+1, 64, "全部", "replaceAll", font)
	}
}

func (e *CodeEditorElement) paintFindField(cvs canvas.Canvas, x, y, fw, fh float64, text string, col int, active bool, ph, action string, font canvas.Font) {
	e.findBtns = append(e.findBtns, ceFindBtn{types.Rect{X: x, Y: y, Width: fw, Height: fh}, action})
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(x, y, fw, fh, 4, bg)
	bd := paint.DefaultStrokePaint()
	if active {
		bd.Color = elPrimary()
	} else {
		bd.Color = elBorder()
	}
	cvs.DrawRoundedRect(x+0.5, y+0.5, fw-1, fh-1, 4, bd)

	rr := []rune(text)
	if col > len(rr) {
		col = len(rr)
	}
	compActive := active && e.findActive && e.composition != "" // 查找/替换框内显示 IME 组合预览
	tx := x + 6
	baseY := canvas.BaselineFor(y, fh, font.Size, canvas.VAlignMiddle)
	dark := types.ColorFromRGB(0x24, 0x29, 0x2E)

	if text == "" && !compActive {
		canvas.DrawTextAligned(cvs, ph, types.Rect{X: tx, Y: y, Width: fw - 12, Height: fh}, font, types.ColorFromRGB(180, 180, 180), canvas.HAlignLeft, canvas.VAlignMiddle)
	} else {
		// 绘制 text[:col] + 组合预览 + text[col:]
		before := string(rr[:col])
		cvs.DrawText(before, tx, baseY, font, mkPaint(dark))
		bw := cvs.MeasureText(before, font).Width
		compW := 0.0
		if compActive {
			cvs.DrawText(e.composition, tx+bw, baseY, font, mkPaint(dark))
			compW = cvs.MeasureText(e.composition, font).Width
			ul := paint.DefaultStrokePaint()
			ul.Color = types.ColorFromRGB(0x40, 0x9E, 0xFF)
			ul.StrokeWidth = 1
			cvs.DrawLine(tx+bw, baseY+2, tx+bw+compW, baseY+2, ul)
		}
		cvs.DrawText(string(rr[col:]), tx+bw+compW, baseY, font, mkPaint(dark))
	}
	// 光标 + IME 候选定位（候选窗口落到查找/替换框光标处，而非编辑区光标）
	if active {
		cx := tx + cvs.MeasureText(string(rr[:col]), font).Width
		if compActive {
			cx += cvs.MeasureText(e.composition, font).Width
		}
		cp := paint.DefaultStrokePaint()
		cp.Color = dark
		cp.StrokeWidth = 1.2
		cvs.DrawLine(cx, y+4, cx, y+fh-4, cp)
		e.cursorClientX = cx
		e.cursorClientY = y + 4 // 光标顶部（与 InputElement 一致，候选紧贴下方不偏下）
	}
}

func (e *CodeEditorElement) findButton(cvs canvas.Canvas, x, y float64, label, action string, on bool, font canvas.Font) {
	rect := types.Rect{X: x, Y: y, Width: 20, Height: ceFindBarH - 8}
	if on {
		hl := paint.DefaultPaint()
		hl.Color = types.ColorFromRGB(0xD9, 0xEC, 0xFF)
		cvs.DrawRoundedRect(rect.X, rect.Y, rect.Width, rect.Height, 3, hl)
	}
	col := elTextRegular()
	if on {
		col = elPrimary()
	}
	canvas.DrawTextAligned(cvs, label, rect, font, col, canvas.HAlignCenter, canvas.VAlignMiddle)
	e.findBtns = append(e.findBtns, ceFindBtn{rect, action})
}

func (e *CodeEditorElement) findButton2(cvs canvas.Canvas, x, y, bw float64, label, action string, font canvas.Font) {
	rect := types.Rect{X: x, Y: y + 2, Width: bw, Height: ceFindBarH - 10}
	bg := paint.DefaultPaint()
	bg.Color = types.ColorFromRGB(0xF0, 0xF2, 0xF5)
	cvs.DrawRoundedRect(rect.X, rect.Y, rect.Width, rect.Height, 4, bg)
	bd := paint.DefaultStrokePaint()
	bd.Color = elBorder()
	cvs.DrawRoundedRect(rect.X+0.5, rect.Y+0.5, rect.Width-1, rect.Height-1, 4, bd)
	canvas.DrawTextAligned(cvs, label, rect, font, elTextRegular(), canvas.HAlignCenter, canvas.VAlignMiddle)
	e.findBtns = append(e.findBtns, ceFindBtn{rect, action})
}

// findBarHit 处理查找栏点击。返回是否命中。
func (e *CodeEditorElement) findBarHit(x, y float64) bool {
	for _, b := range e.findBtns {
		if x >= b.rect.X && x <= b.rect.X+b.rect.Width && y >= b.rect.Y && y <= b.rect.Y+b.rect.Height {
			switch b.action {
			case "focusFind":
				e.findField = 0
			case "focusReplace":
				e.findField = 1
			case "case":
				e.findCase = !e.findCase
				e.recomputeMatches()
			case "prev":
				e.prevMatch()
			case "next":
				e.nextMatch()
			case "close":
				e.closeFind()
			case "replace":
				e.replaceCurrent()
			case "replaceAll":
				e.replaceAll()
			}
			repaint()
			return true
		}
	}
	return false
}
