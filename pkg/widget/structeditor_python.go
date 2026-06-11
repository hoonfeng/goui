package widget

import (
	"regexp"
	"strings"
)

// ─── Python LanguageProvider ───────────────────────────────────
// 用正则解析 Python 源码：顶级函数、类、模块级变量。
// 解析较 Go 轻量（无标准 AST），覆盖常见构造即可。

type pyLangProvider struct{}

func (pyLangProvider) Name() string      { return "py" }
func (pyLangProvider) Schema() *SESchema { return DefaultSchema() }

// pyImportRe 匹配 import / from … import。
var pyImportRe = regexp.MustCompile(`^(import\s+|from\s+\S+\s+import\s+)`)

// pyClassRe 匹配 class 定义：class Foo(Base):  捕获类名、基类、注释。
var pyClassRe = regexp.MustCompile(`^class\s+(\w+)\s*(?:\(([^)]*)\))?\s*:`)

// pyDefRe 匹配函数/方法定义：def foo(a, b):  捕获函数名、参数。
var pyDefRe = regexp.MustCompile(`^def\s+(\w+)\s*\(([^)]*)\)\s*:`)

// pyGlobalRe 匹配模块级赋值：X = ... 或 X: type = ...
var pyGlobalRe = regexp.MustCompile(`^(\w+)\s*(?::\s*(\w+))?\s*=`)

// pyDecorRe 匹配装饰器（跳过装饰器行，因为 def/class 会捕获后续行）。
var pyDecorRe = regexp.MustCompile(`^\s*@`)

func (pyLangProvider) Parse(src string) (*SEProgram, error) {
	p := &SEProgram{}
	lines := strings.Split(src, "\n")
	inClass := "" // 当前所在类名

	// 先收集 doc comment（前一行非空非构造为注释）
	lastComment := ""

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 注释行
		if strings.HasPrefix(trimmed, "#") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			continue
		}
		// 空行
		if trimmed == "" {
			continue
		}
		// 装饰器 → 跳过（def/class 紧随其后捕获）
		if pyDecorRe.MatchString(trimmed) {
			continue
		}

		// import / from ... import
		if pyImportRe.MatchString(trimmed) {
			p.Imports = append(p.Imports, trimmed)
			lastComment = ""
			continue
		}

		// class 定义
		if m := pyClassRe.FindStringSubmatch(trimmed); m != nil {
			td := SEType{
				Name: m[1],
				Kind: SETypeStruct, // Python class → struct
				Note: lastComment,
			}
			if m[2] != "" {
				td.TypeExpr = m[2] // 基类列表
			}
			p.Types = append(p.Types, td)
			inClass = m[1]
			lastComment = ""
			continue
		}

		// 函数/方法定义
		if m := pyDefRe.FindStringSubmatch(trimmed); m != nil {
			sub := SESub{
				Name: m[1],
				Note: lastComment,
			}
			// 解析参数
			params := strings.Split(m[2], ",")
			for _, pr := range params {
				pr = strings.TrimSpace(pr)
				if pr == "" || pr == "self" || pr == "cls" {
					continue
				}
				// 支持 "name: type = default" 格式
				parts := strings.SplitN(pr, ":", 2)
				name := strings.TrimSpace(parts[0])
				typ := ""
				if len(parts) > 1 {
					// 去掉默认值
					typPart := strings.SplitN(strings.TrimSpace(parts[1]), "=", 2)
					typ = strings.TrimSpace(typPart[0])
				}
				sub.Params = append(sub.Params, SEVar{Name: name, Type: typ})
			}
			// 如果是类中的方法，设置接收器
			if inClass != "" {
				sub.Recv = "(self)"
			}
			// 提取函数体：扫描后续缩进行
			bodyStart := i + 1
			for bodyStart < len(lines) && strings.TrimSpace(lines[bodyStart]) == "" {
				bodyStart++
			}
			if bodyStart < len(lines) {
				// 检测本函数体的缩进级别
				indent := countIndent(lines[bodyStart])
				bodyLines := []string{}
				for j := bodyStart; j < len(lines); j++ {
					if strings.TrimSpace(lines[j]) == "" {
						bodyLines = append(bodyLines, "")
						continue
					}
					if countIndent(lines[j]) < indent {
						break
					}
					// 去掉公共缩进
					bodyLines = append(bodyLines, lines[j][indent:])
				}
				sub.Body = strings.Join(bodyLines, "\n")
				// 更新 i 到 body 末尾
				i = bodyStart + len(bodyLines) - 1
			}
			// 回到类的子程序还是顶级函数？
			if inClass != "" {
				// 作为类的方法 → 使用类型名+方法名作为子程序名
				// 同时将方法加到类型的 Sub 中
				p.Subs = append(p.Subs, sub)
			} else {
				p.Subs = append(p.Subs, sub)
				inClass = "" // 离开类
			}
			lastComment = ""
			continue
		}

		// 模块级赋值（全局变量/常量）
		if inClass == "" {
			if m := pyGlobalRe.FindStringSubmatch(trimmed); m != nil {
				name := m[1]
				typ := m[2]
				sv := SEVar{Name: name, Type: typ, Note: lastComment}
				// 全大写 → 常量
				if strings.ToUpper(name) == name && len(name) > 1 {
					p.Consts = append(p.Consts, sv)
				} else {
					p.Globals = append(p.Globals, sv)
				}
				lastComment = ""
				continue
			}
		}
		lastComment = ""
	}
	return p, nil
}

func (pyLangProvider) Generate(p *SEProgram) string {
	var b strings.Builder
	// package 等价物（Python 无 package 声明，直接写 import）
	for i := range p.Imports {
		b.WriteString(p.Imports[i] + "\n")
	}
	// 常量
	for i := range p.Consts {
		if i == 0 && len(p.Imports) > 0 {
			b.WriteString("\n")
		}
		v := &p.Consts[i]
		if v.Note != "" {
			b.WriteString("# " + v.Note + "\n")
		}
		b.WriteString(v.Name)
		if v.Type != "" {
			b.WriteString(": " + v.Type)
		}
		b.WriteString(" = " + extractVal(v.Note) + "\n")
	}
	// 全局变量
	for i := range p.Globals {
		if i == 0 && (len(p.Consts) > 0 || len(p.Imports) > 0) {
			b.WriteString("\n")
		}
		v := &p.Globals[i]
		if v.Note != "" {
			b.WriteString("# " + v.Note + "\n")
		}
		b.WriteString(v.Name)
		if v.Type != "" {
			b.WriteString(": " + v.Type)
		}
		b.WriteString(" = None\n")
	}
	// 类型定义（Python classes）
	for i := range p.Types {
		td := &p.Types[i]
		if i > 0 || len(p.Globals) > 0 || len(p.Consts) > 0 || len(p.Imports) > 0 {
			b.WriteString("\n")
		}
		if td.Note != "" {
			b.WriteString("# " + td.Note + "\n")
		}
		b.WriteString("class " + td.Name)
		if td.TypeExpr != "" {
			b.WriteString("(" + td.TypeExpr + ")")
		}
		b.WriteString(":\n")
		// 查找这个类型对应的方法（通过 Recv 匹配）
		hasMethods := false
		for si := range p.Subs {
			sub := &p.Subs[si]
			if sub.Recv != "" {
				hasMethods = true
				b.WriteString("\tdef " + sub.Name + "(")
				// 参数：self + 其他参数
				params := []string{}
				if !hasSelf(sub.Params) {
					params = append(params, "self")
				}
				for _, pr := range sub.Params {
					params = append(params, pr.Name)
				}
				b.WriteString(strings.Join(params, ", ") + "):\n")
				if sub.Body != "" {
					for _, ln := range strings.Split(sub.Body, "\n") {
						if ln == "" {
							b.WriteString("\n")
						} else {
							b.WriteString("\t\t" + ln + "\n")
						}
					}
				} else {
					b.WriteString("\t\tpass\n")
				}
			}
		}
		if !hasMethods {
			// 有字段 → __init__
			if len(td.Fields) > 0 {
				b.WriteString("\tdef __init__(self")
				for _, f := range td.Fields {
					b.WriteString(", " + f.Name)
				}
				b.WriteString("):\n")
				for _, f := range td.Fields {
					b.WriteString("\t\tself." + f.Name + " = " + f.Name + "\n")
				}
			} else {
				b.WriteString("\tpass\n")
			}
		}
	}
	// 子程序（顶级函数）
	for i := range p.Subs {
		sub := &p.Subs[i]
		if sub.Recv != "" && !isStandalone(sub) {
			continue // 已作为方法输出在 class 中
		}
		if i > 0 || len(p.Types) > 0 || len(p.Globals) > 0 || len(p.Consts) > 0 || len(p.Imports) > 0 {
			b.WriteString("\n")
		}
		if sub.Note != "" {
			b.WriteString("# " + sub.Note + "\n")
		}
		b.WriteString("def " + sub.Name + "(")
		params := make([]string, 0, len(sub.Params))
		for _, pr := range sub.Params {
			params = append(params, pr.Name)
		}
		b.WriteString(strings.Join(params, ", ") + "):\n")
		if sub.Body != "" {
			for _, ln := range strings.Split(sub.Body, "\n") {
				if ln == "" {
					b.WriteString("\n")
				} else {
					b.WriteString("\t" + ln + "\n")
				}
			}
		} else {
			b.WriteString("\tpass\n")
		}
	}
	return b.String()
}

// countIndent 计算缩进级别（空格数）。
func countIndent(s string) int {
	n := 0
	for _, ch := range s {
		if ch == ' ' {
			n++
		} else if ch == '\t' {
			n += 4
		} else {
			break
		}
	}
	return n
}

// extractVal 从注释中提取值（用于常量生成时的回退）。
func extractVal(note string) string {
	if strings.Contains(note, " | ") {
		parts := strings.SplitN(note, " | ", 2)
		return parts[0]
	}
	return "..."
}

func hasSelf(params []SEVar) bool {
	for _, p := range params {
		if p.Name == "self" || p.Name == "cls" {
			return true
		}
	}
	return false
}

func isStandalone(sub *SESub) bool {
	// 如果子程序在顶层且没有明确的接收器关联类型，是顶级函数
	// Python 中所有方法都有 self/cls 或 Recv 标记
	return sub.Recv == ""
}

func init() {
	RegisterProvider(pyLangProvider{}, "python", "py")
}
