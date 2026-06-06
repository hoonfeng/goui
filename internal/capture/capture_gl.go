//go:build windows

package capture

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

// OpenGL 函数（通过 opengl32.dll）
var (
	opengl32        = syscall.NewLazyDLL("opengl32.dll")
	glReadPixels_   = opengl32.NewProc("glReadPixels")
	glGetError_     = opengl32.NewProc("glGetError")
	wglGetCurrentContext_ = opengl32.NewProc("wglGetCurrentContext")
)

// OpenGL 常量
const (
	_GL_BACK       = 0x0405     // GL_BACK
	_GL_FRONT      = 0x0404     // GL_FRONT
	_GL_RGBA       = 0x1908     // GL_RGBA
	_GL_UNSIGNED_BYTE = 0x1401 // GL_UNSIGNED_BYTE
	_GL_NO_ERROR   = 0          // GL_NO_ERROR
)

// ─────────────────────────────────────────────────────────────
// GLCapturer — 使用 OpenGL glReadPixels 截取渲染表面
// ─────────────────────────────────────────────────────────────

// GLCapturer 使用 OpenGL glReadPixels 从后缓冲区读取像素数据。
// 必须在 OpenGL 上下文当前的线程中调用 Capture()。
type GLCapturer struct {
	width  int
	height int
}

// newGLCapturer 创建 OpenGL 截图捕获器。
// width 和 height 指定视口尺寸。
func newGLCapturer(width, height int) *GLCapturer {
	return &GLCapturer{
		width:  width,
		height: height,
	}
}

func (c *GLCapturer) Capture() (*image.RGBA, error) {
	return c.CaptureRect(image.Rect(0, 0, c.width, c.height))
}

func (c *GLCapturer) CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	// 检查 OpenGL 上下文
	ctx, _, _ := wglGetCurrentContext_.Call()
	if ctx == 0 {
		return nil, fmt.Errorf("capture: no OpenGL context current")
	}

	imgW := rect.Dx()
	imgH := rect.Dy()
	if imgW <= 0 || imgH <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
	}

	// 创建输出图像
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))

	// glReadPixels：从后缓冲区读取像素
	// 注意：OpenGL 原点在左下角，图像需要垂直翻转
	glReadPixels_.Call(
		uintptr(int32(rect.Min.X)),
		uintptr(int32(rect.Min.Y)),
		uintptr(int32(imgW)),
		uintptr(int32(imgH)),
		_GL_RGBA,
		_GL_UNSIGNED_BYTE,
		uintptr(unsafe.Pointer(&img.Pix[0])),
	)

	// 检查 OpenGL 错误
	errCode, _, _ := glGetError_.Call()
	if errCode != _GL_NO_ERROR {
		return nil, fmt.Errorf("capture: glReadPixels error code: %d", errCode)
	}

	// 垂直翻转（OpenGL 原点在左下角，Go image 原点在左上角）
	flipVertically(img.Pix, imgW, imgH)

	return img, nil
}

func (c *GLCapturer) Size() (int, int) {
	return c.width, c.height
}

// ─────────────────────────────────────────────────────────────
// 内部函数
// ─────────────────────────────────────────────────────────────

// flipVertically 垂直翻转 RGBA 像素数据。
func flipVertically(pix []byte, width, height int) {
	rowSize := width * 4
	for y := 0; y < height/2; y++ {
		top := y * rowSize
		bottom := (height - 1 - y) * rowSize
		for x := 0; x < rowSize; x++ {
			pix[top+x], pix[bottom+x] = pix[bottom+x], pix[top+x]
		}
	}
}

// openGLContextAvailable 检查当前线程是否有 OpenGL 上下文。
var openGLContextAvailable = func() bool { return false }

// init 初始化 OpenGL 上下文可用性检查。
func init() {
	openGLContextAvailable = func() bool {
		ctx, _, _ := wglGetCurrentContext_.Call()
		return ctx != 0
	}
}

// Ensure interface compliance
var _ Capturer = (*GLCapturer)(nil)
