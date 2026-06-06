package agent

// 只读 git 工具：git_status / git_diff / git_log。复刻参考源的读类 git 工具。
// 标 ReadOnly（免审批）——即便手动审核模式下也能直接查仓库状态，无需用户每次点确认。
// 写类 git（add/commit）走 run_command（需审批），不在此处（破坏性操作交人确认）。

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func registerGitTools(r *Registry, root string) {
	r.Register(&Tool{
		Name:        "git_status",
		Description: "查看 git 工作区状态（当前分支 + 已修改/暂存/未跟踪文件，porcelain 紧凑格式）。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := runGit(ctx, root, "status", "--porcelain=v1", "--branch")
			if err != nil {
				return "", err
			}
			trimmed := strings.TrimSpace(out)
			// porcelain --branch 首行恒为 "## <branch>"；仅此一行=工作区干净。
			// 非 ## 开头（如 fatal: not a git repository）原样返回，别误标「干净」。
			if strings.HasPrefix(trimmed, "##") && !strings.Contains(trimmed, "\n") {
				return out + "（工作区干净）", nil
			}
			return out, nil
		},
	})

	r.Register(&Tool{
		Name:        "git_diff",
		Description: "查看 git 改动。file 可选（限定单个文件）；staged=true 看已暂存(--cached)的改动，否则看工作区未暂存改动。",
		Parameters: objSchema(props{
			"file":   strProp("可选：限定单个文件路径"),
			"staged": boolProp("看已暂存(--cached)改动，默认看未暂存"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			gitArgs := []string{"diff"}
			if argBool(args, "staged") {
				gitArgs = append(gitArgs, "--cached")
			}
			if f := strings.TrimSpace(argStr(args, "file")); f != "" {
				gitArgs = append(gitArgs, "--", f)
			}
			out, err := runGit(ctx, root, gitArgs...)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(out) == "" || out == "（无输出）" {
				return "（无改动）", nil
			}
			return out, nil
		},
	})

	r.Register(&Tool{
		Name:        "git_log",
		Description: "查看最近提交历史（单行格式）。count 限定条数（默认 15）；file 可选（限定某文件的历史）。",
		Parameters: objSchema(props{
			"count": intProp("条数（默认 15）"),
			"file":  strProp("可选：限定某文件的提交历史"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			count := clampInt(argInt(args, "count", 15), 15, 1, 200)
			gitArgs := []string{"log", "--oneline", "-n", strconv.Itoa(count)}
			if f := strings.TrimSpace(argStr(args, "file")); f != "" {
				gitArgs = append(gitArgs, "--", f)
			}
			out, err := runGit(ctx, root, gitArgs...)
			if err != nil {
				return "", err
			}
			return out, nil
		},
	})
}

// runGit 在 dir 执行一条 git 子命令（30s 超时）。core.quotepath=false 让非 ASCII 文件名正常显示。
// git 非零退出（如目录非 git 仓库）：有输出则连同返回（让 agent 看到原因），无输出则作 error。
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	full := append([]string{"-c", "core.quotepath=false"}, args...)
	c := exec.CommandContext(cctx, "git", full...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	res := capOutput(string(out), 16000)
	if cctx.Err() == context.DeadlineExceeded {
		return res + "\n[git 超时 30s 已终止]", nil
	}
	if err != nil {
		if strings.TrimSpace(res) == "" {
			return "", fmt.Errorf("git %s 失败: %v", strings.Join(args, " "), err)
		}
		return res, nil // 有输出（如 fatal: not a git repository）→ 回给 agent
	}
	if strings.TrimSpace(res) == "" {
		return "（无输出）", nil
	}
	return res, nil
}
