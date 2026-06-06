package widget

import (
	"strconv"
	"strings"
	"time"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
)

// StructEditor 单元格编辑：按 section/row/col 读写 program 数据，单元格内文本编辑 + 键盘导航。

func atoiSafe(s string) int { n, _ := strconv.Atoi(s); return n }

// curSchema 取生效的表结构（兜底 DefaultSchema），并缓存回 e.schema。
func (e *StructEditorElement) curSchema() *SESchema {
	if e.schema == nil {
		e.schema = DefaultSchema()
	}
	return e.schema
}

// importCols 导入区的列定义（单列字符串）。
var importCols = []SECol{{Title: "导入", Field: SEFieldName, Weight: 1.0}}

// cols 返回某区段的列定义（按 schema）。func 声明行绑定 SESub（非 SEVar），单独处理，返回 nil。
func (e *StructEditorElement) cols(section string) []SECol {
	sc := e.curSchema()
	switch {
	case section == "imports":
		return importCols
	case section == "globals":
		return sc.Globals
	case strings.HasPrefix(section, "params:"):
		return sc.Params
	case strings.HasPrefix(section, "returns:"):
		return sc.Returns
	case strings.HasPrefix(section, "locals:"):
		return sc.Locals
	}
	return nil
}

func (e *StructEditorElement) varPtr(section string, row int) *SEVar {
	switch {
	case section == "globals":
		if row >= 0 && row < len(e.program.Globals) {
			return &e.program.Globals[row]
		}
	case strings.HasPrefix(section, "params:"):
		si := atoiSafe(section[7:])
		if si < len(e.program.Subs) && row >= 0 && row < len(e.program.Subs[si].Params) {
			return &e.program.Subs[si].Params[row]
		}
	case strings.HasPrefix(section, "locals:"):
		si := atoiSafe(section[7:])
		if si < len(e.program.Subs) && row >= 0 && row < len(e.program.Subs[si].Locals) {
			return &e.program.Subs[si].Locals[row]
		}
	case strings.HasPrefix(section, "returns:"):
		si := atoiSafe(section[8:])
		if si < len(e.program.Subs) && row >= 0 && row < len(e.program.Subs[si].Returns) {
			return &e.program.Subs[si].Returns[row]
		}
	}
	return nil
}

func (e *StructEditorElement) cellValue(section string, row, col int) string {
	if strings.HasPrefix(section, "func:") { // 函数声明行绑定 SESub（非 SEVar）：col0=函数名, col1=注释
		if si := atoiSafe(section[5:]); si < len(e.program.Subs) {
			switch col {
			case 0:
				return e.program.Subs[si].Name
			case 1:
				return e.program.Subs[si].Note
			}
		}
		return ""
	}
	if section == "imports" { // 导入区：单列字符串，直读 program.Imports
		if row >= 0 && row < len(e.program.Imports) {
			return e.program.Imports[row]
		}
		return ""
	}
	cols := e.cols(section)
	if col < 0 || col >= len(cols) {
		return ""
	}
	if v := e.varPtr(section, row); v != nil {
		return v.field(cols[col].Field)
	}
	return ""
}

func (e *StructEditorElement) setCellValue(section string, row, col int, val string) {
	if strings.HasPrefix(section, "func:") { // 函数声明行绑定 SESub（非 SEVar）：col0=函数名, col1=注释
		if si := atoiSafe(section[5:]); si < len(e.program.Subs) {
			switch col {
			case 0:
				e.program.Subs[si].Name = val
			case 1:
				e.program.Subs[si].Note = val
			}
		}
		return
	}
	if section == "imports" { // 导入区：写回 program.Imports（行超出则补齐，支持空表点击占位行加第一个）
		if row < 0 {
			return
		}
		for len(e.program.Imports) <= row {
			e.program.Imports = append(e.program.Imports, "")
		}
		e.program.Imports[row] = val
		return
	}
	cols := e.cols(section)
	if col < 0 || col >= len(cols) {
		return
	}
	if v := e.varPtr(section, row); v != nil {
		v.setField(cols[col].Field, val)
	}
}

func (e *StructEditorElement) sectionMaxCol(section string) int {
	if strings.HasPrefix(section, "func:") {
		return 1 // 函数声明：函数名(0) / 注释(1)
	}
	if c := e.cols(section); len(c) > 0 {
		return len(c) - 1
	}
	return 0
}

func (e *StructEditorElement) sectionRows(section string) int {
	switch {
	case strings.HasPrefix(section, "func:"):
		return 1 // 函数声明只有一行
	case section == "imports":
		if n := len(e.program.Imports); n > 0 {
			return n
		}
		if e.se.lang == "go" {
			return 1 // 空导入的占位行（可点击/编辑加第一个）
		}
		return 0
	case section == "globals":
		return len(e.program.Globals)
	case strings.HasPrefix(section, "params:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			return len(e.program.Subs[si].Params)
		}
	case strings.HasPrefix(section, "locals:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			return len(e.program.Subs[si].Locals)
		}
	case strings.HasPrefix(section, "returns:"):
		if si := atoiSafe(section[8:]); si < len(e.program.Subs) {
			return len(e.program.Subs[si].Returns)
		}
	}
	return 0
}

// ── 编辑 ──

func (e *StructEditorElement) beginEdit(section string, row, col int) {
	cols := e.cols(section)
	if col >= 0 && col < len(cols) && cols[col].Check { // 复选框列：点击切换 是/空（如参考/传址），不进文本编辑
		if v := e.varPtr(section, row); v != nil {
			f := cols[col].Field
			if v.field(f) == "是" {
				v.setField(f, "")
			} else {
				v.setField(f, "是")
			}
		}
		e.selSection, e.selRow, e.selCol = section, row, col
		e.editing = false
		repaint()
		return
	}
	e.selSection, e.selRow, e.selCol = section, row, col
	e.editing = true
	e.editCol = len([]rune(e.cellValue(section, row, col)))
	e.focusTime = time.Now()
	repaint()
}

func (e *StructEditorElement) commitEdit() { e.editing = false }

func (e *StructEditorElement) editInsert(ch rune) {
	val := []rune(e.cellValue(e.selSection, e.selRow, e.selCol))
	if e.editCol > len(val) {
		e.editCol = len(val)
	}
	nv := string(val[:e.editCol]) + string(ch) + string(val[e.editCol:])
	e.setCellValue(e.selSection, e.selRow, e.selCol, nv)
	e.editCol++
	e.focusTime = time.Now()
	repaint()
}

func (e *StructEditorElement) handleEditKey(k *event.KeyEvent) bool {
	if k.Mods&event.ModCtrl != 0 { // Ctrl+Enter 增行 / Ctrl+Delete 删行
		switch k.Key {
		case "Enter":
			if e.selSection != "" {
				e.insertRow(e.selSection, e.selRow+1)
			}
		case "Delete", "Backspace":
			if e.selSection != "" && e.selRow >= 0 {
				e.deleteRow(e.selSection, e.selRow)
			}
		}
		return true
	}
	if !e.editing { // 非编辑态：方向键移单元格、Enter/F2 进入编辑
		switch k.Key {
		case "Enter", "F2":
			if e.selRow >= 0 {
				e.beginEdit(e.selSection, e.selRow, e.selCol)
			}
		case "ArrowUp":
			e.moveSelect(-1, 0)
		case "ArrowDown":
			e.moveSelect(1, 0)
		case "ArrowLeft":
			e.moveSelect(0, -1)
		case "ArrowRight":
			e.moveSelect(0, 1)
		}
		return true
	}
	val := []rune(e.cellValue(e.selSection, e.selRow, e.selCol))
	if e.editCol > len(val) {
		e.editCol = len(val)
	}
	switch k.Key {
	case "Escape":
		e.commitEdit()
		repaint()
	case "Enter":
		e.commitEdit()
		if strings.HasPrefix(e.selSection, "func:") {
			break // 函数声明行不增行（只有一行）
		}
		if n := e.sectionRows(e.selSection); e.selRow+1 >= n {
			e.insertRow(e.selSection, n) // 末尾行回车→追加新行并进入编辑
		} else {
			e.beginEdit(e.selSection, e.selRow+1, e.selCol) // 否则移到下一行同列
		}
	case "Tab":
		e.commitEdit()
		e.moveCell(0, 1)
	case "Backspace":
		if e.editCol > 0 {
			e.setCellValue(e.selSection, e.selRow, e.selCol, string(val[:e.editCol-1])+string(val[e.editCol:]))
			e.editCol--
			e.focusTime = time.Now()
			repaint()
		}
	case "Delete":
		if e.editCol < len(val) {
			e.setCellValue(e.selSection, e.selRow, e.selCol, string(val[:e.editCol])+string(val[e.editCol+1:]))
			e.focusTime = time.Now()
			repaint()
		}
	case "ArrowLeft":
		if e.editCol > 0 {
			e.editCol--
			e.focusTime = time.Now()
			repaint()
		}
	case "ArrowRight":
		if e.editCol < len(val) {
			e.editCol++
			e.focusTime = time.Now()
			repaint()
		}
	case "Home":
		e.editCol = 0
		repaint()
	case "End":
		e.editCol = len(val)
		repaint()
	}
	return true
}

// moveCell 移动到相邻单元格（同 section），进入编辑。
func (e *StructEditorElement) moveCell(dRow, dCol int) {
	maxCol := e.sectionMaxCol(e.selSection)
	r, c := e.selRow+dRow, e.selCol+dCol
	if c > maxCol {
		c = 0
		r++
	}
	if c < 0 {
		c = maxCol
		r--
	}
	n := e.sectionRows(e.selSection)
	if r < 0 {
		r = 0
	}
	if r >= n {
		if n == 0 {
			return
		}
		r = n - 1
	}
	e.beginEdit(e.selSection, r, c)
}

func (e *StructEditorElement) isCellCursorVisible() bool {
	if !e.editing {
		return false
	}
	return time.Since(e.focusTime).Milliseconds()%1000 < 530
}

func (e *StructEditorElement) measure(s string) float64 {
	if s == "" {
		return 0
	}
	if e.lastCanvas != nil {
		return e.lastCanvas.MeasureText(s, ceCellFont()).Width
	}
	return canvas.MeasureTextGlobal(s, ceCellFont()).Width
}

func ceCellFont() canvas.Font { f := canvas.DefaultFont(); f.Size = 13; return f }

// ── 增删行 / 选区移动（批③）──

// insertStrAt / removeStrAt 字符串切片增删（导入区用）。
func insertStrAt(s []string, i int, v string) []string {
	if i < 0 {
		i = 0
	}
	if i > len(s) {
		i = len(s)
	}
	s = append(s, "")
	copy(s[i+1:], s[i:])
	s[i] = v
	return s
}

func removeStrAt(s []string, i int) []string {
	if i < 0 || i >= len(s) {
		return s
	}
	return append(s[:i], s[i+1:]...)
}

func insertVarAt(s []SEVar, i int, v SEVar) []SEVar {
	if i < 0 {
		i = 0
	}
	if i > len(s) {
		i = len(s)
	}
	s = append(s, SEVar{})
	copy(s[i+1:], s[i:])
	s[i] = v
	return s
}

func removeVarAt(s []SEVar, i int) []SEVar {
	if i < 0 || i >= len(s) {
		return s
	}
	return append(s[:i], s[i+1:]...)
}

// insertRow 在 section 的 at 位置插入一空行，并进入第一格编辑。
func (e *StructEditorElement) insertRow(section string, at int) {
	switch {
	case section == "imports":
		e.program.Imports = insertStrAt(e.program.Imports, at, `""`)
	case section == "globals":
		e.program.Globals = insertVarAt(e.program.Globals, at, SEVar{})
	case strings.HasPrefix(section, "params:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			e.program.Subs[si].Params = insertVarAt(e.program.Subs[si].Params, at, SEVar{})
		}
	case strings.HasPrefix(section, "locals:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			e.program.Subs[si].Locals = insertVarAt(e.program.Subs[si].Locals, at, SEVar{})
		}
	case strings.HasPrefix(section, "returns:"):
		if si := atoiSafe(section[8:]); si < len(e.program.Subs) {
			e.program.Subs[si].Returns = insertVarAt(e.program.Subs[si].Returns, at, SEVar{})
		}
	}
	e.beginEdit(section, at, 0)
}

// deleteRow 删除 section 的 row 行。
func (e *StructEditorElement) deleteRow(section string, row int) {
	switch {
	case section == "imports":
		e.program.Imports = removeStrAt(e.program.Imports, row)
	case section == "globals":
		e.program.Globals = removeVarAt(e.program.Globals, row)
	case strings.HasPrefix(section, "params:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			e.program.Subs[si].Params = removeVarAt(e.program.Subs[si].Params, row)
		}
	case strings.HasPrefix(section, "locals:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			e.program.Subs[si].Locals = removeVarAt(e.program.Subs[si].Locals, row)
		}
	case strings.HasPrefix(section, "returns:"):
		if si := atoiSafe(section[8:]); si < len(e.program.Subs) {
			e.program.Subs[si].Returns = removeVarAt(e.program.Subs[si].Returns, row)
		}
	}
	e.commitEdit()
	if n := e.sectionRows(section); e.selRow >= n {
		e.selRow = n - 1
	}
	repaint()
}

// autoDeclareVars 自动声明变量：扫描子程序逻辑代码里「行首 变量 ＝ …」的赋值，
// 把未声明的赋值目标自动加入该子程序的局部变量表（默认整数型）。
func (e *StructEditorElement) autoDeclareVars(si int) {
	if si >= len(e.program.Subs) {
		return
	}
	sub := &e.program.Subs[si]
	// fixed：不可重声明（全局/参数/手动局部/子程序名）
	fixed := map[string]bool{}
	for i := range e.program.Globals {
		fixed[e.program.Globals[i].Name] = true
	}
	for i := range sub.Params {
		fixed[sub.Params[i].Name] = true
	}
	for i := range e.program.Subs {
		fixed[e.program.Subs[i].Name] = true // 子程序名不当变量
	}
	autoIdx := map[string]int{} // 已自动声明的局部变量 → 在 Locals 的索引
	for i := range sub.Locals {
		if sub.Locals[i].Note == "自动声明" {
			autoIdx[sub.Locals[i].Name] = i
		} else {
			fixed[sub.Locals[i].Name] = true // 手动声明的局部，类型由用户掌控
		}
	}
	for _, line := range strings.Split(sub.Body, "\n") {
		name, rhs := assignTarget(line)
		if name == "" || rhs == "" || isEYReserved(name) || fixed[name] {
			continue // 值为空不急着声明（避免输入中途按默认类型定型）
		}
		typ := e.inferAssignType(rhs)
		if idx, ok := autoIdx[name]; ok {
			sub.Locals[idx].Type = typ // 随输入持续重新推导类型（如 1→1.5 由整数型变小数型）
		} else {
			sub.Locals = append(sub.Locals, SEVar{Name: name, Type: typ, Note: "自动声明"})
			autoIdx[name] = len(sub.Locals) - 1
		}
	}
}

// assignTarget 解析一行「行首赋值」：返回目标变量名 + 右值（如 "积 ＝ 积 × i" → "积","积 × i"）；非赋值行返回 "","".
// 只认行首标识符紧跟 ＝/=（排除 == 比较），故命令调用「输出调试文本 (…)」「计次循环首 (…)」不会被误判。
func assignTarget(line string) (name, rhs string) {
	rs := []rune(strings.TrimLeft(line, " \t"))
	if len(rs) == 0 || !isIdentStart(rs[0]) || rs[0] == '＝' || rs[0] == '=' {
		return "", ""
	}
	i := 1
	for i < len(rs) && isIdentPart(rs[i]) && rs[i] != '＝' && rs[i] != '=' {
		i++
	}
	nm := string(rs[:i])
	for i < len(rs) && rs[i] == ' ' { // 跳过变量名与等号间空格
		i++
	}
	if i >= len(rs) {
		return "", ""
	}
	if rs[i] == '＝' || (rs[i] == '=' && !(i+1 < len(rs) && rs[i+1] == '=')) {
		return nm, strings.TrimSpace(string(rs[i+1:]))
	}
	return "", ""
}

// isEYReserved 是否为易语言关键字/类型（不能当变量自动声明）。
func isEYReserved(name string) bool {
	return ceLangEY.keywords[name] || ceLangEY.types[name]
}

// inferAssignType 按右值推导变量类型：函数调用→被调子程序返回类型；否则按字面量。
func (e *StructEditorElement) inferAssignType(rhs string) string {
	rhs = strings.TrimSpace(rhs)
	for i := range e.program.Subs { // 函数调用：用被调子程序的(单个)返回类型
		nm := e.program.Subs[i].Name
		if nm != "" && strings.HasPrefix(rhs, nm) && len(e.program.Subs[i].Returns) == 1 {
			return e.program.Subs[i].Returns[0].Type
		}
	}
	return inferEYLiteralType(rhs)
}

// inferEYLiteralType 按字面量推导易语言类型：字符串→文本型、真假→逻辑型、小数→小数型、整数→整数型。
func inferEYLiteralType(rhs string) string {
	if rhs == "" {
		return "整数型"
	}
	if strings.HasPrefix(rhs, "\"") || strings.HasPrefix(rhs, "“") || strings.HasPrefix(rhs, "「") {
		return "文本型"
	}
	switch rhs {
	case "真", "假", "true", "false":
		return "逻辑型"
	}
	dot, ok := false, true
	for i, r := range rhs {
		switch {
		case r == '-' && i == 0:
		case r == '.':
			if dot {
				ok = false
			}
			dot = true
		case r < '0' || r > '9':
			ok = false
		}
		if !ok {
			break
		}
	}
	if ok {
		if dot {
			return "小数型"
		}
		return "整数型"
	}
	return "整数型" // 表达式/未知 → 默认整数型
}

// moveSelect 非编辑态移动选中单元格（不进入编辑）。
func (e *StructEditorElement) moveSelect(dRow, dCol int) {
	if e.selSection == "" {
		return
	}
	maxCol := e.sectionMaxCol(e.selSection)
	r, c := e.selRow+dRow, e.selCol+dCol
	if c > maxCol {
		c = 0
		r++
	}
	if c < 0 {
		c = maxCol
		r--
	}
	n := e.sectionRows(e.selSection)
	if n == 0 {
		return
	}
	if r < 0 {
		r = 0
	}
	if r >= n {
		r = n - 1
	}
	e.selRow, e.selCol = r, c
	repaint()
}
