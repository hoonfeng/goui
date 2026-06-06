// Package state 持有 companion 的应用状态（Go 结构体 + goui SetState，见 ../AGENTS.md §5）。
// 不照搬 Zustand：状态结构体是布局的唯一真相来源，变更后由持有它的 State 调 SetState 触发重建。
package state

// Panels 三个停靠区的可见性与尺寸——窗壳布局的唯一真相来源。
type Panels struct {
	Left, Right, Bottom    bool    // 各停靠区是否展开
	LeftW, RightW, BottomH float64 // 各停靠区尺寸（像素）
}

// DefaultPanels 默认三区全开，IDE 常用尺寸。
func DefaultPanels() *Panels {
	return &Panels{
		Left: true, Right: true, Bottom: true,
		LeftW: 260, RightW: 400, BottomH: 200,
	}
}

// 尺寸约束（拖动 resize 时夹紧，避免拖没/拖爆）。
const (
	MinSideW = 160
	MaxSideW = 640
	MinBotH  = 120
	MaxBotH  = 480
)

// Toggle 翻转某停靠区的展开状态。
func (p *Panels) Toggle(z Zone) {
	switch z {
	case ZoneLeft:
		p.Left = !p.Left
	case ZoneRight:
		p.Right = !p.Right
	case ZoneBottom:
		p.Bottom = !p.Bottom
	}
}

// Zone 标识一个停靠区。
type Zone int

const (
	ZoneLeft Zone = iota
	ZoneRight
	ZoneBottom
)

// Clamp 把 v 夹到 [lo, hi]。
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
