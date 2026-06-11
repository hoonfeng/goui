package widget

import (
	"testing"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
)

// TestAnimatedBuilderRebuildsOnTick 验证通用动画构建器：controller 每帧推进时
// 用最新值重建子树。
func TestAnimatedBuilderRebuildsOnTick(t *testing.T) {
	animation.ResetScheduler()
	ctrl := animation.NewController(100*time.Millisecond, animation.Linear)

	var lastWidth float64
	ab := &AnimatedBuilder{
		Controller: ctrl,
		Builder: func(v float64) Widget {
			lastWidth = animation.LerpFloat(0, 100, v)
			return &Container{Width: lastWidth, Height: 10}
		},
	}
	el := CreateElementFor(ab).(*StatefulElement)
	el.Mount(nil, 0)
	buildTreeRec(el) // 初始构建（InitState 已 hook 到 controller）

	ctrl.Start()
	base := time.Unix(2000, 0)
	animation.Tick(base)
	animation.Tick(base.Add(50 * time.Millisecond)) // v=0.5 → 触发 SetState

	buildTreeRec(el) // relayout 重建，Builder 拿到新值
	if lastWidth < 40 || lastWidth > 60 {
		t.Errorf("builder width=%.1f, want ~50 at half progress", lastWidth)
	}
}
