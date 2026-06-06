package widget

import (
	"strings"
	"testing"

	"github.com/user/goui/internal/event"
)

// TestCodeEditorReloadToken 验证「切换文件」受控重载：ReloadToken 变 → 内容重置；不变 → 保留。
func TestCodeEditorReloadToken(t *testing.T) {
	a := NewCodeEditor("go", "package a\nfunc A() {}")
	el := a.CreateElement().(*CodeEditorElement)
	if got := strings.Join(el.lines, "\n"); got != "package a\nfunc A() {}" {
		t.Fatalf("初始内容不符: %q", got)
	}
	// 切换文件：新内容 B + ReloadToken+1（goui relayout 复用 Element → 走 Update）
	b := NewCodeEditor("go", "package b\nvar X = 1")
	b.ReloadToken = a.ReloadToken + 1
	el.Update(b)
	if got := strings.Join(el.lines, "\n"); got != "package b\nvar X = 1" {
		t.Errorf("ReloadToken 变化后应重置为 B，实际 %q", got)
	}
	// token 不变 → 保留运行时编辑（不被 relayout 冲掉）
	el.lines = []string{"edited"}
	c := NewCodeEditor("go", "package c")
	c.ReloadToken = b.ReloadToken
	el.Update(c)
	if got := strings.Join(el.lines, "\n"); got != "edited" {
		t.Errorf("token 未变不应重置，实际 %q", got)
	}
}

// TestCodeHighlightGo 验证 Go 词法：关键字 + 注释识别。
func TestCodeHighlightGo(t *testing.T) {
	toks, st := tokenizeLine([]rune(`func main() { // hi`), ceLangGo, stNormal)
	if st != stNormal {
		t.Errorf("行尾状态应为 normal，实际 %d", st)
	}
	kw, comment := false, false
	for _, tk := range toks {
		if tk.kind == tkKeyword {
			kw = true
		}
		if tk.kind == tkComment {
			comment = true
		}
	}
	if !kw {
		t.Errorf("应识别关键字 func；toks=%+v", toks)
	}
	if !comment {
		t.Errorf("应识别 // 注释；toks=%+v", toks)
	}
}

// TestCodeHighlightBlockComment 验证跨行块注释状态传递。
func TestCodeHighlightBlockComment(t *testing.T) {
	_, st := tokenizeLine([]rune(`x := 1 /* open`), ceLangGo, stNormal)
	if st != stBlockComment {
		t.Errorf("行尾应处于块注释状态，实际 %d", st)
	}
	toks2, st2 := tokenizeLine([]rune(`still */ y := 2`), ceLangGo, stBlockComment)
	if st2 != stNormal {
		t.Error("块注释应在本行闭合")
	}
	if len(toks2) == 0 || toks2[0].kind != tkComment {
		t.Errorf("首 token 应为注释；toks=%+v", toks2)
	}
}

// TestCodeEditorEdit 验证插入 + 回车自动缩进。
func TestCodeEditorEdit(t *testing.T) {
	ce := NewCodeEditor("go", "ab")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor = cePos{0, 2}
	e.anchor = e.cursor // 无选区
	e.insertStr("c")
	if e.text() != "abc" {
		t.Errorf("插入后应为 abc，实际 %q", e.text())
	}
	ce2 := NewCodeEditor("go", "    x")
	e2 := ce2.CreateElement().(*CodeEditorElement)
	e2.cursor = cePos{0, 5}
	e2.anchor = e2.cursor // 无选区
	e2.insertNewline()
	if e2.text() != "    x\n    " {
		t.Errorf("回车自动缩进失败，实际 %q", e2.text())
	}
	if e2.cursor.line != 1 || e2.cursor.col != 4 {
		t.Errorf("光标应在 (1,4)，实际 %+v", e2.cursor)
	}
}

// TestCodeEditorSelection 验证跨行选区取文本 + 删除合并。
func TestCodeEditorSelection(t *testing.T) {
	ce := NewCodeEditor("go", "hello\nworld")
	e := ce.CreateElement().(*CodeEditorElement)
	e.anchor = cePos{0, 2}
	e.cursor = cePos{1, 2}
	if e.selText() != "llo\nwo" {
		t.Errorf("跨行选区文本应为 'llo\\nwo'，实际 %q", e.selText())
	}
	e.deleteSel()
	if e.text() != "herld" {
		t.Errorf("删除跨行选区后应为 herld，实际 %q", e.text())
	}
}

// TestCodeEditorUndoRedo 验证撤销/重做。
func TestCodeEditorUndoRedo(t *testing.T) {
	ce := NewCodeEditor("go", "")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor, e.anchor = cePos{0, 0}, cePos{0, 0}
	e.recordUndo("insert")
	e.insertStr("a")
	e.breakUndo()
	e.recordUndo("insert")
	e.insertStr("b")
	if e.text() != "ab" {
		t.Fatalf("应为 ab，实际 %q", e.text())
	}
	e.undo()
	if e.text() != "a" {
		t.Errorf("撤销一次应为 a，实际 %q", e.text())
	}
	e.undo()
	if e.text() != "" {
		t.Errorf("撤销两次应为空，实际 %q", e.text())
	}
	e.redo()
	if e.text() != "a" {
		t.Errorf("重做应为 a，实际 %q", e.text())
	}
}

// TestCodeEditorAutoPair 验证自动配对 + over-type。
func TestCodeEditorAutoPair(t *testing.T) {
	ce := NewCodeEditor("go", "")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor, e.anchor = cePos{0, 0}, cePos{0, 0}
	e.handleCharInput('(')
	if e.text() != "()" {
		t.Errorf("自动配对应为 ()，实际 %q", e.text())
	}
	if e.cursor.col != 1 {
		t.Errorf("光标应在配对中间(1)，实际 %d", e.cursor.col)
	}
	e.handleCharInput(')')
	if e.text() != "()" || e.cursor.col != 2 {
		t.Errorf("over-type 应跳过不插入、光标右移；text=%q col=%d", e.text(), e.cursor.col)
	}
}

// TestCodeEditorWrapSelection 验证选区被括号包裹。
func TestCodeEditorWrapSelection(t *testing.T) {
	ce := NewCodeEditor("go", "ab")
	e := ce.CreateElement().(*CodeEditorElement)
	e.anchor, e.cursor = cePos{0, 0}, cePos{0, 2}
	e.handleCharInput('(')
	if e.text() != "(ab)" {
		t.Errorf("包裹选区应为 (ab)，实际 %q", e.text())
	}
}

// TestCodeEditorBracketMatch 验证括号匹配查找。
func TestCodeEditorBracketMatch(t *testing.T) {
	ce := NewCodeEditor("go", "x(ab)y")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor = cePos{0, 1} // 紧邻 '('
	a, b, ok := e.findMatchingBracket()
	if !ok {
		t.Fatal("应找到匹配括号")
	}
	if a.col != 1 || b.col != 4 {
		t.Errorf("匹配应为 col 1↔4，实际 a=%+v b=%+v", a, b)
	}
}

// TestCodeEditorFind 验证查找匹配统计 + 大小写敏感。
func TestCodeEditorFind(t *testing.T) {
	ce := NewCodeEditor("go", "Foo foo FOO\nfoo")
	e := ce.CreateElement().(*CodeEditorElement)
	e.findQuery = "foo"
	e.findCase = false
	e.recomputeMatches()
	if len(e.matches) != 4 {
		t.Errorf("不区分大小写应找到 4 处，实际 %d", len(e.matches))
	}
	e.findCase = true
	e.recomputeMatches()
	if len(e.matches) != 2 {
		t.Errorf("区分大小写应找到 2 处，实际 %d", len(e.matches))
	}
}

// TestCodeEditorReplaceAll 验证全部替换。
func TestCodeEditorReplaceAll(t *testing.T) {
	ce := NewCodeEditor("go", "foo bar foo")
	e := ce.CreateElement().(*CodeEditorElement)
	e.findQuery = "foo"
	e.replaceQuery = "X"
	e.recomputeMatches()
	e.replaceAll()
	if e.text() != "X bar X" {
		t.Errorf("全部替换后应为 'X bar X'，实际 %q", e.text())
	}
}

// TestCodeEditorReplaceCurrent 验证替换当前匹配。
func TestCodeEditorReplaceCurrent(t *testing.T) {
	ce := NewCodeEditor("go", "aa aa")
	e := ce.CreateElement().(*CodeEditorElement)
	e.findQuery = "aa"
	e.replaceQuery = "b"
	e.recomputeMatches()
	e.curMatch = 0
	e.replaceCurrent()
	if e.text() != "b aa" {
		t.Errorf("替换当前后应为 'b aa'，实际 %q", e.text())
	}
}

// TestCodeEditorFolds 验证折叠区域检测 + 折叠/展开可见行。
func TestCodeEditorFolds(t *testing.T) {
	src := "func a() {\n  x := 1\n  y := 2\n}\nfunc b() {}"
	ce := NewCodeEditor("go", src)
	e := ce.CreateElement().(*CodeEditorElement)
	if !e.isFoldStart(0) {
		t.Errorf("第 0 行应可折叠；ranges=%+v", e.foldRanges)
	}
	if end, ok := e.foldEndOf(0); !ok || end != 3 {
		t.Errorf("第 0 行折叠应止于第 3 行，实际 %d ok=%v", end, ok)
	}
	if e.isFoldStart(4) { // func b() {} 同行闭合，不可折叠
		t.Errorf("同行 {} 不应可折叠")
	}
	if len(e.visRows) != 5 {
		t.Errorf("初始 5 行全可见，实际 %d", len(e.visRows))
	}
	e.toggleFold(0)
	if len(e.visRows) != 2 || e.visRows[0] != 0 || e.visRows[1] != 4 {
		t.Errorf("折叠后应只见 [0,4]，实际 %v", e.visRows)
	}
	e.toggleFold(0)
	if len(e.visRows) != 5 {
		t.Errorf("展开后应恢复 5 行，实际 %d", len(e.visRows))
	}
}

// TestCodeEditorMultiCaretSameLine 验证同行多光标同时插入（靠前编辑修正靠后光标）。
func TestCodeEditorMultiCaretSameLine(t *testing.T) {
	ce := NewCodeEditor("go", "abcd")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor, e.anchor = cePos{0, 1}, cePos{0, 1}
	e.extraCarets = []caretSel{{cePos{0, 3}, cePos{0, 3}}}
	e.editEachCaret("insert", func() { e.insertStr("X") })
	if e.text() != "aXbcXd" {
		t.Errorf("同行多光标插入应为 'aXbcXd'，实际 %q", e.text())
	}
	if len(e.extraCarets) != 1 {
		t.Errorf("应保留 1 个额外光标，实际 %d", len(e.extraCarets))
	}
}

// TestCodeEditorMultiCaretCrossLine 验证跨行多光标同时插入。
func TestCodeEditorMultiCaretCrossLine(t *testing.T) {
	ce := NewCodeEditor("go", "ab\ncd")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor, e.anchor = cePos{0, 2}, cePos{0, 2}
	e.extraCarets = []caretSel{{cePos{1, 2}, cePos{1, 2}}}
	e.editEachCaret("insert", func() { e.insertStr("X") })
	if e.text() != "abX\ncdX" {
		t.Errorf("跨行多光标插入应为 'abX\\ncdX'，实际 %q", e.text())
	}
}

// TestCodeEditorMultiCaretNewline 验证多光标回车（行数变化时后续光标偏移正确）。
func TestCodeEditorMultiCaretNewline(t *testing.T) {
	ce := NewCodeEditor("go", "ab")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor, e.anchor = cePos{0, 0}, cePos{0, 0}
	e.extraCarets = []caretSel{{cePos{0, 2}, cePos{0, 2}}}
	e.editEachCaret("newline", e.insertNewline)
	if e.text() != "\nab\n" {
		t.Errorf("多光标回车应为 '\\nab\\n'，实际 %q", e.text())
	}
}

// TestCodeEditorCompletion 验证词法补全：文件标识符 + 关键字候选 + 接受替换。
func TestCodeEditorCompletion(t *testing.T) {
	ce := NewCodeEditor("go", "result := 1\nx := re")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor = cePos{1, 7} // "x := re" 末尾
	e.anchor = e.cursor
	e.updateCompletion()
	if !e.completing {
		t.Fatal("应触发补全")
	}
	hasResult, hasReturn := false, false
	for i, it := range e.compItems {
		if it.label == "result" {
			hasResult = true
			e.compSel = i
		}
		if it.label == "return" {
			hasReturn = true
		}
	}
	if !hasResult {
		t.Errorf("候选应含文件标识符 'result'；items=%+v", e.compItems)
	}
	if !hasReturn {
		t.Errorf("候选应含关键字 'return'")
	}
	e.acceptCompletion()
	if e.lines[1] != "x := result" {
		t.Errorf("接受补全后应为 'x := result'，实际 %q", e.lines[1])
	}
}

// TestRegisterLanguage 验证自定义语言注册 + 高亮生效。
func TestRegisterLanguage(t *testing.T) {
	RegisterLanguage("mylang", LangSpec{
		Keywords:    []string{"when", "unless"},
		Types:       []string{"yes", "no"},
		LineComment: ";",
		Quotes:      "\"",
	})
	lang := ceLangFor("MyLang") // 不区分大小写
	if !lang.keywords["when"] || !lang.types["yes"] || lang.lineComment != ";" {
		t.Fatalf("自定义语言定义未生效：%+v", lang)
	}
	toks, _ := tokenizeLine([]rune(`when yes ; 注释`), lang, stNormal)
	kw, comment := false, false
	for _, tk := range toks {
		if tk.kind == tkKeyword {
			kw = true
		}
		if tk.kind == tkComment {
			comment = true
		}
	}
	if !kw {
		t.Error("自定义关键字 when 应高亮为关键字")
	}
	if !comment {
		t.Error("应识别 ; 行注释")
	}
}

// TestBuiltinLanguages 验证新增内置语言。
func TestBuiltinLanguages(t *testing.T) {
	if !ceLangFor("rust").keywords["fn"] {
		t.Error("rust 应识别 fn")
	}
	if !ceLangFor("cpp").keywords["typedef"] {
		t.Error("c/cpp 应识别 typedef")
	}
	if !ceLangFor("java").keywords["public"] {
		t.Error("java 应识别 public")
	}
}

// TestRegisterLanguageJSON 验证从 JSON 配置注册语言 + 别名。
func TestRegisterLanguageJSON(t *testing.T) {
	err := RegisterLanguageJSON([]byte(`{"name":"jsonlang","aliases":["jl"],"keywords":["alpha","beta"],"types":["gamma"],"lineComment":"//","quotes":"\""}`))
	if err != nil {
		t.Fatal(err)
	}
	if !ceLangFor("jsonlang").keywords["alpha"] {
		t.Error("JSON 注册的关键字未生效")
	}
	if !ceLangFor("jl").keywords["beta"] {
		t.Error("别名 jl 未生效")
	}
	if !ceLangFor("jsonlang").types["gamma"] {
		t.Error("JSON 注册的类型未生效")
	}
}

// TestFrontendConfigLangs 验证内置前端/配置文件语言。
func TestFrontendConfigLangs(t *testing.T) {
	cases := map[string]string{
		"css": "color", "scss": "display", "html": "div", "vue": "span",
		"yaml": "true", "sql": "SELECT", "bash": "echo", "toml": "true",
		"ts": "interface", "tsx": "type", "csharp": "namespace", "php": "function",
		"ruby": "def", "kotlin": "fun", "swift": "guard", "lua": "function",
		"dart": "factory", "powershell": "param",
	}
	for lang, kw := range cases {
		if !ceLangFor(lang).keywords[kw] {
			t.Errorf("%s 应识别关键字 %q", lang, kw)
		}
	}
}

// TestStructEditor 验证表格化结构编辑器数据模型 + 实例化。
func TestStructEditor(t *testing.T) {
	p := DemoFactorialProgram()
	if len(p.Globals) != 2 {
		t.Errorf("程序集变量应 2 个，实际 %d", len(p.Globals))
	}
	if len(p.Subs) != 2 || p.Subs[0].Name != "阶乘" {
		t.Errorf("子程序结构不对：%+v", p.Subs)
	}
	if p.Subs[0].Body == "" {
		t.Errorf("阶乘子程序应有逻辑代码 Body")
	}
	se := NewStructEditor(p)
	el := se.CreateElement().(*StructEditorElement)
	if el.program != p {
		t.Error("program 未关联到 element")
	}
}

// TestStructEditorCellEdit 验证单元格编辑写回 program 数据。
func TestStructEditorCellEdit(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	e.beginEdit("globals", 0, 0) // 程序集变量 结果 的名称
	if e.cellValue("globals", 0, 0) != "结果" {
		t.Fatalf("读取应为 结果，实际 %q", e.cellValue("globals", 0, 0))
	}
	e.editInsert('X') // 末尾插入
	if p.Globals[0].Name != "结果X" {
		t.Errorf("编辑后应为 结果X，实际 %q", p.Globals[0].Name)
	}
	e.beginEdit("locals:0", 1, 3) // 阶乘子程序 局部变量「积」的备注
	e.editInsert('!')
	if p.Subs[0].Locals[1].Note != "累乘结果!" {
		t.Errorf("局部变量备注编辑后应为 累乘结果!，实际 %q", p.Subs[0].Locals[1].Note)
	}
	// 退格
	e.handleEditKey(&event.KeyEvent{Key: "Backspace"})
	if p.Subs[0].Locals[1].Note != "累乘结果" {
		t.Errorf("退格后应为 累乘结果，实际 %q", p.Subs[0].Locals[1].Note)
	}
}

// TestStructEditorBodyEditor 验证「程序」区内嵌 CodeEditor：实例复用 + 编辑写回 Body。
func TestStructEditorBodyEditor(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	cel := e.ensureBodyEd(0, 400)
	if cel == nil {
		t.Fatal("内嵌编辑器创建失败")
	}
	if e.ensureBodyEd(0, 400) != cel {
		t.Error("ensureBodyEd 应复用同一实例（增删变量行不重建，编辑状态不丢）")
	}
	// 模拟在内嵌编辑器里编辑→OnChange 写回 sub.Body
	cel.lines = append(cel.lines, "输出调试文本 (积)")
	if cel.ed.OnChange != nil {
		cel.ed.OnChange(cel.text())
	}
	if p.Subs[0].Body != cel.text() {
		t.Errorf("内嵌编辑器编辑应写回 Body：Body=%q text=%q", p.Subs[0].Body, cel.text())
	}
}

// TestStructEditorRowOps 验证增删行 + 非编辑态方向键移格（批③）。
func TestStructEditorRowOps(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)

	// 增行：阶乘局部变量表末尾插入（原 2 个：i、积）
	n0 := len(p.Subs[0].Locals)
	e.insertRow("locals:0", n0)
	if len(p.Subs[0].Locals) != n0+1 {
		t.Fatalf("增行后应 %d 个局部变量，实际 %d", n0+1, len(p.Subs[0].Locals))
	}
	if !e.editing || e.selSection != "locals:0" || e.selRow != n0 || e.selCol != 0 {
		t.Errorf("增行后应进入新行首格编辑，实际 sec=%q row=%d col=%d editing=%v", e.selSection, e.selRow, e.selCol, e.editing)
	}

	// 删行：删掉刚加的空行，回到 2 个
	e.deleteRow("locals:0", n0)
	if len(p.Subs[0].Locals) != n0 {
		t.Errorf("删行后应回 %d 个，实际 %d", n0, len(p.Subs[0].Locals))
	}

	// 非编辑态方向键移格
	e.commitEdit()
	e.selSection, e.selRow, e.selCol = "globals", 0, 0
	e.moveSelect(1, 0) // 下移一行
	if e.selRow != 1 {
		t.Errorf("下移后应在第 1 行，实际 %d", e.selRow)
	}
	e.moveSelect(0, 1) // 右移一列
	if e.selCol != 1 {
		t.Errorf("右移后应在第 1 列，实际 %d", e.selCol)
	}
	if e.editing {
		t.Error("方向键移格不应进入编辑态")
	}
	// 列回绕：globals 共 4 列(0..3)，在末列右移应回到下一行第 0 列
	e.selRow, e.selCol = 0, 3
	e.moveSelect(0, 1)
	if e.selRow != 1 || e.selCol != 0 {
		t.Errorf("末列右移应回绕到 (1,0)，实际 (%d,%d)", e.selRow, e.selCol)
	}
}

// TestStructEditorEnterAddsRow 验证末行回车增行（取代「+ 增行」按钮）。
func TestStructEditorEnterAddsRow(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	n := len(p.Subs[0].Locals)
	e.beginEdit("locals:0", n-1, 0) // 最后一行
	e.handleEditKey(&event.KeyEvent{Key: "Enter"})
	if len(p.Subs[0].Locals) != n+1 {
		t.Errorf("末行回车应增行：%d→%d", n, len(p.Subs[0].Locals))
	}
	if !e.editing || e.selRow != n {
		t.Errorf("增行后应在新行编辑：row=%d editing=%v", e.selRow, e.editing)
	}
	// 非末行回车应只下移、不增行
	m := len(p.Subs[0].Locals)
	e.beginEdit("locals:0", 0, 0)
	e.handleEditKey(&event.KeyEvent{Key: "Enter"})
	if len(p.Subs[0].Locals) != m || e.selRow != 1 {
		t.Errorf("非末行回车应下移不增行：rows=%d row=%d", len(p.Subs[0].Locals), e.selRow)
	}
}

// TestStructEditorAutoDeclare 验证代码里赋值未声明变量→自动声明进局部变量表 + 类型推导。
func TestStructEditorAutoDeclare(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	cel := e.ensureBodyEd(0, 400)
	cel.lines = append(cel.lines,
		"计数 ＝ 10",       // 整数型
		"名字 ＝ \"小明\"", // 文本型
		"开关 ＝ 真",       // 逻辑型
		"比率 ＝ 1.5",      // 小数型
		"结果2 ＝ 阶乘 (3)", // 函数调用→阶乘返回整数型
	)
	cel.ed.OnChange(cel.text()) // 写回 Body
	cel.ed.OnEnter()            // 回车触发自动声明
	got := map[string]string{}
	for _, v := range p.Subs[0].Locals {
		got[v.Name] = v.Type
	}
	for name, want := range map[string]string{"计数": "整数型", "名字": "文本型", "开关": "逻辑型", "比率": "小数型", "结果2": "整数型"} {
		if got[name] != want {
			t.Errorf("%s 应推导为 %s，实际 %q", name, want, got[name])
		}
	}
	// 关键字/命令调用/已声明 不应被声明
	n := len(p.Subs[0].Locals)
	cel.lines = append(cel.lines, "返回 (计数)", "输出调试文本 (积)", "积 ＝ 1")
	cel.ed.OnChange(cel.text())
	cel.ed.OnEnter()
	if len(p.Subs[0].Locals) != n {
		t.Errorf("关键字/命令/已声明不应再增：%d→%d", n, len(p.Subs[0].Locals))
	}
	// sub 1（_启动）自动声明也要生效
	cel1 := e.ensureBodyEd(1, 400)
	b1 := len(p.Subs[1].Locals)
	cel1.lines = append(cel1.lines, "数量 ＝ 5")
	cel1.ed.OnChange(cel1.text())
	cel1.ed.OnEnter()
	if len(p.Subs[1].Locals) != b1+1 || p.Subs[1].Locals[b1].Name != "数量" {
		t.Errorf("_启动 自动声明「数量」失败：%+v", p.Subs[1].Locals)
	}
}

// TestStructEditorFuncCellEdit 验证函数声明行可编辑：函数名/返回类型写回。
func TestStructEditorFuncCellEdit(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	e.beginEdit("func:0", 0, 0) // 阶乘 的函数名格
	if e.cellValue("func:0", 0, 0) != "阶乘" {
		t.Fatalf("函数名应读到 阶乘，实际 %q", e.cellValue("func:0", 0, 0))
	}
	e.editInsert('2')
	if p.Subs[0].Name != "阶乘2" {
		t.Errorf("函数名编辑后应为 阶乘2，实际 %q", p.Subs[0].Name)
	}
	e.beginEdit("func:0", 0, 1) // 函数注释格（返回值已移到 returns 子表）
	if e.cellValue("func:0", 0, 1) != "递归求 n 的阶乘" {
		t.Fatalf("函数注释应读到，实际 %q", e.cellValue("func:0", 0, 1))
	}
	e.editInsert('!')
	if p.Subs[0].Note != "递归求 n 的阶乘!" {
		t.Errorf("函数注释编辑后应写回，实际 %q", p.Subs[0].Note)
	}
	// 返回值在 returns 子表（多返回值一行一个）
	if e.cellValue("returns:0", 0, 1) != "整数型" {
		t.Errorf("返回值类型应读到 整数型，实际 %q", e.cellValue("returns:0", 0, 1))
	}
	e.beginEdit("returns:0", 0, 1)
	e.editInsert('X')
	if p.Subs[0].Returns[0].Type != "整数型X" {
		t.Errorf("返回值类型编辑应写回，实际 %q", p.Subs[0].Returns[0].Type)
	}
}

// TestStructEditorAddSub 验证 Ctrl+M 新建子程序并进入函数名编辑。
func TestStructEditorAddSub(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	n := len(p.Subs)
	e.HandleEvent(event.NewKeyEvent(event.TypeKeyDown, 0, "M", event.ModCtrl))
	if len(p.Subs) != n+1 {
		t.Fatalf("Ctrl+M 应新建子程序：%d→%d", n, len(p.Subs))
	}
	if !e.editing || e.selSection != "func:"+itoaCE(n) || e.selCol != 0 {
		t.Errorf("新建后应进入新函数名编辑：sec=%q col=%d editing=%v", e.selSection, e.selCol, e.editing)
	}
	e.editInsert('X') // 直接命名新函数
	if p.Subs[n].Name != "新建子程序X" {
		t.Errorf("新函数名应可编辑，实际 %q", p.Subs[n].Name)
	}
}

// TestStructEditorParamRef 验证参数「参考」列点击切换 + 备注移到 col4。
func TestStructEditorParamRef(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	if p.Subs[0].Params[0].Ref != "" {
		t.Fatal("初始参考应空")
	}
	e.beginEdit("params:0", 0, 3) // 点参考列→切换为「是」
	if p.Subs[0].Params[0].Ref != "是" || e.editing {
		t.Errorf("点参考列应切「是」且不进编辑，Ref=%q editing=%v", p.Subs[0].Params[0].Ref, e.editing)
	}
	e.beginEdit("params:0", 0, 3) // 再点→取消
	if p.Subs[0].Params[0].Ref != "" {
		t.Errorf("再点应取消参考，实际 %q", p.Subs[0].Params[0].Ref)
	}
	e.beginEdit("params:0", 0, 4) // 备注现在是 col4
	e.editInsert('X')
	if p.Subs[0].Params[0].Note != "求 n 的阶乘X" {
		t.Errorf("参数备注(col4)编辑失败：%q", p.Subs[0].Params[0].Note)
	}
}

// TestStructEditorCompletion 验证代码补全注入已声明变量/命令。
func TestStructEditorCompletion(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	cel := e.ensureBodyEd(0, 400)
	has := func(items []compItem, label string) bool {
		for _, it := range items {
			if it.label == label {
				return true
			}
		}
		return false
	}
	if !has(cel.collectCompletions("结"), "结果") {
		t.Error("补全应含全局变量「结果」")
	}
	if !has(cel.collectCompletions("积"), "积") {
		t.Error("补全应含局部变量「积」")
	}
	if !has(cel.collectCompletions("输出"), "输出调试文本") {
		t.Error("补全应含命令「输出调试文本」")
	}
	if !has(cel.collectCompletions("阶"), "阶乘") {
		t.Error("补全应含子程序「阶乘」")
	}
}

// TestStructEditorSerialize 验证「文本↔表格」双向序列化无损 round-trip。
func TestStructEditorSerialize(t *testing.T) {
	p := DemoFactorialProgram()
	p.Subs[0].Params[0].Ref = "是"
	text := p.Serialize()
	p2 := ParseProgram(text)
	if p2.Serialize() != text {
		t.Errorf("round-trip 文本不一致：\n--- 原 ---\n%s\n--- 回 ---\n%s", text, p2.Serialize())
	}
	if len(p2.Globals) != 2 || p2.Globals[0].Name != "结果" || p2.Globals[0].Type != "整数型" {
		t.Errorf("全局变量解析不对：%+v", p2.Globals)
	}
	if len(p2.Subs) != 2 || p2.Subs[0].Name != "阶乘" || retString(p2.Subs[0].Returns) != "整数型" || p2.Subs[0].Note != "递归求 n 的阶乘" {
		t.Errorf("子程序头解析不对：%+v", p2.Subs[0])
	}
	if p2.Subs[0].Params[0].Ref != "是" {
		t.Errorf("参数参考未保留：%q", p2.Subs[0].Params[0].Ref)
	}
	if p2.Subs[0].Body != p.Subs[0].Body {
		t.Errorf("代码体不一致：\n原 %q\n回 %q", p.Subs[0].Body, p2.Subs[0].Body)
	}
	if len(p2.Subs[0].Locals) != 2 || p2.Subs[0].Locals[1].Name != "积" {
		t.Errorf("局部变量解析不对：%+v", p2.Subs[0].Locals)
	}
}

// TestPinyinInitials 验证汉字→拼音首字母（GBK 区位法）。
func TestPinyinInitials(t *testing.T) {
	cases := map[string]string{"结果": "jg", "输出": "sc", "调试开关": "dskg", "abc结果": "abcjg"}
	for in, want := range cases {
		if got := pinyinInitials(in); got != want {
			t.Errorf("pinyinInitials(%q)=%q，期望 %q", in, got, want)
		}
	}
}

// TestStructEditorPinyinCompletion 验证补全支持拼音首字母匹配。
func TestStructEditorPinyinCompletion(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	cel := e.ensureBodyEd(0, 400)
	has := func(items []compItem, label string) bool {
		for _, it := range items {
			if it.label == label {
				return true
			}
		}
		return false
	}
	if !has(cel.collectCompletions("jg"), "结果") {
		t.Error("拼音首字母 jg 应匹配「结果」")
	}
	if !has(cel.collectCompletions("sc"), "输出调试文本") {
		t.Error("拼音首字母 sc 应匹配「输出调试文本」")
	}
}

// TestEnclosingBracketSkipsStringComment 验证括号匹配忽略字符串/注释里的括号。
func TestEnclosingBracketSkipsStringComment(t *testing.T) {
	code := "func f() {\n\ts := \"a{b\" // 注释 }\n\tx := 1\n}"
	e := NewCodeEditor("go", code).CreateElement().(*CodeEditorElement)
	e.cursor = cePos{2, 0} // x := 1 行
	ol, _, cl, _, ok := e.enclosingBracket()
	if !ok || ol != 0 || cl != 3 {
		t.Errorf("应匹配 func 的 {}（行 0→3，忽略串/注释里的括号），实际 ol=%d cl=%d ok=%v", ol, cl, ok)
	}
}

// TestStructEditorGoRoundTrip 验证接 Go：源码→表格(ParseGo)→源码(ToGo) 解析正确 + idempotent。
func TestStructEditorGoRoundTrip(t *testing.T) {
	src := "package main\n\nvar 计数 int\n\n// 阶乘 求 n 的阶乘\nfunc 阶乘(n int) int {\n\t积 := 1\n\tfor i := 1; i <= n; i++ {\n\t\t积 *= i\n\t}\n\treturn 积\n}\n\nfunc 多值(a, b int) (int, error) {\n\treturn a + b, nil\n}\n"
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo 失败：%v", err)
	}
	if len(p.Globals) != 1 || p.Globals[0].Name != "计数" || p.Globals[0].Type != "int" {
		t.Errorf("包级变量解析不对：%+v", p.Globals)
	}
	if len(p.Subs) != 2 {
		t.Fatalf("函数数不对：%d", len(p.Subs))
	}
	if p.Subs[0].Name != "阶乘" || retString(p.Subs[0].Returns) != "int" || p.Subs[0].Note != "阶乘 求 n 的阶乘" {
		t.Errorf("阶乘头不对：%+v", p.Subs[0])
	}
	if len(p.Subs[0].Params) != 1 || p.Subs[0].Params[0].Name != "n" || p.Subs[0].Params[0].Type != "int" {
		t.Errorf("阶乘参数不对：%+v", p.Subs[0].Params)
	}
	if p.Subs[0].Body == "" {
		t.Error("函数体未保留")
	}
	if retString(p.Subs[1].Returns) != "(int, error)" { // 多返回值
		t.Errorf("多返回值不对：%q", retString(p.Subs[1].Returns))
	}
	if len(p.Subs[1].Returns) != 2 { // 多返回值应是 2 行
		t.Errorf("多返回值应 2 个，实际 %d", len(p.Subs[1].Returns))
	}
	if len(p.Subs[1].Params) != 2 || p.Subs[1].Params[1].Name != "b" {
		t.Errorf("多参数解析不对：%+v", p.Subs[1].Params)
	}
	// idempotent round-trip：ToGo 产物能重新解析且再导出稳定
	got := p.ToGo()
	p2, err := ParseGo(got)
	if err != nil {
		t.Fatalf("ToGo 产物无法重新解析：%v\n%s", err, got)
	}
	if got2 := p2.ToGo(); got != got2 {
		t.Errorf("round-trip 不稳定：\n--- got ---\n%s\n--- got2 ---\n%s", got, got2)
	}
}

// TestStructEditorGoLocals 验证 Go 模式：函数体里的局部变量(var/:=)解析回填局部变量表（代码→表格）。
func TestStructEditorGoLocals(t *testing.T) {
	p, err := ParseGo("package main\n\nfunc f(n int) int {\n\tx := 5\n\tvar s string\n\t名字 := \"hi\"\n\t比率 := 1.5\n\treturn n\n}\n")
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	loc := map[string]string{}
	for _, v := range p.Subs[0].Locals {
		loc[v.Name] = v.Type
	}
	if loc["x"] != "int" || loc["s"] != "string" || loc["名字"] != "string" || loc["比率"] != "float64" {
		t.Errorf("局部变量推导不对：%+v", p.Subs[0].Locals)
	}
	if _, ok := loc["n"]; ok {
		t.Error("参数 n 不应入局部变量")
	}
	// 代码区再加一个局部变量 → 回车回填
	e := NewStructEditor(p).WithLang("go").CreateElement().(*StructEditorElement)
	e.ensureBodyEd(0, 400)
	cel := e.bodyEds[0]
	cel.lines = append(cel.lines, "总数 := 100")
	cel.ed.OnChange(cel.text())
	e.goExtractLocals(0)
	found := false
	for _, v := range e.program.Subs[0].Locals {
		if v.Name == "总数" && v.Type == "int" {
			found = true
		}
	}
	if !found {
		t.Errorf("代码加局部变量后应回填表格：%+v", e.program.Subs[0].Locals)
	}
}

// TestStructEditorGoComments 验证 Go 注释的解析与 round-trip 保留。
func TestStructEditorGoComments(t *testing.T) {
	src := "package main\n\n// 计数 全局计数器\nvar 计数 int\n\n// 阶乘 求 n 的阶乘\nfunc 阶乘(n int) int {\n\tx := 5 // 行内注释\n\t// 独立注释\n\treturn x\n}\n"
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	if p.Globals[0].Note != "计数 全局计数器" {
		t.Errorf("全局变量 doc 注释应入备注：%q", p.Globals[0].Note)
	}
	if p.Subs[0].Note != "阶乘 求 n 的阶乘" {
		t.Errorf("函数 doc 注释应入注释列：%q", p.Subs[0].Note)
	}
	out := p.ToGo()
	for _, want := range []string{"计数 全局计数器", "阶乘 求 n 的阶乘", "行内注释", "独立注释"} {
		if !strings.Contains(out, want) {
			t.Errorf("ToGo 丢了注释 %q：\n%s", want, out)
		}
	}
}

// TestStructEditorGoParamComments 验证参数/返回值行尾注释→备注列 + ToGo round-trip 还原。
func TestStructEditorGoParamComments(t *testing.T) {
	src := "package main\n\nfunc 阶乘(\n\tn int, // 求 n 的阶乘\n) (\n\t结果 int, // 阶乘结果\n) {\n\treturn 1\n}\n"
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	if len(p.Subs[0].Params) != 1 || p.Subs[0].Params[0].Note != "求 n 的阶乘" {
		t.Errorf("参数行尾注释应入备注列：%+v", p.Subs[0].Params)
	}
	if len(p.Subs[0].Returns) != 1 || p.Subs[0].Returns[0].Note != "阶乘结果" {
		t.Errorf("返回值行尾注释应入备注列：%+v", p.Subs[0].Returns)
	}
	out := p.ToGo()
	if !strings.Contains(out, "求 n 的阶乘") || !strings.Contains(out, "阶乘结果") {
		t.Errorf("ToGo 应把备注还原成行尾注释：\n%s", out)
	}
	p2, err := ParseGo(out) // idempotent round-trip
	if err != nil {
		t.Fatalf("re-parse: %v\n%s", err, out)
	}
	if p2.Subs[0].Params[0].Note != "求 n 的阶乘" || p2.Subs[0].Returns[0].Note != "阶乘结果" {
		t.Errorf("round-trip 注释不稳定：参数%+v 返回值%+v", p2.Subs[0].Params, p2.Subs[0].Returns)
	}
}

// TestStructEditorGoLocalComments 验证 Go 局部变量(var/:=)的行尾注释→局部变量备注列。
func TestStructEditorGoLocalComments(t *testing.T) {
	p, err := ParseGo("package main\n\nfunc f() {\n\tx := 5 // 计数器\n\tvar s string // 名字\n\ty := 0\n}\n")
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	loc := map[string]string{}
	for _, v := range p.Subs[0].Locals {
		loc[v.Name] = v.Note
	}
	if loc["x"] != "计数器" {
		t.Errorf("x 备注应「计数器」，实际 %q", loc["x"])
	}
	if loc["s"] != "名字" {
		t.Errorf("s 备注应「名字」，实际 %q", loc["s"])
	}
	if loc["y"] != "" {
		t.Errorf("y 无注释备注应空，实际 %q", loc["y"])
	}
}

// TestCodeEditorCursorRef 验证 CursorRef 跨重建恢复光标/滚动（切换视图保持位置）。
func TestCodeEditorCursorRef(t *testing.T) {
	ref := &CECursorState{Line: 3, Col: 1, ScrollX: 10, ScrollY: 50}
	ce := NewCodeEditor("go", "a\nb\nc\ndd\ne")
	ce.CursorRef = ref
	e := ce.CreateElement().(*CodeEditorElement)
	if e.cursor.line != 3 || e.cursor.col != 1 {
		t.Errorf("应从 CursorRef 恢复光标 (3,1)，实际 (%d,%d)", e.cursor.line, e.cursor.col)
	}
	if e.scrollY != 50 || e.scrollX != 10 {
		t.Errorf("应恢复滚动 (10,50)，实际 (%v,%v)", e.scrollX, e.scrollY)
	}
}

// TestStructEditorScrollRef 验证 ScrollRef 跨重建恢复滚动位置。
func TestStructEditorScrollRef(t *testing.T) {
	sy := 120.0
	se := NewStructEditor(DemoFactorialProgram())
	se.ScrollRef = &sy
	e := se.CreateElement().(*StructEditorElement)
	if e.scrollY != 120 {
		t.Errorf("应从 ScrollRef 恢复 scrollY 120，实际 %v", e.scrollY)
	}
}

// TestStructEditorGoBlankLines 验证 ToGo 不压缩连续空行（gofmt 会压，我们保留用户的空行）。
func TestStructEditorGoBlankLines(t *testing.T) {
	src := "package main\n\nfunc f() {\n\tx := 1\n\n\n\ty := 2\n}\n" // x、y 间 2 个空行
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	out := p.ToGo()
	if !strings.Contains(out, "\n\n\n") { // 2 个连续空行 → 至少 \n\n\n
		t.Errorf("连续空行被压缩了（应保留）：\n%q", out)
	}
}

// TestStructEditorGoInferTypes 验证 := 声明的类型推导（函数调用/复合字面量/make/len/比较/算术）。
func TestStructEditorGoInferTypes(t *testing.T) {
	src := "package main\n\nfunc 阶乘(n int) int { return n }\n\nfunc g() {\n\ta := 阶乘(5)\n\tb := []int{1, 2}\n\tc := make(map[string]int)\n\td := a > 3\n\te := a + 1\n\tf := \"hi\"\n\tn := len(b)\n}\n"
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	loc := map[string]string{} // g 是 Subs[1]
	for _, v := range p.Subs[1].Locals {
		loc[v.Name] = v.Type
	}
	for name, want := range map[string]string{
		"a": "int",            // 阶乘(5)→阶乘返回 int
		"b": "[]int",          // 复合字面量
		"c": "map[string]int", // make
		"d": "bool",           // a > 3 比较
		"e": "int",            // a + 1 算术
		"f": "string",
		"n": "int", // len
	} {
		if loc[name] != want {
			t.Errorf("%s := … 应推导 %q，实际 %q", name, want, loc[name])
		}
	}
}

// TestStructEditorGoBlankLinesEdges 验证函数体首/尾的空行也保留（bodyText 不再 Trim 掉）。
func TestStructEditorGoBlankLinesEdges(t *testing.T) {
	src := "package main\n\nfunc f() {\n\n\tx := 1\n\n}\n" // 体首、体尾各 1 空行
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	out := p.ToGo()
	if !strings.Contains(out, "{\n\n") {
		t.Errorf("函数体首空行丢失：\n%s", out)
	}
	if !strings.Contains(out, "\n\n}") {
		t.Errorf("函数体尾空行丢失：\n%s", out)
	}
}

// TestStructEditorGoMultiReturnInfer 验证 x, y := f() 多返回值按被调函数返回类型逐一推导。
func TestStructEditorGoMultiReturnInfer(t *testing.T) {
	src := "package main\n\nfunc divmod(a, b int) (int, int) {\n\treturn a / b, a % b\n}\n\nfunc use() {\n\tq, r := divmod(10, 3)\n\t_ = q\n\t_ = r\n}\n"
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	var use *SESub
	for i := range p.Subs {
		if p.Subs[i].Name == "use" {
			use = &p.Subs[i]
		}
	}
	if use == nil {
		t.Fatal("没找到 use 子程序")
	}
	got := map[string]string{}
	for _, l := range use.Locals {
		got[l.Name] = l.Type
	}
	if got["q"] != "int" || got["r"] != "int" {
		t.Errorf("多返回值推导失败：q=%q r=%q（期望都为 int），Locals=%+v", got["q"], got["r"], use.Locals)
	}
}

// TestStructEditorGoFuncBlankLines 验证函数之间的多空行（用户分段）经 ParseGo→ToGo 保留。
func TestStructEditorGoFuncBlankLines(t *testing.T) {
	src := "package main\n\nfunc a() {\n\tprintln(1)\n}\n\n\nfunc b() {\n\tprintln(2)\n}\n" // a 与 b 间 2 空行
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	out := p.ToGo()
	if !strings.Contains(out, "}\n\n\nfunc b") {
		t.Errorf("函数间 2 空行未保留：\n%s", out)
	}
}

// TestSEProgramGoDocForBody 验证虚拟文档生成 + 函数体首行号映射（内嵌编辑器接 gopls 的地基）。
func TestSEProgramGoDocForBody(t *testing.T) {
	p := &SEProgram{
		Globals: []SEVar{{Name: "g", Type: "int"}},
		Subs: []SESub{
			{Name: "a", Body: "x := 1"},
			{Name: "b", Params: []SEVar{{Name: "n", Type: "int"}}, Body: "z := 9"},
		},
	}
	// 覆盖第 1 个子程序(b)的函数体为多行，校验 bodyLine 指向体首行
	doc, bodyLine := p.GoDocForBody(1, "y := 2\nprintln(y)")
	lines := strings.Split(doc, "\n")
	if bodyLine < 0 || bodyLine >= len(lines) {
		t.Fatalf("bodyLine=%d 越界，doc:\n%s", bodyLine, doc)
	}
	if strings.TrimSpace(lines[bodyLine]) != "y := 2" {
		t.Errorf("bodyLine=%d 行内容=%q（期望 y := 2）\ndoc:\n%s", bodyLine, lines[bodyLine], doc)
	}
	// 体行在完整文档里应有一个 \t 缩进（colOff=1 的依据）
	if !strings.HasPrefix(lines[bodyLine], "\t") {
		t.Errorf("体行未带 \\t 缩进：%q", lines[bodyLine])
	}
	// 必须是合法 Go（能被 ParseGo 解析，证明虚拟文档可喂 gopls）
	if _, err := ParseGo(doc); err != nil {
		t.Errorf("虚拟文档非法 Go：%v\n%s", err, doc)
	}
	// 第二行映射：body 第 1 行(println(y)) 应在 bodyLine+1
	if strings.TrimSpace(lines[bodyLine+1]) != "println(y)" {
		t.Errorf("体第二行映射错：%q", lines[bodyLine+1])
	}
}

// TestMarkdownHighlight 验证 Markdown 结构化分词：标题/围栏代码块/粗体。
func TestMarkdownHighlight(t *testing.T) {
	toks, _ := tokenizeMarkdown([]rune("# Hello"), stNormal)
	if len(toks) != 1 || toks[0].kind != tkKeyword {
		t.Errorf("标题应整行 keyword：%+v", toks)
	}
	if _, st := tokenizeMarkdown([]rune("```go"), stNormal); st != stFence {
		t.Errorf("``` 应进入 stFence，得 %d", st)
	}
	in, st2 := tokenizeMarkdown([]rune("x := 1"), stFence)
	if st2 != stFence || len(in) != 1 || in[0].kind != tkString {
		t.Errorf("围栏内应整行 string 且保持 stFence：%+v st=%d", in, st2)
	}
	if _, st3 := tokenizeMarkdown([]rune("```"), stFence); st3 != stNormal {
		t.Errorf("闭合 ``` 应回 stNormal，得 %d", st3)
	}
	bt, _ := tokenizeMarkdown([]rune("a **b** c"), stNormal)
	found := false
	for _, tk := range bt {
		if tk.kind == tkType && tk.start == 2 && tk.end == 7 {
			found = true
		}
	}
	if !found {
		t.Errorf("**b** 应为 type 色 [2,7)：%+v", bt)
	}
}

// TestHTMLHighlight 验证 HTML 结构化分词：标签名/属性名/属性值 + 跨行注释。
func TestHTMLHighlight(t *testing.T) {
	toks, _ := tokenizeHTML([]rune(`<div class="x">`), stNormal)
	var hasTag, hasAttr, hasVal bool
	for _, tk := range toks {
		switch tk.kind {
		case tkKeyword:
			hasTag = true
		case tkType:
			hasAttr = true
		case tkString:
			hasVal = true
		}
	}
	if !hasTag || !hasAttr || !hasVal {
		t.Errorf("HTML 标签/属性/值高亮缺失：tag=%v attr=%v val=%v\n%+v", hasTag, hasAttr, hasVal, toks)
	}
	if _, st := tokenizeHTML([]rune("<!-- open"), stNormal); st != stBlockComment {
		t.Errorf("未闭合 <!-- 应进入 stBlockComment，得 %d", st)
	}
}

// TestStructEditorGoImports 验证 import 声明双向：解析进 Imports、ToGo 还原（单/多/别名）。
func TestStructEditorGoImports(t *testing.T) {
	src := "package main\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n)\n\nfunc f() {\n\tfmt.Println(strings.ToUpper(\"hi\"))\n}\n"
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	if len(p.Imports) != 2 {
		t.Fatalf("imports=%v 期望 2", p.Imports)
	}
	out := p.ToGo()
	if !strings.Contains(out, `"fmt"`) || !strings.Contains(out, `"strings"`) {
		t.Errorf("import 丢失:\n%s", out)
	}
	if !strings.Contains(out, "import (") {
		t.Errorf("多 import 应用括号组:\n%s", out)
	}
	// 单 import + 别名
	p2, _ := ParseGo("package main\n\nimport m \"math\"\n\nfunc g() float64 { return m.Pi }\n")
	if len(p2.Imports) != 1 || p2.Imports[0] != `m "math"` {
		t.Errorf("别名 import 解析错: %v", p2.Imports)
	}
	if out2 := p2.ToGo(); !strings.Contains(out2, `import m "math"`) {
		t.Errorf("单 import 别名生成错:\n%s", out2)
	}
	// round-trip 仍是合法 Go + 虚拟文档含 import（gopls 可解析 stdlib 成员）
	if _, err := ParseGo(out); err != nil {
		t.Errorf("round-trip 非法 Go: %v\n%s", err, out)
	}
	if doc, _ := p.GoDocForBody(0, "fmt.Print(1)"); !strings.Contains(doc, `"fmt"`) {
		t.Errorf("GoDocForBody 应含 import 供 gopls:\n%s", doc)
	}
}

// TestStructEditorImportEmptyAdd 验证无 import 时也有占位行，点击编辑即加第一个 import。
func TestStructEditorImportEmptyAdd(t *testing.T) {
	se := NewStructEditor(&SEProgram{Subs: []SESub{{Name: "f"}}}).WithLang("go")
	e := se.CreateElement().(*StructEditorElement)
	if n := e.sectionRows("imports"); n != 1 {
		t.Errorf("空导入应有 1 个占位行，得 %d", n)
	}
	e.setCellValue("imports", 0, 0, `"fmt"`) // 编辑占位行 → 加第一个
	if len(e.program.Imports) != 1 || e.program.Imports[0] != `"fmt"` {
		t.Fatalf("空表加 import 失败: %v", e.program.Imports)
	}
	if !strings.Contains(e.program.ToGo(), `import "fmt"`) {
		t.Errorf("ToGo 应含 import \"fmt\":\n%s", e.program.ToGo())
	}
}

// TestStructEditorGoRefPointer 验证 Go 指针参数 *T ↔ 表格「参考(传址)」双向映射。
func TestStructEditorGoRefPointer(t *testing.T) {
	src := "package main\n\nfunc f(n *int, s string) {\n\t*n = 1\n}\n"
	p, err := ParseGo(src)
	if err != nil {
		t.Fatalf("ParseGo: %v", err)
	}
	sub := p.Subs[0]
	if len(sub.Params) != 2 {
		t.Fatalf("参数数=%d 期望2", len(sub.Params))
	}
	if sub.Params[0].Type != "int" || sub.Params[0].Ref != "是" {
		t.Errorf("指针参数 n 应 type=int ref=是，得 type=%q ref=%q", sub.Params[0].Type, sub.Params[0].Ref)
	}
	if sub.Params[1].Ref != "" {
		t.Errorf("非指针参数 s 不应有参考，得 ref=%q", sub.Params[1].Ref)
	}
	if out := p.ToGo(); !strings.Contains(out, "n *int") {
		t.Errorf("ToGo 应还原指针参数 n *int：\n%s", out)
	}
}

type testMiniProvider struct{}

func (testMiniProvider) Name() string { return "mini" }
func (testMiniProvider) Schema() *SESchema {
	cols := []SECol{{Title: "名称", Field: SEFieldName, Weight: 0.5}, {Title: "类型", Field: SEFieldType, Weight: 0.5}}
	return &SESchema{Globals: cols, Params: cols, Returns: cols, Locals: cols}
}
func (testMiniProvider) Parse(src string) (*SEProgram, error) { return &SEProgram{}, nil }
func (testMiniProvider) Generate(p *SEProgram) string         { return "" }

// TestStructEditorSchemaProvider 验证「接新语言只需注册 provider + 给 schema」的扩展点。
func TestStructEditorSchemaProvider(t *testing.T) {
	RegisterProvider(testMiniProvider{}, "mini")
	if providerFor("mini") == nil {
		t.Fatal("provider 未注册成功")
	}
	if n := len(providerFor("mini").Schema().Globals); n != 2 {
		t.Errorf("mini schema Globals 列数=%d 期望2", n)
	}
	se := NewStructEditor(&SEProgram{}).WithLang("mini")
	if n := len(se.effectiveSchema().Globals); n != 2 {
		t.Errorf("WithLang(mini) 应取 mini 的 2 列 schema，得 %d 列", n)
	}
	// 内置 go provider 仍在，schema 为默认 5 列参数表
	if n := len(providerFor("go").Schema().Params); n != 5 {
		t.Errorf("go provider 参数表应 5 列，得 %d", n)
	}
}

// TestStructEditorFold 验证子程序折叠/展开开关。
func TestStructEditorFold(t *testing.T) {
	p := DemoFactorialProgram()
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	if e.collapsed[0] {
		t.Fatal("初始不应折叠")
	}
	e.ToggleFold(0)
	if !e.collapsed[0] {
		t.Error("折叠后 collapsed[0] 应为 true")
	}
	e.ToggleFold(0)
	if e.collapsed[0] {
		t.Error("再次切换应展开")
	}
}

// TestCodeEditorBackspaceMerge 验证退格在行首合并到上一行。
func TestCodeEditorBackspaceMerge(t *testing.T) {
	ce := NewCodeEditor("go", "ab\ncd")
	e := ce.CreateElement().(*CodeEditorElement)
	e.cursor = cePos{1, 0}
	e.anchor = e.cursor
	e.backspace()
	if e.text() != "abcd" {
		t.Errorf("行首退格应合并为 abcd，实际 %q", e.text())
	}
	if e.cursor.line != 0 || e.cursor.col != 2 {
		t.Errorf("光标应在 (0,2)，实际 %+v", e.cursor)
	}
}
