package tools

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser 在系统默认浏览器中打开指定 URL。
// 支持 Windows / macOS / Linux 三平台。
func OpenBrowser(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("url 不能为空")
	}

	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("cmd", "/C", "start", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default: // linux
		err = exec.Command("xdg-open", url).Start()
	}

	if err != nil {
		return "", fmt.Errorf("打开浏览器失败: %w", err)
	}
	return fmt.Sprintf("已打开浏览器: %s", url), nil
}
