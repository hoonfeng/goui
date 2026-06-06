package widget

import (
	"strings"
)

// StructEditor 的「文本 ↔ 表格」双向序列化：
// Serialize 把表格化的 SEProgram 导出成可读文本，ParseProgram 再把文本解析回 SEProgram，互不丢失（round-trip）。
// 这是「通用代码表格化」的地基——后续接各语言 parser 即可把真实代码文本映射进这套表格模型。
//
// 文本格式（行前缀 + " | " 分隔字段）：
//   变量 名称 | 类型 | 数组 | 备注                  —— 程序集（全局）变量
//   函数 名称 | 返回类型 | 注释                      —— 子程序开始
//   参数 名称 | 类型 | 数组 | 参考 | 备注            —— 函数参数
//   局部 名称 | 类型 | 数组 | 备注                   —— 局部变量
//   代码:                                            —— 逻辑代码块开始
//   ...原样多行代码...
//   代码结束                                         —— 逻辑代码块结束

// Serialize 把 SEProgram 导出成可读文本。
func (p *SEProgram) Serialize() string {
	var b strings.Builder
	for i := range p.Globals {
		v := &p.Globals[i]
		b.WriteString("变量 " + joinFields(v.Name, v.Type, v.Array, v.Note) + "\n")
	}
	for i := range p.Subs {
		s := &p.Subs[i]
		b.WriteString("函数 " + joinFields(s.Name, s.Note) + "\n")
		for j := range s.Returns {
			v := &s.Returns[j]
			b.WriteString("返回 " + joinFields(v.Name, v.Type) + "\n")
		}
		for j := range s.Params {
			v := &s.Params[j]
			b.WriteString("参数 " + joinFields(v.Name, v.Type, v.Array, v.Ref, v.Note) + "\n")
		}
		for j := range s.Locals {
			v := &s.Locals[j]
			b.WriteString("局部 " + joinFields(v.Name, v.Type, v.Array, v.Note) + "\n")
		}
		b.WriteString("代码:\n")
		b.WriteString(s.Body)
		if !strings.HasSuffix(s.Body, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("代码结束\n")
	}
	return b.String()
}

// ParseProgram 把文本解析回 SEProgram（Serialize 的逆）。
func ParseProgram(text string) *SEProgram {
	p := &SEProgram{}
	lines := strings.Split(text, "\n")
	subIdx := -1 // 当前子程序索引
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(t, "变量 "):
			f := splitFields(t[len("变量 "):], 4)
			p.Globals = append(p.Globals, SEVar{Name: f[0], Type: f[1], Array: f[2], Note: f[3]})
		case strings.HasPrefix(t, "函数 "):
			f := splitFields(t[len("函数 "):], 2)
			p.Subs = append(p.Subs, SESub{Name: f[0], Note: f[1]})
			subIdx = len(p.Subs) - 1
		case strings.HasPrefix(t, "返回 ") && subIdx >= 0:
			f := splitFields(t[len("返回 "):], 2)
			p.Subs[subIdx].Returns = append(p.Subs[subIdx].Returns, SEVar{Name: f[0], Type: f[1]})
		case strings.HasPrefix(t, "参数 ") && subIdx >= 0:
			f := splitFields(t[len("参数 "):], 5)
			p.Subs[subIdx].Params = append(p.Subs[subIdx].Params, SEVar{Name: f[0], Type: f[1], Array: f[2], Ref: f[3], Note: f[4]})
		case strings.HasPrefix(t, "局部 ") && subIdx >= 0:
			f := splitFields(t[len("局部 "):], 4)
			p.Subs[subIdx].Locals = append(p.Subs[subIdx].Locals, SEVar{Name: f[0], Type: f[1], Array: f[2], Note: f[3]})
		case t == "代码:" && subIdx >= 0:
			var body []string
			for i++; i < len(lines) && strings.TrimSpace(lines[i]) != "代码结束"; i++ {
				body = append(body, lines[i])
			}
			p.Subs[subIdx].Body = strings.Join(body, "\n")
		}
	}
	return p
}

// joinFields 用 " | " 连接字段（导出）。
func joinFields(fields ...string) string { return strings.Join(fields, " | ") }

// splitFields 按 "|" 拆出 n 个字段并去空白（解析），不足补空。
func splitFields(s string, n int) []string {
	parts := strings.Split(s, "|")
	out := make([]string, n)
	for i := 0; i < n; i++ {
		if i < len(parts) {
			out[i] = strings.TrimSpace(parts[i])
		}
	}
	return out
}
