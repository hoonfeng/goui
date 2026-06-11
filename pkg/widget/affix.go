package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Affix 固钉（滚动地基应用）───────────────────────────────────
// 1:1 复刻 el-affix(基础)：置于 ScrollView 内容中，当滚动使其原位置超出
// 视口顶部(减 Offset)时，固定在视口顶部 Offset 处；否则随内容正常滚动。
// 占位由 Layout 的子尺寸保留；固定时用画布平移把子画到固定位置(脱离滚动流)。
// 设计：向上沿 parent 链找 ScrollViewElement 读 scrollOffset，每帧重绘据此决定固定与否。
// 限制：goui 自绘层级固定，Affix 之后的内容若滚到固定区会盖住固定头——
// 故 Affix 宜置于滚动内容顶部(典型场景:吸顶导航/工具条)。

type Affix struct {
	SingleChildWidget
	Offset float64 // 固定时距视口顶的距离(默认 0)
}

func NewAffix(child Widget) *Affix {
	return &Affix{SingleChildWidget: SingleChildWidget{Child: child}}
}

func (a *Affix) WithOffset(o float64) *Affix { a.Offset = o; return a }

func (a *Affix) CreateElement() Element {
	return &AffixElement{BaseElement: BaseElement{widget: a}, affix: a}
}

type AffixElement struct {
	BaseElement
	affix   *Affix
	child   Element
	fixedDY float64 // 固定时子的画布平移量(Paint 算出，paintFixed 使用)
}

func (e *AffixElement) Build() []Element {
	if e.affix.Child != nil {
		e.child = CreateElementFor(e.affix.Child)
		e.child.Mount(e, 0)
		e.children = []Element{e.child}
		return e.children
	}
	e.children = nil
	return nil
}

func (e *AffixElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		res := e.child.Layout(ctx)
		e.child.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

// findScrollView 沿 parent 链向上找最近的 ScrollViewElement(Affix 所在的滚动容器)。
func (e *AffixElement) findScrollView() *ScrollViewElement {
	for p := e.Parent(); p != nil; p = p.Parent() {
		if sv, ok := p.(*ScrollViewElement); ok {
			return sv
		}
	}
	return nil
}

func (e *AffixElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child == nil {
		return
	}
	if sv := e.findScrollView(); sv != nil {
		myPos := e.Offset()
		fixedY := sv.Offset().Y + e.affix.Offset
		if myPos.Y < fixedY {
			// 固定：登记给 ScrollView，留待其在所有内容之上重画
			// (此处不画，避免被后续内容在绘制顺序上覆盖)
			e.fixedDY = fixedY - myPos.Y
			sv.registerFixedAffix(e)
			return
		}
	}
	// 未触发固定：子随内容正常绘制
	e.child.Paint(cvs, offset)
}

// paintFixed 由 ScrollView 在 content 之上调用，把子画到视口固定位置(脱离滚动)。
func (e *AffixElement) paintFixed(cvs canvas.Canvas, offset types.Point) {
	if e.child == nil {
		return
	}
	cvs.Save()
	cvs.Translate(0, e.fixedDY)
	e.child.Paint(cvs, offset)
	cvs.Restore()
}

func (e *AffixElement) Update(newWidget Widget) {
	if na, ok := newWidget.(*Affix); ok {
		e.affix = na
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
