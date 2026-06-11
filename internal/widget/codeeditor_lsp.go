package widget

import (
	"strings"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/lsp"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// CodeEditor 的 LSP 集成：异步启动语言服务器、文档同步、语义补全、诊断波浪线。
// 线程模型：读协程(lsp.Client)写 lspPending/diagnostics 加锁，UI 线程下一帧消费（drain）+ 重绘。

// startLSP 异步启动语言服务器、initialize、didOpen（在 goroutine 上跑，~2.5s）。
func (e *CodeEditorElement) startLSP(server, workspace string) {
	e.lspServer = server
	c, err := lsp.Acquire(server, e.ed.LSPArgs, workspace) // 共享池：复用现有或首次起（非单例，多语言并存）
	if err != nil || c == nil {
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
	if !e.lspReady {
		return
	}
	cl := e.lspClient
	uri := e.lspURI
	at := e.cursor
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

// cleanHoverMarkdown 去 ``` 代码围栏、折叠多余空行，得可直接逐行画的纯文本。
func cleanHoverMarkdown(md string) string {
	md = strings.ReplaceAll(md, "\r\n", "\n")
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
		out = append(out, compItem{label: it.Label, kind: lspKindToCompKind(it.Kind), detail: it.Detail})
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
