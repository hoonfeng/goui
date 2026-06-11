package widget

import (
	"strings"
	"testing"
)

// TestDeclarativeCoverage 验证扩展批的所有组件 type 都已注册（声明式不再返回 [unknown] 占位）。
func TestDeclarativeCoverage(t *testing.T) {
	extTypes := []string{
		// 批 A 展示
		"Space", "Statistic", "Result", "Descriptions", "Timeline", "Skeleton",
		// 批 B 容器
		"Loading", "ScrollView", "Splitter", "Watermark", "PageHeader", "Stack", "Positioned", "Carousel", "Dialog", "Drawer",
		// 批 C item-list
		"Tabs", "Collapse", "Steps", "Breadcrumb", "Dropdown", "Anchor", "Menu", "Table", "Transfer",
		// 批 D 表单
		"DatePicker", "TimePicker", "TimeSelect", "ColorPicker", "Autocomplete", "Mention", "InputTag", "Upload",
		"Cascader", "SelectV2", "TreeSelect", "TreeV2", "Form", "Textarea", "Markdown",
		// 批 E 收尾
		"DateTimePicker", "Calendar", "ConfigProvider", "Affix", "RichText", "CodeEditor",
		// 代码表格化编辑器（内容即源码，能声明式）
		"CodeWorkbench", "StructEditor",
	}
	for _, ty := range extTypes {
		w := BuildFromSpec(ComponentSpec{Type: ty}, nil)
		if w == nil {
			t.Errorf("%s: 构建为 nil", ty)
			continue
		}
		if txt, ok := w.(*Text); ok && strings.HasPrefix(txt.Text, "[unknown") {
			t.Errorf("%s: 未注册声明式（得到 %q）", ty, txt.Text)
		}
	}
}

// TestDeclarativeItemList 验证 item-list 组件能从 props 数组正确解析子项。
func TestDeclarativeItemList(t *testing.T) {
	// Table：columns + data
	w := BuildFromSpec(ComponentSpec{Type: "Table", Props: map[string]any{
		"columns": []any{
			map[string]any{"label": "姓名", "prop": "name"},
			map[string]any{"label": "年龄", "prop": "age", "width": 80.0},
		},
		"data": []any{
			map[string]any{"name": "张三", "age": 18},
		},
	}}, nil)
	tbl, ok := w.(*Table)
	if !ok {
		t.Fatalf("Table 类型错误：%T", w)
	}
	if len(tbl.Columns) != 2 || tbl.Columns[0].Label != "姓名" || tbl.Columns[1].Prop != "age" {
		t.Errorf("Table 列解析错：%+v", tbl.Columns)
	}
	if len(tbl.Data) != 1 || tbl.Data[0]["name"] != "张三" || tbl.Data[0]["age"] != "18" {
		t.Errorf("Table 数据解析错：%+v", tbl.Data)
	}

	// Steps：嵌套 items
	w2 := BuildFromSpec(ComponentSpec{Type: "Steps", Props: map[string]any{
		"active": 1.0,
		"items": []any{
			map[string]any{"title": "第一步", "description": "描述1"},
			map[string]any{"title": "第二步"},
		},
	}}, nil)
	st := w2.(*Steps)
	if len(st.Steps) != 2 || st.Steps[0].Title != "第一步" || st.Active != 1 {
		t.Errorf("Steps 解析错：%+v active=%d", st.Steps, st.Active)
	}
}
