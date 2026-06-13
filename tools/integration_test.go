package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestToolWorkflow 集成测试：模拟 Agent 使用多个工具完成一个完整工作流。
// 场景：创建临时目录 → 写入文件 → 读取文件验证 → 列出目录 → 删除文件 → 验证删除。
func TestToolWorkflow(t *testing.T) {
	dir := t.TempDir()

	// 1. WriteFile — 创建一个 Go 源文件
	fp := filepath.Join(dir, "hello.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, tools!")
}
`
	result, err := WriteFile(fp, content)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if !strings.Contains(result, "已写入") {
		t.Errorf("WriteFile unexpected result: %s", result)
	}

	// 2. ReadFile — 读取并验证内容
	got, err := ReadFile(fp, 0, 0)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(got, "Hello, tools!") {
		t.Errorf("ReadFile content mismatch, got: %s", got)
	}

	// 3. ReadFile offset/limit — 读片段（第3-4行：import "fmt" 及空行）
	got, err = ReadFile(fp, 3, 2)
	if err != nil {
		t.Fatalf("ReadFile offset failed: %v", err)
	}
	if !strings.Contains(got, "import") {
		t.Errorf("ReadFile offset expected import, got: %s", got)
	}

	// 4. ListDir — 列出目录，验证文件存在
	list, err := ListDir(dir, "*.go")
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}
	if !strings.Contains(list, "hello.go") {
		t.Errorf("ListDir missing hello.go: %s", list)
	}

	// 5. DeleteFile — 删除文件
	result, err = DeleteFile(fp)
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}
	if !strings.Contains(result, "已删除") {
		t.Errorf("DeleteFile unexpected: %s", result)
	}

	// 6. 验证删除
	if _, err := os.Stat(fp); !os.IsNotExist(err) {
		t.Error("File should not exist after DeleteFile")
	}
}
