package widget

import "github.com/hoonfeng/goui/pkg/types"

// ─── PageHeader 页头（el-page-header，拼装）──────────────────────
// 返回链接 + 竖分隔 + 标题/副标题 + 右侧操作槽。复用 Link / H3 / Text / Div。

type PageHeader struct {
	StatelessWidget
	Title    string
	SubTitle string
	OnBack   func()
	Extra    Widget
}

func NewPageHeader(title string) *PageHeader            { return &PageHeader{Title: title} }
func (p *PageHeader) WithSubTitle(s string) *PageHeader { p.SubTitle = s; return p }
func (p *PageHeader) WithOnBack(fn func()) *PageHeader  { p.OnBack = fn; return p }
func (p *PageHeader) WithExtra(w Widget) *PageHeader    { p.Extra = w; return p }

func (p *PageHeader) Build(ctx BuildContext) Widget {
	left := []interface{}{Style{FlexDirection: "row", AlignItems: "center", Gap: 12}}
	if p.OnBack != nil {
		left = append(left,
			NewLink("← 返回", p.OnBack),
			Div(Style{Width: 1, Height: 16, BackgroundColor: types.ColorRef(220, 223, 230)}))
	}
	left = append(left, H3(p.Title))
	if p.SubTitle != "" {
		left = append(left, NewText(p.SubTitle, elTextSecondary()))
	}
	leftBox := Div(left...)

	if p.Extra != nil {
		return Div(Style{FlexDirection: "row", JustifyContent: "space-between", AlignItems: "center", Padding: types.EdgeInsets(10)},
			leftBox, p.Extra)
	}
	return Div(Style{Padding: types.EdgeInsets(10)}, leftBox)
}
