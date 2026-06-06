package widget

import "strings"

// ─── DateTimePicker 日期时间（el-datetime-picker，组合）─────────
// 拼装 DatePicker(日期) + TimeSelect(时间)，StatefulWidget 维护两段、
// 任一变化即合成 "YYYY-MM-DD HH:MM" 回调。

type DateTimePicker struct {
	StatefulWidget
	Value    string // "2024-06-15 09:00"
	OnChange func(string)
}

func NewDateTimePicker() *DateTimePicker                              { return &DateTimePicker{} }
func (d *DateTimePicker) WithValue(v string) *DateTimePicker          { d.Value = v; return d }
func (d *DateTimePicker) WithOnChange(fn func(string)) *DateTimePicker { d.OnChange = fn; return d }

func (d *DateTimePicker) CreateState() State { return &dateTimeState{} }

type dateTimeState struct {
	BaseState
	date   string
	time   string
	inited bool
}

func (s *dateTimeState) cfg() *DateTimePicker {
	if w, ok := s.Widget().(*DateTimePicker); ok {
		return w
	}
	return &DateTimePicker{}
}

func (s *dateTimeState) fire() {
	v := strings.TrimSpace(s.date + " " + s.time)
	if fn := s.cfg().OnChange; fn != nil {
		fn(v)
	}
}

func (s *dateTimeState) Build(ctx BuildContext) Widget {
	if !s.inited {
		s.inited = true
		parts := strings.SplitN(s.cfg().Value, " ", 2)
		if len(parts) >= 1 {
			s.date = parts[0]
		}
		if len(parts) == 2 {
			s.time = parts[1]
		}
	}
	return Div(Style{FlexDirection: "row", Gap: 10, AlignItems: "center"},
		NewDatePicker().WithValue(s.date).WithPlaceholder("日期").WithOnChange(func(d string) {
			s.date = d
			s.SetState()
			s.fire()
		}),
		NewTimeSelect().WithRange("00:00", "23:30", "00:30").WithValue(s.time).WithOnChange(func(t string) {
			s.time = t
			s.SetState()
			s.fire()
		}),
	)
}
