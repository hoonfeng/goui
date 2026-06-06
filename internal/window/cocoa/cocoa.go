//go:build darwin

// Package cocoa 提供了 macOS 平台的 Cocoa + NSOpenGL 窗口实现。
package cocoa

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework OpenGL -framework Carbon
#import <Cocoa/Cocoa.h>
#import <OpenGL/OpenGL.h>
#import <OpenGL/gl.h>

// Application 委托
@interface AppDelegate : NSObject <NSApplicationDelegate, NSWindowDelegate>
@property (assign) BOOL running;
@property (assign) BOOL needsDisplay;
@end

static AppDelegate *sharedDelegate = nil;

@implementation AppDelegate
- (id)init {
	self = [super init];
	if (self) {
		self.running = YES;
		self.needsDisplay = NO;
	}
	return self;
}
- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
	return YES;
}
- (NSApplicationTerminateReply)applicationShouldTerminate:(NSApplication *)sender {
	self.running = NO;
	return NSTerminateCancel;
}
- (void)windowWillClose:(NSNotification *)notification {
	self.running = NO;
}
- (void)windowDidResize:(NSNotification *)notification {
	self.needsDisplay = YES;
}
@end

// OpenGL View
@interface GOpenGLView : NSOpenGLView
@end

@implementation GOpenGLView
- (void)drawRect:(NSRect)dirtyRect {
	[[self openGLContext] makeCurrentContext];
	[[self openGLContext] update];
	// 标记需要绘制，由 Go 层处理
	AppDelegate *delegate = (AppDelegate*)[NSApplication sharedApplication].delegate;
	delegate.needsDisplay = YES;
}
- (void)keyDown:(NSEvent *)event {
	// 由 Go 层通过 event monitor 处理
	[super keyDown:event];
}
@end

// 创建窗口函数
static void* CreateCocoaWindow(const char* title, int width, int height, BOOL resizable) {
	NSApplication *app = [NSApplication sharedApplication];
	[app setActivationPolicy:NSApplicationActivationPolicyRegular];

	// 创建委托
	if (sharedDelegate == nil) {
		sharedDelegate = [[AppDelegate alloc] init];
	}
	[app setDelegate:sharedDelegate];

	// 创建窗口
	NSRect frame = NSMakeRect(0, 0, width, height);
	NSUInteger styleMask = NSTitledWindowMask | NSClosableWindowMask | NSMiniaturizableWindowMask;
	if (resizable) {
		styleMask |= NSResizableWindowMask;
	}

	NSWindow *window = [[NSWindow alloc] initWithContentRect:frame
		styleMask:styleMask
		backing:NSBackingStoreBuffered
		defer:NO];
	[window setTitle:[NSString stringWithUTF8String:title]];
	[window setDelegate:sharedDelegate];
	[window center];

	// 创建 OpenGL 属性
	NSOpenGLPixelFormatAttribute attrs[] = {
		NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersionLegacy,
		NSOpenGLPFADoubleBuffer,
		NSOpenGLPFAColorSize, 32,
		NSOpenGLPFADepthSize, 24,
		NSOpenGLPFAStencilSize, 8,
		NSOpenGLPFAAccelerated,
		0
	};
	NSOpenGLPixelFormat *pf = [[NSOpenGLPixelFormat alloc] initWithAttributes:attrs];
	if (pf == nil) {
		return nil;
	}

	// 创建 OpenGL View
	GOpenGLView *glView = [[GOpenGLView alloc] initWithFrame:frame pixelFormat:pf];
	[window setContentView:glView];
	[window makeFirstResponder:glView];

	// 设置窗口可见
	[window makeKeyAndOrderFront:nil];
	[app activateIgnoringOtherApps:YES];

	return (__bridge_retained void*)window;
}

// 交换缓冲区
static void SwapCocoaBuffers(void* windowPtr) {
	NSWindow *window = (__bridge NSWindow*)windowPtr;
	NSOpenGLView *glView = (NSOpenGLView*)[window contentView];
	[[glView openGLContext] flushBuffer];
}

// 使 OpenGL 上下文当前
static void MakeCocoaCurrent(void* windowPtr) {
	NSWindow *window = (__bridge NSWindow*)windowPtr;
	NSOpenGLView *glView = (NSOpenGLView*)[window contentView];
	[[glView openGLContext] makeCurrentContext];
}

// 调整窗口尺寸
static void ResizeCocoaWindow(void* windowPtr, int width, int height) {
	NSWindow *window = (__bridge NSWindow*)windowPtr;
	NSRect frame = [window frame];
	frame.size.width = width;
	frame.size.height = height;
	[window setFrame:frame display:YES];
}

// 获取窗口尺寸
static void GetCocoaWindowSize(void* windowPtr, int* width, int* height) {
	NSWindow *window = (__bridge NSWindow*)windowPtr;
	NSRect frame = [window contentRectForFrameRect:[window frame]];
	*width = (int)frame.size.width;
	*height = (int)frame.size.height;
}

// 设置窗口标题
static void SetCocoaWindowTitle(void* windowPtr, const char* title) {
	NSWindow *window = (__bridge NSWindow*)windowPtr;
	[window setTitle:[NSString stringWithUTF8String:title]];
}

// 处理事件（一次迭代）
static int ProcessCocoaEvents(void* windowPtr) {
	NSApplication *app = [NSApplication sharedApplication];
	NSEvent *event = [app nextEventMatchingMask:NSAnyEventMask
		untilDate:[NSDate distantPast]
		inMode:NSDefaultRunLoopMode
		dequeue:YES];
	if (event) {
		[app sendEvent:event];
		[app updateWindows];
	}
	return (sharedDelegate != nil && sharedDelegate.running) ? 1 : 0;
}

// 检查是否需要重绘
static int NeedsCocoaDisplay(void) {
	if (sharedDelegate == nil) return 0;
	return sharedDelegate.needsDisplay ? 1 : 0;
}

// 重置重绘标志
static void ResetCocoaDisplayFlag(void) {
	if (sharedDelegate != nil) {
		sharedDelegate.needsDisplay = NO;
	}
}

// 关闭窗口
static void CloseCocoaWindow(void* windowPtr) {
	NSWindow *window = (__bridge_transfer NSWindow*)windowPtr;
	[window close];
}

// 获取窗口是否关闭
static int IsCocoaWindowClosed(void* windowPtr) {
	if (sharedDelegate == nil) return 1;
	return sharedDelegate.running ? 0 : 1;
}

// 获取窗口原生句柄
static uintptr_t GetCocoaWindowHandle(void* windowPtr) {
	return (uintptr_t)windowPtr;
}
*/
import "C"

import (
	"log"
	"sync"
	"unsafe"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/window"
)

// CocoaWindow 实现 window.Window 接口
type CocoaWindow struct {
	windowPtr  unsafe.Pointer
	width      int
	height     int
	title      string
	closed     bool
	dispatcher *event.Dispatcher
	vSync      bool
	mu         sync.Mutex
}

// NewWindow 创建新的 macOS 窗口
func NewWindow(config window.WindowConfig) (window.Window, error) {
	titleC := C.CString(config.Title)
	defer C.free(unsafe.Pointer(titleC))

	resizable := C.BOOL(0)
	if config.Resizable {
		resizable = C.BOOL(1)
	}

	ptr := C.CreateCocoaWindow(titleC, C.int(config.Width), C.int(config.Height), resizable)
	if ptr == nil {
		return nil, log.Output(2, "cocoa: failed to create window")
	}

	w := &CocoaWindow{
		windowPtr:  ptr,
		width:      config.Width,
		height:     config.Height,
		title:      config.Title,
		vSync:      config.VSync,
		dispatcher: event.NewDispatcher(),
	}

	// 获取实际尺寸
	var cw, ch C.int
	C.GetCocoaWindowSize(ptr, &cw, &ch)
	w.width = int(cw)
	w.height = int(ch)

	// 注册到 window 包
	window.NewWindow = NewWindow

	return w, nil
}

func (w *CocoaWindow) Title() string                      { return w.title }
func (w *CocoaWindow) Size() (int, int)                   { return w.width, w.height }
func (w *CocoaWindow) Position() (int, int)               { return 0, 0 }
func (w *CocoaWindow) EventDispatcher() *event.Dispatcher { return w.dispatcher }
func (w *CocoaWindow) NativeHandle() uintptr {
	return uintptr(C.GetCocoaWindowHandle(w.windowPtr))
}
func (w *CocoaWindow) IsClosed() bool { return w.closed }

func (w *CocoaWindow) SetTitle(title string) {
	w.title = title
	titleC := C.CString(title)
	defer C.free(unsafe.Pointer(titleC))
	C.SetCocoaWindowTitle(w.windowPtr, titleC)
}

func (w *CocoaWindow) SetSize(width, height int) {
	w.width = width
	w.height = height
	C.ResizeCocoaWindow(w.windowPtr, C.int(width), C.int(height))
}

func (w *CocoaWindow) SetPosition(x, y int) {
	// Cocoa 窗口位置通过 setFrame 设置
	// 简化版本不实现
}

func (w *CocoaWindow) Show() {
	// 窗口默认已在创建时显示
}

func (w *CocoaWindow) Hide() {
	// Cocoa 窗口隐藏: [window orderOut:nil]
	// 简化版本不实现
}

func (w *CocoaWindow) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed && w.windowPtr != nil {
		w.closed = true
		C.CloseCocoaWindow(w.windowPtr)
		w.windowPtr = nil
	}
}

func (w *CocoaWindow) SetVSync(enabled bool) {
	w.vSync = enabled
	// NSOpenGL 的 vsync 通过 CGLSetParameter 设置
	// 简化版本不实现
}

func (w *CocoaWindow) SwapBuffers() {
	if w.windowPtr != nil {
		C.SwapCocoaBuffers(w.windowPtr)
	}
}

func (w *CocoaWindow) MakeCurrent() error {
	if w.windowPtr == nil {
		return log.Output(2, "cocoa: window is nil")
	}
	C.MakeCocoaCurrent(w.windowPtr)
	return nil
}

// ProcessEvents 处理 Cocoa 事件
func (w *CocoaWindow) ProcessEvents() bool {
	if w.closed || w.windowPtr == nil {
		return false
	}

	// 处理一个事件循环迭代
	ret := C.ProcessCocoaEvents(w.windowPtr)
	if ret == 0 {
		w.closed = true
		return false
	}

	// 检查是否需要重绘
	if C.NeedsCocoaDisplay() != 0 {
		C.ResetCocoaDisplayFlag()
		w.dispatcher.Dispatch(event.NewBaseEvent(event.TypeWindowPaint))
	}

	// 获取最新尺寸
	var cw, ch C.int
	C.GetCocoaWindowSize(w.windowPtr, &cw, &ch)
	newW := int(cw)
	newH := int(ch)
	if newW != w.width || newH != w.height {
		w.width = newW
		w.height = newH
		w.dispatcher.Dispatch(event.NewResizeEvent(newW, newH))
	}

	return !w.closed
}

// WaitMessage 阻塞等待下一条 Cocoa 事件。
// macOS 的 run loop 天然是阻塞的，此方法作为接口占位。
func (w *CocoaWindow) WaitMessage() {
	// macOS 使用 dispatch_sync 阻塞等待主线程事件，ProcessEvents 本身会阻塞
	// 在 Cocoa 上通过 NSApplication run 处理事件，此方法保持空实现
}

// Init 初始化 macOS 平台支持
func Init() {
	window.NewWindow = NewWindow
}

func init() {
	Init()
}
