// 右键上下文菜单（复刻参考源：文件树 / 编辑器标签 / 终端）。复用 goui 的 widget.ShowContextMenu
// （Overlay 菜单）+ widget.ContextArea（右键触发、布局透传）。新建/重命名走 showPrompt 输入框，
// 删除走 widget.ShowConfirm。详见 AGENTS.md。
//
//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
	"github.com/user/goui/internal/window"
)

// openFileViaDialog 弹系统「打开文件」对话框，选中即在编辑器打开（标题栏「文件→打开文件」Ctrl+O）。
func openFileViaDialog() {
	if window.OpenFileDialog == nil {
		return
	}
	var h uintptr
	if application != nil && application.Window != nil {
		h = application.Window.NativeHandle()
	}
	if p := window.OpenFileDialog(h, "打开文件", ""); p != "" {
		theEditor.open(p)
		theFileTree.selectPath(p)
	}
}

// pickFolder 弹系统“选择文件夹”对话框，返回选中路径（取消=空）。
func pickFolder(title string) string {
	if window.OpenFolderDialog == nil {
		return ""
	}
	var h uintptr
	if application != nil && application.Window != nil {
		h = application.Window.NativeHandle()
	}
	return window.OpenFolderDialog(h, title)
}

// openFolderViaDialog 打开文件夹：用选中文件夹替换整个工作区（VS Code「Open Folder」）。
func openFolderViaDialog() {
	if p := pickFolder("打开文件夹"); p != "" {
		setProject(p)
	}
}

// addFolderViaDialog 添加文件夹到工作区：把选中文件夹加进来，变多根工作区（VS Code「Add Folder to Workspace」）。
func addFolderViaDialog() {
	if p := pickFolder("添加文件夹到工作区"); p != "" {
		addFolder(p)
	}
}

// workspaceRootMenu 多根工作区里，右键某个根文件夹的菜单：排序（首个=Agent 主文件夹）+ 移除。
func workspaceRootMenu(x, y float64, path string) {
	i := indexOfFolder(path)
	items := []widget.MenuItem{
		mi("plus", "新建文件", func() { newEntryIn(path, false) }),
		mi("folder-plus", "新建文件夹", func() { newEntryIn(path, true) }),
		sep(),
		mi("terminal", "在终端打开", func() { theTerminal.openDir(path) }),
		mi("folder-open", "在资源管理器中显示", func() { revealInExplorer(path, true) }),
	}
	if i > 0 { // 非首位 → 可一键设为首选（拖拽手柄也能排序）
		items = append(items, sep(), mi("star", "设为首选项目（Agent 主文件夹）", func() { setPrimaryFolder(path) }))
	}
	items = append(items, sep(), miD("circle-x", "从工作区移除文件夹", func() { removeFolder(path) }))
	showMenu(x, y, items)
}

// mi 菜单项（icon=Lucide 图标名，空=无图标）。
func mi(icon, label string, onClick func()) widget.MenuItem {
	return widget.MenuItem{Icon: icon, Label: label, OnClick: onClick, Enabled: true}
}

// miD 危险菜单项（红字/红底，如删除）。
func miD(icon, label string, onClick func()) widget.MenuItem {
	return widget.MenuItem{Icon: icon, Label: label, OnClick: onClick, Enabled: true, Danger: true}
}

// sep 分组分隔线。
func sep() widget.MenuItem { return widget.MenuItem{Separator: true} }

// showMenu 用 GitHub 深色配色弹出右键菜单（统一各处外观，匹配参考源深色上下文菜单）。
func showMenu(x, y float64, items []widget.MenuItem) {
	widget.ShowContextMenuStyled(x, y, items, *ghBgTertiary, ghText, *ghAccentEmph, *ghBorder)
}

// copyToClipboard 写剪贴板（无实现则忽略）。
func copyToClipboard(s string) {
	if widget.ClipboardWrite != nil {
		widget.ClipboardWrite(s)
	}
}

// addToChat 把文本追加进对话输入框草稿并刷新（「添加到对话」动作）。
func addToChat(text string) {
	d := strings.TrimSpace(theChatState.store.Draft)
	if d != "" {
		d += "\n"
	}
	theChatState.store.Draft = d + text
	theChatState.sendSeq++ // 受控刷新输入框显示新草稿
	theChatState.SetState()
}

// revealInExplorer 在资源管理器中定位文件 / 打开目录（Windows）。explorer 退出码常非 0，忽略。
func revealInExplorer(path string, isDir bool) {
	if isDir {
		_ = exec.Command("explorer", path).Start()
	} else {
		_ = exec.Command("explorer", "/select,"+path).Start()
	}
}

// showPrompt 输入对话框（标题 + 单行输入 + 取消/确定）；确定且非空回调输入值。
func showPrompt(title, initial string, onOk func(string)) {
	val := initial
	in := widget.NewInput("", nil)
	in.Text = initial
	in.OnTextChanged = func(s string) { val = s }
	in.Color = ghText
	in.CursorColor = ghText
	in.BGColor = *ghBgPrimary
	in.BorderColor = *ghBorder
	in.FocusBorderColor = *ghAccent
	in.HoverBorderColor = *ghBorder
	var id int
	dlg := widget.NewDialog(title, widget.Div(widget.Style{Padding: types.EdgeInsets(4)}, in)).WithWidth(360).WithFooter(
		widget.NewButton("取消", func() { widget.HideOverlay(id) }).WithColor(*ghBgTertiary).WithTextColor(ghText),
		widget.NewButton("确定", func() {
			widget.HideOverlay(id)
			if v := strings.TrimSpace(val); v != "" {
				onOk(v)
			}
		}).WithColor(*ghAccentEmph).WithTextColor(cWhite),
	)
	id = widget.ShowDialog(dlg)
}

// ─── 文件树菜单 ───────────────────────────────────────────

func dirOf(n *fileNode) string {
	if n.isDir {
		return n.path
	}
	return filepath.Dir(n.path)
}

// relForFileTree 取相对所属工作区文件夹的 slash 路径（多根：找包含它的那个根）。
func relForFileTree(p string) string {
	for _, r := range theFileTree.roots {
		if rel, err := filepath.Rel(r.path, p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	if rel, err := filepath.Rel(currentRoot(), p); err == nil {
		return filepath.ToSlash(rel)
	}
	return p
}

// fileNodeMenuItems 文件/文件夹节点右键菜单项（拆出便于测试）。
func fileNodeMenuItems(n *fileNode) []widget.MenuItem {
	var items []widget.MenuItem
	if n.isDir {
		ic, lbl := "chevron-right", "展开"
		if n.expanded {
			ic, lbl = "chevron-down", "折叠"
		}
		items = append(items, mi(ic, lbl, func() { theFileTree.toggle(n) }))
	} else {
		items = append(items, mi("file-code", "打开", func() { theEditor.open(n.path); theFileTree.selectPath(n.path) }))
	}
	return append(items,
		sep(),
		mi("plus", "新建文件", func() { newEntryIn(dirOf(n), false) }),
		mi("folder-plus", "新建文件夹", func() { newEntryIn(dirOf(n), true) }),
		sep(),
		mi("square-pen", "重命名", func() { renameEntry(n) }),
		miD("trash-2", "删除", func() { deleteEntry(n) }),
		sep(),
		mi("copy", "复制名称", func() { copyToClipboard(n.name) }),
		mi("copy", "复制相对路径", func() { copyToClipboard(relForFileTree(n.path)) }),
		mi("copy", "复制绝对路径", func() { copyToClipboard(n.path) }),
		sep(),
		mi("message-square", "添加到对话", func() {
			pfx := "参考文件："
			if n.isDir {
				pfx = "参考目录："
			}
			addToChat(pfx + relForFileTree(n.path))
		}),
		mi("terminal", "在终端打开", func() { theTerminal.openDir(dirOf(n)) }),
		mi("folder-open", "在资源管理器中打开", func() { revealInExplorer(n.path, n.isDir) }),
		mi("refresh-cw", "刷新", func() { theFileTree.refresh() }),
	)
}

func fileNodeMenu(x, y float64, n *fileNode) { showMenu(x, y, fileNodeMenuItems(n)) }

// fileTreeEmptyItems 文件树空白处右键菜单项（根目录操作）。
func fileTreeEmptyItems() []widget.MenuItem {
	root := currentRoot()
	return []widget.MenuItem{
		mi("plus", "新建文件", func() { newEntryIn(root, false) }),
		mi("folder-plus", "新建文件夹", func() { newEntryIn(root, true) }),
		sep(),
		mi("refresh-cw", "刷新", func() { theFileTree.refresh() }),
		mi("folder-open", "在资源管理器中打开", func() { revealInExplorer(root, true) }),
		mi("terminal", "在终端打开", func() { theTerminal.openDir(root) }),
	}
}

func fileTreeEmptyMenu(x, y float64) { showMenu(x, y, fileTreeEmptyItems()) }

// newEntryIn 在 dir 下新建文件/文件夹（弹输入框取名）。
func newEntryIn(dir string, isDir bool) {
	title := "新建文件"
	if isDir {
		title = "新建文件夹"
	}
	showPrompt(title, "", func(name string) {
		p := filepath.Join(dir, name)
		var err error
		if isDir {
			err = os.MkdirAll(p, 0o755)
		} else {
			err = os.WriteFile(p, nil, 0o644)
		}
		if err != nil {
			widget.ShowAlert("出错", err.Error(), widget.MsgWarning, nil)
			return
		}
		theFileTree.refresh()
		if !isDir {
			theEditor.open(p)
		}
	})
}

// renameEntry 重命名节点（弹输入框）。
func renameEntry(n *fileNode) {
	showPrompt("重命名", n.name, func(name string) {
		np := filepath.Join(filepath.Dir(n.path), name)
		if err := os.Rename(n.path, np); err != nil {
			widget.ShowAlert("出错", err.Error(), widget.MsgWarning, nil)
			return
		}
		theFileTree.refresh()
	})
}

// deleteEntry 删除节点（先确认）。
func deleteEntry(n *fileNode) {
	kind := "文件"
	if n.isDir {
		kind = "文件夹"
	}
	widget.ShowConfirm("删除"+kind, "确定删除"+kind+"「"+n.name+"」？此操作不可撤销。", widget.MsgWarning,
		func() {
			if err := os.RemoveAll(n.path); err != nil {
				widget.ShowAlert("出错", err.Error(), widget.MsgWarning, nil)
				return
			}
			theFileTree.refresh()
		}, nil)
}

// ─── 编辑器标签菜单 ───────────────────────────────────────

func editorTabItems(i int) []widget.MenuItem {
	return []widget.MenuItem{
		mi("x", "关闭", func() { theEditor.close(i) }),
		mi("", "关闭其他", func() { theEditor.closeOthers(i) }),
		mi("", "关闭所有", func() { theEditor.closeAll() }),
		sep(),
		mi("copy", "复制路径", func() {
			if t := theEditor.tabAt(i); t != nil {
				copyToClipboard(t.path)
			}
		}),
		mi("copy", "复制目录路径", func() {
			if t := theEditor.tabAt(i); t != nil {
				copyToClipboard(filepath.Dir(t.path))
			}
		}),
		sep(),
		mi("message-square", "添加到对话", func() {
			if t := theEditor.tabAt(i); t != nil {
				addToChat("参考文件：" + relForFileTree(t.path))
			}
		}),
	}
}

func editorTabMenu(x, y float64, i int) { showMenu(x, y, editorTabItems(i)) }

// ─── 终端菜单 ─────────────────────────────────────────────

func terminalItems() []widget.MenuItem {
	return []widget.MenuItem{
		mi("copy", "复制全部", func() { copyToClipboard(theTerminal.copyAll()) }),
		mi("clipboard", "粘贴", func() { theTerminal.pasteToInput() }),
		sep(),
		mi("message-square", "添加到对话", func() { addToChat("```\n" + theTerminal.copyAll() + "```") }),
		mi("eraser", "清屏", func() { theTerminal.clearScreen() }),
	}
}

func terminalMenu(x, y float64) { showMenu(x, y, terminalItems()) }

// ─── 编辑器内容菜单（代码编辑器/结构化工作台的右键自定义菜单，取代组件自带）─────────

func editorContentItems() []widget.MenuItem {
	codeEd := widget.HasFocusedEditor()        // 代码视图：聚焦 CodeEditor
	tableEd := widget.HasFocusedStructEditor() // 表格视图：聚焦 StructEditor
	en := codeEd || tableEd
	// 复制/剪切/粘贴：代码视图走 RunEditorCommand，表格视图走选中单元格操作。
	copyFn := func() {
		if codeEd {
			widget.RunEditorCommand("copy")
		} else if s := widget.StructEditorCopyCell(); s != "" {
			copyToClipboard(s)
		}
	}
	cutFn := func() {
		if codeEd {
			widget.RunEditorCommand("cut")
		} else if s := widget.StructEditorCopyCell(); s != "" {
			copyToClipboard(s)
			widget.StructEditorPasteCell("")
		}
	}
	pasteFn := func() {
		if codeEd {
			widget.RunEditorCommand("paste")
		} else if widget.ClipboardRead != nil {
			widget.StructEditorPasteCell(widget.ClipboardRead())
		}
	}
	items := []widget.MenuItem{
		{Icon: "undo-2", Label: "撤销", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("undo") }, Shortcut: "Ctrl+Z"},
		{Icon: "redo-2", Label: "重做", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("redo") }, Shortcut: "Ctrl+Y"},
		sep(),
		{Icon: "scissors", Label: "剪切", Enabled: en, OnClick: cutFn, Shortcut: "Ctrl+X"},
		{Icon: "copy", Label: "复制", Enabled: en, OnClick: copyFn, Shortcut: "Ctrl+C"},
		{Icon: "clipboard", Label: "粘贴", Enabled: en, OnClick: pasteFn, Shortcut: "Ctrl+V"},
		sep(),
		{Icon: "list", Label: "全选", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("selectAll") }, Shortcut: "Ctrl+A"},
		{Icon: "braces", Label: "格式化文档", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("format") }},
	}
	// 自动换行（切换；开启时左侧显示对勾，作勾选标记）——对照参考补齐。
	wrapIcon := ""
	if widget.EditorWrapEnabled() {
		wrapIcon = "check"
	}
	items = append(items, sep(), widget.MenuItem{Icon: wrapIcon, Label: "自动换行", Enabled: codeEd, OnClick: func() { widget.RunEditorCommand("toggleWrap") }})
	// 文件操作（对照参考补齐：在资源管理器中显示 / 复制路径 / 复制文件名）
	if t := theEditor.activeTab(); t != nil {
		items = append(items, sep(),
			mi("folder-open", "在资源管理器中显示", func() { revealInExplorer(t.path, false) }),
			mi("copy", "复制路径", func() { copyToClipboard(t.path) }),
			mi("file", "复制文件名", func() { copyToClipboard(filepath.Base(t.path)) }),
		)
	}
	// 添加到对话：有选中代码→加代码块；否则加文件引用。
	items = append(items, sep(), mi("message-square", "添加到对话", func() {
		if sel := widget.FocusedEditorSelection(); sel != "" {
			addToChat("```\n" + sel + "\n```")
		} else if t := theEditor.activeTab(); t != nil {
			addToChat("参考文件：" + relForFileTree(t.path))
		}
	}))
	// 结构化语言（有 LanguageProvider）→ 追加「代码⇄表格」视图切换。
	if t := theEditor.activeTab(); t != nil && widget.HasProvider(t.lang) {
		icon, label := "code", "切换到代码视图"
		if widget.WorkbenchModeIsText() {
			icon, label = "table-2", "切换到表格视图"
		}
		items = append(items, sep(), mi(icon, label, widget.ToggleWorkbenchView))
	}
	return items
}

func editorContentMenu(x, y float64) { showMenu(x, y, editorContentItems()) }
