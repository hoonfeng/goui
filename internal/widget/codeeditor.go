package widget

import (
	"go/format"
	"strings"
	"sync"
	"time"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/lsp"
	"github.com/user/goui/internal/types"
)

// ─── CodeEditor 代码编辑器（对标 Monaco 编辑体验，第一批：行号 + 语法高亮 + 编辑）──
// 自绘：左侧行号栏(gutter) + 编辑区(按 token 着色) + 当前行高亮 + 等宽字体 + 横竖滚动。
// 编辑内核用 (行,列) 模型（贴合代码编辑器/Monaco）。后续批次加折叠/查找/多光标/LSP 补全。

// CECompletion 外部补全项（StructEditor 注入：变量/子程序/命令）。Detail 显示在候选右侧（如「变量」「命令」）。
type CECompletion struct {
	Label  string
	Detail string
}

// CECursorState 跨 Element 重建保持的光标/滚动状态（CodeWorkbench 切换视图时复用）。
type CECursorState struct {
	Line, Col        int
	ScrollX, ScrollY float64
}

// cePos 光标/选区位置（行、列均为 rune 索引）。
type cePos struct{ line, col int }

func cePosLess(a, b cePos) bool {
	if a.line != b.line {
		return a.line < b.line
	}
	return a.col < b.col
}
func cePosEq(a, b cePos) bool { return a.line == b.line && a.col == b.col }

type CodeEditor struct {
	StatelessWidget
	Width    float64
	Height   float64
	Language string
	// 受控重载令牌：改变它（配合 SetState/relayout）→ Update 把运行时内容重置为新的 initial
	// （切换打开的文件用；goui 复用 Element，否则旧文件内容残留）。同 Input.ResetToken 套路。
	ReloadToken int
	// 跳转到行：RevealToken 改变（配合 SetState）→ 把光标移到 RevealLine(1 基)并滚动到可见。
	// 供搜索/Git 等点击结果跳转用。RevealLine<=0 不跳。
	RevealLine  int
	RevealToken int
	initial     string
	OnChange    func(string)
	OnEnter     func()             // 按回车换行后触发（StructEditor 用作「回车自动声明变量」时机）
	OnCursorMove func(line, col int) // 光标位置变化回调（行/列均 0 基，状态栏 Ln/Col 显示用）
	ExtraIdents func() []CECompletion // 外部补全源（StructEditor 注入：已声明变量/子程序/命令库）
	Minimap     bool               // 是否显示右侧缩略图（默认开）
	WordWrap    bool               // 软自动换行：长行按编辑区宽折成多视觉行（默认关，可 toggleWrap 切换）
	FontSize    float64            // 等宽字号（<=0 用默认 14）
	FontFamily  string             // 等宽字体族（空=Consolas）
	FontBold      bool             // 整体加粗
	FontItalic    bool             // 整体斜体
	FontUnderline bool             // 整体下划线
	Embedded bool // 嵌入模式：去掉独立白卡圆角/聚焦蓝框，无缝融入父容器（StructEditor 用）

	LineNumberOffset int            // 行号起始偏移：显示行号 = 实际行 + 此值（StructEditor 全局连续行号用）
	IndentGuides     bool           // 缩进连线：按缩进画淡竖线，块结构(首…尾)连成一条
	GutterOverride   float64        // >0 时强制行号栏宽度（StructEditor 用，与变量表行号列等宽对齐成一条）
	CursorRef        *CECursorState // 跨重建保持光标/滚动（CodeWorkbench 切换视图用）

	LSPServer    string   // 语言服务器可执行路径（如 gopls / typescript-language-server）；空=纯词法补全
	LSPArgs      []string // 语言服务器启动参数（如 --stdio）
	LSPWorkspace string   // 工作区根 file:// URI
	LSPFile      string   // 文档 file:// URI
	LSPLangID    string   // LSP languageId（如 "typescript"；空=用 Language）

	// OnGoToDefinition 转到定义：runCommand("gotoDefinition") 触发，编辑器经 LSP 解析光标处定义，
	// 拿到目标 → 回调宿主打开该文件并跳到行(1 基)。无 LSP 或无定义则不触发。
	OnGoToDefinition func(file string, line, col int)
	// OnReferences 查找引用：runCommand("findReferences") → 回调宿主列出所有引用位置（供面板/对话框展示+跳转）。
	OnReferences func(refs []CodeLoc)
	// OnDocumentSymbols 文档符号大纲：runCommand("documentSymbol") → 回调宿主列出当前文件符号（供大纲/转到符号）。
	OnDocumentSymbols func(syms []CodeSym)
}

// CodeLoc 一个代码位置（文件 + 行/列，行 1 基）。查找引用结果用。
type CodeLoc struct {
	File string
	Line int
	Col  int
}

// CodeSym 一个文档符号（名 + 种类 + 行 1 基 + 嵌套深度）。大纲/转到符号用。
type CodeSym struct {
	Name  string
	Kind  int
	Line  int
	Depth int
}

// WithLSP 接入语言服务器（gopls 等）做语义补全 + 诊断。
// server: 可执行路径; workspaceURI: 工作区根 file:// URI; fileURI: 文档 file:// URI。
func (c *CodeEditor) WithLSP(server, workspaceURI, fileURI string) *CodeEditor {
	c.LSPServer, c.LSPWorkspace, c.LSPFile = server, workspaceURI, fileURI
	return c
}

// NewCodeEditor 创建代码编辑器。language: "go"/"json"/"js"/"python"…（未知→Go 兜底）。
func NewCodeEditor(language, initial string) *CodeEditor {
	return &CodeEditor{Width: 680, Height: 360, Language: language, initial: initial, Minimap: true}
}
func (c *CodeEditor) WithSize(w, h float64) *CodeEditor     { c.Width, c.Height = w, h; return c }
func (c *CodeEditor) OnChanged(fn func(string)) *CodeEditor { c.OnChange = fn; return c }
func (c *CodeEditor) WithMinimap(on bool) *CodeEditor       { c.Minimap = on; return c }
func (c *CodeEditor) WithFontSize(s float64) *CodeEditor    { c.FontSize = s; return c }
func (c *CodeEditor) WithFontFamily(f string) *CodeEditor   { c.FontFamily = f; return c }

// WithFontStyle 设置整体加粗/斜体/下划线（编辑器全文统一样式，外观字体控件用）。
func (c *CodeEditor) WithFontStyle(bold, italic, underline bool) *CodeEditor {
	c.FontBold, c.FontItalic, c.FontUnderline = bold, italic, underline
	return c
}

// expandTabs 把制表符展开为 4 空格（字体无 tab 字形会渲染成豆腐块，统一在数据层换成空格）。
func expandTabs(s string) string { return strings.ReplaceAll(s, "\t", "    ") }

const ceIndentSize = 4 // 缩进单位（空格数），缩进连线按此分级

// leadingSpaces 返回字符串前导空格数（用于缩进连线分级）。
func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}

// revealLine 把光标移到 line(1 基)并滚动到可见（顶部留几行上下文）。供跳转到行用。
func (e *CodeEditorElement) revealLine(line int) {
	row := line - 1
	if row < 0 {
		row = 0
	}
	e.cursor = e.clampPos(cePos{row, 0})
	e.anchor = e.cursor
	if y := float64(e.cursor.line-3) * ceLineH; y > 0 { // 目标行上方留 3 行（预定位，Paint 再精确跟随）
		e.scrollY = y
	} else {
		e.scrollY = 0
	}
	e.computeVisible()
	e.cursorMoved = true // 让 Paint 用换行/折叠感知逻辑把光标精确滚入视野
	e.MarkNeedsPaint()
}

func (c *CodeEditor) CreateElement() Element {
	lines := strings.Split(expandTabs(c.initial), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	fsz := c.FontSize
	if fsz <= 0 {
		fsz = 14
	}
	fam := c.FontFamily
	if fam == "" {
		fam = "Consolas"
	}
	fw := canvas.FontWeightNormal
	if c.FontBold {
		fw = canvas.FontWeightBold
	}
	fst := canvas.FontStyleNormal
	if c.FontItalic {
		fst = canvas.FontStyleItalic
	}
	e := &CodeEditorElement{
		BaseElement: BaseElement{widget: c},
		ed:          c,
		lastReload:  c.ReloadToken,
		lastReveal:  c.RevealToken,
		lines:       lines,
		lang:        ceLangFor(c.Language),
		font:        canvas.Font{Family: fam, Size: fsz, Weight: fw, Style: fst, Underline: c.FontUnderline},
		folded:      map[int]bool{},
		showMinimap: c.Minimap,
		wrap:        c.WordWrap,
		wrapDirty:   true,
	}
	e.rehighlight()
	e.computeVisible()
	if c.CursorRef != nil { // 跨重建恢复光标/滚动（切换视图保持位置）
		e.cursor = e.clampPos(cePos{c.CursorRef.Line, c.CursorRef.Col})
		e.anchor = e.cursor
		e.scrollX, e.scrollY = c.CursorRef.ScrollX, c.CursorRef.ScrollY
	}
	if c.RevealLine > 0 { // 初次即跳转到行（新建标签打开搜索结果）
		e.revealLine(c.RevealLine)
	}
	if c.LSPServer != "" {
		e.lspURI = c.LSPFile
		e.lspServer = c.LSPServer
		if !c.Embedded { // 独立编辑器立即启动；嵌入体编辑器聚焦时再启动（避免 N 子程序 = N 个 gopls）
			go e.startLSP(c.LSPServer, c.LSPWorkspace) // 异步启动（initialize 耗时，勿卡 UI）
		}
	}
	return e
}

const (
	ceLineH    = 21.0 // 行高（font 14 → 21）
	ceGutterPad = 10.0
	ceTextPad  = 8.0
)

type CodeEditorElement struct {
	BaseElement
	ed         *CodeEditor
	lastReload int // 上次 ReloadToken，用于检测「切换文件」需重置内容
	lastReveal int // 上次 RevealToken，用于检测「跳转到行」
	lines      []string
	cursor     cePos
	anchor     cePos // == cursor 表示无选区
	lang       *ceLang
	font       canvas.Font

	hl [][]ceToken // 每行 token（仅编辑后重算）

	// enclosingBracket 缓存：bracketRev 每次 rehighlight(内容变) 自增；命中条件 = rev 与光标行均未变，
	// 避免大文件每帧全文扫描括号（活动缩进折线每帧要它）。
	bracketRev    int
	brkCacheRev   int
	brkCacheLine  int
	brkCacheValid bool
	brkOL, brkOC, brkCL, brkCC int
	brkOK         bool

	scrollX, scrollY float64
	focused          bool
	hovered          bool
	focusTime        time.Time
	cursorMoved      bool
	selecting        bool

	gutterW float64 // 行号栏宽度（Layout 算）

	vbarThumb, hbarThumb       types.Rect
	vbarFactor, hbarFactor     float64
	draggingVBar, draggingHBar bool
	dragStartMouse             float64
	dragStartScroll            float64

	undoStack []ceSnapshot // 撤销栈（保存编辑前快照）
	redoStack []ceSnapshot
	lastEdit  string // 上一次编辑类型，用于合并连续输入/删除

	composition       string        // IME 组合预览串（中文拼音未确认时）
	compositionCursor int           // 组合串内光标位置
	cursorClientX     float64       // Paint 缓存：光标客户区 X（IME 候选定位）
	cursorClientY     float64       // Paint 缓存：光标顶部客户区 Y
	lastCanvas        canvas.Canvas // Paint 缓存：用与渲染一致的 Skia 测量（避免 MeasureTextGlobal 中文漂移）

	measCache map[string]float64 // 文本→测量宽度缓存（同字体），免每帧逐行重测；measFont 变则清
	measFont  canvas.Font

	// 查找替换
	findActive   bool         // 查找栏是否打开
	replaceShown bool         // 是否显示替换行
	findQuery    string       // 查找串
	findCol      int          // 查找框内光标
	replaceQuery string       // 替换串
	replaceCol   int          // 替换框内光标
	findField    int          // 当前焦点字段：0=查找框 1=替换框
	findCase     bool         // 区分大小写
	matches      []ceMatch    // 所有匹配（文档顺序）
	curMatch     int          // 当前匹配下标
	findOrigin   cePos        // 打开查找时的光标位置（定位首个匹配，避免输入时漂移）
	findBtns     []ceFindBtn  // 查找栏按钮命中区（Paint 算）

	// 代码折叠
	foldRanges  []foldRange  // 可折叠区域（{ 到匹配 }，跨行）
	folded      map[int]bool // 折叠起始行 → 是否已折叠
	visRows     []int        // 折叠后可见行的实际行号（按显示顺序）
	actualToVis map[int]int  // 实际行号 → 可见行索引（隐藏行映射到其上方可见行）

	// 软自动换行（详见 codeeditor_wrap.go）
	wrap      bool      // 是否开启换行（从 CodeEditor.WordWrap 复制，可 toggleWrap 翻转）
	wrapSegs  []wrapSeg // 视觉段（显示顺序，含折叠）；关闭时每可见行一个整行段
	wrapW     float64   // 上次构建用的编辑区文本宽，宽度变 → 重建
	wrapDirty bool      // 内容/折叠/可见行变化后置脏，下次 Paint 惰性重建

	// 缩略图 minimap
	showMinimap  bool       // 是否显示右侧缩略图
	miniRect     types.Rect // Paint 缓存：缩略图可点击区域
	miniScrollY  float64    // Paint 缓存：缩略图自身滚动偏移
	miniDragging bool       // 正在拖动缩略图

	// 多光标
	extraCarets  []caretSel // 额外光标（主光标仍是 cursor/anchor）
	suppressUndo bool       // 多光标编辑期间抑制重复 recordUndo

	// 智能补全
	completing bool       // 补全弹窗是否打开
	compItems  []compItem // 当前候选
	compSel    int        // 选中项
	compScroll int        // 弹窗滚动
	compStart  cePos      // 补全替换的起始位置（词前缀起点）

	// LSP（语言服务器，可选；接 gopls/tsserver 等做语义补全 + 诊断 + 跳转/引用/大纲/悬停）
	lspClient   *lsp.Client
	lspURI      string
	lspServer   string // 当前客户端启动用的服务器（切到需别的服务器的语言→重启，见 Update）
	lspVer      int
	lspReady    bool
	lspStarting bool // 嵌入体编辑器聚焦后正在启动 gopls（防重复启动）
	// 虚拟文档包装（内嵌函数体编辑器用）：把本地文本/坐标映射到完整 Go 文档喂 gopls。
	// lspWrap(本地文本) → (完整文档, 行偏移, 列偏移)；nil 表不包装（独立整文件编辑）。
	lspWrap            func(string) (string, int, int)
	lspLineOff, lspColOff int
	lspMu       sync.Mutex       // 保护 lspPending/diagnostics/pendingDef（读协程 vs UI 线程）
	lspPending  []compItem       // LSP 补全结果，待 UI 线程消费
	lspPendGen  int              // 待消费结果对应的补全代
	lspGen      int              // 当前补全请求代（防过期）
	diagnostics []lsp.Diagnostic // 当前诊断
	pendingDef  *pendingJump     // 转到定义结果，待 UI 线程回调宿主（跨线程：定义请求在协程）
	pendingRefs []CodeLoc        // 查找引用结果，待 UI 线程回调宿主
	pendingSyms []CodeSym        // 文档符号结果，待 UI 线程回调宿主

	hoverText    string       // 悬停内容（showHover 命令触发画浮层；空=不显示）
	hoverCursor  cePos        // 悬停时光标位置（光标移开则关闭浮层）
	pendingHover *hoverResult // 悬停结果，待 UI 线程显示

	pendingFormats []lsp.TextEdit // 格式化结果，待 UI 线程应用编辑

	// 诊断悬停提示
	diagHoverMsg  string // 鼠标悬停在诊断波浪线上时的诊断消息；空=不显示
	diagHoverLine int    // 悬停的诊断所在行
	diagHoverCol  int    // 悬停的诊断所在列
}

// hoverResult 异步悬停结果（内容 + 请求时的光标位置）。
type hoverResult struct {
	text string
	at   cePos
}

// pendingJump 转到定义的目标（文件 + 行/列，1 基行）。
type pendingJump struct {
	file string
	line int
	col  int
}

// caretSel 一个光标 + 其选区。
type caretSel struct{ cursor, anchor cePos }

// foldRange 一个可折叠区域（起止行号，含起始行）。
type foldRange struct{ start, end int }

// ceMatch 一处匹配（行内 [start,end)）。
type ceMatch struct{ start, end cePos }

// ceFindBtn 查找栏按钮命中区 + 动作标识。
type ceFindBtn struct {
	rect   types.Rect
	action string
}

// CursorClientPos 返回光标客户区坐标，供 app 层定位 IME 候选窗口（满足 imeCapable 接口）。
func (e *CodeEditorElement) CursorClientPos() (x, y float64) {
	return e.cursorClientX, e.cursorClientY
}

// ceSnapshot 文本+光标快照，供撤销/重做。
type ceSnapshot struct {
	lines  []string
	cursor cePos
	anchor cePos
}

func (e *CodeEditorElement) Build() []Element { return nil }

// ── 文本/位置工具 ──

func (e *CodeEditorElement) text() string { return strings.Join(e.lines, "\n") }

func (e *CodeEditorElement) lineRunes(line int) []rune {
	if line < 0 || line >= len(e.lines) {
		return nil
	}
	return []rune(e.lines[line])
}

// measure 用与渲染一致的 Skia 测量文本宽度（Paint 期间有 lastCanvas）；否则回退全局测量。
// 关键：定位（光标/点击/选区/IME 候选）必须用 Skia 实测，MeasureTextGlobal 对中文逐字累积偏差会漂移。
func (e *CodeEditorElement) measure(s string) float64 {
	if s == "" {
		return 0
	}
	// 测量缓存（同字体下 文本→宽度）：免每帧 maxLineWidth 等对整文件逐行重做 Skia CGO 测量。
	if e.measFont != e.font { // 字体变了（缩放等）→ 整体失效
		e.measCache = nil
		e.measFont = e.font
	}
	if w, ok := e.measCache[s]; ok {
		return w
	}
	var w float64
	if e.lastCanvas != nil {
		w = e.lastCanvas.MeasureText(s, e.font).Width
	} else {
		w = canvas.MeasureTextGlobal(s, e.font).Width
	}
	if e.measCache == nil || len(e.measCache) >= 20000 { // 上限防无界增长（光标子串多变）
		e.measCache = make(map[string]float64, 512)
	}
	e.measCache[s] = w
	return w
}

func (e *CodeEditorElement) clampPos(p cePos) cePos {
	if p.line < 0 {
		p.line = 0
	}
	if p.line >= len(e.lines) {
		p.line = len(e.lines) - 1
	}
	ln := len(e.lineRunes(p.line))
	if p.col < 0 {
		p.col = 0
	}
	if p.col > ln {
		p.col = ln
	}
	return p
}

func (e *CodeEditorElement) hasSel() bool { return !cePosEq(e.cursor, e.anchor) }
func (e *CodeEditorElement) sortedSel() (lo, hi cePos) {
	if cePosLess(e.cursor, e.anchor) {
		return e.cursor, e.anchor
	}
	return e.anchor, e.cursor
}

func (e *CodeEditorElement) selText() string {
	if !e.hasSel() {
		return ""
	}
	lo, hi := e.sortedSel()
	if lo.line == hi.line {
		r := e.lineRunes(lo.line)
		return string(r[lo.col:hi.col])
	}
	var sb strings.Builder
	first := e.lineRunes(lo.line)
	sb.WriteString(string(first[lo.col:]))
	for l := lo.line + 1; l < hi.line; l++ {
		sb.WriteByte('\n')
		sb.WriteString(e.lines[l])
	}
	sb.WriteByte('\n')
	last := e.lineRunes(hi.line)
	sb.WriteString(string(last[:hi.col]))
	return sb.String()
}

// deleteSel 删除选区，光标落到选区起点。返回是否删除了内容。
func (e *CodeEditorElement) deleteSel() bool {
	if !e.hasSel() {
		return false
	}
	lo, hi := e.sortedSel()
	loR := e.lineRunes(lo.line)
	hiR := e.lineRunes(hi.line)
	merged := string(loR[:lo.col]) + string(hiR[hi.col:])
	nl := append([]string{}, e.lines[:lo.line]...)
	nl = append(nl, merged)
	nl = append(nl, e.lines[hi.line+1:]...)
	e.lines = nl
	e.cursor = lo
	e.anchor = lo
	return true
}

// insertStr 在光标处插入文本（可含 \n），先删选区。
func (e *CodeEditorElement) insertStr(s string) {
	s = expandTabs(s) // 粘贴/输入的 tab 也展开，避免豆腐块
	if e.hasSel() {
		e.deleteSel()
	}
	cur := e.lineRunes(e.cursor.line)
	before := string(cur[:e.cursor.col])
	after := string(cur[e.cursor.col:])
	parts := strings.Split(s, "\n")
	if len(parts) == 1 {
		e.lines[e.cursor.line] = before + parts[0] + after
		e.cursor.col += len([]rune(parts[0]))
	} else {
		newLines := make([]string, 0, len(parts))
		newLines = append(newLines, before+parts[0])
		for k := 1; k < len(parts)-1; k++ {
			newLines = append(newLines, parts[k])
		}
		lastPart := parts[len(parts)-1]
		newLines = append(newLines, lastPart+after)
		nl := append([]string{}, e.lines[:e.cursor.line]...)
		nl = append(nl, newLines...)
		nl = append(nl, e.lines[e.cursor.line+1:]...)
		e.lines = nl
		e.cursor.line += len(parts) - 1
		e.cursor.col = len([]rune(lastPart))
	}
	e.anchor = e.cursor
	e.afterEdit()
}

// leadingIndent 返回某行的前导空白（用于回车自动缩进）。
func leadingIndent(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

func (e *CodeEditorElement) insertNewline() {
	if e.hasSel() {
		e.deleteSel()
	}
	indent := leadingIndent(e.lines[e.cursor.line])
	e.insertStr("\n" + indent)
	if e.ed.OnEnter != nil {
		e.ed.OnEnter()
	}
}

// ceAutoPairs 自动配对：开符 → 闭符（引号 open==close）。
var ceAutoPairs = map[rune]rune{'(': ')', '[': ']', '{': '}', '"': '"', '\'': '\'', '`': '`'}

// handleCharInput 处理一个可打印字符的输入，含自动配对/包裹/over-type。
func (e *CodeEditorElement) handleCharInput(ch rune) {
	if e.hasMultiCaret() { // 多光标：纯插入（不自动配对，简化）
		e.editEachCaret("insert", func() { e.insertStr(string(ch)) })
		return
	}
	closeCh, isPair := ceAutoPairs[ch]
	line := e.lineRunes(e.cursor.line)
	// 1. 有选区 + 开括号/引号 → 用括号包裹选区
	if isPair && e.hasSel() {
		e.recordUndo("wrap")
		sel := e.selText()
		e.insertStr(string(ch) + sel + string(closeCh))
		return
	}
	// 2. over-type：输入闭合符/引号且光标右侧正是它 → 跳过（光标右移）
	if e.cursor.col < len(line) && line[e.cursor.col] == ch && strings.ContainsRune(")]}\"'`", ch) {
		e.breakUndo()
		e.moveCursor(0, 1, false)
		return
	}
	// 3. 开括号/引号 → 插入配对，光标居中
	if isPair {
		// 引号防误配对：紧贴标识符时当普通输入（如 don't）
		if (ch == '"' || ch == '\'' || ch == '`') && e.cursor.col < len(line) && isIdentPart(line[e.cursor.col]) {
			e.recordUndo("insert")
			e.insertStr(string(ch))
			return
		}
		e.recordUndo("pair")
		e.insertStr(string(ch) + string(closeCh))
		e.cursor.col-- // 回到配对中间
		e.anchor = e.cursor
		e.cursorMoved = true
		repaint()
		return
	}
	// 4. 普通字符
	e.recordUndo("insert")
	e.insertStr(string(ch))
}

func (e *CodeEditorElement) backspace() {
	if e.deleteSel() {
		e.afterEdit()
		return
	}
	if e.cursor.col > 0 {
		r := e.lineRunes(e.cursor.line)
		// 删空配对：左为开符、右为其配对 → 一起删
		if e.cursor.col < len(r) {
			if cl, ok := ceAutoPairs[r[e.cursor.col-1]]; ok && r[e.cursor.col] == cl {
				e.lines[e.cursor.line] = string(r[:e.cursor.col-1]) + string(r[e.cursor.col+1:])
				e.cursor.col--
				e.anchor = e.cursor
				e.afterEdit()
				return
			}
		}
		e.lines[e.cursor.line] = string(r[:e.cursor.col-1]) + string(r[e.cursor.col:])
		e.cursor.col--
	} else if e.cursor.line > 0 {
		prev := e.lineRunes(e.cursor.line - 1)
		cur := e.lines[e.cursor.line]
		col := len(prev)
		e.lines[e.cursor.line-1] = string(prev) + cur
		e.lines = append(e.lines[:e.cursor.line], e.lines[e.cursor.line+1:]...)
		e.cursor.line--
		e.cursor.col = col
	}
	e.anchor = e.cursor
	e.afterEdit()
}

func (e *CodeEditorElement) deleteForward() {
	if e.deleteSel() {
		e.afterEdit()
		return
	}
	r := e.lineRunes(e.cursor.line)
	if e.cursor.col < len(r) {
		e.lines[e.cursor.line] = string(r[:e.cursor.col]) + string(r[e.cursor.col+1:])
	} else if e.cursor.line < len(e.lines)-1 {
		e.lines[e.cursor.line] = string(r) + e.lines[e.cursor.line+1]
		e.lines = append(e.lines[:e.cursor.line+1], e.lines[e.cursor.line+2:]...)
	}
	e.anchor = e.cursor
	e.afterEdit()
}

// ── 括号匹配 ──

var ceBracketOpen = map[rune]rune{'(': ')', '[': ']', '{': '}'}
var ceBracketClose = map[rune]rune{')': '(', ']': '[', '}': '{'}

// findMatchingBracket 若光标紧邻括号，返回该括号位置 a 与其匹配位置 b。
func (e *CodeEditorElement) findMatchingBracket() (a, b cePos, ok bool) {
	check := func(p cePos) (cePos, cePos, bool) {
		r := e.lineRunes(p.line)
		if p.col < 0 || p.col >= len(r) {
			return cePos{}, cePos{}, false
		}
		ch := r[p.col]
		if cl, isOpen := ceBracketOpen[ch]; isOpen {
			if m, f := e.scanBracket(p, ch, cl, +1); f {
				return p, m, true
			}
		}
		if op, isClose := ceBracketClose[ch]; isClose {
			if m, f := e.scanBracket(p, ch, op, -1); f {
				return p, m, true
			}
		}
		return cePos{}, cePos{}, false
	}
	if x, y, okk := check(cePos{e.cursor.line, e.cursor.col}); okk { // 光标右侧字符
		return x, y, true
	}
	if e.cursor.col > 0 {
		if x, y, okk := check(cePos{e.cursor.line, e.cursor.col - 1}); okk { // 光标左侧字符
			return x, y, true
		}
	}
	return cePos{}, cePos{}, false
}

// scanBracket 从 from 按 dir 找匹配括号（self=起点括号、match=目标括号）。
func (e *CodeEditorElement) scanBracket(from cePos, self, match rune, dir int) (cePos, bool) {
	depth := 1
	p := from
	for steps := 0; steps < 20000; steps++ { // 扫描上限，防大文件卡顿
		p = e.advancePos(p, dir)
		if p.line < 0 {
			return cePos{}, false
		}
		r := e.lineRunes(p.line)
		if p.col < 0 || p.col >= len(r) {
			continue
		}
		switch r[p.col] {
		case self:
			depth++
		case match:
			depth--
			if depth == 0 {
				return p, true
			}
		}
	}
	return cePos{}, false
}

// advancePos 在文本里按 dir 前进一个字符位置（跨行）；越界返回 line=-1。
func (e *CodeEditorElement) advancePos(p cePos, dir int) cePos {
	if dir > 0 {
		if p.col+1 < len(e.lineRunes(p.line)) {
			return cePos{p.line, p.col + 1}
		}
		if p.line+1 < len(e.lines) {
			return cePos{p.line + 1, 0}
		}
		return cePos{-1, 0}
	}
	if p.col-1 >= 0 {
		return cePos{p.line, p.col - 1}
	}
	if p.line-1 >= 0 {
		return cePos{p.line - 1, len(e.lineRunes(p.line-1)) - 1}
	}
	return cePos{-1, 0}
}

func (e *CodeEditorElement) afterEdit() {
	e.cursor = e.clampPos(e.cursor)
	e.anchor = e.clampPos(e.anchor)
	e.rehighlight()
	e.computeVisible() // 编辑后重新检测可折叠区域 + 可见行
	e.lspDidChange()   // 同步文档给语言服务器
	e.cursorMoved = true
	e.resetBlink()
	if e.ed.OnChange != nil {
		e.ed.OnChange(e.text())
	}
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// rehighlight 重算所有行的 token（跨行状态串联）。
func (e *CodeEditorElement) rehighlight() {
	e.bracketRev++ // 内容变 → enclosingBracket 缓存失效
	e.hl = make([][]ceToken, len(e.lines))
	st := stNormal
	for i, line := range e.lines {
		var toks []ceToken
		toks, st = tokenizeLine([]rune(line), e.lang, st)
		e.hl[i] = toks
	}
}

// ── 撤销/重做 ──

func (e *CodeEditorElement) snapshot() ceSnapshot {
	return ceSnapshot{append([]string{}, e.lines...), e.cursor, e.anchor}
}

// recordUndo 在每次编辑【前】调用。kind 用于合并：连续 insert/delete 合并为一个撤销单元。
func (e *CodeEditorElement) recordUndo(kind string) {
	if e.suppressUndo { // 多光标编辑期间只在外层记一次
		return
	}
	coalesce := (kind == "insert" && e.lastEdit == "insert") ||
		(kind == "delete" && e.lastEdit == "delete")
	if !coalesce {
		e.undoStack = append(e.undoStack, e.snapshot())
		e.redoStack = nil
		if len(e.undoStack) > 800 { // 限制栈深
			e.undoStack = e.undoStack[len(e.undoStack)-800:]
		}
	}
	e.lastEdit = kind
}

// breakUndo 断开合并（光标移动/点击后，下次输入另起撤销单元）。
func (e *CodeEditorElement) breakUndo() { e.lastEdit = "" }

func (e *CodeEditorElement) canUndo() bool { return len(e.undoStack) > 0 }
func (e *CodeEditorElement) canRedo() bool { return len(e.redoStack) > 0 }

func (e *CodeEditorElement) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	e.redoStack = append(e.redoStack, e.snapshot())
	snap := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.restoreSnap(snap)
}

func (e *CodeEditorElement) redo() {
	if len(e.redoStack) == 0 {
		return
	}
	e.undoStack = append(e.undoStack, e.snapshot())
	snap := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.restoreSnap(snap)
}

func (e *CodeEditorElement) restoreSnap(s ceSnapshot) {
	e.lines = append([]string{}, s.lines...)
	e.cursor = e.clampPos(s.cursor)
	e.anchor = e.clampPos(s.anchor)
	e.extraCarets = nil // 撤销/重做回到单光标
	e.lastEdit = ""
	e.rehighlight()
	e.cursorMoved = true
	e.resetBlink()
	if e.ed.OnChange != nil {
		e.ed.OnChange(e.text())
	}
	repaint()
}

func (e *CodeEditorElement) resetBlink() { e.focusTime = time.Now() }
func (e *CodeEditorElement) isCursorVisible() bool {
	if !e.focused {
		return false
	}
	return time.Since(e.focusTime).Milliseconds()%1000 < 530
}
func (e *CodeEditorElement) Focus() {
	e.focused = true
	focusedCodeEditor = e
	lastFocusedCodeEditor = e // 记住最近聚焦者：宿主右键菜单弹出时本编辑器会失焦，命令/勾选仍要命中它
	e.resetBlink()
	repaint()
}
func (e *CodeEditorElement) Blur() {
	e.focused = false
	if focusedCodeEditor == e {
		focusedCodeEditor = nil // 注意：不清 lastFocusedCodeEditor，否则菜单一弹出命令就没目标了
	}
	repaint()
}
func (e *CodeEditorElement) IsFocused() bool { return e.focused }

// focusedCodeEditor 当前聚焦的代码编辑器；lastFocusedCodeEditor 最近聚焦的（失焦后仍保留）。
var (
	focusedCodeEditor     *CodeEditorElement
	lastFocusedCodeEditor *CodeEditorElement
)

// activeCodeEditor 命令派发目标：优先当前聚焦，否则最近聚焦——宿主菜单（覆盖层）弹出会令编辑器失焦，
// 但菜单里的编辑命令/勾选仍应作用于刚才那个编辑器，故用最近聚焦兜底。
func activeCodeEditor() *CodeEditorElement {
	if focusedCodeEditor != nil {
		return focusedCodeEditor
	}
	return lastFocusedCodeEditor
}

// RunEditorCommand 把编辑命令派发给当前(或最近)聚焦的代码编辑器（无则返回 false）。
// cmd: undo / redo / cut / copy / paste / selectAll / format / toggleWrap。供右键菜单/标题栏「编辑」菜单调用。
func RunEditorCommand(cmd string) bool {
	ed := activeCodeEditor()
	if ed == nil {
		return false
	}
	ed.runCommand(cmd)
	return true
}

// SuppressEditorContextMenu 为 true 时 CodeEditor 不弹自带右键菜单，放行事件冒泡给宿主自定义菜单。
var SuppressEditorContextMenu bool

// HasFocusedEditor 当前(或最近)是否有代码编辑器（供宿主菜单决定剪切/复制等项是否可用）。
func HasFocusedEditor() bool { return activeCodeEditor() != nil }

// EditorWrapEnabled 软自动换行是否开启（现为全局开关，等价 WordWrapEnabled）。保留兼容旧调用。
func EditorWrapEnabled() bool { return globalWordWrap }

// runCommand 执行一条编辑命令（与右键菜单 contextItems 同源逻辑）。
func (e *CodeEditorElement) runCommand(cmd string) {
	switch cmd {
	case "gotoDefinition":
		e.requestDefinition() // 转到定义（LSP 解析光标处 → 回调宿主打开目标）
	case "findReferences":
		e.requestReferences() // 查找引用（→ 回调宿主列出所有引用）
	case "documentSymbol":
		e.requestDocumentSymbol() // 文档符号大纲（→ 回调宿主列出符号）
	case "showHover":
		e.requestHover() // 悬停信息（→ 编辑器内画浮层）
	case "undo":
		e.undo()
	case "redo":
		e.redo()
	case "cut":
		if e.hasSel() && ClipboardWrite != nil {
			ClipboardWrite(e.selText())
			e.recordUndo("cut")
			e.deleteSel()
			e.afterEdit()
		}
	case "copy":
		if e.hasSel() && ClipboardWrite != nil {
			ClipboardWrite(e.selText())
		}
	case "paste":
		if ClipboardRead != nil {
			e.recordUndo("paste")
			e.insertStr(ClipboardRead())
		}
	case "selectAll":
		e.anchor = cePos{0, 0}
		last := len(e.lines) - 1
		e.cursor = cePos{last, len(e.lineRunes(last))}
		repaint()
	case "format":
		e.formatGo()
	case "toggleWrap":
		ToggleWordWrap() // 改全局开关：各编辑器下次绘制经 ensureWrapSegs 跟随（不依赖焦点）
	}
}

// formatGo 用 gofmt 格式化当前文档（仅当内容是合法 Go；非 Go/语法错误则原样不动）。
func (e *CodeEditorElement) formatGo() {
	src := e.text()
	out, err := format.Source([]byte(src))
	if err != nil {
		return // 非 Go 或有语法错误 → 不动
	}
	formatted := strings.TrimRight(string(out), "\n")
	if formatted == strings.TrimRight(src, "\n") {
		return // 已是规范格式
	}
	e.recordUndo("format")
	e.lines = strings.Split(formatted, "\n")
	if len(e.lines) == 0 {
		e.lines = []string{""}
	}
	if e.cursor.line >= len(e.lines) {
		e.cursor.line = len(e.lines) - 1
	}
	if n := len(e.lineRunes(e.cursor.line)); e.cursor.col > n {
		e.cursor.col = n
	}
	e.anchor = e.cursor
	e.afterEdit()
	repaint()
}

// EmbeddedContentHeight 嵌入模式下按内容算自然高度（可见行数*行高 + 边距，至少 3 行）。
// 使代码区像变量表一样"恰好包住内容"地流式排布、融入 StructEditor，纵向不再单独滚动（由外层滚）。
func (e *CodeEditorElement) EmbeddedContentHeight() float64 {
	rows := len(e.visRows)
	if e.wrap && len(e.wrapSegs) > 0 { // 换行：按视觉段数算高度（每段一行）
		rows = len(e.wrapSegs)
	}
	if rows < 3 {
		rows = 3 // 至少留 3 行，太矮不便点击
	}
	h := float64(rows)*ceLineH + 8
	if !e.wrap && e.maxLineWidth() > e.size.Width-e.gutterW-ceTextPad {
		h += sbThick // 长行需要横向滚动条占位（换行时无横条）
	}
	return h
}

func repaint() {
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// ── Layout ──

func (e *CodeEditorElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w, h := e.ed.Width, e.ed.Height
	if w <= 0 {
		w = 680
	}
	if h <= 0 {
		h = 360
	}
	// 行号栏宽度：按最大行号位数（含全局偏移）
	digits := len(itoaCE(len(e.lines) + e.ed.LineNumberOffset))
	if digits < 2 {
		digits = 2
	}
	charW := canvas.MeasureTextGlobal("0", e.font).Width
	e.gutterW = float64(digits)*charW + ceGutterPad + ceFoldW // 行号 + 右侧折叠箭头列
	if e.ed.GutterOverride > 0 {
		e.gutterW = e.ed.GutterOverride // StructEditor 统一行号栏宽度，与变量表行号列对齐
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func itoaCE(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
