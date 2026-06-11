package widget

import (
	"regexp"
	"strings"
)

// ─── Java LanguageProvider ─────────────────────────────────────
// 用正则解析 Java 源码：类、接口、方法、字段。

type javaLangProvider struct{}

func (javaLangProvider) Name() string      { return "java" }
func (javaLangProvider) Schema() *SESchema { return DefaultSchema() }

// javaImportRe 匹配 import 语句
var javaImportRe = regexp.MustCompile(`^import\s+(?:static\s+)?(\S+)\s*;`)

// javaClassRe 匹配 class 定义
var javaClassRe = regexp.MustCompile(`^(?:public\s+|private\s+|protected\s+)?(?:abstract\s+|final\s+)?(?:strictfp\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?(?:\s+implements\s+([^{]+))?`)

// javaInterfaceRe 匹配 interface 定义
var javaInterfaceRe = regexp.MustCompile(`^(?:public\s+|private\s+|protected\s+)?interface\s+(\w+)(?:\s+extends\s+(\w+))?`)

// javaMethodRe 匹配方法声明（放在类内）
var javaMethodRe = regexp.MustCompile(`^\s*(?:public|private|protected|static|final|abstract|synchronized|native)\s+(?:\S+\s+)*(\w+)\s*\(([^)]*)\)\s*(?:throws\s+\S+)?\s*[{;]`)

// javaFieldRe 匹配字段声明
var javaFieldRe = regexp.MustCompile(`^\s*(?:public|private|protected|static|final|volatile|transient)\s+(\S+)\s+(\w+)\s*(?:=\s*[^;]+)?\s*;`)

func (javaLangProvider) Parse(src string) (*SEProgram, error) {
	p := &SEProgram{}
	lines := strings.Split(src, "\n")
	lastComment := ""
	inClass := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			lastComment = strings.TrimPrefix(trimmed, "//")
			lastComment = strings.TrimPrefix(lastComment, "/*")
			lastComment = strings.TrimSuffix(lastComment, "*/")
			lastComment = strings.TrimSpace(lastComment)
			if strings.Contains(trimmed, "/*") && !strings.Contains(trimmed, "*/") {
				// 多行块注释，跳过直到 */
				for i++; i < len(lines); i++ {
					if strings.Contains(lines[i], "*/") {
						break
					}
				}
			}
			continue
		}

		// import
		if m := javaImportRe.FindStringSubmatch(trimmed); m != nil {
			p.Imports = append(p.Imports, m[1])
			lastComment = ""
			continue
		}

		// class
		if m := javaClassRe.FindStringSubmatch(trimmed); m != nil {
			td := SEType{Name: m[1], Kind: SETypeStruct, Note: lastComment}
			if m[2] != "" {
				td.TypeExpr = "extends " + m[2]
			}
			if m[3] != "" {
				if td.TypeExpr != "" {
					td.TypeExpr += " implements " + strings.TrimSpace(m[3])
				} else {
					td.TypeExpr = "implements " + strings.TrimSpace(m[3])
				}
			}
			p.Types = append(p.Types, td)
			inClass = true
			lastComment = ""
			continue
		}

		// interface
		if m := javaInterfaceRe.FindStringSubmatch(trimmed); m != nil {
			td := SEType{Name: m[1], Kind: SETypeInterface, Note: lastComment}
			if m[2] != "" {
				td.TypeExpr = "extends " + m[2]
			}
			p.Types = append(p.Types, td)
			inClass = true
			lastComment = ""
			continue
		}

		// 类内方法
		if inClass {
			if m := javaMethodRe.FindStringSubmatch(trimmed); m != nil {
				sub := SESub{
					Name:   m[1],
					Params: parseJavaParams(m[2]),
					Note:   lastComment,
				}
				p.Subs = append(p.Subs, sub)
				lastComment = ""
				continue
			}
			// 类内字段
			if m := javaFieldRe.FindStringSubmatch(trimmed); m != nil {
				sv := SEVar{
					Name: m[2],
					Type: m[1],
					Note: lastComment,
				}
				// 添加到最后一个类型的字段中
				if len(p.Types) > 0 {
					td := &p.Types[len(p.Types)-1]
					td.Fields = append(td.Fields, sv)
				}
				lastComment = ""
				continue
			}
		}

		// 检查大括号深度变化
		braceCount := strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		if braceCount < 0 && inClass {
			// 可能出了类
		}
		lastComment = ""
	}
	return p, nil
}

func parseJavaParams(s string) []SEVar {
	parts := strings.Split(s, ",")
	var out []SEVar
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "..." {
			continue
		}
		// "Type name" or "Type... name" or "name"
		fields := strings.Fields(p)
		if len(fields) >= 2 {
			// last field is name
			name := fields[len(fields)-1]
			if strings.HasPrefix(name, "//") {
				continue
			}
			typ := strings.Join(fields[:len(fields)-1], " ")
			out = append(out, SEVar{Name: name, Type: typ})
		} else if len(fields) == 1 {
			out = append(out, SEVar{Name: fields[0]})
		}
	}
	return out
}

func (javaLangProvider) Generate(p *SEProgram) string {
	var b strings.Builder
	hasPackage := false
	b.WriteString("// Generated from structured editor\n")
	for i := range p.Imports {
		if !hasPackage && i == 0 {
			b.WriteString("package com.example;\n\n")
			hasPackage = true
		}
		b.WriteString("import " + p.Imports[i] + ";\n")
	}
	// 类型定义（class/interface）
	for i := range p.Types {
		td := &p.Types[i]
		b.WriteString("\n")
		if td.Note != "" {
			b.WriteString("// " + td.Note + "\n")
		}
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
		default:
			b.WriteString("class " + td.Name)
			if td.TypeExpr != "" {
				b.WriteString(" " + td.TypeExpr)
			}
			b.WriteString(" {\n")
			for _, f := range td.Fields {
				b.WriteString("\t" + f.Type + " " + f.Name + ";\n")
			}
			// 查找该类的方法
			b.WriteString("}\n")
		}
	}
	return b.String()
}

// ─── Rust LanguageProvider ─────────────────────────────────────
// 用正则解析 Rust 源码：fn、struct、enum、impl、trait。

type rustLangProvider struct{}

func (rustLangProvider) Name() string      { return "rust" }
func (rustLangProvider) Schema() *SESchema { return DefaultSchema() }

// rustFnRe 匹配 fn 声明
var rustFnRe = regexp.MustCompile(`^(?:pub\s+)?(?:unsafe\s+)?(?:async\s+)?fn\s+(\w+)(?:\s*<([^>]+)>)?\s*\(([^)]*)\)\s*(?:->\s*(\S+))?`)

// rustStructRe 匹配 struct 定义
var rustStructRe = regexp.MustCompile(`^(?:pub\s+)?struct\s+(\w+)(?:\s*<([^>]+)>)?`)

// rustEnumRe 匹配 enum 定义
var rustEnumRe = regexp.MustCompile(`^(?:pub\s+)?enum\s+(\w+)(?:\s*<([^>]+)>)?`)

// rustTraitRe 匹配 trait 定义
var rustTraitRe = regexp.MustCompile(`^(?:pub\s+)?trait\s+(\w+)(?:\s*<([^>]+)>)?`)

// rustImplRe 匹配 impl 块
var rustImplRe = regexp.MustCompile(`^(?:unsafe\s+)?impl(?:\s*<([^>]+)>)?\s+(\w+(?:\s*<[^>]+>)?)\s*(?:for\s+(\w+(?:\s*<[^>]+>)?))?`)

// rustUseRe 匹配 use 导入
var rustUseRe = regexp.MustCompile(`^use\s+(\S+)\s*;`)

func (rustLangProvider) Parse(src string) (*SEProgram, error) {
	p := &SEProgram{}
	lines := strings.Split(src, "\n")
	lastComment := ""
	inImpl := ""
	braceDepth := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			lastComment = strings.TrimPrefix(trimmed, "/*")
			lastComment = strings.TrimSuffix(lastComment, "*/")
			lastComment = strings.TrimSpace(lastComment)
			continue
		}

		// impl 块内：把 fn 当作目标类型的方法提取（Recv=类型名），按花括号深度判断块结束。
		if inImpl != "" {
			if m := rustFnRe.FindStringSubmatch(trimmed); m != nil {
				sub := SESub{
					Name:   m[1],
					Recv:   inImpl, // 方法接收器 = impl 的目标类型
					Note:   lastComment,
					Params: parseRustParams(m[3], m[2]),
				}
				if len(m) > 4 && m[4] != "" {
					sub.Returns = []SEVar{{Type: m[4]}}
				}
				p.Subs = append(p.Subs, sub)
				lastComment = ""
			}
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth <= 0 { // impl 块闭合，退出
				inImpl, braceDepth = "", 0
			}
			continue
		}

		// use
		if m := rustUseRe.FindStringSubmatch(trimmed); m != nil {
			p.Imports = append(p.Imports, m[1])
			lastComment = ""
			continue
		}

		// struct
		if m := rustStructRe.FindStringSubmatch(trimmed); m != nil {
			td := SEType{Name: m[1], Kind: SETypeStruct, Note: lastComment}
			if m[2] != "" {
				td.TypeParams = parseGenericParams(m[2])
			}
			// 读取 struct body
			bodyStart := strings.Index(line, "{")
			if bodyStart < 0 {
				// 后续行找 {
				for j := i + 1; j < len(lines); j++ {
					if strings.Contains(lines[j], "{") {
						bodyStart = j
						break
					}
				}
			}
			p.Types = append(p.Types, td)
			lastComment = ""
			continue
		}

		// enum
		if m := rustEnumRe.FindStringSubmatch(trimmed); m != nil {
			td := SEType{Name: m[1], Kind: SETypeInterface, Note: lastComment}
			if m[2] != "" {
				td.TypeParams = parseGenericParams(m[2])
			}
			p.Types = append(p.Types, td)
			lastComment = ""
			continue
		}

		// trait
		if m := rustTraitRe.FindStringSubmatch(trimmed); m != nil {
			td := SEType{Name: m[1], Kind: SETypeInterface, Note: lastComment}
			if m[2] != "" {
				td.TypeParams = parseGenericParams(m[2])
			}
			p.Types = append(p.Types, td)
			lastComment = ""
			continue
		}

		// fn（顶级函数）
		if m := rustFnRe.FindStringSubmatch(trimmed); m != nil && inImpl == "" {
			ret := ""
			if len(m) > 4 && m[4] != "" {
				ret = m[4]
			}
			sub := SESub{
				Name: m[1], Note: lastComment,
				Params: parseRustParams(m[3], m[2]),
			}
			if ret != "" {
				sub.Returns = []SEVar{{Type: ret}}
			}
			p.Subs = append(p.Subs, sub)
			lastComment = ""
			continue
		}

		// impl
		if m := rustImplRe.FindStringSubmatch(trimmed); m != nil {
			targetType := m[2]
			if m[3] != "" {
				targetType = m[3] // impl Trait for Type
			}
			inImpl = targetType
			braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
			lastComment = ""
			continue
		}

		lastComment = ""
	}
	return p, nil
}

func parseRustParams(s string, generics string) []SEVar {
	parts := strings.Split(s, ",")
	var out []SEVar
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "self" || p == "&self" || p == "&mut self" {
			continue
		}
		// "name: Type"
		pp := strings.SplitN(p, ":", 2)
		name := strings.TrimSpace(pp[0])
		typ := ""
		if len(pp) > 1 {
			typ = strings.TrimSpace(pp[1])
		}
		out = append(out, SEVar{Name: name, Type: typ})
	}
	return out
}

func parseGenericParams(s string) []SEVar {
	parts := strings.Split(s, ",")
	var out []SEVar
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, SEVar{Name: p})
	}
	return out
}

func (rustLangProvider) Generate(p *SEProgram) string {
	var b strings.Builder
	b.WriteString("// Generated from structured editor\n")
	for i := range p.Imports {
		b.WriteString("use " + p.Imports[i] + ";\n")
	}
	// 类型
	for i := range p.Types {
		td := &p.Types[i]
		b.WriteString("\n")
		if td.Note != "" {
			b.WriteString("// " + td.Note + "\n")
		}
		switch td.Kind {
		case SETypeStruct:
			b.WriteString("struct " + td.Name)
			if len(td.TypeParams) > 0 {
				b.WriteString(typeParamsGo(td.TypeParams))
			}
			b.WriteString(" {\n")
			for _, f := range td.Fields {
				b.WriteString("\t" + f.Name + ": " + f.Type + ",\n")
			}
			b.WriteString("}\n")
		case SETypeInterface:
			b.WriteString("trait " + td.Name)
			if len(td.TypeParams) > 0 {
				b.WriteString(typeParamsGo(td.TypeParams))
			}
			b.WriteString(" {\n")
			for _, m := range td.Methods {
				b.WriteString("\tfn " + m.Name + "(" + m.Type + ");\n")
			}
			b.WriteString("}\n")
		}
	}
	// 函数
	for i := range p.Subs {
		sub := &p.Subs[i]
		b.WriteString("\n")
		if sub.Note != "" {
			b.WriteString("// " + sub.Note + "\n")
		}
		b.WriteString("fn " + sub.Name + "(")
		params := make([]string, 0, len(sub.Params))
		for _, pr := range sub.Params {
			s := pr.Name
			if pr.Type != "" {
				s += ": " + pr.Type
			}
			params = append(params, s)
		}
		b.WriteString(strings.Join(params, ", ") + ")")
		if len(sub.Returns) > 0 {
			b.WriteString(" -> " + sub.Returns[0].Type)
		}
		b.WriteString(" {\n")
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
	RegisterProvider(javaLangProvider{}, "java")
	RegisterProvider(rustLangProvider{}, "rust", "rs")
}
