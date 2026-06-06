// Package lsp 是一个极简 LSP（Language Server Protocol）客户端，用于接 gopls/tsserver
// 等语言服务器，为编辑器提供语义补全、诊断、悬停、跳转。纯标准库实现，无 CGO。
package lsp

// Position LSP 位置（行、列均 0 基，列按 UTF-16 码元——ASCII/常见情形等同 rune）。
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range 区间。
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// 补全项种类（LSP CompletionItemKind 子集，用于图标）。
const (
	KindText        = 1
	KindMethod      = 2
	KindFunction    = 3
	KindConstructor = 4
	KindField       = 5
	KindVariable    = 6
	KindClass       = 7
	KindInterface   = 8
	KindModule      = 9
	KindProperty    = 10
	KindKeyword     = 14
	KindStruct      = 22
	KindTypeParam   = 27
)

// CompletionItem 一个补全候选。
type CompletionItem struct {
	Label      string `json:"label"`
	Kind       int    `json:"kind"`
	Detail     string `json:"detail"`
	InsertText string `json:"insertText"`
}

// CompletionList 补全结果（也可能直接是 []CompletionItem）。
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// 诊断严重级别。
const (
	SeverityError   = 1
	SeverityWarning = 2
	SeverityInfo    = 3
	SeverityHint    = 4
)

// Diagnostic 一条诊断（错误/警告）。
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source"`
}

// PublishDiagnosticsParams 服务器推送的诊断通知参数。
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Hover 悬停信息。
type Hover struct {
	Contents MarkupContent `json:"contents"`
}
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}
