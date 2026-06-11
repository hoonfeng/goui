// editorpanel 对外接口 —— 编辑器面板。外部经 Editor 单例(Open/OpenAt/Save/…)用;
// 菜单(耦合 ctxmenu/chat)与欢迎页配置由 main 注入(同 termpanel.OnContextMenu 模式)。
//
//go:build windows

package editorpanel

import "github.com/user/goui/internal/widget"

// 注入回调/配置(菜单耦合 ctxmenu/chat、欢迎页配置在 main，故注入而非本包持有)。
var (
	OnContentMenu func(x, y float64)        // 编辑器内容区右键菜单(main 注入 editorContentMenu)
	OnTabMenu     func(x, y float64, i int) // 编辑器标签右键菜单(带标签索引；main 注入 editorTabMenu)
	WelcomeSpec   widget.ComponentSpec      // 欢迎页配置(main 解析 welcome.json 后注入)

	OnReferences func(refs []widget.CodeLoc) // 转到引用结果回调(main 注入 editorReferences，结果列进面板)
	OnSymbols    func(syms []widget.CodeSym) // 文档大纲结果回调(main 注入 editorSymbols)
)

// Area 中列编辑区入口(main 的 midColumn 调用)。
func Area() widget.Widget { return &EditorPanel{} }

// Reset 复位编辑器单例(测试用)。
func Reset() { Editor = &editorState{} }

// NewTabForTest 造一个仅含路径的标签(测试用，不读盘)。
func NewTabForTest(path string) *editorTab { return &editorTab{path: path} }

// ─── editorState 外部访问 ───

// Tabs 当前打开的标签（外部只读遍历）。
func (e *editorState) Tabs() []*editorTab { return e.tabs }

// SetTabs 直接设标签集（测试用；变参以便外部传 NewTabForTest 的返回值，无需命名内部类型）。
func (e *editorState) SetTabs(ts ...*editorTab) { e.tabs = ts }

// BumpReload 重载令牌 +1（设置面板改字体/字号后，令各标签编辑器按令牌重建）。
func (e *editorState) BumpReload() { e.reload++ }

// ─── editorTab 字段访问(ctxmenu / 测试用，避免导出内部字段名) ───

func (t *editorTab) Path() string    { return t.path }
func (t *editorTab) Lang() string    { return t.lang }
func (t *editorTab) Content() string { return t.content }
func (t *editorTab) Dirty() bool     { return t.dirty }
func (t *editorTab) SetDirty(b bool) { t.dirty = b }
