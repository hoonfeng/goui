package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestPoolReuse 共享池：同(服务器+工作区)复用同一 Client（非单例但共享，不重启）；CloseAll 全关。
func TestPoolReuse(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls 不在 PATH，跳过")
	}
	defer CloseAll()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n\ngo 1.21\n"), 0o644)
	ws := "file:///" + filepath.ToSlash(dir)

	c1, err := Acquire("gopls", nil, ws)
	if err != nil || c1 == nil {
		t.Skip("gopls 启动失败，跳过")
	}
	c2, _ := Acquire("gopls", nil, ws)
	if c1 != c2 {
		t.Error("同 server+workspace 两次 Acquire 应复用同一 Client（共享、不重启）")
	}
	if n := PoolSize(); n != 1 {
		t.Errorf("应只 1 个共享 server，得 %d", n)
	}
	CloseAll()
	if n := PoolSize(); n != 0 {
		t.Errorf("CloseAll 后应 0，得 %d", n)
	}
}
