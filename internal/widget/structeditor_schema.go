package widget

// StructEditor 的「结构配置」层：表格有哪些表、每表哪些列、列标题/宽度/类型，全由 SESchema 数据驱动，
// 不再硬编码列索引。这是表格框架「去语言化」的第①层——接新语言只需给一份 Schema（见 LanguageProvider）。

// SEVar 字段标识（schema 列 Field 绑定到哪个字段）。导出供外部自定义 schema 用。
const (
	SEFieldName  = "name"
	SEFieldType  = "type"
	SEFieldArray = "array"
	SEFieldRef   = "ref"
	SEFieldNote  = "note"
)

// SECol 表格一列的配置。
type SECol struct {
	Title  string  // 表头文字
	Field  string  // 绑定 SEVar 的字段（seField*）
	Weight float64 // 列宽权重（相对，按内容区宽分配）
	Check  bool    // true=复选框列（如参考/传址），点击切换 是/空，不进文本编辑
}

// SESchema 表格编辑器的结构配置：四类表各自的列定义。不同语言/场景给不同 schema。
type SESchema struct {
	Globals []SECol // 程序集变量表
	Params  []SECol // 参数表
	Returns []SECol // 返回值表
	Locals  []SECol // 局部变量表
}

// field 按字段标识读 SEVar 的值。
func (v *SEVar) field(f string) string {
	switch f {
	case SEFieldName:
		return v.Name
	case SEFieldType:
		return v.Type
	case SEFieldArray:
		return v.Array
	case SEFieldRef:
		return v.Ref
	case SEFieldNote:
		return v.Note
	}
	return ""
}

// setField 按字段标识写 SEVar 的值。
func (v *SEVar) setField(f, val string) {
	switch f {
	case SEFieldName:
		v.Name = val
	case SEFieldType:
		v.Type = val
	case SEFieldArray:
		v.Array = val
	case SEFieldRef:
		v.Ref = val
	case SEFieldNote:
		v.Note = val
	}
}

// DefaultSchema 默认表结构（与历史硬编码列完全一致：名称|类型|数组|(参考)|备注）。
// 易语言、Go 都用它；接列结构不同的语言时给自定义 Schema。
func DefaultSchema() *SESchema {
	return &SESchema{
		Globals: []SECol{
			{Title: "程序集变量", Field: SEFieldName, Weight: 0.22},
			{Title: "类型", Field: SEFieldType, Weight: 0.18},
			{Title: "初始值", Field: SEFieldRef, Weight: 0.20}, // Ref 列存初始值（全局变量无传址语义）
			{Title: "数组", Field: SEFieldArray, Weight: 0.10},
			{Title: "备注", Field: SEFieldNote, Weight: 0.30},
		},
		Params: []SECol{
			{Title: "参数", Field: SEFieldName, Weight: 0.22},
			{Title: "类型", Field: SEFieldType, Weight: 0.18},
			{Title: "数组", Field: SEFieldArray, Weight: 0.10},
			{Title: "参考", Field: SEFieldRef, Weight: 0.10, Check: true},
			{Title: "备注", Field: SEFieldNote, Weight: 0.40},
		},
		Returns: []SECol{
			{Title: "返回值", Field: SEFieldName, Weight: 0.26},
			{Title: "类型", Field: SEFieldType, Weight: 0.30},
			{Title: "备注", Field: SEFieldNote, Weight: 0.44},
		},
		Locals: []SECol{
			{Title: "局部变量", Field: SEFieldName, Weight: 0.26},
			{Title: "类型", Field: SEFieldType, Weight: 0.22},
			{Title: "数组", Field: SEFieldArray, Weight: 0.12},
			{Title: "备注", Field: SEFieldNote, Weight: 0.40},
		},
	}
}

// colWidths 把列权重换算成实际像素宽（按内容区总宽 tw 分配）。
func colWidths(cols []SECol, tw float64) []float64 {
	cw := make([]float64, len(cols))
	for i := range cols {
		cw[i] = tw * cols[i].Weight
	}
	return cw
}

// colEdges 列累计边界（不含最后一列右边，供画列分隔线，与单元格边界严格一致）。
func colEdges(cw []float64) []float64 {
	edges := make([]float64, 0, len(cw))
	acc := 0.0
	for i := 0; i < len(cw)-1; i++ {
		acc += cw[i]
		edges = append(edges, acc)
	}
	return edges
}
