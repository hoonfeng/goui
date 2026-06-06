package widget

import "sort"

// 多光标：主光标 cursor/anchor + 额外 extraCarets。Ctrl+点击加光标、同时编辑、Esc 取消。

func (e *CodeEditorElement) hasMultiCaret() bool { return len(e.extraCarets) > 0 }

// forEachCaret 对每个光标（主+额外）依次设为 e.cursor/anchor、执行 fn、读回。
// 用于光标移动（不改文本，顺序无关）。
func (e *CodeEditorElement) forEachCaret(fn func()) {
	main := caretSel{e.cursor, e.anchor}
	fn()
	main.cursor, main.anchor = e.cursor, e.anchor
	for i := range e.extraCarets {
		e.cursor, e.anchor = e.extraCarets[i].cursor, e.extraCarets[i].anchor
		fn()
		e.extraCarets[i].cursor, e.extraCarets[i].anchor = e.cursor, e.anchor
	}
	e.cursor, e.anchor = main.cursor, main.anchor
	e.dedupCarets()
}

// editEachCaret 多光标同时编辑（v1 各光标按点处理，无独立选区）。用【全局 rune 偏移】跟踪：
// 升序处理，每次编辑测净增量 delta，把后续光标全局偏移 +delta——这样靠前编辑改变靠后光标位置时仍正确。
func (e *CodeEditorElement) editEachCaret(kind string, op func()) {
	if len(e.extraCarets) == 0 {
		e.recordUndo(kind)
		op()
		return
	}
	e.recordUndo(kind)
	e.suppressUndo = true
	defer func() { e.suppressUndo = false }()

	globals := make([]int, 0, len(e.extraCarets)+1)
	globals = append(globals, e.ceGlobalOf(e.cursor))
	for _, c := range e.extraCarets {
		globals = append(globals, e.ceGlobalOf(c.cursor))
	}
	sort.Ints(globals)
	globals = dedupInts(globals)

	for i := 0; i < len(globals); i++ {
		p := e.cePosFromGlobal(globals[i])
		e.cursor, e.anchor = p, p
		before := e.totalRunes()
		op()
		delta := e.totalRunes() - before
		globals[i] = e.ceGlobalOf(e.cursor)
		for j := i + 1; j < len(globals); j++ {
			globals[j] += delta
		}
	}
	// 写回：第一个为主光标，其余为额外光标
	e.cursor = e.cePosFromGlobal(globals[0])
	e.anchor = e.cursor
	e.extraCarets = e.extraCarets[:0]
	for i := 1; i < len(globals); i++ {
		p := e.cePosFromGlobal(globals[i])
		e.extraCarets = append(e.extraCarets, caretSel{p, p})
	}
}

// ── 全局偏移换算（多光标编辑用）──
func (e *CodeEditorElement) ceGlobalOf(p cePos) int {
	g := 0
	for i := 0; i < p.line && i < len(e.lines); i++ {
		g += len([]rune(e.lines[i])) + 1
	}
	return g + p.col
}
func (e *CodeEditorElement) cePosFromGlobal(g int) cePos {
	if g < 0 {
		g = 0
	}
	for i := 0; i < len(e.lines); i++ {
		ll := len([]rune(e.lines[i]))
		if g <= ll {
			return cePos{i, g}
		}
		g -= ll + 1
	}
	last := len(e.lines) - 1
	if last < 0 {
		return cePos{0, 0}
	}
	return cePos{last, len([]rune(e.lines[last]))}
}
func (e *CodeEditorElement) totalRunes() int {
	n := 0
	for i, l := range e.lines {
		if i > 0 {
			n++ // 行间换行符
		}
		n += len([]rune(l))
	}
	return n
}
func dedupInts(s []int) []int {
	out := s[:0]
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}

// dedupCarets 去掉与主光标或彼此重合的额外光标。
func (e *CodeEditorElement) dedupCarets() {
	if len(e.extraCarets) == 0 {
		return
	}
	seen := map[cePos]bool{e.cursor: true}
	out := e.extraCarets[:0]
	for _, c := range e.extraCarets {
		if seen[c.cursor] {
			continue
		}
		seen[c.cursor] = true
		out = append(out, c)
	}
	e.extraCarets = out
}

// addCaretAt 在指定位置加一个光标（Ctrl+点击）：原主光标转为额外，主光标移到 p。
func (e *CodeEditorElement) addCaretAt(p cePos) {
	e.extraCarets = append(e.extraCarets, caretSel{e.cursor, e.anchor})
	e.cursor = p
	e.anchor = p
	e.dedupCarets()
	e.cursorMoved = true
	e.breakUndo()
	repaint()
}

// clearExtraCarets 取消所有额外光标（Esc）。
func (e *CodeEditorElement) clearExtraCarets() {
	if len(e.extraCarets) > 0 {
		e.extraCarets = nil
		repaint()
	}
}
