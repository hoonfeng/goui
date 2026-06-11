package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/types"
)

type testRebuildRoot struct {
	StatefulWidget
	st *testRebuildState
}

func (r *testRebuildRoot) CreateState() State {
	r.st = &testRebuildState{}
	return r.st
}

type testRebuildState struct {
	BaseState
	label string
}

func (s *testRebuildState) Build(ctx BuildContext) Widget {
	return NewText(s.label, types.ColorBlack)
}

// TestBumpRebuildForcesRebuild 验证全局强制重建（语言切换场景）能绕过脏缓存，
// 让未 SetState 的子树也按最新数据重建。
func TestBumpRebuildForcesRebuild(t *testing.T) {
	root := &testRebuildRoot{}
	el := CreateElementFor(root).(*StatefulElement)
	el.Mount(nil, 0)
	root.st.label = "A"
	buildTreeRec(el)

	text := func() string {
		if len(el.Children()) == 0 {
			return ""
		}
		te, ok := el.Children()[0].(*TextElement)
		if !ok {
			return ""
		}
		return te.text.Text
	}
	if text() != "A" {
		t.Fatalf("initial=%q, want A", text())
	}

	// 改数据但不 SetState、不 BumpRebuild → 脏缓存生效，仍是旧值
	root.st.label = "B"
	buildTreeRec(el)
	if text() != "A" {
		t.Errorf("without bump=%q, want cached A", text())
	}

	// 全局强制重建（模拟语言切换）→ 刷新为新值
	BumpRebuild()
	buildTreeRec(el)
	if text() != "B" {
		t.Errorf("after bump=%q, want B", text())
	}
}
