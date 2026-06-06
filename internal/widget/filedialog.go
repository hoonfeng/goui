package widget

// OpenFileDialog 由 app 层桥接到平台实现（见 app.go），弹出系统“打开文件”对话框。
// title: 标题; filter: 过滤器（"图片|*.png;*.jpg" 竖线分隔，多组用换行分隔）。
// 返回选择的绝对路径；取消或平台未实现返回空串。
// 组件层（如富文本插入图片）通过此变量调用，避免直接依赖 window 包。
var OpenFileDialog func(title, filter string) string
