package widget

import (
	"strings"
	"testing"
)

func cascaderTestOptions() []CascaderOption {
	return []CascaderOption{
		{Label: "浙江", Value: "zj", Children: []CascaderOption{
			{Label: "杭州", Value: "hz", Children: []CascaderOption{
				{Label: "西湖", Value: "xh"},
				{Label: "余杭", Value: "yh"},
			}},
			{Label: "宁波", Value: "nb"},
		}},
		{Label: "江苏", Value: "js"},
	}
}

func initedPanel(value []string) *cascaderPanelElement {
	el := (&cascaderPanel{options: cascaderTestOptions(), value: value}).CreateElement().(*cascaderPanelElement)
	el.ensureInit()
	return el
}

// TestCascaderSelectedLabels 触发器沿 value 解析路径标签。
func TestCascaderSelectedLabels(t *testing.T) {
	c := NewCascader(cascaderTestOptions()).WithValue([]string{"zj", "hz", "xh"})
	if got := strings.Join(c.selectedLabels(), "/"); got != "浙江/杭州/西湖" {
		t.Errorf("selectedLabels=%q want 浙江/杭州/西湖", got)
	}
}

// TestCascaderColumns value 还原后展开的列数与末列内容。
func TestCascaderColumns(t *testing.T) {
	el := initedPanel([]string{"zj", "hz"})
	cols := el.columns()
	if len(cols) != 3 { // 根 + 浙江children + 杭州children
		t.Fatalf("columns=%d want 3", len(cols))
	}
	if len(cols[2]) != 2 { // 西湖/余杭
		t.Errorf("第3列=%d want 2", len(cols[2]))
	}
}

// TestCascaderClickExpand 点有子节点 → 展开下一列，不回填。
func TestCascaderClickExpand(t *testing.T) {
	p := &cascaderPanel{options: cascaderTestOptions()}
	var got []string
	p.onSelect = func(v, l []string) { got = v }
	el := p.CreateElement().(*cascaderPanelElement)
	el.ensureInit()
	el.clickAt(0, 0) // 浙江(有子)
	if got != nil {
		t.Error("点有子节点不应回填")
	}
	if len(el.columns()) != 2 {
		t.Errorf("展开后列数=%d want 2", len(el.columns()))
	}
}

// TestCascaderClickLeaf 点叶子 → 回填整条路径 value/label。
func TestCascaderClickLeaf(t *testing.T) {
	p := &cascaderPanel{options: cascaderTestOptions()}
	var gotV, gotL []string
	p.onSelect = func(v, l []string) { gotV, gotL = v, l }
	el := p.CreateElement().(*cascaderPanelElement)
	el.ensureInit()
	el.clickAt(0, 0) // 浙江
	el.clickAt(1, 0) // 杭州
	el.clickAt(2, 0) // 西湖(叶子)
	if strings.Join(gotV, "/") != "zj/hz/xh" {
		t.Errorf("回填 values=%v want zj/hz/xh", gotV)
	}
	if strings.Join(gotL, "/") != "浙江/杭州/西湖" {
		t.Errorf("回填 labels=%v want 浙江/杭州/西湖", gotL)
	}
}

// TestCascaderClickResetsTail 改点上游列 → 截断后续路径。
func TestCascaderClickResetsTail(t *testing.T) {
	el := initedPanel([]string{"zj", "hz", "xh"})
	if len(el.columns()) != 3 {
		t.Fatalf("初始应展开3列, got %d", len(el.columns()))
	}
	el.clickAt(0, 1) // 江苏(叶子) → activePath 截断为 [1]
	if len(el.activePath) != 1 || el.activePath[0] != 1 {
		t.Errorf("activePath=%v want [1]", el.activePath)
	}
}

// TestCascaderCellAt 局部坐标命中正确的列/行。
func TestCascaderCellAt(t *testing.T) {
	el := initedPanel([]string{"zj", "hz"}) // 3 列
	// 第 1 列(杭州那列) row0
	col, idx := el.cellAt(cascaderColW+10, cascaderPadV+cascaderItemH/2)
	if col != 1 || idx != 0 {
		t.Errorf("cellAt=(%d,%d) want (1,0)", col, idx)
	}
}
