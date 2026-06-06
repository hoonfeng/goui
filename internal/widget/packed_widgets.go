package widget

import "github.com/user/goui/internal/types"

// ─── 快速拼装组件（展示/布局类，Build 直接组合现有 widget，省去自绘）──
// 注：Card/Divider/Tree(treeview)/Image 等已存在，此处只补缺失的。

// ── Space 间距容器（一行/一列等距排列子项）──
type Space struct {
	StatelessWidget
	Children []Widget
	Size     float64
	Vertical bool
}

func NewSpace(size float64, children ...Widget) *Space { return &Space{Children: children, Size: size} }
func (s *Space) Vert() *Space                          { s.Vertical = true; return s }

func (s *Space) Build(ctx BuildContext) Widget {
	dir := "row"
	if s.Vertical {
		dir = "column"
	}
	args := []interface{}{Style{FlexDirection: dir, Gap: s.Size, AlignItems: "center"}}
	for _, c := range s.Children {
		args = append(args, c)
	}
	return Div(args...)
}

// ── Statistic 统计数值（标题 + 大号数值 + 可选后缀）──
type Statistic struct {
	StatelessWidget
	Title  string
	Value  string
	Suffix string
}

func NewStatistic(title, value string) *Statistic   { return &Statistic{Title: title, Value: value} }
func (s *Statistic) WithSuffix(x string) *Statistic { s.Suffix = x; return s }

func (s *Statistic) Build(ctx BuildContext) Widget {
	val := s.Value
	if s.Suffix != "" {
		val += " " + s.Suffix
	}
	return Div(Style{FlexDirection: "column", Gap: 2},
		NewText(s.Title, elTextSecondary()),
		H2(val),
	)
}

// ── Result 结果页（状态图标 + 标题 + 副标题 + 操作）──
type Result struct {
	StatelessWidget
	Status   string // success(默认)/error/warning/info
	Title    string
	SubTitle string
	Extra    Widget
}

func NewResult(status, title string) *Result    { return &Result{Status: status, Title: title} }
func (r *Result) WithSubTitle(s string) *Result { r.SubTitle = s; return r }
func (r *Result) WithExtra(w Widget) *Result    { r.Extra = w; return r }

func resultIcon(status string) string {
	switch status {
	case "error":
		return "❌"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	default:
		return "✅"
	}
}

func (r *Result) Build(ctx BuildContext) Widget {
	args := []interface{}{Style{FlexDirection: "column", AlignItems: "center", Gap: 8, Padding: types.EdgeInsets(28)}}
	args = append(args, H1(resultIcon(r.Status)), H3(r.Title))
	if r.SubTitle != "" {
		args = append(args, NewText(r.SubTitle, elTextSecondary()))
	}
	if r.Extra != nil {
		args = append(args, r.Extra)
	}
	return Div(args...)
}

// ── Descriptions 描述列表（label-value 行 + 分隔线 + 边框）──
type DescItem struct {
	Label string
	Value string
}

type Descriptions struct {
	StatelessWidget
	Title string
	Items []DescItem
}

func NewDescriptions(items ...DescItem) *Descriptions    { return &Descriptions{Items: items} }
func (d *Descriptions) WithTitle(t string) *Descriptions { d.Title = t; return d }

func (d *Descriptions) Build(ctx BuildContext) Widget {
	args := []interface{}{Style{
		FlexDirection:   "column",
		BorderColor:     types.ColorRef(235, 238, 245),
		BorderWidth:     1,
		BorderRadius:    4,
		BackgroundColor: types.ColorRef(255, 255, 255),
	}}
	if d.Title != "" {
		args = append(args,
			Div(Style{Padding: types.EdgeInsetsLTRB(16, 12, 16, 12)}, H3(d.Title)),
			Div(Style{Height: 1, BackgroundColor: types.ColorRef(235, 238, 245)}))
	}
	for i, it := range d.Items {
		args = append(args, Div(Style{FlexDirection: "row", Padding: types.EdgeInsetsLTRB(16, 11, 16, 11), Gap: 12},
			Div(Style{Width: 110}, NewText(it.Label, elTextSecondary())),
			NewText(it.Value, elTextPrimary()),
		))
		if i < len(d.Items)-1 {
			args = append(args, Div(Style{Height: 1, BackgroundColor: types.ColorRef(235, 238, 245)}))
		}
	}
	return Div(args...)
}

// ── Timeline 时间线（节点圆点 + 竖线 + 内容/时间）──
type TimelineItem struct {
	Time    string
	Content string
	Color   types.Color // 零值→主色
}

type Timeline struct {
	StatelessWidget
	Items []TimelineItem
}

func NewTimeline(items ...TimelineItem) *Timeline { return &Timeline{Items: items} }

func (t *Timeline) Build(ctx BuildContext) Widget {
	args := []interface{}{Style{FlexDirection: "column"}}
	for i, it := range t.Items {
		col := it.Color
		if col.A == 0 {
			col = elPrimary()
		}
		vlineH := 36.0
		if i == len(t.Items)-1 {
			vlineH = 0
		}
		left := Div(Style{FlexDirection: "column", AlignItems: "center", Width: 16, Gap: 2},
			Div(Style{Width: 10, Height: 10, BorderRadius: 5, BackgroundColor: &col}),
			Div(Style{Width: 2, Height: vlineH, BackgroundColor: types.ColorRef(228, 231, 237)}),
		)
		content := Div(Style{FlexDirection: "column", Gap: 2, Padding: types.EdgeInsetsLTRB(0, 0, 0, 18)},
			NewText(it.Content, elTextPrimary()),
			Small(it.Time),
		)
		args = append(args, Div(Style{FlexDirection: "row", Gap: 12}, left, content))
	}
	return Div(args...)
}

// ── Skeleton 骨架屏（灰条占位）──
type Skeleton struct {
	StatelessWidget
	Rows int
}

func NewSkeleton(rows int) *Skeleton { return &Skeleton{Rows: rows} }

func (s *Skeleton) Build(ctx BuildContext) Widget {
	n := s.Rows
	if n <= 0 {
		n = 3
	}
	args := []interface{}{Style{FlexDirection: "column", Gap: 12}}
	for i := 0; i < n; i++ {
		w := 0.0 // 0=自动撑满
		if i == n-1 {
			w = 180 // 末行短
		}
		args = append(args, Div(Style{Width: w, Height: 16, BorderRadius: 4, BackgroundColor: types.ColorRef(240, 242, 245)}))
	}
	return Div(args...)
}

