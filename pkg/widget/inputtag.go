package widget

import (
	"strings"

	"github.com/hoonfeng/goui/pkg/types"
)

// ─── InputTag 标签输入（el-input-tag）──────────────────────────
// 复用 Tag + Input：Input 回车提交(自带清空)→添加 Tag；Tag 的 × 删除。
// StatefulWidget 维护 tags。

type InputTag struct {
	StatefulWidget
	Tags        []string
	Placeholder string
	OnChange    func([]string)
}

func NewInputTag(tags ...string) *InputTag { return &InputTag{Tags: tags} }
func (i *InputTag) WithOnChange(fn func([]string)) *InputTag { i.OnChange = fn; return i }

func (i *InputTag) CreateState() State { return &inputTagState{} }

type inputTagState struct {
	BaseState
	tags   []string
	inited bool
}

func (s *inputTagState) cfg() *InputTag {
	if w, ok := s.Widget().(*InputTag); ok {
		return w
	}
	return &InputTag{}
}

func (s *inputTagState) fire() {
	if fn := s.cfg().OnChange; fn != nil {
		fn(append([]string{}, s.tags...))
	}
}

func (s *inputTagState) add(t string) {
	s.tags = append(s.tags, t)
	s.SetState()
	s.fire()
}

func (s *inputTagState) remove(i int) {
	if i >= 0 && i < len(s.tags) {
		s.tags = append(s.tags[:i], s.tags[i+1:]...)
		s.SetState()
		s.fire()
	}
}

func (s *inputTagState) Build(ctx BuildContext) Widget {
	if !s.inited {
		s.inited = true
		s.tags = append([]string{}, s.cfg().Tags...)
	}
	kids := []interface{}{Style{
		FlexDirection: "row", AlignItems: "center", Gap: 6, Padding: types.EdgeInsets(6),
		BorderColor:     types.ColorRef(220, 223, 230), BorderWidth: 1, BorderRadius: 4,
		BackgroundColor: types.ColorRef(255, 255, 255),
	}}
	for i, t := range s.tags {
		idx := i
		tag := NewTag(t).WithClosable(true)
		tag.OnClose = func() { s.remove(idx) }
		kids = append(kids, tag)
	}
	in := NewInput(orI18n(s.cfg().Placeholder, "el.inputtag.placeholder"), nil)
	in.OnSubmit = func(text string) {
		if t := strings.TrimSpace(text); t != "" {
			s.add(t)
		}
	}
	kids = append(kids, in)
	return Div(kids...)
}
