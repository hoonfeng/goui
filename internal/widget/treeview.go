package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── TreeView 树形控件（对应 Element Plus el-tree；类型名沿用 TreeView，未改名）──
// 节点展开/折叠/选择；treev2.go(el-tree-v2) 复用本文件的 TreeNode。

// TreeNode 树节点数据
type TreeNode struct {
	Label    string      // 节点显示文本
	Expanded bool        // 是否展开
	Selected bool        // 是否选中
	Children []*TreeNode // 子节点列表
	Data     interface{} // 自定义数据
}

// TreeView 树形视图控件
type TreeView struct {
	StatelessWidget
	RootNodes  []*TreeNode          // 根节点列表
	OnSelect   func(node *TreeNode) // 选中回调
	OnToggle   func(node *TreeNode) // 展开/折叠回调
	ItemHeight float64              // 行高（默认24）
	Indent     float64              // 缩进宽度（默认20）
}

// CreateElement 创建 TreeViewElement
func (tv *TreeView) CreateElement() Element {
	return &TreeViewElement{
		BaseElement: BaseElement{widget: tv},
		treeView:    tv,
	}
}

// TreeViewElement TreeView 的运行时 Element
type TreeViewElement struct {
	BaseElement
	treeView    *TreeView
	visibleRows []*visibleRow // 展开后的可见行列表
	totalHeight float64       // 总内容高度
}

// visibleRow 表示树中一个可见行
type visibleRow struct {
	node     *TreeNode // 对应的树节点
	level    int       // 缩进层级
	hasArrow bool      // 是否有展开箭头（有子节点）
}

// Build 返回 nil（无子控件，直接绘制）
func (e *TreeViewElement) Build() []Element {
	return nil
}

// Layout 计算布局
func (e *TreeViewElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.updateVisibleRows()

	itemHeight := e.treeView.ItemHeight
	if itemHeight <= 0 {
		itemHeight = 24
	}

	e.totalHeight = float64(len(e.visibleRows)) * itemHeight

	// 尺寸根据约束确定
	width := ctx.Constraints.MaxWidth
	if ctx.Constraints.MaxWidth >= float64(1<<30) {
		width = 200 // 无界时给默认宽度
	}

	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  width,
		Height: e.totalHeight,
	})

	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制树形视图
func (e *TreeViewElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()

	itemHeight := e.treeView.ItemHeight
	if itemHeight <= 0 {
		itemHeight = 24
	}
	indent := e.treeView.Indent
	if indent <= 0 {
		indent = 20
	}

	// 绘制每一行
	for i, row := range e.visibleRows {
		y := pos.Y + float64(i)*itemHeight

		// 选中行背景
		if row.node.Selected {
			selectedPaint := paint.DefaultPaint()
			selectedPaint.Color = types.ColorFromRGB(220, 230, 250)
			cvs.DrawRect(pos.X, y, e.size.Width, itemHeight, selectedPaint)
		}

		// 缩进 + 展开箭头
		x := pos.X + float64(row.level)*indent + 4

		// 展开箭头（如果有子节点）
		if row.hasArrow {
			arrowPaint := paint.DefaultPaint()
			arrowPaint.Color = types.ColorFromRGB(80, 80, 80)

			arrowSize := 8.0
			arrowY := y + (itemHeight-arrowSize)/2

			if row.node.Expanded {
				// ▼ 向下箭头（展开状态）
				cvs.DrawPath(e.downArrowPath(x+2, arrowY, arrowSize), arrowPaint)
			} else {
				// ▶ 向右箭头（折叠状态）
				cvs.DrawPath(e.rightArrowPath(x+2, arrowY, arrowSize), arrowPaint)
			}
		}

		// 节点图标（文件夹/文件）
		iconX := x + 16
		iconY := y + (itemHeight-12)/2
		iconPaint := paint.DefaultPaint()
		iconPaint.Color = types.ColorFromRGB(200, 170, 50) // 文件夹色

		if row.hasArrow {
			// 文件夹图标
			if row.node.Expanded {
				cvs.DrawPath(e.folderOpenPath(iconX, iconY, 12, 10), iconPaint)
			} else {
				cvs.DrawPath(e.folderClosedPath(iconX, iconY, 12, 10), iconPaint)
			}
		} else {
			// 文件图标
			iconPaint.Color = types.ColorFromRGB(150, 150, 150)
			cvs.DrawPath(e.filePath(iconX, iconY, 10, 12), iconPaint)
		}

		// 节点文本
		textX := iconX + 16
		textY := y + 4
		textPaint := paint.DefaultPaint()
		textPaint.Color = types.ColorFromRGB(30, 30, 30)
		cvs.DrawText(row.node.Label, textX, textY, canvas.Font{Size: 13}, textPaint)
	}
}

// HandleEvent 处理鼠标事件
func (e *TreeViewElement) HandleEvent(ev event.Event) bool {
	if ev.Type() != event.TypeMouseDown {
		return false
	}

	mouseEv, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}

	pos := e.Offset()
	itemHeight := e.treeView.ItemHeight
	if itemHeight <= 0 {
		itemHeight = 24
	}
	indent := e.treeView.Indent
	if indent <= 0 {
		indent = 20
	}

	// 计算点击的是哪一行
	relY := mouseEv.Y - pos.Y
	rowIndex := int(relY / itemHeight)

	if rowIndex < 0 || rowIndex >= len(e.visibleRows) {
		return false
	}

	row := e.visibleRows[rowIndex]

	// 计算点击区域：箭头区域
	x := pos.X + float64(row.level)*indent + 4
	arrowAreaX := x + 16 // 箭头区域

	relX := mouseEv.X - pos.X

	// 检查是否点击了展开箭头
	if row.hasArrow && relX >= x && relX <= arrowAreaX+4 {
		// 切换展开/折叠状态
		row.node.Expanded = !row.node.Expanded
		e.MarkNeedsLayout()
		e.MarkNeedsPaint()
		if e.treeView.OnToggle != nil {
			e.treeView.OnToggle(row.node)
		}
		return true
	}

	// 点击节点本身：选中
	if !row.node.Selected {
		// 取消之前的所有选中
		e.deselectAll()
		row.node.Selected = true
		e.MarkNeedsPaint()
		if e.treeView.OnSelect != nil {
			e.treeView.OnSelect(row.node)
		}
	}
	return true
}

// deselectAll 取消所有节点的选中状态
func (e *TreeViewElement) deselectAll() {
	for _, root := range e.treeView.RootNodes {
		deselectRecursive(root)
	}
}

func deselectRecursive(node *TreeNode) {
	node.Selected = false
	for _, child := range node.Children {
		deselectRecursive(child)
	}
}

// updateVisibleRows 更新可见行列表
func (e *TreeViewElement) updateVisibleRows() {
	e.visibleRows = nil
	for _, node := range e.treeView.RootNodes {
		flattenTree(node, 0, &e.visibleRows)
	}
}

// flattenTree 递归展平树为可见行列表
func flattenTree(node *TreeNode, level int, rows *[]*visibleRow) {
	*rows = append(*rows, &visibleRow{
		node:     node,
		level:    level,
		hasArrow: len(node.Children) > 0,
	})

	// 如果展开，递归添加子节点
	if node.Expanded {
		for _, child := range node.Children {
			flattenTree(child, level+1, rows)
		}
	}
}

// --- 绘制辅助方法 ---

// rightArrowPath 右侧箭头（▶）
func (e *TreeViewElement) rightArrowPath(x, y, size float64) *canvas.Path {
	p := &canvas.Path{}
	half := size / 2
	p.MoveTo(x, y)
	p.LineTo(x+half, y+half)
	p.LineTo(x, y+size)
	p.Close()
	return p
}

// downArrowPath 向下箭头（▼）
func (e *TreeViewElement) downArrowPath(x, y, size float64) *canvas.Path {
	p := &canvas.Path{}
	half := size / 2
	p.MoveTo(x, y)
	p.LineTo(x+size, y)
	p.LineTo(x+half, y+half)
	p.Close()
	return p
}

// folderClosedPath 关闭的文件夹图标
func (e *TreeViewElement) folderClosedPath(x, y, w, h float64) *canvas.Path {
	p := &canvas.Path{}
	p.MoveTo(x, y+2)
	p.LineTo(x+2, y)
	p.LineTo(x+w*0.4, y)
	p.LineTo(x+w*0.4+2, y+2)
	p.LineTo(x+w-2, y+2)
	p.LineTo(x+w, y+4)
	p.LineTo(x+w, y+h-2)
	p.LineTo(x+w-2, y+h)
	p.LineTo(x+2, y+h)
	p.LineTo(x, y+h-2)
	p.Close()
	return p
}

// folderOpenPath 打开的文件夹图标
func (e *TreeViewElement) folderOpenPath(x, y, w, h float64) *canvas.Path {
	p := &canvas.Path{}
	p.MoveTo(x, y+2)
	p.LineTo(x+2, y)
	p.LineTo(x+w*0.4, y)
	p.LineTo(x+w*0.4+2, y+2)
	p.LineTo(x+w*0.6, y+2)
	p.LineTo(x+w*0.6+2, y+4)
	p.LineTo(x+w-2, y+4)
	p.LineTo(x+w, y+6)
	p.LineTo(x+w, y+h-2)
	p.LineTo(x+w-2, y+h)
	p.LineTo(x+2, y+h)
	p.LineTo(x, y+h-2)
	p.Close()
	return p
}

// filePath 文件图标
func (e *TreeViewElement) filePath(x, y, w, h float64) *canvas.Path {
	p := &canvas.Path{}
	// 文件主体
	p.MoveTo(x+2, y)
	p.LineTo(x+w-2, y)
	p.LineTo(x+w, y+2)
	p.LineTo(x+w, y+h-2)
	p.LineTo(x+w-2, y+h)
	p.LineTo(x+2, y+h)
	p.LineTo(x, y+h-2)
	p.LineTo(x, y+2)
	p.Close()
	return p
}
