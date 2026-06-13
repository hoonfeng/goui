package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFile(t *testing.T) {
	// 创建临时文件
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// 读全文
	got, err := ReadFile(fp, 0, 0)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if got != content {
		t.Errorf("ReadFile full want %q, got %q", content, got)
	}

	// 读片段 offset=2, limit=2
	got, err = ReadFile(fp, 2, 2)
	if err != nil {
		t.Fatalf("ReadFile offset failed: %v", err)
	}
	want := "line2\nline3"
	if got != want {
		t.Errorf("ReadFile offset want %q, got %q", want, got)
	}

	// 读不存在的文件
	_, err = ReadFile(filepath.Join(dir, "nope.txt"), 0, 0)
	if err == nil {
		t.Error("ReadFile should fail on missing file")
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "sub", "test.txt")
	content := "hello world"

	got, err := WriteFile(fp, content)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if !strings.Contains(got, "已写入") {
		t.Errorf("WriteFile response unexpected: %s", got)
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("WriteFile content want %q, got %q", content, string(data))
	}
}

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(fp, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := DeleteFile(fp)
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}
	if !strings.Contains(got, "已删除") {
		t.Errorf("DeleteFile response unexpected: %s", got)
	}

	if _, err := os.Stat(fp); !os.IsNotExist(err) {
		t.Error("DeleteFile did not remove the file")
	}

	// 删不存在的文件
	_, err = DeleteFile(filepath.Join(dir, "nope.txt"))
	if err == nil {
		t.Error("DeleteFile should fail on missing file")
	}
}

func TestListDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)

	// 列出全部
	got, err := ListDir(dir, "")
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}
	if !strings.Contains(got, "sub/") {
		t.Error("ListDir should contain sub/")
	}
	if !strings.Contains(got, "a.go") {
		t.Error("ListDir should contain a.go")
	}

	// 按模式过滤
	got, err = ListDir(dir, "*.go")
	if err != nil {
		t.Fatalf("ListDir pattern failed: %v", err)
	}
	if !strings.Contains(got, "a.go") {
		t.Error("ListDir pattern should contain a.go")
	}
	if strings.Contains(got, "b.txt") {
		t.Error("ListDir pattern should not contain b.txt")
	}
}
