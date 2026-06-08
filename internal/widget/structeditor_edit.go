package widget

import (
	"strconv"
	"strings"
	"time"

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
	case section == "consts":
		return constCols
	case section == "types":
		return typeDefCols
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
	case section == "consts":
		if row >= 0 && row < len(e.program.Consts) {
			return &e.program.Consts[row]
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
	// types 区段：绑定 SEType
	if section == "types" {
		if row >= 0 && row < len(e.program.Types) {
			td := &e.program.Types[row]
			switch col {
			case 0:
				return td.Name
			case 1:
				return string(td.Kind)
			case 2:
				return typeMembersSummary(td) // 成员摘要：字段/方法列表（直观）
			case 3:
				return td.Note
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
	// types 区段：写 SEType
	if section == "types" {
		if row >= 0 && row < len(e.program.Types) {
			td := &e.program.Types[row]
			switch col {
			case 0:
				td.Name = val
			case 1:
				td.Kind = SETypeKind(val)
			case 2:
				if td.Kind == SETypeAlias {
					td.TypeExpr = val
				}
				// else: 只读字段（计数），不建议直接编辑
			case 3:
				td.Note = val
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
	if section == "types" {
		return 3 // types 有 4 列(0..3)
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
	case section == "consts":
		return len(e.program.Consts)
	case section == "types":
		return len(e.program.Types)
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
	newVal := string(val[:e.editCol]) + string(ch) + string(val[e.editCol:])
	e.setCellValue(e.selSection, e.selRow, e.selCol, newVal)
	e.editCol++
	repaint()
}

func (e *StructEditorElement) editDelete() {
	val := []rune(e.cellValue(e.selSection, e.selRow, e.selCol))
	if len(val) > 0 && e.editCol > 0 {
		newVal := string(val[:e.editCol-1]) + string(val[e.editCol:])
		e.setCellValue(e.selSection, e.selRow, e.selCol, newVal)
		e.editCol--
	}
	repaint()
}

func (e *StructEditorElement) editBackspace() { e.editDelete() }

func (e *StructEditorElement) editCursorLeft() {
	if e.editCol > 0 {
		e.editCol--
	}
}

func (e *StructEditorElement) editCursorRight() {
	val := []rune(e.cellValue(e.selSection, e.selRow, e.selCol))
	if e.editCol < len(val) {
		e.editCol++
	}
}

func (e *StructEditorElement) editHome()  { e.editCol = 0 }
func (e *StructEditorElement) editEnd() {
	e.editCol = len([]rune(e.cellValue(e.selSection, e.selRow, e.selCol)))
}

// ── 键盘导航 ──

func (e *StructEditorElement) handleKey(ev event.KeyEvent) {
	switch ev.Key {
	case event.KeyEscape:
		if e.editing {
			e.commitEdit()
		} else {
			e.selSection, e.selRow, e.selCol = "", -1, -1
			repaint()
		}
	case event.KeyUp:
		if e.editing {
			e.commitEdit()
		}
		if e.selRow > 0 {
			e.selRow--
		}
		repaint()
	case event.KeyDown:
		if e.editing {
			e.commitEdit()
		}
		if e.selRow < e.sectionRows(e.selSection)-1 {
			e.selRow++
		}
		repaint()
	case event.KeyEnter:
		if e.editing {
			e.commitEdit()
		}
		if e.selRow < e.sectionRows(e.selSection)-1 {
			e.selRow++
		} else {
			// 末尾回车：新增一行（仅在变量表支持，函数声明行禁止），并进入新行首格编辑
			if !strings.HasPrefix(e.selSection, "func:") && e.selSection != "" {
				e.addRow(e.selSection)
				e.selRow = e.sectionRows(e.selSection) - 1
				e.selCol, e.editing, e.editCol = 0, true, 0
			}
		}
		repaint()
	case event.KeyTab, event.KeyRight:
		if e.editing {
			e.editCursorRight()
			return
		}
		if ev.Key == event.KeyTab {
			e.commitEdit()
			maxCol := e.sectionMaxCol(e.selSection)
			if e.selCol < maxCol && e.selSection != "" {
				e.selCol++
			}
			repaint()
		}
	case event.KeyShiftTab, event.KeyLeft:
		if e.editing {
			e.editCursorLeft()
			return
		}
		if ev.Key == event.KeyShiftTab {
			e.commitEdit()
			if e.selCol > 0 {
				e.selCol--
			}
			repaint()
		}
	case event.KeyHome:
		e.editHome()
	case event.KeyEnd:
		e.editEnd()
	case event.KeyDelete:
		e.editDelete()
	case event.KeyBackspace:
		e.editBackspace()
	default:
		if ev.Char != 0 && !e.isControl(ev.Key) {
			if !e.editing {
				e.beginEdit(e.selSection, e.selRow, e.selCol)
			}
			e.editInsert(ev.Char)
		}
	}
}

func (e *StructEditorElement) isControl(key event.Key) bool {
	switch key {
	case event.KeyUp, event.KeyDown, event.KeyLeft, event.KeyRight,
		event.KeyEnter, event.KeyTab, event.KeyEscape,
		event.KeyHome, event.KeyEnd, event.KeyDelete, event.KeyBackspace:
		return true
	}
	return false
}

// addRow 在末尾追加一新行（变量表用）。
func (e *StructEditorElement) addRow(section string) {
	switch {
	case section == "imports":
		e.program.Imports = append(e.program.Imports, "")
	case section == "globals":
		e.program.Globals = append(e.program.Globals, SEVar{})
	case section == "consts":
		e.program.Consts = append(e.program.Consts, SEVar{})
	case section == "types":
		e.program.Types = append(e.program.Types, SEType{Kind: SETypeStruct})
	case strings.HasPrefix(section, "params:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			e.program.Subs[si].Params = append(e.program.Subs[si].Params, SEVar{})
		}
	case strings.HasPrefix(section, "locals:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			e.program.Subs[si].Locals = append(e.program.Subs[si].Locals, SEVar{})
		}
	case strings.HasPrefix(section, "returns:"):
		if si := atoiSafe(section[8:]); si < len(e.program.Subs) {
			e.program.Subs[si].Returns = append(e.program.Subs[si].Returns, SEVar{})
		}
	}
}

// deleteRow 删除第 row 行（变量表用）。
func (e *StructEditorElement) deleteRow(section string, row int) {
	switch {
	case section == "imports":
		if row >= 0 && row < len(e.program.Imports) {
			e.program.Imports = append(e.program.Imports[:row], e.program.Imports[row+1:]...)
		}
	case section == "globals":
		if row >= 0 && row < len(e.program.Globals) {
			e.program.Globals = append(e.program.Globals[:row], e.program.Globals[row+1:]...)
		}
	case section == "consts":
		if row >= 0 && row < len(e.program.Consts) {
			e.program.Consts = append(e.program.Consts[:row], e.program.Consts[row+1:]...)
		}
	case section == "types":
		if row >= 0 && row < len(e.program.Types) {
			e.program.Types = append(e.program.Types[:row], e.program.Types[row+1:]...)
		}
	case strings.HasPrefix(section, "params:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) && row >= 0 && row < len(e.program.Subs[si].Params) {
			e.program.Subs[si].Params = append(e.program.Subs[si].Params[:row], e.program.Subs[si].Params[row+1:]...)
		}
	case strings.HasPrefix(section, "locals:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) && row >= 0 && row < len(e.program.Subs[si].Locals) {
			e.program.Subs[si].Locals = append(e.program.Subs[si].Locals[:row], e.program.Subs[si].Locals[row+1:]...)
		}
	case strings.HasPrefix(section, "returns:"):
		if si := atoiSafe(section[8:]); si < len(e.program.Subs) && row >= 0 && row < len(e.program.Subs[si].Returns) {
			e.program.Subs[si].Returns = append(e.program.Subs[si].Returns[:row], e.program.Subs[si].Returns[row+1:]...)
		}
	}
}

// ── 行删除快捷键（Ctrl+D / Ctrl+Shift+K）已在 codeeditor 变量表截获 ──

// ── 鼠标 / 触摸 ──

func (e *StructEditorElement) handlePointer(ev event.MouseEvent) {
	pos := e.Offset()
	cx := ev.X - pos.X
	cy := ev.Y - pos.Y

	if ev.Type() == event.TypeMouseDown {
		if e.miniRect.Width > 0 && cx >= e.miniRect.X-pos.X && cx <= e.miniRect.X-pos.X+e.miniRect.Width {
			e.minimapJump(ev.Y - pos.Y)
			return
		}
		// 折叠三角
		for _, fh := range e.foldHits {
			if cx >= fh.rect.X && cx <= fh.rect.X+fh.rect.Width &&
				cy >= fh.rect.Y && cy <= fh.rect.Y+fh.rect.Height {
				if fh.sub < 0 {
					// 程序集变量折叠
					e.globalsCollapsed = !e.globalsCollapsed
				} else {
					if e.collapsed == nil {
						e.collapsed = map[int]bool{}
					}
					e.collapsed[fh.sub] = !e.collapsed[fh.sub]
				}
				repaint()
				return
			}
		}
		// 单元格命中
		for _, c := range e.cells {
			if cx >= c.rect.X && cx <= c.rect.X+c.rect.Width &&
				cy >= c.rect.Y && cy <= c.rect.Y+c.rect.Height {
				e.beginEdit(c.section, c.row, c.col)
				return
			}
		}
		// 空白点击取消选中
		e.selSection, e.selRow, e.selCol = "", -1, -1
		repaint()
	}
}
