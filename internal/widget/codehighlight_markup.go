package widget

// 结构化标记语言的专用分词器（Markdown / HTML）。关键字模型只能近似，这里按真实语法切分：
// Markdown：标题/列表/引用/粗斜体/行内代码/链接/```围栏代码块；HTML：标签名/属性名/属性值/注释。
// 通过 ceLang.custom 钩子接入，tokenizeLine 命中即委派。

func init() {
	ceLangMarkdown.custom = tokenizeMarkdown
	ceLangHTML.custom = tokenizeHTML
}

// tokenizeMarkdown 按行高亮 Markdown。in/out 用 stFence 跟踪 ``` 围栏代码块（跨行）。
func tokenizeMarkdown(runes []rune, in ceLineState) ([]ceToken, ceLineState) {
	n := len(runes)
	ind := 0
	for ind < n && (runes[ind] == ' ' || runes[ind] == '\t') {
		ind++
	}
	// ``` 围栏：切换代码块状态，围栏行本身当注释色
	if n-ind >= 3 && runes[ind] == '`' && runes[ind+1] == '`' && runes[ind+2] == '`' {
		if in == stFence {
			return []ceToken{{0, n, tkComment}}, stNormal
		}
		return []ceToken{{0, n, tkComment}}, stFence
	}
	if in == stFence { // 代码块内：整行字符串色
		if n == 0 {
			return nil, stFence
		}
		return []ceToken{{0, n, tkString}}, stFence
	}
	if ind < n && runes[ind] == '#' { // 标题 # ## ###
		return []ceToken{{0, n, tkKeyword}}, stNormal
	}
	if ind < n && runes[ind] == '>' { // 引用块
		return []ceToken{{0, n, tkComment}}, stNormal
	}
	var toks []ceToken
	i := 0
	// 列表标记：- / * / + 后跟空格，或 有序 "数字."
	if ind < n {
		r := runes[ind]
		if (r == '-' || r == '*' || r == '+') && ind+1 < n && runes[ind+1] == ' ' {
			toks = append(toks, ceToken{ind, ind + 1, tkKeyword})
			i = ind + 1
		} else if isDigit(r) {
			j := ind
			for j < n && isDigit(runes[j]) {
				j++
			}
			if j < n && runes[j] == '.' {
				toks = append(toks, ceToken{ind, j + 1, tkKeyword})
				i = j + 1
			}
		}
	}
	// 行内：`代码`  **粗体**/*斜体*  [文字](链接)
	for i < n {
		switch r := runes[i]; {
		case r == '`': // 行内代码
			j := i + 1
			for j < n && runes[j] != '`' {
				j++
			}
			if j < n {
				j++
			}
			toks = append(toks, ceToken{i, j, tkString})
			i = j
		case r == '*' || r == '_': // 粗体/斜体
			double := i+1 < n && runes[i+1] == r
			width := 1
			if double {
				width = 2
			}
			j := i + width
			closed := -1
			for j < n {
				if runes[j] == r && (!double || (j+1 < n && runes[j+1] == r)) {
					closed = j
					break
				}
				j++
			}
			if closed >= 0 {
				end := closed + width
				toks = append(toks, ceToken{i, end, tkType})
				i = end
			} else {
				i++ // 未闭合，当普通文本
			}
		case r == '[': // 链接 [文字](url)
			j := i + 1
			for j < n && runes[j] != ']' {
				j++
			}
			if j+1 < n && runes[j] == ']' && runes[j+1] == '(' {
				k := j + 2
				for k < n && runes[k] != ')' {
					k++
				}
				if k < n {
					k++
				}
				toks = append(toks, ceToken{i, j + 1, tkFunc}) // [文字]
				toks = append(toks, ceToken{j + 1, k, tkString}) // (url)
				i = k
			} else {
				i++
			}
		default:
			i++
		}
	}
	return toks, stNormal
}

// tokenizeHTML 按结构高亮 HTML：<标签 属性="值"> + <!-- 注释 -->（跨行用 stBlockComment）。
func tokenizeHTML(runes []rune, in ceLineState) ([]ceToken, ceLineState) {
	n := len(runes)
	var toks []ceToken
	i := 0
	cmtEnd := []rune("-->")
	if in == stBlockComment { // 续接跨行注释
		if j := indexRunes(runes, cmtEnd, 0); j >= 0 {
			toks = append(toks, ceToken{0, j + 3, tkComment})
			i = j + 3
		} else {
			if n > 0 {
				toks = append(toks, ceToken{0, n, tkComment})
			}
			return toks, stBlockComment
		}
	}
	for i < n {
		r := runes[i]
		if r == '<' && hasPrefixRunes(runes, []rune("<!--"), i) { // 注释
			if j := indexRunes(runes, cmtEnd, i+4); j >= 0 {
				toks = append(toks, ceToken{i, j + 3, tkComment})
				i = j + 3
				continue
			}
			toks = append(toks, ceToken{i, n, tkComment})
			return toks, stBlockComment
		}
		if r == '<' { // 标签 <name ...> 或 </name>
			toks = append(toks, ceToken{i, i + 1, tkPunct})
			j := i + 1
			if j < n && runes[j] == '/' {
				toks = append(toks, ceToken{j, j + 1, tkPunct})
				j++
			}
			s := j // 标签名
			for j < n && (isIdentPart(runes[j]) || runes[j] == '-' || runes[j] == '!') {
				j++
			}
			if j > s {
				toks = append(toks, ceToken{s, j, tkKeyword})
			}
			for j < n && runes[j] != '>' { // 属性
				switch c := runes[j]; {
				case c == '"' || c == '\'': // 属性值
					k := j + 1
					for k < n && runes[k] != c {
						k++
					}
					if k < n {
						k++
					}
					toks = append(toks, ceToken{j, k, tkString})
					j = k
				case isIdentStart(c): // 属性名
					k := j + 1
					for k < n && (isIdentPart(runes[k]) || runes[k] == '-' || runes[k] == ':') {
						k++
					}
					toks = append(toks, ceToken{j, k, tkType})
					j = k
				case c == '=' || c == '/':
					toks = append(toks, ceToken{j, j + 1, tkPunct})
					j++
				default:
					j++
				}
			}
			if j < n && runes[j] == '>' {
				toks = append(toks, ceToken{j, j + 1, tkPunct})
				j++
			}
			i = j
			continue
		}
		// 文本内容：跳到下一个 '<'
		j := i
		for j < n && runes[j] != '<' {
			j++
		}
		i = j
	}
	return toks, stNormal
}
