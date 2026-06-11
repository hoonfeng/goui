package widget

import (
	"math"
	"testing"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// TestTransformHitPointInverse 验证 Transform 的命中点逆变换：把视觉(父)坐标
// 正确反映射回子坐标系，使缩放/旋转后的子元素命中判定与渲染一致。
func TestTransformHitPointInverse(t *testing.T) {
	mk := func(tr *Transform) *transformElement {
		el := tr.CreateElement().(*transformElement)
		el.Mount(nil, 0)
		el.Build()
		el.SetPosition(types.Point{X: 0, Y: 0})
		el.Layout(&layout.LayoutContext{Constraints: layout.Tight(20, 20)})
		return el
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }

	// 缩放 2 倍，中心 (10,10)：中心不变；视觉边缘 (20,10) → 子坐标 (15,10)
	s := mk(&Transform{Scale: 2, Child: &Container{Width: 20, Height: 20}})
	if cx, cy := s.TransformHitPoint(10, 10); !near(cx, 10) || !near(cy, 10) {
		t.Errorf("scale center inverse=(%.2f,%.2f), want (10,10)", cx, cy)
	}
	if cx, cy := s.TransformHitPoint(20, 10); !near(cx, 15) || !near(cy, 10) {
		t.Errorf("scale edge inverse=(%.2f,%.2f), want (15,10)", cx, cy)
	}

	// 旋转 90°（围绕中心）：视觉 (10,15) ← 子 (15,10)，逆变换应还原
	r := mk(&Transform{Rotation: 90, Child: &Container{Width: 20, Height: 20}})
	if cx, cy := r.TransformHitPoint(10, 15); !near(cx, 15) || !near(cy, 10) {
		t.Errorf("rotate90 inverse=(%.2f,%.2f), want (15,10)", cx, cy)
	}

	// 无变换：原样返回
	n := mk(&Transform{Child: &Container{Width: 20, Height: 20}})
	if cx, cy := n.TransformHitPoint(7, 3); !near(cx, 7) || !near(cy, 3) {
		t.Errorf("identity inverse=(%.2f,%.2f), want (7,3)", cx, cy)
	}
}
