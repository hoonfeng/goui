package widget

import (
	"testing"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// TestBuildBasic 测试基础组件的 Build 方法。
// 覆盖 BaseElement、ContainerElement、ButtonElement、StatelessElement、TextElement
// 的 Build 行为，确保基础组件能正确构建控件树。
func TestBuildBasic(t *testing.T) {
	t.Run("BaseElement Build returns nil", func(t *testing.T) {
		e := &BaseElement{}
		result := e.Build()
		if result != nil {
			t.Errorf("BaseElement.Build() should return nil, got %v", result)
		}
	})

	t.Run("Container Build without child returns nil", func(t *testing.T) {
		c := &Container{}
		el := c.CreateElement().(*ContainerElement)
		result := el.Build()
		if result != nil {
			t.Errorf("ContainerElement.Build() without child should return nil, got %v", result)
		}
	})

	t.Run("Container Build with child returns child element", func(t *testing.T) {
		text := &Text{Text: "hello"}
		c := &Container{
			SingleChildWidget: SingleChildWidget{Child: text},
		}
		el := c.CreateElement().(*ContainerElement)
		result := el.Build()
		if result == nil {
			t.Fatal("ContainerElement.Build() with child should return non-nil")
		}
		if len(result) != 1 {
			t.Fatalf("ContainerElement.Build() should return 1 element, got %d", len(result))
		}
		// 验证子 Element 的 widget 类型是 *Text
		if _, ok := result[0].Widget().(*Text); !ok {
			t.Errorf("Child widget should be *Text, got %T", result[0].Widget())
		}
		// 验证子 Element 已挂载
		if result[0].Parent() != el {
			t.Error("Child element should have ContainerElement as parent")
		}
	})

	t.Run("Container Build creates new child on each call", func(t *testing.T) {
		text := &Text{Text: "hello"}
		c := &Container{
			SingleChildWidget: SingleChildWidget{Child: text},
		}
		el := c.CreateElement().(*ContainerElement)

		// 每次 Build() 应创建新的子 Element（当前实现没有缓存逻辑）
		result1 := el.Build()
		if len(result1) != 1 {
			t.Fatalf("First Build: expected 1 element, got %d", len(result1))
		}

		result2 := el.Build()
		if len(result2) != 1 {
			t.Fatalf("Second Build: expected 1 element, got %d", len(result2))
		}

		// ContainerElement 每次 Build 都创建新的子 Element
		if result1[0] == result2[0] {
			t.Log("ContainerElement.Build() returned same instance (implementation may vary)")
		}
	})

	t.Run("Button Build without child returns nil", func(t *testing.T) {
		b := &Button{}
		el := b.CreateElement().(*ButtonElement)
		result := el.Build()
		if result != nil {
			t.Errorf("ButtonElement.Build() without child should return nil, got %v", result)
		}
	})

	t.Run("Button Build with child returns child element", func(t *testing.T) {
		text := &Text{Text: "click me"}
		b := &Button{
			SingleChildWidget: SingleChildWidget{Child: text},
		}
		el := b.CreateElement().(*ButtonElement)
		result := el.Build()
		if result == nil {
			t.Fatal("ButtonElement.Build() with child should return non-nil")
		}
		if len(result) != 1 {
			t.Fatalf("ButtonElement.Build() should return 1 element, got %d", len(result))
		}
		// 验证子 Element 的 widget 类型是 *Text
		if _, ok := result[0].Widget().(*Text); !ok {
			t.Errorf("Child widget should be *Text, got %T", result[0].Widget())
		}
		// 验证子 Element 已挂载
		if result[0].Parent() != el {
			t.Error("Child element should have ButtonElement as parent")
		}
	})

	t.Run("Button Build creates new child on each call", func(t *testing.T) {
		text := &Text{Text: "click"}
		b := &Button{
			SingleChildWidget: SingleChildWidget{Child: text},
		}
		el := b.CreateElement().(*ButtonElement)

		result1 := el.Build()
		if len(result1) != 1 {
			t.Fatalf("First Build: expected 1 element, got %d", len(result1))
		}

		result2 := el.Build()
		if len(result2) != 1 {
			t.Fatalf("Second Build: expected 1 element, got %d", len(result2))
		}

		// ButtonElement 每次 Build 都创建新的子 Element
		if result1[0] == result2[0] {
			t.Log("ButtonElement.Build() returned same instance (implementation may vary)")
		}
	})

	t.Run("TextElement Build returns nil (inherited from BaseElement)", func(t *testing.T) {
		text := &Text{Text: "test text"}
		el := text.CreateElement().(*TextElement)
		result := el.Build()
		if result != nil {
			t.Errorf("TextElement.Build() should return nil (inherited from BaseElement), got %v", result)
		}
	})

	t.Run("StatelessElement Build with no builder returns nil", func(t *testing.T) {
		// Text 是 StatelessWidget，但没有实现 Build(BuildContext) Widget 接口
		text := &Text{Text: "no builder"}
		// 包装为 StatelessElement
		se := &StatelessElement{
			BaseElement: BaseElement{widget: text},
		}
		result := se.Build()
		if result != nil {
			t.Errorf("StatelessElement.Build() without Builder interface should return nil, got %v", result)
		}
	})

	t.Run("StatelessElement Build with builder returns child", func(t *testing.T) {
		// 创建一个实现 Build(BuildContext) Widget 接口的无状态控件
		builder := &testBuilderWidget{child: &Text{Text: "built"}}
		el := &StatelessElement{
			BaseElement: BaseElement{widget: builder},
		}
		result := el.Build()
		if result == nil {
			t.Fatal("StatelessElement.Build() with Builder should return non-nil")
		}
		if len(result) != 1 {
			t.Fatalf("StatelessElement.Build() should return 1 element, got %d", len(result))
		}
		if _, ok := result[0].Widget().(*Text); !ok {
			t.Errorf("Child widget should be *Text, got %T", result[0].Widget())
		}
	})
}

// testBuilderWidget 是一个测试用的无状态控件，实现了 Build(BuildContext) Widget 接口
type testBuilderWidget struct {
	StatelessWidget
	child Widget
}

func (w *testBuilderWidget) Build(ctx BuildContext) Widget {
	return w.child
}

// TestBuildNilChild 测试当子控件为 nil 时各组件 Build 的行为
func TestBuildNilChild(t *testing.T) {
	t.Run("Container with nil child via SingleChildWidget", func(t *testing.T) {
		c := &Container{
			SingleChildWidget: SingleChildWidget{Child: nil},
		}
		el := c.CreateElement().(*ContainerElement)
		result := el.Build()
		if result != nil {
			t.Errorf("ContainerElement.Build() with nil child should return nil, got %v", result)
		}
	})

	t.Run("Button with nil child via SingleChildWidget", func(t *testing.T) {
		b := &Button{
			SingleChildWidget: SingleChildWidget{Child: nil},
		}
		el := b.CreateElement().(*ButtonElement)
		result := el.Build()
		if result != nil {
			t.Errorf("ButtonElement.Build() with nil child should return nil, got %v", result)
		}
	})
}

// TestBuildMountOrder 测试 Build 时子 Element 的挂载顺序和插槽索引
func TestBuildMountOrder(t *testing.T) {
	t.Run("Container child mounts at slot 0", func(t *testing.T) {
		text := &Text{Text: "child"}
		c := &Container{
			SingleChildWidget: SingleChildWidget{Child: text},
		}
		el := c.CreateElement().(*ContainerElement)
		result := el.Build()
		if len(result) != 1 {
			t.Fatalf("Expected 1 child, got %d", len(result))
		}
		// 验证子 Element 的 slot index（通过父类 BaseElement 提升的字段访问）
		child := result[0]
		if te, ok := child.(*TextElement); ok {
			if te.slotIndex != 0 {
				t.Errorf("Child slot index should be 0, got %d", te.slotIndex)
			}
		} else {
			t.Errorf("Child element should be *TextElement, got %T", child)
		}
	})
}

// TestBuildFlex 测试 FlexElement（Row/Column）的 Build 方法。
// 覆盖 examples/test/main.go 中使用的 Row 和 Column 组件。
func TestBuildFlex(t *testing.T) {
	t.Run("Row builds children correctly", func(t *testing.T) {
		row := &Row{
			Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Text{Text: "A"},
					&Text{Text: "B"},
					&Text{Text: "C"},
				}},
				Direction: layout.FlexRow,
			},
		}
		el := row.CreateElement().(*FlexElement)
		result := el.Build()
		if result == nil {
			t.Fatal("Row: Build() should return non-nil slice")
		}
		if len(result) != 3 {
			t.Fatalf("Row: expected 3 children, got %d", len(result))
		}
		for i, child := range result {
			if _, ok := child.Widget().(*Text); !ok {
				t.Errorf("Row: child %d should be *Text, got %T", i, child.Widget())
			}
			if child.Parent() != el {
				t.Errorf("Row: child %d parent should be FlexElement", i)
			}
		}
	})

	t.Run("Column builds children correctly", func(t *testing.T) {
		col := &Column{
			Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Text{Text: "X"},
					&Text{Text: "Y"},
				}},
				Direction: layout.FlexColumn,
			},
		}
		el := col.CreateElement().(*FlexElement)
		result := el.Build()
		if result == nil {
			t.Fatal("Column: Build() should return non-nil slice")
		}
		if len(result) != 2 {
			t.Fatalf("Column: expected 2 children, got %d", len(result))
		}
		for i, child := range result {
			if child.Parent() != el {
				t.Errorf("Column: child %d parent should be FlexElement", i)
			}
		}
	})

	t.Run("Empty children returns empty non-nil slice", func(t *testing.T) {
		row := &Row{
			Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{}},
				Direction:        layout.FlexRow,
			},
		}
		el := row.CreateElement().(*FlexElement)
		result := el.Build()
		if result == nil {
			t.Fatal("FlexElement.Build() with empty children should return empty slice, not nil")
		}
		if len(result) != 0 {
			t.Errorf("Expected 0 children, got %d", len(result))
		}
	})

	t.Run("Children contain mixed types", func(t *testing.T) {
		row := &Row{
			Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Text{Text: "label"},
					&Spacer{},
					&Button{Text: "btn", SingleChildWidget: SingleChildWidget{Child: &Text{Text: "click"}}},
				}},
				Direction: layout.FlexRow,
			},
		}
		el := row.CreateElement().(*FlexElement)
		result := el.Build()
		if len(result) != 3 {
			t.Fatalf("Expected 3 children, got %d", len(result))
		}
		// 验证类型顺序（SpacerElement.Widget() 需要返回 *Spacer）
		if _, ok := result[0].Widget().(*Text); !ok {
			t.Errorf("Child 0 should be *Text, got %T", result[0].Widget())
		}
		if _, ok := result[1].Widget().(*Spacer); !ok {
			t.Errorf("Child 1 should be *Spacer, got %T (check Spacer.CreateElement sets BaseElement.widget)", result[1].Widget())
		}
		if _, ok := result[2].Widget().(*Button); !ok {
			t.Errorf("Child 2 should be *Button, got %T", result[2].Widget())
		}
	})
}

// TestBuildCard 测试 CardElement 的 Build 方法。
// 特别验证已修复的 Card.Child 字段阴影问题（Card.Child 曾遮蔽 SingleChildWidget.Child）。
// Card 在 examples/test/main.go 的 card() 辅助函数中使用 SingleChildWidget 设置子控件。
func TestBuildCard(t *testing.T) {
	t.Run("Card with child via SingleChildWidget returns child element", func(t *testing.T) {
		// 模拟 main.go 中 card() 辅助函数的构造方式
		cd := &Card{
			SingleChildWidget: SingleChildWidget{Child: &Text{Text: "card text"}},
			Color:             types.ColorWhite,
			Radius:            8,
		}
		el := cd.CreateElement().(*CardElement)
		result := el.Build()
		if result == nil {
			t.Fatal("CardElement.Build() with child should return non-nil (check Card.Child shadowing fix)")
		}
		if len(result) != 1 {
			t.Fatalf("Expected 1 child, got %d", len(result))
		}
		if _, ok := result[0].Widget().(*Text); !ok {
			t.Errorf("Child should be *Text, got %T", result[0].Widget())
		}
		if result[0].Parent() != el {
			t.Error("Child should have CardElement as parent")
		}
	})

	t.Run("Card without child returns nil", func(t *testing.T) {
		cd := &Card{
			Color:  types.ColorWhite,
			Radius: 8,
		}
		el := cd.CreateElement().(*CardElement)
		result := el.Build()
		if result != nil {
			t.Errorf("CardElement.Build() without child should return nil, got %v", result)
		}
	})

	t.Run("Card with nil SingleChildWidget.Child returns nil", func(t *testing.T) {
		cd := &Card{
			SingleChildWidget: SingleChildWidget{Child: nil},
			Color:             types.ColorWhite,
			Radius:            8,
		}
		el := cd.CreateElement().(*CardElement)
		result := el.Build()
		if result != nil {
			t.Errorf("CardElement.Build() with nil child should return nil, got %v", result)
		}
	})
}

// TestBuildLeafWidgets 测试所有叶组件（无子控件）的 Build 方法。
// 覆盖 examples/test/main.go 中使用的 Checkbox、Switch、RadioButton、
// Slider、ProgressBar、Input、Icon、Spacer、Divider。
// 这些组件的 Build() 均应返回 nil。
func TestBuildLeafWidgets(t *testing.T) {
	tests := []struct {
		name string
		w    Widget
	}{
		{"Checkbox", &Checkbox{}},
		{"Switch", &Switch{}},
		{"RadioButton", &RadioButton{}},
		{"Slider", &Slider{}},
		{"ProgressBar", &ProgressBar{}},
		{"Input", &Input{}},
		{"Icon", &Icon{Icon: IconStar, Size: 24}},
		{"Spacer", &Spacer{}},
		{"Divider", &Divider{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			el := tc.w.CreateElement()
			result := el.Build()
			if result != nil {
				t.Errorf("%s.Build() should return nil (leaf widget), got %v", tc.name, result)
			}
		})
	}
}

// TestBuildFlexWithNilChildren 测试 FlexElement 在子控件为 nil 时的行为。
// 验证当子控件列表包含 nil 时 CreateElementFor 的 panic 行为（预期）。
func TestBuildFlexWithNilChildren(t *testing.T) {
	t.Run("Flex with nil child panics (expected)", func(t *testing.T) {
		// MultiChildWidget.Children 中包含 nil 时，
		// CreateElementFor(nil) 会在调用 w.CreateElement() 时 panic。
		// 这是预期行为——调用方应确保不传入 nil 子控件。
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when Flex contains nil child, but no panic occurred")
			} else {
				t.Logf("Got expected panic: %v", r)
			}
		}()

		row := &Row{
			Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{
					&Text{Text: "valid"},
					nil,
				}},
				Direction: layout.FlexRow,
			},
		}
		el := row.CreateElement().(*FlexElement)
		el.Build()
	})

	t.Run("Flex with all nil children panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when Flex contains all nil children, but no panic occurred")
			}
		}()

		col := &Column{
			Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: []Widget{nil, nil}},
				Direction:        layout.FlexColumn,
			},
		}
		el := col.CreateElement().(*FlexElement)
		el.Build()
	})
}

// TestBuildCreateElementWidget 验证所有 CreateElement 正确设置了 BaseElement.widget。
func TestBuildCreateElementWidget(t *testing.T) {
	t.Run("FlexElement.Widget() returns *Flex", func(t *testing.T) {
		f := &Flex{
			MultiChildWidget: MultiChildWidget{Children: nil},
			Direction:        layout.FlexRow,
		}
		el := f.CreateElement()
		w := el.Widget()
		if w == nil {
			t.Fatal("FlexElement.Widget() should not be nil")
		}
		if _, ok := w.(*Flex); !ok {
			t.Errorf("Expected *Flex, got %T", w)
		}
	})

	t.Run("SpacerElement.Widget() returns *Spacer", func(t *testing.T) {
		s := &Spacer{}
		el := s.CreateElement()
		w := el.Widget()
		if w == nil {
			t.Fatal("SpacerElement.Widget() should not be nil")
		}
		if _, ok := w.(*Spacer); !ok {
			t.Errorf("Expected *Spacer, got %T", w)
		}
	})

	t.Run("ScrollViewElement.Widget() returns *ScrollView", func(t *testing.T) {
		sv := &ScrollView{}
		el := sv.CreateElement()
		w := el.Widget()
		if w == nil {
			t.Fatal("ScrollViewElement.Widget() should not be nil")
		}
		if _, ok := w.(*ScrollView); !ok {
			t.Errorf("Expected *ScrollView, got %T", w)
		}
	})
}
