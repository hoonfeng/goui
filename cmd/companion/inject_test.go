//go:build windows

// 验证哲学/Skills 注入系统提示。用 t.TempDir 作工作区根，绝不碰真实 .pair。
package main

import (
	"strings"
	"testing"
)

func TestPhilosophyPrompt(t *testing.T) {
	saved := theSettings
	defer func() { theSettings = saved }()
	theSettings = defaultSettings()

	theSettings.PhilosophyEnabled = false
	if philosophyPrompt() != "" {
		t.Fatal("未启用时应为空")
	}
	theSettings.PhilosophyEnabled = true
	theSettings.PhilosophySelected = []string{"tao-te-ching", "sunzi-bingfa"}
	theSettings.PhilosophyRoles = map[string]string{"general": "保持简洁务实"}
	p := philosophyPrompt()
	for _, want := range []string{"道德经", "孙子兵法", "保持简洁务实"} {
		if !strings.Contains(p, want) {
			t.Fatalf("哲学提示缺少 %q：%s", want, p)
		}
	}
	if strings.Contains(p, "黄帝阴符经") {
		t.Fatal("未选中的经典不应出现")
	}
}

func TestSkillsPrompt(t *testing.T) {
	savedWS := workspaceFolders
	defer func() { workspaceFolders = savedWS }()
	workspaceFolders = []string{t.TempDir()} // 隔离工作区根 → skillsRoot 落在 temp/.pair/skills（t.TempDir 自动清理）

	if skillsPrompt() != "" {
		t.Fatal("无技能时应为空")
	}
	if err := writeSkill(skillEntry{Name: "always-rule", Mode: "always", Description: "始终规范", Content: "务必遵守 X"}); err != nil {
		t.Fatal(err)
	}
	if err := writeSkill(skillEntry{Name: "auto-helper", Mode: "auto", Description: "按需帮手"}); err != nil {
		t.Fatal(err)
	}
	if err := writeSkill(skillEntry{Name: "manual-only", Mode: "manual", Description: "手动"}); err != nil {
		t.Fatal(err)
	}
	p := skillsPrompt()
	for _, want := range []string{"可用技能", "always-rule", "务必遵守 X", "auto-helper", "按需帮手"} {
		if !strings.Contains(p, want) {
			t.Fatalf("技能提示缺少 %q：%s", want, p)
		}
	}
	if strings.Contains(p, "manual-only") {
		t.Fatal("手动技能不应注入")
	}
}
