package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListDir 列出目录下的文件/子目录（目录在前）。
// path 为目标目录，pattern 可选通配符过滤（如 *.go）。
func ListDir(path, pattern string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %w", err)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})
	var b strings.Builder
	for _, e := range entries {
		if pattern != "" && !e.IsDir() {
			if ok, _ := filepath.Match(pattern, e.Name()); !ok {
				continue
			}
		}
		if e.IsDir() {
			b.WriteString(e.Name() + "/\n")
		} else {
			sz := int64(-1)
			if fi, err := e.Info(); err == nil {
				sz = fi.Size()
			}
			fmt.Fprintf(&b, "%s\t%d\n", e.Name(), sz)
		}
	}
	if b.Len() == 0 {
		return "（空目录或无匹配）", nil
	}
	return b.String(), nil
}
