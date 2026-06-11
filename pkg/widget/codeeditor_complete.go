package widget

import (
	"sort"
	"strings"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// 智能补全（第一步：词法补全——当前文件标识符 + 语言关键字/类型；UI/触发/接受完整）。
// 数据源 collectCompletions 为可替换点：后续接 LSP(gopls) 时改由语言服务器返回候选。

type compKind uint8

const (
	ckIdent   compKind = iota // 文件标识符
	ckKeyword                 // 关键字
	ckType                    // 类型/内置
)

type compItem struct {
	label  string
	kind   compKind
	detail string // 类型签名（LSP 提供，如 func(...) error）
}

const (
	ceCompRowH = 20.0
	ceCompMaxN = 8 // 弹窗最多显示行数
)

// wordBeforeCursor 返回光标前的标识符前缀 + 其起始位置。
func (e *CodeEditorElement) wordBeforeCursor() (string, cePos) {
	r := e.lineRunes(e.cursor.line)
	c := e.cursor.col
	if c > len(r) {
		c = len(r)
	}
	s := c
	for s > 0 && isIdentPart(r[s-1]) {
		s--
	}
	return string(r[s:c]), cePos{e.cursor.line, s}
}

// collectCompletions 词法补全候选：文件标识符（优先）+ 关键字 + 类型，前缀匹配去重。
func (e *CodeEditorElement) collectCompletions(prefix string) []compItem {
	lp := strings.ToLower(prefix)
	seen := map[string]bool{}
	var ids, kws, tys, ext []compItem
	matchPrefix := func(w string) bool { // 原文前缀 或 拼音首字母前缀（如 "scts"→"输出调试…"）
		if lp == "" {
			return true
		}
		if strings.HasPrefix(strings.ToLower(w), lp) {
			return true
		}
		return strings.HasPrefix(pinyinInitials(w), lp)
	}
	add := func(w string, k compKind, dst *[]compItem) {
		if w == "" || seen[w] || !matchPrefix(w) {
			return
		}
		seen[w] = true
		*dst = append(*dst, compItem{label: w, kind: k})
	}
	for _, line := range e.lines { // 文件里所有标识符
		r := []rune(line)
		for i := 0; i < len(r); {
			if isIdentStart(r[i]) {
				j := i + 1
				for j < len(r) && isIdentPart(r[j]) {
					j++
				}
				add(string(r[i:j]), ckIdent, &ids)
				i = j
			} else {
				i++
			}
		}
	}
	for w := range e.lang.keywords {
		add(w, ckKeyword, &kws)
	}
	for w := range e.lang.types {
		add(w, ckType, &tys)
	}
	if e.ed.ExtraIdents != nil { // StructEditor 注入：已声明变量/子程序/命令（优先展示）
		for _, ci := range e.ed.ExtraIdents() {
			if ci.Label == "" || seen[ci.Label] || !matchPrefix(ci.Label) {
				continue
			}
			seen[ci.Label] = true
			ext = append(ext, compItem{label: ci.Label, kind: ckIdent, detail: ci.Detail})
		}
	}
	sort.Slice(ext, func(i, j int) bool { return ext[i].label < ext[j].label })
	sort.Slice(ids, func(i, j int) bool { return ids[i].label < ids[j].label })
	sort.Slice(kws, func(i, j int) bool { return kws[i].label < kws[j].label })
	sort.Slice(tys, func(i, j int) bool { return tys[i].label < tys[j].label })
	out := make([]compItem, 0, len(ext)+len(ids)+len(kws)+len(tys))
	out = append(out, ext...) // 变量/子程序/命令排最前
	out = append(out, ids...)
	out = append(out, kws...)
	out = append(out, tys...)
	return out
}

// updateCompletion 据光标前词前缀刷新候选；空前缀/无候选/唯一且等于前缀则关闭。
func (e *CodeEditorElement) updateCompletion() {
	if e.hasMultiCaret() || e.hasSel() {
		e.completing = false
		return
	}
	prefix, start := e.wordBeforeCursor()
	if len([]rune(prefix)) < 1 {
		// 前缀空：若紧跟 "." 且有 LSP，触发成员补全（如 fmt.→Println…）
		r := e.lineRunes(e.cursor.line)
		if e.lspReady && e.cursor.col > 0 && e.cursor.col-1 < len(r) && r[e.cursor.col-1] == '.' {
			e.compItems = nil
			e.compStart = e.cursor
			e.compSel = 0
			e.compScroll = 0
			e.completing = true
			e.lspGen++
			e.requestLSPCompletion("", e.cursor.line, e.cursor.col, e.lspGen)
			return
		}
		e.completing = false
		return
	}
	items := e.collectCompletions(prefix)
	// 无 LSP 时：词法空/唯一且等于前缀→关闭。有 LSP 时即使词法空也请求语义候选。
	if !e.lspReady && (len(items) == 0 || (len(items) == 1 && items[0].label == prefix)) {
		e.completing = false
		return
	}
	e.compItems = items
	e.compStart = start
	e.compSel = 0
	e.compScroll = 0
	e.completing = true
	e.lspGen++ // 词法候选先显示；若有 LSP 再异步请求语义候选替换
	if e.lspReady {
		e.requestLSPCompletion(prefix, e.cursor.line, e.cursor.col, e.lspGen)
	}
}

// triggerCompletion 手动触发（Ctrl+Space）。
func (e *CodeEditorElement) triggerCompletion() {
	prefix, start := e.wordBeforeCursor()
	items := e.collectCompletions(prefix)
	if len(items) == 0 {
		return
	}
	e.compItems = items
	e.compStart = start
	e.compSel = 0
	e.compScroll = 0
	e.completing = true
	e.lspGen++
	if e.lspReady {
		e.requestLSPCompletion(prefix, e.cursor.line, e.cursor.col, e.lspGen)
	}
	repaint()
}

func (e *CodeEditorElement) cancelCompletion() { e.completing = false }

func (e *CodeEditorElement) compMove(d int) {
	n := len(e.compItems)
	if n == 0 {
		return
	}
	e.compSel = ((e.compSel+d)%n + n) % n
	if e.compSel < e.compScroll {
		e.compScroll = e.compSel
	}
	if e.compSel >= e.compScroll+ceCompMaxN {
		e.compScroll = e.compSel - ceCompMaxN + 1
	}
	repaint()
}

// acceptCompletion 用选中候选替换词前缀。
func (e *CodeEditorElement) acceptCompletion() {
	if !e.completing || e.compSel < 0 || e.compSel >= len(e.compItems) {
		return
	}
	label := e.compItems[e.compSel].label
	e.recordUndo("complete")
	r := e.lineRunes(e.compStart.line)
	col := e.cursor.col
	if col > len(r) {
		col = len(r)
	}
	e.lines[e.compStart.line] = string(r[:e.compStart.col]) + label + string(r[col:])
	e.cursor = cePos{e.compStart.line, e.compStart.col + len([]rune(label))}
	e.anchor = e.cursor
	e.completing = false
	e.afterEdit()
}

// ── 渲染 ──

func ceCompColor(k compKind) types.Color {
	switch k {
	case ckKeyword:
		return types.ColorFromRGB(0xD7, 0x3A, 0x49)
	case ckType:
		return types.ColorFromRGB(0x6F, 0x42, 0xC1)
	default:
		return types.ColorFromRGB(0x24, 0x29, 0x2E)
	}
}
func ceCompIcon(k compKind) string {
	switch k {
	case ckKeyword:
		return "K"
	case ckType:
		return "T"
	default:
		return "abc"
	}
}

// paintCompletion 画补全弹窗（编辑器内容之上，可超出编辑区）。left/top 为编辑区文本起点。
func (e *CodeEditorElement) paintCompletion(cvs canvas.Canvas, left, top float64) {
	if !e.completing || len(e.compItems) == 0 {
		return
	}
	font := e.font
	// 弹窗宽度：按可见项最长 label（含图标列）
	maxW := 80.0
	visN := len(e.compItems)
	if visN > ceCompMaxN {
		visN = ceCompMaxN
	}
	for i := 0; i < visN; i++ {
		it := e.compItems[e.compScroll+i]
		if wd := e.measure(it.label) + e.measure(it.detail) + 64; wd > maxW {
			maxW = wd
		}
	}
	if maxW > 380 {
		maxW = 380
	}
	x := e.posX(e.compStart.line, e.compStart.col, left)
	rowsH := float64(visN)*ceCompRowH + 4
	yTop := e.posTopY(e.cursor.line, e.cursor.col, top) + ceLineH
	pos := e.Offset()
	if yTop+rowsH > pos.Y+e.size.Height-2 { // 下方放不下→往上弹
		yTop = e.posTopY(e.cursor.line, e.cursor.col, top) - rowsH
	}

	// 阴影 + 背景 + 边框
	sh := paint.DefaultPaint()
	sh.Color = types.ColorFromRGBA(0, 0, 0, 30)
	cvs.DrawRoundedRect(x, yTop+2, maxW, rowsH, 5, sh)
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(x, yTop, maxW, rowsH, 5, bg)
	bd := paint.DefaultStrokePaint()
	bd.Color = elBorder()
	cvs.DrawRoundedRect(x+0.5, yTop+0.5, maxW-1, rowsH-1, 5, bd)

	for i := 0; i < visN; i++ {
		idx := e.compScroll + i
		it := e.compItems[idx]
		ry := yTop + 2 + float64(i)*ceCompRowH
		if idx == e.compSel { // 选中高亮
			hl := paint.DefaultPaint()
			hl.Color = types.ColorFromRGB(0xE8, 0xF0, 0xFE)
			cvs.DrawRoundedRect(x+2, ry, maxW-4, ceCompRowH, 3, hl)
		}
		// 类型图标
		ic := canvas.DefaultFont()
		ic.Size = 11
		canvas.DrawTextAligned(cvs, ceCompIcon(it.kind), types.Rect{X: x + 4, Y: ry, Width: 32, Height: ceCompRowH}, ic, ceCompColor(it.kind), canvas.HAlignLeft, canvas.VAlignMiddle)
		// label + 签名(detail，灰显)
		canvas.DrawTextAligned(cvs, it.label, types.Rect{X: x + 40, Y: ry, Width: maxW - 44, Height: ceCompRowH}, font, types.ColorFromRGB(0x24, 0x29, 0x2E), canvas.HAlignLeft, canvas.VAlignMiddle)
		if it.detail != "" {
			lw := e.measure(it.label)
			dx := x + 40 + lw + 8
			if dx < x+maxW-8 {
				df := canvas.DefaultFont()
				df.Size = 11.5
				canvas.DrawTextAligned(cvs, it.detail, types.Rect{X: dx, Y: ry, Width: x + maxW - 6 - dx, Height: ceCompRowH}, df, types.ColorFromRGB(0x8A, 0x90, 0x99), canvas.HAlignLeft, canvas.VAlignMiddle)
			}
		}
	}
	// 滚动指示（更多项）
	if len(e.compItems) > ceCompMaxN {
		sb := paint.DefaultPaint()
		sb.Color = types.ColorFromRGB(193, 193, 193)
		th := rowsH * float64(ceCompMaxN) / float64(len(e.compItems))
		ty := yTop + (rowsH-th)*float64(e.compScroll)/float64(len(e.compItems)-ceCompMaxN)
		cvs.DrawRoundedRect(x+maxW-4, ty, 3, th, 1.5, sb)
	}
}

// paintHover 画悬停信息浮层（showHover 命令/鼠标悬停触发）：符号上方（放不下→下方）一个含类型签名/文档的框。
// 命令触发：光标移开（cursor != hoverCursor）即关闭。
// 鼠标触发：鼠标移开锚点（mouseHoverPos != hoverAnchor）即关闭；e.cursor（编辑光标）不影响。
func (e *CodeEditorElement) paintHover(cvs canvas.Canvas, left, top float64) {
	if e.hoverText == "" {
		return
	}

	if e.hoverByMouse {
		// 鼠标触发：用像素距离判断，>8px 视为移开。
		// mouseX/mouseY 由 TypeMouseMove 每帧刷新；hoverMouseX/Y 在触发时写入。
		dx := e.mouseX - e.hoverMouseX
		dy := e.mouseY - e.hoverMouseY
		if dx*dx+dy*dy > 64 { // 8px 阈值
			e.hoverText = ""
			e.hoverByMouse = false
			return
		}
	} else {
		if !cePosEq(e.cursor, e.hoverCursor) {
			e.hoverText = ""
			return
		}
	}

	// 按 \n 拆分基础行，并做自动换行（长行折到 maxBoxW 内）
	const maxBoxW = 480.0
	const pad = 18.0
	wrapW := maxBoxW - pad // 文本实际可用宽
	rawLines := strings.Split(e.hoverText, "\n")
	var lines []string
	for _, ln := range rawLines {
		wrapped := e.wrapHoverLine(ln, wrapW)
		lines = append(lines, wrapped...)
		if len(lines) > 32 {
			lines = append(lines[:32], "…")
			break
		}
	}
	// 根据折行后的行重新算 maxW（以最宽行为准）
	maxW := 80.0
	for _, ln := range lines {
		if w := e.measure(ln) + pad; w > maxW {
			maxW = w
		}
	}
	if maxW > maxBoxW {
		maxW = maxBoxW
	}
	boxH := float64(len(lines))*ceLineH + 10
	x := e.posX(e.hoverCursor.line, e.hoverCursor.col, left)
	yTop := e.posTopY(e.hoverCursor.line, e.hoverCursor.col, top) - boxH - 4
	pos := e.Offset()
	if yTop < pos.Y+2 {
		yTop = e.posTopY(e.hoverCursor.line, e.hoverCursor.col, top) + ceLineH + 2
	}
	if x+maxW > pos.X+e.size.Width-2 {
		x = pos.X + e.size.Width - 2 - maxW
	}
	if x < pos.X+2 {
		x = pos.X + 2
	}
	sh := paint.DefaultPaint()
	sh.Color = types.ColorFromRGBA(0, 0, 0, 30)
	cvs.DrawRoundedRect(x, yTop+2, maxW, boxH, 5, sh)
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(x, yTop, maxW, boxH, 5, bg)
	bd := paint.DefaultStrokePaint()
	bd.Color = elBorder()
	cvs.DrawRoundedRect(x+0.5, yTop+0.5, maxW-1, boxH-1, 5, bd)
	for i, ln := range lines {
		canvas.DrawTextAligned(cvs, ln, types.Rect{X: x + 8, Y: yTop + 5 + float64(i)*ceLineH, Width: maxW - 14, Height: ceLineH}, e.font, elTextPrimary(), canvas.HAlignLeft, canvas.VAlignMiddle)
	}
}
