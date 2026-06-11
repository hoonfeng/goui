// lsptest — 验证 LSP 客户端能接上 gopls 并拿到语义补全（端到端）。
// 运行: go run ./examples/lsptest/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoonfeng/goui/internal/lsp"
)

const goSrc = `package main

import "fmt"

func main() {
	fmt.
}
`

func pathToURI(p string) string {
	p = filepath.ToSlash(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // Windows: F:/... → /F:/...
	}
	return "file://" + p
}

func main() {
	// 1. 建临时 Go 模块
	dir, err := os.MkdirTemp("", "lsptest")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module lsptest\n\ngo 1.24\n"), 0644)
	mainGo := filepath.Join(dir, "main.go")
	os.WriteFile(mainGo, []byte(goSrc), 0644)

	// 2. 找 gopls
	gopls := "gopls"
	if p := os.Getenv("GOPLS"); p != "" {
		gopls = p
	} else if _, err := os.Stat(`F:\MyGolangPrograms\bin\gopls.exe`); err == nil {
		gopls = `F:\MyGolangPrograms\bin\gopls.exe`
	}

	c, err := lsp.NewClient(gopls)
	if err != nil {
		fmt.Println("❌ 启动 gopls 失败:", err)
		os.Exit(1)
	}
	defer c.Close()

	diagCh := make(chan int, 8)
	c.OnDiagnostics = func(uri string, ds []lsp.Diagnostic) {
		fmt.Printf("📋 诊断 %d 条 (%s)\n", len(ds), filepath.Base(uri))
		for _, d := range ds {
			fmt.Printf("   L%d: %s\n", d.Range.Start.Line+1, d.Message)
		}
		diagCh <- len(ds)
	}

	// 3. 握手 + 打开文档
	if err := c.Initialize(pathToURI(dir)); err != nil {
		fmt.Println("❌ initialize 失败:", err)
		os.Exit(1)
	}
	fmt.Println("✅ gopls 已初始化")
	uri := pathToURI(mainGo)
	c.DidOpen(uri, "go", goSrc)
	time.Sleep(2500 * time.Millisecond) // 等 gopls 加载工作区

	// 4. 请求 fmt. 处补全（第 6 行 0 基=5，"\tfmt." char 5）
	items, err := c.Completion(uri, 5, 5)
	if err != nil {
		fmt.Println("❌ completion 失败:", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 补全候选 %d 个，前 10 个:\n", len(items))
	for i, it := range items {
		if i >= 10 {
			break
		}
		fmt.Printf("   %-16s kind=%d  %s\n", it.Label, it.Kind, it.Detail)
	}
}
