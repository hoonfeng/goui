// Package event 定义了 goui 框架的事件系统。
package event

// Type 表示事件类型
type Type int

const (
	TypeUnknown Type = iota
	TypeMouseDown
	TypeMouseUp
	TypeMouseMove
	TypeMouseWheel
	TypeMouseEnter
	TypeMouseLeave
	TypeMouseDoubleClick
	TypeContextMenu
	TypeDragStart
	TypeDragMove
	TypeDragEnd
	TypeDragEnter
	TypeDragOver
	TypeDragLeave
	TypeDrop
	TypeKeyDown
	TypeKeyUp
	TypeKeyChar
	TypeWindowClose
	TypeWindowResize
	TypeWindowPaint
	TypeWindowMove
	TypeWindowDPIChange
	TypeFocusIn
	TypeFocusOut
	TypeTouchStart
	TypeTouchMove
	TypeTouchEnd
	TypeIMEComposition // IME 输入法组合字符串更新
)

// MouseButton 鼠标按键
type MouseButton int

const (
	ButtonNone   MouseButton = 0
	ButtonLeft   MouseButton = 1
	ButtonRight  MouseButton = 2
	ButtonMiddle MouseButton = 3
)

// ModifierKeys 修饰键
type ModifierKeys int

const (
	ModNone  ModifierKeys = 0
	ModShift ModifierKeys = 1 << iota
	ModCtrl
	ModAlt
	ModSuper // Windows/Command 键
)

// Event 是事件的基础接口
type Event interface {
	Type() Type
	Target() interface{}
	SetTarget(t interface{})
	StopPropagation()
	IsPropagationStopped() bool
	PreventDefault()
	IsDefaultPrevented() bool
	Timestamp() int64
}

// BaseEvent 事件基类
type BaseEvent struct {
	EventType            Type
	TargetObj            interface{}
	StopPropagationFlag  bool
	DefaultPreventedFlag bool
	TimestampVal         int64
}

// NewBaseEvent 创建基础事件
func NewBaseEvent(typ Type) *BaseEvent {
	return &BaseEvent{EventType: typ}
}

func (e *BaseEvent) Type() Type                       { return e.EventType }
func (e *BaseEvent) Target() interface{}               { return e.TargetObj }
func (e *BaseEvent) SetTarget(t interface{})           { e.TargetObj = t }
func (e *BaseEvent) StopPropagation()                  { e.StopPropagationFlag = true }
func (e *BaseEvent) IsPropagationStopped() bool         { return e.StopPropagationFlag }
func (e *BaseEvent) PreventDefault()                    { e.DefaultPreventedFlag = true }
func (e *BaseEvent) IsDefaultPrevented() bool           { return e.DefaultPreventedFlag }
func (e *BaseEvent) Timestamp() int64                   { return e.TimestampVal }

// MouseEvent 鼠标事件
type MouseEvent struct {
	BaseEvent
	X, Y   float64
	Button MouseButton
	Mods   ModifierKeys
	Clicks int // 点击次数（1=单击，2=双击）
	DeltaX float64
	DeltaY float64
}

// NewMouseEvent 创建鼠标事件
func NewMouseEvent(typ Type, x, y float64, button MouseButton, mods ModifierKeys) *MouseEvent {
	return &MouseEvent{
		BaseEvent: BaseEvent{EventType: typ},
		X:         x, Y: y,
		Button:    button,
		Mods:      mods,
		Clicks:    1,
	}
}

// KeyEvent 键盘事件
type KeyEvent struct {
	BaseEvent
	KeyCode uint32    // 虚拟键码
	Key     string    // 字符表示
	Char    rune      // 输入的字符
	Mods    ModifierKeys
	Repeat  bool
}

// NewKeyEvent 创建键盘事件
func NewKeyEvent(typ Type, keyCode uint32, key string, mods ModifierKeys) *KeyEvent {
	return &KeyEvent{
		BaseEvent: BaseEvent{EventType: typ},
		KeyCode:   keyCode,
		Key:       key,
		Mods:      mods,
	}
}

// Key 是 KeyEvent.Key 的命名类型（string 别名），配套下列按键名常量，
// 便于 switch ev.Key { case event.KeyEscape: } 这种写法（值与 win32 getKeyName 一致）。
type Key = string

const (
	KeyBackspace Key = "Backspace"
	KeyTab       Key = "Tab"
	KeyEnter     Key = "Enter"
	KeyEscape    Key = "Escape"
	KeySpace     Key = "Space"
	KeyLeft      Key = "ArrowLeft"
	KeyUp        Key = "ArrowUp"
	KeyRight     Key = "ArrowRight"
	KeyDown      Key = "ArrowDown"
	KeyHome      Key = "Home"
	KeyEnd       Key = "End"
	KeyPageUp    Key = "PageUp"
	KeyPageDown  Key = "PageDown"
	KeyDelete    Key = "Delete"
	KeyInsert    Key = "Insert"
	// KeyShiftTab 占位：win32 不产生此名（Shift+Tab 实为 Tab + ModShift），
	// 上层若要区分应判 ev.Key==KeyTab && ev.Mods&ModShift != 0。此常量仅供编译期引用。
	KeyShiftTab Key = "ShiftTab"
)

// ResizeEvent 窗口大小变化事件
type ResizeEvent struct {
	BaseEvent
	Width  int
	Height int
}

// DPIChangeEvent 屏幕 DPI 变化事件
type DPIChangeEvent struct {
	BaseEvent
	OldDPI float64
	NewDPI float64
}

// NewDPIChangeEvent 创建 DPI 变化事件
func NewDPIChangeEvent(oldDPI, newDPI float64) *DPIChangeEvent {
	return &DPIChangeEvent{
		BaseEvent: BaseEvent{EventType: TypeWindowDPIChange},
		OldDPI:    oldDPI,
		NewDPI:    newDPI,
	}
}

// NewResizeEvent 创建窗口大小变化事件
func NewResizeEvent(width, height int) *ResizeEvent {
	return &ResizeEvent{
		BaseEvent: BaseEvent{EventType: TypeWindowResize},
		Width:     width,
		Height:    height,
	}
}

// IMECompositionEvent IME 组合字符串事件
type IMECompositionEvent struct {
	BaseEvent
	Composition string // 当前组合字符串
	CursorPos   int    // 组合字符串中的光标位置
}

// NewIMECompositionEvent 创建 IME 组合字符串事件
func NewIMECompositionEvent(composition string, cursorPos int) *IMECompositionEvent {
	return &IMECompositionEvent{
		BaseEvent:   BaseEvent{EventType: TypeIMEComposition},
		Composition: composition,
		CursorPos:   cursorPos,
	}
}

// DragEvent 拖拽事件
type DragEvent struct {
	BaseEvent
	X, Y    float64
	DeltaX  float64
	DeltaY  float64
	Button  MouseButton
	Mods    ModifierKeys
	Data    interface{} // 拖拽携带的自定义数据
}

// NewDragEvent 创建拖拽事件
func NewDragEvent(typ Type, x, y float64, button MouseButton, mods ModifierKeys) *DragEvent {
	return &DragEvent{
		BaseEvent: BaseEvent{EventType: typ},
		X:         x, Y: y,
		Button:    button,
		Mods:      mods,
	}
}

// EventHandler 事件处理函数类型
type EventHandler func(event Event)

// Dispatcher 事件分发器
type Dispatcher struct {
	listeners map[Type][]EventHandler
}

// NewDispatcher 创建事件分发器
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		listeners: make(map[Type][]EventHandler),
	}
}

// AddEventListener 注册事件监听器
func (d *Dispatcher) AddEventListener(typ Type, handler EventHandler) {
	if d.listeners == nil {
		d.listeners = make(map[Type][]EventHandler)
	}
	d.listeners[typ] = append(d.listeners[typ], handler)
}

// Dispatch 分发事件到所有注册的监听器
func (d *Dispatcher) Dispatch(event Event) {
	if d.listeners == nil {
		return
	}
	if handlers, ok := d.listeners[event.Type()]; ok {
		for _, handler := range handlers {
			handler(event)
		}
	}
}
