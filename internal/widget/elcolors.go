package widget

import "github.com/user/goui/internal/types"

// ─── el 调色板（Element Plus 设计 token 的单一真相源）──────────────
// 组件 Paint 时统一调这些函数取色，而非裸写 ColorFromRGB(...)。
// 全部从全局主题 globalTheme 读 → SetTheme() 换肤时所有组件一处生效。
// （同包直接读 globalTheme，避免 CurrentTheme() 的整 struct 值拷贝。）

func elPrimary() types.Color       { return activeTheme().PrimaryColor }     // #409EFF
func elSuccess() types.Color       { return activeTheme().SuccessColor }     // #67C23A
func elWarning() types.Color       { return activeTheme().WarningColor }     // #E6A23C
func elDanger() types.Color        { return activeTheme().ErrorColor }       // #F56C6C
func elInfo() types.Color          { return activeTheme().InfoColor }        // #909399
func elTextPrimary() types.Color   { return activeTheme().TextColor }        // #303133
func elTextRegular() types.Color   { return activeTheme().TextRegular }      // #606266
func elTextSecondary() types.Color { return activeTheme().SecondaryText }    // #909399
func elPlaceholder() types.Color   { return activeTheme().PlaceholderColor } // #a8abb2
func elBorder() types.Color        { return activeTheme().BorderColor }      // #dcdfe6
func elBorderLight() types.Color   { return activeTheme().BorderLight }      // #e4e7ed
func elBorderLighter() types.Color { return activeTheme().BorderLighter }    // #ebeef5
func elFill() types.Color          { return activeTheme().FillColor }        // #f5f7fa
func elSurface() types.Color       { return activeTheme().SurfaceColor }     // #ffffff 卡片/输入框/弹层底
func elBg() types.Color            { return activeTheme().BGColor }          // #f5f7fa 页面背景
func elWhite() types.Color         { return activeTheme().White }            // 纯白（深色主题仍白，如对勾）
func elDivider() types.Color       { return activeTheme().DividerColor }     // #e4e7ed 分割线
func elDisabled() types.Color      { return activeTheme().DisabledColor }    // #a8abb2 禁用

// SetPrimaryColor 一处切换主色：联动更新所有主色相关的主题字段
// （通用 PrimaryColor + 各组件主题里的主色），实现"换肤一处生效"。
// 自绘组件经 elPrimary() 跟随；用组件子主题的（Button/Checkbox/Switch…）这里一并刷新。
func SetPrimaryColor(c types.Color) {
	t := globalTheme
	applyPrimaryColor(&t, c)
	globalTheme = t
}

// applyPrimaryColor 把主色写入 Theme 的所有主色相关字段（通用 PrimaryColor + 各组件子主题）。
// 全局 SetPrimaryColor 与局部 ConfigProvider 共用，确保换主色时自绘 + 子主题组件一致变色。
func applyPrimaryColor(t *Theme, c types.Color) {
	t.PrimaryColor = c
	t.Button.PrimaryColor = c
	t.Button.DefaultColor = c
	t.Checkbox.ActiveColor = c
	t.Checkbox.HoverBorderColor = c
	t.Switch.ActiveColor = c
	t.Slider.ActiveColor = c
	t.Slider.ThumbColor = c
	t.Radio.ActiveColor = c
	t.Radio.HoverBorderColor = c
	t.ProgressBar.Color = c
	t.Tag.PrimaryColor = c
	t.Input.FocusBorderColor = c
}
