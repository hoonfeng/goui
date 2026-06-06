package widget

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

// ─── 语法高亮内核（数据驱动词法分析）──────────────────────────────
// 给定一行文本 + 进入该行时的跨行状态（块注释/原始字符串），产出该行的着色 token
// 与离开该行的状态。加新语言只需新增一份 ceLang 定义，tokenizer 通用。
// 这是 CodeEditor 第一批的高亮地基；后续接 LSP 做语义着色时，这套词法着色作为兜底。

type ceTokKind uint8

const (
	tkText    ceTokKind = iota // 普通文本/标识符
	tkKeyword                  // 关键字
	tkType                     // 类型/内置类型/常量
	tkString                   // 字符串/字符
	tkComment                  // 注释
	tkNumber                   // 数字字面量
	tkFunc                     // 函数名（标识符后紧跟 "("）
	tkPunct                    // 标点/运算符
)

// ceToken 行内一段着色区间 [start,end)（rune 列索引）。
type ceToken struct {
	start, end int
	kind       ceTokKind
}

// ceLineState 跨行词法状态。
type ceLineState uint8

const (
	stNormal ceLineState = iota
	stBlockComment // 处于 /* ... */ 中
	stRawString    // 处于 `...`（Go 原始字符串）中
	stFence        // 处于 Markdown ``` 围栏代码块中
)

// ceLang 一种语言的词法定义。
type ceLang struct {
	keywords    map[string]bool
	types       map[string]bool
	lineComment string // 行注释前缀，如 "//"、"#"
	blockStart  string // 块注释起始，如 "/*"（空=无）
	blockEnd    string // 块注释结束，如 "*/"
	quotes      string // 普通字符串引号字符集合，如 "\"'"
	rawQuote    byte   // 原始字符串引号（跨行），如 '`'；0=无
	// custom 专用分词器：结构化标记语言（Markdown/HTML 等）关键字模型不够用，
	// 给一份按语法切分的函数；nil 则走通用关键字模型分词。
	custom func([]rune, ceLineState) ([]ceToken, ceLineState)
}

func isIdentStart(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r >= 0x80
}
func isIdentPart(r rune) bool { return isIdentStart(r) || (r >= '0' && r <= '9') }
func isDigit(r rune) bool     { return r >= '0' && r <= '9' }

// tokenizeLine 对一行做词法分析。in 为进入该行时的状态，返回 tokens + 离开该行的状态。
func tokenizeLine(runes []rune, lang *ceLang, in ceLineState) ([]ceToken, ceLineState) {
	if lang.custom != nil { // 结构化标记语言走专用分词器
		return lang.custom(runes, in)
	}
	var toks []ceToken
	n := len(runes)
	i := 0

	// 续接跨行块注释
	if in == stBlockComment && lang.blockEnd != "" {
		end := []rune(lang.blockEnd)
		if j := indexRunes(runes, end, 0); j >= 0 {
			toks = append(toks, ceToken{0, j + len(end), tkComment})
			i = j + len(end)
			in = stNormal
		} else {
			if n > 0 {
				toks = append(toks, ceToken{0, n, tkComment})
			}
			return toks, stBlockComment
		}
	}
	// 续接跨行原始字符串
	if in == stRawString && lang.rawQuote != 0 {
		if j := indexRune(runes, rune(lang.rawQuote), 0); j >= 0 {
			toks = append(toks, ceToken{0, j + 1, tkString})
			i = j + 1
			in = stNormal
		} else {
			if n > 0 {
				toks = append(toks, ceToken{0, n, tkString})
			}
			return toks, stRawString
		}
	}

	for i < n {
		r := runes[i]

		// 行注释
		if lang.lineComment != "" && hasPrefixRunes(runes, []rune(lang.lineComment), i) {
			toks = append(toks, ceToken{i, n, tkComment})
			i = n
			break
		}
		// 块注释
		if lang.blockStart != "" && hasPrefixRunes(runes, []rune(lang.blockStart), i) {
			bs := []rune(lang.blockStart)
			be := []rune(lang.blockEnd)
			if j := indexRunes(runes, be, i+len(bs)); j >= 0 {
				toks = append(toks, ceToken{i, j + len(be), tkComment})
				i = j + len(be)
			} else {
				toks = append(toks, ceToken{i, n, tkComment})
				return toks, stBlockComment
			}
			continue
		}
		// 原始字符串（跨行）
		if lang.rawQuote != 0 && r == rune(lang.rawQuote) {
			if j := indexRune(runes, rune(lang.rawQuote), i+1); j >= 0 {
				toks = append(toks, ceToken{i, j + 1, tkString})
				i = j + 1
			} else {
				toks = append(toks, ceToken{i, n, tkString})
				return toks, stRawString
			}
			continue
		}
		// 普通字符串/字符（行内闭合，处理转义）
		if strings.ContainsRune(lang.quotes, r) {
			j := i + 1
			for j < n {
				if runes[j] == '\\' { // 转义
					j += 2
					continue
				}
				if runes[j] == r {
					j++
					break
				}
				j++
			}
			if j > n {
				j = n
			}
			toks = append(toks, ceToken{i, j, tkString})
			i = j
			continue
		}
		// 数字
		if isDigit(r) || (r == '.' && i+1 < n && isDigit(runes[i+1])) {
			j := i + 1
			for j < n && (isIdentPart(runes[j]) || runes[j] == '.' || runes[j] == 'x' || runes[j] == 'X') {
				j++
			}
			toks = append(toks, ceToken{i, j, tkNumber})
			i = j
			continue
		}
		// 标识符 / 关键字 / 类型 / 函数
		if isIdentStart(r) {
			j := i + 1
			for j < n && isIdentPart(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			kind := tkText
			switch {
			case lang.keywords[word]:
				kind = tkKeyword
			case lang.types[word]:
				kind = tkType
			default:
				// 标识符后紧跟 '(' 视为函数名
				k := j
				for k < n && runes[k] == ' ' {
					k++
				}
				if k < n && runes[k] == '(' {
					kind = tkFunc
				}
			}
			toks = append(toks, ceToken{i, j, kind})
			i = j
			continue
		}
		// 标点/运算符
		if isPunct(r) {
			toks = append(toks, ceToken{i, i + 1, tkPunct})
			i++
			continue
		}
		// 其他（空白等）当普通文本，合并连续
		j := i + 1
		for j < n && !isIdentStart(runes[j]) && !isDigit(runes[j]) && !isPunct(runes[j]) &&
			!strings.ContainsRune(lang.quotes, runes[j]) {
			j++
		}
		i = j
	}
	return toks, stNormal
}

func isPunct(r rune) bool {
	return strings.ContainsRune("+-*/%=<>!&|^~?:;,.()[]{}", r)
}

// ── rune 切片小工具 ──
func hasPrefixRunes(s, prefix []rune, at int) bool {
	if at+len(prefix) > len(s) {
		return false
	}
	for k := range prefix {
		if s[at+k] != prefix[k] {
			return false
		}
	}
	return true
}
func indexRunes(s, sub []rune, from int) int {
	if len(sub) == 0 {
		return from
	}
	for i := from; i+len(sub) <= len(s); i++ {
		if hasPrefixRunes(s, sub, i) {
			return i
		}
	}
	return -1
}
func indexRune(s []rune, r rune, from int) int {
	for i := from; i < len(s); i++ {
		if s[i] == r {
			return i
		}
	}
	return -1
}

// ── 语言定义表 ──

func words(list ...string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, w := range list {
		m[w] = true
	}
	return m
}

var ceLangGo = &ceLang{
	keywords: words("break", "case", "chan", "const", "continue", "default", "defer", "else",
		"fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map",
		"package", "range", "return", "select", "struct", "switch", "type", "var"),
	types: words("bool", "string", "int", "int8", "int16", "int32", "int64", "uint", "uint8",
		"uint16", "uint32", "uint64", "uintptr", "byte", "rune", "float32", "float64",
		"complex64", "complex128", "error", "any", "true", "false", "nil", "iota",
		"make", "len", "cap", "new", "append", "copy", "delete", "close", "panic", "recover"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'", rawQuote: '`',
}

// ceLangEY 易语言（中文关键字）——用于 StructEditor 内嵌逻辑代码的高亮。
// 中文字符 isIdentStart 视为标识符(r>=0x80)，故中文命令能整词匹配关键字。
var ceLangEY = &ceLang{
	keywords: words("如果", "如果真", "否则", "返回", "判断",
		"计次循环首", "计次循环尾", "判断循环首", "判断循环尾", "变量循环首", "变量循环尾",
		"循环", "到", "步长", "跳出循环", "结束", "子程序", "参数", "局部变量", "赋值"),
	types: words("整数型", "文本型", "逻辑型", "小数型", "字节型", "双精度小数型",
		"短整数型", "长整数型", "日期时间型", "字节集", "真", "假"),
	quotes: "\"",
}

var ceLangJSON = &ceLang{
	keywords: words("true", "false", "null"),
	types:    words(),
	quotes:   "\"",
}

var ceLangJS = &ceLang{
	keywords: words("break", "case", "catch", "class", "const", "continue", "debugger",
		"default", "delete", "do", "else", "export", "extends", "finally", "for", "function",
		"if", "import", "in", "instanceof", "new", "return", "super", "switch", "this", "throw",
		"try", "typeof", "var", "void", "while", "with", "yield", "let", "async", "await", "of"),
	types:       words("true", "false", "null", "undefined", "NaN", "Infinity", "console", "document", "window"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'`",
}

var ceLangPython = &ceLang{
	keywords: words("and", "as", "assert", "async", "await", "break", "class", "continue", "def",
		"del", "elif", "else", "except", "finally", "for", "from", "global", "if", "import", "in",
		"is", "lambda", "nonlocal", "not", "or", "pass", "raise", "return", "try", "while", "with", "yield"),
	types:       words("True", "False", "None", "self", "print", "len", "range", "int", "str", "float", "list", "dict", "set", "tuple"),
	lineComment: "#", quotes: "\"'",
}

var ceLangRust = &ceLang{
	keywords: words("as", "async", "await", "break", "const", "continue", "crate", "dyn", "else",
		"enum", "extern", "fn", "for", "if", "impl", "in", "let", "loop", "match", "mod", "move",
		"mut", "pub", "ref", "return", "self", "static", "struct", "super", "trait", "type", "unsafe",
		"use", "where", "while"),
	types: words("bool", "char", "str", "String", "i8", "i16", "i32", "i64", "i128", "isize",
		"u8", "u16", "u32", "u64", "u128", "usize", "f32", "f64", "Vec", "Option", "Result", "Box",
		"true", "false", "Some", "None", "Ok", "Err"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

var ceLangC = &ceLang{
	keywords: words("auto", "break", "case", "const", "continue", "default", "do", "else", "enum",
		"extern", "for", "goto", "if", "register", "return", "sizeof", "static", "struct", "switch",
		"typedef", "union", "volatile", "while", "class", "namespace", "template", "public", "private",
		"protected", "virtual", "new", "delete", "using", "include", "define"),
	types: words("void", "char", "short", "int", "long", "float", "double", "signed", "unsigned",
		"bool", "size_t", "wchar_t", "true", "false", "NULL", "nullptr", "string", "vector"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

var ceLangJava = &ceLang{
	keywords: words("abstract", "assert", "break", "case", "catch", "class", "continue", "default",
		"do", "else", "enum", "extends", "final", "finally", "for", "if", "implements", "import",
		"instanceof", "interface", "native", "new", "package", "private", "protected", "public",
		"return", "static", "super", "switch", "synchronized", "this", "throw", "throws", "try",
		"void", "volatile", "while"),
	types: words("boolean", "byte", "char", "short", "int", "long", "float", "double", "String",
		"Object", "Integer", "Boolean", "List", "Map", "true", "false", "null"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

// ── 前端 + 配置文件语言（标记/数据类用关键字+注释模型，为近似高亮）──

var ceLangCSS = &ceLang{
	keywords: words("color", "background", "background-color", "display", "position", "flex",
		"grid", "margin", "padding", "border", "width", "height", "font", "font-size", "font-weight",
		"top", "left", "right", "bottom", "align-items", "justify-content", "flex-direction", "gap",
		"opacity", "z-index", "overflow", "transition", "transform", "animation", "box-shadow"),
	types: words("none", "block", "inline", "flex", "grid", "absolute", "relative", "fixed", "auto",
		"inherit", "initial", "center", "left", "right", "bold", "solid", "hidden", "px", "em", "rem",
		"important", "rgba", "rgb", "hsl", "var", "calc"),
	blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

var ceLangSCSS = &ceLang{
	keywords:    ceLangCSS.keywords,
	types:       ceLangCSS.types,
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

var ceLangHTML = &ceLang{
	keywords: words("html", "head", "body", "div", "span", "a", "p", "ul", "ol", "li", "table",
		"tr", "td", "th", "form", "input", "button", "img", "script", "style", "link", "meta",
		"title", "h1", "h2", "h3", "h4", "header", "footer", "nav", "section", "article", "main",
		"label", "select", "option", "textarea", "DOCTYPE"),
	types:      words("class", "id", "src", "href", "type", "name", "value", "style", "rel", "alt", "width", "height"),
	blockStart: "<!--", blockEnd: "-->", quotes: "\"'",
}

var ceLangYAML = &ceLang{
	keywords:    words("true", "false", "null", "yes", "no", "on", "off"),
	types:       words(),
	lineComment: "#", quotes: "\"'",
}

var ceLangTOML = &ceLang{
	keywords:    words("true", "false"),
	types:       words(),
	lineComment: "#", quotes: "\"'",
}

var ceLangSQL = &ceLang{
	keywords: words("SELECT", "FROM", "WHERE", "INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE",
		"CREATE", "TABLE", "DROP", "ALTER", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "ON", "AS",
		"GROUP", "BY", "ORDER", "HAVING", "LIMIT", "DISTINCT", "AND", "OR", "NOT", "NULL", "PRIMARY",
		"KEY", "FOREIGN", "REFERENCES", "INDEX", "UNIQUE", "DEFAULT", "AUTO_INCREMENT",
		"select", "from", "where", "insert", "into", "values", "update", "set", "delete", "join", "and", "or"),
	types: words("INT", "INTEGER", "VARCHAR", "CHAR", "TEXT", "DATE", "DATETIME", "TIMESTAMP",
		"BOOLEAN", "FLOAT", "DOUBLE", "DECIMAL", "BIGINT", "true", "false"),
	lineComment: "--", blockStart: "/*", blockEnd: "*/", quotes: "'\"",
}

var ceLangShell = &ceLang{
	keywords: words("if", "then", "else", "elif", "fi", "for", "in", "do", "done", "while", "until",
		"case", "esac", "function", "return", "break", "continue", "echo", "export", "local", "source",
		"alias", "exit", "cd", "set", "unset", "read", "shift"),
	types:       words("true", "false"),
	lineComment: "#", quotes: "\"'",
}

var ceLangTS = &ceLang{
	keywords: words("abstract", "as", "async", "await", "break", "case", "catch", "class", "const",
		"continue", "debugger", "declare", "default", "delete", "do", "else", "enum", "export", "extends",
		"finally", "for", "from", "function", "get", "if", "implements", "import", "in", "infer",
		"instanceof", "interface", "is", "keyof", "let", "namespace", "new", "of", "override", "private",
		"protected", "public", "readonly", "return", "satisfies", "set", "static", "super", "switch",
		"this", "throw", "try", "type", "typeof", "var", "void", "while", "with", "yield"),
	types: words("number", "string", "boolean", "any", "unknown", "never", "void", "object", "symbol",
		"bigint", "true", "false", "null", "undefined", "Array", "Promise", "Record", "Partial", "Readonly",
		"Map", "Set", "console", "document", "window"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'`",
}

var ceLangCSharp = &ceLang{
	keywords: words("using", "namespace", "class", "struct", "interface", "enum", "public", "private",
		"protected", "internal", "static", "void", "return", "if", "else", "for", "foreach", "while", "do",
		"switch", "case", "break", "continue", "new", "this", "base", "virtual", "override", "abstract",
		"sealed", "async", "await", "try", "catch", "finally", "throw", "var", "const", "readonly", "ref",
		"out", "params", "get", "set", "in", "is", "as", "record", "partial", "yield", "lock"),
	types: words("int", "long", "short", "byte", "sbyte", "uint", "ulong", "ushort", "bool", "char",
		"string", "float", "double", "decimal", "object", "dynamic", "Task", "List", "Dictionary",
		"true", "false", "null"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

var ceLangPHP = &ceLang{
	keywords: words("echo", "print", "if", "else", "elseif", "while", "for", "foreach", "as", "function",
		"fn", "return", "class", "trait", "public", "private", "protected", "static", "new", "abstract",
		"final", "interface", "extends", "implements", "namespace", "use", "try", "catch", "finally",
		"throw", "global", "const", "switch", "case", "break", "continue", "default", "do", "instanceof",
		"yield", "match", "require", "include"),
	types: words("true", "false", "null", "int", "string", "float", "bool", "array", "object", "void",
		"mixed", "callable", "self", "parent", "static"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

var ceLangRuby = &ceLang{
	keywords: words("def", "end", "if", "elsif", "else", "unless", "while", "until", "for", "in", "do",
		"begin", "rescue", "ensure", "raise", "return", "yield", "class", "module", "require",
		"require_relative", "attr_accessor", "attr_reader", "attr_writer", "puts", "print", "then", "case",
		"when", "next", "break", "lambda", "proc", "and", "or", "not", "self", "new"),
	types:       words("Integer", "String", "Float", "Array", "Hash", "Symbol", "nil", "true", "false", "Object"),
	lineComment: "#", quotes: "\"'",
}

var ceLangKotlin = &ceLang{
	keywords: words("fun", "val", "var", "if", "else", "when", "for", "while", "do", "return", "class",
		"object", "interface", "enum", "data", "sealed", "open", "override", "abstract", "private", "public",
		"protected", "internal", "import", "package", "companion", "init", "constructor", "is", "as", "in",
		"by", "this", "super", "try", "catch", "finally", "throw", "break", "continue", "suspend", "inline",
		"reified", "typealias"),
	types: words("Int", "Long", "Short", "Byte", "Boolean", "Char", "String", "Float", "Double", "Unit",
		"Any", "Nothing", "List", "Map", "Set", "Array", "true", "false", "null"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

var ceLangSwift = &ceLang{
	keywords: words("func", "let", "var", "if", "else", "guard", "switch", "case", "default", "for",
		"while", "repeat", "return", "class", "struct", "enum", "protocol", "extension", "import", "public",
		"private", "internal", "fileprivate", "open", "static", "override", "init", "deinit", "self", "super",
		"is", "as", "in", "where", "try", "catch", "throw", "throws", "rethrows", "defer", "break",
		"continue", "fallthrough", "async", "await", "some", "any"),
	types: words("Int", "Double", "Float", "String", "Bool", "Character", "Array", "Dictionary", "Set",
		"Optional", "Any", "Void", "Self", "true", "false", "nil"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"",
}

var ceLangLua = &ceLang{
	keywords: words("function", "end", "if", "then", "else", "elseif", "while", "do", "for", "in", "repeat",
		"until", "return", "local", "and", "or", "not", "break", "goto"),
	types:       words("true", "false", "nil", "self"),
	lineComment: "--", blockStart: "--[[", blockEnd: "]]", quotes: "\"'",
}

var ceLangDart = &ceLang{
	keywords: words("void", "var", "final", "const", "class", "extends", "implements", "with", "abstract",
		"import", "library", "part", "if", "else", "for", "while", "do", "switch", "case", "default", "break",
		"continue", "return", "new", "this", "super", "try", "catch", "finally", "throw", "async", "await",
		"yield", "get", "set", "factory", "enum", "mixin", "typedef", "is", "as", "in", "late", "required"),
	types: words("int", "double", "num", "bool", "String", "List", "Map", "Set", "dynamic", "Object",
		"Future", "Stream", "void", "true", "false", "null"),
	lineComment: "//", blockStart: "/*", blockEnd: "*/", quotes: "\"'",
}

var ceLangPowerShell = &ceLang{
	keywords: words("function", "if", "elseif", "else", "switch", "foreach", "for", "while", "do", "until",
		"return", "param", "begin", "process", "end", "try", "catch", "finally", "throw", "break", "continue",
		"filter", "in", "class", "enum", "using"),
	types:       words("true", "false", "null"),
	lineComment: "#", blockStart: "<#", blockEnd: "#>", quotes: "\"'",
}

// ceLangMarkdown 用专用结构化分词器(custom，见 codehighlight_markup.go)：标题/列表/引用/粗斜体/行内代码/```围栏。
var ceLangMarkdown = &ceLang{keywords: words(), types: words()}

// ceLangFor 按名称取语言定义（自定义优先，未知→Go 兜底）。
// LangSpec 自定义语言的词法定义（用于语法高亮 + 词法补全）。
// 只需填关键字/类型/注释符/引号——CodeEditor 的高亮是数据驱动的，无需写解析器。
type LangSpec struct {
	Keywords    []string // 关键字（高亮为红）
	Types       []string // 类型/内置/常量（高亮为紫）
	LineComment string   // 行注释前缀，如 "//"、"#"、"--"
	BlockStart  string   // 块注释起始，如 "/*"（空=无）
	BlockEnd    string   // 块注释结束，如 "*/"
	Quotes      string   // 字符串引号字符集，如 "\"'"
	RawQuote    rune     // 原始字符串引号（跨行），如 '`'；0=无
}

var customLangs = map[string]*ceLang{}

// RegisterLanguage 注册自定义语言（或覆盖内置）的高亮/词法补全定义。
// name 即 NewCodeEditor 的 language 参数（不区分大小写）。
//
//	widget.RegisterLanguage("rust", widget.LangSpec{
//	    Keywords: []string{"fn", "let", "mut", "if", "else", "match", "impl", "pub", "use"},
//	    Types:    []string{"i32", "u32", "String", "bool", "Vec", "Option", "true", "false"},
//	    LineComment: "//", BlockStart: "/*", BlockEnd: "*/", Quotes: "\"'",
//	})
func RegisterLanguage(name string, spec LangSpec) {
	rq := byte(0)
	if spec.RawQuote != 0 {
		rq = byte(spec.RawQuote)
	}
	customLangs[strings.ToLower(name)] = &ceLang{
		keywords:    words(spec.Keywords...),
		types:       words(spec.Types...),
		lineComment: spec.LineComment,
		blockStart:  spec.BlockStart,
		blockEnd:    spec.BlockEnd,
		quotes:      spec.Quotes,
		rawQuote:    rq,
	}
}

// langJSON 配置文件里的语言定义（JSON 友好：rawQuote 用单字符字符串）。
type langJSON struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases"`
	Keywords    []string `json:"keywords"`
	Types       []string `json:"types"`
	LineComment string   `json:"lineComment"`
	BlockStart  string   `json:"blockStart"`
	BlockEnd    string   `json:"blockEnd"`
	Quotes      string   `json:"quotes"`
	RawQuote    string   `json:"rawQuote"`
}

// RegisterLanguageJSON 从 JSON 配置注册语言（含别名）。配置示例：
//
//	{"name":"mylang","aliases":["ml"],"keywords":["when","then"],"types":["yes","no"],
//	 "lineComment":"#","blockStart":"/*","blockEnd":"*/","quotes":"\"'","rawQuote":"`"}
func RegisterLanguageJSON(data []byte) error {
	var j langJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	if j.Name == "" {
		return errors.New("language config: name required")
	}
	rq := rune(0)
	if j.RawQuote != "" {
		rq = []rune(j.RawQuote)[0]
	}
	spec := LangSpec{
		Keywords: j.Keywords, Types: j.Types,
		LineComment: j.LineComment, BlockStart: j.BlockStart, BlockEnd: j.BlockEnd,
		Quotes: j.Quotes, RawQuote: rq,
	}
	RegisterLanguage(j.Name, spec)
	for _, a := range j.Aliases {
		RegisterLanguage(a, spec)
	}
	return nil
}

// LoadLanguageFile 从 JSON 文件加载并注册语言；目录则加载其中所有 *.json。
func LoadLanguageFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return RegisterLanguageJSON(data)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			if data, err := os.ReadFile(path + "/" + e.Name()); err == nil {
				RegisterLanguageJSON(data)
			}
		}
	}
	return nil
}

func ceLangFor(name string) *ceLang {
	if l, ok := customLangs[strings.ToLower(name)]; ok { // 自定义注册优先
		return l
	}
	switch strings.ToLower(name) {
	case "go", "golang":
		return ceLangGo
	case "ey", "易语言", "estr":
		return ceLangEY
	case "json", "jsonc", "json5":
		return ceLangJSON
	case "js", "javascript", "jsx", "mjs", "cjs":
		return ceLangJS
	case "ts", "typescript", "tsx":
		return ceLangTS
	case "py", "python":
		return ceLangPython
	case "rust", "rs":
		return ceLangRust
	case "c", "cpp", "c++", "h", "hpp":
		return ceLangC
	case "java":
		return ceLangJava
	case "css":
		return ceLangCSS
	case "scss", "less", "sass":
		return ceLangSCSS
	case "html", "htm", "xml", "vue", "svg":
		return ceLangHTML
	case "yaml", "yml":
		return ceLangYAML
	case "toml", "ini", "conf", "env":
		return ceLangTOML
	case "sql":
		return ceLangSQL
	case "sh", "bash", "shell", "zsh", "dockerfile":
		return ceLangShell
	case "cs", "csharp", "c#":
		return ceLangCSharp
	case "php":
		return ceLangPHP
	case "rb", "ruby":
		return ceLangRuby
	case "kt", "kotlin", "kts":
		return ceLangKotlin
	case "swift":
		return ceLangSwift
	case "lua":
		return ceLangLua
	case "dart":
		return ceLangDart
	case "ps1", "powershell", "psm1":
		return ceLangPowerShell
	case "md", "markdown":
		return ceLangMarkdown
	default:
		return ceLangGo
	}
}
