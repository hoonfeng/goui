package widget

import (
	"strings"

	"github.com/hoonfeng/goui/pkg/event"
)

// 本文件补齐多语言 StructEditor 重构中「被调用但尚未定义」的方法，使 包 + 测试 编译通过。
// handleEditKey / inMinimap / insertRow / moveSelect / sectionVarsRef 为确定性实现。
// （autoDeclareVars 及其推导helpers 已在 structeditor_autodecl.go 按原实现还原。）

// handleEditKey 焦点在结构编辑器时的键盘处理：包装既有 handleKey，返回是否消费。
func (e *StructEditorElement) handleEditKey(ev *event.KeyEvent) bool {
	if ev == nil {
		return false
	}
	e.handleKey(*ev)
	return true
}

// inMinimap 判断坐标 (x,y) 是否落在缩略图区域（miniRect，与鼠标事件同坐标系）。
func (e *StructEditorElement) inMinimap(x, y float64) bool {
	r := e.miniRect
	return r.Width > 0 && r.Height > 0 &&
		x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// sectionVarsRef 返回某变量类区段的切片指针（globals/consts/params:/locals:/returns:）；非变量区段返回 nil。
func (e *StructEditorElement) sectionVarsRef(section string) *[]SEVar {
	switch {
	case section == "globals":
		return &e.program.Globals
	case section == "consts":
		return &e.program.Consts
	case strings.HasPrefix(section, "params:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			return &e.program.Subs[si].Params
		}
	case strings.HasPrefix(section, "locals:"):
		if si := atoiSafe(section[7:]); si < len(e.program.Subs) {
			return &e.program.Subs[si].Locals
		}
	case strings.HasPrefix(section, "returns:"):
		if si := atoiSafe(section[8:]); si < len(e.program.Subs) {
			return &e.program.Subs[si].Returns
		}
	case strings.HasPrefix(section, "typefields:"):
		return e.typeMembers(atoiSafe(section[11:]))
	}
	return nil
}

// insertRow 在 section 的变量表 idx 处插入空行，选中新行首格并进入编辑。
func (e *StructEditorElement) insertRow(section string, idx int) {
	vp := e.sectionVarsRef(section)
	if vp == nil {
		return
	}
	v := *vp
	if idx < 0 {
		idx = 0
	}
	if idx > len(v) {
		idx = len(v)
	}
	v = append(v, SEVar{})
	copy(v[idx+1:], v[idx:])
	v[idx] = SEVar{}
	*vp = v
	e.selSection, e.selRow, e.selCol, e.editing, e.editCol = section, idx, 0, true, 0
	repaint()
}

// moveSelect 非编辑态方向键移格：移动 (drow, dcol)，列末右移回绕到下一行首列、首列左移回绕到上一行末列。
func (e *StructEditorElement) moveSelect(drow, dcol int) {
	maxCol := e.sectionMaxCol(e.selSection)
	e.selCol += dcol
	if e.selCol > maxCol {
		e.selCol = 0
		drow++
	} else if e.selCol < 0 {
		e.selCol = maxCol
		drow--
	}
	e.selRow += drow
	rows := e.sectionRows(e.selSection)
	if e.selRow < 0 {
		e.selRow = 0
	}
	if rows > 0 && e.selRow >= rows {
		e.selRow = rows - 1
	}
	e.editing = false
	repaint()
}
