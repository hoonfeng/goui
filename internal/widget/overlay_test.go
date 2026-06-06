package widget

import (
	"testing"

	"github.com/user/goui/internal/types"
)

var (
	ovAnchor = types.Rect{X: 300, Y: 200, Width: 100, Height: 40}
	ovVP     = types.Size{Width: 800, Height: 600}
	ovC      = types.Size{Width: 120, Height: 50}
)

func TestOverlayPlaceBottom(t *testing.T) {
	pos, pl := computeOverlayPosition(ovAnchor, ovC, PlacementBottom, ovVP, 8)
	if pos.Y != 248 { // 200+40+8
		t.Errorf("bottom Y=%.0f, want 248", pos.Y)
	}
	if pos.X != 290 { // 300+(100-120)/2
		t.Errorf("bottom X(居中)=%.0f, want 290", pos.X)
	}
	if pl != PlacementBottom {
		t.Errorf("placement=%d, want bottom", pl)
	}
}

func TestOverlayPlaceTop(t *testing.T) {
	pos, _ := computeOverlayPosition(ovAnchor, ovC, PlacementTop, ovVP, 8)
	if pos.Y != 142 { // 200-50-8
		t.Errorf("top Y=%.0f, want 142", pos.Y)
	}
}

func TestOverlayFlipTopToBottom(t *testing.T) {
	a := types.Rect{X: 300, Y: 20, Width: 100, Height: 40} // 贴顶
	pos, pl := computeOverlayPosition(a, ovC, PlacementTop, ovVP, 8)
	if pl != PlacementBottom {
		t.Errorf("贴顶 top 放不下应翻转到 bottom，got placement=%d", pl)
	}
	if pos.Y != 68 { // 20+40+8
		t.Errorf("翻转后 Y=%.0f, want 68", pos.Y)
	}
}

func TestOverlayFlipBottomToTop(t *testing.T) {
	a := types.Rect{X: 300, Y: 540, Width: 100, Height: 40} // 贴底
	_, pl := computeOverlayPosition(a, ovC, PlacementBottom, ovVP, 8)
	if pl != PlacementTop {
		t.Errorf("贴底 bottom 放不下应翻转到 top，got placement=%d", pl)
	}
}

func TestOverlayCrossAlign(t *testing.T) {
	ps, _ := computeOverlayPosition(ovAnchor, ovC, PlacementBottomStart, ovVP, 8)
	if ps.X != 300 {
		t.Errorf("bottom-start X=%.0f, want 300(对齐锚点左缘)", ps.X)
	}
	pe, _ := computeOverlayPosition(ovAnchor, ovC, PlacementBottomEnd, ovVP, 8)
	if pe.X != 280 { // 300+100-120
		t.Errorf("bottom-end X=%.0f, want 280(对齐锚点右缘)", pe.X)
	}
}

func TestOverlayRightLeft(t *testing.T) {
	pr, _ := computeOverlayPosition(ovAnchor, ovC, PlacementRight, ovVP, 8)
	if pr.X != 408 { // 300+100+8
		t.Errorf("right X=%.0f, want 408", pr.X)
	}
	if pr.Y != 195 { // 200+(40-50)/2
		t.Errorf("right Y(居中)=%.0f, want 195", pr.Y)
	}
	pl, _ := computeOverlayPosition(ovAnchor, ovC, PlacementLeft, ovVP, 8)
	if pl.X != 172 { // 300-120-8
		t.Errorf("left X=%.0f, want 172", pl.X)
	}
}

func TestOverlayShiftIntoViewport(t *testing.T) {
	a := types.Rect{X: 750, Y: 200, Width: 100, Height: 40} // 靠右边缘
	pos, _ := computeOverlayPosition(a, ovC, PlacementBottom, ovVP, 8)
	if pos.X+ovC.Width > ovVP.Width {
		t.Errorf("浮层右边界 %.0f 超出视口 %.0f", pos.X+ovC.Width, ovVP.Width)
	}
	if pos.X != 680 { // 800-120
		t.Errorf("shift 后 X=%.0f, want 680", pos.X)
	}
}
