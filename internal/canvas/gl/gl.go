// Package gl 提供了 OpenGL 1.1 核心函数的 Go 绑定。
// 使用 Windows 的 opengl32.dll 导出函数，通过 syscall 直接调用。
// 注意：这是最小绑定，仅包含 goui 需要的 OpenGL 1.1 函数。
package gl

import (
	"math"
	"syscall"
	"unsafe"
)

var (
	opengl32 = syscall.NewLazyDLL("opengl32.dll")

	// OpenGL 1.1 核心函数
	glClear_            = opengl32.NewProc("glClear")
	glClearColor_       = opengl32.NewProc("glClearColor")
	glClearDepth_       = opengl32.NewProc("glClearDepth")
	glBegin_            = opengl32.NewProc("glBegin")
	glEnd_              = opengl32.NewProc("glEnd")
	glVertex2f_         = opengl32.NewProc("glVertex2f")
	glVertex2d_         = opengl32.NewProc("glVertex2d")
	glColor4f_          = opengl32.NewProc("glColor4f")
	glColor4ub_         = opengl32.NewProc("glColor4ub")
	glPushMatrix_       = opengl32.NewProc("glPushMatrix")
	glPopMatrix_        = opengl32.NewProc("glPopMatrix")
	glTranslatef_       = opengl32.NewProc("glTranslatef")
	glRotatef_          = opengl32.NewProc("glRotatef")
	glScalef_           = opengl32.NewProc("glScalef")
	glViewport_         = opengl32.NewProc("glViewport")
	glOrtho_            = opengl32.NewProc("glOrtho")
	glMatrixMode_       = opengl32.NewProc("glMatrixMode")
	glLoadIdentity_     = opengl32.NewProc("glLoadIdentity")
	glFlush_            = opengl32.NewProc("glFlush")
	glGetError_         = opengl32.NewProc("glGetError")
	glEnable_           = opengl32.NewProc("glEnable")
	glDisable_          = opengl32.NewProc("glDisable")
	glBlendFunc_        = opengl32.NewProc("glBlendFunc")
	glHint_             = opengl32.NewProc("glHint")
	glLineWidth_        = opengl32.NewProc("glLineWidth")
	glPointSize_        = opengl32.NewProc("glPointSize")
	glScissor_          = opengl32.NewProc("glScissor")
	glEnableClientState_ = opengl32.NewProc("glEnableClientState")
	glDisableClientState_ = opengl32.NewProc("glDisableClientState")
	glVertexPointer_    = opengl32.NewProc("glVertexPointer")
	glDrawArrays_       = opengl32.NewProc("glDrawArrays")
	glColorPointer_     = opengl32.NewProc("glColorPointer")

	// 纹理函数
	glGenTextures_      = opengl32.NewProc("glGenTextures")
	glDeleteTextures_   = opengl32.NewProc("glDeleteTextures")
	glBindTexture_      = opengl32.NewProc("glBindTexture")
	glTexParameteri_    = opengl32.NewProc("glTexParameteri")
	glTexImage2D_       = opengl32.NewProc("glTexImage2D")
	glTexEnvi_          = opengl32.NewProc("glTexEnvi")
	glTexCoord2f_       = opengl32.NewProc("glTexCoord2f")
	glIsTexture_        = opengl32.NewProc("glIsTexture")

	// WGL 上下文函数
	wglGetCurrentContext_ = opengl32.NewProc("wglGetCurrentContext")
)

// OpenGL 常量
const (
	ClearColorBuffer   = 0x00004000
	ClearDepthBuffer   = 0x00000100
	ClearStencilBuffer = 0x00000400

	MatrixProjection = 0x1701
	MatrixModelView  = 0x1700
	MatrixTexture    = 0x1702

	PrimitivePoints    = 0x0000
	PrimitiveLines     = 0x0001
	PrimitiveLineLoop  = 0x0002
	PrimitiveLineStrip = 0x0003
	PrimitiveTriangles = 0x0004
	PrimitiveTriFan    = 0x0006
	PrimitiveQuads     = 0x0007
	PrimitivePolygon   = 0x0009

	EnableBlend        = 0x0BE2
	EnableTexture2D    = 0x0DE1
	EnableScissorTest  = 0x0C11
	EnablePointSmooth  = 0x0B10
	EnableLineSmooth   = 0x0B20
	EnablePolygonSmooth = 0x0B41

	BlendSrcAlpha          = 0x0302
	BlendOneMinusSrcAlpha  = 0x0303
	BlendOne               = 0x0001
	BlendZero              = 0x0000
	BlendSrcColor          = 0x0300

	HintPerspective  = 0x0C50
	HintPointSmooth  = 0x0C51
	HintLineSmooth   = 0x0C52
	HintPolygonSmooth = 0x0C53
	HintFastest      = 0x1102
	HintNicest       = 0x1106
	HintDontCare     = 0x1100

	TextureWrapS       = 0x2802
	TextureWrapT       = 0x2803
	TextureMagFilter   = 0x2800
	TextureMinFilter   = 0x2801
	TextureWrapClamp   = 0x812F
	TextureWrapRepeat  = 0x2901
	TextureLinear      = 0x2601
	TextureNearest     = 0x2600

	TexEnvMode         = 0x2200
	TexEnvModulate     = 0x2100
	TexEnvDecal        = 0x2101
	TexEnvReplace      = 0x2102

	ClientVertexArray  = 0x8074
	ClientColorArray   = 0x8076

	NoError = 0

	// 像素格式常量（用于 TexImage2D / DrawPixels）
	RGBA          = 0x1908
	RGBA8         = 0x8058
	UnsignedByte  = 0x1401
)

// Clear 清除缓冲区
func Clear(mask uint32) {
	glClear_.Call(uintptr(mask))
}

// ClearColor 设置清除颜色
func ClearColor(r, g, b, a float32) {
	glClearColor_.Call(
		uintptr(math.Float32bits(r)),
		uintptr(math.Float32bits(g)),
		uintptr(math.Float32bits(b)),
		uintptr(math.Float32bits(a)),
	)
}

// ClearDepth 设置深度缓冲区清除值
func ClearDepth(depth float64) {
	glClearDepth_.Call(uintptr(math.Float64bits(depth)))
}

// Begin 开始绘制图元
func Begin(mode uint32) {
	glBegin_.Call(uintptr(mode))
}

// End 结束绘制图元
func End() {
	glEnd_.Call()
}

// Vertex2f 指定二维顶点（float32）
func Vertex2f(x, y float32) {
	glVertex2f_.Call(f32(x), f32(y))
}

// Vertex2d 指定二维顶点（float64）
func Vertex2d(x, y float64) {
	glVertex2d_.Call(f64(x), f64(y))
}

// Color4f 设置当前颜色（float32 分量 0.0-1.0）
func Color4f(r, g, b, a float32) {
	glColor4f_.Call(f32(r), f32(g), f32(b), f32(a))
}

// Color4ub 设置当前颜色（uint8 分量 0-255）
func Color4ub(r, g, b, a uint8) {
	glColor4ub_.Call(uintptr(r), uintptr(g), uintptr(b), uintptr(a))
}

// PushMatrix 压入当前矩阵
func PushMatrix() {
	glPushMatrix_.Call()
}

// PopMatrix 弹出矩阵
func PopMatrix() {
	glPopMatrix_.Call()
}

// Translatef 平移变换
func Translatef(x, y, z float32) {
	glTranslatef_.Call(f32(x), f32(y), f32(z))
}

// Rotatef 旋转变换（角度制）
func Rotatef(angle, x, y, z float32) {
	glRotatef_.Call(f32(angle), f32(x), f32(y), f32(z))
}

// Scalef 缩放变换
func Scalef(x, y, z float32) {
	glScalef_.Call(f32(x), f32(y), f32(z))
}

// Viewport 设置视口
func Viewport(x, y, width, height int32) {
	glViewport_.Call(uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

// Ortho 设置正交投影
func Ortho(left, right, bottom, top, near, far float64) {
	glOrtho_.Call(f64(left), f64(right), f64(bottom), f64(top), f64(near), f64(far))
}

// MatrixMode 设置当前矩阵模式
func MatrixMode(mode uint32) {
	glMatrixMode_.Call(uintptr(mode))
}

// LoadIdentity 加载单位矩阵
func LoadIdentity() {
	glLoadIdentity_.Call()
}

// Flush 刷新 OpenGL 命令
func Flush() {
	glFlush_.Call()
}

// GetError 获取 OpenGL 错误
func GetError() uint32 {
	ret, _, _ := glGetError_.Call()
	return uint32(ret)
}

// Enable 启用 OpenGL 特性
func Enable(cap uint32) {
	glEnable_.Call(uintptr(cap))
}

// Disable 禁用 OpenGL 特性
func Disable(cap uint32) {
	glDisable_.Call(uintptr(cap))
}

// BlendFunc 设置混合函数
func BlendFunc(sfactor, dfactor uint32) {
	glBlendFunc_.Call(uintptr(sfactor), uintptr(dfactor))
}

// Hint 设置提示模式
func Hint(target, mode uint32) {
	glHint_.Call(uintptr(target), uintptr(mode))
}

// LineWidth 设置线宽
func LineWidth(width float32) {
	glLineWidth_.Call(f32(width))
}

// PointSize 设置点大小
func PointSize(size float32) {
	glPointSize_.Call(f32(size))
}

// Scissor 设置裁剪矩形
func Scissor(x, y, width, height int32) {
	glScissor_.Call(uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

// EnableClientState 启用客户端状态
func EnableClientState(array uint32) {
	glEnableClientState_.Call(uintptr(array))
}

// DisableClientState 禁用客户端状态
func DisableClientState(array uint32) {
	glDisableClientState_.Call(uintptr(array))
}

// VertexPointer 设置顶点指针
func VertexPointer(size int32, type_, stride int32, ptr unsafe.Pointer) {
	glVertexPointer_.Call(uintptr(size), uintptr(type_), uintptr(stride), uintptr(ptr))
}

// ColorPointer 设置颜色指针
func ColorPointer(size int32, type_, stride int32, ptr unsafe.Pointer) {
	glColorPointer_.Call(uintptr(size), uintptr(type_), uintptr(stride), uintptr(ptr))
}

// DrawArrays 绘制数组
func DrawArrays(mode uint32, first, count int32) {
	glDrawArrays_.Call(uintptr(mode), uintptr(first), uintptr(count))
}

// --- 上下文状态函数 ---

// HasContext 检查当前线程是否有有效的 OpenGL 渲染上下文。
// 在调用任何 OpenGL 函数前调用此函数可避免因上下文缺失导致的崩溃。
func HasContext() bool {
	ctx, _, _ := wglGetCurrentContext_.Call()
	return ctx != 0
}

// IsTexture 检查指定的纹理 ID 是否是有效的 OpenGL 纹理名称。
func IsTexture(texture uint32) bool {
	if texture == 0 {
		return false
	}
	ret, _, _ := glIsTexture_.Call(uintptr(texture))
	return ret != 0
}

// --- 纹理函数 ---

// GenTextures 生成纹理名称。
// 在调用底层 glGenTextures 前检查上下文有效性，若无上下文则返回 0 以避免崩溃。
func GenTextures(n int32) uint32 {
	if !HasContext() {
		return 0
	}
	var texture uint32
	glGenTextures_.Call(uintptr(n), uintptr(unsafe.Pointer(&texture)))
	return texture
}

// DeleteTextures 删除纹理
func DeleteTextures(n int32, textures *uint32) {
	glDeleteTextures_.Call(uintptr(n), uintptr(unsafe.Pointer(textures)))
}

// BindTexture 绑定纹理
func BindTexture(target, texture uint32) {
	glBindTexture_.Call(uintptr(target), uintptr(texture))
}

// TexParameteri 设置纹理参数
func TexParameteri(target, pname, param uint32) {
	glTexParameteri_.Call(uintptr(target), uintptr(pname), uintptr(param))
}

// TexImage2D 上传二维纹理图像
func TexImage2D(target, level, internalFormat uint32, width, height int32, border uint32, format, type_ uint32, pixels unsafe.Pointer) {
	glTexImage2D_.Call(
		uintptr(target), uintptr(level), uintptr(internalFormat),
		uintptr(width), uintptr(height),
		uintptr(border), uintptr(format), uintptr(type_),
		uintptr(pixels),
	)
}

// TexEnvi 设置纹理环境
func TexEnvi(target, pname, param uint32) {
	glTexEnvi_.Call(uintptr(target), uintptr(pname), uintptr(param))
}

// TexCoord2f 设置纹理坐标
func TexCoord2f(s, t float32) {
	glTexCoord2f_.Call(f32(s), f32(t))
}

// --- 辅助函数 ---

func f32(f float32) uintptr {
	return uintptr(math.Float32bits(f))
}

func f64(f float64) uintptr {
	return uintptr(math.Float64bits(f))
}
