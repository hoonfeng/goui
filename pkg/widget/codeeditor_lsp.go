package widget

import (
	"log"
	"strings"
	"unicode"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/lsp"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// CodeEditor 的 LSP 集成：异步启动语言服务器、文档同步、语义补全、诊断波浪线。
// 线程模型：读协程(lsp.Client)写 lspPending/diagnostics 加锁，UI 线程下一帧消费（drain）+ 重绘。

// startLSP 异步启动语言服务器、initialize、didOpen（在 goroutine 上跑，~2.5s）。
func (e *CodeEditorElement) startLSP(server, workspace string) {
	e.lspServer = server
	c, err := lsp.Acquire(server, e.ed.LSPArgs, workspace) // 共享池：复用现有或首次起（非单例，多语言并存）
	if err != nil || c == nil {
		if err != nil {
			log.Printf("❌ LSP 启动失败 %s: %v", server, err)
		}
		MessageError("语言服务器 " + server + " 启动失败，部分智能功能不可用（补全/诊断/悬停）")
		return
	}
	e.bindDiagnostics(c) // 活动编辑器认领该 client 的诊断推送（共享 client，后设的赢）
	c.DidOpen(e.lspURI, e.lspLangID(), e.lspDoc())
	e.lspClient = c
	e.lspReady = true
	repaint()
}

// bindDiagnostics 让本编辑器认领某共享 client 的诊断推送（只收本文件 uri 的）。
func (e *CodeEditorElement) bindDiagnostics(c *lsp.Client) {
	c.OnDiagnostics = func(uri string, ds []lsp.Diagnostic) {
		if uri != e.lspURI {
			return
		}
		e.lspMu.Lock()
		e.diagnostics = ds
		e.lspMu.Unlock()
		repaint()
	}
}

// lspDoc 返回喂给语言服务器的文档文本，并更新行/列偏移。
// 内嵌函数体编辑器经 lspWrap 把本地片段映射到完整 Go 文档；独立编辑器直接用本文。
func (e *CodeEditorElement) lspDoc() string {
	if e.lspWrap == nil {
		e.lspLineOff, e.lspColOff = 0, 0
		return e.text()
	}
	full, lo, co := e.lspWrap(e.text())
	e.lspLineOff, e.lspColOff = lo, co
	return full
}

// lspLangID LSP languageId（如 "typescript"；空=用 Language。gopls 等用 ext 名 "go" 也可）。
func (e *CodeEditorElement) lspLangID() string {
	if e.ed.LSPLangID != "" {
		return e.ed.LSPLangID
	}
	return e.ed.Language
}

// EnsureLSP 内嵌体编辑器获得焦点时按需启动 gopls（独立编辑器在 CreateElement 已启动）。
func (e *CodeEditorElement) EnsureLSP() {
	if e.ed == nil || e.ed.LSPServer == "" || e.lspReady || e.lspStarting {
		return
	}
	e.lspStarting = true
	e.lspURI = e.ed.LSPFile
	go e.startLSP(e.ed.LSPServer, e.ed.LSPWorkspace)
}

// lspDidChange 编辑后把文档全量同步给服务器。
func (e *CodeEditorElement) lspDidChange() {
	if !e.lspReady {
		return
	}
	e.lspVer++
	e.lspClient.DidChange(e.lspURI, e.lspDoc(), e.lspVer)
}

// requestLSPCompletion 异步请求语义补全；结果经 lspPending 转 UI 线程消费。
func (e *CodeEditorElement) requestLSPCompletion(prefix string, line, char, gen int) {
	cl := e.lspClient
	uri := e.lspURI
	line += e.lspLineOff // 内嵌体编辑器：本地坐标 → 完整文档坐标
	char += e.lspColOff
	go func() {
		items, err := cl.Completion(uri, line, char)
		if err != nil || len(items) == 0 {
			return
		}
		conv := convertLSPItems(items, prefix)
		if len(conv) == 0 {
			return
		}
		e.lspMu.Lock()
		e.lspPending = conv
		e.lspPendGen = gen
		e.lspMu.Unlock()
		repaint()
	}()
}

// drainLSPCompletion 在 UI 线程消费 LSP 结果（若仍是当前补全且未过期则替换候选）。Paint 开头调。
func (e *CodeEditorElement) drainLSPCompletion() {
	e.lspMu.Lock()
	pend, gen := e.lspPending, e.lspPendGen
	e.lspPending = nil
	def := e.pendingDef
	e.pendingDef = nil
	refs := e.pendingRefs
	e.pendingRefs = nil
	syms := e.pendingSyms
	e.pendingSyms = nil
	hov := e.pendingHover
	e.pendingHover = nil
	fmts := e.pendingFormats
	e.pendingFormats = nil
	e.lspMu.Unlock()
	if hov != nil { // 悬停结果 → 设浮层内容（paintHover 据 hoverCursor 显示/关闭）
		e.hoverText = hov.text
		e.hoverCursor = hov.at
	}
	if pend != nil && e.completing && gen == e.lspGen {
		e.compItems = pend
		if e.compSel >= len(e.compItems) {
			e.compSel = 0
		}
		e.compScroll = 0
	}
	if def != nil && e.ed != nil && e.ed.OnGoToDefinition != nil { // 转到定义：UI 线程回调宿主打开目标
		e.ed.OnGoToDefinition(def.file, def.line, def.col)
	}
	if refs != nil && e.ed != nil && e.ed.OnReferences != nil { // 查找引用：回调宿主展示
		e.ed.OnReferences(refs)
	}
	if syms != nil && e.ed != nil && e.ed.OnDocumentSymbols != nil { // 文档符号：回调宿主展示
		e.ed.OnDocumentSymbols(syms)
	}
	if fmts != nil { // 格式化结果：应用编辑到缓冲区
		e.applyTextEdits(fmts)
	}
}

// switchLSPFile 切换 LSP 跟踪的文件（标签编辑器复用单 Element 跨文件）：同一 gopls，关旧文档、开新文档。
func (e *CodeEditorElement) switchLSPFile(newURI string) {
	if !e.lspReady || newURI == "" || newURI == e.lspURI {
		return
	}
	e.lspClient.DidClose(e.lspURI)
	e.lspURI = newURI
	e.lspVer = 0
	e.lspMu.Lock()
	e.diagnostics = nil // 旧文件诊断清掉
	e.lspMu.Unlock()
	e.lspClient.DidOpen(e.lspURI, e.lspLangID(), e.lspDoc())
	e.bindDiagnostics(e.lspClient) // 重认领诊断（共享 client 可能被别的编辑器改了 OnDiagnostics）
}

// restartLSP 语言服务器变了（标签编辑器切到需别的服务器的语言，如 .ts→.py）→ 关旧客户端、起新服务器。
func (e *CodeEditorElement) restartLSP(nc *CodeEditor) {
	if e.lspClient != nil {
		e.lspClient.DidClose(e.lspURI) // 只关旧文件文档；旧 server 是共享池常驻，别的文件还在用
		e.lspClient = nil
	}
	e.lspReady = false
	e.lspStarting = false
	e.lspMu.Lock()
	e.diagnostics = nil
	e.lspMu.Unlock()
	e.hoverText = ""
	e.lspServer = nc.LSPServer
	e.lspURI = nc.LSPFile
	if nc.LSPServer != "" && !nc.Embedded {
		go e.startLSP(nc.LSPServer, nc.LSPWorkspace) // Acquire 新语言的共享 server
	}
}

// requestDefinition 在光标处请求转到定义（异步，勿卡 UI）；拿到目标 → 经 pendingDef 转 UI 线程回调宿主。
func (e *CodeEditorElement) requestDefinition() {
	if !e.lspReady || e.ed == nil || e.ed.OnGoToDefinition == nil {
		return
	}
	cl := e.lspClient
	uri := e.lspURI
	lineOff := e.lspLineOff
	line := e.cursor.line + lineOff // 内嵌体编辑器：本地 → 完整文档坐标
	char := e.cursor.col + e.lspColOff
	go func() {
		locs, err := cl.Definition(uri, line, char)
		if err != nil || len(locs) == 0 {
			return
		}
		file := uriToPath(locs[0].URI)
		if file == "" {
			return
		}
		e.lspMu.Lock()
		e.pendingDef = &pendingJump{file: file, line: locs[0].Range.Start.Line - lineOff + 1, col: locs[0].Range.Start.Character}
		e.lspMu.Unlock()
		repaint()
	}()
}

// requestReferences 光标处查找引用（异步），结果经 pendingRefs 转 UI 线程回调宿主。
func (e *CodeEditorElement) requestReferences() {
	if !e.lspReady || e.ed == nil || e.ed.OnReferences == nil {
		return
	}
	cl := e.lspClient
	uri := e.lspURI
	line := e.cursor.line + e.lspLineOff
	char := e.cursor.col + e.lspColOff
	go func() {
		locs, err := cl.References(uri, line, char, true)
		if err != nil || len(locs) == 0 {
			return
		}
		out := make([]CodeLoc, 0, len(locs))
		for _, l := range locs {
			if f := uriToPath(l.URI); f != "" { // 引用是各文件绝对位置，不减偏移
				out = append(out, CodeLoc{File: f, Line: l.Range.Start.Line + 1, Col: l.Range.Start.Character})
			}
		}
		if len(out) == 0 {
			return
		}
		e.lspMu.Lock()
		e.pendingRefs = out
		e.lspMu.Unlock()
		repaint()
	}()
}

// requestDocumentSymbol 请求当前文件符号大纲（异步），结果（扁平化）经 pendingSyms 转 UI 线程回调宿主。
func (e *CodeEditorElement) requestDocumentSymbol() {
	if !e.lspReady || e.ed == nil || e.ed.OnDocumentSymbols == nil {
		return
	}
	cl := e.lspClient
	uri := e.lspURI
	go func() {
		syms, err := cl.DocumentSymbol(uri)
		if err != nil || len(syms) == 0 {
			return
		}
		out := flattenSyms(syms, 0, nil)
		if len(out) == 0 {
			return
		}
		e.lspMu.Lock()
		e.pendingSyms = out
		e.lspMu.Unlock()
		repaint()
	}()
}

// requestHover 光标处请求悬停信息（异步），结果经 pendingHover 转 UI 线程画浮层。
func (e *CodeEditorElement) requestHover() {
	e.requestHoverAt(e.cursor)
}

// requestHoverAt 在指定位置请求悬停信息（异步），结果经 pendingHover 转 UI 线程画浮层。
// 供鼠标悬停（200ms 延迟）和命令触发两种入口使用。
func (e *CodeEditorElement) requestHoverAt(at cePos) {
	if !e.lspReady {
		return
	}
	cl := e.lspClient
	uri := e.lspURI
	line := at.line + e.lspLineOff
	char := at.col + e.lspColOff
	go func() {
		h, err := cl.HoverAt(uri, line, char)
		if err != nil || h == nil || h.Contents.Value == "" {
			return
		}
		e.lspMu.Lock()
		e.pendingHover = &hoverResult{text: cleanHoverMarkdown(h.Contents.Value), at: at}
		e.lspMu.Unlock()
		repaint()
	}()
}

// formatDocument 请求 LSP 格式化整个文档并应用编辑（异步：goroutine 请求 → pendingEdits → UI 线程消费）。
func (e *CodeEditorElement) formatDocument() {
	if !e.lspReady || e.lspClient == nil {
		return
	}
	cl := e.lspClient
	uri := e.lspURI
	tabSize := 4
	insertSpaces := true
	go func() {
		edits, err := cl.Formatting(uri, tabSize, insertSpaces)
		if err != nil || len(edits) == 0 {
			return
		}
		e.lspMu.Lock()
		e.pendingFormats = edits
		e.lspMu.Unlock()
		repaint()
	}()
}

// applyTextEdits 应用 LSP TextEdit 列表到编辑器缓冲区（按行号倒序→不倒序，保证编辑稳定性）。
func (e *CodeEditorElement) applyTextEdits(edits []lsp.TextEdit) {
	if len(edits) == 0 {
		return
	}
	// 记录撤销状态（格式化前快照）
	e.recordUndo("format")
	// 先排序：按起始行号降序（从后往前应用，避免行号偏移）
	for i := 0; i < len(edits); i++ {
		for j := i + 1; j < len(edits); j++ {
			ei, ej := edits[i].Range.Start, edits[j].Range.Start
			if ei.Line < ej.Line || (ei.Line == ej.Line && ei.Character < ej.Character) {
				edits[i], edits[j] = edits[j], edits[i]
			}
		}
	}
	for _, edit := range edits {
		e.applyOneTextEdit(edit)
	}
	e.afterEdit()
}

// applyOneTextEdit 应用单个 TextEdit。
func (e *CodeEditorElement) applyOneTextEdit(edit lsp.TextEdit) {
	start := edit.Range.Start
	end := edit.Range.End
	// 内嵌体编辑器：完整文档坐标 → 本地坐标
	startLine := start.Line - e.lspLineOff
	endLine := end.Line - e.lspLineOff
	startCol := start.Character - e.lspColOff
	endCol := end.Character - e.lspColOff
	// 越界保护
	if startLine < 0 {
		startLine = 0
		startCol = 0
	}
	if startLine >= len(e.lines) {
		return
	}
	if endLine < 0 {
		return
	}
	if endLine >= len(e.lines) {
		endLine = len(e.lines) - 1
		endCol = len([]rune(e.lines[endLine]))
	}
	// 计算新文本的行
	newText := edit.NewText
	newLines := strings.Split(newText, "\n")
	if len(newLines) == 0 {
		newLines = []string{""}
	}
	// 计算替换前的文本
	startRunes := []rune(e.lines[startLine])
	endRunes := []rune(e.lines[endLine])
	if startCol > len(startRunes) {
		startCol = len(startRunes)
	}
	if endCol > len(endRunes) {
		endCol = len(endRunes)
	}
	// 构造新行
	if startLine == endLine {
		// 单行替换
		prefix := string(startRunes[:startCol])
		suffix := string(startRunes[endCol:])
		// newText 可能含换行
		firstNewLine := newLines[0]
		if len(newLines) == 1 {
			e.lines[startLine] = prefix + firstNewLine + suffix
		} else {
			// 多行：第一行替换后半段，中间行插入，最后行前缀
			lastNewLine := newLines[len(newLines)-1]
			e.lines[startLine] = prefix + firstNewLine
			// 插入中间行（在 startLine+1 位置插入 newLines[1..len-1]）
			var after []string
			after = append(after, e.lines[:startLine+1]...)
			for i := 1; i < len(newLines)-1; i++ {
				after = append(after, newLines[i])
			}
			after = append(after, lastNewLine+suffix)
			after = append(after, e.lines[startLine+1:]...)
			e.lines = after
		}
	} else {
		// 多行替换
		prefix := string(startRunes[:startCol])
		suffix := string(endRunes[endCol:])
		firstNewLine := newLines[0]
		lastNewLine := newLines[len(newLines)-1]
		// 构造：前缀 + 新文本 + 后缀
		var result []string
		result = append(result, e.lines[:startLine]...)
		result = append(result, prefix+firstNewLine)
		for i := 1; i < len(newLines)-1; i++ {
			result = append(result, newLines[i])
		}
		result = append(result, lastNewLine+suffix)
		result = append(result, e.lines[endLine+1:]...)
		e.lines = result
	}
}

// cleanText 过滤不可渲染字符：控制字符（保留 \n）、零宽字符、BOM 等 → 替换/删除。
// 确保所有经过此函数的文本在 Skia/Canvas 渲染时不会因字体缺少字形而显示豆腐块（□/?）。
func cleanText(s string) string {
	runes := make([]rune, 0, len([]rune(s)))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			// 保留换行和制表符（制表符由调用方通过 expandTabs 处理）
			runes = append(runes, r)
		case r < 32: // 其他控制字符（0x00-0x1F 除了 \n\t）
			// 替换为空格而不是直接删除，避免单词意外粘连
			runes = append(runes, ' ')
		case r == 0x7F: // DEL
			// 直接删除
		case r == 0xA0: // 不换行空格 → 普通空格
			runes = append(runes, ' ')
		case r == 0xAD: // 软连字符 → 删除
		case r >= 0x200B && r <= 0x200F: // 零宽空格/连接符/左右方向标记 → 删除
		case r == 0x2028 || r == 0x2029: // 行/段分隔符 → 换行
			runes = append(runes, '\n')
		case r >= 0x202A && r <= 0x202E: // 双向文本覆盖标记 → 删除
		case r >= 0x2060 && r <= 0x2064: // 单词连接符/不可见运算符 → 删除
		case r == 0xFEFF: // BOM → 删除
		case r >= 0xFFF0 && r <= 0xFFFD: // 特殊用途/替换字符 → 删除（包括 � U+FFFD）
		case unicode.Is(unicode.Cs, r): // 代理项（surrogate, 0xD800-0xDFFF）→ 删除
		default:
			runes = append(runes, r)
		}
	}
	return string(runes)
}

// cleanHoverMarkdown 去 ``` 代码围栏、折叠多余空行 + 过滤不可渲染字符，得可直接逐行画的纯文本。
func cleanHoverMarkdown(md string) string {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	// 展开制表符（gopls 返回的结构体字段前有 \t，字体无 tab 字形会渲染成豆腐块）
	md = expandTabs(md)
	// 过滤不可渲染字符
	md = cleanText(md)
	var out []string
	blank := false
	for _, ln := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(ln), "```") {
			continue
		}
		if strings.TrimSpace(ln) == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
		}
		out = append(out, ln)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// flattenSyms 把嵌套 DocumentSymbol 拍平成带深度的列表（保留层级缩进），行转 1 基。
func flattenSyms(syms []lsp.DocumentSymbol, depth int, out []CodeSym) []CodeSym {
	for _, s := range syms {
		out = append(out, CodeSym{Name: s.Name, Kind: s.Kind, Line: s.SelectionRange.Start.Line + 1, Depth: depth})
		out = flattenSyms(s.Children, depth+1, out)
	}
	return out
}

// uriToPath file:// URI → 本地路径（Windows: file:///C:/x → C:/x；Unix: file:///x → /x）。
func uriToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return ""
	}
	p := strings.ReplaceAll(strings.TrimPrefix(uri, "file://"), "%20", " ")
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' { // /C:/... → C:/...
		p = p[1:]
	}
	return p
}

func convertLSPItems(items []lsp.CompletionItem, prefix string) []compItem {
	lp := strings.ToLower(prefix)
	out := make([]compItem, 0, len(items))
	for _, it := range items {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(it.Label), lp) {
			continue
		}
		out = append(out, compItem{label: cleanText(it.Label), kind: lspKindToCompKind(it.Kind), detail: cleanText(it.Detail)})
	}
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}

func lspKindToCompKind(k int) compKind {
	switch k {
	case lsp.KindKeyword:
		return ckKeyword
	case lsp.KindClass, lsp.KindInterface, lsp.KindStruct, lsp.KindTypeParam, lsp.KindModule:
		return ckType
	default:
		return ckIdent
	}
}

// paintDiagnostics 在诊断范围下画波浪线（红=错误、橙=警告）。在编辑区裁剪内调用。
func (e *CodeEditorElement) paintDiagnostics(cvs canvas.Canvas, left, top float64) {
	e.lspMu.Lock()
	ds := e.diagnostics
	e.lspMu.Unlock()
	for _, d := range ds {
		ln := d.Range.Start.Line - e.lspLineOff // 完整文档坐标 → 本地坐标（内嵌体编辑器）
		if ln < 0 || ln >= len(e.lines) {
			continue // 落在函数体之外的诊断不在本编辑器显示
		}
		startCol := d.Range.Start.Character - e.lspColOff
		endCol := d.Range.End.Character - e.lspColOff
		if startCol < 0 {
			startCol = 0
		}
		if endCol < 0 {
			endCol = 0
		}
		if d.Range.End.Line != d.Range.Start.Line {
			endCol = len(e.lineRunes(ln))
		}
		col := types.ColorFromRGB(0xE5, 0x1C, 0x23) // 红=error
		if d.Severity == lsp.SeverityWarning {
			col = types.ColorFromRGB(0xE6, 0xA2, 0x3C) // 橙=warning
		}
		// 换行时诊断区间可能跨视觉段，逐段画波浪线。
		e.forSegSpans(ln, startCol, endCol, left, top, false, func(rowTopY, x0, x1 float64) {
			if x1 <= x0 {
				x1 = x0 + 6
			}
			e.drawSquiggle(cvs, x0, x1, rowTopY+ceLineH-3, col)
		})
	}
}

func (e *CodeEditorElement) drawSquiggle(cvs canvas.Canvas, x0, x1, y float64, col types.Color) {
	p := paint.DefaultStrokePaint()
	p.Color = col
	p.StrokeWidth = 1
	const step, amp = 2.5, 1.6
	prevX, prevY := x0, y
	up := true
	for x := x0 + step; x <= x1; x += step {
		ny := y + amp
		if up {
			ny = y - amp
		}
		cvs.DrawLine(prevX, prevY, x, ny, p)
		prevX, prevY = x, ny
		up = !up
	}
}

// paintGutterDiagnostics 在行号栏画诊断标记（红/橙圆点）。
func (e *CodeEditorElement) paintGutterDiagnostics(cvs canvas.Canvas, top float64) {
	e.lspMu.Lock()
	ds := e.diagnostics
	e.lspMu.Unlock()
	if len(ds) == 0 {
		return
	}
	// 建一个行→最高严重度映射（每行只画一个标记：优先错误，其次警告/信息）
	lineSeverity := make(map[int]int)
	for _, d := range ds {
		ln := d.Range.Start.Line - e.lspLineOff
		if ln < 0 || ln >= len(e.lines) {
			continue
		}
		existing, ok := lineSeverity[ln]
		if !ok || d.Severity < existing {
			lineSeverity[ln] = d.Severity
		}
	}
	pos := e.Offset()
	// 遍历可见行，画标记
	for vi := 0; vi < len(e.wrapSegs); vi++ {
		sg := e.wrapSegs[vi]
		if sg.start != 0 { // 只在逻辑行首段画
			continue
		}
		ln := sg.line
		sev, ok := lineSeverity[ln]
		if !ok {
			continue
		}
		ly := top + float64(vi)*ceLineH
		cy := ly + ceLineH/2
		cx := pos.X + 8
		col := types.ColorFromRGB(0xE5, 0x1C, 0x23) // 红=error
		if sev >= lsp.SeverityWarning {
			col = types.ColorFromRGB(0xE6, 0xA2, 0x3C) // 橙=warning/info/hint
		}
		dp := paint.DefaultPaint()
		dp.Color = col
		cvs.DrawCircle(cx, cy, 3.5, dp)
	}
}

// diagnosticAtPos 返回 (行,列) 处第一个诊断消息；无诊断时返回 ""。
func (e *CodeEditorElement) diagnosticAtPos(line, col int) string {
	e.lspMu.Lock()
	ds := e.diagnostics
	e.lspMu.Unlock()
	for _, d := range ds {
		ln := d.Range.Start.Line - e.lspLineOff
		if ln != line {
			continue
		}
		startCol := d.Range.Start.Character - e.lspColOff
		endCol := d.Range.End.Character - e.lspColOff
		if d.Range.End.Line != d.Range.Start.Line {
			endCol = len(e.lineRunes(ln))
		}
		if col >= startCol && col < endCol {
			return d.Message
		}
	}
	return ""
}

// paintDiagHover 画诊断悬停提示浮层（鼠标悬停在诊断波浪线上时显示错误/警告消息）。
func (e *CodeEditorElement) paintDiagHover(cvs canvas.Canvas) {
	if e.diagHoverMsg == "" {
		return
	}
	pos := e.Offset()
	// 按 \n 拆分基础行，并做自动换行
	const maxBoxW = 480.0
	const pad = 18.0
	wrapW := maxBoxW - pad
	rawLines := strings.Split(e.diagHoverMsg, "\n")
	var lines []string
	for _, ln := range rawLines {
		wrapped := e.wrapHoverLine(ln, wrapW)
		lines = append(lines, wrapped...)
		if len(lines) > 20 {
			lines = append(lines[:20], "…")
			break
		}
	}
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
	x := e.posX(e.diagHoverLine, e.diagHoverCol, pos.X+e.gutterW+ceTextPad-e.scrollX)
	yTop := e.posTopY(e.diagHoverLine, e.diagHoverCol, pos.Y+4-e.scrollY) - boxH - 4
	if yTop < pos.Y+2 {
		yTop = e.posTopY(e.diagHoverLine, e.diagHoverCol, pos.Y+4-e.scrollY) + ceLineH + 2
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
		canvas.DrawTextAligned(cvs, ln,
			types.Rect{X: x + 8, Y: yTop + 5 + float64(i)*ceLineH, Width: maxW - 14, Height: ceLineH},
			e.font, elTextPrimary(), canvas.HAlignLeft, canvas.VAlignMiddle)
	}
}

// Unmount 关闭 LSP 客户端（停止子进程）。
func (e *CodeEditorElement) Unmount() {
	if e.lspClient != nil {
		e.lspClient.DidClose(e.lspURI) // 只关本文件文档；共享 server 由池常驻（lsp.CloseAll 退出时统一关）
		e.lspClient = nil
		e.lspReady = false
	}
	if focusedCodeEditor == e { // 销毁时清全局引用，避免命令派发到已卸载的编辑器
		focusedCodeEditor = nil
	}
	if lastFocusedCodeEditor == e {
		lastFocusedCodeEditor = nil
	}
}
