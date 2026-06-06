package agent

// 搜索/导航工具：search_content（正则内容搜索，grep 风格）+ search_files（通配符查找文件）。
// 复刻参考源 src/agent 的 search_content / search_files。两者只读、免审批、限定工作区内，
// 自动跳过 .git/node_modules 等目录与二进制/超大文件（防把 LLM 上下文撑爆）。

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	maxSearchFileSize = 10 << 20 // 10MB：超过则跳过（不读进内存搜索）
	searchSniffBytes  = 8000     // 二进制嗅探：读前 N 字节查空字节
)

// skipDirs 搜索时跳过的目录名（与文件树/grep 惯例一致）。
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	".idea": true, ".vscode": true, ".cache": true,
}

func registerSearchTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "search_content",
		Description: "在工作区内按正则搜索文件内容，返回匹配的「相对路径:行号: 行文本」。" +
			"pattern 为 RE2 正则；path 限定子目录（省略=根）；glob 按文件名过滤（如 *.go）；" +
			"case_insensitive 忽略大小写；max_results 上限（默认 200）。自动跳过 .git/node_modules 等与二进制/超大文件。",
		Parameters: objSchema(props{
			"pattern":          strProp("RE2 正则表达式"),
			"path":             strProp("限定子目录（省略=工作区根）"),
			"glob":             strProp("文件名通配过滤，如 *.go"),
			"case_insensitive": boolProp("忽略大小写"),
			"max_results":      intProp("结果行数上限（默认 200）"),
		}, "pattern"),
		ReadOnly: true,
		Handler:  searchContentHandler(root),
	})

	r.Register(&Tool{
		Name: "search_files",
		Description: "在工作区内按通配符递归查找文件，返回相对路径列表（已排序）。" +
			"pattern 为通配符：不含 / 时匹配文件名（如 *.go、*config*），含 / 时匹配相对路径（如 internal/*/main.go）；" +
			"path 限定子目录；max_results 上限（默认 500）。跳过 .git/node_modules 等。",
		Parameters: objSchema(props{
			"pattern":     strProp("文件名/路径通配符，如 *.go"),
			"path":        strProp("限定子目录（省略=工作区根）"),
			"max_results": intProp("结果上限（默认 500）"),
		}, "pattern"),
		ReadOnly: true,
		Handler:  searchFilesHandler(root),
	})
}

func searchContentHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		pattern := strings.TrimSpace(argStr(args, "pattern"))
		if pattern == "" {
			return "", fmt.Errorf("pattern 不能为空")
		}
		prefix := ""
		if argBool(args, "case_insensitive") {
			prefix = "(?i)"
		}
		re, err := regexp.Compile(prefix + pattern)
		if err != nil {
			return "", fmt.Errorf("正则编译失败: %w", err)
		}
		base, err := searchRoot(root, argStr(args, "path"))
		if err != nil {
			return "", err
		}
		glob := strings.TrimSpace(argStr(args, "glob"))
		max := clampInt(argInt(args, "max_results", 200), 200, 1, 2000)

		var b strings.Builder
		count := 0
		truncated := false
		walkErr := filepath.WalkDir(base, func(p string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil // 跳过无法访问的项
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				if p != base && skipDirs[d.Name()] {
					return fs.SkipDir
				}
				return nil
			}
			if glob != "" {
				if ok, _ := path.Match(glob, d.Name()); !ok {
					return nil
				}
			}
			if info, e := d.Info(); e == nil && info.Size() > maxSearchFileSize {
				return nil
			}
			data, e := os.ReadFile(p)
			if e != nil || isBinary(data) {
				return nil
			}
			rel := relSlash(root, p)
			for i, line := range strings.Split(string(data), "\n") {
				if re.MatchString(line) {
					fmt.Fprintf(&b, "%s:%d: %s\n", rel, i+1, trimLine(line))
					if count++; count >= max {
						truncated = true
						return fs.SkipAll
					}
				}
			}
			return nil
		})
		if walkErr != nil && ctx.Err() != nil {
			return "", ctx.Err()
		}
		if count == 0 {
			return "（未找到匹配）", nil
		}
		res := b.String()
		if truncated {
			res += fmt.Sprintf("[已达上限 %d 条，可能还有更多匹配——请缩小 pattern 或 path]\n", max)
		}
		return capOutput(res, 16000), nil
	}
}

func searchFilesHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		pattern := strings.TrimSpace(argStr(args, "pattern"))
		if pattern == "" {
			return "", fmt.Errorf("pattern 不能为空")
		}
		base, err := searchRoot(root, argStr(args, "path"))
		if err != nil {
			return "", err
		}
		max := clampInt(argInt(args, "max_results", 500), 500, 1, 5000)

		var matches []string
		truncated := false
		walkErr := filepath.WalkDir(base, func(p string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				if p != base && skipDirs[d.Name()] {
					return fs.SkipDir
				}
				return nil
			}
			if matchFile(pattern, d.Name(), relSlash(root, p)) {
				matches = append(matches, relSlash(root, p))
				if len(matches) >= max {
					truncated = true
					return fs.SkipAll
				}
			}
			return nil
		})
		if walkErr != nil && ctx.Err() != nil {
			return "", ctx.Err()
		}
		if len(matches) == 0 {
			return "（未找到匹配文件）", nil
		}
		sort.Strings(matches)
		res := strings.Join(matches, "\n")
		if truncated {
			res += fmt.Sprintf("\n[已达上限 %d 个]", max)
		}
		return capOutput(res, 16000), nil
	}
}

// ─── 辅助 ────────────────────────────────────────────────────

// searchRoot 解析搜索起点目录（省略=工作区根，限定工作区内）。
func searchRoot(root, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return root, nil
	}
	return resolvePath(root, rel)
}

// matchFile 通配匹配：pattern 含 / 时按相对路径匹配，否则按文件名匹配（均用 path.Match，slash 语义跨平台一致）。
func matchFile(pattern, base, rel string) bool {
	pat := filepath.ToSlash(pattern)
	if strings.Contains(pat, "/") {
		if ok, _ := path.Match(pat, rel); ok {
			return true
		}
	}
	ok, _ := path.Match(pat, base)
	return ok
}

// relSlash 取相对工作区根的 slash 路径（给 LLM 看的稳定相对路径）。
func relSlash(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

// isBinary 嗅探前若干字节是否含空字节（含=视作二进制，跳过文本搜索）。
func isBinary(data []byte) bool {
	n := min(len(data), searchSniffBytes)
	for i := range n {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// trimLine 去首尾空白并按 rune 截断过长行（结果行预览，避免单行撑爆）。
func trimLine(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 200 {
		return string(r[:200]) + "…"
	}
	return s
}

// clampInt 取值约束：v<=0 或越界则回退 def，并夹到 [lo, hi]。
func clampInt(v, def, lo, hi int) int {
	if v <= 0 {
		v = def
	}
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return v
}
