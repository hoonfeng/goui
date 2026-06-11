package widget

import (
	"testing"

	"github.com/hoonfeng/goui/internal/layout"
)

// TestStepsLayout 步骤条固定高度、保留 Active。
func TestStepsLayout(t *testing.T) {
	el := NewSteps(
		Step{Title: "A", Description: "第一步"},
		Step{Title: "B", Description: "第二步"},
		Step{Title: "C"},
	).WithActive(1).CreateElement().(*StepsElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 400, MaxHeight: 100}})
	if el.size.Height != stepsH {
		t.Errorf("height=%.0f, want %.0f", el.size.Height, stepsH)
	}
	if el.steps.Active != 1 {
		t.Errorf("Active=%d, want 1", el.steps.Active)
	}
}
