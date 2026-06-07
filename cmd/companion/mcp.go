package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/user/goui/cmd/companion/agent"
)

// mcpEntry 一个 MCP 服务器配置（mcp.json 一项）。
type mcpEntry struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

func mcpFilePath() string { return filepath.Join(filepath.Dir(settingsPath()), "mcp.json") }

// mcpFile mcp.json 结构（同 Claude Desktop）。
type mcpFile struct {
	MCPServers map[string]struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env,omitempty"`
	} `json:"mcpServers"`
}

// readMCPEntries 读 mcp.json → 按名排序的服务器列表（无文件→nil）。
func readMCPEntries() []mcpEntry {
	data, err := os.ReadFile(mcpFilePath())
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

// writeMCPEntries 写回 mcp.json（{"mcpServers":{...}}）。
func writeMCPEntries(es []mcpEntry) error {
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
	_ = os.MkdirAll(filepath.Dir(mcpFilePath()), 0o755)
	return os.WriteFile(mcpFilePath(), data, 0o644)
}

// upsertMCPEntry 新增/更新一个服务器（按 Name 匹配）。
func upsertMCPEntry(e mcpEntry) error {
	es := readMCPEntries()
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
	return writeMCPEntries(es)
}

// deleteMCPEntry 删除一个服务器。
func deleteMCPEntry(name string) error {
	es := readMCPEntries()
	out := es[:0:0]
	for _, e := range es {
		if e.Name != name {
			out = append(out, e)
		}
	}
	return writeMCPEntries(out)
}

// loadMCPConfigs 对话开始时读取要连接的 MCP 服务器；自动连接关闭→不连。
func loadMCPConfigs() []agent.MCPServerConfig {
	if !theSettings.AutoConnectMCP {
		return nil
	}
	var out []agent.MCPServerConfig
	for _, e := range readMCPEntries() {
		if e.Command == "" {
			continue
		}
		out = append(out, agent.MCPServerConfig{Name: e.Name, Command: e.Command, Args: e.Args, Env: e.Env})
	}
	return out
}
