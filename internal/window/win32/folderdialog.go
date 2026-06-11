// 系统“选择文件夹”对话框。基于 SHBrowseForFolderW + SHGetPathFromIDListW（经典样式，
// 不带 BIF_NEWDIALOGSTYLE 故无需 COM 初始化），PIDL 用 CoTaskMemFree 释放。
//
//go:build windows

package win32

import (
	"syscall"
	"unsafe"

	"github.com/hoonfeng/goui/pkg/window"
)

var (
	procSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	ole32dll                 = syscall.NewLazyDLL("ole32.dll")
	procCoTaskMemFree        = ole32dll.NewProc("CoTaskMemFree")
)

// BROWSEINFOW（64 位自然对齐，与 C 一致：ulFlags 后有 4 字节填充使 lpfn 对齐到 8）。
type browseInfoW struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

const bifReturnOnlyFSDirs = 0x00000001 // 只允许选文件系统目录

func init() { window.OpenFolderDialog = browseForFolderWin32 }

// browseForFolderWin32 弹出“选择文件夹”模态对话框，返回所选目录绝对路径；取消/出错返回空串。
func browseForFolderWin32(hwnd uintptr, title string) string {
	var titlePtr *uint16
	if title != "" {
		if t, err := syscall.UTF16PtrFromString(title); err == nil {
			titlePtr = t
		}
	}
	displayBuf := make([]uint16, 260) // 接收显示名（必需的输出缓冲）
	bi := browseInfoW{
		hwndOwner:      hwnd,
		pszDisplayName: &displayBuf[0],
		lpszTitle:      titlePtr,
		ulFlags:        bifReturnOnlyFSDirs,
	}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "" // 用户取消
	}
	defer procCoTaskMemFree.Call(pidl)
	pathBuf := make([]uint16, 1024)
	if r, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0]))); r == 0 {
		return ""
	}
	return syscall.UTF16ToString(pathBuf)
}
