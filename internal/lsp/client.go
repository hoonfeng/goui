package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Client 一个 LSP 客户端：管理语言服务器子进程，按 JSON-RPC over stdio（Content-Length 分帧）通信。
type Client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	nextID  int
	mu      sync.Mutex
	pending map[int]chan rpcResult
	closed  bool

	// OnDiagnostics 收到服务器诊断推送时回调（在读协程上调用，回调内需自行加锁/转主线程）。
	OnDiagnostics func(uri string, diags []Diagnostic)
}

type rpcResult struct {
	result json.RawMessage
	err    *rpcError
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewClient 启动语言服务器（如 gopls）并开始读协程。serverPath 为可执行文件路径。
func NewClient(serverPath string, args ...string) (*Client, error) {
	cmd := exec.Command(serverPath, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := &Client{cmd: cmd, stdin: stdin, pending: map[int]chan rpcResult{}}
	go c.readLoop(bufio.NewReader(stdout))
	return c, nil
}

func (c *Client) readLoop(r *bufio.Reader) {
	for {
		contentLen := 0
		for { // 读 header 直到空行
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				contentLen, _ = strconv.Atoi(strings.TrimSpace(line[len("Content-Length:"):]))
			}
		}
		if contentLen <= 0 {
			continue
		}
		body := make([]byte, contentLen)
		if _, err := io.ReadFull(r, body); err != nil {
			return
		}
		c.handleMessage(body)
	}
}

func (c *Client) handleMessage(body []byte) {
	var msg struct {
		ID     *json.Number    `json:"id"`
		Method string          `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
		Params json.RawMessage `json:"params"`
	}
	if json.Unmarshal(body, &msg) != nil {
		return
	}
	// 响应（有 id、无 method）
	if msg.ID != nil && msg.Method == "" {
		id, _ := strconv.Atoi(string(*msg.ID))
		c.mu.Lock()
		ch := c.pending[id]
		delete(c.pending, id)
		c.mu.Unlock()
		if ch != nil {
			ch <- rpcResult{msg.Result, msg.Error}
		}
		return
	}
	// 通知
	switch msg.Method {
	case "textDocument/publishDiagnostics":
		if c.OnDiagnostics != nil {
			var p PublishDiagnosticsParams
			if json.Unmarshal(msg.Params, &p) == nil {
				c.OnDiagnostics(p.URI, p.Diagnostics)
			}
		}
	}
}

// call 发请求并等待响应（阻塞）。
func (c *Client) call(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan rpcResult, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	if err := c.write(map[string]interface{}{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	res := <-ch
	if res.err != nil {
		return nil, fmt.Errorf("lsp %s: %s", method, res.err.Message)
	}
	return res.result, nil
}

func (c *Client) notify(method string, params interface{}) error {
	return c.write(map[string]interface{}{"jsonrpc": "2.0", "method": method, "params": params})
}

func (c *Client) write(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}
	_, err = fmt.Fprintf(c.stdin, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return err
}

// Close 关闭客户端（停止服务器）。
func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
}

// ── LSP 方法 ──

// Initialize 握手 + initialized 通知。rootURI 为工作区根（file:// URI）。
func (c *Client) Initialize(rootURI string) error {
	_, err := c.call("initialize", map[string]interface{}{
		"processId": nil,
		"rootUri":   rootURI,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{"snippetSupport": false},
				},
				"publishDiagnostics": map[string]interface{}{},
				"hover":              map[string]interface{}{},
			},
		},
	})
	if err != nil {
		return err
	}
	return c.notify("initialized", map[string]interface{}{})
}

// DidOpen 通知服务器打开了文档。
func (c *Client) DidOpen(uri, lang, text string) error {
	return c.notify("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri, "languageId": lang, "version": 1, "text": text,
		},
	})
}

// DidChange 全量同步文档内容。
func (c *Client) DidChange(uri, text string, version int) error {
	return c.notify("textDocument/didChange", map[string]interface{}{
		"textDocument":   map[string]interface{}{"uri": uri, "version": version},
		"contentChanges": []map[string]interface{}{{"text": text}},
	})
}

// Completion 请求补全（line/char 0 基）。
func (c *Client) Completion(uri string, line, char int) ([]CompletionItem, error) {
	res, err := c.call("textDocument/completion", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     Position{Line: line, Character: char},
	})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 || string(res) == "null" {
		return nil, nil
	}
	var list CompletionList // 结果可能是 {items:[]} 或直接 []
	if json.Unmarshal(res, &list) == nil && list.Items != nil {
		return list.Items, nil
	}
	var items []CompletionItem
	json.Unmarshal(res, &items)
	return items, nil
}
