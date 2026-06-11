package lsp

import "sync"

// 工作区级语言服务器池：同一(服务器+工作区)共用一个 Client，被多文件/多编辑器共享——
// 切换标签不重启、不同语言文件同时打开各用各的 server（非单例）。会话级常驻，CloseAll 退出时统一关停。

var (
	poolMu  sync.Mutex
	clients = map[string]*Client{}
)

func poolKey(server, workspace string) string { return server + "\x00" + workspace }

// Acquire 取（或首次起并 initialize）某(服务器+工作区)的共享 Client；已存在则直接复用，不重启。
// 创建期间持锁（initialize 慢，只首次）——调用方应在协程里调，勿在 UI 线程直接调。
func Acquire(server string, args []string, workspace string) (*Client, error) {
	key := poolKey(server, workspace)
	poolMu.Lock()
	defer poolMu.Unlock()
	if c, ok := clients[key]; ok {
		return c, nil // 复用现有（多文件/切标签都共享，不重启）
	}
	c, err := NewClient(server, args...)
	if err != nil {
		return nil, err
	}
	if err := c.Initialize(workspace); err != nil {
		c.Close()
		return nil, err
	}
	clients[key] = c
	return c, nil
}

// CloseAll 关停所有共享语言服务器（app 退出时调，免子进程残留）。
func CloseAll() {
	poolMu.Lock()
	defer poolMu.Unlock()
	for k, c := range clients {
		c.Close()
		delete(clients, k)
	}
}

// PoolSize 当前存活的语言服务器数（监控/测试用）。
func PoolSize() int {
	poolMu.Lock()
	defer poolMu.Unlock()
	return len(clients)
}
