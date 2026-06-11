package widget

import (
	"regexp"
	"strings"
)

// ─── C/C++ LanguageProvider ───────────────────────────────────
// 用正则解析 C/C++ 源码：函数、类、结构体、全局变量。

type cLangProvider struct{}

func (cLangProvider) Name() string      { return "c" }
func (cLangProvider) Schema() *SESchema { return DefaultSchema() }

// cppIncludeRe 匹配 #include / #import
var cppIncludeRe = regexp.MustCompile(`^\s*#\s*(?:include|import)\s+[<"]`)

// cppFuncRe 匹配函数签名（返回值 函数名(参数) { 或 ;）
// 支持 C 风格和 C++ 风格：int foo(int a, float b)
var cppFuncRe = regexp.MustCompile(`^(?:static|inline|virtual|explicit|friend|const|volatile|unsigned|signed|extern|virtual|override|final|\w+::)*\s*(\w+(?:\s*\*+)?)\s+(\w+)\s*\(([^)]*)\)\s*(?:const|override|final|\s*=\s*0|\s*=\s*default|\s*=\s*delete)?\s*[{;]`)

// cppStructRe 匹配 struct/class 定义
var cppStructRe = regexp.MustCompile(`^(?:typedef\s+)?(?:struct|class|union)\s+(\w+)\s*(?::\s*(?:public|private|protected)\s+(\w+))?`)

// cppGlobalRe 匹配全局变量声明（简单版）
var cppGlobalRe = regexp.MustCompile(`^(?:(?:extern|static|const|volatile|unsigned|signed)\s+)*(\w+(?:\s*\*+)?)\s+(\w+)\s*(?:=|;)`)

func (cLangProvider) Parse(src string) (*SEProgram, error) {
	p := &SEProgram{}
	lines := strings.Split(src, "\n")
	lastComment := ""
	inType := "" // 当前所在的 struct/class 名

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}
		// 注释
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
			if strings.HasPrefix(trimmed, "/*") {
				// 块注释
				lastComment = strings.TrimPrefix(trimmed, "/*")
				lastComment = strings.TrimSuffix(lastComment, "*/")
				lastComment = strings.TrimSpace(lastComment)
				if !strings.Contains(trimmed, "*/") {
					for i++; i < len(lines); i++ {
						if strings.Contains(lines[i], "*/") {
							break
						}
					}
				}
			}
			continue
		}

		// #include
		if cppIncludeRe.MatchString(trimmed) {
			p.Imports = append(p.Imports, trimmed)
			lastComment = ""
			continue
		}

		// struct/class/union
		if m := cppStructRe.FindStringSubmatch(trimmed); m != nil {
			td := SEType{Name: m[1], Kind: SETypeStruct, Note: lastComment}
			if len(m) > 2 && m[2] != "" {
				td.TypeExpr = "inherits " + m[2]
			}
			p.Types = append(p.Types, td)
			inType = m[1]
			lastComment = ""
			continue
		}

		// 函数定义（不在类型内部时作为子程序）
		if inType == "" {
			if m := cppFuncRe.FindStringSubmatch(trimmed); m != nil {
				sub := SESub{
					Name:   m[2],
					Note:   lastComment,
					Params: parseCppParams(m[3]),
				}
				// 返回值
				retType := strings.TrimSpace(m[1])
				if retType != "" {
					sub.Returns = []SEVar{{Type: retType}}
				}
				p.Subs = append(p.Subs, sub)
				lastComment = ""
				continue
			}
		}

		// 类内方法/构造函数
		if inType != "" {
			if m := cppFuncRe.FindStringSubmatch(trimmed); m != nil {
				sub := SESub{
					Name:   m[2],
					Note:   lastComment,
					Params: parseCppParams(m[3]),
					Recv:   "(" + inType + "*)",
				}
				retType := strings.TrimSpace(m[1])
				if retType != "" {
					sub.Returns = []SEVar{{Type: retType}}
				}
				p.Subs = append(p.Subs, sub)
				lastComment = ""
				continue
			}
			// 类内字段
			if m := cppGlobalRe.FindStringSubmatch(trimmed); m != nil {
				sv := SEVar{Name: m[2], Type: m[1], Note: lastComment}
				if len(p.Types) > 0 {
					p.Types[len(p.Types)-1].Fields = append(p.Types[len(p.Types)-1].Fields, sv)
				}
				lastComment = ""
				continue
			}
		}

		// 全局变量
		if inType == "" {
			if m := cppGlobalRe.FindStringSubmatch(trimmed); m != nil {
				p.Globals = append(p.Globals, SEVar{Name: m[2], Type: m[1], Note: lastComment})
				lastComment = ""
				continue
			}
		}

		// 检查大括号深度变化
		bc := strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		if bc < 0 && inType != "" {
			inType = ""
		}
		lastComment = ""
	}
	return p, nil
}

func parseCppParams(s string) []SEVar {
	parts := strings.Split(s, ",")
	var out []SEVar
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "void" || p == "..." {
			continue
		}
		// "Type name" or "Type* name" or "Type& name"
		// Handle default values: "Type name = default"
		p = strings.SplitN(p, "=", 2)[0]
		p = strings.TrimSpace(p)
		fields := strings.Fields(p)
		if len(fields) >= 2 {
			name := fields[len(fields)-1]
			typ := strings.Join(fields[:len(fields)-1], " ")
			out = append(out, SEVar{Name: name, Type: typ})
		} else if len(fields) == 1 {
			out = append(out, SEVar{Name: fields[0]})
		}
	}
	return out
}

func (cLangProvider) Generate(p *SEProgram) string {
	var b strings.Builder
	b.WriteString("// Generated from structured editor\n\n")
	// includes
	for i := range p.Imports {
		b.WriteString(p.Imports[i] + "\n")
	}
	// globals
	for i := range p.Globals {
		if i == 0 && len(p.Imports) > 0 {
			b.WriteString("\n")
		}
		v := &p.Globals[i]
		if v.Note != "" {
			b.WriteString("// " + v.Note + "\n")
		}
		b.WriteString(v.Type + " " + v.Name + ";\n")
	}
	// 类型定义
	for i := range p.Types {
		td := &p.Types[i]
		b.WriteString("\n")
		if td.Note != "" {
			b.WriteString("// " + td.Note + "\n")
		}
		b.WriteString("struct " + td.Name)
		if td.TypeExpr != "" && strings.Contains(td.TypeExpr, "inherits") {
			base := strings.TrimPrefix(td.TypeExpr, "inherits ")
			b.WriteString(" : public " + base)
		}
		b.WriteString(" {\n")
		for _, f := range td.Fields {
			b.WriteString("\t" + f.Type + " " + f.Name + ";\n")
		}
		b.WriteString("};\n")
	}
	// 函数
	for i := range p.Subs {
		sub := &p.Subs[i]
		b.WriteString("\n")
		if sub.Note != "" {
			b.WriteString("// " + sub.Note + "\n")
		}
		ret := "void"
		if len(sub.Returns) > 0 {
			ret = sub.Returns[0].Type
		}
		b.WriteString(ret + " " + sub.Name + "(")
		params := make([]string, 0, len(sub.Params))
		for _, pr := range sub.Params {
			s := pr.Type + " " + pr.Name
			params = append(params, strings.TrimSpace(s))
		}
		b.WriteString(strings.Join(params, ", ") + ") {\n")
		if sub.Body != "" {
			for _, ln := range strings.Split(sub.Body, "\n") {
				if ln == "" {
					b.WriteString("\n")
				} else {
					b.WriteString("\t" + ln + "\n")
				}
			}
		}
		b.WriteString("}\n")
	}
	return b.String()
}

func init() {
	RegisterProvider(cLangProvider{}, "c", "cpp", "c++", "h", "hpp")
}
