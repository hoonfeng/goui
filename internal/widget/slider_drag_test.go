package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/types"
)

// TestSliderDragFollowsContinuously 守护滑块连续跟手：按下后超过拖拽阈值，事件转为 DragMove，
// 滑块须持续跟随（旧 bug：只处理 MouseMove → 拖拽中只动一次）。
func TestSliderDragFollowsContinuously(t *testing.T) {
	var got float64
	s := NewSlider(0, func(v float64) { got = v }).WithRange(0, 100).WithStep(1)
	e := s.CreateElement().(*SliderElement)
	e.size = types.Size{Width: 200, Height: 20} // padding=11, trackW=178

	e.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 11, 10, event.ButtonLeft, 0)) // 吸到左端≈0
	if got > 1 {
		t.Errorf("按下左端应≈0，得 %v", got)
	}
	e.HandleEvent(event.NewDragEvent(event.TypeDragMove, 100, 10, event.ButtonLeft, 0)) // 拖到中间
	if got < 45 || got > 55 {
		t.Errorf("拖到中间应≈50，得 %v（DragMove 未跟手=bug 未修）", got)
	}
	e.HandleEvent(event.NewDragEvent(event.TypeDragMove, 195, 10, event.ButtonLeft, 0)) // 拖到右端
	if got < 95 {
		t.Errorf("拖到右端应≈100，得 %v", got)
	}
	// 拖拽中鼠标移出（MouseLeave）不应停拖
	e.HandleEvent(event.NewMouseEvent(event.TypeMouseLeave, 0, 0, event.ButtonNone, 0))
	if !e.dragging {
		t.Error("拖拽中移出滑块不应停拖")
	}
	e.HandleEvent(event.NewDragEvent(event.TypeDragEnd, 50, 10, event.ButtonLeft, 0))
	if e.dragging {
		t.Error("DragEnd 后应停拖")
	}
}
