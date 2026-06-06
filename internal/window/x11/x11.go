//go:build linux

// Package x11 提供了 Linux 平台的 X11 + GLX 窗口实现。
package x11

/*
#cgo LDFLAGS: -lX11 -lGL
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/keysym.h>
#include <GL/glx.h>
#include <GL/gl.h>
#include <stdlib.h>
*/
import "C"

import (
	"log"
	"sync"
	"unsafe"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/window"
)

// X11Window 实现 window.Window 接口
type X11Window struct {
	display   *C.Display
	screen    int
	window    C.Window
	width     int
	height    int
	title     string
	closed    bool
	dispatcher *event.Dispatcher
	vSync     bool
	glContext C.GLXContext
	mu        sync.Mutex
	atomClose C.Atom
}

// NewWindow 创建新的 X11 窗口
func NewWindow(config window.WindowConfig) (window.Window, error) {
	display := C.XOpenDisplay(nil)
	if display == nil {
		return nil, log.Output(2, "x11: cannot open X display")
	}

	screen := C.XDefaultScreen(display)
	root := C.XRootWindow(display, screen)

	// 创建窗口
	win := C.XCreateSimpleWindow(
		display, root,
		0, 0,
		C.uint(config.Width), C.uint(config.Height),
		0,
		C.XBlackPixel(display, screen),
		C.XWhitePixel(display, screen),
	)
	if win == 0 {
		C.XCloseDisplay(display)
		return nil, log.Output(2, "x11: failed to create window")
	}

	// 设置窗口标题
	titleC := C.CString(config.Title)
	defer C.free(unsafe.Pointer(titleC))
	C.XStoreName(display, win, titleC)

	// 设置事件掩码
	C.XSelectInput(display, win,
		C.ExposureMask|
			C.ButtonPressMask|C.ButtonReleaseMask|
			C.PointerMotionMask|
			C.KeyPressMask|C.KeyReleaseMask|
			C.StructureNotifyMask|
			C.EnterWindowMask|C.LeaveWindowMask|
			C.FocusChangeMask,
	)

	// 设置 WM_DELETE_WINDOW 协议（响应窗口关闭按钮）
	atomClose := C.XInternAtom(display, C.CString("WM_DELETE_WINDOW"), C.False)
	defer C.free(unsafe.Pointer(C.CString("WM_DELETE_WINDOW")))
	C.XSetWMProtocols(display, win, &atomClose, 1)

	// 创建 GLX 上下文
	attr := []int{
		C.GLX_RGBA,
		C.GLX_DOUBLEBUFFER,
		C.GLX_RED_SIZE, 8,
		C.GLX_GREEN_SIZE, 8,
		C.GLX_BLUE_SIZE, 8,
		C.GLX_ALPHA_SIZE, 8,
		C.GLX_DEPTH_SIZE, 24,
		C.None,
	}
	vi := C.glXChooseVisual(display, screen, (*C.int)(unsafe.Pointer(&attr[0])))
	if vi == nil {
		C.XDestroyWindow(display, win)
		C.XCloseDisplay(display)
		return nil, log.Output(2, "x11: glXChooseVisual failed")
	}

	cmap := C.XCreateColormap(display, root, vi.visual, C.AllocNone)
	var swa C.XSetWindowAttributes
	swa.colormap = cmap
	swa.event_mask = C.ExposureMask | C.KeyPressMask | C.ButtonPressMask | C.StructureNotifyMask
	C.XChangeWindowAttributes(display, win, C.CWColormap|C.CWEventMask, &swa)

	glCtx := C.glXCreateContext(display, vi, nil, C.GL_TRUE)
	if glCtx == nil {
		C.XDestroyWindow(display, win)
		C.XCloseDisplay(display)
		return nil, log.Output(2, "x11: glXCreateContext failed")
	}

	// 显示窗口
	C.XMapWindow(display, win)
	C.XFlush(display)

	w := &X11Window{
		display:    display,
		screen:     screen,
		window:     win,
		width:      config.Width,
		height:     config.Height,
		title:      config.Title,
		vSync:      config.VSync,
		dispatcher: event.NewDispatcher(),
		glContext:  glCtx,
		atomClose:  atomClose,
	}

	// 注册到 window 包
	window.NewWindow = NewWindow

	return w, nil
}

func (w *X11Window) Title() string                           { return w.title }
func (w *X11Window) Size() (int, int)                        { return w.width, w.height }
func (w *X11Window) Position() (int, int)                    { return 0, 0 } // XGetGeometry 可获取
func (w *X11Window) EventDispatcher() *event.Dispatcher      { return w.dispatcher }
func (w *X11Window) NativeHandle() uintptr                   { return uintptr(w.window) }
func (w *X11Window) IsClosed() bool                          { return w.closed }

func (w *X11Window) SetTitle(title string) {
	w.title = title
	titleC := C.CString(title)
	defer C.free(unsafe.Pointer(titleC))
	C.XStoreName(w.display, w.window, titleC)
}

func (w *X11Window) SetSize(width, height int) {
	w.width = width
	w.height = height
	C.XResizeWindow(w.display, w.window, C.uint(width), C.uint(height))
}

func (w *X11Window) SetPosition(x, y int) {
	C.XMoveWindow(w.display, w.window, C.int(x), C.int(y))
}

func (w *X11Window) Show() {
	C.XMapWindow(w.display, w.window)
}

func (w *X11Window) Hide() {
	C.XUnmapWindow(w.display, w.window)
}

func (w *X11Window) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.closed = true
		if w.glContext != nil {
			C.glXMakeCurrent(w.display, 0, nil)
			C.glXDestroyContext(w.display, w.glContext)
			w.glContext = nil
		}
		C.XDestroyWindow(w.display, w.window)
		C.XCloseDisplay(w.display)
		w.display = nil
	}
}

func (w *X11Window) SetVSync(enabled bool) {
	w.vSync = enabled
	// GLX 的 vsync 通过 glXSwapIntervalEXT 设置
	// 需要 GLX_EXT_swap_control 或 GLX_SGI_swap_control 扩展
}

func (w *X11Window) SwapBuffers() {
	C.glXSwapBuffers(w.display, w.window)
}

func (w *X11Window) MakeCurrent() error {
	ret := C.glXMakeCurrent(w.display, w.window, w.glContext)
	if ret == 0 {
		return log.Output(2, "x11: glXMakeCurrent failed")
	}
	return nil
}

// ProcessEvents 处理 X11 事件
func (w *X11Window) ProcessEvents() bool {
	if w.closed || w.display == nil {
		return false
	}

	for C.XPending(w.display) > 0 {
		var xev C.XEvent
		C.XNextEvent(w.display, &xev)

		switch typ := xev._type; typ {
		case C.ClientMessage:
			// 检查是否是 WM_DELETE_WINDOW
			cm := (*C.XClientMessageEvent)(unsafe.Pointer(&xev))
			if C.Atom(cm.data.l[0]) == w.atomClose {
				w.dispatcher.Dispatch(event.NewBaseEvent(event.TypeWindowClose))
				return false
			}

		case C.DestroyNotify:
			w.dispatcher.Dispatch(event.NewBaseEvent(event.TypeWindowClose))
			return false

		case C.Expose:
			if xev.pexpose.count == 0 {
				w.dispatcher.Dispatch(event.NewBaseEvent(event.TypeWindowPaint))
			}

		case C.ConfigureNotify:
			ce := (*C.XConfigureEvent)(unsafe.Pointer(&xev))
			newW := int(ce.width)
			newH := int(ce.height)
			if newW != w.width || newH != w.height {
				w.width = newW
				w.height = newH
				w.dispatcher.Dispatch(event.NewResizeEvent(newW, newH))
			}

		case C.ButtonPress:
			be := (*C.XButtonPressedEvent)(unsafe.Pointer(&xev))
			btn := x11ButtonToEventButton(int(be.button))
			ev := event.NewMouseEvent(event.TypeMouseDown,
				float64(be.x), float64(be.y), btn, x11ModsToEventMods(be.state))
			w.dispatcher.Dispatch(ev)

		case C.ButtonRelease:
			be := (*C.XButtonReleasedEvent)(unsafe.Pointer(&xev))
			btn := x11ButtonToEventButton(int(be.button))
			ev := event.NewMouseEvent(event.TypeMouseUp,
				float64(be.x), float64(be.y), btn, x11ModsToEventMods(be.state))
			w.dispatcher.Dispatch(ev)

		case C.MotionNotify:
			me := (*C.XMotionEvent)(unsafe.Pointer(&xev))
			ev := event.NewMouseEvent(event.TypeMouseMove,
				float64(me.x), float64(me.y), event.ButtonNone, x11ModsToEventMods(me.state))
			w.dispatcher.Dispatch(ev)

		case C.EnterNotify:
			ev := event.NewMouseEvent(event.TypeMouseEnter, 0, 0, event.ButtonNone, event.ModNone)
			w.dispatcher.Dispatch(ev)

		case C.LeaveNotify:
			ev := event.NewMouseEvent(event.TypeMouseLeave, 0, 0, event.ButtonNone, event.ModNone)
			w.dispatcher.Dispatch(ev)

		case C.KeyPress:
			ke := (*C.XKeyPressedEvent)(unsafe.Pointer(&xev))
			// 获取 keysym
			ks := C.XLookupKeysym((*C.XKeyEvent)(unsafe.Pointer(&xev)), 0)
			keyName := x11KeysymToString(C.long(ks))
			vkCode := uint32(ks)

			ev := event.NewKeyEvent(event.TypeKeyDown, vkCode, keyName, x11ModsToEventMods(ke.state))
			w.dispatcher.Dispatch(ev)

			// 生成字符事件
			var buf [32]C.char
			len := C.XLookupString((*C.XKeyEvent)(unsafe.Pointer(&xev)),
				(*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)), nil, nil)
			if len > 0 {
				charEv := event.NewKeyEvent(event.TypeKeyChar,
					uint32(buf[0]), C.GoStringN(&buf[0], C.int(len)), x11ModsToEventMods(ke.state))
				charEv.Char = rune(buf[0])
				w.dispatcher.Dispatch(charEv)
			}

		case C.KeyRelease:
			ke := (*C.XKeyReleasedEvent)(unsafe.Pointer(&xev))
			ks := C.XLookupKeysym((*C.XKeyEvent)(unsafe.Pointer(&xev)), 0)
			keyName := x11KeysymToString(C.long(ks))
			ev := event.NewKeyEvent(event.TypeKeyUp, uint32(ks), keyName, x11ModsToEventMods(ke.state))
			w.dispatcher.Dispatch(ev)

		case C.FocusIn:
			w.dispatcher.Dispatch(event.NewBaseEvent(event.TypeFocusIn))

		case C.FocusOut:
			w.dispatcher.Dispatch(event.NewBaseEvent(event.TypeFocusOut))
		}
	}

	return !w.closed
}

// WaitMessage 阻塞等待下一条 X11 事件。
// X11 使用 XPending + XNextEvent 处理事件，ProcessEvents 会阻塞等待消息。
// 此方法作为接口占位，X11 的事件泵送本身会阻塞。
func (w *X11Window) WaitMessage() {
	// X11 中通过 XNextEvent 阻塞等待事件，ProcessEvents 已包含阻塞语义
}

// x11ButtonToEventButton 转换 X11 鼠标按键到事件系统
func x11ButtonToEventButton(btn int) event.MouseButton {
	switch btn {
	case C.Button1:
		return event.ButtonLeft
	case C.Button2:
		return event.ButtonMiddle
	case C.Button3:
		return event.ButtonRight
	case 4: // 滚轮上
		return event.ButtonNone
	case 5: // 滚轮下
		return event.ButtonNone
	default:
		return event.ButtonNone
	}
}

// x11ModsToEventMods 转换 X11 修饰键到事件系统
func x11ModsToEventMods(state C.uint) event.ModifierKeys {
	var mods event.ModifierKeys
	if state&C.ShiftMask != 0 {
		mods |= event.ModShift
	}
	if state&C.ControlMask != 0 {
		mods |= event.ModCtrl
	}
	if state&C.Mod1Mask != 0 { // Alt
		mods |= event.ModAlt
	}
	if state&C.Mod4Mask != 0 { // Super/Windows
		mods |= event.ModSuper
	}
	return mods
}

// x11KeysymToString 转换 X11 keysym 到字符串标识
func x11KeysymToString(ks C.long) string {
	ksMap := map[C.long]string{
		C.XK_BackSpace: "Backspace",
		C.XK_Tab:       "Tab",
		C.XK_Return:    "Enter",
		C.XK_Escape:    "Escape",
		C.XK_Delete:    "Delete",
		C.XK_Home:      "Home",
		C.XK_End:       "End",
		C.XK_Left:      "ArrowLeft",
		C.XK_Right:     "ArrowRight",
		C.XK_Up:        "ArrowUp",
		C.XK_Down:      "ArrowDown",
		C.XK_Page_Up:   "PageUp",
		C.XK_Page_Down: "PageDown",
		C.XK_Shift_L:   "Shift",
		C.XK_Shift_R:   "Shift",
		C.XK_Control_L: "Ctrl",
		C.XK_Control_R: "Ctrl",
		C.XK_Alt_L:     "Alt",
		C.XK_Alt_R:     "Alt",
		C.XK_Super_L:   "Super",
		C.XK_Super_R:   "Super",
		C.XK_space:     "Space",
		C.XK_Insert:    "Insert",
	}
	for xksym, name := range ksMap {
		if ks == xksym {
			return name
		}
	}
	return ""
}

// Init 初始化 X11 平台支持
func Init() {
	window.NewWindow = NewWindow
}

func init() {
	Init()
}
