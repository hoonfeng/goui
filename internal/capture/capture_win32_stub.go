//go:build !windows

package capture

import (
	"fmt"
	"image"
)

// ─────────────────────────────────────────────────────────────
// Win32Capturer Stub（非 Windows 平台不可用）
// ─────────────────────────────────────────────────────────────

type Win32Capturer struct {
	hwnd   uintptr
	width  int
	height int
}

func newWin32Capturer(hwnd uintptr) *Win32Capturer {
	return &Win32Capturer{
		hwnd:   hwnd,
		width:  0,
		height: 0,
	}
}

func (c *Win32Capturer) Capture() (*image.RGBA, error) {
	return nil, fmt.Errorf("capture: Win32 not supported on this platform")
}

func (c *Win32Capturer) CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	return nil, fmt.Errorf("capture: Win32 not supported on this platform")
}

func (c *Win32Capturer) Size() (int, int) {
	return 0, 0
}

var _ Capturer = (*Win32Capturer)(nil)
