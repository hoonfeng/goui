package widget

// 剪贴板访问钩子。
//
// widget 层不直接依赖平台实现，因此通过这两个函数变量解耦：
// 由 app 在初始化时把平台层（如 win32）的剪贴板实现桥接进来。
// 未设置时（例如平台未实现），输入框的复制/剪切/粘贴快捷键安全地无操作。
var (
	// ClipboardRead 返回系统剪贴板中的文本。
	ClipboardRead func() string
	// ClipboardWrite 将文本写入系统剪贴板。
	ClipboardWrite func(text string)
)
