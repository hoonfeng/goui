package widget

import (
	"fmt"
	"strconv"
	"strings"
)

// ─── TimeSelect 时间选择（el-time-select，复用 Select）──────────
// 按 起始/结束/步长 生成时间下拉项，复用 Select 渲染。

type TimeSelect struct {
	StatelessWidget
	Start       string // "09:00"
	End         string // "18:00"
	Step        string // "00:30"
	Value       string
	Placeholder string
	OnChange    func(string)
	Width       float64
}

func NewTimeSelect() *TimeSelect {
	return &TimeSelect{Start: "09:00", End: "18:00", Step: "00:30", Width: 180}
}
func (t *TimeSelect) WithRange(start, end, step string) *TimeSelect {
	t.Start, t.End, t.Step = start, end, step
	return t
}
func (t *TimeSelect) WithValue(v string) *TimeSelect          { t.Value = v; return t }
func (t *TimeSelect) WithOnChange(fn func(string)) *TimeSelect { t.OnChange = fn; return t }

func (t *TimeSelect) Build(ctx BuildContext) Widget {
	return NewSelect(genTimeOptions(t.Start, t.End, t.Step)).
		WithValue(t.Value).WithPlaceholder(orI18n(t.Placeholder, "el.datepicker.selectTime")).WithWidth(t.Width).WithOnChanged(t.OnChange)
}

func genTimeOptions(start, end, step string) []SelectOption {
	s, e, st := hm2min(start), hm2min(end), hm2min(step)
	if st <= 0 {
		st = 30
	}
	var opts []SelectOption
	for m := s; m <= e; m += st {
		v := fmt.Sprintf("%02d:%02d", m/60, m%60)
		opts = append(opts, SelectOption{Label: v, Value: v})
	}
	return opts
}

func hm2min(s string) int {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}
