// 文件菜单里工作区相关命令的实现：新建项目 / 保存工作区 / 管理工作区 / 关闭项目 / 关闭工作区。

//go:build windows

package main

import (
	"os"
	"path/filepath"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// newProjectViaDialog 新建项目：选父目录 → 输项目名 → 建文件夹 → 作为工作区打开。
func newProjectViaDialog() {
	parent := pickFolder("选择新项目的父目录")
	if parent == "" {
		return
	}
	showPrompt("新建项目", "", func(name string) {
		dir := filepath.Join(parent, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			widget.MessageError("创建失败：" + err.Error())
			return
		}
		setProject(dir)
		widget.MessageSuccess("已创建并打开项目「" + name + "」")
	})
}

// saveWorkspaceMenu 保存工作区：把文件夹列表写到主文件夹的 .pair/workspace.json（可入库/分享）。
func saveWorkspaceMenu() {
	if err := saveWorkspaceFile(); err != nil {
		widget.MessageError("保存工作区失败：" + err.Error())
		return
	}
	widget.MessageSuccess("工作区已保存到 .pair/workspace.json")
}

// closeProjectMenu 关闭当前项目（移除主文件夹）。
func closeProjectMenu() {
	if len(workspaceFolders) == 0 {
		widget.MessageInfo("当前没有打开的项目")
		return
	}
	closeProject()
	widget.MessageInfo("已关闭项目")
}

// closeWorkspaceMenu 关闭整个工作区（确认后清空所有文件夹）。
func closeWorkspaceMenu() {
	if len(workspaceFolders) == 0 {
		widget.MessageInfo("工作区已是空的")
		return
	}
	widget.ShowConfirm("关闭工作区", "确定关闭整个工作区（移除全部文件夹）？", widget.MsgWarning,
		func() { clearWorkspace(); widget.MessageInfo("已关闭工作区") }, nil)
}

// ─── 管理工作区对话框 ───────────────────────────────────────
var theWSManager = &wsManagerState{}

type WSManagerBody struct{ widget.StatefulWidget }

func (w *WSManagerBody) CreateState() widget.State { return theWSManager }

type wsManagerState struct{ widget.BaseState }

func showWorkspaceManager() {
	var id int
	dlg := widget.NewDialog("管理工作区文件夹", &WSManagerBody{}).WithWidth(520).WithTransition("fade").WithFooter(
		widget.NewButton("关闭", func() { widget.HideOverlay(id) }).WithColor(*ghBgTertiary).WithTextColor(ghText),
	)
	id = widget.ShowDialog(dlg)
}

func (b *wsManagerState) Build(ctx widget.BuildContext) widget.Widget {
	rows := []widget.Widget{}
	if len(workspaceFolders) == 0 {
		rows = append(rows, widget.Div(widget.Style{Padding: types.EdgeInsets(10)},
			label("（空工作区——点下方「添加文件夹」）", ghTextMuted, 12)))
	}
	for i, f := range workspaceFolders {
		rows = append(rows, b.folderRow(i, f))
	}
	rows = append(rows,
		widget.Div(widget.Style{Height: 12}),
		&widget.Button{
			SingleChildWidget: widget.SingleChildWidget{Child: label("+ 添加文件夹到工作区", cWhite, 12)},
			OnClick:           func() { addFolderViaDialog(); b.SetState() },
			Color:             *ghAccentEmph, MinHeight: 30, Padding: types.EdgeInsetsLTRB(12, 0, 12, 0),
		},
		widget.Div(widget.Style{Height: 4}),
		label("提示：首个文件夹为 Agent 主文件夹；可「设为首选」调整，或在文件树里拖拽排序。", ghTextMuted, 10),
	)
	return widget.Div(widget.Style{Width: 488, FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(2, 2, 2, 2)}, rows)
}

func (b *wsManagerState) folderRow(i int, f string) widget.Widget {
	kids := []widget.Widget{
		widget.Lucide("folder", widget.IconSize(14), widget.IconColor(cText)),
		widget.Div(widget.Style{Width: 6}),
		expand(label1(filepath.Base(f), cText, 12)),
	}
	if i == 0 {
		kids = append(kids,
			widget.Lucide("star", widget.IconSize(12), widget.IconColor(types.ColorFromRGB(229, 192, 123))),
			widget.Div(widget.Style{Width: 4}),
			label("主", types.ColorFromRGB(229, 192, 123), 11),
			widget.Div(widget.Style{Width: 8}))
	} else {
		kids = append(kids,
			&widget.Button{
				SingleChildWidget: widget.SingleChildWidget{Child: label("设为首选", ghText, 11)},
				OnClick:           func() { setPrimaryFolder(f); b.SetState() },
				Color:             *ghBgTertiary, MinHeight: 24, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0),
			},
			widget.Div(widget.Style{Width: 6}))
	}
	kids = append(kids, &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: label("移除", types.ColorFromRGB(240, 120, 110), 11)},
		OnClick:           func() { removeFolder(f); b.SetState() },
		Color:             *ghBgTertiary, MinHeight: 24, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0),
	})
	return widget.Div(widget.Style{Height: 36, FlexDirection: "row", AlignItems: "center",
		Padding: types.EdgeInsetsLTRB(8, 0, 8, 0)}, kids)
}
