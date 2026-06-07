package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/user/goui/cmd/companion/agent"
)

// loadMCPConfigs 读 <配置目录>/mcp.json（格式同 Claude Desktop：
// {"mcpServers":{"名字":{"command":"...","args":[...],"env":{...}}}}）。无文件→nil。
func loadMCPConfigs() []agent.MCPServerConfig {
	path := filepath.Join(filepath.Dir(settingsPath()), "mcp.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f struct {
		MCPServers map[string]struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if json.Unmarshal(data, &f) != nil {
		return nil
	}
	var out []agent.MCPServerConfig
	for name, s := range f.MCPServers {
		if s.Command == "" {
			continue
		}
		out = append(out, agent.MCPServerConfig{Name: name, Command: s.Command, Args: s.Args, Env: s.Env})
	}
	return out
}
