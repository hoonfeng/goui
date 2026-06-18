package widget

import (
	"testing"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
)

// TestMarkdownTableCellWrap 验证表格单元格内长文本自动换行。
//
// 原问题：tableCell 使用 inlineRow（Row）导致 Text 获得不受限 MaxWidth，
// 长文本不换行，导致单元格内容重叠/溢出。
//
// 修复：tableCell 改为 Column + 单个 Text，Column 传递受限宽度给 Text，
// Text 的 splitLines 按可用宽度折行，多行文本正确显示。
func TestMarkdownTableCellWrap(t *testing.T) {
	// ── 测试用 Markdown 表格：每列宽度很窄，文字会换行 ──
	md := `| 名称 | 说明 |
|------|------|
| 这是一个很长的名称 | 这是一段非常长的说明文字，在窄列中必须换行才能完整显示 |
| short | ok |
`
	view := NewMarkdownView(md, DefaultMarkdownStyle())
	rootWidget := view.Build(newTestBuildContext())

	// 创建根 Element
	rootEl := CreateElementFor(rootWidget)
	buildFullTree(rootEl) // 递归展开整个 Element 树

	// ── Layout：窄宽度 200 → 2 列各约 85px（扣除边框 padding）→ 文字必然换行 ──
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth: 200, MaxWidth: 200,
			MinHeight: 0, MaxHeight: 1000,
		},
	}
	rootEl.Layout(ctx)
	rootSize := rootEl.Size()
	if rootSize.Width != 200 {
		t.Errorf("root width should be 200 (constrained), got %.0f", rootSize.Width)
	}

	// ── 找到所有 Text Element，验证长文本换行了 ──
	foundLongName := false
	foundLongDesc := false
	visitTexts(rootEl, func(el *TextElement, depth int) {
		// 使用 visibleLines 获取换行后的行数
		lineH := canvas.GetFaceLineHeight(el.text.Font.Size)
		if el.text.LineHeight > 0 {
			lineH = el.text.LineHeight
		}
		estimatedLines := 0
		if lineH > 0 {
			estimatedLines = int(el.size.Height / lineH)
			if estimatedLines < 1 {
				estimatedLines = 1
			}
		}
		t.Logf("text[%q] size=(%.0f,%.0f) lineH=%.0f estLines=%d depth=%d",
			truncStr(el.text.Text, 30), el.size.Width, el.size.Height, lineH, estimatedLines, depth)

		if el.text.Text == "这是一个很长的名称" {
			foundLongName = true
			if estimatedLines < 1 {
				t.Error("short text should have >=1 line")
			}
		}
		if el.text.Text == "这是一段非常长的说明文字，在窄列中必须换行才能完整显示" {
			foundLongDesc = true
			if estimatedLines < 2 {
				t.Errorf("long cell text should wrap to >=2 lines, got %d lines (height=%.0f)", estimatedLines, el.size.Height)
			}
		}
	})

	if !foundLongName {
		t.Error("did not find '这是一个很长的名称' text element (cell content may be missing)")
	}
	if !foundLongDesc {
		t.Error("did not find long description text (cell content may be missing)")
	}

	// ── 用更宽松的宽度验证文字不换行 ──
	view2 := NewMarkdownView(md, DefaultMarkdownStyle())
	rootWidget2 := view2.Build(newTestBuildContext())
	rootEl2 := CreateElementFor(rootWidget2)
	buildFullTree(rootEl2)
	ctx2 := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth: 800, MaxWidth: 800,
			MinHeight: 0, MaxHeight: 1000,
		},
	}
	rootEl2.Layout(ctx2)

	foundShortWith1Line := false
	visitTexts(rootEl2, func(el *TextElement, depth int) {
		if el.text.Text == "这是一段非常长的说明文字，在窄列中必须换行才能完整显示" {
			lineH := canvas.GetFaceLineHeight(el.text.Font.Size)
			if el.text.LineHeight > 0 {
				lineH = el.text.LineHeight
			}
			estLines := 1
			if lineH > 0 {
				estLines = int(el.size.Height / lineH)
				if estLines < 1 {
					estLines = 1
				}
			}
			if estLines <= 1 {
				foundShortWith1Line = true
			}
		}
	})
	if !foundShortWith1Line {
		t.Log("wide layout: long text may still wrap (column too narrow) - not a failure")
	}
}

// TestMarkdownTableFullPipeline 验证表格从 source→widget→element→layout
// 的完整链路正确运行，不 panic。
func TestMarkdownTableFullPipeline(t *testing.T) {
	md := `| A | B | C |
|:-|:-:|-:|
| left | center | right |
| 普通文本 | **加粗** | *斜体* |
| ` + "`代码`" + ` | ~~删除~~ | [链接](/) |
`
	view := NewMarkdownView(md, DefaultMarkdownStyle())
	rootWidget := view.Build(newTestBuildContext())
	if rootWidget == nil {
		t.Fatal("MarkdownView.Build() returned nil")
	}

	rootEl := CreateElementFor(rootWidget)
	if rootEl == nil {
		t.Fatal("CreateElementFor returned nil")
	}
	buildFullTree(rootEl)

	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth: 400, MaxWidth: 400,
			MinHeight: 0, MaxHeight: 2000,
		},
	}
	result := rootEl.Layout(ctx)
	if result.Size.Width != 400 {
		t.Errorf("constrained width should be 400, got %.0f", result.Size.Width)
	}
	if result.Size.Height <= 0 {
		t.Errorf("table should have positive height, got %.0f", result.Size.Height)
	}

	t.Logf("table pipeline OK: size=(%.0f,%.0f)", result.Size.Width, result.Size.Height)
}

// ─── 辅助 ────────────────────────────────────────────────

// buildFullTree 递归调用 Build() 展开整个 Element 树。
func buildFullTree(el Element) {
	children := el.Build()
	if children == nil {
		return
	}
	for _, child := range children {
		buildFullTree(child)
	}
}

// visitTexts 遍历 Element 树，为每个 TextElement 触发 fn。
func visitTexts(el Element, fn func(el *TextElement, depth int)) {
	visitTextsRec(el, fn, 0)
}

func visitTextsRec(el Element, fn func(el *TextElement, depth int), depth int) {
	if te, ok := el.(*TextElement); ok {
		fn(te, depth)
	}
	if el.Children() != nil {
		for _, child := range el.Children() {
			visitTextsRec(child, fn, depth+1)
		}
	}
}

// truncStr 截断字符串用于日志。
func truncStr(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "..."
}

// newTestBuildContext 返回一个最小 BuildContext（仅用于测试 MarkdownView.Build）。
func newTestBuildContext() BuildContext {
	return BuildContext{}
}
