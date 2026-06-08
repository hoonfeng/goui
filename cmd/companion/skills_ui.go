// Skill 添加/编辑对话框（写回 .pair/skills/<名>/SKILL.md）。

//go:build windows

package main

import (
	"strings"

	"github.com/user/goui/internal/widget"
)

var theSkillEditor = &skillEditorState{}

// SkillEditorBody Skill 编辑对话框主体。
type SkillEditorBody struct{ widget.StatefulWidget }

func (m *SkillEditorBody) CreateState() widget.State { return theSkillEditor }

type skillEditorState struct {
	widget.BaseState
	orig    string
	name    string
	mode    string
	desc    string
	globs   string
	tools   string
	content string
	tok     int
}

// openSkillEditor 打开「添加/编辑 Skill」对话框；保存写回 SKILL.md 后回调刷新。
func openSkillEditor(s skillEntry, onSaved func()) {
	theSkillEditor.orig = s.Name
	theSkillEditor.name = s.Name
	theSkillEditor.mode = s.Mode
	if theSkillEditor.mode == "" {
		theSkillEditor.mode = "auto"
	}
	theSkillEditor.desc = s.Description
	theSkillEditor.globs = s.Globs
	theSkillEditor.tools = s.Tools
	theSkillEditor.content = s.Content
	theSkillEditor.tok++
	title := "添加 Skill"
	if s.Name != "" {
		title = "编辑 Skill"
	}
	var id int
	dlg := widget.NewDialog(title, &SkillEditorBody{}).WithWidth(520).WithTransition("fade").WithFooter(
		widget.NewButton("取消", func() { widget.HideOverlay(id) }).WithColor(*ghBgTertiary).WithTextColor(ghText),
		widget.NewButton("保存", func() {
			name := strings.TrimSpace(theSkillEditor.name)
			if name == "" {
				widget.MessageWarning("请填写 Skill 名称")
				return
			}
			if theSkillEditor.orig != "" && theSkillEditor.orig != name { // 改名→删旧
				_ = deleteSkill(theSkillEditor.orig)
			}
			if err := writeSkill(skillEntry{Name: name, Mode: theSkillEditor.mode, Description: theSkillEditor.desc,
				Globs: theSkillEditor.globs, Tools: theSkillEditor.tools, Content: theSkillEditor.content}); err != nil {
				widget.MessageError("保存失败：" + err.Error())
				return
			}
			if onSaved != nil {
				onSaved()
			}
			widget.MessageSuccess("已保存 Skill「" + name + "」")
			widget.HideOverlay(id)
		}).WithColor(*ghAccentEmph).WithTextColor(cWhite),
	)
	id = widget.ShowDialog(dlg)
}

func (b *skillEditorState) Build(ctx widget.BuildContext) widget.Widget {
	return widget.Div(widget.Style{Width: 488, FlexDirection: "column", AlignItems: "stretch"},
		settingsField("Skill 名称", settingsInput("如 my-custom-skill", b.name, b.tok, func(t string) { b.name = t })),
		settingsField("激活模式", settingsSelect(b.mode, []widget.SelectOption{
			{Label: "自动", Value: "auto"}, {Label: "手动", Value: "manual"}, {Label: "始终", Value: "always"},
		}, func(v string) { b.mode = v; b.SetState() })),
		settingsField("描述", settingsInput("简短描述此 Skill 的功能", b.desc, b.tok, func(t string) { b.desc = t })),
		settingsField("文件匹配 (globs)", settingsInput("*.ts *.tsx", b.globs, b.tok, func(t string) { b.globs = t })),
		settingsField("允许的工具", settingsInput("Read Write Edit", b.tools, b.tok, func(t string) { b.tools = t })),
		settingsField("SKILL.md 内容", settingsTextarea("# 技能说明\n描述触发条件与操作步骤…", b.content, 8, b.tok, func(t string) { b.content = t })),
	)
}
