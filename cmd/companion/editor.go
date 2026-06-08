// 编辑器面板 —— 中列编辑区：多标签页 + 语法高亮 + 编辑 + 保存（Ctrl+S）。
// 点文件树文件 → 新标签（或切到已打开标签）；改动标 dirty(●)；Ctrl+S 写盘。
// 复用 goui 对标 Monaco 的 CodeEditor，切标签靠 ReloadToken 受控重载。详见 AGENTS.md。
//
//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// theEditor 编辑器状态（包级单例，StatefulWidget State，跨 relayout 存活）。
var theEditor = &editorState{}

type editorTab struct {
	path    string
	content string
	lang    string // 扩展名去点（交 ceLangFor / HasProvider）
	dirty   bool   // 有未保存改动
}

type editorState struct {
	widget.BaseState
	tabs     []*editorTab
	active   int
	reload   int // 受控重载令牌：切换/打开/关闭文件时自增 → CodeEditor 重置为新标签内容
	reveal   int // 跳转令牌：openAt 自增 → CodeEditor 跳到 gotoLine
	gotoLine int // 待跳转行(1 基)，供 openAt 跳转用
}

// EditorPanel 编辑器面板组件。
type EditorPanel struct{ widget.StatefulWidget }

func (e *EditorPanel) CreateState() widget.State { return theEditor }

func (e *editorState) activeTab() *editorTab {
	if e.active >= 0 && e.active < len(e.tabs) {
		return e.tabs[e.active]
	}
	return nil
}

// open 打开文件：已打开→切到该标签（无未保存改动时刷新为磁盘最新内容）；否则新建标签读内容。
func (e *editorState) open(path string) {
	for i, t := range e.tabs {
		if t.path == path {
			if !t.dirty {
				loadTabContent(t) // 重新打开已开文件→刷新为磁盘当前内容（无未保存改动时）
			}
			e.switchTo(i)
			return
		}
	}
	t := &editorTab{path: path}
	loadTabContent(t)
	e.tabs = append(e.tabs, t)
	e.active = len(e.tabs) - 1
	e.reload++
	e.persistSession()
	e.SetState()
}

// openAt 打开文件并跳转到第 line 行（1 基）。供搜索/Git 点击结果定位用。
func (e *editorState) openAt(path string, line int) {
	e.gotoLine = line
	e.reveal++
	e.open(path) // open 内部 reload++ + SetState，本次渲染即带上 gotoLine/reveal
}

// reloadIfOpen 若文件已在某标签打开且无未保存改动，则从磁盘重载内容（Agent 改文件后保持编辑器同步）。
// 有未保存改动(dirty)时不覆盖用户编辑。返回是否实际重载。
func (e *editorState) reloadIfOpen(path string) bool {
	for _, t := range e.tabs {
		if t.path == path {
			if t.dirty {
				return false // 有未保存改动，不覆盖
			}
			loadTabContent(t)
			e.reload++
			e.SetState()
			return true
		}
	}
	return false
}

func (e *editorState) switchTo(i int) {
	if i < 0 || i >= len(e.tabs) {
		return
	}
	e.active = i
	e.reload++
	e.persistSession()
	e.SetState()
}

func (e *editorState) close(i int) {
	if i < 0 || i >= len(e.tabs) {
		return
	}
	e.tabs = append(e.tabs[:i], e.tabs[i+1:]...)
	if e.active >= len(e.tabs) {
		e.active = len(e.tabs) - 1
	}
	e.reload++
	e.persistSession()
	e.SetState()
}

// closeOthers 关闭除第 i 个外的所有标签（右键菜单）。
func (e *editorState) closeOthers(i int) {
	if i < 0 || i >= len(e.tabs) {
		return
	}
	e.tabs = []*editorTab{e.tabs[i]}
	e.active = 0
	e.reload++
	e.persistSession()
	e.SetState()
}

// closeAll 关闭所有标签。
func (e *editorState) closeAll() {
	e.tabs = nil
	e.active = 0
	e.reload++
	e.persistSession()
	e.SetState()
}

// tabAt 取第 i 个标签（越界 nil）。
func (e *editorState) tabAt(i int) *editorTab {
	if i < 0 || i >= len(e.tabs) {
		return nil
	}
	return e.tabs[i]
}

// onEdit CodeEditor 内容变化回调：同步到当前标签 + 首次改动标 dirty。
func (e *editorState) onEdit(content string) {
	t := e.activeTab()
	if t == nil {
		return
	}
	t.content = content
	if !t.dirty {
		t.dirty = true
		e.SetState() // 仅首次编辑触发一次 relayout 以显示 ● dirty 标记
	}
}

// save 保存当前标签到磁盘（Ctrl+S）。
func (e *editorState) save() {
	t := e.activeTab()
	if t == nil || !t.dirty {
		return
	}
	if err := os.WriteFile(t.path, []byte(t.content), 0o644); err == nil {
		t.dirty = false
		e.SetState()
	}
}

func loadTabContent(t *editorTab) {
	if isBinaryExt(filepath.Ext(t.path)) {
		t.content, t.lang = "〔二进制文件，不预览〕", ""
		return
	}
	if fi, err := os.Stat(t.path); err == nil && fi.Size() > 2*1024*1024 {
		t.content, t.lang = "〔文件过大（>2MB），不预览〕", ""
		return
	}
	data, err := os.ReadFile(t.path)
	if err != nil {
		t.content, t.lang = "// 读取失败: "+err.Error(), ""
		return
	}
	t.content = string(data)
	t.lang = strings.TrimPrefix(strings.ToLower(filepath.Ext(t.path)), ".")
}

func isBinaryExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".exe", ".dll", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".bmp", ".webp", ".pdf",
		".zip", ".gz", ".tar", ".7z", ".rar", ".ttf", ".otf", ".woff", ".woff2",
		".so", ".a", ".o", ".bin", ".dat", ".db", ".class", ".wasm", ".mp3", ".mp4", ".wav":
		return true
	}
	return false
}

func (e *editorState) Build(ctx widget.BuildContext) widget.Widget {
	if len(e.tabs) == 0 { // 欢迎页（空状态）：居中大 logo + 标题 + 提示 + 打开文件夹
		return widget.Div(
			widget.Style{BackgroundColor: cEditor, FlexDirection: "column", AlignItems: "center", JustifyContent: "center"},
			pairLogoBig(),
			widget.Div(widget.Style{Height: 16}),
			label("Pair CodeAgent", cText, 22),
			widget.Div(widget.Style{Height: 6}),
			label("AI 结对编程助手", cTextDim, 13),
			widget.Div(widget.Style{Height: 22}),
			label("点击左侧文件树打开文件，或", cTextDim, 12),
			widget.Div(widget.Style{Height: 10}),
			&widget.Button{
				SingleChildWidget: widget.SingleChildWidget{Child: label("打开文件夹", cWhite, 12)},
				OnClick:           openFolderViaDialog,
				Color:             *ghAccentEmph, BorderRadius: 5, Padding: types.EdgeInsetsLTRB(14, 6, 14, 6),
			},
		)
	}
	t := e.activeTab()
	var editor widget.Widget
	// 有 LanguageProvider 的语言用 CodeWorkbench（结构化/代码视图可切换）；
	// 无 provider 的语言用标准 CodeEditor（语法高亮）。
	if widget.HasProvider(t.lang) {
		wb := widget.NewCodeWorkbench(t.content).WithSize(9000, 9000).WithLang(t.lang)
		wb.ReloadToken = e.reload
		wb.OnChange = e.onEdit
		editor = wb
	} else {
		ed := widget.NewCodeEditor(t.lang, t.content).WithSize(9000, 9000)
		ed.ReloadToken = e.reload
		ed.RevealLine = e.gotoLine
		ed.RevealToken = e.reveal
		ed.OnChange = e.onEdit
		ed.FontSize = float64(theSettings.EditorFontSize) // 外观设置：字号（0=默认 14）
		ed.Minimap = !theSettings.HideMinimap             // 外观设置：minimap（默认开）
		editor = ed
	}
	// 编辑器内容包一层 ContextArea：右键弹 companion 自定义菜单（撤销/剪贴/全选 + 结构化语言的视图切换），
	// 取代 goui 组件自带菜单（CodeEditor 经 SuppressEditorContextMenu 放行右键冒泡到这里）。
	content := &widget.ContextArea{
		SingleChildWidget: widget.SingleChildWidget{Child: editor},
		OnContextMenu:     editorContentMenu,
	}
	return widget.Div(
		widget.Style{BackgroundColor: cEditor, FlexDirection: "column", AlignItems: "stretch"},
		e.tabBar(),
		expand(content),
	)
}

// tabBar 编辑区顶部标签栏：每个打开的文件一个标签（类型图标+名+dirty●/关闭×），点切换、×关闭。
func (e *editorState) tabBar() widget.Widget {
	items := make([]widget.Widget, 0, len(e.tabs)*2)
	for i, t := range e.tabs {
		if i > 0 { // 相邻标签间 1px 竖分隔条（stretch 撑满高）
			items = append(items, widget.Div(widget.Style{Width: 1, BackgroundColor: cBorder}))
		}
		items = append(items, e.tabItem(i, t))
	}
	return widget.Div(
		widget.Style{Height: 34, BackgroundColor: cSide, FlexDirection: "row", AlignItems: "stretch"},
		items,
	)
}

func (e *editorState) tabItem(i int, t *editorTab) widget.Widget {
	active := i == e.active
	bg := *cSide
	txtCol := cTextDim
	if active {
		bg = *cEditor
		txtCol = cText
	}
	icon, iconCol := iconForFile(&fileNode{name: filepath.Base(t.path)})
	row := []widget.Widget{
		widget.Lucide(icon, widget.IconSize(13), widget.IconColor(iconCol)),
		widget.Div(widget.Style{Width: 6}),
		label(filepath.Base(t.path), txtCol, 12),
		widget.Div(widget.Style{Width: 8}),
	}
	if t.dirty {
		row = append(row, widget.Div(widget.Style{Width: 8, Height: 8, BackgroundColor: &cText, BorderRadius: 4}))
	} else {
		row = append(row, e.closeBtn(i))
	}
	tab := &widget.Clickable{
		// 显式 Height 让标签稳定填满标签栏高度并使内容垂直居中（不依赖 stretch 链路，单/多文件一致）。
		// 不再用整框边框（之前因标签按内容高、边框只有文字大小）；相邻标签靠 tabBar 里的分隔条区分。
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 34, FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(12, 0, 8, 0)},
			row,
		)},
		OnClick:    func() { e.switchTo(i) },
		Color:      bg,
		HoverColor: *ftHover,
	}
	return &widget.ContextArea{ // 右键：标签菜单（关闭/关闭其他/复制路径/添加到对话）
		SingleChildWidget: widget.SingleChildWidget{Child: tab},
		OnContextMenu:     func(x, y float64) { editorTabMenu(x, y, i) },
	}
}

// closeBtn 标签关闭×：StopPropagation 使点×只关闭、不触发外层标签切换。
func (e *editorState) closeBtn(i int) widget.Widget {
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Padding: types.EdgeInsets(3)},
			widget.Lucide("x", widget.IconSize(12), widget.IconColor(cTextDim)),
		)},
		OnClick:         func() { e.close(i) },
		StopPropagation: true,
		HoverColor:      *ftHover,
	}
}

// editorArea 中列编辑区入口（midColumn 调用）。
func editorArea() widget.Widget { return &EditorPanel{} }
