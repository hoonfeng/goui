// AppendColumn — 增量追加列容器。
//
// 专为 Agent 流式追加场景设计：数据只在末尾追加，已有项不变。
// AppendColumn 维护自己的 Element 池（类似 VirtualList 的 slot 池），
// 新增数据时只创建新 Element，已有 Element 不重新构建。
//
// 相比 Column 在 SetState 时全量重建所有子节点（即使只有末尾变化），
// AppendColumn 避免了对已存在子节点的 Build/Layout 级联，大幅降低 GC 压力。
package widget

import (
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/types"
)

type AppendColumn struct {
	StatelessWidget
	Children []Widget // 子控件列表，尾部追加
}

func (a *AppendColumn) CreateElement() Element {
	return &AppendColumnElement{
		BaseElement: BaseElement{widget: a},
		col:         a,
		childEls:    make([]Element, 0, 32),
	}
}

// AppendColumnElement AppendColumn 的运行时 Element。
// 维护子 Element 池，每次 Build 只处理新增的子节点。
type AppendColumnElement struct {
	BaseElement
	col      *AppendColumn
	childEls []Element // 已创建的子 Element（按序，长度 <= col.Children）
}

// Build 增量构建子 Element：只处理新追加的子节点，已有节点复用。
func (e *AppendColumnElement) Build() []Element {
	n := len(e.col.Children)

	// 对于已有子 Element，就地复用（不 Update，不 re-build）
	// 注意：如果已有子节点的 Widget 配置真的变化了，外部需
	// 通过 ItemVersion 机制或手动 Update 触发重绘。
	// AppendColumn 的设计假设只有追加、没有修改。

	// 新增子节点：创建新 Element
	if n > len(e.childEls) {
		for i := len(e.childEls); i < n; i++ {
			w := e.col.Children[i]
			if w == nil {
				// 占位：用 nil 占位保持索引对齐
				e.childEls = append(e.childEls, nil)
				continue
			}
			el := CreateElementFor(w)
			el.Mount(e, len(e.childEls))
			e.childEls = append(e.childEls, el)
		}
	}

	// 如果子节点变少了（极少发生），释放末尾的多余 Element
	if n < len(e.childEls) {
		for i := n; i < len(e.childEls); i++ {
			if e.childEls[i] != nil {
				e.childEls[i].Unmount()
			}
		}
		e.childEls = e.childEls[:n]
	}

	// 构建 children 列表
	if cap(e.children) < len(e.childEls) {
		e.children = make([]Element, 0, len(e.childEls))
	}
	e.children = e.children[:0]
	for _, el := range e.childEls {
		if el != nil {
			e.children = append(e.children, el)
		}
	}
	return e.children
}

// Layout 委托给所有子 Element（FlexColumn 布局）。
func (e *AppendColumnElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	var usedH float64
	maxW := ctx.Constraints.MaxWidth

	for _, el := range e.childEls {
		if el == nil {
			continue
		}
		childCtx := &layout.LayoutContext{
			Constraints: layout.BoxConstraints{
				MinWidth:  0,
				MaxWidth:  maxW,
				MinHeight: 0,
				MaxHeight: ctx.Constraints.MaxHeight - usedH,
			},
		}
		res := el.Layout(childCtx)
		el.SetPosition(types.Point{X: 0, Y: usedH})
		w := res.Size.Width
		if w > maxW {
			maxW = w
		}
		usedH += res.Size.Height
	}

	e.size = ctx.Constraints.Constrain(types.Size{Width: maxW, Height: usedH})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制所有子 Element。
func (e *AppendColumnElement) Paint(cvs canvas.Canvas, offset types.Point) {
	for _, el := range e.childEls {
		if el != nil {
			el.Paint(cvs, offset)
		}
	}
}

// Update 更新配置。
func (e *AppendColumnElement) Update(newWidget Widget) {
	if ac, ok := newWidget.(*AppendColumn); ok {
		e.col = ac
	}
	e.BaseElement.Update(newWidget)
}
