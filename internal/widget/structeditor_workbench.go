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
	lang          string // 语言（默认 "go"）；决定 LanguageProvider(解析/生成) 与代码区高亮
	ReloadToken   int           // 递增时强制重新初始化（切换文件用）
	OnChange      func(string)  // 内容变化回调（用户编辑或视图切换）
}

// NewCodeWorkbench 以一段源码初始化（默认 Go、进表格视图）。其他语言用 WithLang 切换。
func NewCodeWorkbench(goSrc string) *CodeWorkbench {
	return &CodeWorkbench{Width: 760, Height: 600, initialGo: goSrc, lang: "go"}
}
func (w *CodeWorkbench) WithSize(wd, h float64) *CodeWorkbench { w.Width, w.Height = wd, h; return w }

// WithLang 设置语言（需已注册对应 LanguageProvider，如 "go"/"ey"）。
func (w *CodeWorkbench) WithLang(l string) *CodeWorkbench { w.lang = l; return w }
func (w *CodeWorkbench) CreateState() State               { return &cwState{} }

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
		s.mode = "table"
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
	btn := "切换到代码视图"
	if s.mode == "text" {
		btn = "切换到表格视图"
	}
	var content Widget
	if s.mode == "table" {
		se := NewStructEditor(s.program).WithLang(c.lang).WithSize(c.Width, c.Height-46)
		se.ScrollRef = s.tableScroll // 切回表格恢复滚动位置
		content = se
	} else {
		ed := NewCodeEditor(c.lang, s.codeText).WithSize(c.Width, c.Height-46)
		ed.IndentGuides = true
		ed.CursorRef = s.codeCur // 切回代码恢复光标/滚动
		ed.OnChange = func(t string) {
			s.codeText = t
			if c.OnChange != nil {
				c.OnChange(t)
			}
		}
		content = ed
	}
	return Div(
		Style{Width: c.Width, Height: c.Height, FlexDirection: "column", Gap: 8},
		NewButton(btn, s.toggle).WithMinWidth(150),
		content,
	)
}
