// 搜索面板 —— 左栏「搜索」内容：复刻参考 SearchPanel（搜索框 + 大小写/全词/正则开关 +
// 结果树：文件头可折叠 + 命中行 行号+高亮匹配）。跨文件内容搜索，点命中行跳编辑器。详见 AGENTS.md。
//
//go:build windows

package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

var searchHL = types.ColorFromRGB(87, 64, 0) // 命中高亮底（暗金）

const searchMaxMatches = 500

// searchMatch 一行命中：行号 + 行文本（已左 trim）+ 匹配字节区间（相对 trim 后文本）。
type searchMatch struct {
	line   int
	text   string
	ranges [][2]int
}

type searchFile struct {
	rel     string
	abspath string
	matches []searchMatch
}

var theSearch = &searchState{collapsed: map[string]bool{}}

// SearchPanel 搜索面板组件。
type SearchPanel struct{ widget.StatefulWidget }

func (p *SearchPanel) CreateState() widget.State { return theSearch }

type searchState struct {
	widget.BaseState
	query                           string
	caseSensitive, wholeWord, regex bool
	showReplace                     bool
	replaceText                     string
	files                           []searchFile
	totalMatches                    int
	searched                        bool
	capped                          bool
	errMsg                          string
	collapsed                       map[string]bool
	previewRe                       *regexp.Regexp // 替换预览用（Build 时按 replaceText 编译；nil=不预览）
}

// compile 据查询 + 选项构造正则。
func (s *searchState) compile() (*regexp.Regexp, error) {
	pat := s.query
	if !s.regex {
		pat = regexp.QuoteMeta(pat)
	}
	if s.wholeWord {
		pat = `\b` + pat + `\b`
	}
	if !s.caseSensitive {
		pat = `(?i)` + pat
	}
	return regexp.Compile(pat)
}

var searchSkipDir = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true,
	".cache": true, ".idea": true, ".vscode": true, "bin": true, "obj": true,
}

// run 跨文件搜索（同步遍历工作区；跳过 .git/node_modules 等、>1MB、二进制；命中上限 500）。
func (s *searchState) run() {
	s.searched = true
	s.files, s.totalMatches, s.capped, s.errMsg = nil, 0, false, ""
	if strings.TrimSpace(s.query) == "" {
		s.SetState()
		return
	}
	re, err := s.compile()
	if err != nil {
		s.errMsg = "正则错误：" + err.Error()
		s.SetState()
		return
	}
	root := theFileTree.rootPath
	filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if s.totalMatches >= searchMaxMatches {
			s.capped = true
			return filepath.SkipAll
		}
		if d.IsDir() {
			if searchSkipDir[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if info, e := d.Info(); e == nil && info.Size() > 1<<20 {
			return nil // >1MB 跳过
		}
		data, e := os.ReadFile(path)
		if e != nil || bytes.IndexByte(data, 0) >= 0 {
			return nil // 读失败 / 含 NUL → 视为二进制
		}
		var matches []searchMatch
		for i, line := range strings.Split(string(data), "\n") {
			locs := re.FindAllStringIndex(line, -1)
			if len(locs) == 0 {
				continue
			}
			trimmed := strings.TrimLeft(line, " \t")
			shift := len(line) - len(trimmed)
			var ranges [][2]int
			for _, l := range locs {
				a, b := l[0]-shift, l[1]-shift
				if a < 0 {
					a = 0
				}
				if b > len(trimmed) {
					b = len(trimmed)
				}
				if a < b {
					ranges = append(ranges, [2]int{a, b})
				}
			}
			matches = append(matches, searchMatch{line: i + 1, text: trimmed, ranges: ranges})
			s.totalMatches += len(locs)
			if s.totalMatches >= searchMaxMatches {
				break
			}
		}
		if len(matches) > 0 {
			rel, _ := filepath.Rel(root, path)
			s.files = append(s.files, searchFile{rel: filepath.ToSlash(rel), abspath: path, matches: matches})
		}
		return nil
	})
	s.SetState()
}

func (s *searchState) toggleFile(rel string) { s.collapsed[rel] = !s.collapsed[rel]; s.SetState() }

func searchInputStyle(in *widget.Input) {
	in.Color = ghText
	in.CursorColor = ghText
	in.BGColor = *ghBgPrimary
	in.BorderColor = *ghBorder
	in.FocusBorderColor = *ghAccent
	in.HoverBorderColor = *ghBorder
}

// confirmReplaceAll 先确认（替换不可撤销）。
func (s *searchState) confirmReplaceAll() {
	if strings.TrimSpace(s.query) == "" || len(s.files) == 0 {
		return
	}
	msg := fmt.Sprintf("将在 %d 个文件中把匹配替换为「%s」，此操作不可撤销。确定？", len(s.files), s.replaceText)
	widget.ShowConfirm("全部替换", msg, widget.MsgWarning, func() { s.replaceAll() }, nil)
}

// replaceAll 全部替换：写回 + 重搜 + 提示。
func (s *searchState) replaceAll() {
	n := s.doReplace()
	s.run() // 重搜：结果应减少/清空
	widget.ShowAlert("替换完成", fmt.Sprintf("已在 %d 个文件中替换。", n), widget.MsgInfo, nil)
}

// doReplace 对所有命中文件做整文件正则替换并写回（非正则=字面替换；正则=支持 $1）。返回改动文件数。
func (s *searchState) doReplace() int {
	re, err := s.compile()
	if err != nil {
		return 0
	}
	n := 0
	for _, f := range s.files {
		data, e := os.ReadFile(f.abspath)
		if e != nil {
			continue
		}
		var out string
		if s.regex {
			out = re.ReplaceAllString(string(data), s.replaceText)
		} else {
			out = re.ReplaceAllLiteralString(string(data), s.replaceText)
		}
		if out != string(data) {
			if os.WriteFile(f.abspath, []byte(out), 0o644) == nil {
				n++
				theEditor.reloadIfOpen(f.abspath) // 已打开则刷新编辑器
			}
		}
	}
	return n
}

// ─── UI ───────────────────────────────────────────────────────

func (s *searchState) Build(ctx widget.BuildContext) widget.Widget {
	s.previewRe = nil // 替换预览：替换模式且有替换文本时编译一次，供 matchRow 复用
	if s.showReplace && s.replaceText != "" {
		if re, err := s.compile(); err == nil {
			s.previewRe = re
		}
	}
	rows := []widget.Widget{s.searchBar()}
	switch {
	case s.errMsg != "":
		rows = append(rows, widget.Div(widget.Style{Padding: types.EdgeInsets(10)}, label(s.errMsg, gitRed, 11)))
	case s.searched && len(s.files) == 0:
		rows = append(rows, expand(gitMessage("search", "无结果", "没有匹配的内容")))
	case s.searched:
		rows = append(rows, s.stats())
		res := []widget.Widget{}
		for _, f := range s.files {
			s.fileBlock(&res, f)
		}
		rows = append(rows, expand(widget.NewScrollView(flexCol(res...))))
	}
	return widget.Div(
		widget.Style{BackgroundColor: cSide, FlexDirection: "column", AlignItems: "stretch"},
		rows,
	)
}

func (s *searchState) searchBar() widget.Widget {
	in := widget.NewInput("", func(t string) { s.query = t }).WithPlaceholder("搜索").WithOnSubmit(func(string) { s.run() })
	searchInputStyle(in)
	rows := []widget.Widget{
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			expand(in),
			widget.Div(widget.Style{Width: 4}),
			ftIconBtn("search", s.run),
		),
		widget.Div(widget.Style{Height: 6}),
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			searchToggle("Aa", "区分大小写", s.caseSensitive, func() { s.caseSensitive = !s.caseSensitive; s.run() }),
			widget.Div(widget.Style{Width: 4}),
			searchToggle("全词", "全字匹配", s.wholeWord, func() { s.wholeWord = !s.wholeWord; s.run() }),
			widget.Div(widget.Style{Width: 4}),
			searchToggle(".*", "正则", s.regex, func() { s.regex = !s.regex; s.run() }),
			widget.Div(widget.Style{Width: 4}),
			searchToggle("替换", "替换模式", s.showReplace, func() { s.showReplace = !s.showReplace; s.SetState() }),
		),
	}
	if s.showReplace {
		rin := widget.NewInput("替换为...", func(t string) { s.replaceText = t; s.SetState() }).WithOnSubmit(func(string) { s.confirmReplaceAll() })
		searchInputStyle(rin)
		rows = append(rows,
			widget.Div(widget.Style{Height: 6}),
			widget.Div(
				widget.Style{FlexDirection: "row", AlignItems: "center"},
				expand(rin),
				widget.Div(widget.Style{Width: 4}),
				&widget.Button{
					SingleChildWidget: widget.SingleChildWidget{Child: label("全部替换", cWhite, 11)},
					OnClick:           s.confirmReplaceAll,
					Color:             gitRed, MinHeight: 24, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0),
				},
			),
		)
	}
	return widget.Div(
		widget.Style{Padding: types.EdgeInsets(6), FlexDirection: "column", AlignItems: "stretch",
			BackgroundColor: cSide, BorderColor: cBorder, BorderWidth: 1},
		rows,
	)
}

func (s *searchState) stats() widget.Widget {
	txt := fmtMatchStats(s.totalMatches, len(s.files))
	if s.capped {
		txt += "（已截断 " + itoa(searchMaxMatches) + "）"
	}
	return widget.Div(
		widget.Style{Height: 22, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
		label(txt, cTextDim, 10),
	)
}

// fileBlock 一个文件的结果：文件头（可折叠）+ 命中行。
func (s *searchState) fileBlock(out *[]widget.Widget, f searchFile) {
	chev := "chevron-down"
	if s.collapsed[f.rel] {
		chev = "chevron-right"
	}
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(chev, widget.IconSize(13), widget.IconColor(cTextDim)),
			widget.Div(widget.Style{Width: 3}),
			widget.Lucide("file-text", widget.IconSize(13), widget.IconColor(cTextDim)),
			widget.Div(widget.Style{Width: 5}),
			expand(label(f.rel, cText, 12)),
			label(itoa(len(f.matches)), cTextDim, 10),
		)},
		OnClick:    func() { s.toggleFile(f.rel) },
		HoverColor: *ftHover,
	}
	*out = append(*out, header)
	if s.collapsed[f.rel] {
		return
	}
	for _, m := range f.matches {
		*out = append(*out, s.matchRow(f.abspath, m))
	}
}

func (s *searchState) matchRow(abspath string, m searchMatch) widget.Widget {
	seg := []widget.Widget{
		widget.Div(widget.Style{Width: 34, FlexDirection: "row", AlignItems: "center"}, label(itoa(m.line), cTextDim, 10)),
	}
	seg = append(seg, highlightedLine(m.text, m.ranges)...)
	mainRow := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 22, Padding: types.EdgeInsetsLTRB(20, 0, 6, 0), FlexDirection: "row", AlignItems: "center"},
			seg,
		)},
		OnClick:    func() { theEditor.openAt(abspath, m.line) },
		HoverColor: *ftHover,
	}
	if s.previewRe == nil {
		return mainRow
	}
	var repl string // 替换预览：该行替换后的样子（绿色）
	if s.regex {
		repl = s.previewRe.ReplaceAllString(m.text, s.replaceText)
	} else {
		repl = s.previewRe.ReplaceAllLiteralString(m.text, s.replaceText)
	}
	if repl == m.text {
		return mainRow // 该行无变化（不应发生，保险）
	}
	if len(repl) > 200 {
		repl = repl[:200]
	}
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		mainRow,
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(40, 0, 6, 2), FlexDirection: "row", AlignItems: "center"},
			label(repl, gitGreen, 11.5),
		),
	)
}

// highlightedLine 把行按匹配区间切成 普通/高亮 段。
func highlightedLine(text string, ranges [][2]int) []widget.Widget {
	if len(text) > 200 {
		text = text[:200] // 超长行截断，避免行过宽
	}
	var segs []widget.Widget
	prev := 0
	for _, r := range ranges {
		a, b := r[0], r[1]
		if a >= len(text) || a < prev {
			continue
		}
		if b > len(text) {
			b = len(text)
		}
		if a > prev {
			segs = append(segs, label(text[prev:a], cTextDim, 11.5))
		}
		segs = append(segs, widget.Div(
			widget.Style{BackgroundColor: &searchHL, Padding: types.EdgeInsetsLTRB(1, 0, 1, 0)},
			label(text[a:b], cText, 11.5),
		))
		prev = b
	}
	if prev < len(text) {
		segs = append(segs, label(text[prev:], cTextDim, 11.5))
	}
	return segs
}

// searchToggle 选项开关（激活高亮）。
func searchToggle(text, _ string, on bool, onClick func()) widget.Widget {
	bg, tc := *cTitle, cTextDim
	if on {
		bg, tc = *ghAccentEmph, cWhite
	}
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: label(text, tc, 11)},
		OnClick:           onClick,
		Color:             bg,
		MinWidth:          30,
		MinHeight:         22,
		Padding:           types.EdgeInsetsLTRB(6, 0, 6, 0),
	}
}

func fmtMatchStats(matches, files int) string {
	return itoa(matches) + " 处匹配 · " + itoa(files) + " 个文件"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
