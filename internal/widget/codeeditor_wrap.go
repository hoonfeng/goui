package widget

// 软自动换行（word wrap）：开启后每个【可见逻辑行】按编辑区像素宽切成 ≥1 个「视觉段」，
// 渲染/光标/点击/选区/方向键全部以「视觉段(wrapSegs)」为单位排布。关闭时每个可见行恰好
// 一个整行段 {line,0,len}，与不换行的旧行为逐字节一致（零回归）。
//
// 视觉行模型统一入口：
//   - e.wrapSegs：显示顺序的视觉段列表（含折叠，故由 visRows 派生）。
//   - segRowOf(line,col)：逻辑位置 → wrapSegs 下标（视觉行号）。
//   - posX/posTopY：逻辑位置 → 段内 x / 视觉行顶部 y（替代裸 colToX/lineTopY 做定位）。
// 这样折叠用的 visIndexOf 仍在，wrap 叠加在其上：wrap 关→段数==可见行数，segRowOf==visIndexOf。

// wrapSeg 一个视觉段：逻辑行 line 的 rune 区间 [start,end)（end 不含）。
type wrapSeg struct{ line, start, end int }

// invalidateWrap 标记需重建视觉段（内容/折叠/可见行变化后调用）。
func (e *CodeEditorElement) invalidateWrap() { e.wrapDirty = true }

// globalWordWrap 全局软自动换行开关：所有代码编辑器共享。菜单全局切换，不依赖聚焦哪个编辑器
// （右键菜单是覆盖层，弹出即令编辑器失焦，故 per-editor 的开关在菜单里根本切不动——必须全局）。
var globalWordWrap bool

// ToggleWordWrap 翻转全局软换行——所有打开的代码编辑器下次绘制即跟随（与焦点无关）。供宿主菜单调用。
func ToggleWordWrap() { globalWordWrap = !globalWordWrap; repaint() }

// WordWrapEnabled 全局软换行是否开启。供菜单勾选状态。
func WordWrapEnabled() bool { return globalWordWrap }

// ensureWrapSegs 惰性重建视觉段：宽度变 / 标脏 / 与可见行数对不上时才重算。
// viewW = 编辑区文本可视宽（Paint 算出的 editorViewW）。
func (e *CodeEditorElement) ensureWrapSegs(viewW float64) {
	if want := globalWordWrap || e.ed.WordWrap; want != e.wrap { // 跟随全局开关（或本编辑器配置常开）
		e.wrap = want
		e.wrapDirty = true
	}
	if !e.wrapDirty && e.wrapW == viewW && e.wrapSegs != nil {
		return
	}
	e.rebuildWrapSegs(viewW)
}

// rebuildWrapSegs 据当前 visRows + 换行开关重建视觉段。
func (e *CodeEditorElement) rebuildWrapSegs(viewW float64) {
	e.wrapW = viewW
	e.wrapDirty = false
	e.wrapSegs = e.wrapSegs[:0]
	for _, line := range e.visRows {
		runes := e.lineRunes(line)
		if !e.wrap || viewW <= 0 || len(runes) == 0 {
			// 不换行（或空行/无宽度）：整行一个段，行为同旧版。
			e.wrapSegs = append(e.wrapSegs, wrapSeg{line, 0, len(runes)})
			continue
		}
		e.appendLineSegs(line, runes, viewW)
	}
	if len(e.wrapSegs) == 0 { // 永远至少一段（空文档=一个空段），避免下游除零/越界
		e.wrapSegs = append(e.wrapSegs, wrapSeg{0, 0, 0})
	}
}

// appendLineSegs 把单个逻辑行按 viewW 像素宽切成若干段，优先在空白处断行，否则 rune 间硬断。
// 决不丢字符：每段至少含 1 个 rune。
func (e *CodeEditorElement) appendLineSegs(line int, runes []rune, viewW float64) {
	start := 0
	n := len(runes)
	for start < n {
		// 从 start 起，找放得下 viewW 的最大结束位置 end（>start）。
		end := start
		for end < n {
			w := e.measure(string(runes[start : end+1]))
			if w > viewW && end > start {
				break // 再加一个就超宽 → end 即断点（end 不含）
			}
			end++
		}
		if end <= start {
			end = start + 1 // 保底：单字符超宽也得吃一个，避免死循环
		}
		if end < n {
			// 优先在空白边界断：在 (start, end] 里找最靠后的空白，断在其后。
			brk := -1
			for k := end; k > start; k-- {
				if runes[k-1] == ' ' || runes[k-1] == '\t' {
					brk = k
					break
				}
			}
			if brk > start {
				end = brk
			}
		}
		e.wrapSegs = append(e.wrapSegs, wrapSeg{line, start, end})
		start = end
	}
}

// segRowOf 逻辑位置 (line,col) → wrapSegs 下标（视觉行号）。
// 恰好落在某段 end 的位置：若该段是本行最后一段则归本段，否则归下一段（贴合断点后续字符）。
func (e *CodeEditorElement) segRowOf(line, col int) int {
	last := -1
	for i, s := range e.wrapSegs {
		if s.line != line {
			if last >= 0 { // 已扫过本行所有段，col 超出 → 落最后一段
				return last
			}
			continue
		}
		last = i
		lineLastSeg := i+1 >= len(e.wrapSegs) || e.wrapSegs[i+1].line != line
		if col < s.end || (col == s.end && lineLastSeg) {
			return i
		}
	}
	if last >= 0 {
		return last
	}
	return 0
}

// segAt 返回 wrapSegs[row]（越界 clamp）。
func (e *CodeEditorElement) segAt(row int) wrapSeg {
	if len(e.wrapSegs) == 0 {
		return wrapSeg{0, 0, 0}
	}
	if row < 0 {
		row = 0
	}
	if row >= len(e.wrapSegs) {
		row = len(e.wrapSegs) - 1
	}
	return e.wrapSegs[row]
}

// posTopY 逻辑位置 (line,col) → 其所在视觉行的顶部 y（top 为视觉行 0 的顶部）。
func (e *CodeEditorElement) posTopY(line, col int, top float64) float64 {
	return top + float64(e.segRowOf(line, col))*ceLineH
}

// posX 逻辑位置 (line,col) → 段内绘制 x（left 为文本起点）。
// x = 段起列到 col 的子串像素宽（CJK 由 measure 实测）。
func (e *CodeEditorElement) posX(line, col int, left float64) float64 {
	s := e.wrapSegs[e.segRowOf(line, col)]
	r := e.lineRunes(line)
	c := col
	if c < s.start {
		c = s.start
	}
	if c > len(r) {
		c = len(r)
	}
	if c <= s.start {
		return left
	}
	return left + e.measure(string(r[s.start:c]))
}

// segColAtX 在视觉段 s 内，把段内局部 x（已扣起点）换算为逻辑列（clamp 到 [s.start,s.end]）。
func (e *CodeEditorElement) segColAtX(s wrapSeg, lx float64) int {
	r := e.lineRunes(s.line)
	if lx <= 0 {
		return s.start
	}
	col := s.start
	for col < s.end {
		w := e.measure(string(r[s.start : col+1]))
		if w > lx {
			prev := e.measure(string(r[s.start:col]))
			if lx-prev > w-lx { // 取更近一侧
				col++
			}
			return col
		}
		col++
	}
	return s.end
}

// forSegSpans 把逻辑行 line 的列区间 [c0,c1) 拆到它覆盖的每个视觉段上，
// 对每段回调 fn(rowTopY, x0, x1)（x0/x1 已含 left 偏移）。供选区/匹配/诊断逐视觉行高亮。
// crossNL=true 时（跨逻辑行选区的非末行）在本行最后一段尾部补一小段表示选中了换行。
func (e *CodeEditorElement) forSegSpans(line, c0, c1 int, left, top float64, crossNL bool, fn func(rowTopY, x0, x1 float64)) {
	r := e.lineRunes(line)
	if c1 > len(r) {
		c1 = len(r)
	}
	for i, s := range e.wrapSegs {
		if s.line != line {
			continue
		}
		lineLastSeg := i+1 >= len(e.wrapSegs) || e.wrapSegs[i+1].line != line
		// 本段与 [c0,c1) 的交集
		a, b := c0, c1
		if a < s.start {
			a = s.start
		}
		if b > s.end {
			b = s.end
		}
		rowTopY := top + float64(i)*ceLineH
		if a < b {
			x0 := left + e.measure(string(r[s.start:a]))
			x1 := left + e.measure(string(r[s.start:b]))
			if crossNL && lineLastSeg {
				x1 += 6 // 跨行换行的视觉提示
			}
			fn(rowTopY, x0, x1)
		} else if crossNL && lineLastSeg && c1 >= s.end {
			// 选区覆盖到行尾但本段无可见字符（如空行）：仍画换行提示小段
			x0 := left + e.measure(string(r[s.start:s.end]))
			fn(rowTopY, x0, x0+6)
		}
	}
}
