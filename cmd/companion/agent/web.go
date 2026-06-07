// 联网工具：web_fetch —— 抓取一个 http(s) 网页并返回纯文本（去 HTML 标签），供 agent 查阅文档/网页。
// 纯 Go、无 GUI/CGO 依赖；可用 httptest 离线测。

package agent

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
	reBlockTag    = regexp.MustCompile(`(?i)</(p|div|li|tr|h[1-6]|section|article|header|footer|ul|ol|table|blockquote)>|<br\s*/?>`)
	reTag         = regexp.MustCompile(`(?s)<[^>]+>`)
	reBlankLines  = regexp.MustCompile(`\n{3,}`)
	reInlineWS    = regexp.MustCompile(`[ \t]{2,}`)
)

// htmlToText 把 HTML 粗略转可读纯文本：去 script/style、块标签→换行、去其余标签、解实体、压空白。
func htmlToText(s string) string {
	s = reScriptStyle.ReplaceAllString(s, " ")
	s = reBlockTag.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimSpace(reInlineWS.ReplaceAllString(ln, " "))
	}
	s = strings.Join(lines, "\n")
	s = reBlankLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// registerWebTools 注册联网工具：web_fetch。
func registerWebTools(r *Registry) {
	r.Register(&Tool{
		Name:        "web_fetch",
		Description: "抓取一个 http(s) 网页并返回其纯文本内容（去除 HTML 标签，超长截断）。用于查阅在线文档、API 参考、网页。",
		Parameters:  objSchema(props{"url": strProp("要抓取的网页 URL（必须 http:// 或 https://）")}, "url"),
		ReadOnly:    true,
		Handler:     webFetch,
	})
}

const webFetchMaxBody = 2 << 20 // 2MB 读取上限
const webFetchMaxOut = 20000    // 返回文本上限

// webFetch 抓取 URL 内容并转纯文本。抽出便于离线测试。
func webFetch(ctx context.Context, args map[string]any) (string, error) {
	url := strings.TrimSpace(argStr(args, "url"))
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("仅支持 http(s) URL：%q", url)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "companion-agent/1.0")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, webFetchMaxBody))
	if err != nil {
		return "", err
	}
	text := htmlToText(string(body))
	if len(text) > webFetchMaxOut {
		text = text[:webFetchMaxOut] + "\n…（内容已截断）"
	}
	return fmt.Sprintf("URL: %s\nHTTP %d %s\n\n%s", url, resp.StatusCode, http.StatusText(resp.StatusCode), text), nil
}
