package widget

// CodeWorkbench 同一个 Go 程序的「传统代码视图 ⇄ 表格代码视图」切换器。
// 顶部一个切换按钮：
//   表格视图 = StructEditor(Go 模式)，ParseGo 把 Go 代码变成可编辑表格；
//   代码视图 = CodeEditor，显示 ToGo 生成的 Go 源码。
// 切换时双向同步：表格→ToGo→文本、文本→ParseGo→表格（同一份程序，互不丢失）。

type CodeWorkbench struct {
	StatefulWidget
	Width, Height float64
	initialGo     string
	lang          string       // 语言（默认 "go"）；决定 LanguageProvider(解析/生成) 与代码区高亮
	fontFamily    string       // 代码视图等宽字体族（空=Consolas）
	fontSize      float64      // 代码视图字号（<=0 用默认 14）
	fontBold      bool         // 代码视图加粗
	fontItalic    bool         // 代码视图斜体
	fontUnderline bool         // 代码视图下划线
	ReloadToken   int          // 递增时强制重新初始化（切换文件用）
	OnChange      func(string) // 内容变化回调（用户编辑或视图切换）
	OnCursorMove  func(line, col int) // 光标位置变化回调（0 基，转发给宿主状态栏显示 Ln/Col）

	// LSP（接 gopls / tsserver 等）：代码视图的整文件编辑器据此做语义补全/诊断/转到定义/查找引用/大纲/悬停。空=纯词法。
	LSPServer, LSPWorkspace, LSPFile, LSPLangID string
	LSPArgs                                     []string
	OnGoToDefinition                            func(file string, line, col int)
	OnReferences                                func(refs []CodeLoc)
	OnDocumentSymbols                           func(syms []CodeSym)
}

// WithLSP 给代码视图接入语言服务器（gopls 等）。companion 标签编辑器对 Go 文件用。
func (w *CodeWorkbench) WithLSP(server, workspaceURI, fileURI string) *CodeWorkbench {
	w.LSPServer, w.LSPWorkspace, w.LSPFile = server, workspaceURI, fileURI
	return w
}

// NewCodeWorkbench 以一段源码初始化（默认 Go、进表格视图）。其他语言用 WithLang 切换。
func NewCodeWorkbench(goSrc string) *CodeWorkbench {
	return &CodeWorkbench{Width: 760, Height: 600, initialGo: goSrc, lang: "go"}
}
func (w *CodeWorkbench) WithSize(wd, h float64) *CodeWorkbench { w.Width, w.Height = wd, h; return w }

// WithLang 设置语言（需已注册对应 LanguageProvider，如 "go"/"ey"）。
func (w *CodeWorkbench) WithLang(l string) *CodeWorkbench { w.lang = l; return w }

// WithFontFamily 设置代码视图的等宽字体族（空=Consolas）。
func (w *CodeWorkbench) WithFontFamily(f string) *CodeWorkbench { w.fontFamily = f; return w }

// WithFontSize 设置代码视图字号（<=0 用默认 14）。
func (w *CodeWorkbench) WithFontSize(s float64) *CodeWorkbench { w.fontSize = s; return w }

// WithFontStyle 设置代码视图加粗/斜体/下划线。
func (w *CodeWorkbench) WithFontStyle(bold, italic, underline bool) *CodeWorkbench {
	w.fontBold, w.fontItalic, w.fontUnderline = bold, italic, underline
	return w
}

// theCwState 代码工作台的包级单例状态——避免宿主（编辑器面板）因 dirty 重建时嵌套 StatefulWidget
// 状态丢失（否则切换视图后宿主重建→工作台被重置回表格视图，表现为「切换代码视图点击失败」）。
var theCwState = &cwState{}

func (w *CodeWorkbench) CreateState() State { return theCwState }

// ToggleWorkbenchView 切换代码工作台「代码⇄表格」视图（供宿主自定义右键菜单调用）。
func ToggleWorkbenchView() { theCwState.toggle() }

// WorkbenchModeIsText 代码工作台当前是否为代码(文本)视图。
func WorkbenchModeIsText() bool { return theCwState.mode == "text" }

// provider 取当前语言的适配器（未注册兜底 Go）。
func (s *cwState) provider() LanguageProvider {
	if p := providerFor(s.cfg().lang); p != nil {
		return p
	}
	return goLangProvider{}
}

type cwState struct {
	BaseState
	program     *SEProgram
	codeText    string
	savedGo     string // 进表格时的 ToGo 基线：切回时若表格没改过(ToGo 仍等于它)，还原原始 codeText（保留用户空行/格式）
	mode        string // "table" / "text"
	inited      bool
	lastReload  int            // 上次 ReloadToken，用于检测「切换文件」需重置内容
	codeCur     *CECursorState // 代码视图光标/滚动（切换间保持）
	tableScroll *float64       // 表格视图滚动位置（切换间保持）
}

func (s *cwState) cfg() *CodeWorkbench {
	if w, ok := s.Widget().(*CodeWorkbench); ok {
		return w
	}
	return &CodeWorkbench{}
}

// toggle 切换视图并同步内容（经 LanguageProvider 解析/生成，语言无关）。
func (s *cwState) toggle() {
	prov := s.provider()
	c := s.cfg()
	if s.mode == "table" { // 表格→代码
		if cur := prov.Generate(s.program); cur != s.savedGo {
			s.codeText = cur // 表格被编辑过 → 用新生成的代码
		} // 否则表格没动 → 保留原始 codeText（不丢用户空行/格式）
		s.mode = "text"
		if c.OnChange != nil {
			c.OnChange(s.codeText)
		}
	} else { // 代码→表格：解析代码文本（失败则保留原表格），记生成基线
		if p, err := prov.Parse(s.codeText); err == nil && p != nil {
			s.program = p
			s.savedGo = prov.Generate(p)
		}
		s.mode = "table"
		if c.OnChange != nil {
			c.OnChange(s.codeText)
		}
	}
	s.SetState()
}

func (s *cwState) Build(ctx BuildContext) Widget {
	c := s.cfg()
	if c.ReloadToken != s.lastReload {
		s.lastReload = c.ReloadToken
		s.inited = false
	}
	if !s.inited {
		s.inited = true
		s.mode = "text" // 默认代码视图（结构化表格视图暂缓，仍可经右键菜单切换）
		s.codeCur = &CECursorState{}
		s.tableScroll = new(float64)
		prov := s.provider()
		s.codeText = c.initialGo
		if p, err := prov.Parse(s.codeText); err == nil && p != nil {
			s.program = p
		} else {
			s.program = &SEProgram{}
		}
		s.savedGo = prov.Generate(s.program) // 生成基线（切回代码时判表格是否被改）
	}
	var content Widget
	if s.mode == "table" {
		se := NewStructEditor(s.program).WithLang(c.lang).WithSize(c.Width, c.Height)
		se.ScrollRef = s.tableScroll // 切回表格恢复滚动位置
		content = se
	} else {
		ed := NewCodeEditor(c.lang, s.codeText).WithSize(c.Width, c.Height).WithFontFamily(c.fontFamily).WithFontSize(c.fontSize).WithFontStyle(c.fontBold, c.fontItalic, c.fontUnderline)
		ed.IndentGuides = true
		ed.CursorRef = s.codeCur   // 切回代码恢复光标/滚动
		ed.OnCursorMove = c.OnCursorMove // 光标位置变化 → 转发到宿主状态栏
		ed.ReloadToken = c.ReloadToken // 跟随工作台重载令牌：切文件时重置内容 + 切 LSP 文档
		if c.LSPServer != "" {     // 代码视图整文件直接喂语言服务器（无需函数体坐标映射）
			ed.LSPServer, ed.LSPArgs, ed.LSPWorkspace, ed.LSPFile, ed.LSPLangID = c.LSPServer, c.LSPArgs, c.LSPWorkspace, c.LSPFile, c.LSPLangID
			ed.OnGoToDefinition = c.OnGoToDefinition
			ed.OnReferences = c.OnReferences
			ed.OnDocumentSymbols = c.OnDocumentSymbols
		}
		ed.OnChange = func(t string) {
			s.codeText = t
			if c.OnChange != nil {
				c.OnChange(t)
			}
		}
		content = ed
	}
	// 直接返回内容（无切换按钮，视图切换走右键菜单 widget.ToggleWorkbenchView）；
	// 内容用真实视口宽（宿主 Expanded 约束），表格列不再被撑出屏外。
	return content
}
