package widget

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
)

// 接真实 Go：用 go/parser + go/ast 把 Go 源码解析进 SEProgram 表格模型（ParseGo），
// 再用 go/format 从表格生成回 Go 源码（ToGo）。这让「文本↔表格双向」落到真实 Go 代码。
//
// 映射：包级 var → 程序集变量(Globals)；func → 子程序(SESub)——
//   函数名→Name、返回类型(多值拼成 "(a, b)")→Ret、参数→Params、doc 注释→Note、函数体源码(dedent)→Body。
//   Go 的局部变量在函数体内(var/:=)，不单独抽到 Locals（保留在 Body 里），符合 Go 语义。

// ParseGo 把 Go 源码解析成 SEProgram。
func ParseGo(src string) (*SEProgram, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	p := &SEProgram{}
	for _, imp := range f.Imports { // import 声明 → Imports（保留别名 / _ / .）
		s := imp.Path.Value // 含引号，如 "fmt"
		if imp.Name != nil {
			s = imp.Name.Name + " " + s // 别名 m "math" / _ "embed" / . "x"
		}
		p.Imports = append(p.Imports, s)
	}
	trailing := lineCommentsByLine(fset, f.Comments) // 行号→注释（参数/返回值行尾注释 ast 不挂 Field，按行号匹配）
	subRet := subReturnTypes(f.Decls)                // 子程序单返回值类型，供 := 调用推导
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.VAR: // 包级变量
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					typ := exprStr(vs.Type)
					for ni, name := range vs.Names {
						// 一行多个变量：每个名字配自己的值（Values[ni]），不再统一取 Values[0]。
						val := ""
						if ni < len(vs.Values) {
							val = exprStr(vs.Values[ni])
						}
						p.Globals = append(p.Globals, SEVar{Name: name.Name, Type: typ, Ref: val, Note: commentText(d.Doc)})
					}
				}
			case token.CONST: // 常量声明
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					typ := exprStr(vs.Type)
					for ni, name := range vs.Names {
						// 一行多个常量：每个名字配自己的值（Values[ni]），不再统一取 Values[0]。
						val := ""
						if ni < len(vs.Values) {
							val = exprStr(vs.Values[ni])
						}
						p.Consts = append(p.Consts, SEVar{Name: name.Name, Type: typ, Array: val, Note: commentText(d.Doc)})
					}
				}
			case token.TYPE: // 类型定义（struct / interface / alias）
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					td := SEType{Name: ts.Name.Name, Note: commentText(d.Doc)}
					startPos := ts.Pos()
					if ts.Doc != nil {
						startPos = ts.Doc.Pos()
					}
					td.BlankBefore = blankLinesBefore(src, fset, startPos)
					// 泛型类型参数
					if ts.TypeParams != nil {
						td.TypeParams = parseGoFields(ts.TypeParams.List, fset, trailing)
					}
					switch tt := ts.Type.(type) {
					case *ast.StructType:
						td.Kind = SETypeStruct
						if tt.Fields != nil {
							for _, f := range tt.Fields.List {
								ft := exprStr(f.Type)
								note := trailing[fset.Position(f.End()).Line]
								if len(f.Names) == 0 {
									// 嵌入字段（匿名）
									td.Fields = append(td.Fields, SEVar{Name: "", Type: ft, Note: note})
								} else {
									for _, n := range f.Names {
										td.Fields = append(td.Fields, SEVar{Name: n.Name, Type: ft, Note: note})
									}
								}
							}
						}
					case *ast.InterfaceType:
						td.Kind = SETypeInterface
						if tt.Methods != nil {
							for _, f := range tt.Methods.List {
								sig := exprStr(f.Type)
								note := trailing[fset.Position(f.End()).Line]
								if len(f.Names) == 0 {
									// 嵌入接口
									td.Methods = append(td.Methods, SEVar{Name: "", Type: sig, Note: note})
								} else {
									for _, n := range f.Names {
										td.Methods = append(td.Methods, SEVar{Name: n.Name, Type: sig, Note: note})
									}
								}
							}
						}
					default:
						// type alias / type definition: type X = Y 或 type X Y
						td.Kind = SETypeAlias
						td.TypeExpr = exprStr(ts.Type)
					}
					p.Types = append(p.Types, td)
				}
			}
		case *ast.FuncDecl:
			sub := SESub{Name: d.Name.Name, Note: commentText(d.Doc)}
			startPos := d.Pos() // 含 doc 注释则从 doc 起算，准确数函数前的空行
			if d.Doc != nil {
				startPos = d.Doc.Pos()
			}
			sub.BlankBefore = blankLinesBefore(src, fset, startPos)
			// 方法接收器
			if d.Recv != nil && len(d.Recv.List) > 0 {
				sub.Recv = recvString(d.Recv.List[0])
			}
			// 泛型类型参数
			if d.Type.TypeParams != nil {
				sub.TypeParams = parseGoFields(d.Type.TypeParams.List, fset, trailing)
			}
			if d.Type.Params != nil {
				for _, field := range d.Type.Params.List {
					typ, ref := paramTypeRef(field.Type)               // 指针参数 *T → 类型 T + 参考(传址)
					note := trailing[fset.Position(field.End()).Line] // 参数行尾注释→备注列
					if len(field.Names) == 0 {
						sub.Params = append(sub.Params, SEVar{Type: typ, Ref: ref, Note: note})
					}
					for _, name := range field.Names {
						sub.Params = append(sub.Params, SEVar{Name: name.Name, Type: typ, Ref: ref, Note: note})
					}
				}
			}
			sub.Returns = parseResults(d.Type.Results, fset, trailing)
			sub.Body = bodyText(src, fset, d.Body)
			if d.Body != nil { // 函数体里的局部变量(var/:=)→局部变量表（代码→表格）
				sub.Locals = extractGoLocals(d.Body, sub.Params, fset, trailing, subRet)
			}
			p.Subs = append(p.Subs, sub)
		}
	}
	return p, nil
}

// ToGo 从 SEProgram 生成 Go 源码（不经 gofmt，保留用户空行/分段）。
func (p *SEProgram) ToGo() string {
	doc, _ := p.goDoc(-1, "")
	return doc
}

// GoDocForBody 生成「以 body 覆盖第 si 个子程序函数体」的完整 Go 文档，并返回该函数体首行的 0 基行号。
// 用于把内嵌函数体编辑器的本地坐标映射到完整文档，喂给 gopls 做语义补全/诊断。
func (p *SEProgram) GoDocForBody(si int, body string) (doc string, bodyLine int) {
	return p.goDoc(si, body)
}

// goDoc 生成完整 Go 文档；overrideSi>=0 时该子程序函数体改用 overrideBody，并返回其体首行 0 基行号。
func (p *SEProgram) goDoc(overrideSi int, overrideBody string) (string, int) {
	var b strings.Builder
	line, bodyLine := 0, 0
	wr := func(s string) { b.WriteString(s); line += strings.Count(s, "\n") }
	wr("package main\n")

	// ── imports ──
	if len(p.Imports) > 0 {
		wr("\n")
		if len(p.Imports) == 1 {
			wr("import " + p.Imports[0] + "\n")
		} else {
			wr("import (\n")
			for _, im := range p.Imports {
				wr("\t" + im + "\n")
			}
			wr(")\n")
		}
	}

	// ── 常量 ──
	for i := range p.Consts {
		v := &p.Consts[i]
		if i == 0 && (len(p.Imports) > 0 || line > 0) {
			wr("\n")
		}
		if v.Note != "" {
			wr("// " + v.Note + "\n")
		}
		val := v.Array // 常量值（独立列）
		switch {
		case val != "" && v.Type != "":
			wr("const " + v.Name + " " + v.Type + " = " + val + "\n")
		case val != "":
			wr("const " + v.Name + " = " + val + "\n")
		case v.Type != "":
			wr("const " + v.Name + " " + v.Type + "\n")
		default:
			wr("const " + v.Name + "\n")
		}
	}

	// ── 全局变量 ──
	for i := range p.Globals {
		v := &p.Globals[i]
		if i == 0 && (len(p.Consts) > 0 || len(p.Imports) > 0 || line > 0) {
			wr("\n")
		}
		if v.Note != "" {
			wr("// " + v.Note + "\n")
		}
		val := v.Ref // 初始值（独立列）
		switch {
		case val != "" && v.Type != "":
			wr("var " + v.Name + " " + v.Type + " = " + val + "\n")
		case val != "":
			wr("var " + v.Name + " = " + val + "\n")
		default:
			wr("var " + v.Name + " " + v.Type + "\n")
		}
	}

	// ── 类型定义 ──
	for i := range p.Types {
		td := &p.Types[i]
		nb := td.BlankBefore
		if nb < 1 {
			nb = 1
		}
		wr(strings.Repeat("\n", nb))
		if td.Note != "" {
			wr("// " + td.Note + "\n")
		}
		// 泛型参数
		tparams := typeParamsGo(td.TypeParams)
		switch td.Kind {
		case SETypeStruct:
			wr("type " + td.Name + tparams + " struct {\n")
			for _, f := range td.Fields {
				if f.Name != "" {
					wr("\t" + f.Name + " " + f.Type + "\n")
				} else {
					// 嵌入字段
					wr("\t" + f.Type + "\n")
				}
			}
			wr("}\n")
		case SETypeInterface:
			wr("type " + td.Name + tparams + " interface {\n")
			for _, m := range td.Methods {
				if m.Name != "" {
					wr("\t" + m.Name + " " + m.Type + "\n")
				} else {
					// 嵌入接口
					wr("\t" + m.Type + "\n")
				}
			}
			wr("}\n")
		case SETypeAlias:
			wr("type " + td.Name + " " + td.TypeExpr + "\n")
		}
	}

	// ── 子程序（函数/方法） ──
	for i := range p.Subs {
		s := &p.Subs[i]
		nb := s.BlankBefore // 还原函数前空行（保留用户分段），至少 1 行隔开
		if nb < 1 {
			nb = 1
		}
		wr(strings.Repeat("\n", nb))
		if s.Note != "" {
			wr("// " + s.Note + "\n")
		}
		tparams := typeParamsGo(s.TypeParams)
		wr("func " + s.Recv + s.Name + tparams + "(" + paramsGo(s.Params) + ") " + retGo(s.Returns) + "{\n")
		bd := s.Body
		if i == overrideSi { // "{" 行已写完，line 即体首行 0 基行号
			bd = overrideBody
			bodyLine = line
		}
		for _, ln := range strings.Split(bd, "\n") {
			if ln == "" {
				wr("\n")
			} else {
				wr("\t" + ln + "\n")
			}
		}
		wr("}\n")
	}
	return b.String(), bodyLine
}

// goExtractLocals 解析子程序函数体（包一层 func 再 parse），提取局部变量回填局部变量表 + 重绘。
// 在 Go 模式的「回车」时调（代码→表格实时同步）；语法不完整则不动表格。
func (e *StructEditorElement) goExtractLocals(si int) {
	if si >= len(e.program.Subs) {
		return
	}
	sub := &e.program.Subs[si]
	src := "package p\nfunc _w() {\n" + sub.Body + "\n}\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return
	}
	trailing := lineCommentsByLine(fset, f.Comments)
	subRet := map[string][]string{} // 已声明子程序的返回类型列表，供 := 调用推导（含多返回）
	for i := range e.program.Subs {
		if rts := returnsTypes(e.program.Subs[i].Returns); len(rts) > 0 {
			subRet[e.program.Subs[i].Name] = rts
		}
	}
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == "_w" {
			sub.Locals = extractGoLocals(fd.Body, sub.Params, fset, trailing, subRet)
			break
		}
	}
	repaint()
}

// extractGoLocals 遍历函数体 AST，收集 var/短声明(:=) 的局部变量（排除参数/空白名）；
// trailing 是「行号→注释」表，按声明语句结束行取行尾注释→Note。
func extractGoLocals(body *ast.BlockStmt, params []SEVar, fset *token.FileSet, trailing map[int]string, subRet map[string][]string) []SEVar {
	declared := map[string]bool{}
	for i := range params {
		declared[params[i].Name] = true
	}
	var locals []SEVar
	add := func(name, typ, note string) {
		if name == "_" || declared[name] {
			return
		}
		declared[name] = true
		locals = append(locals, SEVar{Name: name, Type: typ, Note: note})
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.GenDecl:
			if d.Tok == token.VAR {
				for _, spec := range d.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						typ := exprStr(vs.Type)
						note := trailing[fset.Position(vs.End()).Line]
						for i, name := range vs.Names {
							t := typ
							if t == "" && i < len(vs.Values) {
								t = inferGoType(vs.Values[i], subRet)
							}
							add(name.Name, t, note)
						}
					}
				}
			}
		case *ast.AssignStmt:
			if d.Tok == token.DEFINE { // x := ...
				note := trailing[fset.Position(d.End()).Line]
				if len(d.Rhs) == 1 && len(d.Lhs) > 1 { // x, y := f()：用被调函数的返回类型列表逐一对应
					var rts []string
					if call, ok := d.Rhs[0].(*ast.CallExpr); ok {
						if id, ok := call.Fun.(*ast.Ident); ok {
							rts = subRet[id.Name]
						}
					}
					for i, lhs := range d.Lhs {
						if id, ok := lhs.(*ast.Ident); ok {
							t := ""
							if i < len(rts) {
								t = rts[i]
							}
							add(id.Name, t, note)
						}
					}
				} else { // x := … / x, y := a, b：右值逐一推导
					for i, lhs := range d.Lhs {
						if id, ok := lhs.(*ast.Ident); ok {
							t := ""
							if i < len(d.Rhs) {
								t = inferGoType(d.Rhs[i], subRet)
							}
							add(id.Name, t, note)
						}
					}
				}
			}
		}
		return true
	})
	return locals
}

// subReturnTypes 收集所有 func 的返回值类型列表（name→[]type，展平具名/多值），供 := 调用推导。
func subReturnTypes(decls []ast.Decl) map[string][]string {
	m := map[string][]string{}
	for _, decl := range decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			if rts := funcReturnTypes(fd.Type.Results); len(rts) > 0 {
				m[fd.Name.Name] = rts
			}
		}
	}
	return m
}

// funcReturnTypes 把返回值字段列表展平成类型切片（具名 "(a, b int)" → [int,int]）。
func funcReturnTypes(rs *ast.FieldList) []string {
	if rs == nil {
		return nil
	}
	var out []string
	for _, f := range rs.List {
		ts := exprStr(f.Type)
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			out = append(out, ts)
		}
	}
	return out
}

// inferGoType 由右值推导 Go 类型（:= 无显式类型时用）：字面量/复合字面量/取地址/比较/算术/
// 内建调用(make/new/len…)/类型转换/已知子程序调用(subRet)。推不出→""。
func inferGoType(x ast.Expr, subRet map[string][]string) string {
	switch v := x.(type) {
	case *ast.BasicLit:
		switch v.Kind {
		case token.INT:
			return "int"
		case token.FLOAT:
			return "float64"
		case token.STRING:
			return "string"
		case token.CHAR:
			return "rune"
		case token.IMAG:
			return "complex128"
		}
	case *ast.Ident:
		if v.Name == "true" || v.Name == "false" {
			return "bool"
		}
	case *ast.CompositeLit: // []int{…} / map[k]v{…} / Foo{…}
		return exprStr(v.Type)
	case *ast.ParenExpr:
		return inferGoType(v.X, subRet)
	case *ast.UnaryExpr:
		switch v.Op {
		case token.AND: // &Foo{…} → *Foo
			if inner := inferGoType(v.X, subRet); inner != "" {
				return "*" + inner
			}
		case token.NOT: // !x → bool
			return "bool"
		default:
			return inferGoType(v.X, subRet)
		}
	case *ast.BinaryExpr:
		switch v.Op {
		case token.LAND, token.LOR, token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
			return "bool" // 比较/逻辑 → bool
		}
		if t := inferGoType(v.X, subRet); t != "" { // 算术 → 操作数类型
			return t
		}
		return inferGoType(v.Y, subRet)
	case *ast.CallExpr:
		if id, ok := v.Fun.(*ast.Ident); ok {
			if rts, ok := subRet[id.Name]; ok && len(rts) == 1 { // 已知子程序(单返回值)
				return rts[0]
			}
			return inferBuiltinCall(id.Name, v.Args)
		}
	}
	return ""
}

// inferBuiltinCall 推导内建函数/类型转换的结果类型。
func inferBuiltinCall(name string, args []ast.Expr) string {
	switch name {
	case "make":
		if len(args) > 0 {
			return exprStr(args[0])
		}
	case "new":
		if len(args) > 0 {
			return "*" + exprStr(args[0])
		}
	case "len", "cap", "copy":
		return "int"
	case "append":
		if len(args) > 0 {
			return inferGoType(args[0], nil)
		}
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune", "float32", "float64", "bool", "error":
		return name // 类型转换 T(x)
	}
	return ""
}

// ── helpers ──

func exprStr(x ast.Expr) string {
	if x == nil {
		return ""
	}
	return types.ExprString(x)
}

// recvString 把接收器字段转为 "(r *T)" 字符串。
func recvString(field *ast.Field) string {
	typ := exprStr(field.Type)
	if len(field.Names) == 0 {
		return "(" + typ + ")"
	}
	return "(" + field.Names[0].Name + " " + typ + ")"
}

// parseGoFields 把 AST 字段列表解析为 SEVar 切片（用于类型参数/泛型）。
func parseGoFields(fields []*ast.Field, fset *token.FileSet, trailing map[int]string) []SEVar {
	var out []SEVar
	for _, f := range fields {
		ft := exprStr(f.Type)
		note := trailing[fset.Position(f.End()).Line]
		if len(f.Names) == 0 {
			out = append(out, SEVar{Type: ft, Note: note})
		} else {
			for _, n := range f.Names {
				out = append(out, SEVar{Name: n.Name, Type: ft, Note: note})
			}
		}
	}
	return out
}

// typeParamsGo 把类型参数列表生成 "[T any, U comparable]" 字符串。
func typeParamsGo(params []SEVar) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, 0, len(params))
	for i := range params {
		parts = append(parts, fieldGo(&params[i]))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// parseResults 把返回值字段列表解析成 []SEVar（多返回值一行一个；无名→Name 空；行尾注释→Note）。
func parseResults(results *ast.FieldList, fset *token.FileSet, trailing map[int]string) []SEVar {
	if results == nil {
		return nil
	}
	var out []SEVar
	for _, f := range results.List {
		ts := exprStr(f.Type)
		note := trailing[fset.Position(f.End()).Line]
		if len(f.Names) == 0 {
			out = append(out, SEVar{Type: ts, Note: note})
		} else {
			for _, n := range f.Names {
				out = append(out, SEVar{Name: n.Name, Type: ts, Note: note})
			}
		}
	}
	return out
}

// lineCommentsByLine 建「行号→注释文本」表（go/parser 对函数参数/返回值的行尾注释不挂到 Field，故按行号匹配）。
func lineCommentsByLine(fset *token.FileSet, comments []*ast.CommentGroup) map[int]string {
	m := map[int]string{}
	for _, cg := range comments {
		for _, c := range cg.List {
			m[fset.Position(c.Slash).Line] = cleanComment(c.Text)
		}
	}
	return m
}

// cleanComment 去掉注释定界符（// 或 /* */）。
func cleanComment(s string) string {
	if strings.HasPrefix(s, "//") {
		return strings.TrimSpace(s[2:])
	}
	if strings.HasPrefix(s, "/*") {
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "/*"), "*/"))
	}
	return strings.TrimSpace(s)
}

// paramsGo 生成参数列表：无注释→单行 "a int, b int"；有注释→多行+行尾 //（gofmt 保留）。
func paramsGo(params []SEVar) string {
	if len(params) == 0 {
		return ""
	}
	if !anyNote(params) {
		parts := make([]string, 0, len(params))
		for i := range params {
			parts = append(parts, fieldGo(&params[i]))
		}
		return strings.Join(parts, ", ")
	}
	var b strings.Builder
	b.WriteString("\n")
	for i := range params {
		b.WriteString("\t" + fieldGo(&params[i]) + ",")
		if params[i].Note != "" {
			b.WriteString(" // " + params[i].Note)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// retGo 生成返回值（含尾空格）：无→""；单无名无注释→"int "；否则 "(…) "，有注释则多行+行尾 //。
func retGo(returns []SEVar) string {
	if len(returns) == 0 {
		return ""
	}
	if !anyNote(returns) {
		if len(returns) == 1 && returns[0].Name == "" {
			return returns[0].Type + " "
		}
		parts := make([]string, 0, len(returns))
		for i := range returns {
			parts = append(parts, fieldGo(&returns[i]))
		}
		return "(" + strings.Join(parts, ", ") + ") "
	}
	var b strings.Builder
	b.WriteString("(\n")
	for i := range returns {
		b.WriteString("\t" + fieldGo(&returns[i]) + ",")
		if returns[i].Note != "" {
			b.WriteString(" // " + returns[i].Note)
		}
		b.WriteString("\n")
	}
	b.WriteString(") ")
	return b.String()
}

// returnsTypes 把返回值表展平成类型切片（供 := 多返回推导）。
func returnsTypes(rs []SEVar) []string {
	out := make([]string, 0, len(rs))
	for i := range rs {
		out = append(out, rs[i].Type)
	}
	return out
}

// paramTypeRef 解析参数类型：
//   - *T（简单指针，星号下直接是标识符）→ (T, "是")，表格中以"参考"列打勾表示指针传参
//   - **T、*[]int、*map[k]v、*func() 等复杂指针 → (完整类型, "")，保留完整类型字符串
//   - 非指针 → (类型, "")。
func paramTypeRef(t ast.Expr) (typ, ref string) {
	if star, ok := t.(*ast.StarExpr); ok {
		// 只对 *Ident（简单指针类型 *T）做拆分，显示为类型=T + 参考=是
		// 对 **T、*[]int、*struct{…}、*map[k]v、*func(…) 等复杂指针保留完整类型
		if _, ok := star.X.(*ast.Ident); ok {
			return exprStr(star.X), "是"
		}
	}
	return exprStr(t), ""
}

func fieldGo(v *SEVar) string {
	typ := v.Type
	if v.Ref == "是" { // 参考(传址) = Go 指针参数 *T
		typ = "*" + typ
	}
	if v.Name != "" {
		return v.Name + " " + typ
	}
	return typ
}

func anyNote(vars []SEVar) bool {
	for i := range vars {
		if vars[i].Note != "" {
			return true
		}
	}
	return false
}

func commentText(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(doc.Text(), "\n", " "))
}

// bodyText 取函数体大括号内部源码并去掉公共缩进（dedent 一层）。
// 只去掉 { 后/} 前那一个必然的换行与缩进，保留用户在体首/体尾打的额外空行。
func bodyText(src string, fset *token.FileSet, body *ast.BlockStmt) string {
	if body == nil {
		return ""
	}
	lo := fset.Position(body.Lbrace).Offset + 1
	hi := fset.Position(body.Rbrace).Offset
	if lo < 0 || hi > len(src) || lo > hi {
		return ""
	}
	inner := strings.TrimPrefix(src[lo:hi], "\n") // 去掉 { 后那一个换行（额外空行保留）
	if i := strings.LastIndex(inner, "\n"); i >= 0 && strings.TrimSpace(inner[i+1:]) == "" {
		inner = inner[:i] // 去掉 } 前那一行的缩进空白（额外空行保留）
	}
	return dedent(inner)
}

// blankLinesBefore 数 pos 所在行之前、紧邻的连续空行数（用于保留函数间分段空行）。
func blankLinesBefore(src string, fset *token.FileSet, pos token.Pos) int {
	off := fset.Position(pos).Offset
	if off < 0 || off > len(src) {
		return 0
	}
	i := off // 回到本行行首
	for i > 0 && src[i-1] != '\n' {
		i--
	}
	count := 0
	for i > 0 { // 逐行往前看，是空行就计数
		j := i - 1 // 指向上一行末尾的 '\n'
		k := j
		for k > 0 && src[k-1] != '\n' {
			k--
		}
		if strings.TrimSpace(src[k:j]) == "" {
			count++
			i = k
		} else {
			break
		}
	}
	return count
}

// dedent 去掉所有非空行的公共前导空白。
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	min := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if min < 0 || n < min {
			min = n
		}
	}
	if min <= 0 {
		return s
	}
	for i, l := range lines {
		if len(l) >= min {
			lines[i] = l[min:]
		}
	}
	return strings.Join(lines, "\n")
}

// Ensure fmt is used (for potential future debug formatting)
var _ = fmt.Sprintf
