package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
)

func layoutTable(t *Table, w float64) *TableElement {
	el := t.CreateElement().(*TableElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: w, MaxHeight: 1000}})
	return el
}

// TestTableColWidths 固定列优先，剩余平分给弹性列。
func TestTableColWidths(t *testing.T) {
	el := layoutTable(NewTable([]TableColumn{
		{Label: "A", Prop: "a", Width: 100},
		{Label: "B", Prop: "b"},
		{Label: "C", Prop: "c"},
	}, nil), 500)
	if el.colWidths[0] != 100 {
		t.Errorf("固定列宽=%v, want 100", el.colWidths[0])
	}
	if el.colWidths[1] != 200 || el.colWidths[2] != 200 {
		t.Errorf("弹性列宽=%v,%v, want 200,200(剩余400平分)", el.colWidths[1], el.colWidths[2])
	}
}

// TestTableHeight 高度 = 表头 + 行数×行高。
func TestTableHeight(t *testing.T) {
	el := layoutTable(NewTable([]TableColumn{{Label: "A", Prop: "a"}},
		[]map[string]string{{"a": "1"}, {"a": "2"}}), 300)
	want := tableHeaderH + 2*tableRowH
	if el.size.Height != want {
		t.Errorf("height=%v, want %v(表头+2行)", el.size.Height, want)
	}
}

// TestTableHoverRow 鼠标移到第 1 行(0-indexed)高亮该行。
func TestTableHoverRow(t *testing.T) {
	el := layoutTable(NewTable([]TableColumn{{Label: "A", Prop: "a"}},
		[]map[string]string{{"a": "1"}, {"a": "2"}, {"a": "3"}}), 300)
	y := tableHeaderH + tableRowH + 10
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseMove, 50, y, event.ButtonNone, event.ModNone))
	if el.hoverRow != 1 {
		t.Errorf("hoverRow=%d, want 1", el.hoverRow)
	}
}
