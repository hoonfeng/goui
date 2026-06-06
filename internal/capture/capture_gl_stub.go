//go:build !windows

package capture

import (
	"fmt"
	"image"
)

// ─────────────────────────────────────────────────────────────
// GLCapturer Stub（非 Windows 平台不可用）
// ─────────────────────────────────────────────────────────────

type GLCapturer struct {
	width  int
	height int
}

func newGLCapturer(width, height int) *GLCapturer {
	return &GLCapturer{
		width:  width,
		height: height,
	}
}

func (c *GLCapturer) Capture() (*image.RGBA, error) {
	return nil, fmt.Errorf("capture: OpenGL capture not supported on this platform")
}

func (c *GLCapturer) CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	return nil, fmt.Errorf("capture: OpenGL capture not supported on this platform")
}

func (c *GLCapturer) Size() (int, int) {
	return 0, 0
}

var _ Capturer = (*GLCapturer)(nil)
