package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile 把 content 完整写入 path（覆盖；父目录自动创建）。
func WriteFile(path, content string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("创建父目录失败: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return fmt.Sprintf("已写入 %s（%d 字节）", path, len(content)), nil
}
