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
	roots     []*fileNode         // 工作区各文件夹的根节点（VS Code 多根）
	active    string              // 当前选中文件路径
	gitStatus map[string]gitBadge // 绝对路径→git 状态徽标（每次 Build 重建）
	// 多根拖拽排序（拖根文件夹的手柄重排；首个=Agent 主文件夹）
	dragPath        string  // 正在拖的根路径（""=未拖）
	dragLastY       float64 // 上次光标 Y（累积位移判定换位）
	dragStartPrimary string // 拖拽开始时的主文件夹（结束时判主是否变→是否重建 agent）
}

const rootRowH = 26.0 // 根文件夹行高（拖拽换位步长）

func (s *fileTreeState) ensure() {
	if len(s.roots) == 0 {
		s.buildRoots()
	}
}

// buildRoots 据 workspaceFolders 构建各根（保留原展开态）；无 SetState。
func (s *fileTreeState) buildRoots() {
	exp := map[string]bool{} // 快照展开态
	for _, r := range s.roots {
		collectExpanded(r, exp)
	}
	s.roots = nil
	folders := workspaceFolders
	if len(folders) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			wd = "."
		}
		folders = []string{wd} // 未打开工作区→运行目录兜底
	}
	for _, p := range folders {
		r := &fileNode{name: filepath.Base(p), path: p, isDir: true, expanded: true}
		loadChildren(r)
		reExpand(r, exp)
		s.roots = append(s.roots, r)
	}
}

// rebuildRoots 工作区文件夹变化后重建 + 刷新（project.go syncWorkspace 调）。
func (s *fileTreeState) rebuildRoots() {
	s.buildRoots()
	s.SetState()
}

func (s *fileTreeState) Build(ctx widget.BuildContext) widget.Widget {
	s.ensure()
	theGit.ensure()                 // 触发 git 状态异步加载（完成后 git drain 会 refresh 文件树→徽标显现）
	s.gitStatus = gitStatusMap()    // 据当前 git 状态标记改动文件（未加载则 nil）
	rows := []widget.Widget{}
	if len(s.roots) == 1 {
		s.flatten(s.roots[0].children, 0, &rows) // 单文件夹：直接显示内容（名字在头部）
	} else {
		for idx, r := range s.roots { // 多根：每个文件夹作可折叠根节（VS Code 风格；首个带主文件夹星标）
			rows = append(rows, s.rootRow(r, idx))
			if r.expanded {
				s.flatten(r.children, 1, &rows)
			}
		}
	}
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

// toolbar 文件树头部：工作区图标 + 工作区名（醒目，让用户看到打开的是哪个项目）+ 打开文件夹 + 刷新。
func (s *fileTreeState) toolbar() widget.Widget {
	return widget.Div(
		widget.Style{Height: 30, Padding: types.EdgeInsetsLTRB(8, 0, 6, 0), FlexDirection: "row", AlignItems: "center",
			BackgroundColor: cSide, BorderColor: cBorder, BorderWidth: 1},
		widget.Lucide("folder", widget.IconSize(13), widget.IconColor(cText)),
		widget.Div(widget.Style{Width: 6}),
		expand(label1(projectName(), cText, 12)),     // 工作区名（单文件夹名 / 多根「工作区 (N)」）
		ftIconBtn("folder-plus", addFolderViaDialog), // 添加文件夹到工作区（VS Code 多根）
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

// rootRow 多根工作区里，每个文件夹的可折叠根行：大写名 + chevron + idx==0 金色星标(Agent 主文件夹)
// + 右侧拖拽手柄(按住上下拖重排，首个=主文件夹)。点行折叠/展开，右键菜单。
func (s *fileTreeState) rootRow(r *fileNode, idx int) widget.Widget {
	chev := "chevron-down"
	if !r.expanded {
		chev = "chevron-right"
	}
	trail, trailCol := "", types.Color{}
	if idx == 0 { // 主文件夹（Agent 首选）：金色星标
		trail, trailCol = "star", types.ColorFromRGB(229, 192, 123)
	}
	// 整行可拖（DragRow 自绘叶子）：点行折叠/展开；长按或拖动→重排（首个=主文件夹）；右键菜单。
	return &widget.DragRow{
		LeadIcon: chev, LeadColor: cTextDim,
		Icon: "folder", Text: strings.ToUpper(r.name), TextColor: cText, TextSize: 11,
		TrailIcon: trail, TrailColor: trailCol,
		Height: rootRowH, Indent: 6,
		Bg: *cSide, HoverBg: *ftHover, Active: r.path == s.dragPath,
		OnTap:       func() { s.toggle(r) },
		OnContext:   func(x, y float64) { workspaceRootMenu(x, y, r.path) },
		OnDragStart: func(y float64) { s.onRootDragStart(r.path, y) },
		OnDragMove:  func(y float64) { s.onRootDragMove(y) },
		OnDragEnd:   func() { s.onRootDragEnd() },
	}
}

// onRootDragStart 手柄按下开始拖某根文件夹。
func (s *fileTreeState) onRootDragStart(path string, y float64) {
	s.dragPath = path
	s.dragLastY = y
	s.dragStartPrimary = currentRoot()
	s.SetState()
}

// onRootDragMove 拖动中：光标每移过一行高，就与相邻根实时换位（首个=主文件夹）。
func (s *fileTreeState) onRootDragMove(y float64) {
	if s.dragPath == "" {
		return
	}
	for y <= s.dragLastY-rootRowH { // 向上够一行高 → 上移
		i := indexOfFolder(s.dragPath)
		if i <= 0 {
			break
		}
		s.swapRoots(i, i-1)
		s.dragLastY -= rootRowH
	}
	for y >= s.dragLastY+rootRowH { // 向下够一行高 → 下移
		i := indexOfFolder(s.dragPath)
		if i < 0 || i >= len(workspaceFolders)-1 {
			break
		}
		s.swapRoots(i, i+1)
		s.dragLastY += rootRowH
	}
}

// onRootDragEnd 结束拖拽：落盘新顺序；主文件夹变了才重建 agent。
func (s *fileTreeState) onRootDragEnd() {
	if s.dragPath == "" {
		return
	}
	s.dragPath = ""
	syncWorkspace(currentRoot() != s.dragStartPrimary)
}

// swapRoots 拖拽中实时换两根（换 workspaceFolders + s.roots，保留展开态，不落盘——结束时统一落盘）。
func (s *fileTreeState) swapRoots(i, j int) {
	workspaceFolders[i], workspaceFolders[j] = workspaceFolders[j], workspaceFolders[i]
	if i < len(s.roots) && j < len(s.roots) {
		s.roots[i], s.roots[j] = s.roots[j], s.roots[i]
	}
	s.SetState()
}

// refresh 重读工作区各根的文件系统，保留展开状态。
func (s *fileTreeState) refresh() {
	s.rebuildRoots()
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
