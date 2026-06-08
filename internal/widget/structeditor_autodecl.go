package widget

import "strings"

// 易语言（EY）自动声明变量 —— 多语言重构中此实现被移除，按原版还原。
// 测试 TestStructEditorAutoDeclare 依赖；调用点见 structeditor.go 的非 Go 分支 + 内嵌编辑器 OnEnter。

// autoDeclareVars 自动声明变量：扫描子程序逻辑代码里「行首 变量 ＝ …」的赋值，
// 把未声明的赋值目标自动加入该子程序的局部变量表（带类型推导）。
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
// 只认行首标识符紧跟 ＝/=（排除 == 比较），故命令调用「输出调试文本 (…)」不会被误判。
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
