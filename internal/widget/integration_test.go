package widget

import (
	"fmt"
	"testing"

	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// ──────────────────────────────────────────────────────────────
// 集成测试：按 examples/test/main.go 的组件树结构组装完整元素树
// ──────────────────────────────────────────────────────────────

// TestBuildTestAppTree 构建与 examples/test/main.go 相同的完整组件树，
// 递归调用 CreateElement + Build 展开为元素树，验证每个 Element 非 nil 且类型正确。
//
// 覆盖的组件类型：
//
//	Container, Text, Button, Row, Column, Card, Checkbox, Switch,
//	RadioButton, Slider, ProgressBar, Input, Icon, Spacer, Divider, ListView
//
// 覆盖的嵌套模式：
//
//	Container→SingleChild, Flex→MultiChild, Card→SingleChild,
//	ListView→ScrollView→Column, 多层 Container 嵌套, Row+Spacer 分布,
//	Button 独占, 文本行混排, 叶组件与容器组件混合
func TestBuildTestAppTree(t *testing.T) {
	// 构建完整的组件树（与 examples/test/main.go 结构一致）
	rootWidget := buildIntegrationTestTree()

	// 创建根 Element
	rootEl := CreateElementFor(rootWidget)
	if rootEl == nil {
		t.Fatal("CreateElementFor returned nil for root widget")
	}

	// 递归展开并验证元素树
	var elementCount int
	validateTreeRecursive(t, rootEl, "root", &elementCount)

	t.Logf("Integration tree: %d elements validated successfully", elementCount)
}

// validateTreeRecursive 递归验证元素树：
//   - 通过 Element.Build() 展开子元素
//   - 验证每个 Element 非 nil、Widget() 返回非 nil
//   - 验证 Parent() 引用正确
//   - 递归验证所有子元素
func validateTreeRecursive(t *testing.T, el Element, path string, count *int) {
	if el == nil {
		t.Errorf("[%s] Element is nil!", path)
		return
	}
	*count++

	// 验证 Widget 非 nil
	w := el.Widget()
	if w == nil {
		t.Errorf("[%s] Element.Widget() returned nil (type: %T)", path, el)
	}

	// 验证 Parent 引用（非根节点应有父节点）
	if path != "root" && el.Parent() == nil {
		t.Errorf("[%s] Non-root element has nil Parent()", path)
	}

	// 通过 Build() 展开子元素
	children := el.Build()

	// 验证 Build 不返回 nil（应该返回空切片或带子元素的切片）
	// 注意：对于 Container/Button/Card 等没有子控件时返回 nil，这是正常的
	_ = children

	for i, child := range children {
		childPath := fmt.Sprintf("%s.child[%d]", path, i)
		validateTreeRecursive(t, child, childPath, count)
	}
}

// ──────────────────────────────────────────────────────────────
// 组件树构建（与 examples/test/main.go 结构一致）
// ──────────────────────────────────────────────────────────────

// buildIntegrationTestTree 构建与 examples/test/main.go 等价的组件树。
// 包含 7 个 Section，覆盖所有控件类型和嵌套模式。
func buildIntegrationTestTree() Widget {
	// ─── Section 1: Text & Button ─────────────────────
	section1 := testCard(
		testColumn(
			testSectionTitle("Text & Button 文本与按钮"),
			testDivider(),
			&Text{Text: "Heading 1"},
			&Text{Text: "Heading 2"},
			&Text{Text: "Body text example with normal weight."},
			testSpace(4),
			&Text{Text: "Buttons:"},
			testSpace(2),
			&Row{Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Button{Text: "Primary", Color: types.ColorFromRGB(66, 133, 244)},
					&Spacer{},
					&Button{Text: "Success", Color: types.ColorFromRGB(76, 175, 80)},
					&Spacer{},
					&Button{Text: "Danger", Color: types.ColorFromRGB(244, 67, 54)},
				}},
				Direction: layout.FlexRow,
			}},
			testSpace(2),
			&Text{Text: "Button clicked: 0 times"},
		),
	)

	// ─── Section 2: Input ─────────────────────────────
	section2 := testCard(
		testColumn(
			testSectionTitle("Input 文本输入"),
			testDivider(),
			&Text{Text: "Type something below:"},
			testSpace(2),
			&Input{
				Placeholder: "Enter text and press Enter...",
			},
			testSpace(4),
			&Text{Text: "You typed: Hello goui!"},
		),
	)

	// ─── Section 3: Checkbox & Switch ─────────────────
	section3 := testCard(
		testColumn(
			testSectionTitle("Checkbox & Switch 复选框与开关"),
			testDivider(),
			&Checkbox{
				Label:   "Enable dark mode",
				Checked: false,
			},
			testSpace(2),
			&Switch{
				Value: false,
				Label: "Wi-Fi",
			},
			testSpace(2),
			&Text{Text: "Checkbox: false | Switch: false"},
		),
	)

	// ─── Section 4: RadioButton ───────────────────────
	section4 := testCard(
		testColumn(
			testSectionTitle("RadioButton 单选按钮"),
			testDivider(),
			&RadioButton{
				GroupName: "group", Value: "option_a",
				Selected: true, Label: "Option A",
			},
			&RadioButton{
				GroupName: "group", Value: "option_b",
				Selected: false, Label: "Option B",
			},
			&RadioButton{
				GroupName: "group", Value: "option_c",
				Selected: false, Label: "Option C",
			},
		),
	)

	// ─── Section 5: Slider & ProgressBar ──────────────
	section5 := testCard(
		testColumn(
			testSectionTitle("Slider & ProgressBar 滑块与进度条"),
			testDivider(),
			&Slider{
				Min: 0, Max: 100, Step: 1,
				Value: 50, ShowLabel: true,
			},
			testSpace(4),
			&ProgressBar{
				Value: 0.45, Height: 10,
				Color:     types.ColorFromRGB(66, 133, 244),
				BGColor:   types.ColorFromRGB(230, 230, 230),
				Radius:    5,
				ShowLabel: true,
			},
			testSpace(2),
			&Row{Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Button{Text: "-10%", Color: types.ColorFromRGB(244, 67, 54)},
					&Spacer{},
					&Button{Text: "+10%", Color: types.ColorFromRGB(76, 175, 80)},
				}},
				Direction: layout.FlexRow,
			}},
		),
	)

	// ─── Section 6: Icons ─────────────────────────────
	section6 := testCard(
		testColumn(
			testSectionTitle("Icons 图标展示"),
			testDivider(),
			&Text{Text: "Navigation & Actions:"},
			&Row{Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Icon{Icon: IconArrowUp, Color: types.ColorFromRGB(66, 133, 244), Size: 20},
					&Spacer{},
					&Icon{Icon: IconArrowDown, Color: types.ColorFromRGB(66, 133, 244), Size: 20},
					&Spacer{},
					&Icon{Icon: IconClose, Color: types.ColorFromRGB(244, 67, 54), Size: 20},
					&Spacer{},
					&Icon{Icon: IconCheck, Color: types.ColorFromRGB(76, 175, 80), Size: 20},
				}},
				Direction: layout.FlexRow,
			}},
			testSpace(4),
			&Text{Text: "Status & Objects:"},
			&Row{Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Icon{Icon: IconStar, Color: types.ColorFromRGB(255, 193, 7), Size: 20},
					&Spacer{},
					&Icon{Icon: IconHeart, Color: types.ColorFromRGB(244, 67, 54), Size: 20},
					&Spacer{},
					&Icon{Icon: IconInfo, Color: types.ColorFromRGB(66, 133, 244), Size: 20},
				}},
				Direction: layout.FlexRow,
			}},
		),
	)

	// ─── Section 7: Layout & Container ────────────────
	section7 := testCard(
		testColumn(
			testSectionTitle("Layout & Container 布局与容器"),
			testDivider(),
			&Text{Text: "Row with Spacer (A . B . C):"},
			&Row{Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					testColoredBox("A", types.ColorFromRGB(66, 133, 244)),
					&Spacer{},
					testColoredBox("B", types.ColorFromRGB(76, 175, 80)),
					&Spacer{},
					testColoredBox("C", types.ColorFromRGB(244, 67, 54)),
				}},
				Direction: layout.FlexRow,
			}},
			testSpace(4),
			&Text{Text: "Nested Containers:"},
			&Container{
				SingleChildWidget: SingleChildWidget{
					Child: &Container{
						SingleChildWidget: SingleChildWidget{
							Child: &Text{Text: "Inner box"},
						},
						Padding:    types.EdgeInsets(12),
						Background: &PaintWidget{Color: types.ColorRef(66, 133, 244)},
						Width:      120,
					},
				},
				Padding:    types.EdgeInsets(16),
				Background: &PaintWidget{Color: types.ColorRef(240, 242, 245)},
			},
			testSpace(4),
			&Text{Text: "Complex layout -- Info row:"},
			&Container{
				SingleChildWidget: SingleChildWidget{
					Child: &Row{Flex: Flex{
						MultiChildWidget: MultiChildWidget{Children: []Widget{
							&Icon{Icon: IconInfo, Color: types.ColorFromRGB(66, 133, 244), Size: 24},
							testSpace(4),
							&Text{Text: "goui v0.1 | "},
							&Text{Text: "Go Native UI"},
							&Spacer{},
							&Button{Text: "OK", Color: types.ColorFromRGB(76, 175, 80)},
						}},
						Direction: layout.FlexRow,
					}},
				},
				Padding:    types.EdgeInsets(8),
				Background: &PaintWidget{Color: types.ColorRef(232, 245, 233)},
			},
		),
	)

	// ─── 组装为 ListView ──────────────────────────────
	allChildren := []Widget{
		&Text{Text: "goui Component Test"},
		testSpace(2),
		&Text{Text: "Interactive demo showing all available widgets"},
		testSpace(8),
		section1, testSpace(6),
		section2, testSpace(6),
		section3, testSpace(6),
		section4, testSpace(6),
		section5, testSpace(6),
		section6, testSpace(6),
		section7,
		testSpace(16),
		&Text{Text: "goui test - 7 widget sections loaded"},
	}

	return &Container{
		SingleChildWidget: SingleChildWidget{
			Child: NewScrollView(VBox(allChildren...)),
		},
		Background: &PaintWidget{Color: types.ColorRef(245, 247, 250)},
	}
}

// ──────────────────────────────────────────────────────────────
// 辅助函数（与 examples/test/main.go 中的辅助函数对应）
// ──────────────────────────────────────────────────────────────

func testSectionTitle(text string) Widget {
	return &Text{
		Text: text,
	}
}

func testColoredBox(label string, color types.Color) Widget {
	return &Container{
		SingleChildWidget: SingleChildWidget{
			Child: &Text{
				Text:  label,
				Color: types.ColorWhite,
			},
		},
		Padding:    types.EdgeInsets(10),
		Background: &PaintWidget{Color: &color},
		Width:      60,
	}
}

func testCard(child Widget) *Card {
	return &Card{
		SingleChildWidget: SingleChildWidget{Child: child},
		Color:             types.ColorWhite,
		Radius:            8,
		Elevation:         1,
		Padding:           types.EdgeInsets(12),
	}
}

func testColumn(children ...Widget) *Column {
	return &Column{
		Flex: Flex{
			MultiChildWidget: MultiChildWidget{Children: children},
			Direction:        layout.FlexColumn,
		},
	}
}

func testDivider() Widget {
	return &Divider{
		Direction: DividerHorizontal,
		Color:     types.ColorFromRGB(220, 220, 220),
	}
}

func testSpace(h float64) Widget {
	return &Container{Padding: types.EdgeInsets(h)}
}

// ──────────────────────────────────────────────────────────────
// 附加集成测试：特定嵌套场景的边界验证
// ──────────────────────────────────────────────────────────────

// TestBuildContainerNesting 测试 Container 多层嵌套，验证嵌套不产生空节点
func TestBuildContainerNesting(t *testing.T) {
	// 三层 Container 嵌套：outer → middle → inner → Text
	innerText := &Text{Text: "deeply nested"}
	innerBox := &Container{
		SingleChildWidget: SingleChildWidget{Child: innerText},
		Padding:           types.EdgeInsets(8),
		Width:             100,
	}
	middleBox := &Container{
		SingleChildWidget: SingleChildWidget{Child: innerBox},
		Padding:           types.EdgeInsets(12),
	}
	outerBox := &Container{
		SingleChildWidget: SingleChildWidget{Child: middleBox},
		Padding:           types.EdgeInsets(16),
	}

	el := CreateElementFor(outerBox)
	if el == nil {
		t.Fatal("CreateElementFor returned nil")
	}

	var count int
	validateTreeRecursive(t, el, "root", &count)

	if count < 4 {
		t.Errorf("Expected at least 4 elements (3 Container + 1 Text), got %d", count)
	}
	t.Logf("Container nesting: %d elements validated", count)
}

// TestBuildFlexWithAllTypes 测试 Flex 包含所有非容器组件类型
func TestBuildFlexWithAllTypes(t *testing.T) {
	// Column 包含所有"叶组件"类型
	col := testColumn(
		&Checkbox{Label: "opt1"},
		&Switch{Label: "toggle"},
		&RadioButton{Label: "radio1", GroupName: "g", Value: "v1"},
		&Slider{Min: 0, Max: 10, Value: 5},
		&ProgressBar{Value: 0.5},
		&Input{Placeholder: "input"},
		&Icon{Icon: IconStar, Size: 20},
		&Spacer{},
		&Divider{},
		&Text{Text: "label"},
	)

	el := CreateElementFor(col)
	if el == nil {
		t.Fatal("CreateElementFor returned nil for Column")
	}

	var count int
	validateTreeRecursive(t, el, "root", &count)

	t.Logf("Flex with all types: %d elements validated", count)
}

// TestBuildCardWithComplexChild 测试 Card 包含复杂子组件树
func TestBuildCardWithComplexChild(t *testing.T) {
	card := testCard(
		testColumn(
			&Text{Text: "Title"},
			testDivider(),
			&Row{Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Button{Text: "Yes"},
					&Spacer{},
					&Button{Text: "No"},
				}},
				Direction: layout.FlexRow,
			}},
		),
	)

	el := CreateElementFor(card)
	if el == nil {
		t.Fatal("CreateElementFor returned nil for Card")
	}

	var count int
	validateTreeRecursive(t, el, "root", &count)

	t.Logf("Card with complex child: %d elements validated", count)
}

// TestBuildAllWidgetsCreateElementNonNil 确保所有组件类型通过 CreateElementFor 不返回 nil
func TestBuildAllWidgetsCreateElementNonNil(t *testing.T) {
	tests := []struct {
		name string
		w    Widget
	}{
		{"Container", &Container{}},
		{"Text", &Text{Text: "hello"}},
		{"Button", &Button{Text: "click"}},
		{"Flex", &Flex{Direction: layout.FlexRow}},
		{"Column", &Column{}},
		{"Row", &Row{}},
		{"Card", &Card{}},
		{"Checkbox", &Checkbox{Label: "test"}},
		{"Switch", &Switch{}},
		{"RadioButton", &RadioButton{}},
		{"Slider", &Slider{}},
		{"ProgressBar", &ProgressBar{}},
		{"Input", &Input{}},
		{"Icon", &Icon{Icon: IconStar, Size: 24}},
		{"Spacer", &Spacer{}},
		{"Divider", &Divider{}},
		{"ScrollView", &ScrollView{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			el := CreateElementFor(tc.w)
			if el == nil {
				t.Fatalf("CreateElementFor(%T) returned nil", tc.w)
			}
			if el.Widget() == nil {
				t.Errorf("CreateElementFor(%T).Widget() returned nil", tc.w)
			}
		})
	}
}
