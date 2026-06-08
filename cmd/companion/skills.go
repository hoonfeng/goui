// Skills 子系统：每个 skill = .pair/skills/<名>/SKILL.md（带 frontmatter）。
// 读取/解析/写入/删除，供 Skills 设置 tab 管理。实际激活注入 agent 后续接入。

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

func skillsRoot() string { return filepath.Join(currentRoot(), ".pair", "skills") }

// readSkills 读 .pair/skills/*/SKILL.md → 按名排序的 skill 列表（无目录→nil）。
func readSkills() []skillEntry {
	ents, err := os.ReadDir(skillsRoot())
	if err != nil {
		return nil
	}
	var out []skillEntry
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		md, err := os.ReadFile(filepath.Join(skillsRoot(), e.Name(), "SKILL.md"))
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

// writeSkill 写 .pair/skills/<名>/SKILL.md（带 frontmatter）。
func writeSkill(s skillEntry) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("名称必填")
	}
	dir := filepath.Join(skillsRoot(), s.Name)
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

// deleteSkill 删除整个 skill 目录。
func deleteSkill(name string) error { return os.RemoveAll(filepath.Join(skillsRoot(), name)) }

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

// skillsPrompt 把可用技能拼进 agent 系统提示：always→全文（始终遵循）、auto→名称+描述（按相关性自取）、manual→跳过。
func skillsPrompt() string {
	skills := readSkills()
	var body strings.Builder
	for _, s := range skills {
		switch s.Mode {
		case "manual":
			continue // 手动激活，不自动注入
		case "always":
			body.WriteString("\n\n## 技能：" + s.Name + "（始终遵循）\n")
			if s.Description != "" {
				body.WriteString(s.Description + "\n")
			}
			body.WriteString(strings.TrimSpace(s.Content))
		default: // auto / 空
			body.WriteString("\n- 「" + s.Name + "」：" + s.Description)
		}
	}
	if body.Len() == 0 {
		return ""
	}
	return "\n\n# 可用技能（Skills）\n按需运用以下技能（标「始终遵循」的务必遵循，其余按相关性自行采用）：" + body.String()
}
