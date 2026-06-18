// 系统"选择文件夹"模态对话框。
// 使用 IFileOpenDialog 现代对话框（Vista+ 样式），设置 FOS_PICKFOLDERS 标志
// 使其看起来和标准"打开文件"对话框一样，但只允许选择文件夹。
// 替代旧的 SHBrowseForFolderW（经典树形样式）。
//
//go:build windows

package win32

import (
	"log"
	"syscall"
	"unsafe"

	"github.com/hoonfeng/goui/pkg/window"
)

// ─── COM GUID 常量 ──────────────────────────────────────────

// CLSID_FileOpenDialog
var clsidFileOpenDialog = guid{
	0xDC1C5A9C, 0xE88A, 0x4DDE,
	[8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7},
}

// IID_IFileOpenDialog
var iidIFileOpenDialog = guid{
	0xD57C7288, 0xD4AD, 0x4768,
	[8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60},
}

// IID_IShellItem
var iidIShellItem = guid{
	0x43826D1E, 0xE718, 0x42EE,
	[8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE},
}

// ─── COM 基础类型 ───────────────────────────────────────────

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// ─── Win32 DLL 与函数 ───────────────────────────────────────

var (
	ole32                 = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx    = ole32.NewProc("CoInitializeEx")
	procCoCreateInstance  = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree     = ole32.NewProc("CoTaskMemFree")
	comInitialized        bool
)

const (
	coinitApartmentThreaded = 0x02 // COINIT_APARTMENTTHREADED
	clsctxAll               = 0x17 // CLSCTX_ALL
	sOK                     = 0    // S_OK
)

// ─── IFileDialog 选项 ───────────────────────────────────────

const (
	fosPickFolders     = 0x0020 // FOS_PICKFOLDERS：文件夹选择模式
	fosForceFilesystem = 0x0040 // FOS_FORCEFILESYSTEM：只允许文件系统项
	fosNoChangeDir     = 0x0008 // FOS_NOCHANGEDIR：不改变当前目录
)

// SIGDN (ShellItem GetDisplayName 选项)
const sigdnFilesysPath = 0x80058000 // SIGDN_FILESYSPATH

// ─── IFileDialog vtable 偏移（IFileDialog 继承 IUnknown）───
//
// IUnknown:
//   0: QueryInterface(this, riid, ppv)
//   1: AddRef(this)
//   2: Release(this)
//
// IFileDialog:
//   3: Show(this, hwndOwner)
//   4: SetFileTypes(...)
//   5: SetFileTypeIndex(...)
//   6: GetFileTypeIndex(...)
//   7: Advise(...)
//   8: Unadvise(...)
//   9: SetOptions(this, fos)
//  10: GetOptions(...)
//  11: SetDefaultFolder(...)
//  12: SetFolder(...)
//  13: GetFolder(...)
//  14: GetCurrentSelection(...)
//  15: SetFileName(...)
//  16: GetFileName(...)
//  17: SetTitle(this, pszTitle)
//  18: SetOkButtonLabel(...)
//  19: SetFileNameLabel(...)
//  20: GetResult(this, ppsi) → IShellItem
//
// IFileOpenDialog:
//  21: GetResults(...)
//  22: GetSelectedItems(...)

// ─── IShellItem vtable 偏移 ─────────────────────────────────
//
//   0: QueryInterface
//   1: AddRef
//   2: Release
//   3: BindToHandler
//   4: GetParent
//   5: GetDisplayName(this, sigdnName, ppszName) → LPWSTR
//   6: GetAttributes
//   7: Compare

// ─── 辅助：调用 COM 接口虚方法 ────────────────────────────

// comVtblCall 通过 vtable 调用 COM 接口方法。
// vtbl: 接口虚函数表指针; index: 方法索引（从0开始）; args: 参数列表（不含 this）
func comVtblCall(this uintptr, index int, args ...uintptr) uintptr {
	// this 是 COM 接口指针，其前 8 字节是 vtable 指针
	vtbl := *(*uintptr)(unsafe.Pointer(this))
	// 从 vtable 中取出方法指针
	method := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(index)*8))
	// 将参数拼接：this 指针（传入接口指针，非 vtable 指针）
	callArgs := make([]uintptr, 0, 1+len(args))
	callArgs = append(callArgs, this) // this = 接口指针
	callArgs = append(callArgs, args...)
	ret, _, _ := syscall.SyscallN(method, callArgs...)
	return ret
}

// init 注入文件夹选择对话框实现
func init() { window.OpenFolderDialog = browseForFolderWin32 }

// browseForFolderWin32 弹出"选择文件夹"模态对话框（Vista+ IFileOpenDialog 风格），
// 返回所选目录绝对路径；取消或出错返回空串。
func browseForFolderWin32(hwnd uintptr, title string) string {
	// 1. 确保 COM 已初始化（STA 模式）
	if !comInitialized {
		ret, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
		if ret == sOK || ret == 1 { // S_OK 或 S_FALSE（已初始化）
			comInitialized = true
		} else {
			log.Printf("goui/win32: CoInitializeEx failed: 0x%X", ret)
			return ""
		}
	}

	// 2. 创建 IFileOpenDialog 实例
	var pfd uintptr // IFileOpenDialog 指针
	ret, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,                                          // pUnkOuter = nil
		clsctxAll,
		uintptr(unsafe.Pointer(&iidIFileOpenDialog)),
		uintptr(unsafe.Pointer(&pfd)),
	)
	if ret != sOK || pfd == 0 {
		log.Printf("goui/win32: CoCreateInstance failed: 0x%X", ret)
		return ""
	}

	// 3. COM vtable 在 comVtblCall 内部提取，直接传 pfd

	// 4. 设置选项：文件夹选择模式
	comVtblCall(pfd, 9, uintptr(fosPickFolders|fosForceFilesystem|fosNoChangeDir)) // SetOptions

	// 5. 设置标题
	if title != "" {
		titlePtr, err := syscall.UTF16PtrFromString(title)
		if err == nil {
			comVtblCall(pfd, 17, uintptr(unsafe.Pointer(titlePtr))) // SetTitle
		}
	}

	// 6. 显示对话框（模态）
	ret = comVtblCall(pfd, 3, hwnd) // Show
	if ret != sOK {
		// 用户取消（HRESULT = 0x800704C7 = ERROR_CANCELLED）
		comVtblCall(pfd, 2) // Release
		return ""
	}

	// 7. 获取结果 ShellItem
	var psi uintptr // IShellItem 指针
	ret = comVtblCall(pfd, 20, uintptr(unsafe.Pointer(&psi))) // GetResult
	if ret != sOK || psi == 0 {
		comVtblCall(pfd, 2) // Release dialog
		return ""
	}

	// 8. ShellItem vtable 在 comVtblCall 内部提取，直接传 psi

	// 9. 获取文件系统路径
	var pathPtr uintptr // LPWSTR（由 CoTaskMemAlloc 分配）
	ret = comVtblCall(psi, 5, sigdnFilesysPath, uintptr(unsafe.Pointer(&pathPtr))) // GetDisplayName
	if ret != sOK || pathPtr == 0 {
		comVtblCall(psi, 2) // Release shell item
		comVtblCall(pfd, 2)   // Release dialog
		return ""
	}

	// 10. 转换路径字符串
	path := syscall.UTF16ToString((*[1024]uint16)(unsafe.Pointer(pathPtr))[:])

	// 11. 释放 CoTaskMemAlloc 分配的字符串
	procCoTaskMemFree.Call(pathPtr)

	// 12. 释放 ShellItem
	comVtblCall(psi, 2) // Release

	// 13. 释放对话框
	comVtblCall(pfd, 2) // Release

	return path
}
