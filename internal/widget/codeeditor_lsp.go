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
	c, err := lsp.NewClient(server)
	if err != nil {
		return
	}
	c.OnDiagnostics = func(uri string, ds []lsp.Diagnostic) {
		if uri != e.lspURI {
			return
		}
		e.lspMu.Lock()
		e.diagnostics = ds
		e.lspMu.Unlock()
		repaint()
	}
	if err := c.Initialize(workspace); err != nil {
		c.Close()
		return
	}
	c.DidOpen(e.lspURI, e.ed.Language, e.lspDoc())
	e.lspClient = c
	e.lspReady = true
	repaint()
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
	e.lspMu.Unlock()
	if pend != nil && e.completing && gen == e.lspGen {
		e.compItems = pend
		if e.compSel >= len(e.compItems) {
			e.compSel = 0
		}
		e.compScroll = 0
	}
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
		e.lspClient.Close()
		e.lspClient = nil
		e.lspReady = false
	}
}
