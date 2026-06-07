// 记忆工具：memory_write/read/list/search —— 跨会话持久记忆，存在工作区 .companion/memory/ 下，
// 每条一个 .md（frontmatter: name/type/description + 正文）。让 agent 记住项目知识/用户偏好/教训。

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func memoryDir(root string) string { return filepath.Join(root, ".companion", "memory") }

// safeMemName 把名字里的路径危险字符(/ \ : . 空格)换成 -，防路径穿越；保留 CJK 等其它字符。
func safeMemName(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '.', ' ':
			return '-'
		}
		return r
	}, strings.TrimSpace(s))
}

func frontmatterField(text, key string) string {
	for _, ln := range strings.Split(text, "\n") {
		if strings.HasPrefix(ln, key+":") {
			return strings.TrimSpace(strings.TrimPrefix(ln, key+":"))
		}
	}
	return ""
}

// listMemories 列出 dir 下记忆（filter 非空则按关键词过滤名/摘要/正文）。
func listMemories(dir, filter string) string {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return "（暂无记忆）"
	}
	filter = strings.ToLower(filter)
	var lines []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		text := string(data)
		if filter != "" && !strings.Contains(strings.ToLower(text), filter) {
			continue
		}
		lines = append(lines, "- "+strings.TrimSuffix(e.Name(), ".md")+"："+frontmatterField(text, "description"))
	}
	if len(lines) == 0 {
		return "（无匹配记忆）"
	}
	return strings.Join(lines, "\n")
}

func registerMemoryTools(r *Registry, root string) {
	dir := memoryDir(root)

	r.Register(&Tool{
		Name: "memory_write",
		Description: "写入/更新一条持久记忆（跨会话保留在 .companion/memory/）。name 唯一标识；" +
			"type: user(用户偏好)/feedback(纠正与确认的做法)/project(项目决策约束)/reference(外部资源指针)；description 一句话摘要；content 正文。",
		Parameters: objSchema(props{
			"name":        strProp("唯一名（短）"),
			"type":        strProp("user/feedback/project/reference"),
			"description": strProp("一句话摘要"),
			"content":     strProp("正文"),
		}, "name", "description", "content"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := safeMemName(argStr(args, "name"))
			if name == "" {
				return "", fmt.Errorf("name 不能为空")
			}
			typ := argStr(args, "type")
			if typ == "" {
				typ = "project"
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", err
			}
			body := fmt.Sprintf("---\nname: %s\ntype: %s\ndescription: %s\n---\n\n%s\n",
				name, typ, argStr(args, "description"), argStr(args, "content"))
			if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(body), 0o644); err != nil {
				return "", err
			}
			return "已记忆：" + name, nil
		},
	})

	r.Register(&Tool{
		Name:        "memory_read",
		Description: "按 name 读取一条记忆的全文。",
		Parameters:  objSchema(props{"name": strProp("记忆名")}, "name"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := safeMemName(argStr(args, "name"))
			data, err := os.ReadFile(filepath.Join(dir, name+".md"))
			if err != nil {
				return "", fmt.Errorf("无此记忆: %s", name)
			}
			return string(data), nil
		},
	})

	r.Register(&Tool{
		Name:        "memory_list",
		Description: "列出所有记忆（名 + 摘要）。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return listMemories(dir, ""), nil
		},
	})

	r.Register(&Tool{
		Name:        "memory_search",
		Description: "按关键词搜索记忆（匹配名/摘要/正文），返回命中条目的名+摘要。",
		Parameters:  objSchema(props{"query": strProp("关键词")}, "query"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			q := strings.TrimSpace(argStr(args, "query"))
			if q == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			return listMemories(dir, q), nil
		},
	})
}
