// Git 面板 —— 左栏「Git」内容：复刻参考 GitPanel（仓库名/分支/领先落后 + 全部暂存/提交/拉取/推送
// + 已暂存/冲突/已修改/未跟踪 四段可折叠列表，每行状态徽标 + 暂存/取消暂存/丢弃动作）。
// git 经 `git -C <root> ...` CLI 跑；porcelain 解析分类。详见 AGENTS.md。
//
//go:build windows

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// ─── Git 状态色（复刻参考调色板）──────────────────────────────
var (
	gitGreen  = types.ColorFromHex("#3fb950") // 已暂存/新增
	gitOrange = types.ColorFromHex("#d29922") // 已修改/重命名
	gitRed    = types.ColorFromHex("#f85149") // 删除/冲突
	gitGray   = types.ColorFromRGB(110, 118, 129)
	gitBlue   = types.ColorFromHex("#58a6ff")
)

// runGit 在 root 下跑 git，返回 stdout；失败带 stderr。
func runGit(root string, args ...string) (string, error) {
	full := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", full...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), fmt.Errorf("%s", msg)
	}
	return out.String(), nil
}

// gitEntry 一个变更项。x/y 为 porcelain 的暂存/工作区状态字符。
type gitEntry struct {
	path string
	x, y byte
}

// gitCommit 一条提交记录（提交历史用）。
type gitCommit struct {
	hash, short, author, date, msg string
}

// ─── Git 面板（有状态，包级单例）──────────────────────────────

var theGit = &gitState{collapsed: map[string]bool{}}

// GitPanel Git 面板组件。
type GitPanel struct{ widget.StatefulWidget }

func (g *GitPanel) CreateState() widget.State { return theGit }

type gitState struct {
	widget.BaseState
	root          string
	branch        string
	ahead, behind int
	isRepo        bool
	loaded        bool
	errMsg        string
	staged        []gitEntry
	conflict      []gitEntry
	modified      []gitEntry
	untracked     []gitEntry
	branches      []string
	commits       []gitCommit
	collapsed     map[string]bool
}

func (g *gitState) ensure() {
	if g.loaded {
		return
	}
	g.reload()
}

// reload 重读 git 状态（首次/刷新/动作后）。同步调用 git CLI。
func (g *gitState) reload() {
	g.loaded = true
	g.root = theFileTree.rootPath
	g.branch, g.ahead, g.behind, g.errMsg = "", 0, 0, ""
	g.staged, g.conflict, g.modified, g.untracked = nil, nil, nil, nil
	g.branches, g.commits = nil, nil

	if out, err := runGit(g.root, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(out) != "true" {
		g.isRepo = false
		return
	}
	g.isRepo = true
	if b, err := runGit(g.root, "branch", "--show-current"); err == nil {
		g.branch = strings.TrimSpace(b)
	}
	if out, err := runGit(g.root, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		fmt.Sscanf(strings.TrimSpace(out), "%d\t%d", &g.ahead, &g.behind)
	}
	out, err := runGit(g.root, "status", "--porcelain")
	if err != nil {
		g.errMsg = err.Error()
		return
	}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		path := line[3:]
		if i := strings.Index(path, " -> "); i >= 0 { // 重命名：取新名
			path = path[i+4:]
		}
		g.categorize(x, y, strings.TrimSpace(path))
	}
	// 分支列表（切换用）
	if out, err := runGit(g.root, "branch", "--format=%(refname:short)"); err == nil {
		for _, b := range strings.Split(strings.TrimSpace(out), "\n") {
			if b = strings.TrimSpace(b); b != "" {
				g.branches = append(g.branches, b)
			}
		}
	}
	// 提交历史（最近 20 条）：full|short|author|相对日期|主题
	if out, err := runGit(g.root, "log", "--max-count=20", "--pretty=format:%H|%h|%an|%cr|%s"); err == nil {
		for _, ln := range strings.Split(out, "\n") {
			if p := strings.SplitN(ln, "|", 5); len(p) == 5 {
				g.commits = append(g.commits, gitCommit{hash: p[0], short: p[1], author: p[2], date: p[3], msg: p[4]})
			}
		}
	}
}

// checkoutBranch 切换分支（git checkout）。
func (g *gitState) checkoutBranch(b string) {
	if b == "" || b == g.branch {
		return
	}
	g.act("checkout", b)
}

func (g *gitState) categorize(x, y byte, path string) {
	e := gitEntry{path: path, x: x, y: y}
	switch {
	case x == '?' && y == '?':
		g.untracked = append(g.untracked, e)
	case x == 'U' || y == 'U' || (x == 'D' && y == 'D') || (x == 'A' && y == 'A'):
		g.conflict = append(g.conflict, e)
	default:
		if x != ' ' && x != '?' {
			g.staged = append(g.staged, e)
		}
		if y != ' ' && y != '?' {
			g.modified = append(g.modified, e)
		}
	}
}

func (g *gitState) changeCount() int {
	return len(g.staged) + len(g.conflict) + len(g.modified) + len(g.untracked)
}

// badge 据状态字符给徽标符号+色（复刻参考 ? + - → ~ !）。
func badge(st byte, staged bool) (string, types.Color) {
	switch st {
	case '?':
		return "?", gitGray
	case 'A':
		return "+", gitGreen
	case 'D':
		return "-", gitRed
	case 'R', 'C':
		return "→", gitOrange
	case 'U':
		return "!", gitRed
	default: // M 等
		if staged {
			return "~", gitGreen
		}
		return "~", gitOrange
	}
}

// ─── 动作（git CLI；完成后 reload + relayout）──────────────────

func (g *gitState) act(args ...string) {
	if _, err := runGit(g.root, args...); err != nil {
		widget.ShowAlert("Git 出错", err.Error(), widget.MsgWarning, nil)
	}
	g.reload()
	g.SetState()
	theFileTree.refresh() // 文件可能新增/删除
}

func (g *gitState) stageAll()           { g.act("add", "-A") }
func (g *gitState) stageFile(p string)  { g.act("add", "--", p) }
func (g *gitState) unstageFile(p string) { g.act("reset", "-q", "HEAD", "--", p) }
func (g *gitState) discardFile(p string) {
	widget.ShowConfirm("丢弃更改", "确定丢弃「"+p+"」的工作区更改？不可撤销。", widget.MsgWarning,
		func() { g.act("checkout", "--", p) }, nil)
}
func (g *gitState) push() { g.act("push") }
func (g *gitState) pull() { g.act("pull", "--ff-only") }
// commit 弹提交对话框（提交信息 必填 + 详细描述 可选，复刻参考 CommitDialog）。
func (g *gitState) commit() {
	if len(g.staged) == 0 {
		widget.ShowAlert("提交", "没有已暂存的更改。", widget.MsgWarning, nil)
		return
	}
	var msg, desc string
	msgIn := commitInput("提交信息（必填）", 2, func(t string) { msg = t })
	descIn := commitInput("详细描述（可选）", 3, func(t string) { desc = t })
	var id int
	body := widget.Div(
		widget.Style{Width: 400, FlexDirection: "column", AlignItems: "stretch"},
		msgIn,
		widget.Div(widget.Style{Height: 8}),
		descIn,
		widget.Div(widget.Style{Height: 6}),
		label(fmt.Sprintf("仅提交已暂存的更改（%d 项）", len(g.staged)), gitGray, 10),
	)
	dlg := widget.NewDialog("提交变更", body).WithWidth(440).WithFooter(
		widget.NewButton("取消", func() { widget.HideOverlay(id) }).WithColor(*ghBgTertiary).WithTextColor(ghText),
		widget.NewButton("提交", func() {
			if strings.TrimSpace(msg) == "" {
				return
			}
			widget.HideOverlay(id)
			args := []string{"commit", "-m", msg}
			if strings.TrimSpace(desc) != "" {
				args = append(args, "-m", desc)
			}
			g.act(args...)
		}).WithColor(*ghAccentEmph).WithTextColor(cWhite),
	)
	id = widget.ShowDialog(dlg)
}

// commitInput 提交对话框的多行输入框（深色）。
func commitInput(placeholder string, rows int, onChanged func(string)) *widget.Input {
	in := widget.NewInput(placeholder, onChanged)
	in.Multiline = true
	in.Rows = rows
	in.Color = ghText
	in.CursorColor = ghText
	in.BGColor = *ghBgPrimary
	in.BorderColor = *ghBorder
	in.FocusBorderColor = *ghAccent
	in.HoverBorderColor = *ghBorder
	return in
}

func (g *gitState) toggleSection(name string) { g.collapsed[name] = !g.collapsed[name]; g.SetState() }

// ─── UI ───────────────────────────────────────────────────────

func (g *gitState) Build(ctx widget.BuildContext) widget.Widget {
	g.ensure()
	if !g.isRepo {
		return gitMessage("git-branch", "非 Git 仓库", "此目录未初始化 Git")
	}
	body := []widget.Widget{g.repoBar(), g.actionBar()}
	if g.changeCount() == 0 && len(g.commits) == 0 {
		body = append(body, expand(gitMessage("circle-check", "工作区干净", "没有未提交的变更")))
	} else {
		secs := []widget.Widget{}
		if g.changeCount() == 0 {
			secs = append(secs, widget.Div( // 干净但有历史：一行轻提示
				widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
				widget.Lucide("circle-check", widget.IconSize(13), widget.IconColor(gitGreen)),
				widget.Div(widget.Style{Width: 6}), label("工作区干净", cTextDim, 11),
			))
		} else {
			g.section(&secs, "已暂存", "staged", g.staged, gitGreen, true)
			g.section(&secs, "冲突", "conflict", g.conflict, gitRed, false)
			g.section(&secs, "已修改", "modified", g.modified, gitOrange, false)
			g.section(&secs, "未跟踪", "untracked", g.untracked, gitGray, false)
		}
		g.commitHistory(&secs)
		body = append(body, expand(widget.NewScrollView(flexCol(secs...))))
	}
	return widget.Div(
		widget.Style{BackgroundColor: cSide, FlexDirection: "column", AlignItems: "stretch"},
		body,
	)
}

// repoBar 顶部：分支名 + 领先/落后 + 刷新。
func (g *gitState) repoBar() widget.Widget {
	br := g.branch
	if br == "" {
		br = "（无分支）"
	}
	kids := []widget.Widget{
		widget.Lucide("git-branch", widget.IconSize(13), widget.IconColor(gitBlue)),
		widget.Div(widget.Style{Width: 6}),
		g.branchSelector(br),
	}
	if g.ahead > 0 {
		kids = append(kids, widget.Div(widget.Style{Width: 8}), label(fmt.Sprintf("↑%d", g.ahead), cTextDim, 10))
	}
	if g.behind > 0 {
		kids = append(kids, widget.Div(widget.Style{Width: 6}), label(fmt.Sprintf("↓%d", g.behind), cTextDim, 10))
	}
	kids = append(kids, expand(widget.Div(widget.Style{})), ftIconBtn("refresh-cw", func() { g.reload(); g.SetState() }))
	return widget.Div(
		widget.Style{Height: 32, Padding: types.EdgeInsetsLTRB(10, 0, 6, 0), BackgroundColor: cSide,
			BorderColor: cBorder, BorderWidth: 1, FlexDirection: "row", AlignItems: "center"},
		kids,
	)
}

// branchSelector 当前分支 + 下拉切换（>1 分支才下拉，否则纯标签）。
func (g *gitState) branchSelector(br string) widget.Widget {
	if len(g.branches) <= 1 {
		return label(br, cText, 12)
	}
	items := make([]widget.DropdownItem, 0, len(g.branches))
	for _, b := range g.branches {
		items = append(items, widget.DropdownItem{Label: b, Command: b, Checked: b == g.branch})
	}
	trigger := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			label(br, cText, 12),
			widget.Div(widget.Style{Width: 3}),
			widget.Lucide("chevron-down", widget.IconSize(12), widget.IconColor(cTextDim)),
		)},
		Color: *cSide, MinHeight: 22, Padding: types.EdgeInsetsLTRB(2, 0, 2, 0),
	}
	return widget.NewDropdown(trigger, items...).WithOnCommand(g.checkoutBranch).WithPlacement(widget.PlacementBottomStart)
}

// commitHistory 追加「提交历史」可折叠段 + 提交行。
func (g *gitState) commitHistory(out *[]widget.Widget) {
	if len(g.commits) == 0 {
		return
	}
	key := "history"
	chev := "chevron-down"
	if g.collapsed[key] {
		chev = "chevron-right"
	}
	*out = append(*out, &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(chev, widget.IconSize(13), widget.IconColor(gitBlue)),
			widget.Div(widget.Style{Width: 4}),
			label(fmt.Sprintf("提交历史 (%d)", len(g.commits)), gitBlue, 11),
		)},
		OnClick:    func() { g.toggleSection(key) },
		HoverColor: *ftHover,
	})
	if g.collapsed[key] {
		return
	}
	for _, c := range g.commits {
		*out = append(*out, g.commitRow(c))
	}
}

// commitRow 一条提交：短哈希(点击复制全哈希) + 主题 + 相对日期。
func (g *gitState) commitRow(c gitCommit) widget.Widget {
	full := c.hash
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 22, Padding: types.EdgeInsetsLTRB(16, 0, 6, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Div(widget.Style{Width: 48, FlexDirection: "row", AlignItems: "center"}, label(c.short, gitBlue, 10)),
			expand(label(c.msg, cText, 11)),
			widget.Div(widget.Style{Width: 6}),
			label(c.date, cTextDim, 10),
		)},
		OnClick:    func() { copyToClipboard(full) },
		HoverColor: *ftHover,
	}
}

// actionBar 全部暂存 / 提交 / 拉取 / 推送。
func (g *gitState) actionBar() widget.Widget {
	return widget.Div(
		widget.Style{Padding: types.EdgeInsets(6), FlexDirection: "row", AlignItems: "center"},
		gitBtn("全部暂存", gitGreen, len(g.modified)+len(g.untracked) > 0, g.stageAll),
		widget.Div(widget.Style{Width: 6}),
		gitBtn("提交", gitBlue, len(g.staged) > 0, g.commit),
		expand(widget.Div(widget.Style{})),
		ftIconBtn("arrow-down-to-line", g.pull),
		ftIconBtn("arrow-up-from-line", g.push),
	)
}

// section 一段（标题 + 计数，可折叠）+ 文件行。
func (g *gitState) section(out *[]widget.Widget, title, key string, items []gitEntry, accent types.Color, stagedSec bool) {
	if len(items) == 0 {
		return
	}
	chev := "chevron-down"
	if g.collapsed[key] {
		chev = "chevron-right"
	}
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(chev, widget.IconSize(13), widget.IconColor(accent)),
			widget.Div(widget.Style{Width: 4}),
			label(fmt.Sprintf("%s (%d)", title, len(items)), accent, 11),
		)},
		OnClick:    func() { g.toggleSection(key) },
		HoverColor: *ftHover,
	}
	*out = append(*out, header)
	if g.collapsed[key] {
		return
	}
	for _, e := range items {
		*out = append(*out, g.fileRow(e, stagedSec))
	}
}

// fileRow 文件行：状态徽标 + 路径 + 动作按钮（暂存/取消暂存/丢弃）。
func (g *gitState) fileRow(e gitEntry, stagedSec bool) widget.Widget {
	st := e.x
	if !stagedSec {
		st = e.y
	}
	sym, col := badge(st, stagedSec)
	p := e.path
	kids := []widget.Widget{
		widget.Div(widget.Style{Width: 14, FlexDirection: "row", AlignItems: "center"}, label(sym, col, 12)),
		widget.Div(widget.Style{Width: 4}),
		expand(label(shortGitPath(p), cText, 12)),
	}
	if stagedSec {
		kids = append(kids, gitRowBtn("minus", "取消暂存", func() { g.unstageFile(p) }))
	} else if e.x == '?' && e.y == '?' {
		kids = append(kids, gitRowBtn("plus", "暂存", func() { g.stageFile(p) }))
	} else {
		kids = append(kids, gitRowBtn("plus", "暂存", func() { g.stageFile(p) }),
			gitRowBtn("trash-2", "丢弃", func() { g.discardFile(p) }))
	}
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(16, 0, 6, 0), FlexDirection: "row", AlignItems: "center"},
			kids,
		)},
		OnClick:    func() { openGitEntry(p) },
		HoverColor: *ftHover,
	}
}

// shortGitPath 取路径末段（目录前缀略灰，简化为只显文件名+父目录）。
func shortGitPath(p string) string {
	p = strings.ReplaceAll(p, "/", "\\")
	parts := strings.Split(p, "\\")
	if len(parts) <= 2 {
		return p
	}
	return ".../" + parts[len(parts)-2] + "/" + parts[len(parts)-1]
}

func openGitEntry(rel string) {
	abs := rel
	if theGit.root != "" {
		abs = theGit.root + "\\" + strings.ReplaceAll(rel, "/", "\\")
	}
	if line := firstChangedLine(theGit.root, rel); line > 0 {
		theEditor.openAt(abs, line) // 跳到首个改动行
	} else {
		theEditor.open(abs)
	}
}

// firstChangedLine 取文件 git diff 首个 hunk 的新文件起始行（+N）；无改动/新文件返回 0（打开顶部）。
func firstChangedLine(root, rel string) int {
	out, err := runGit(root, "diff", "-U0", "HEAD", "--", rel)
	if err != nil || strings.TrimSpace(out) == "" {
		out, err = runGit(root, "diff", "-U0", "--cached", "--", rel) // 仅暂存的改动
		if err != nil || strings.TrimSpace(out) == "" {
			return 0
		}
	}
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, "@@") {
			return parseHunkNewLine(ln)
		}
	}
	return 0
}

// parseHunkNewLine 解析 hunk 头 "@@ -a,b +c,d @@" 的 c（新文件起始行）。解析失败返回 0。
func parseHunkNewLine(s string) int {
	i := strings.IndexByte(s, '+')
	if i < 0 {
		return 0
	}
	rest := s[i+1:]
	n := 0
	for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
		n++
	}
	v, _ := strconv.Atoi(rest[:n])
	return v
}

// gitBtn 动作栏文本按钮（启用时着色，禁用置灰）。
func gitBtn(text string, color types.Color, enabled bool, onClick func()) widget.Widget {
	c := *cTitle
	tc := cTextDim
	if enabled {
		c = color
		tc = cWhite
	}
	b := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: label(text, tc, 11)},
		Color:             c,
		MinHeight:         24,
		Padding:           types.EdgeInsetsLTRB(10, 0, 10, 0),
	}
	if enabled {
		b.OnClick = onClick
	}
	return b
}

// gitRowBtn 行内小图标按钮。
func gitRowBtn(icon, _ string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: icon, IconSize: 13, TextColor: cTextDim,
		OnClick: onClick, Color: *cSide, MinWidth: 22, MinHeight: 22,
	}
}

// gitMessage 居中提示（空仓/干净/错误）。
func gitMessage(icon, title, sub string) widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "center", JustifyContent: "center", Padding: types.EdgeInsets(20)},
		widget.Lucide(icon, widget.IconSize(28), widget.IconColor(cTextDim)),
		widget.Div(widget.Style{Height: 8}),
		label(title, cText, 12),
		widget.Div(widget.Style{Height: 3}),
		label(sub, cTextDim, 11),
	)
}
