// Skills 子系统（三级：系统内置只读 / 用户全局 / 项目 .pair）。
// 每个 skill = <级目录>/<名>/SKILL.md（带 frontmatter）。读取/解析/写入/删除 + 启用态，供 Skills 设置 tab 管理。

//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// skillEntry 一个 Skill（SKILL.md 的 frontmatter + 正文）。
type skillEntry struct {
	Name        string
	Description string
	Mode        string // auto / manual / always
	Globs       string
	Tools       string
	Content     string
}

// Skills 三级复用通用层级常量（system/user/project，定义在 mcp.go）。
var skillLevels = []struct{ id, name string }{
	{mcpLevelSystem, "系统级"}, {mcpLevelUser, "用户级"}, {mcpLevelProject, "项目级"},
}

// skillsRootFor 某级 skills 目录（用户数据目录=安装目录，故系统级与用户级都在安装目录 config 下）：
// 系统级=config/skills（shipped，UI 只读、文件管理）；用户级=config/user-skills（可增删改）；
// 项目级=工作区 .pair/skills。
func skillsRootFor(level string) string {
	switch level {
	case mcpLevelSystem:
		return filepath.Join(configDir(), "skills")
	case mcpLevelUser:
		return filepath.Join(configDir(), "user-skills")
	case mcpLevelProject:
		return filepath.Join(currentRoot(), ".pair", "skills")
	}
	return ""
}

// readSkillsLevel 读某级 skills 目录（目录缺失→nil）。
func readSkillsLevel(level string) []skillEntry {
	root := skillsRootFor(level)
	if root == "" {
		return nil
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []skillEntry
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		md, err := os.ReadFile(filepath.Join(root, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		s := parseSkillMD(string(md))
		if s.Name == "" {
			s.Name = e.Name()
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// parseSkillMD 解析 SKILL.md：--- frontmatter --- 的 key: value + 正文。
func parseSkillMD(md string) skillEntry {
	var s skillEntry
	md = strings.ReplaceAll(md, "\r\n", "\n")
	if strings.HasPrefix(md, "---\n") {
		if end := strings.Index(md[4:], "\n---"); end >= 0 {
			fm := md[4 : 4+end]
			s.Content = strings.TrimPrefix(md[4+end+4:], "\n")
			for _, line := range strings.Split(fm, "\n") {
				k, v, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				k, v = strings.TrimSpace(k), strings.TrimSpace(v)
				switch k {
				case "name":
					s.Name = v
				case "description":
					s.Description = v
				case "mode", "activation":
					s.Mode = v
				case "globs":
					s.Globs = v
				case "tools", "allowed-tools":
					s.Tools = v
				}
			}
			return s
		}
	}
	s.Content = md
	return s
}

// writeSkill 写某级 <名>/SKILL.md（带 frontmatter）。系统级只读→忽略。
func writeSkill(level string, s skillEntry) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("名称必填")
	}
	root := skillsRootFor(level)
	if root == "" {
		return nil
	}
	dir := filepath.Join(root, s.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: " + s.Name + "\n")
	if s.Description != "" {
		b.WriteString("description: " + s.Description + "\n")
	}
	if s.Mode != "" {
		b.WriteString("mode: " + s.Mode + "\n")
	}
	if s.Globs != "" {
		b.WriteString("globs: " + s.Globs + "\n")
	}
	if s.Tools != "" {
		b.WriteString("tools: " + s.Tools + "\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(s.Content)
	return os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(b.String()), 0o644)
}

// deleteSkill 删除某级的 skill 目录。系统级只读→忽略。
func deleteSkill(level, name string) error {
	root := skillsRootFor(level)
	if root == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(root, name))
}

// skillEnabled 某 skill 是否启用：override 优先，默认全开。
func skillEnabled(level, name string) bool {
	if v, ok := theSettings.SkillEnabledOverrides[level+"::"+name]; ok {
		return v
	}
	return true
}

// setSkillEnabled 改 skill 启用态（存 override map）。
func setSkillEnabled(level, name string, on bool) {
	if theSettings.SkillEnabledOverrides == nil {
		theSettings.SkillEnabledOverrides = map[string]bool{}
	}
	theSettings.SkillEnabledOverrides[level+"::"+name] = on
	saveSettings()
}

// skillModeLabel 激活模式中文名。
func skillModeLabel(m string) string {
	switch m {
	case "auto":
		return "自动"
	case "manual":
		return "手动"
	case "always":
		return "始终"
	}
	return m
}

// skillsPrompt 合并三级（项目>用户>系统同名去重、按启用过滤）拼进 agent 系统提示：
// always→全文（始终遵循）、auto→名称+描述（按需）、manual→跳过。
func skillsPrompt() string {
	var body strings.Builder
	seen := map[string]bool{}
	for _, lv := range []string{mcpLevelProject, mcpLevelUser, mcpLevelSystem} {
		for _, s := range readSkillsLevel(lv) {
			if seen[s.Name] || !skillEnabled(lv, s.Name) {
				continue
			}
			seen[s.Name] = true
			switch s.Mode {
			case "manual":
				continue
			case "always":
				body.WriteString("\n\n## 技能：" + s.Name + "（始终遵循）\n")
				if s.Description != "" {
					body.WriteString(s.Description + "\n")
				}
				body.WriteString(strings.TrimSpace(s.Content))
			default:
				body.WriteString("\n- 「" + s.Name + "」：" + s.Description)
			}
		}
	}
	if body.Len() == 0 {
		return ""
	}
	return "\n\n# 可用技能（Skills）\n按需运用以下技能（标「始终遵循」的务必遵循，其余按相关性自行采用）：" + body.String()
}
