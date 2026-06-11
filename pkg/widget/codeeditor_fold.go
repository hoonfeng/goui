package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// 代码折叠：按 { } 配对找可折叠区域（跳过字符串/注释里的括号），gutter 显示 ▼/▶ 箭头，
// 折叠后隐藏区域内行。渲染/光标/点击都基于「可见行」visRows（折叠后行号不连续）。

const ceFoldW = 14.0 // 折叠箭头列宽（行号栏右侧）

// inStrOrComment 判断某位置是否落在字符串/注释 token 内（折叠时跳过其中的括号）。
func (e *CodeEditorElement) inStrOrComment(line, col int) bool {
	if line < 0 || line >= len(e.hl) {
		return false
	}
	for _, tk := range e.hl[line] {
		if col >= tk.start && col < tk.end && (tk.kind == tkString || tk.kind == tkComment) {
			return true
		}
	}
	return false
}

// computeFolds 扫描所有行，按 { } 配对算可折叠区域。
func (e *CodeEditorElement) computeFolds() {
	e.foldRanges = e.foldRanges[:0]
	var stack []int
	for li := range e.lines {
		r := e.lineRunes(li)
		for ci, ch := range r {
			if ch != '{' && ch != '}' {
				continue
			}
			if e.inStrOrComment(li, ci) {
				continue
			}
			if ch == '{' {
				stack = append(stack, li)
			} else if len(stack) > 0 {
				start := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if li > start { // 跨行才可折叠
					e.foldRanges = append(e.foldRanges, foldRange{start, li})
				}
			}
		}
	}
}

// foldEndOf 返回起始行 start 对应折叠区域的结束行（取最外层=最大 end）。
func (e *CodeEditorElement) foldEndOf(start int) (int, bool) {
	end, ok := -1, false
	for _, f := range e.foldRanges {
		if f.start == start && f.end > end {
			end, ok = f.end, true
		}
	}
	return end, ok
}

func (e *CodeEditorElement) isFoldStart(line int) bool {
	_, ok := e.foldEndOf(line)
	return ok
}

// computeVisible 据当前折叠状态算可见行列表 + 实际行→可见索引映射。
func (e *CodeEditorElement) computeVisible() {
	e.computeFolds()
	hidden := make([]bool, len(e.lines))
	for start, on := range e.folded {
		if !on {
			continue
		}
		if end, ok := e.foldEndOf(start); ok {
			for l := start + 1; l <= end && l < len(e.lines); l++ {
				hidden[l] = true
			}
		}
	}
	e.visRows = e.visRows[:0]
	if e.actualToVis == nil {
		e.actualToVis = map[int]int{}
	}
	for k := range e.actualToVis {
		delete(e.actualToVis, k)
	}
	lastVis := 0
	for li := 0; li < len(e.lines); li++ {
		if !hidden[li] {
			lastVis = len(e.visRows)
			e.visRows = append(e.visRows, li)
		}
		e.actualToVis[li] = lastVis
	}
	e.invalidateWrap() // 可见行变了 → 视觉段需重建
}

// visIndexOf 实际行号 → 可见行索引（隐藏行返回其上方可见行索引）。
func (e *CodeEditorElement) visIndexOf(line int) int {
	if v, ok := e.actualToVis[line]; ok {
		return v
	}
	return 0
}

// toggleFold 折叠/展开某可折叠起始行。
func (e *CodeEditorElement) toggleFold(start int) {
	if !e.isFoldStart(start) {
		return
	}
	if e.folded == nil {
		e.folded = map[int]bool{}
	}
	e.folded[start] = !e.folded[start]
	if e.folded[start] { // 折叠时把落在隐藏区的光标移到折叠起始行末
		if end, ok := e.foldEndOf(start); ok && e.cursor.line > start && e.cursor.line <= end {
			e.cursor = cePos{start, len(e.lineRunes(start))}
			e.anchor = e.cursor
		}
	}
	e.computeVisible()
	e.cursorMoved = true
	repaint()
}

// paintFoldArrow 在行号栏右侧画折叠箭头（▼ 展开 / ▶ 折叠，自绘小三角不依赖字体）。
func (e *CodeEditorElement) paintFoldArrow(cvs canvas.Canvas, gx, ly float64, folded bool) {
	cx := gx + ceFoldW/2
	cy := ly + ceLineH/2
	ap := paint.DefaultStrokePaint()
	ap.Color = types.ColorFromRGB(0x6A, 0x73, 0x7D)
	ap.StrokeWidth = 1.4
	if folded { // ▶
		cvs.DrawLine(cx-2, cy-4, cx+3, cy, ap)
		cvs.DrawLine(cx+3, cy, cx-2, cy+4, ap)
	} else { // ▼
		cvs.DrawLine(cx-4, cy-2, cx, cy+3, ap)
		cvs.DrawLine(cx, cy+3, cx+4, cy-2, ap)
	}
}

// foldArrowAt 若 (x,y) 命中某可见折叠起始行的箭头，返回该起始行。
func (e *CodeEditorElement) foldArrowAt(x, y float64) (int, bool) {
	pos := e.Offset()
	gx := pos.X + e.gutterW - ceFoldW
	if x < gx || x > pos.X+e.gutterW {
		return 0, false
	}
	contentTop := pos.Y + 4
	vi := int((y - contentTop + e.scrollY) / ceLineH)
	if vi < 0 || vi >= len(e.wrapSegs) {
		return 0, false
	}
	sg := e.wrapSegs[vi]
	if sg.start != 0 { // 折叠箭头只在逻辑行首段
		return 0, false
	}
	if e.isFoldStart(sg.line) {
		return sg.line, true
	}
	return 0, false
}
