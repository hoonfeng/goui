package widget

import (
	"math"
	"reflect"
	"sort"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Table 表格（el-table：排序 + 多选 + 可展开行 + 固定列 + 固定表头竖滚 + slot）──
// 列定义 + 行数据 → 表头 + 数据行。Sortable 列点列头排序(order 显示顺序)；
// Selectable 首列 checkbox(全选/行选)；Expandable 首列 ▶ 展开行；
// Fixed("left"/"right") 列固定不随水平滚动(中间列滚动、固定列贴边+阴影)；
// Height>0 表格固定高度→表头固定 + body 区内部竖滚 + 右侧竖滚条(与横向固定列正交)；
// Render 列单元格嵌任意 widget。
// 横向布局区：展开▶ → 多选☑ → [左固定列 | 中间滚动列 | 右固定列]。

type TableColumn struct {
	Label    string
	Prop     string
	Width    float64                            // 0=平分剩余(无固定列时)；有固定列时必填，默认 100
	Align    string                             // "left"(默认)/"center"/"right"
	Sortable bool                               // 点列头排序
	Fixed    string                             // ""/"left"/"right" 固定列
	Render   func(row map[string]string) Widget // 设了则单元格渲染此 widget，替代纯文本
}

type Table struct {
	StatelessWidget
	Columns         []TableColumn
	Data            []map[string]string
	Stripe          bool
	Selectable      bool // 首列多选 checkbox
	Expandable      bool // 首列 ▶ 展开行
	RenderExpand    func(row map[string]string) Widget
	OnSelectChange  func(rows []int)
	OnSortChange    func(prop string, asc bool)
	OnExpandChange  func(row int, expanded bool)
	DefaultExpanded []int   // 初始展开的行(原行索引)
	DefaultScrollX  float64 // 初始水平滚动(主要用于无交互的可视验证)
	Height          float64 // >0 固定高度：表头固定 + body 内部竖滚
}

func NewTable(columns []TableColumn, data []map[string]string) *Table {
	return &Table{Columns: columns, Data: data}
}
func (t *Table) WithStripe(s bool) *Table     { t.Stripe = s; return t }
func (t *Table) WithSelectable(s bool) *Table { t.Selectable = s; return t }

// WithExpand 开启可展开行，fn 返回每行的展开内容。
func (t *Table) WithExpand(fn func(row map[string]string) Widget) *Table {
	t.Expandable = true
	t.RenderExpand = fn
	return t
}

// WithDefaultExpanded 设置初始展开的行(原行索引)。
func (t *Table) WithDefaultExpanded(rows ...int) *Table {
	t.DefaultExpanded = rows
	return t
}

// WithHeight 设置表格固定高度：表头固定、body 区内部竖向滚动。
func (t *Table) WithHeight(h float64) *Table {
	t.Height = h
	return t
}

const (
	tableRowH      = 44.0
	tableHeaderH   = 44.0
	tableCellPad   = 12.0
	tableCheckW    = 44.0
	tableExpandW   = 44.0
	tableExpandPad = 8.0  // 展开区上下内边距
	tableHBarH     = 10.0 // 水平滚动条高度
	tableVBarW     = 10.0 // 垂直滚动条宽度
)

func (t *Table) CreateElement() Element {
	return &TableElement{BaseElement: BaseElement{widget: t}, table: t, hoverRow: -1, sortCol: -1}
}

type TableElement struct {
	BaseElement
	table       *Table
	colWidths   []float64
	checkW      float64
	expandW     float64
	hoverRow    int // 显示行(displayRow)
	cellEls     map[int]Element
	expandEls   map[int]Element // 原行索引 → 展开内容 element
	expandH     map[int]float64 // 原行索引 → 展开区高度
	rowTops     []float64       // 显示行 → top(相对 header 底部，含上方展开区累加)
	contentH    float64         // 所有行+展开总高
	bodyH       float64         // body 可视高度(Height>0 时 = Height-表头-hbar；否则 = contentH)
	order       []int           // 显示顺序 → 原行索引
	sortCol     int             // -1=未排序
	sortAsc     bool
	selected    map[int]bool // 原行索引
	expanded    map[int]bool // 原行索引
	scrollX     float64      // 中间列水平滚动偏移
	maxScrollX  float64
	scrollY     float64 // body 垂直滚动偏移
	maxScrollY  float64
	frozenLeftW float64   // 左固定总宽(含 expand/check)
	rightFixedW float64   // 右固定总宽
	colX        []float64 // 每列渲染起点 x(相对 pos，中间列已减 scrollX)
	colZone     []int     // 0=左固定 1=中间滚动 2=右固定
	hbarDrag    bool
	hDragMouseX float64
	hDragScroll float64
	vbarDrag    bool
	vDragMouseY float64
	vDragScroll float64
	inited      bool
}

func (e *TableElement) ensureInit() {
	if e.inited {
		return
	}
	e.inited = true
	e.selected = map[int]bool{}
	e.expanded = map[int]bool{}
	for _, r := range e.table.DefaultExpanded {
		if r >= 0 && r < len(e.table.Data) {
			e.expanded[r] = true
		}
	}
	e.scrollX = e.table.DefaultScrollX
	e.resetOrder()
}

func (e *TableElement) resetOrder() {
	e.order = e.order[:0]
	for i := range e.table.Data {
		e.order = append(e.order, i)
	}
}

func (e *TableElement) applySort() {
	e.resetOrder()
	if e.sortCol < 0 || e.sortCol >= len(e.table.Columns) {
		return
	}
	prop := e.table.Columns[e.sortCol].Prop
	asc := e.sortAsc
	sort.SliceStable(e.order, func(a, b int) bool {
		va := e.table.Data[e.order[a]][prop]
		vb := e.table.Data[e.order[b]][prop]
		if asc {
			return va < vb
		}
		return va > vb
	})
}

func (e *TableElement) Build() []Element {
	e.ensureInit()
	if e.cellEls == nil {
		e.cellEls = map[int]Element{}
	}
	ncol := len(e.table.Columns)
	var kids []Element
	used := map[int]bool{}
	for c, col := range e.table.Columns {
		if col.Render == nil {
			continue
		}
		for r, row := range e.table.Data {
			w := col.Render(row)
			if w == nil {
				continue
			}
			key := r*ncol + c
			el, ok := e.cellEls[key]
			if ok && reflect.TypeOf(el.Widget()) == reflect.TypeOf(w) {
				el.Update(w)
			} else {
				if ok {
					el.Unmount()
				}
				el = CreateElementFor(w)
				el.Mount(e, key)
				e.cellEls[key] = el
			}
			used[key] = true
			kids = append(kids, el)
		}
	}
	for k, el := range e.cellEls {
		if !used[k] {
			el.Unmount()
			delete(e.cellEls, k)
		}
	}

	// 展开内容 element（只为已展开的行构建）
	if e.table.Expandable && e.table.RenderExpand != nil {
		if e.expandEls == nil {
			e.expandEls = map[int]Element{}
		}
		usedExp := map[int]bool{}
		for r, row := range e.table.Data {
			if !e.expanded[r] {
				continue
			}
			w := e.table.RenderExpand(row)
			if w == nil {
				continue
			}
			el, ok := e.expandEls[r]
			if ok && reflect.TypeOf(el.Widget()) == reflect.TypeOf(w) {
				el.Update(w)
			} else {
				if ok {
					el.Unmount()
				}
				el = CreateElementFor(w)
				el.Mount(e, 1000000+r) // key 偏移避免与 cellEls 冲突
				e.expandEls[r] = el
			}
			usedExp[r] = true
			kids = append(kids, el)
		}
		for r, el := range e.expandEls {
			if !usedExp[r] {
				el.Unmount()
				delete(e.expandEls, r)
			}
		}
	}

	e.children = kids
	return kids
}

// computeCols 计算每列渲染 x、所属区(zone)、各固定区宽度与最大水平滚动。
func (e *TableElement) computeCols(W, leftW float64, enableScroll bool) {
	var leftFixedW, rightFixedW, midW float64
	for i, c := range e.table.Columns {
		switch c.Fixed {
		case "left":
			leftFixedW += e.colWidths[i]
		case "right":
			rightFixedW += e.colWidths[i]
		default:
			midW += e.colWidths[i]
		}
	}
	frozenLeftW := leftW + leftFixedW
	midViewW := W - frozenLeftW - rightFixedW
	if midViewW < 0 {
		midViewW = 0
	}
	if enableScroll {
		e.maxScrollX = math.Max(0, midW-midViewW)
	} else {
		e.maxScrollX = 0
	}
	if e.scrollX > e.maxScrollX {
		e.scrollX = e.maxScrollX
	}
	if e.scrollX < 0 {
		e.scrollX = 0
	}
	e.frozenLeftW = frozenLeftW
	e.rightFixedW = rightFixedW
	e.colX = e.colX[:0]
	e.colZone = e.colZone[:0]
	var lacc, macc, racc float64
	for i, c := range e.table.Columns {
		switch c.Fixed {
		case "left":
			e.colX = append(e.colX, leftW+lacc)
			e.colZone = append(e.colZone, 0)
			lacc += e.colWidths[i]
		case "right":
			e.colX = append(e.colX, W-rightFixedW+racc)
			e.colZone = append(e.colZone, 2)
			racc += e.colWidths[i]
		default:
			e.colX = append(e.colX, frozenLeftW+macc-e.scrollX)
			e.colZone = append(e.colZone, 1)
			macc += e.colWidths[i]
		}
	}
}

func (e *TableElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.ensureInit()
	w := ctx.Constraints.MaxWidth
	if w >= float64(1<<30) {
		w = 600
	}
	e.checkW = 0
	if e.table.Selectable {
		e.checkW = tableCheckW
	}
	e.expandW = 0
	if e.table.Expandable {
		e.expandW = tableExpandW
	}
	leftW := e.expandW + e.checkW
	availW := w - leftW

	hasFixed := false
	for _, c := range e.table.Columns {
		if c.Fixed != "" {
			hasFixed = true
			break
		}
	}

	// 列宽：有固定列时各列用自身 Width(兜底 100)；否则平分剩余
	e.colWidths = e.colWidths[:0]
	if hasFixed {
		for _, c := range e.table.Columns {
			cw := c.Width
			if cw <= 0 {
				cw = 100
			}
			e.colWidths = append(e.colWidths, cw)
		}
	} else {
		fixedW, flexCount := 0.0, 0
		for _, c := range e.table.Columns {
			if c.Width > 0 {
				fixedW += c.Width
			} else {
				flexCount++
			}
		}
		flexW := 0.0
		if flexCount > 0 {
			if rem := availW - fixedW; rem > 0 {
				flexW = rem / float64(flexCount)
			}
		}
		for _, c := range e.table.Columns {
			if c.Width > 0 {
				e.colWidths = append(e.colWidths, c.Width)
			} else {
				e.colWidths = append(e.colWidths, flexW)
			}
		}
	}
	// 竖滚时为右侧竖滚条留出宽度
	colsW := w
	if e.table.Height > 0 {
		colsW = w - tableVBarW
	}
	e.computeCols(colsW, leftW, hasFixed)

	// 先 Layout 展开内容，拿到每个展开区的高度
	e.expandH = map[int]float64{}
	for r, el := range e.expandEls {
		res := el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: availW - tableCellPad*2, MinHeight: 0, MaxHeight: float64(1 << 20),
		}})
		e.expandH[r] = res.Size.Height + tableExpandPad*2
	}

	// 累加每个显示行的 top（展开行下方插入展开区，后续行下移）
	e.rowTops = e.rowTops[:0]
	y := 0.0
	for _, r := range e.order {
		e.rowTops = append(e.rowTops, y)
		y += tableRowH
		if e.table.Expandable && e.expanded[r] {
			y += e.expandH[r]
		}
	}
	e.contentH = y

	// 垂直滚动：Height>0 固定高度 → body 内部竖滚
	hbarSpace := 0.0
	if e.maxScrollX > 0 {
		hbarSpace = tableHBarH
	}
	if e.table.Height > 0 {
		e.bodyH = e.table.Height - tableHeaderH - hbarSpace
		if e.bodyH < 0 {
			e.bodyH = 0
		}
		e.maxScrollY = math.Max(0, e.contentH-e.bodyH)
		if e.scrollY > e.maxScrollY {
			e.scrollY = e.maxScrollY
		}
		if e.scrollY < 0 {
			e.scrollY = 0
		}
		e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: e.table.Height})
	} else {
		e.bodyH = e.contentH
		e.maxScrollY = 0
		e.scrollY = 0
		e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: tableHeaderH + e.contentH + hbarSpace})
	}

	// slot 单元格按"显示行" + 列渲染 x 定位（含竖向 scrollY 偏移）
	ncol := len(e.table.Columns)
	for c, col := range e.table.Columns {
		if col.Render == nil {
			continue
		}
		cx := e.colX[c]
		cw := e.colWidths[c]
		for disp, r := range e.order {
			el := e.cellEls[r*ncol+c]
			if el == nil {
				continue
			}
			res := el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
				MinWidth: 0, MaxWidth: cw - tableCellPad*2, MinHeight: 0, MaxHeight: tableRowH,
			}})
			ry := tableHeaderH + e.rowTops[disp] - e.scrollY
			el.SetPosition(types.Point{X: cx + tableCellPad, Y: ry + (tableRowH-res.Size.Height)/2})
		}
	}

	// 展开内容定位（行下方，缩进对齐数据区，含 scrollY）
	for disp, r := range e.order {
		el := e.expandEls[r]
		if el == nil {
			continue
		}
		el.SetPosition(types.Point{X: leftW + tableCellPad, Y: tableHeaderH + e.rowTops[disp] + tableRowH + tableExpandPad - e.scrollY})
	}

	return layout.LayoutResult{Size: e.size}
}

func tableHAlign(a string) canvas.TextHAlign {
	switch a {
	case "center":
		return canvas.HAlignCenter
	case "right":
		return canvas.HAlignRight
	default:
		return canvas.HAlignLeft
	}
}

func drawTableCheck(cvs canvas.Canvas, cx, cy float64, checked bool) {
	if checked {
		f := paint.DefaultPaint()
		f.Color = elPrimary()
		cvs.DrawRoundedRect(cx-7, cy-7, 14, 14, 3, f)
		ck := paint.DefaultStrokePaint()
		ck.Color = elSurface()
		ck.StrokeWidth = 1.6
		cvs.DrawLine(cx-4, cy, cx-1, cy+3, ck)
		cvs.DrawLine(cx-1, cy+3, cx+4, cy-3, ck)
	} else {
		b := paint.DefaultStrokePaint()
		b.Color = elPlaceholder()
		b.StrokeWidth = 1.2
		cvs.DrawRoundedRect(cx-6.5, cy-6.5, 13, 13, 3, b)
	}
}

// drawExpandArrow 画展开列的 ▶(收)/▼(展) 箭头。
func drawExpandArrow(cvs canvas.Canvas, ax, ay float64, expanded bool) {
	ap := paint.DefaultStrokePaint()
	ap.Color = elTextSecondary()
	ap.StrokeWidth = 1.5
	if expanded {
		cvs.DrawLine(ax-4, ay-2, ax, ay+2, ap)
		cvs.DrawLine(ax, ay+2, ax+4, ay-2, ap)
	} else {
		cvs.DrawLine(ax-2, ay-4, ax+2, ay, ap)
		cvs.DrawLine(ax+2, ay, ax-2, ay+4, ap)
	}
}

func (e *TableElement) allSelected() bool {
	if len(e.table.Data) == 0 {
		return false
	}
	return len(e.selected) == len(e.table.Data)
}

// paintZonedCols 对三个横向区(中/左/右)分别 clip 后调 fn(i) 画属于该区的列。
// clipY/clipH 限定纵向范围(表头层=表头区；body 层=body 区)。
func (e *TableElement) paintZonedCols(cvs canvas.Canvas, pos types.Point, W, clipY, clipH float64, fn func(i int)) {
	midViewW := W - e.frozenLeftW - e.rightFixedW
	if midViewW > 0 {
		cvs.Save()
		cvs.ClipRect(pos.X+e.frozenLeftW, clipY, midViewW, clipH)
		for i := range e.table.Columns {
			if e.colZone[i] == 1 {
				fn(i)
			}
		}
		cvs.Restore()
	}
	if e.frozenLeftW > 0 {
		cvs.Save()
		cvs.ClipRect(pos.X, clipY, e.frozenLeftW, clipH)
		for i := range e.table.Columns {
			if e.colZone[i] == 0 {
				fn(i)
			}
		}
		cvs.Restore()
	}
	if e.rightFixedW > 0 {
		cvs.Save()
		cvs.ClipRect(pos.X+W-e.rightFixedW, clipY, e.rightFixedW, clipH)
		for i := range e.table.Columns {
			if e.colZone[i] == 2 {
				fn(i)
			}
		}
		cvs.Restore()
	}
}

func (e *TableElement) Paint(cvs canvas.Canvas, offset types.Point) {
	e.ensureInit()
	pos := e.Offset()
	t := e.table
	W := e.size.Width
	leftW := e.expandW + e.checkW
	bodyTop := pos.Y + tableHeaderH
	lp := paint.DefaultStrokePaint()
	lp.Color = elBorderLighter()
	lp.StrokeWidth = 1

	// ── 表头背景(全宽固定) ──
	hbg := paint.DefaultPaint()
	hbg.Color = elFill()
	cvs.DrawRect(pos.X, pos.Y, W, tableHeaderH, hbg)

	// ── body 层：行装饰(bg/stripe/hover/展开bg/分隔线 + expand箭头/checkbox)，clip 到 body 区，行 Y 减 scrollY ──
	cvs.Save()
	cvs.ClipRect(pos.X, bodyTop, W, e.bodyH)
	for disp, r := range e.order {
		ry := bodyTop + e.rowTops[disp] - e.scrollY
		if ry+tableRowH < bodyTop || ry > bodyTop+e.bodyH {
			// 行完全在可视区外，跳过装饰（展开区可能仍需画背景，单独判断）
		}
		if disp == e.hoverRow {
			rp := paint.DefaultPaint()
			rp.Color = types.ColorFromRGB(236, 245, 255)
			cvs.DrawRect(pos.X, ry, W, tableRowH, rp)
		} else if t.Stripe && disp%2 == 1 {
			rp := paint.DefaultPaint()
			rp.Color = types.ColorFromRGB(250, 250, 252)
			cvs.DrawRect(pos.X, ry, W, tableRowH, rp)
		}
		if t.Expandable && e.expanded[r] {
			exTop := ry + tableRowH
			exH := e.expandH[r]
			bp := paint.DefaultPaint()
			bp.Color = types.ColorFromRGB(250, 250, 252)
			cvs.DrawRect(pos.X, exTop, W, exH, bp)
			cvs.DrawLine(pos.X, exTop+exH, pos.X+W, exTop+exH, lp)
		} else {
			cvs.DrawLine(pos.X, ry+tableRowH, pos.X+W, ry+tableRowH, lp)
		}
		if t.Expandable {
			drawExpandArrow(cvs, pos.X+e.expandW/2, ry+tableRowH/2, e.expanded[r])
		}
		if t.Selectable {
			drawTableCheck(cvs, pos.X+e.expandW+e.checkW/2, ry+tableRowH/2, e.selected[r])
		}
	}
	cvs.Restore()

	// body 列内容(横向三批 clip，纵向限 body 区)
	e.paintZonedCols(cvs, pos, W, bodyTop, e.bodyH, func(i int) { e.paintColBody(cvs, i, pos, offset) })

	// body 展开内容 element
	cvs.Save()
	cvs.ClipRect(pos.X, bodyTop, W, e.bodyH)
	for _, el := range e.expandEls {
		el.Paint(cvs, offset)
	}
	cvs.Restore()

	// ── 表头层：列标题 + 排序箭头(横向三批 clip，纵向限表头区) ──
	e.paintZonedCols(cvs, pos, W, pos.Y, tableHeaderH, func(i int) { e.paintColHeader(cvs, i, pos) })
	if t.Selectable {
		drawTableCheck(cvs, pos.X+e.expandW+e.checkW/2, pos.Y+tableHeaderH/2, e.allSelected())
	}
	cvs.DrawLine(pos.X, pos.Y+tableHeaderH, pos.X+W, pos.Y+tableHeaderH, lp)

	// ── 固定列阴影(贯穿表头+body) ──
	shadowH := tableHeaderH + e.bodyH
	if e.scrollX > 0.5 && e.frozenLeftW > leftW {
		e.paintFixedShadow(cvs, pos.X+e.frozenLeftW, pos.Y, shadowH, true)
	}
	if e.scrollX < e.maxScrollX-0.5 && e.rightFixedW > 0 {
		e.paintFixedShadow(cvs, pos.X+W-e.rightFixedW, pos.Y, shadowH, false)
	}

	// ── 滚动条：横(body 底部) + 竖(body 右侧) ──
	if e.maxScrollX > 0 {
		e.paintHBar(cvs, pos, W)
	}
	if e.maxScrollY > 0 {
		e.paintVBar(cvs, pos, W)
	}
}

// paintColHeader 画第 i 列表头：标题 + 排序箭头。
func (e *TableElement) paintColHeader(cvs canvas.Canvas, i int, pos types.Point) {
	col := e.table.Columns[i]
	cw := e.colWidths[i]
	cx := pos.X + e.colX[i]
	hf := canvas.DefaultFont()
	hf.Size = 14
	hf.Weight = canvas.FontWeightBold
	cr := types.Rect{X: cx + tableCellPad, Y: pos.Y, Width: cw - tableCellPad*2, Height: tableHeaderH}
	canvas.DrawTextAligned(cvs, col.Label, cr, hf, elTextRegular(), tableHAlign(col.Align), canvas.VAlignMiddle)
	if col.Sortable {
		e.paintSortArrows(cvs, cx+cw-16, pos.Y+tableHeaderH/2, i)
	}
}

// paintColBody 画第 i 列的所有数据行(文字或 slot)，Y 含 scrollY 偏移。
func (e *TableElement) paintColBody(cvs canvas.Canvas, i int, pos, offset types.Point) {
	col := e.table.Columns[i]
	cw := e.colWidths[i]
	cx := pos.X + e.colX[i]
	bodyTop := pos.Y + tableHeaderH
	if col.Render == nil {
		cf := canvas.DefaultFont()
		cf.Size = 14
		for disp, r := range e.order {
			ry := bodyTop + e.rowTops[disp] - e.scrollY
			dr := types.Rect{X: cx + tableCellPad, Y: ry, Width: cw - tableCellPad*2, Height: tableRowH}
			canvas.DrawTextAligned(cvs, e.table.Data[r][col.Prop], dr, cf, elTextRegular(), tableHAlign(col.Align), canvas.VAlignMiddle)
		}
	} else {
		ncol := len(e.table.Columns)
		for _, r := range e.order {
			if el := e.cellEls[r*ncol+i]; el != nil {
				el.Paint(cvs, offset)
			}
		}
	}
}

func (e *TableElement) paintFixedShadow(cvs canvas.Canvas, x, y, h float64, leftSide bool) {
	for k := 0; k < 6; k++ {
		sp := paint.DefaultPaint()
		a := 36 - k*6
		if a < 0 {
			a = 0
		}
		sp.Color = types.ColorFromRGBA(0, 0, 0, uint8(a))
		var lx float64
		if leftSide {
			lx = x + float64(k)
		} else {
			lx = x - float64(k) - 1
		}
		cvs.DrawRect(lx, y, 1, h, sp)
	}
}

func (e *TableElement) hbarMetrics(W float64) (thumbW, travel, trackX, viewW float64) {
	viewW = W - e.frozenLeftW - e.rightFixedW
	midW := viewW + e.maxScrollX
	if midW <= 0 {
		return 0, 0, e.frozenLeftW, viewW
	}
	thumbW = viewW * viewW / midW
	if thumbW < 20 {
		thumbW = 20
	}
	if thumbW > viewW {
		thumbW = viewW
	}
	travel = viewW - thumbW
	trackX = e.frozenLeftW
	return
}

func (e *TableElement) paintHBar(cvs canvas.Canvas, pos types.Point, W float64) {
	thumbW, travel, trackX, viewW := e.hbarMetrics(W)
	by := pos.Y + tableHeaderH + e.bodyH
	track := paint.DefaultPaint()
	track.Color = types.ColorFromRGB(245, 245, 247)
	cvs.DrawRect(pos.X+trackX, by, viewW, tableHBarH, track)
	ratio := 0.0
	if e.maxScrollX > 0 {
		ratio = e.scrollX / e.maxScrollX
	}
	th := paint.DefaultPaint()
	th.Color = types.ColorFromRGB(193, 193, 193)
	cvs.DrawRoundedRect(pos.X+trackX+ratio*travel, by+2, thumbW, tableHBarH-4, 3, th)
}

func (e *TableElement) vbarMetrics() (thumbH, travel float64) {
	if e.contentH <= 0 {
		return 0, 0
	}
	thumbH = e.bodyH * e.bodyH / e.contentH
	if thumbH < 20 {
		thumbH = 20
	}
	if thumbH > e.bodyH {
		thumbH = e.bodyH
	}
	travel = e.bodyH - thumbH
	return
}

func (e *TableElement) paintVBar(cvs canvas.Canvas, pos types.Point, W float64) {
	thumbH, travel := e.vbarMetrics()
	bx := pos.X + W - tableVBarW
	by := pos.Y + tableHeaderH
	track := paint.DefaultPaint()
	track.Color = types.ColorFromRGB(245, 245, 247)
	cvs.DrawRect(bx, by, tableVBarW, e.bodyH, track)
	ratio := 0.0
	if e.maxScrollY > 0 {
		ratio = e.scrollY / e.maxScrollY
	}
	th := paint.DefaultPaint()
	th.Color = types.ColorFromRGB(193, 193, 193)
	cvs.DrawRoundedRect(bx+2, by+ratio*travel, tableVBarW-4, thumbH, 3, th)
}

func (e *TableElement) paintSortArrows(cvs canvas.Canvas, x, cy float64, col int) {
	up := paint.DefaultPaint()
	up.Color = elPlaceholder()
	down := paint.DefaultPaint()
	down.Color = elPlaceholder()
	if e.sortCol == col && e.sortAsc {
		up.Color = elPrimary()
	}
	if e.sortCol == col && !e.sortAsc {
		down.Color = elPrimary()
	}
	pu := canvas.NewPath()
	pu.MoveTo(x, cy-5)
	pu.LineTo(x-4, cy-1)
	pu.LineTo(x+4, cy-1)
	pu.Close()
	cvs.DrawPath(pu, up)
	pd := canvas.NewPath()
	pd.MoveTo(x, cy+5)
	pd.LineTo(x-4, cy+1)
	pd.LineTo(x+4, cy+1)
	pd.Close()
	cvs.DrawPath(pd, down)
}

// rowAtY 把局部 y 反查为显示行(disp)，落在展开区/表头/滚动条/可视区外返回 -1。
func (e *TableElement) rowAtY(ly float64) int {
	if ly < tableHeaderH || ly > tableHeaderH+e.bodyH {
		return -1
	}
	y := ly - tableHeaderH + e.scrollY
	for disp := range e.order {
		top := e.rowTops[disp]
		if y >= top && y < top+tableRowH {
			return disp
		}
	}
	return -1
}

// colAtX 把局部 x 反查为列索引；中间列须落在可视窗口内。
func (e *TableElement) colAtX(lx float64) int {
	for i := range e.table.Columns {
		if e.colZone[i] == 1 && (lx < e.frozenLeftW || lx > e.size.Width-e.rightFixedW) {
			continue
		}
		cx := e.colX[i]
		if lx >= cx && lx < cx+e.colWidths[i] {
			return i
		}
	}
	return -1
}

func (e *TableElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	lx, ly := me.X-pos.X, me.Y-pos.Y
	switch ev.Type() {
	case event.TypeMouseWheel:
		// Shift+滚轮 / 无竖滚时 → 横滚；否则竖滚
		if (me.Mods&event.ModShift != 0 || e.maxScrollY <= 0) && e.maxScrollX > 0 {
			dx := me.DeltaX
			if dx == 0 {
				dx = me.DeltaY
			}
			if dx != 0 {
				old := e.scrollX
				e.scrollX = clampF(e.scrollX-dx*scrollWheelStep, 0, e.maxScrollX)
				if e.scrollX != old && OnNeedsLayout != nil {
					OnNeedsLayout()
				}
				return true
			}
		}
		if e.maxScrollY > 0 && me.DeltaY != 0 {
			old := e.scrollY
			e.scrollY = clampF(e.scrollY-me.DeltaY*scrollWheelStep, 0, e.maxScrollY)
			if e.scrollY != old && OnNeedsLayout != nil {
				OnNeedsLayout()
			}
			return true
		}
		return false
	case event.TypeMouseMove, event.TypeMouseEnter:
		if e.vbarDrag {
			_, travel := e.vbarMetrics()
			if travel > 0 {
				old := e.scrollY
				e.scrollY = clampF(e.vDragScroll+(me.Y-e.vDragMouseY)/travel*e.maxScrollY, 0, e.maxScrollY)
				if e.scrollY != old && OnNeedsLayout != nil {
					OnNeedsLayout()
				}
			}
			return true
		}
		if e.hbarDrag {
			_, travel, _, _ := e.hbarMetrics(e.size.Width)
			if travel > 0 {
				old := e.scrollX
				e.scrollX = clampF(e.hDragScroll+(me.X-e.hDragMouseX)/travel*e.maxScrollX, 0, e.maxScrollX)
				if e.scrollX != old && OnNeedsLayout != nil {
					OnNeedsLayout()
				}
			}
			return true
		}
		row := e.rowAtY(ly)
		if row != e.hoverRow {
			e.hoverRow = row
			e.MarkNeedsPaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hoverRow = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseUp:
		if e.vbarDrag {
			e.vbarDrag = false
			return true
		}
		if e.hbarDrag {
			e.hbarDrag = false
			return true
		}
	case event.TypeMouseDown:
		// 竖滚条拖动(右侧 body 区)
		if e.maxScrollY > 0 && lx >= e.size.Width-tableVBarW && ly >= tableHeaderH && ly <= tableHeaderH+e.bodyH {
			e.vbarDrag = true
			e.vDragMouseY = me.Y
			e.vDragScroll = e.scrollY
			if RequestPointerCapture != nil {
				RequestPointerCapture(e)
			}
			return true
		}
		// 横滚条拖动(body 底部)
		if e.maxScrollX > 0 && ly >= tableHeaderH+e.bodyH {
			e.hbarDrag = true
			e.hDragMouseX = me.X
			e.hDragScroll = e.scrollX
			if RequestPointerCapture != nil {
				RequestPointerCapture(e)
			}
			return true
		}
		// 表头
		if ly < tableHeaderH {
			if e.table.Selectable && lx >= e.expandW && lx < e.expandW+e.checkW {
				e.toggleAll()
				return true
			}
			if lx >= e.expandW+e.checkW {
				if col := e.colAtX(lx); col >= 0 && e.table.Columns[col].Sortable {
					e.clickSort(col)
				}
			}
			return true
		}
		// 数据行
		disp := e.rowAtY(ly)
		if disp < 0 {
			return false
		}
		r := e.order[disp]
		if e.table.Expandable && lx < e.expandW {
			e.toggleExpand(r)
			return true
		}
		if e.table.Selectable && lx >= e.expandW && lx < e.expandW+e.checkW {
			e.toggleRow(r)
			return true
		}
	}
	return false
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (e *TableElement) clickSort(col int) {
	if e.sortCol == col {
		e.sortAsc = !e.sortAsc
	} else {
		e.sortCol, e.sortAsc = col, true
	}
	e.applySort()
	if e.table.OnSortChange != nil {
		e.table.OnSortChange(e.table.Columns[col].Prop, e.sortAsc)
	}
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	}
}

func (e *TableElement) toggleExpand(r int) {
	e.expanded[r] = !e.expanded[r]
	if e.table.OnExpandChange != nil {
		e.table.OnExpandChange(r, e.expanded[r])
	}
	if OnNeedsLayout != nil {
		OnNeedsLayout()
	}
}

func (e *TableElement) toggleRow(r int) {
	if e.selected[r] {
		delete(e.selected, r)
	} else {
		e.selected[r] = true
	}
	e.fireSelect()
	e.MarkNeedsPaint()
}

func (e *TableElement) toggleAll() {
	if e.allSelected() {
		e.selected = map[int]bool{}
	} else {
		e.selected = map[int]bool{}
		for i := range e.table.Data {
			e.selected[i] = true
		}
	}
	e.fireSelect()
	e.MarkNeedsPaint()
}

func (e *TableElement) fireSelect() {
	if e.table.OnSelectChange == nil {
		return
	}
	var rows []int
	for i := range e.table.Data {
		if e.selected[i] {
			rows = append(rows, i)
		}
	}
	e.table.OnSelectChange(rows)
}

func (e *TableElement) Update(newWidget Widget) {
	if nt, ok := newWidget.(*Table); ok {
		e.table = nt
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
