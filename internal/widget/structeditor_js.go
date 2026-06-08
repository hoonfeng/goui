package widget

import (
	"regexp"
	"strings"
)

// ─── JavaScript LanguageProvider ───────────────────────────────
// 用正则解析 JS/TS/JSX/TSX 源码：顶级函数、类、const/let/var。

type jsLangProvider struct{}

func (jsLangProvider) Name() string      { return "js" }
func (jsLangProvider) Schema() *SESchema { return DefaultSchema() }

// jsImportRe 匹配 import / require
var jsImportRe = regexp.MustCompile(`^(import\s+|const\s+\w+\s*=\s*require\s*\(|var\s+\w+\s*=\s*require\s*\()`)

// jsFuncRe 匹配 function 声明：function foo(a, b) 或 async function foo()
var jsFuncRe = regexp.MustCompile(`^(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)`)

// jsArrowRe 匹配 const/let foo = (...) => ...
var jsArrowRe = regexp.MustCompile(`^(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s*)?\(([^)]*)\)\s*=>`)

// jsClassRe 匹配 class 定义
var jsClassRe = regexp.MustCompile(`^class\s+(\w+)(?:\s+extends\s+(\w+))?(?:\s+implements\s+(\w+))?`)

// jsMethodRe 匹配类中的方法定义（缩进后）
var jsMethodRe = regexp.MustCompile(`^\s*(?:async\s+)?(\w+)\s*\(([^)]*)\)\s*{`)

// JS 注释
var jsCommentRe = regexp.MustCompile(`^\s*//`)

type tsLangProvider struct{ jsLangProvider }

func (tsLangProvider) Name() string { return "ts" }

// tsInterfaceRe 匹配 interface 定义
var tsInterfaceRe = regexp.MustCompile(`^interface\s+(\w+)(?:\s+extends\s+(\w+))?`)

// tsTypeRe 匹配 type 定义
var tsTypeRe = regexp.MustCompile(`^type\s+(\w+)\s*=`)

func (jsLangProvider) Parse(src string) (*SEProgram, error) {
	return parseJSLike(src, false)
}

func (tsLangProvider) Parse(src string) (*SEProgram, error) {
	return parseJSLike(src, true)
}

func parseJSLike(src string, isTS bool) (*SEProgram, error) {
	p := &SEProgram{}
	lines := strings.Split(src, "\n")
	lastComment := ""
	inClass := false
	braceDepth := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 跳过空行/注释
		if trimmed == "" || jsCommentRe.MatchString(trimmed) {
			if strings.HasPrefix(trimmed, "//") {
				lastComment = strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
			}
			continue
		}

		// import
		if strings.HasPrefix(trimmed, "import ") {
			p.Imports = append(p.Imports, trimmed)
			lastComment = ""
			continue
		}

		// TS: interface
		if isTS {
			if m := tsInterfaceRe.FindStringSubmatch(trimmed); m != nil {
				td := SEType{
					Name: m[1], Kind: SETypeInterface, Note: lastComment,
				}
				if m[2] != "" {
					td.TypeExpr = "extends " + m[2]
				}
				p.Types = append(p.Types, td)
				inClass = true
				lastComment = ""
				continue
			}
			// TS: type alias
			if m := tsTypeRe.FindStringSubmatch(trimmed); m != nil {
				td := SEType{
					Name: m[1], Kind: SETypeAlias, Note: lastComment,
				}
				td.TypeExpr = strings.TrimSpace(trimmed[strings.Index(trimmed, "=")+1:])
				p.Types = append(p.Types, td)
				lastComment = ""
				continue
			}
		}

		// class
		if m := jsClassRe.FindStringSubmatch(trimmed); m != nil {
			td := SEType{
				Name: m[1], Kind: SETypeStruct, Note: lastComment,
			}
			if m[2] != "" {
				td.TypeExpr = "extends " + m[2]
			}
			p.Types = append(p.Types, td)
			inClass = true
			braceDepth = 1 // 跳过 body 行
			lastComment = ""
			continue
		}

		// function 声明
		if m := jsFuncRe.FindStringSubmatch(trimmed); m != nil {
			sub := SESub{
				Name: m[1], Note: lastComment,
				Params: parseJSParams(m[2]),
			}
			p.Subs = append(p.Subs, sub)
			lastComment = ""
			continue
		}

		// arrow function: const foo = (...) => ...
		if m := jsArrowRe.FindStringSubmatch(trimmed); m != nil {
			sub := SESub{
				Name: m[1], Note: lastComment,
				Params: parseJSParams(m[2]),
			}
			p.Subs = append(p.Subs, sub)
			lastComment = ""
			continue
		}

		// const/let/var 声明（非箭头函数）
		if strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "let ") || strings.HasPrefix(trimmed, "var ") {
			// 排除箭头函数
			if !jsArrowRe.MatchString(trimmed) {
				name := extractJSVarName(trimmed)
				if name != "" {
					sv := SEVar{Name: name, Note: lastComment}
					p.Globals = append(p.Globals, sv)
				}
			}
			lastComment = ""
			continue
		}

		lastComment = ""
	}
	return p, nil
}

// parseJSParams 解析 JS 参数列表
func parseJSParams(s string) []SEVar {
	parts := strings.Split(s, ",")
	var out []SEVar
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// 支持 "name: type" 和 "name = default"
		name := strings.SplitN(p, ":", 2)[0]
		name = strings.TrimSpace(strings.SplitN(name, "=", 2)[0])
		name = strings.TrimSpace(strings.SplitN(name, "?", 2)[0])
		if name != "" {
			out = append(out, SEVar{Name: name})
		}
	}
	return out
}

func extractJSVarName(s string) string {
	// const {a, b} = ... / const [a, b] = ...
	if strings.Contains(s, "{") || strings.Contains(s, "[") {
		return ""
	}
	parts := strings.Fields(s)
	if len(parts) >= 2 {
		return strings.TrimRight(parts[1], ";")
	}
	return ""
}

func (jsLangProvider) Generate(p *SEProgram) string {
	return generateJSLike(p, false)
}

func (tsLangProvider) Generate(p *SEProgram) string {
	return generateJSLike(p, true)
}

func generateJSLike(p *SEProgram, isTS bool) string {
	var b strings.Builder
	for i := range p.Imports {
		b.WriteString(p.Imports[i] + "\n")
	}
	// 全局变量
	for i := range p.Globals {
		v := &p.Globals[i]
		if i == 0 && len(p.Imports) > 0 {
			b.WriteString("\n")
		}
		if v.Note != "" {
			b.WriteString("// " + v.Note + "\n")
		}
		b.WriteString("let " + v.Name + ";\n")
	}
	// 类型定义
	for i := range p.Types {
		td := &p.Types[i]
		b.WriteString("\n")
		if td.Note != "" {
			b.WriteString("// " + td.Note + "\n")
		}
		if isTS {
			switch td.Kind {
			case SETypeInterface:
				b.WriteString("interface " + td.Name)
				if td.TypeExpr != "" && strings.HasPrefix(td.TypeExpr, "extends") {
					b.WriteString(" " + td.TypeExpr)
				}
				b.WriteString(" {\n")
				for _, m := range td.Methods {
					b.WriteString("\t" + m.Name + "(" + m.Type + ");\n")
				}
				b.WriteString("}\n")
				continue
			case SETypeAlias:
				b.WriteString("type " + td.Name + " = " + td.TypeExpr + ";\n")
				continue
			}
		}
		// class
		b.WriteString("class " + td.Name)
		if td.TypeExpr != "" {
			b.WriteString(" " + td.TypeExpr)
		}
		b.WriteString(" {\n")
		b.WriteString("\tconstructor() {\n")
		for _, f := range td.Fields {
			b.WriteString("\t\tthis." + f.Name + " = " + f.Type + ";\n")
		}
		b.WriteString("\t}\n")
		b.WriteString("}\n")
	}
	// 函数
	for i := range p.Subs {
		sub := &p.Subs[i]
		b.WriteString("\n")
		if sub.Note != "" {
			b.WriteString("// " + sub.Note + "\n")
		}
		b.WriteString("function " + sub.Name + "(")
		params := make([]string, 0, len(sub.Params))
		for _, pr := range sub.Params {
			params = append(params, pr.Name)
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
	RegisterProvider(jsLangProvider{}, "javascript", "js", "jsx", "mjs", "cjs")
	RegisterProvider(tsLangProvider{}, "typescript", "ts", "tsx")
}
