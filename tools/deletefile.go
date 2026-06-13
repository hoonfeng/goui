package tools

import (
	"fmt"
	"os"
)

// DeleteFile 删除一个文件（不可恢复，谨慎）。
func DeleteFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("文件不存在: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("delete_file 不删目录：%s", path)
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("删除失败: %w", err)
	}
	return fmt.Sprintf("已删除 %s", path), nil
}
