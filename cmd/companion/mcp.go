package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/user/goui/cmd/companion/agent"
)

// mcpEntry 一个 MCP 服务器配置。
type mcpEntry struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// MCP 三级：系统（内置只读）/ 用户（全局）/ 项目（.pair）。
const (
	mcpLevelSystem  = "system"
	mcpLevelUser    = "user"
	mcpLevelProject = "project"
)

var mcpLevels = []struct{ id, name string }{
	{mcpLevelSystem, "系统级"}, {mcpLevelUser, "用户级"}, {mcpLevelProject, "项目级"},
}

// systemMCPDefaults 内置（系统级）MCP 服务器，只读。对齐参考 DEFAULTS.mcpServers。
func systemMCPDefaults() []mcpEntry {
	return []mcpEntry{
		{Name: "filesystem", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem"}},
		{Name: "git", Command: "uvx", Args: []string{"mcp-server-git"}},
		{Name: "fetch", Command: "uvx", Args: []string{"mcp-server-fetch"}},
		{Name: "memory", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-memory"}},
	}
}

// mcpPathFor 用户/项目级 mcp.json 路径（系统级无文件→""）。
func mcpPathFor(level string) string {
	switch level {
	case mcpLevelUser:
		return filepath.Join(configDir(), "mcp.json")
	case mcpLevelProject:
		return filepath.Join(currentRoot(), ".pair", "mcp.json")
	}
	return ""
}

type mcpFile struct {
	MCPServers map[string]struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env,omitempty"`
	} `json:"mcpServers"`
}

func readMCPFile(path string) []mcpEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f mcpFile
	if json.Unmarshal(data, &f) != nil {
		return nil
	}
	var out []mcpEntry
	for name, s := range f.MCPServers {
		out = append(out, mcpEntry{Name: name, Command: s.Command, Args: s.Args, Env: s.Env})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// readMCPLevel 读某一级的服务器（系统=内置，用户/项目=各自 mcp.json）。
func readMCPLevel(level string) []mcpEntry {
	if level == mcpLevelSystem {
		return systemMCPDefaults()
	}
	return readMCPFile(mcpPathFor(level))
}

func writeMCPFile(path string, es []mcpEntry) error {
	m := map[string]any{}
	for _, e := range es {
		if e.Name == "" {
			continue
		}
		entry := map[string]any{"command": e.Command}
		if len(e.Args) > 0 {
			entry["args"] = e.Args
		}
		if len(e.Env) > 0 {
			entry["env"] = e.Env
		}
		m[e.Name] = entry
	}
	data, err := json.MarshalIndent(map[string]any{"mcpServers": m}, "", "  ")
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return os.WriteFile(path, data, 0o644)
}

// upsertMCPEntry 新增/更新某一级的服务器（系统级只读→忽略）。
func upsertMCPEntry(level string, e mcpEntry) error {
	path := mcpPathFor(level)
	if path == "" {
		return nil
	}
	es := readMCPFile(path)
	found := false
	for i := range es {
		if es[i].Name == e.Name {
			es[i] = e
			found = true
			break
		}
	}
	if !found {
		es = append(es, e)
	}
	return writeMCPFile(path, es)
}

// deleteMCPEntry 删除某一级的服务器（系统级只读→忽略）。
func deleteMCPEntry(level, name string) error {
	path := mcpPathFor(level)
	if path == "" {
		return nil
	}
	es := readMCPFile(path)
	out := es[:0:0]
	for _, e := range es {
		if e.Name != name {
			out = append(out, e)
		}
	}
	return writeMCPFile(path, out)
}

// mcpEnabled 是否启用：override 优先；默认系统级仅 filesystem 开、用户/项目级全开。
func mcpEnabled(level, name string) bool {
	if v, ok := theSettings.MCPEnabledOverrides[level+"::"+name]; ok {
		return v
	}
	if level == mcpLevelSystem {
		return name == "filesystem"
	}
	return true
}

// setMCPEnabled 改某服务器启用态（存 override map）。
func setMCPEnabled(level, name string, on bool) {
	if theSettings.MCPEnabledOverrides == nil {
		theSettings.MCPEnabledOverrides = map[string]bool{}
	}
	theSettings.MCPEnabledOverrides[level+"::"+name] = on
	saveSettings()
}

// loadMCPConfigs 合并三级（项目>用户>系统，同名去重）、按启用过滤，供 agent 连接；自动连接关→不连。
func loadMCPConfigs() []agent.MCPServerConfig {
	if !theSettings.AutoConnectMCP {
		return nil
	}
	var out []agent.MCPServerConfig
	seen := map[string]bool{}
	for _, lv := range []string{mcpLevelProject, mcpLevelUser, mcpLevelSystem} {
		for _, e := range readMCPLevel(lv) {
			if e.Command == "" || seen[e.Name] || !mcpEnabled(lv, e.Name) {
				continue
			}
			seen[e.Name] = true
			out = append(out, agent.MCPServerConfig{Name: e.Name, Command: e.Command, Args: e.Args, Env: e.Env})
		}
	}
	return out
}
