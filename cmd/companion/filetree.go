// 文件树面板 —— 左栏「文件」内容：读真实文件系统，懒加载、展开/折叠、按类型图标着色、
// 点击文件夹展开、点击文件选中（后续接编辑器）。VS Code 深色风。详见 AGENTS.md。
//
//go:build windows

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// ─── 文件树配色（VS Code 深色）──────────────────────────────
var (
	ftFolder   = types.ColorFromRGB(134, 174, 224) // 文件夹图标：柔和蓝
	ftJson     = types.ColorFromRGB(250, 204, 21)  // json/yaml：黄
	ftHover    = types.ColorRef(42, 45, 46)         // 行悬停 #2a2d2e（VS Code list hover）
	ftSelected = types.ColorRef(9, 71, 113)         // 行选中 #094771（VS Code list active）
)

// extColor 按扩展名给代码文件上色（其余走调用方默认）。
func extColor(ext string) types.Color {
	switch ext {
	case ".go":
		return types.ColorFromRGB(0, 173, 216) // cyan
	case ".ts", ".tsx":
		return types.ColorFromRGB(49, 120, 198)
	case ".js", ".jsx", ".mjs":
		return types.ColorFromRGB(247, 223, 30)
	case ".py":
		return types.ColorFromRGB(82, 139, 195)
	case ".rs":
		return types.ColorFromRGB(222, 165, 132)
	case ".html":
		return types.ColorFromRGB(227, 76, 38)
	case ".css", ".scss":
		return types.ColorFromRGB(66, 184, 131)
	case ".sh", ".bat", ".ps1":
		return types.ColorFromRGB(63, 185, 80)
	default:
		return cText
	}
}

// iconForFile 返回节点的 Lucide 图标名 + 颜色。
func iconForFile(n *fileNode) (string, types.Color) {
	if n.isDir {
		if n.expanded {
			return "folder-open", ftFolder
		}
		return "folder", ftFolder
	}
	ext := strings.ToLower(filepath.Ext(n.name))
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".rs", ".c", ".cpp", ".h", ".java",
		".py", ".rb", ".php", ".cs", ".swift", ".kt", ".html", ".css", ".scss", ".sh", ".bat", ".ps1":
		return "file-code", extColor(ext)
	case ".json", ".yaml", ".yml", ".toml", ".mod", ".sum", ".lock":
		return "braces", ftJson
	case ".md", ".txt", ".rst", ".log":
		return "file-text", cTextDim
	default:
		return "file", cTextDim
	}
}

// ─── 文件树模型 ───────────────────────────────────────────

type fileNode struct {
	name     string
	path     string
	isDir    bool
	children []*fileNode
	expanded bool
	loaded   bool // 子节点是否已读
}

// loadChildren 读目录子项：目录在前、各按名排序（不区分大小写）。失败置 loaded 防重试。
func loadChildren(n *fileNode) {
	n.loaded = true
	entries, err := os.ReadDir(n.path)
	if err != nil {
		return
	}
	var dirs, files []*fileNode
	for _, e := range entries {
		c := &fileNode{name: e.Name(), path: filepath.Join(n.path, e.Name()), isDir: e.IsDir()}
		if c.isDir {
			dirs = append(dirs, c)
		} else {
			files = append(files, c)
		}
	}
	byName := func(s []*fileNode) {
		sort.Slice(s, func(i, j int) bool { return strings.ToLower(s[i].name) < strings.ToLower(s[j].name) })
	}
	byName(dirs)
	byName(files)
	n.children = append(dirs, files...)
}

// ─── 文件树面板（有状态，包级单例：跨 relayout 存活，同 ChatPanel）──

var theFileTree = &fileTreeState{}

// FileTreePanel 文件树面板组件。
type FileTreePanel struct{ widget.StatefulWidget }

func (f *FileTreePanel) CreateState() widget.State { return theFileTree }

type fileTreeState struct {
	widget.BaseState
	root      *fileNode
	rootPath  string
	active    string              // 当前选中文件路径
	gitStatus map[string]gitBadge // 绝对路径→git 状态徽标（每次 Build 重建）
}

func (s *fileTreeState) ensure() {
	if s.root != nil {
		return
	}
	wd := currentRoot() // 当前项目根（打开的文件夹或运行目录）
	s.rootPath = wd
	s.root = &fileNode{name: filepath.Base(wd), path: wd, isDir: true, expanded: true}
	loadChildren(s.root)
}

func (s *fileTreeState) Build(ctx widget.BuildContext) widget.Widget {
	s.ensure()
	theGit.ensure()                 // 触发 git 状态异步加载（完成后 git drain 会 refresh 文件树→徽标显现）
	s.gitStatus = gitStatusMap()    // 据当前 git 状态标记改动文件（未加载则 nil）
	rows := []widget.Widget{}
	s.flatten(s.root.children, 0, &rows)
	panel := widget.Div(
		widget.Style{BackgroundColor: cSide, FlexDirection: "column", AlignItems: "stretch"},
		s.toolbar(),
		expand(widget.NewScrollView(flexCol(rows...))),
	)
	return &widget.ContextArea{ // 右键空白处：根目录菜单（行的右键已 StopPropagation，不会冒到这）
		SingleChildWidget: widget.SingleChildWidget{Child: panel},
		OnContextMenu:     func(x, y float64) { fileTreeEmptyMenu(x, y) },
	}
}

// toolbar 文件树工具条：当前仅刷新（新建文件/文件夹待接 Prompt 对话框）。
func (s *fileTreeState) toolbar() widget.Widget {
	return widget.Div(
		widget.Style{Height: 28, Padding: types.EdgeInsetsLTRB(6, 0, 6, 0), FlexDirection: "row", AlignItems: "center",
			BackgroundColor: cSide, BorderColor: cBorder, BorderWidth: 1},
		expand(label(filepath.Base(s.rootPath), cTextDim, 11)),
		ftIconBtn("refresh-cw", s.refresh),
	)
}

func (s *fileTreeState) flatten(nodes []*fileNode, depth int, out *[]widget.Widget) {
	for _, n := range nodes {
		*out = append(*out, s.row(n, depth))
		if n.isDir && n.expanded {
			s.flatten(n.children, depth+1, out)
		}
	}
}

// row 单行：整行可点（Clickable，铺满宽 + 选中/悬停高亮）+ 缩进 + 图标 + 文件名。
func (s *fileTreeState) row(n *fileNode, depth int) widget.Widget {
	icon, iconCol := iconForFile(n)
	bg := types.Color{}
	if n.path == s.active {
		bg = *ftSelected
	}
	indent := 8.0 + float64(depth)*14
	// git 状态：改动文件名变色 + 行尾状态符（M/?/+ 等）。
	nameCol := cText
	var trailing widget.Widget = widget.Div(widget.Style{})
	if gb, ok := s.gitStatus[n.path]; ok {
		nameCol = gb.col
		trailing = label(gb.sym, gb.col, 11)
	}
	row := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, FlexDirection: "row", AlignItems: "center",
				Padding: types.EdgeInsetsLTRB(indent, 0, 8, 0)},
			widget.Lucide(icon, widget.IconSize(15), widget.IconColor(iconCol)),
			widget.Div(widget.Style{Width: 6}),
			expand(label(n.name, nameCol, 12.5)),
			trailing,
		)},
		OnClick:    func() { s.onClick(n) },
		Color:      bg,
		HoverColor: *ftHover,
	}
	return &widget.ContextArea{ // 右键：文件/文件夹菜单（StopPropagation 不冒泡到空白区菜单）
		SingleChildWidget: widget.SingleChildWidget{Child: row},
		OnContextMenu:     func(x, y float64) { fileNodeMenu(x, y, n) },
	}
}

// toggle 展开/折叠目录（右键菜单用，等价点击目录）。
func (s *fileTreeState) toggle(n *fileNode) {
	if !n.isDir {
		return
	}
	if !n.loaded {
		loadChildren(n)
	}
	n.expanded = !n.expanded
	s.SetState()
}

// selectPath 选中某路径（高亮）。
func (s *fileTreeState) selectPath(p string) { s.active = p; s.SetState() }

func (s *fileTreeState) onClick(n *fileNode) {
	if n.isDir {
		if !n.loaded {
			loadChildren(n)
		}
		n.expanded = !n.expanded
	} else {
		s.active = n.path
		theEditor.open(n.path) // 在中列编辑区打开（全局 relayout 让编辑区重建读取）
	}
	s.SetState()
}

// setRoot 切换工作区根目录（打开文件夹）：重建文件树、清选中、relayout。
func (s *fileTreeState) setRoot(p string) {
	if p == "" {
		return
	}
	s.rootPath = p
	s.active = ""
	s.root = &fileNode{name: filepath.Base(p), path: p, isDir: true, expanded: true}
	loadChildren(s.root)
	s.SetState()
}

// refresh 重读文件系统，保留原展开状态（按路径快照→重展开）。
func (s *fileTreeState) refresh() {
	if s.root == nil {
		return
	}
	exp := map[string]bool{}
	collectExpanded(s.root, exp)
	s.root = &fileNode{name: filepath.Base(s.rootPath), path: s.rootPath, isDir: true, expanded: true}
	loadChildren(s.root)
	reExpand(s.root, exp)
	s.SetState()
}

func collectExpanded(n *fileNode, exp map[string]bool) {
	for _, c := range n.children {
		if c.isDir && c.expanded {
			exp[c.path] = true
			collectExpanded(c, exp)
		}
	}
}

func reExpand(n *fileNode, exp map[string]bool) {
	for _, c := range n.children {
		if c.isDir && exp[c.path] {
			loadChildren(c)
			c.expanded = true
			reExpand(c, exp)
		}
	}
}

// ftIconBtn 文件树工具条图标按钮。
func ftIconBtn(icon string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: icon, IconSize: 14, TextColor: cTextDim,
		OnClick: onClick, Color: *cSide,
		MinWidth: 24, MinHeight: 22,
	}
}
