package widget

import (
	"fmt"

	"github.com/user/goui/internal/types"
)

// ─── Transfer 穿梭框（el-transfer 核心，拼装版）──────────────────
// 复用 Checkbox + Button + Div：StatefulWidget 维护左右分配(rightKeys)与勾选(checked)，
// Build 直接组合两个面板(每项一个 Checkbox) + 中间 ›/‹ Button。

type TransferItem struct {
	Key   string
	Label string
}

type Transfer struct {
	StatefulWidget
	Items      []TransferItem
	LeftTitle  string
	RightTitle string
	OnChange   func(rightKeys []string)
}

func NewTransfer(items ...TransferItem) *Transfer {
	return &Transfer{Items: items, LeftTitle: "源列表", RightTitle: "目标列表"}
}
func (t *Transfer) WithTitles(l, r string) *Transfer         { t.LeftTitle, t.RightTitle = l, r; return t }
func (t *Transfer) WithOnChange(fn func([]string)) *Transfer { t.OnChange = fn; return t }

func (t *Transfer) CreateState() State {
	return &transferState{rightKeys: map[string]bool{}, checked: map[string]bool{}}
}

type transferState struct {
	BaseState
	rightKeys map[string]bool
	checked   map[string]bool
}

func (s *transferState) cfg() *Transfer {
	if w, ok := s.Widget().(*Transfer); ok {
		return w
	}
	return &Transfer{}
}

func (s *transferState) sideItems(right bool) []TransferItem {
	var out []TransferItem
	for _, it := range s.cfg().Items {
		if s.rightKeys[it.Key] == right {
			out = append(out, it)
		}
	}
	return out
}

func (s *transferState) countChecked(right bool) int {
	n := 0
	for _, it := range s.cfg().Items {
		if s.rightKeys[it.Key] == right && s.checked[it.Key] {
			n++
		}
	}
	return n
}

// move 把选中项在两侧间移动。
func (s *transferState) move(toRight bool) {
	t := s.cfg()
	for _, it := range t.Items {
		if s.rightKeys[it.Key] != toRight && s.checked[it.Key] {
			s.rightKeys[it.Key] = toRight
			s.checked[it.Key] = false
		}
	}
	if t.OnChange != nil {
		var rk []string
		for _, it := range t.Items {
			if s.rightKeys[it.Key] {
				rk = append(rk, it.Key)
			}
		}
		t.OnChange(rk)
	}
	s.SetState()
}

// panel 一侧面板：边框 + 表头(标题/计数) + 复用 Checkbox 的项列表。
func (s *transferState) panel(title string, items []TransferItem) Widget {
	kids := []interface{}{Style{
		Width: 210, Height: 300, FlexDirection: "column",
		BorderColor: types.ColorRef(220, 223, 230), BorderWidth: 1, BorderRadius: 5,
		BackgroundColor: types.ColorRef(255, 255, 255), Overflow: "hidden",
	}}
	// 表头
	kids = append(kids, Div(Style{
		FlexDirection: "row", JustifyContent: "space-between", AlignItems: "center",
		Padding: types.EdgeInsetsLTRB(12, 0, 12, 0), Height: 40,
		BackgroundColor: types.ColorRef(245, 247, 250),
	},
		NewText(title, elTextPrimary()),
		NewText(fmt.Sprintf("%d 项", len(items)), elTextSecondary()),
	))
	// 列表项 —— 复用 Checkbox
	for _, it := range items {
		key := it.Key
		kids = append(kids, Div(Style{Padding: types.EdgeInsetsLTRB(12, 6, 12, 6)},
			NewCheckbox(it.Label, s.checked[key], func(v bool) {
				s.checked[key] = v
				s.SetState()
			}),
		))
	}
	kids = append(kids, &Spacer{}) // 撑满剩余：强制走 Flex column、内容顶对齐、两面板等高
	return Div(kids...)
}

func (s *transferState) Build(ctx BuildContext) Widget {
	t := s.cfg()
	blue := elPrimary()
	gray := elBorderLight()
	leftCol, rightCol := blue, blue
	if s.countChecked(false) == 0 {
		leftCol = gray
	}
	if s.countChecked(true) == 0 {
		rightCol = gray
	}
	// 中间 ›/‹ —— 复用 Button
	mid := Div(Style{FlexDirection: "column", Gap: 10, AlignItems: "center", JustifyContent: "center", Height: 300, Padding: types.EdgeInsetsLTRB(12, 0, 12, 0)},
		NewButton("›", func() { s.move(true) }).WithColor(leftCol),
		NewButton("‹", func() { s.move(false) }).WithColor(rightCol),
	)
	return Div(Style{FlexDirection: "row", AlignItems: "start"},
		s.panel(t.LeftTitle, s.sideItems(false)),
		mid,
		s.panel(t.RightTitle, s.sideItems(true)),
	)
}
