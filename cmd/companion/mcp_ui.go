// MCP 服务器添加/编辑对话框（写回 mcp.json）。

//go:build windows

package main

import (
	"strings"

	"github.com/user/goui/internal/widget"
)

var theMCPEditor = &mcpEditorState{}

// MCPEditorBody MCP 编辑对话框主体。
type MCPEditorBody struct{ widget.StatefulWidget }

func (m *MCPEditorBody) CreateState() widget.State { return theMCPEditor }

type mcpEditorState struct {
	widget.BaseState
	level   string // 写入的层级（user/project；系统级不可编辑）
	orig    string // 原名（编辑时判断是否改名）
	name    string
	command string
	args    string // 空格分隔
	tok     int
}

// openMCPEditor 打开「添加/编辑 MCP 服务器」对话框；保存写回该 level 的 mcp.json 后回调 onSaved 刷新。
func openMCPEditor(level string, e mcpEntry, onSaved func()) {
	theMCPEditor.level = level
	theMCPEditor.orig = e.Name
	theMCPEditor.name = e.Name
	theMCPEditor.command = e.Command
	theMCPEditor.args = strings.Join(e.Args, " ")
	theMCPEditor.tok++
	title := "添加 MCP 服务器"
	if e.Name != "" {
		title = "编辑 MCP 服务器"
	}
	var id int
	dlg := widget.NewDialog(title, &MCPEditorBody{}).WithWidth(460).WithTransition("fade").WithFooter(
		widget.NewButton("取消", func() { widget.HideOverlay(id) }).WithColor(*ghBgTertiary).WithTextColor(ghText),
		widget.NewButton("保存", func() {
			name := strings.TrimSpace(theMCPEditor.name)
			if name == "" {
				widget.MessageWarning("请填写服务器名称")
				return
			}
			cmd := strings.TrimSpace(theMCPEditor.command)
			if cmd == "" {
				widget.MessageWarning("请填写命令")
				return
			}
			if theMCPEditor.orig != "" && theMCPEditor.orig != name { // 改名→删旧增新
				_ = deleteMCPEntry(theMCPEditor.level, theMCPEditor.orig)
			}
			if err := upsertMCPEntry(theMCPEditor.level, mcpEntry{Name: name, Command: cmd, Args: strings.Fields(theMCPEditor.args)}); err != nil {
				widget.MessageError("保存失败：" + err.Error())
				return
			}
			if onSaved != nil {
				onSaved()
			}
			widget.MessageSuccess("已保存 MCP 服务器「" + name + "」（重开对话生效）")
			widget.HideOverlay(id)
		}).WithColor(*ghAccentEmph).WithTextColor(cWhite),
	)
	id = widget.ShowDialog(dlg)
}

func (b *mcpEditorState) Build(ctx widget.BuildContext) widget.Widget {
	return widget.Div(widget.Style{Width: 428, FlexDirection: "column", AlignItems: "stretch"},
		settingsField("名称", settingsInput("如 filesystem", b.name, b.tok, func(t string) { b.name = t })),
		settingsField("命令", settingsInput("npx / uvx / node", b.command, b.tok, func(t string) { b.command = t })),
		settingsField("参数（空格分隔）", settingsInput("-y @modelcontextprotocol/server-filesystem", b.args, b.tok, func(t string) { b.args = t })),
		widget.Div(widget.Style{Height: 4}),
		label("环境变量 / 传输协议（streamable-http）等高级项可在 mcp.json 手编。", ghTextMuted, 10),
	)
}
