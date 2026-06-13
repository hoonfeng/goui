// Package tools 提供基础工具函数，供 Agent 调用。
package tools

import (
	"fmt"
	"os"
	"strings"
)

// ReadFile 读取文件内容。path 为工作区内路径。
// offset 可选起始行号(1基)，limit 可选读取行数；省略则读全文（超2000行截断提示翻页）。
func ReadFile(path string, offset, limit int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	if offset <= 0 && limit <= 0 {
		lines := strings.Split(string(data), "\n")
		if len(lines) > 2000 {
			return strings.Join(lines[:2000], "\n") + fmt.Sprintf("\n…[文件共 %d 行，仅显示前 2000；用 offset/limit 读其余]", len(lines)), nil
		}
		return string(data), nil
	}
	lines := strings.Split(string(data), "\n")
	start := offset - 1
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		return "", fmt.Errorf("offset %d 超出文件行数 %d", offset, len(lines))
	}
	end := len(lines)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return strings.Join(lines[start:end], "\n"), nil
}

// writeFile 内部写文件函数（包内共享）。
func writeFile(path, content string) error {
	dir := path
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			dir = path[:i]
			break
		}
	}
	if dir != path {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
