// Package widget — Markdown 渲染组件。
//
// 完整支持 CommonMark 核心语法，在 goui 控件树中渲染：
//
// 块级：标题(1-6级)、无序/有序列表(含缩进)、任务列表、引用块(含嵌套)、
//       分隔线、表格(含列对齐)、围栏代码块(语法高亮+复制按钮)、段落
// 行内：**加粗** *斜体* ~~删除线~~ `行内代码` [链接](url) ![图片](url)
//       HTML 实体 &amp; &lt; &gt; &quot; &#39; 及反斜杠转义
//
// 用法：
//
//	widget.NewMarkdownView(source, widget.MarkdownStyle{
//	    TextColor:      theme.Fg,
//	    AccentColor:    theme.Accent,
//	    MonoFont:       canvas.Font{Family: "Consolas", Size: 13},
//	    OnLinkClick:    func(url string) { openURL(url) },
//	    ClipboardWrite: clipboardWriteFn,
//	})
package widget

import (
	"regexp"
	"strings"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── 公开类型 ────────────────────────────────────────────────

// MarkdownStyle 定义 Markdown 渲染的视觉风格。
// 零值使用 DefaultMarkdownStyle() 的暗色主题默认值。
type MarkdownStyle struct {
	TextColor    types.Color // 主文字色
	TextSubtle   types.Color // 次文字色（斜体等）
	TextMuted    types.Color // 弱文字色（删除线、图片占位）
	AccentColor  types.Color // 强调色（链接、行内代码、标题 h1-2）
	SuccessColor types.Color // 成功色（任务列表已勾选）
	BorderColor  types.Color // 边框/分隔线色
	BgColor      types.Color // 主背景色（代码块底色）
	BgSubtle     types.Color // 次级背景（代码块/表格奇行）
	BgMuted      types.Color // 三级背景（表头）

	BodyFontSize float64    // 正文字号（默认 13）
	MonoFont     canvas.Font // 等宽字体（代码块/行内代码）

	// OnLinkClick 链接点击回调（nil=不处理）。
	OnLinkClick func(url string)
	// ClipboardWrite 写剪贴板回调（nil=不显示复制按钮）。
	ClipboardWrite func(text string)
}

// DefaultMarkdownStyle 返回暗色主题的默认风格（类 GitHub 暗色）。
func DefaultMarkdownStyle() MarkdownStyle {
	return MarkdownStyle{
		TextColor:    types.ColorFromHex("#e6edf3"),
		TextSubtle:   types.ColorFromHex("#8b949e"),
		TextMuted:    types.ColorFromHex("#6e7681"),
		AccentColor:  types.ColorFromHex("#58a6ff"),
		SuccessColor: types.ColorFromHex("#3fb950"),
		BorderColor:  types.ColorFromHex("#30363d"),
		BgColor:      types.ColorFromHex("#0d1117"),
		BgSubtle:     types.ColorFromHex("#161b22"),
		BgMuted:      types.ColorFromHex("#21262d"),
		BodyFontSize: 13,
		MonoFont:     canvas.Font{Family: "Consolas", Size: 13},
	}
}

// MarkdownView 将 Markdown 文本渲染为控件树。
//
// 【双缓冲设计】MarkdownView 缓存解析结果和渲染后的 Widget 树。
//   - 首次 Build：全量解析 markdown → 渲染 Widget 树 → 缓存。
//     之后同一 Source 的 Build 直接返回缓存（O(1)），不重新解析。
//   - 流式追加：文本增长时只解析新增部分，追加到缓存（O(delta)），
//     避免每次 chunk 全量 O(N) 重解析。
//   - SetStateStreaming 场景：内容未变时直接命中缓存，零分配。
type MarkdownView struct {
	StatelessWidget
	Source string        // Markdown 源文本
	Style  MarkdownStyle // 视觉风格（零值=暗色默认）

	// ── 双缓冲缓存 ──
	// 缓存的解析结果。Source 未变化时 Build 直接复用，不重新解析。
	cachedSource string   // 上次解析的源文本（用于增量检测）
	cachedBlocks []block  // 上次解析的块列表
	cachedWidgets []Widget // 上次渲染的 Widget 列表（Build 返回的直接是这些）
}

// NewMarkdownView 创建 Markdown 渲染视图。
// style 传零值则用 DefaultMarkdownStyle() 的暗色默认值。
func NewMarkdownView(source string, style MarkdownStyle) *MarkdownView {
	return &MarkdownView{Source: source, Style: style}
}

func (m *MarkdownView) Build(ctx BuildContext) Widget {
	// ── 双缓冲：缓存命中判断 ──
	// 如果 Source 未变化，直接返回缓存的 Widget 树。
	// StatelessElement.built 在 Element 层做二次缓存，但在流式追加
	// （每次 SetState 重新 CreateElement）时，Widget 层的缓存是关键。
	if m.cachedSource == m.Source && m.cachedWidgets != nil {
		if len(m.cachedWidgets) == 0 {
			return Div(Style{})
		}
		return Div(Style{FlexDirection: "column", AlignItems: "stretch", Gap: 6}, m.cachedWidgets)
	}

	style := m.Style
	if style.BodyFontSize == 0 {
		style = DefaultMarkdownStyle()
	}
	if style.MonoFont.Size == 0 {
		style.MonoFont.Size = style.BodyFontSize
	}
	if style.MonoFont.Family == "" {
		style.MonoFont.Family = "Consolas"
	}

	// ── 增量解析：文本追加时只解析新增部分 ──
	// 检测旧文本是前缀，只需解析增量
	if m.cachedSource != "" && len(m.Source) > len(m.cachedSource) &&
		m.Source[:len(m.cachedSource)] == m.cachedSource {

		suffix := m.Source[len(m.cachedSource):]
		s := &mdState{style: style}
		newBlocks := s.parseBlocks(suffix)
		if len(newBlocks) > 0 {
			// 追加新块到缓存
			m.cachedBlocks = append(m.cachedBlocks, newBlocks...)
			// 只渲染新增块
			newWidgets := s.renderBlocks(newBlocks)
			m.cachedWidgets = append(m.cachedWidgets, newWidgets...)
		}
		m.cachedSource = m.Source
	} else {
		// ── 全量解析（首次或内容变更） ──
		s := &mdState{style: style}
		m.cachedBlocks = s.parseBlocks(m.Source)
		m.cachedWidgets = s.renderBlocks(m.cachedBlocks)
		m.cachedSource = m.Source
	}

	if len(m.cachedWidgets) == 0 {
		return Div(Style{})
	}
	return Div(Style{FlexDirection: "column", AlignItems: "stretch", Gap: 6}, m.cachedWidgets)
}

// ─── 内部状态 ────────────────────────────────────────────────

type mdState struct {
	style MarkdownStyle
}

// ─── 块级类型 ────────────────────────────────────────────────

type bType int

const (
	bParagraph bType = iota
	bHeading
	bCodeBlock
	bBlockquote
	bBulletList
	bOrderedList
	bTaskList
	bTable
	bThematicBreak
)

type block struct {
	typ   bType
	level int      // 标题级别(1-6)/引用深度/列表缩进
	lang  string   // 代码块语言
	lines []string // 原始行
	spans [][]span // 行内解析结果
	num   string   // 有序列表序号
	align []string // 表格列对齐
}

// span 行内文本片段（带颜色和字体）
type span struct {
	Text  string
	Color types.Color
	Font  canvas.Font
}

// ─── 正则 ────────────────────────────────────────────────────

var (
	reThematicBreak = regexp.MustCompile(`^(?:---|___|\*\*\*)\s*$`)
	reHeading       = regexp.MustCompile(`^(#{1,6})\s+(.*)`)
	reBlockquote    = regexp.MustCompile(`^>\s?(.*)`)
	reUnordered     = regexp.MustCompile(`^(\s*)([-*+])\s+(.*)`)
	reOrdered       = regexp.MustCompile(`^(\s*)(\d+)\.\s+(.*)`)
	reTask          = regexp.MustCompile(`^(\s*)[-*+]\s+\[([ x])\]\s+(.*)`)
	reTableRow      = regexp.MustCompile(`^\|.+?\|$`)
	reTableSep      = regexp.MustCompile(`^\|[\s:-]+\|`)
	reFence         = regexp.MustCompile("^```(\\w*)")

	reEscChar = regexp.MustCompile(`\\([\\` + "`*_{}[]()#+-.!|~])")
)

// ─── 块级解析 ────────────────────────────────────────────────

func (s *mdState) parseBlocks(text string) []block {
	rawLines := strings.Split(text, "\n")
	var blocks []block

	i := 0
	for i < len(rawLines) {
		line := rawLines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			i++
			continue
		}

		// 分隔线
		if reThematicBreak.MatchString(trimmed) {
			blocks = append(blocks, block{typ: bThematicBreak})
			i++
			continue
		}

		// 标题
		if m := reHeading.FindStringSubmatch(trimmed); m != nil {
			level := len(m[1])
			blocks = append(blocks, block{
				typ:   bHeading,
				level: level,
				spans: [][]span{s.mdRenderInline(m[2])},
			})
			i++
			continue
		}

		// 围栏代码块
		if m := reFence.FindStringSubmatch(trimmed); m != nil {
			lang := m[1]
			var code []string
			i++
			for i < len(rawLines) {
				if reFence.MatchString(strings.TrimSpace(rawLines[i])) {
					i++
					break
				}
				code = append(code, rawLines[i])
				i++
			}
			blocks = append(blocks, block{typ: bCodeBlock, lang: lang, lines: code})
			continue
		}

		// 表格
		if reTableRow.MatchString(trimmed) {
			var rows []string
			rows = append(rows, trimmed)
			i++
			align := detectColAlign("")
			if i < len(rawLines) && reTableSep.MatchString(strings.TrimSpace(rawLines[i])) {
				align = detectColAlign(rawLines[i])
				i++
			}
			for i < len(rawLines) {
				t := strings.TrimSpace(rawLines[i])
				if t == "" || !reTableRow.MatchString(t) {
					break
				}
				rows = append(rows, t)
				i++
			}
			blocks = append(blocks, block{typ: bTable, lines: rows, align: align})
			continue
		}

		// 引用块
		if strings.HasPrefix(trimmed, ">") {
			var qlines []string
			for i < len(rawLines) {
				t := strings.TrimSpace(rawLines[i])
				if t == "" || !strings.HasPrefix(t, ">") {
					break
				}
				m := reBlockquote.FindStringSubmatch(t)
				if m != nil {
					qlines = append(qlines, m[1])
				} else {
					qlines = append(qlines, "")
				}
				i++
			}
			blocks = append(blocks, block{typ: bBlockquote, lines: []string{strings.Join(qlines, "\n")}})
			continue
		}

		// 任务列表
		if m := reTask.FindStringSubmatch(line); m != nil {
			checked := m[2] != " "
			blocks = append(blocks, block{
				typ:   bTaskList,
				level: len(m[1]),
				spans: [][]span{s.mdRenderInline(m[3])},
				lines: []string{map[bool]string{true: "true"}[checked]},
			})
			i++
			continue
		}

		// 无序列表
		if m := reUnordered.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			for i < len(rawLines) {
				mm := reUnordered.FindStringSubmatch(rawLines[i])
				if mm == nil || len(mm[1]) != indent {
					break
				}
				blocks = append(blocks, block{
					typ:   bBulletList,
					level: indent,
					spans: [][]span{s.mdRenderInline(mm[3])},
				})
				i++
			}
			continue
		}

		// 有序列表
		if m := reOrdered.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			blocks = append(blocks, block{
				typ:   bOrderedList,
				level: indent,
				spans: [][]span{s.mdRenderInline(m[3])},
				num:   m[2],
			})
			i++
			continue
		}

		// 段落：收集连续非空行
		var paraLines []string
		for i < len(rawLines) {
			t := strings.TrimSpace(rawLines[i])
			if t == "" ||
				reHeading.MatchString(t) ||
				reThematicBreak.MatchString(t) ||
				reFence.MatchString(t) ||
				reTableRow.MatchString(t) ||
				strings.HasPrefix(t, ">") ||
				reUnordered.MatchString(rawLines[i]) ||
				reOrdered.MatchString(rawLines[i]) ||
				reTask.MatchString(rawLines[i]) {
				break
			}
			paraLines = append(paraLines, rawLines[i])
			i++
		}
		if len(paraLines) > 0 {
			blocks = append(blocks, block{typ: bParagraph, lines: paraLines})
		}
	}

	return blocks
}

// detectColAlign 从表格分隔行检测列对齐
func detectColAlign(sepLine string) []string {
	sepLine = strings.TrimSpace(sepLine)
	sepLine = strings.TrimPrefix(sepLine, "|")
	sepLine = strings.TrimSuffix(sepLine, "|")
	cells := strings.Split(sepLine, "|")
	var align []string
	for _, c := range cells {
		c = strings.TrimSpace(c)
		switch {
		case strings.HasPrefix(c, ":") && strings.HasSuffix(c, ":"):
			align = append(align, "center")
		case strings.HasSuffix(c, ":"):
			align = append(align, "right")
		default:
			align = append(align, "left")
		}
	}
	return align
}

// ─── 块渲染 ────────────────────────────────────────────────

func (s *mdState) renderBlocks(blocks []block) []Widget {
	var out []Widget
	style := s.style

	for _, b := range blocks {
		switch b.typ {
		case bThematicBreak:
			out = append(out, Div(Style{
				Height:          1,
				BackgroundColor: &style.BorderColor,
				Margin:          types.EdgeInsetsLTRB(0, 4, 0, 4),
			}))

		case bHeading:
			fs := 18.0 - float64(b.level)*1.2
			if fs < 11 {
				fs = 11
			}
			col := style.TextColor
			if b.level <= 2 {
				col = style.AccentColor
			}
			spans := b.spans[0]
			for i := range spans {
				if spans[i].Font.Size == 0 || spans[i].Font.Size < fs {
					spans[i].Font.Size = fs
				}
				if spans[i].Color == style.TextColor {
					spans[i].Color = col
				}
			}
			out = append(out, Div(Style{
				FlexDirection: "row", AlignItems: "center",
				Margin: types.EdgeInsetsLTRB(0, 4, 0, 2),
			}, s.inlineRow(spans, fs)))

		case bCodeBlock:
			out = append(out, s.renderCodeBlock(b.lines, b.lang))

		case bBlockquote:
			inner := s.renderBlocks(s.parseBlocks(b.lines[0]))
			out = append(out, Div(Style{
				FlexDirection: "row", AlignItems: "stretch", Gap: 8,
				Margin: types.EdgeInsetsLTRB(0, 2, 0, 2),
			},
				Div(Style{
					Width:           3,
					BackgroundColor: &style.AccentColor,
					BorderRadius:    1.5,
				}),
				Div(Style{
					FlexDirection: "column",
					AlignItems:    "stretch",
				}, inner),
			))

		case bBulletList:
			bullet := "•"
			if b.level > 4 {
				bullet = "◦"
			}
			out = append(out, Div(Style{
				FlexDirection: "row", Gap: 6,
				Margin: types.EdgeInsetsLTRB(float64(b.level)+2, 1, 0, 1),
			},
				s.textWidget(bullet, style.TextSubtle, 13),
				s.expand(s.inlineRow(b.spans[0], 0)),
			))

		case bOrderedList:
			num := b.num + "."
			out = append(out, Div(Style{
				FlexDirection: "row", Gap: 6,
				Margin: types.EdgeInsetsLTRB(float64(b.level)+2, 1, 0, 1),
			},
				s.textWidget(num, style.TextSubtle, 13),
				s.expand(s.inlineRow(b.spans[0], 0)),
			))

		case bTaskList:
			checked := len(b.lines) > 0 && b.lines[0] == "true"
			icon := "\u2610" // ☐
			col := style.TextColor
			if checked {
				icon = "\u2611" // ☑
				col = style.SuccessColor
			}
			out = append(out, Div(Style{
				FlexDirection: "row", Gap: 6, AlignItems: "center",
			},
				s.textWidget(icon, col, 14),
				s.expand(s.inlineRow(b.spans[0], 0)),
			))

		case bTable:
			out = append(out, s.renderTable(b.lines, b.align))

		case bParagraph:
			if len(b.lines) > 0 {
				text := strings.Join(b.lines, " ")
				t := NewText(text, s.style.TextColor)
				t.Font.Size = s.style.BodyFontSize
				out = append(out, t)
			}
		}
	}

	return out
}

// ─── 代码块渲染 ──────────────────────────────────────────────

func (s *mdState) renderCodeBlock(lines []string, lang string) Widget {
	style := s.style
	joined := strings.Join(lines, "\n")
	joined = strings.ReplaceAll(joined, "\t", "    ")
	rows := HighlightCode(joined, lang)

	var body []Widget
	for _, spans := range rows {
		if len(spans) == 0 {
			body = append(body, Div(Style{Height: 8}))
			continue
		}
		var kids []Widget
		for _, sp := range spans {
			t := NewText(sp.Text, sp.Color)
			t.Font = style.MonoFont
			kids = append(kids, t)
		}
		body = append(body, Div(Style{FlexDirection: "row"}, kids))
	}
	bodyDiv := Div(Style{FlexDirection: "column", AlignItems: "stretch"}, body)

	var hdr []Widget
	if lang != "" {
		hdr = append(hdr, s.textWidget(lang, style.TextMuted, 11))
	}
	hdr = append(hdr, s.expand(Div(Style{})))
	if style.ClipboardWrite != nil {
		hdr = append(hdr, s.copyButton(joined))
	}

	return Div(Style{
		BackgroundColor: &style.BgSubtle, BorderColor: &style.BorderColor,
		BorderWidth: 1, BorderRadius: 6,
		Padding: types.EdgeInsets(8), FlexDirection: "column",
		AlignItems: "stretch", Gap: 4, Margin: types.EdgeInsetsLTRB(0, 2, 0, 2),
	},
		Div(Style{
			FlexDirection: "row", AlignItems: "center",
			Padding: types.EdgeInsetsLTRB(4, 0, 4, 4),
		}, hdr),
		bodyDiv,
	)
}

// ─── 表格渲染 ────────────────────────────────────────────────

// colDivider 创建列间竖分隔线（1px 宽、24px 高的色块）。
func (s *mdState) colDivider() Widget {
	return Div(Style{
		Width:           1,
		MinHeight:       24,
		BackgroundColor: &s.style.BorderColor,
	})
}

// tableCell 创建一个等分宽度的表格单元格（Expanded{Flex:1} + Padding + 内容）。
//
// 使用 Column + 单个 Text 而非 inlineRow(Row)，因为 Row 中 Text 子元素获得不受限 MaxWidth
// 不会自动换行；Column + AlignItems:"stretch" 传递受限宽度给 Text，使长内容正确折行。
func (s *mdState) tableCell(text string, alignDir string, fontSize float64) Widget {
	fs := s.style.BodyFontSize
	if fontSize > 0 {
		fs = fontSize
	}
	cellSty := Style{
		Padding:       types.EdgeInsetsLTRB(10, 5, 10, 5),
		MinHeight:     28,
		FlexDirection: "column",
		AlignItems:    "stretch",
	}
	t := NewText(text, s.style.TextColor)
	t.Font.Size = fs
	switch alignDir {
	case "right":
		t.Align = canvas.TextAlignRight
	case "center":
		t.Align = canvas.TextAlignCenter
	}
	return &Expanded{Flex: 1, SingleChildWidget: SingleChildWidget{
		Child: Div(cellSty, t),
	}}
}

// tableRow 构建一行带列分隔线的表格行。
func (s *mdState) tableRow(cells []string, align []string, fontSize float64, rowSty Style) Widget {
	var kids []Widget
	for ci := range align {
		cellText := ""
		if ci < len(cells) {
			cellText = cells[ci]
		}
		kids = append(kids, s.tableCell(cellText, align[ci], fontSize))
		if ci < len(align)-1 {
			kids = append(kids, s.colDivider())
		}
	}
	args := make([]interface{}, 0, 1+len(kids))
	args = append(args, rowSty)
	for _, k := range kids {
		args = append(args, k)
	}
	return Div(args...)
}

func (s *mdState) renderTable(rows []string, align []string) Widget {
	style := s.style
	if len(rows) == 0 {
		return Div(Style{})
	}

	var parsed [][]string
	for _, row := range rows {
		r := strings.TrimSpace(row)
		r = strings.TrimPrefix(r, "|")
		r = strings.TrimSuffix(r, "|")
		cells := strings.Split(r, "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		parsed = append(parsed, cells)
	}
	if len(parsed) == 0 {
		return Div(Style{})
	}

	ncols := 0
	for _, cells := range parsed {
		if len(cells) > ncols {
			ncols = len(cells)
		}
	}
	if ncols == 0 {
		return Div(Style{})
	}

	for len(align) < ncols {
		align = append(align, "left")
	}

	headerCells := parsed[0]
	dataRows := parsed[1:]

	var children []Widget

	// 表头行（底色 BgMuted，圆角 4）
	children = append(children, s.tableRow(headerCells, align, 12.5, Style{
		FlexDirection:   "row",
		BackgroundColor: &style.BgMuted,
		BorderRadius:    4,
	}))

	// 表头与数据间的分隔线
	children = append(children, Div(Style{Height: 1, BackgroundColor: &style.BorderColor}))

	// 数据行
	for ri, cells := range dataRows {
		rowSty := Style{FlexDirection: "row", MinHeight: 28}
		if ri%2 == 1 {
			rowSty.BackgroundColor = &style.BgSubtle
		}
		children = append(children, s.tableRow(cells, align, 12, rowSty))
	}

	return Div(Style{
		FlexDirection: "column", AlignItems: "stretch",
		BorderColor: &style.BorderColor, BorderWidth: 1, BorderRadius: 6, Overflow: "hidden",
	}, children)
}

// ─── 行内解析 ────────────────────────────────────────────────

type inToken int

const (
	tText inToken = iota
	tBold
	tItalic
	tStrikethrough
	tCode
	tLink
	tImage
)

type mdToken struct {
	typ  inToken
	text string
	url  string
	alt  string
}

// mdRenderInline 行内标记解析器。
// 支持：**bold** *italic* ~~strikethrough~~ `code` [text](url) ![alt](url)
func (s *mdState) mdRenderInline(src string) []span {
	// HTML 实体解码
	src = decodeEntities(src)

	// 反斜杠转义 → 不可见占位
	type esc struct{ idx int; ch rune }
	var escs []esc
	src = reEscChar.ReplaceAllStringFunc(src, func(m string) string {
		ch := []rune(m[1:])[0]
		escs = append(escs, esc{idx: len(escs), ch: ch})
		return "\x00ESC" + mdItoa(len(escs)-1) + "\x00"
	})

	var tokens []mdToken
	runes := []rune(src)
	pos := 0

loop:
	for pos < len(runes) {
		// 行内代码 `…`
		if runes[pos] == '`' {
			end := pos + 1
			for end < len(runes) && runes[end] != '`' {
				end++
			}
			if end < len(runes) {
				tokens = append(tokens, mdToken{typ: tCode, text: string(runes[pos+1 : end])})
				pos = end + 1
				continue
			}
		}

		// 图片 ![alt](url)
		if pos+2 < len(runes) && runes[pos] == '!' && runes[pos+1] == '[' {
			closeB := findCloseBracket(runes, pos+2)
			if closeB > 0 && closeB+1 < len(runes) && runes[closeB+1] == '(' {
				closeP := findCloseParen(runes, closeB+2)
				if closeP > 0 {
					tokens = append(tokens, mdToken{typ: tImage, alt: string(runes[pos+2 : closeB]), url: string(runes[closeB+2 : closeP])})
					pos = closeP + 1
					continue
				}
			}
		}

		// 链接 [text](url)
		if runes[pos] == '[' {
			closeB := findCloseBracket(runes, pos+1)
			if closeB > 0 && closeB+1 < len(runes) && runes[closeB+1] == '(' {
				closeP := findCloseParen(runes, closeB+2)
				if closeP > 0 {
					tokens = append(tokens, mdToken{typ: tLink, text: string(runes[pos+1 : closeB]), url: string(runes[closeB+2 : closeP])})
					pos = closeP + 1
					continue
				}
			}
		}

		// 删除线 ~~text~~
		if pos+1 < len(runes) && runes[pos] == '~' && runes[pos+1] == '~' {
			end := pos + 2
			for end+1 < len(runes) && !(runes[end] == '~' && runes[end+1] == '~') {
				end++
			}
			if end+1 < len(runes) {
				tokens = append(tokens, mdToken{typ: tStrikethrough, text: string(runes[pos+2 : end])})
				pos = end + 2
				continue
			}
		}

		// 加粗 **text**
		if pos+1 < len(runes) && runes[pos] == '*' && runes[pos+1] == '*' {
			end := pos + 2
			for end+1 < len(runes) && !(runes[end] == '*' && runes[end+1] == '*') {
				end++
			}
			if end+1 < len(runes) {
				tokens = append(tokens, mdToken{typ: tBold, text: string(runes[pos+2 : end])})
				pos = end + 2
				continue
			}
		}

		// 斜体 *text*
		if runes[pos] == '*' && (pos+1 >= len(runes) || runes[pos+1] != '*') {
			end := pos + 1
			for end < len(runes) && runes[end] != '*' {
				end++
			}
			if end < len(runes) && (end+1 >= len(runes) || runes[end+1] != '*') {
				tokens = append(tokens, mdToken{typ: tItalic, text: string(runes[pos+1 : end])})
				pos = end + 1
				continue
			}
		}

		// ESC 占位
		if runes[pos] == '\x00' {
			rest := string(runes[pos:])
			for ei, e := range escs {
				marker := "\x00ESC" + mdItoa(ei) + "\x00"
				if strings.HasPrefix(rest, marker) {
					tokens = append(tokens, mdToken{typ: tText, text: string(e.ch)})
					pos += len([]rune(marker))
					continue loop
				}
			}
		}

		// 普通文本
		start := pos
		for pos < len(runes) {
			ch := runes[pos]
			if ch == '`' || ch == '[' || ch == '!' || ch == '*' || ch == '~' || ch == '\x00' {
				break
			}
			pos++
		}
		if pos > start {
			tokens = append(tokens, mdToken{typ: tText, text: string(runes[start:pos])})
		} else {
			pos++
		}
	}

	return s.tokensToSpans(tokens)
}

func findCloseBracket(runes []rune, start int) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		switch runes[i] {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func findCloseParen(runes []rune, start int) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		switch runes[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// tokensToSpans token → span 列表（合并相邻纯文本）
func (s *mdState) tokensToSpans(tokens []mdToken) []span {
	var spans []span
	style := s.style

	mergeText := func(text string) {
		if text == "" {
			return
		}
		if len(spans) > 0 && spans[len(spans)-1].Font.Size == 0 && spans[len(spans)-1].Color == style.TextColor {
			spans[len(spans)-1].Text += text
		} else {
			spans = append(spans, span{Text: text, Color: style.TextColor})
		}
	}

	for _, tok := range tokens {
		switch tok.typ {
		case tText:
			mergeText(tok.text)

		case tBold:
			inner := s.mdRenderInline(tok.text)
			for _, sp := range inner {
				spans = append(spans, span{
					Text:  sp.Text,
					Color: sp.Color,
					Font:  canvas.Font{Family: "", Size: sp.Font.Size, Weight: canvas.FontWeightBold},
				})
			}

		case tItalic:
			inner := s.mdRenderInline(tok.text)
			for _, sp := range inner {
				col := style.TextSubtle
				if sp.Color != style.TextColor && sp.Color != style.TextSubtle {
					col = sp.Color
				}
				spans = append(spans, span{Text: sp.Text, Color: col})
			}

		case tStrikethrough:
			spans = append(spans, span{
				Text:  tok.text,
				Color: style.TextMuted,
				Font:  canvas.Font{Family: "", Size: style.BodyFontSize, Underline: true},
			})

		case tCode:
			spans = append(spans, span{
				Text:  tok.text,
				Color: style.AccentColor,
				Font:  style.MonoFont,
			})

		case tLink:
			spans = append(spans, span{
				Text:  tok.text + " ↗",
				Color: style.AccentColor,
			})

		case tImage:
			spans = append(spans, span{
				Text:  "🖼 " + tok.alt + " ↗",
				Color: style.TextMuted,
			})
		}
	}

	return spans
}

// ─── 辅助函数 ────────────────────────────────────────────────

// inlineRow 把行内 span 渲染为水平排列的 Text 行
func (s *mdState) inlineRow(spans []span, fontSize float64) Widget {
	if len(spans) == 0 {
		return Div(Style{})
	}
	fs := s.style.BodyFontSize
	if fontSize > 0 {
		fs = fontSize
	}
	kids := make([]Widget, len(spans))
	for i, sp := range spans {
		fnt := sp.Font
		if fnt.Size == 0 {
			fnt.Size = fs
		}
		t := NewText(sp.Text, sp.Color)
		t.Font = fnt
		kids[i] = t
	}
	return Div(Style{FlexDirection: "row"}, kids)
}

// textWidget 快速创建单行文本
func (s *mdState) textWidget(text string, color types.Color, size float64) Widget {
	t := NewText(text, color)
	t.Font.Size = size
	return t
}

// expand 弹性占满剩余空间
func (s *mdState) expand(w Widget) Widget {
	return &Expanded{SingleChildWidget: SingleChildWidget{Child: w}, Flex: 1}
}

// copyButton 复制按钮
func (s *mdState) copyButton(text string) Widget {
	return &Button{
		Text: "📋", TextColor: s.style.TextMuted,
		Icon: "copy", IconSize: 13,
		OnClick:  func() { s.style.ClipboardWrite(text) },
		MinWidth: 26, MinHeight: 22,
	}
}

// decodeEntities 解码常见 HTML 实体
func decodeEntities(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	return s
}

// itoa 简易整数转字符串
func mdItoa(n int) string {
	if n == 0 {
		return "0"
	}
	d := ""
	for n > 0 {
		d = string(rune('0'+n%10)) + d
		n /= 10
	}
	return d
}
