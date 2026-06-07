package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolsReadWriteEditList(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// write_file（含自动建父目录）
	out, err := reg.Execute(ctx, "write_file", `{"path":"sub/a.txt","content":"hello WORLD"}`)
	if err != nil {
		t.Fatalf("write_file: %v", err)
	}
	if !strings.Contains(out, "已写入") {
		t.Errorf("write_file 返回 %q", out)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "a.txt")); string(b) != "hello WORLD" {
		t.Errorf("写入内容 = %q", b)
	}

	// read_file
	out, err = reg.Execute(ctx, "read_file", `{"path":"sub/a.txt"}`)
	if err != nil || out != "hello WORLD" {
		t.Errorf("read_file = %q, err=%v", out, err)
	}

	// edit_file（唯一替换）
	if _, err = reg.Execute(ctx, "edit_file", `{"path":"sub/a.txt","old_string":"WORLD","new_string":"GOUI"}`); err != nil {
		t.Fatalf("edit_file: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "a.txt")); string(b) != "hello GOUI" {
		t.Errorf("edit 后 = %q", b)
	}

	// edit_file：old_string 非唯一 → 报错
	os.WriteFile(filepath.Join(dir, "dup.txt"), []byte("x x x"), 0o644)
	if _, err = reg.Execute(ctx, "edit_file", `{"path":"dup.txt","old_string":"x","new_string":"y"}`); err == nil {
		t.Error("edit_file 非唯一 old_string 应报错")
	}

	// list_files
	out, err = reg.Execute(ctx, "list_files", `{}`)
	if err != nil {
		t.Fatalf("list_files: %v", err)
	}
	if !strings.Contains(out, "sub/") || !strings.Contains(out, "dup.txt") {
		t.Errorf("list_files = %q", out)
	}
	// list_files + pattern
	if out, _ = reg.Execute(ctx, "list_files", `{"pattern":"*.txt"}`); !strings.Contains(out, "dup.txt") {
		t.Errorf("pattern 过滤 = %q", out)
	}
}

func TestToolsPathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	for _, p := range []string{"../escape.txt", "../../etc/hosts", "sub/../../out.txt"} {
		if _, err := reg.Execute(context.Background(), "read_file", `{"path":"`+p+`"}`); err == nil {
			t.Errorf("越界路径 %q 应被拒", p)
		}
	}
}

func TestToolRunCommand(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	out, err := reg.Execute(context.Background(), "run_command", `{"command":"echo CMD_OK_88"}`)
	if err != nil {
		t.Fatalf("run_command: %v", err)
	}
	if !strings.Contains(out, "CMD_OK_88") {
		t.Errorf("run_command 输出 = %q", out)
	}
}

func TestMoveAndDeleteFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	RegisterDefaultTools(r, dir)
	ctx := context.Background()

	if _, err := r.Execute(ctx, "move_file", `{"from":"a.txt","to":"sub/b.txt"}`); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
		t.Error("a.txt 应已移走")
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "b.txt")); err != nil {
		t.Errorf("sub/b.txt 应存在：%v", err)
	}

	if _, err := r.Execute(ctx, "delete_file", `{"path":"sub/b.txt"}`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "b.txt")); !os.IsNotExist(err) {
		t.Error("b.txt 应已删除")
	}
	if _, err := r.Execute(ctx, "delete_file", `{"path":"sub"}`); err == nil {
		t.Error("delete_file 应拒绝目录")
	}
}

func TestRegistryDefinitions(t *testing.T) {
	reg := NewRegistry()
	RegisterDefaultTools(reg, t.TempDir())
	defs := reg.Definitions()
	if len(defs) != 14 { // read/write/edit/list/run + move/delete + search×2 + git×3 + web_fetch + update_plan
		t.Fatalf("应有 14 个工具定义，得 %d", len(defs))
	}
	if defs[0].Type != "function" || defs[0].Function.Name != "read_file" {
		t.Errorf("首个定义 = %+v", defs[0])
	}
	req, _ := defs[0].Function.Parameters["required"].([]string)
	if len(req) == 0 || req[0] != "path" {
		t.Errorf("read_file required = %v", req)
	}
	// 未知工具 → 报错
	if _, err := reg.Execute(context.Background(), "no_such_tool", `{}`); err == nil {
		t.Error("未知工具应报错")
	}
}
