package widget

// ClickTarget 标记「可点击控件」元素（Button / Clickable / Dropdown 等）。
//
// 用途：无边框窗口的自绘标题栏里，需区分某点是「可点控件」（交 goui 处理点击 → HTCLIENT）
// 还是「空白拖动区」（交系统拖窗/双击最大化 → HTCAPTION）。app 命中测试该点后沿父链查找，
// 命中任一 ClickTarget 即视为可点。这样菜单/按钮可放进标题栏并正常响应，空白处仍能拖动。
type ClickTarget interface{ IsClickTarget() bool }

func (e *ButtonElement) IsClickTarget() bool    { return true }
func (e *ClickableElement) IsClickTarget() bool { return true }
func (e *DropdownElement) IsClickTarget() bool  { return true }
