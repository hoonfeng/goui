package widget

import (
	"strings"
	"time"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// ─── StructEditor 结构化编辑器（对标易语言 IDE：变量表格化 + 逻辑代码化）─────
// 变量是结构化表格(名称|类型|数组|备注)；逻辑是自由代码——每个子程序的「程序」区
// 内嵌一个完整 CodeEditor(语法高亮/补全/多光标)。逻辑写代码顺手、长语句不再挤单元格。
// 关键：变量增删行只 repaint、不 relayout，故内嵌编辑器实例长存、编辑状态(光标/输入)不丢。

// SEVar 一个变量（变量表的一行）。
type SEVar struct {
	Name  string // 名称
	Type  string // 类型
	Array string // 数组维度（如 "0" / "3,4"）
	Ref   string // 参考（"是"/空）：参数传址（仅参数表用，变量表忽略）
	Note  string // 备注
}

// SETypeKind 类型定义种类。
type SETypeKind string

const (
	SETypeStruct   SETypeKind = "struct"   // struct 类型
	SETypeInterface SETypeKind = "interface" // interface 类型
	SETypeAlias    SETypeKind = "alias"    // 类型别名/定义（type X = Y / type X Y）
)

// SEType 一个类型定义（struct / interface / alias）。
type SEType struct {
	Name      string      // 类型名
	Kind      SETypeKind  // 类型种类
	Note      string      // 注释
	Fields    []SEVar     // struct 字段列表（Name=字段名, Type=类型, Note=注释/tag）
	Methods   []SEVar     // interface 方法列表（Name=方法名, Type=方法签名, Note=注释）
	TypeExpr  string      // alias/typedef 的底层类型表达式（如 "string" / "func(int)bool"）
	TypeParams []SEVar    // 泛型类型参数（Name=参数名, Type=约束）
	BlankBefore int       // 前方空行数
}

// SESub 一个子程序（函数）。逻辑用自由代码文本（喂给内嵌 CodeEditor），变量用表格。
type SESub struct {
	Name        string  // 名称
	Recv        string  // 方法接收器，如 "(r *Receiver)"（空=普通函数）
	Returns     []SEVar // 返回值列表（多返回值一行一个；元素 Name 可空、Type 为类型）
	Note        string  // 函数注释（说明本子程序作用）
	Params      []SEVar // 参数表
	Locals      []SEVar // 局部变量表
	Body        string  // 逻辑代码（自由文本，内嵌 CodeEditor 编辑）
	BlankBefore int     // 本函数前的空行数（ParseGo 捕获，ToGo 还原，保留用户分段；<1 时按 1）
	TypeParams  []SEVar // 泛型类型参数（Name=参数名, Type=约束）
}

// retString 把返回值列表拼成文本（0→""；单个无名→"int"；多个/具名→"(a int, b error)"）。
func retString(returns []SEVar) string {
	if len(returns) == 0 {
		return ""
	}
	named := false
	parts := make([]string, 0, len(returns))
	for _, r := range returns {
		if r.Name != "" {
			named = true
			parts = append(parts, r.Name+" "+r.Type)
		} else {
			parts = append(parts, r.Type)
		}
	}
	if len(parts) == 1 && !named {
		return parts[0]
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// SEProgram 整个程序。
type SEProgram struct {
	Imports []string  // 导入（每项为 import spec 内容，如 `"fmt"` / `m "math"` / `_ "embed"`）
	Globals []SEVar   // 程序集变量表
	Consts  []SEVar   // 常量声明（Name=常量名, Type=类型, Note=注释/值）
	Types   []SEType  // 类型定义（struct / interface / alias/typedef）
	Subs    []SESub   // 子程序（函数/方法）列表
}

type StructEditor struct {
	StatelessWidget
	Width     float64
	Height    float64
	program   *SEProgram
	lang      string    // 内嵌代码编辑器语言："ey"(易语言,默认) / "go" / 其他
	ScrollRef *float64  // 跨重建保持竖向滚动位置（CodeWorkbench 切换视图用）
	schema      *SESchema // 表结构配置（列定义）；nil→按 lang 的 provider 取，再兜底 DefaultSchema()
	ShowMinimap bool      // 右侧缩略图（默认开；窗口太窄自动不显示）

	// LSP（仅 go 模式生效）：内嵌函数体编辑器经 lspWrap 把片段映射到完整 Go 文档喂 gopls。
	// 聚焦某子程序代码区时才启动该编辑器的 gopls（避免 N 子程序 = N 进程）。
	LSPServer, LSPWorkspace, LSPFile string
}

func NewStructEditor(p *SEProgram) *StructEditor {
	if p == nil {
		p = &SEProgram{}
	}
	return &StructEditor{Width: 720, Height: 460, program: p, lang: "ey", ShowMinimap: true}
}
func (s *StructEditor) WithSize(w, h float64) *StructEditor { s.Width, s.Height = w, h; return s }

// WithMinimap 开/关右侧缩略图。
func (s *StructEditor) WithMinimap(on bool) *StructEditor { s.ShowMinimap = on; return s }

// WithLang 设置内嵌代码编辑器语言（"go" 则逻辑区按 Go 高亮/补全）。
func (s *StructEditor) WithLang(l string) *StructEditor { s.lang = l; return s }

// WithSchema 自定义表结构（列定义）。不设则按 lang 的 LanguageProvider 取，再兜底 DefaultSchema()。
func (s *StructEditor) WithSchema(sc *SESchema) *StructEditor { s.schema = sc; return s }

// effectiveSchema 取生效的表结构：显式 schema > 语言 provider 的 schema > DefaultSchema()。
func (s *StructEditor) effectiveSchema() *SESchema {
	if s.schema != nil {
		return s.schema
	}
	if p := providerFor(s.lang); p != nil {
		if sc := p.Schema(); sc != nil {
			return sc
		}
	}
	return DefaultSchema()
}

// WithLSP 给 go 模式的内嵌函数体编辑器接语言服务器（gopls）。fileURI 指向真实模块内的暂存 .go 文件，
// 内嵌编辑器会把整张表的 ToGo 文档以该 URI 喂给 gopls（in-memory overlay），按函数体行号偏移映射坐标。
func (s *StructEditor) WithLSP(server, workspaceURI, fileURI string) *StructEditor {
	s.LSPServer, s.LSPWorkspace, s.LSPFile = server, workspaceURI, fileURI
	return s
}

func (s *StructEditor) CreateElement() Element {
	e := &StructEditorElement{BaseElement: BaseElement{widget: s}, se: s, program: s.program,
		schema: s.effectiveSchema(), showMinimap: s.ShowMinimap, selSection: "", selRow: -1, selCol: -1}
	if s.ScrollRef != nil { // 跨重建恢复滚动位置（切换视图保持）
		e.scrollY = *s.ScrollRef
	}
	return e
}

const (
	seRowH    = 26.0
	seHeaderH = 24.0
	sePad     = 10.0
)

// seCellHit 记录一个单元格的命中矩形 + 定位（区段/行/列）。
type seCellHit struct {
	rect     types.Rect
	section  string
	row, col int
}

// seFoldHit 子程序折叠三角的命中区。
type seFoldHit struct {
	rect types.Rect
	sub  int
}

type StructEditorElement struct {
	BaseElement
	se      *StructEditor
	program *SEProgram
	schema  *SESchema // 表结构配置（列定义），渲染/编辑都按它走，不再硬编码列

	scrollY    float64
	contentH   float64
	cells      []seCellHit  // Paint 缓存：所有可编辑单元格的命中区
	foldHits   []seFoldHit  // Paint 缓存：子程序折叠三角命中区
	showMinimap bool        // 右侧缩略图开关（从 se.ShowMinimap）
	miniRect   types.Rect   // Paint 缓存：缩略图区域（点击/拖动跳转命中）
	miniSegs   []seMiniSeg  // Paint 缓存：缩略图内容段（程序集/各子程序）
	font       canvas.Font  // Paint 缓存：当前等宽字体（gutter/单元格文本共用）
	draggingMini bool       // 正在拖动缩略图视口框
	collapsed  map[int]bool // 折叠（收起）的子程序
	globalsCollapsed bool   // 程序集变量表是否收起（折叠后只剩表头，行号仍累加）
	selSection string
	selRow     int
	selCol     int
	focused    bool

	editing    bool          // 是否在编辑选中单元格
	editCol    int           // 编辑光标（rune 列）
	focusTime  time.Time     // 光标闪烁
	lastCanvas canvas.Canvas // Paint 缓存：Skia 测量（中文不漂移）

	bodyEds                map[int]*CodeEditorElement // 各子程序「程序」区内嵌的代码编辑器（lazy 创建，长存）
	activeCE               *CodeEditorElement         // 当前焦点所在的内嵌编辑器（nil=焦点在变量表/空白）
	cellCaretX, cellCaretY float64                    // 变量表单元格编辑光标的客户区坐标（IME 候选定位）
}

const seBodyEdH = 180.0 // 内嵌逻辑代码编辑器高度

// eyCommands 易语言常用命令库（用于代码补全；演示集，后续接真实支持库扩展）。
var eyCommands = []string{
	"输出调试文本", "信息框", "取文本长度", "到文本", "到整数", "到小数",
	"取数组成员数", "加入成员", "删除成员", "清除数组",
	"如果", "如果真", "否则", "判断", "返回",
	"计次循环首", "计次循环尾", "判断循环首", "判断循环尾",
	"变量循环首", "变量循环尾", "跳出循环", "到循环尾",
}

func (e *StructEditorElement) Build() []Element { return nil }

func (e *StructEditorElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w, h := e.se.Width, e.se.Height
	if w <= 0 {
		w = 720
	}
	if h <= 0 {
		h = 460
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *StructEditorElement) Focus()          { e.focused = true; repaint() }
func (e *StructEditorElement) Blur()           { e.focused = false; repaint() }
func (e *StructEditorElement) IsFocused() bool { return e.focused }

// DemoFactorialProgram 一个示例程序（计算阶乘，体现易语言风格）。
func DemoFactorialProgram() *SEProgram {
	return &SEProgram{
		Globals: []SEVar{
			{Name: "结果", Type: "整数型", Array: "", Note: "存放计算结果"},
			{Name: "调试开关", Type: "逻辑型", Array: "", Note: ""},
		},
		Subs: []SESub{
			{
				Name: "阶乘", Returns: []SEVar{{Type: "整数型"}}, Note: "递归求 n 的阶乘",
				Params: []SEVar{{Name: "n", Type: "整数型", Array: "", Note: "求 n 的阶乘"}},
				Locals: []SEVar{
					{Name: "i", Type: "整数型", Array: "", Note: "循环变量"},
					{Name: "积", Type: "整数型", Array: "", Note: "累乘结果"},
				},
				Body: "积 ＝ 1\n计次循环首 (i ＝ 1 到 n)\n    积 ＝ 积 × i\n计次循环尾 ()\n返回 (积)",
			},
			{
				Name: "_启动子程序", Note: "程序入口",
				Body: "结果 ＝ 阶乘 (5)\n输出调试文本 (结果)",
			},
		},
	}
}

// ensureBodyEd 取/创建第 si 个子程序「程序」区的内嵌代码编辑器。
// 编辑器是长存实例：变量增删行只 repaint 不 relayout，故不会被销毁、编辑状态(光标/输入/滚动)保留。
func (e *StructEditorElement) ensureBodyEd(si int, w float64) *CodeEditorElement {
	if e.bodyEds == nil {
		e.bodyEds = map[int]*CodeEditorElement{}
	}
	if cel, ok := e.bodyEds[si]; ok {
		return cel
	}
	idx := si
	lang := e.se.lang
	if lang == "" {
		lang = "ey"
	}
	cw := NewCodeEditor(lang, e.program.Subs[si].Body).WithSize(w, seBodyEdH).WithMinimap(false)
	cw.Embedded = true     // 融入 StructEditor：无边框无缝
	cw.IndentGuides = true // 缩进连线（虚实结合）
	if lang == "go" && e.se.LSPServer != "" { // 内嵌体编辑器接 gopls（聚焦时再启动，见 EnsureLSP）
		cw.LSPServer, cw.LSPWorkspace, cw.LSPFile = e.se.LSPServer, e.se.LSPWorkspace, e.se.LSPFile
	}
	cw.OnChange = func(t string) {
		if idx < len(e.program.Subs) {
			e.program.Subs[idx].Body = t // 编辑写回数据（每次）
		}
	}
	cw.OnEnter = func() { // 回车时同步：代码区改 → 回填变量表（Go 用 AST 提取局部变量，易语言用自动声明）
		if idx < len(e.program.Subs) {
			if e.se.lang == "go" {
				e.goExtractLocals(idx)
			} else {
				e.autoDeclareVars(idx)
			}
		}
	}
	cw.ExtraIdents = func() []CECompletion { // 代码补全源：已声明变量 + 子程序 + 命令库
		var out []CECompletion
		for i := range e.program.Globals {
			out = append(out, CECompletion{Label: e.program.Globals[i].Name, Detail: "全局变量"})
		}
		if idx < len(e.program.Subs) {
			s := &e.program.Subs[idx]
			for i := range s.Params {
				out = append(out, CECompletion{Label: s.Params[i].Name, Detail: "参数"})
			}
			for i := range s.Locals {
				out = append(out, CECompletion{Label: s.Locals[i].Name, Detail: "局部变量"})
			}
		}
		for i := range e.program.Subs {
			out = append(out, CECompletion{Label: e.program.Subs[i].Name, Detail: "子程序"})
		}
		if e.se.lang == "ey" { // 命令库仅易语言模式注入（Go 模式靠 go 高亮的关键字/类型补全）
			for _, c := range eyCommands {
				out = append(out, CECompletion{Label: c, Detail: "命令"})
			}
		}
		return out
	}
	cel := CreateElementFor(cw).(*CodeEditorElement)
	if cw.LSPServer != "" { // 本地函数体坐标 → 完整 Go 文档（整表 ToGo）坐标，喂 gopls
		cel.lspWrap = func(body string) (string, int, int) {
			doc, bodyLine := e.program.GoDocForBody(idx, body)
			return doc, bodyLine, 1 // colOff=1：体行在完整文档里有一个 \t 缩进
		}
	}
	cel.Mount(e, idx) // parent=StructEditor，Offset 跟随；每帧 SetPosition 定位
	cel.Layout(&layout.LayoutContext{Constraints: layout.Tight(w, seBodyEdH)})
	e.bodyEds[idx] = cel
	return cel
}

// BodyEditorAt 返回第 i 个子程序的内嵌逻辑编辑器（供测试/演示访问；未渲染过则 nil）。
func (e *StructEditorElement) BodyEditorAt(i int) Element {
	if e.bodyEds == nil {
		return nil
	}
	return e.bodyEds[i]
}

// layoutBodyEd 按内容自适应高度重新布局内嵌编辑器（每帧调），返回其高度。
// 让代码区"恰好包住代码"地流式排布，融入整体；增删代码行高度自然增减，外层 StructEditor 统一滚。
func (e *StructEditorElement) layoutBodyEd(cel *CodeEditorElement, w float64) float64 {
	bh := cel.EmbeddedContentHeight()
	cel.ed.Height = bh
	cel.Layout(&layout.LayoutContext{Constraints: layout.Tight(w, bh)})
	return bh
}

// ToggleFold 折叠/展开第 i 个子程序（供测试/演示程序化调用）。
func (e *StructEditorElement) ToggleFold(i int) { e.toggleSubFold(i) }

// ToggleGlobalsFold 折叠/展开程序集变量表（供测试/演示程序化调用）。
func (e *StructEditorElement) ToggleGlobalsFold() { e.globalsCollapsed = !e.globalsCollapsed; repaint() }

// addSub 新建一个空子程序，并定位到其函数名格进入编辑（Ctrl+M 快捷键）。
func (e *StructEditorElement) addSub() {
	e.program.Subs = append(e.program.Subs, SESub{Name: "新建子程序", Returns: []SEVar{{Type: "整数型"}}})
	si := len(e.program.Subs) - 1
	if e.activeCE != nil {
		e.activeCE.Blur()
		e.activeCE = nil
	}
	e.focused = true
	e.beginEdit("func:"+itoaCE(si), 0, 0) // 进入新函数名编辑（beginEdit 内含 repaint）
}

// currentSubIndex 返回「当前子程序」索引：优先看选中区段(func/params/locals/returns:si)，
// 否则看焦点所在的内嵌代码编辑器；都不在某子程序里则返回 -1。
func (e *StructEditorElement) currentSubIndex() int {
	for _, pre := range []string{"func:", "params:", "locals:", "returns:"} {
		if strings.HasPrefix(e.selSection, pre) {
			return atoiSafe(e.selSection[len(pre):])
		}
	}
	if e.activeCE != nil {
		for si, cel := range e.bodyEds {
			if cel == e.activeCE {
				return si
			}
		}
	}
	return -1
}

// deleteSub 删除第 si 个子程序（Ctrl+Shift+Delete）。bodyEds 以 si 为键，删除后索引整体前移，
// 故全部卸载、置空重建（删子程序不频繁，可接受重建丢内嵌编辑器的光标/滚动态）。
func (e *StructEditorElement) deleteSub(si int) {
	if si < 0 || si >= len(e.program.Subs) {
		return
	}
	e.program.Subs = append(e.program.Subs[:si], e.program.Subs[si+1:]...)
	for _, cel := range e.bodyEds {
		cel.Unmount()
	}
	e.bodyEds = nil
	e.activeCE = nil
	e.collapsed = nil // 折叠态按 si 索引，整体失效，清空（默认全展开）
	e.editing = false
	e.selSection, e.selRow, e.selCol = "", -1, -1
	repaint()
}

// toggleSubFold 折叠/展开某子程序（行号 gl 仍累加，保持全局行号连续）。
func (e *StructEditorElement) toggleSubFold(si int) {
	if e.collapsed == nil {
		e.collapsed = map[int]bool{}
	}
	e.collapsed[si] = !e.collapsed[si]
	repaint()
}

// ceAt 返回坐标命中的内嵌编辑器（无则 nil）。
func (e *StructEditorElement) ceAt(x, y float64) *CodeEditorElement {
	for _, cel := range e.bodyEds {
		o, s := cel.Offset(), cel.Size()
		if x >= o.X && x <= o.X+s.Width && y >= o.Y && y <= o.Y+s.Height {
			return cel
		}
	}
	return nil
}

func (e *StructEditorElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseWheel:
		me, ok := ev.(*event.MouseEvent)
		if !ok || me.DeltaY == 0 {
			break
		}
		// 悬在某内嵌编辑器上且其内容超高→滚它自己的代码；否则滚整个结构编辑器
		if cel := e.ceAt(me.X, me.Y); cel != nil && float64(len(cel.lines))*ceLineH > cel.Size().Height-8 {
			return cel.HandleEvent(ev)
		}
		e.scrollY -= me.DeltaY * 40
		if e.scrollY < 0 {
			e.scrollY = 0
		}
		max := e.contentH - (e.size.Height - 4)
		if max < 0 {
			max = 0
		}
		if e.scrollY > max {
			e.scrollY = max
		}
		ev.StopPropagation()
		repaint()
		return true

	case event.TypeMouseDown:
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			break
		}
		e.focused = true
		if e.inMinimap(me.X, me.Y) { // 缩略图点击/拖动跳转
			e.draggingMini = true
			e.minimapJump(me.Y)
			return true
		}
		for _, fh := range e.foldHits { // 折叠三角（sub=-1 程序集变量表 / >=0 子程序）
			if me.X >= fh.rect.X && me.X <= fh.rect.X+fh.rect.Width && me.Y >= fh.rect.Y && me.Y <= fh.rect.Y+fh.rect.Height {
				if fh.sub < 0 {
					e.globalsCollapsed = !e.globalsCollapsed
					repaint()
				} else {
					e.toggleSubFold(fh.sub)
				}
				return true
			}
		}
		if cel := e.ceAt(me.X, me.Y); cel != nil { // 点进内嵌代码编辑器
			if e.activeCE != nil && e.activeCE != cel {
				e.activeCE.Blur()
			}
			e.activeCE = cel
			cel.EnsureLSP() // go 模式接了 gopls 则聚焦时启动
			e.editing = false
			e.selSection, e.selRow, e.selCol = "", -1, -1
			return cel.HandleEvent(ev)
		}
		if e.activeCE != nil { // 点到编辑器之外→其失焦
			e.activeCE.Blur()
			e.activeCE = nil
		}
		for _, c := range e.cells { // 命中单元格→选中并进入编辑
			if me.X >= c.rect.X && me.X <= c.rect.X+c.rect.Width && me.Y >= c.rect.Y && me.Y <= c.rect.Y+c.rect.Height {
				e.beginEdit(c.section, c.row, c.col)
				return true
			}
		}
		e.commitEdit()
		e.selSection, e.selRow, e.selCol = "", -1, -1
		repaint()
		return true

	case event.TypeMouseMove, event.TypeMouseUp:
		if me, ok := ev.(*event.MouseEvent); ok && e.draggingMini { // 拖动缩略图视口框
			if ev.Type() == event.TypeMouseUp {
				e.draggingMini = false
			} else {
				e.minimapJump(me.Y)
			}
			return true
		}
		if e.activeCE != nil { // 拖选/滚动条拖动转发给内嵌编辑器
			return e.activeCE.HandleEvent(ev)
		}

	case event.TypeKeyChar:
		keyEv, ok := ev.(*event.KeyEvent)
		if !ok {
			break
		}
		if e.activeCE != nil {
			return e.activeCE.HandleEvent(ev)
		}
		if e.editing && keyEv.Char >= 32 {
			e.editInsert(keyEv.Char)
			return true
		}

	case event.TypeKeyDown:
		keyEv, ok := ev.(*event.KeyEvent)
		if !ok {
			break
		}
		if keyEv.Mods&event.ModCtrl != 0 && keyEv.Key == "M" { // Ctrl+M 新建子程序（优先于转发给代码编辑器）
			e.addSub()
			return true
		}
		if keyEv.Mods&event.ModCtrl != 0 && keyEv.Mods&event.ModShift != 0 && (keyEv.Key == "Delete" || keyEv.Key == "Backspace") {
			if si := e.currentSubIndex(); si >= 0 { // Ctrl+Shift+Delete 删除当前子程序（焦点在代码区时亦可，故先于转发）
				e.deleteSub(si)
				return true
			}
		}
		if e.activeCE != nil {
			return e.activeCE.HandleEvent(ev)
		}
		if e.focused {
			return e.handleEditKey(keyEv)
		}
	}
	return false
}

// CursorClientPos 让 app 启用 IME 并把候选窗定位到光标处：
// 焦点在内嵌编辑器→转发它的；否则用变量表单元格编辑光标坐标。
func (e *StructEditorElement) CursorClientPos() (x, y float64) {
	if e.activeCE != nil {
		return e.activeCE.CursorClientPos()
	}
	return e.cellCaretX, e.cellCaretY
}

// Unmount 卸载时关闭所有内嵌编辑器（释放 LSP 等资源）。
func (e *StructEditorElement) Unmount() {
	for _, cel := range e.bodyEds {
		cel.Unmount()
	}
	e.BaseElement.Unmount()
}

func (e *StructEditorElement) Update(newWidget Widget) {
	if ns, ok := newWidget.(*StructEditor); ok {
		e.se = ns
		e.program = ns.program
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
