//go:build windows

// 系统托盘（通知区图标）+ 托盘右键菜单。基于 Shell_NotifyIconW + TrackPopupMenu。
package win32

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/user/goui/internal/window"
)

var (
	shell32 = syscall.NewLazyDLL("shell32.dll")

	procShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procLoadImageW          = user32.NewProc("LoadImageW")
)

const (
	// Shell_NotifyIcon dwMessage
	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	// NOTIFYICONDATA uFlags
	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_INFO    = 0x00000010 // 气泡通知

	// 气泡通知图标级别（dwInfoFlags）
	NIIF_INFO    = 0x00000001
	NIIF_WARNING = 0x00000002
	NIIF_ERROR   = 0x00000003

	// WM_TRAYICON 托盘回调消息（WM_USER+1）。lParam 携带鼠标消息，wParam=uID。
	WM_TRAYICON = 0x0400 + 1

	// TrackPopupMenu uFlags
	TPM_LEFTALIGN   = 0x0000
	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100

	// AppendMenu uFlags
	MF_STRING    = 0x00000000
	MF_GRAYED    = 0x00000001
	MF_SEPARATOR = 0x00000800

	// LoadImage
	IMAGE_ICON      = 1
	LR_LOADFROMFILE = 0x00000010
	LR_DEFAULTSIZE  = 0x00000040
	IDI_APPLICATION = 32512
)

// NOTIFYICONDATAW 与 Win32 tagNOTIFYICONDATAW 内存布局一致（V2+，szTip[128]）。
// 字段自然对齐，cbSize 用 unsafe.Sizeof 取真实大小，现代 Windows 据此识别版本。
type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersionTimeout  uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
}

// trayEntry 一个托盘图标的运行时状态。
type trayEntry struct {
	id           int
	hwnd         uintptr
	uID          uint32
	hIcon        uintptr
	onLeftClick  func()
	menuItems    []window.TrayMenuItem
	onMenuSelect func(id int)
}

var (
	trayMu      sync.Mutex
	trayByID    = map[int]*trayEntry{}
	trayByUID   = map[uint32]*trayEntry{} // uID → entry，供 windowProc 反查
	nextTrayID  int
	nextTrayUID uint32 = 1000
)

// addTrayIconWin32 添加托盘图标，返回托盘 ID（>0）。
func addTrayIconWin32(hwnd uintptr, tooltip, iconPath string, onLeftClick func()) int {
	trayMu.Lock()
	nextTrayID++
	nextTrayUID++
	id, uID := nextTrayID, nextTrayUID
	e := &trayEntry{id: id, hwnd: hwnd, uID: uID, hIcon: loadTrayIcon(iconPath), onLeftClick: onLeftClick}
	trayByID[id] = e
	trayByUID[uID] = e
	trayMu.Unlock()

	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwnd
	nid.UID = uID
	nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	nid.UCallbackMessage = WM_TRAYICON
	nid.HIcon = e.hIcon
	copyUTF16(nid.SzTip[:], tooltip)

	ret, _, _ := procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		trayMu.Lock()
		delete(trayByID, id)
		delete(trayByUID, uID)
		trayMu.Unlock()
		return 0
	}
	return id
}

// setTrayMenuWin32 设置/更新托盘右键菜单与点击回调。
func setTrayMenuWin32(trayID int, items []window.TrayMenuItem, onSelect func(id int)) {
	trayMu.Lock()
	defer trayMu.Unlock()
	if e, ok := trayByID[trayID]; ok {
		e.menuItems = items
		e.onMenuSelect = onSelect
	}
}

// showTrayBalloonWin32 弹出托盘气泡通知（标题 + 正文 + 图标级别：0=信息/1=警告/2=错误）。
func showTrayBalloonWin32(trayID int, title, text string, level int) {
	trayMu.Lock()
	e, ok := trayByID[trayID]
	trayMu.Unlock()
	if !ok {
		return
	}
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = e.hwnd
	nid.UID = e.uID
	nid.UFlags = NIF_INFO
	copyUTF16(nid.SzInfo[:], text)
	copyUTF16(nid.SzInfoTitle[:], title)
	switch level {
	case 1:
		nid.DwInfoFlags = NIIF_WARNING
	case 2:
		nid.DwInfoFlags = NIIF_ERROR
	default:
		nid.DwInfoFlags = NIIF_INFO
	}
	procShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}

// setTrayTooltipWin32 更新托盘悬停提示。
func setTrayTooltipWin32(trayID int, tooltip string) {
	trayMu.Lock()
	e, ok := trayByID[trayID]
	trayMu.Unlock()
	if !ok {
		return
	}
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = e.hwnd
	nid.UID = e.uID
	nid.UFlags = NIF_TIP
	copyUTF16(nid.SzTip[:], tooltip)
	procShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}

// removeTrayIconWin32 移除托盘图标。
func removeTrayIconWin32(trayID int) {
	trayMu.Lock()
	e, ok := trayByID[trayID]
	if ok {
		delete(trayByID, trayID)
		delete(trayByUID, e.uID)
	}
	trayMu.Unlock()
	if !ok {
		return
	}
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = e.hwnd
	nid.UID = e.uID
	procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

// handleTrayMessage 处理 WM_TRAYICON：左键回调 / 右键弹菜单。由 windowProc 调用。
func handleTrayMessage(uID uint32, mouseMsg uint32) {
	trayMu.Lock()
	e, ok := trayByUID[uID]
	trayMu.Unlock()
	if !ok {
		return
	}
	switch mouseMsg {
	case WM_LBUTTONUP:
		if e.onLeftClick != nil {
			e.onLeftClick()
		}
	case WM_RBUTTONUP:
		showTrayMenu(e)
	}
}

// showTrayMenu 在光标处弹出托盘右键菜单（TPM_RETURNCMD 同步返回选中 ID）。
func showTrayMenu(e *trayEntry) {
	if len(e.menuItems) == 0 {
		return
	}
	hmenu, _, _ := procCreatePopupMenu.Call()
	if hmenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hmenu)

	for _, it := range e.menuItems {
		if it.Separator {
			procAppendMenuW.Call(hmenu, MF_SEPARATOR, 0, 0)
			continue
		}
		flags := uintptr(MF_STRING)
		if it.Disabled {
			flags |= MF_GRAYED
		}
		label, _ := syscall.UTF16PtrFromString(it.Label)
		procAppendMenuW.Call(hmenu, flags, uintptr(it.ID), uintptr(unsafe.Pointer(label)))
	}

	// 必须置前台，否则菜单在点击别处时不会消失（Win32 已知行为）。
	procSetForegroundWindow.Call(e.hwnd)
	var pt struct{ X, Y int32 }
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	cmd, _, _ := procTrackPopupMenu.Call(
		hmenu, TPM_RIGHTBUTTON|TPM_RETURNCMD,
		uintptr(pt.X), uintptr(pt.Y), 0, e.hwnd, 0)
	if cmd != 0 && e.onMenuSelect != nil {
		e.onMenuSelect(int(cmd))
	}
}

// loadTrayIcon 从 .ico 文件加载图标；路径空或失败时回退系统默认应用图标。
func loadTrayIcon(path string) uintptr {
	if path != "" {
		if p, err := syscall.UTF16PtrFromString(path); err == nil {
			h, _, _ := procLoadImageW.Call(0, uintptr(unsafe.Pointer(p)),
				IMAGE_ICON, 0, 0, LR_LOADFROMFILE|LR_DEFAULTSIZE)
			if h != 0 {
				return h
			}
		}
	}
	h, _, _ := procLoadIconW.Call(0, IDI_APPLICATION)
	return h
}

// copyUTF16 把字符串写入定长 uint16 缓冲（截断并保证结尾 0）。
func copyUTF16(dst []uint16, s string) {
	src, err := syscall.UTF16FromString(s)
	if err != nil {
		return
	}
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst[:n], src[:n])
	if n > 0 {
		dst[n-1] = 0
	}
}

func init() {
	window.AddTrayIcon = addTrayIconWin32
	window.SetTrayMenu = setTrayMenuWin32
	window.SetTrayTooltip = setTrayTooltipWin32
	window.RemoveTrayIcon = removeTrayIconWin32
	window.ShowTrayBalloon = showTrayBalloonWin32
}
