package widget

import "github.com/hoonfeng/goui/pkg/types"

// ─── Anchor 锚点（el-anchor 侧边导航，拼装版）────────────────────
// 复用 Link 拼装链接列表 + 当前项左侧蓝竖条高亮。点击触发 OnClick(href)。
// 滚动联动(随滚动自动高亮)为进阶；当前 Active 由外部传入。

type AnchorLink struct {
	Title string
	Href  string
}

type Anchor struct {
	StatelessWidget
	Links   []AnchorLink
	Active  string
	OnClick func(href string)
}

func NewAnchor(links ...AnchorLink) *Anchor           { return &Anchor{Links: links} }
func (a *Anchor) WithActive(href string) *Anchor      { a.Active = href; return a }
func (a *Anchor) WithOnClick(fn func(string)) *Anchor { a.OnClick = fn; return a }

func (a *Anchor) Build(ctx BuildContext) Widget {
	rows := []interface{}{Style{FlexDirection: "column", Gap: 2, Padding: types.EdgeInsetsLTRB(2, 4, 4, 4)}}
	for _, l := range a.Links {
		href := l.Href
		bar := types.ColorRef(228, 231, 237)
		if href == a.Active {
			bar = types.ColorRef(64, 158, 255)
		}
		rows = append(rows, Div(Style{FlexDirection: "row", AlignItems: "center", Gap: 12, Padding: types.EdgeInsetsLTRB(0, 5, 0, 5)},
			Div(Style{Width: 2, Height: 18, BackgroundColor: bar}),
			NewLink(l.Title, func() {
				if a.OnClick != nil {
					a.OnClick(href)
				}
			}),
		))
	}
	return Div(rows...)
}
