package widget

import (
	"github.com/hoonfeng/goui/pkg/types"
)

// Theme 定义全局主题配置（类似 CSS 变量/Design Tokens）。
// 所有组件的默认样式通过 Theme 统一管理，支持在应用启动时全局设置。
// 用户可以通过 SetTheme() 注入自己的设计系统/品牌色板，
// 实现"Web 方式"的样式复用和统一切换。
type Theme struct {
	// ── 通用颜色 ──────────────────────────────────
	PrimaryColor   types.Color // 主色（按钮、链接、选中状态）
	SecondaryColor types.Color // 辅助色
	TextColor      types.Color // 主文本色
	SecondaryText  types.Color // 次要文本色
	BGColor        types.Color // 页面背景色
	SurfaceColor   types.Color // 表面色（卡片、输入框）
	BorderColor    types.Color // 边框色
	DividerColor   types.Color // 分割线色
	ErrorColor     types.Color // 错误色
	SuccessColor   types.Color // 成功色
	WarningColor   types.Color // 警告色
	InfoColor      types.Color // 信息色
	DisabledColor  types.Color // 禁用色
	// ── el 语义补充（Element Plus 设计 token，供 elXxx() 调色板读取）──
	TextRegular      types.Color // el 常规文字 #606266
	PlaceholderColor types.Color // el 占位文字 #a8abb2
	BorderLight      types.Color // el 浅边框 #e4e7ed
	BorderLighter    types.Color // el 更浅边框 #ebeef5
	FillColor        types.Color // el 填充背景 #f5f7fa
	White            types.Color // 白色
	Black            types.Color // 黑色

	// ── 组件主题 ──────────────────────────────────
	Checkbox    CheckboxTheme
	Button      ButtonTheme
	Input       InputTheme
	ProgressBar ProgressBarTheme
	Slider      SliderTheme
	Card        CardTheme
	Switch      SwitchTheme
	Radio       RadioTheme
	Tag         TagTheme
	Tooltip      TooltipTheme
	Popover      PopoverTheme
	CodeEditor   CodeEditorTheme
	StructEditor StructEditorTheme
}

// CodeEditorTheme 代码编辑器配色（语法高亮 + 编辑器界面），随主题可换肤。
type CodeEditorTheme struct {
	Background     types.Color // 编辑器背景
	GutterBg       types.Color // 行号栏背景
	GutterText     types.Color // 行号字色
	GutterActiveBg types.Color // 当前行的行号栏底（略深）
	CurrentLineBg  types.Color // 当前行高亮底
	Selection      types.Color // 选区高亮（半透明）
	MinimapBg      types.Color // 缩略图背景
	// 语法 token 配色
	Keyword types.Color // 关键字
	Type    types.Color // 类型/内置/常量
	String  types.Color // 字符串
	Comment types.Color // 注释
	Number  types.Color // 数字
	Func    types.Color // 函数名
	Text    types.Color // 普通文本/标识符
}

// StructEditorTheme 代码表格化编辑器（StructEditor）配色，随主题可换肤。
type StructEditorTheme struct {
	Background  types.Color // 画布背景
	CellBg      types.Color // 单元格/输入框底
	LineColor   types.Color // 网格线
	HeaderBg    types.Color // 表头底
	GutterBg    types.Color // 行号栏底
	GutterText  types.Color // 行号字色
	TextColor   types.Color // 单元格文字
	SelectionBg types.Color // 选中单元格底色
	HeaderText  types.Color // 表头文字
	FuncRowBg   types.Color // 函数声明行蓝底
	MinimapBg   types.Color // 缩略图背景
}

// CheckboxTheme 复选框主题
type CheckboxTheme struct {
	ActiveColor       types.Color // 选中时方框填充色
	InactiveBgColor   types.Color // 未选中时方框背景色
	BorderColor       types.Color // 未选中边框色
	HoverBorderColor  types.Color // 悬停边框色
	CheckColor        types.Color // 勾选标记颜色
	LabelColor        types.Color // 标签文字颜色
	BoxSize           float64     // 方框尺寸
	BorderRadius      float64     // 方框圆角半径
	HoverBgColor      types.Color // 悬停时方框背景色（未选中）
	CheckedHoverColor types.Color // 悬停时选中方框颜色
}

// ButtonTheme 按钮主题
type ButtonTheme struct {
	DefaultColor  types.Color // 默认按钮色
	PrimaryColor  types.Color // 主要按钮色
	SuccessColor  types.Color // 成功按钮色
	DangerColor   types.Color // 危险按钮色
	TextColor     types.Color // 按钮文字颜色
	HoverColor    types.Color // 悬停颜色
	DisabledColor types.Color // 禁用颜色
	MinWidth      float64
	MinHeight     float64
	BorderRadius  float64
}

// InputTheme 输入框主题
type InputTheme struct {
	TextColor        types.Color
	BGColor          types.Color
	BorderColor      types.Color
	FocusBorderColor types.Color
	PlaceholderColor types.Color
	CursorColor      types.Color
	CursorWidth      float64
	BorderRadius     float64
}

// ProgressBarTheme 进度条主题
type ProgressBarTheme struct {
	Color      types.Color
	BGColor    types.Color
	LabelColor types.Color
	Height     float64
	Radius     float64
}

// SliderTheme 滑块主题
type SliderTheme struct {
	ActiveColor   types.Color
	InactiveColor types.Color
	ThumbColor    types.Color
	LabelColor    types.Color
	ThumbRadius   float64
	TrackHeight   float64
}

// CardTheme 卡片主题
type CardTheme struct {
	Color       types.Color
	Radius      float64
	Elevation   int
	BorderColor *types.Color
	BorderWidth float64
	ShadowColor types.Color
}

// SwitchTheme 开关主题
type SwitchTheme struct {
	ActiveColor   types.Color
	InactiveColor types.Color
	ThumbColor    types.Color
	TrackHeight   float64
	ThumbRadius   float64
}

// RadioTheme 单选按钮主题
type RadioTheme struct {
	ActiveColor      types.Color
	InactiveColor    types.Color
	BorderColor      types.Color
	HoverBorderColor types.Color
	LabelColor       types.Color
	DotColor         types.Color
	Size             float64
}

// TagTheme 标签主题（采用 Element Plus 官方语义配色）。
// 五个 type 主色 + light/dark/plain 三种 effect 的派生色在运行时由主色混合得出。
type TagTheme struct {
	PrimaryColor types.Color // primary 主色（#409EFF）
	SuccessColor types.Color // success 主色（#67C23A）
	InfoColor    types.Color // info 主色（#909399）
	WarningColor types.Color // warning 主色（#E6A23C）
	DangerColor  types.Color // danger 主色（#F56C6C）
	BorderRadius float64     // 圆角半径（round 时改用高度的一半）
	FontSize     float64     // default 尺寸字号
}

// TooltipTheme 文字提示主题（dark/light 两种风格）。
type TooltipTheme struct {
	DarkBg      types.Color // dark 背景
	DarkText    types.Color // dark 文字
	LightBg     types.Color // light 背景
	LightText   types.Color // light 文字
	LightBorder types.Color // light 边框
	FontSize    float64
	Radius      float64
	PaddingH    float64
	PaddingV    float64
}

// PopoverTheme 弹出框主题。
type PopoverTheme struct {
	Bg         types.Color
	Border     types.Color
	TitleColor types.Color
	TextColor  types.Color
	TitleSize  float64
	FontSize   float64
	Radius     float64
	PaddingH   float64
	PaddingV   float64
}

// globalTheme 全局默认主题
var globalTheme = DefaultTheme()

// SetTheme 设置全局主题。
// 调用后，所有使用 DefaultXxx() 创建的组件将使用新主题的默认值。
// 这实现了类似 CSS 变量/Design Tokens 的统一样式管理。
func SetTheme(t Theme) {
	globalTheme = t
}

// themeScope 渲染期生效的局部主题（ConfigProvider 在 Paint 子树时设；nil=用全局 globalTheme）。
// 实现「组件/子树级差异化」：作用域内 elXxx()/CurrentTheme() 读它，作用域外不受影响。
var themeScope *Theme

// activeTheme 返回当前生效主题指针：局部作用域(ConfigProvider)优先，否则全局。
func activeTheme() *Theme {
	if themeScope != nil {
		return themeScope
	}
	return &globalTheme
}

// CurrentTheme 返回当前生效主题（局部作用域优先）。
// 组件在 DefaultXxx() 工厂函数 / Paint 中调用此方法获取样式。
func CurrentTheme() Theme {
	return *activeTheme()
}

// DefaultTheme 返回 goui 默认主题。
// 基于 Material Design 3 配色风格，提供清晰、现代的默认样式。
func DefaultTheme() Theme {
	return Theme{
		// 通用颜色（Element Plus 设计 token）
		PrimaryColor:     types.ColorFromRGB(64, 158, 255),  // el #409EFF
		SecondaryColor:   types.ColorFromRGB(103, 194, 58),  // el success #67C23A
		TextColor:        types.ColorFromRGB(48, 49, 51),    // el text-primary #303133
		SecondaryText:    types.ColorFromRGB(144, 147, 153), // el text-secondary #909399
		BGColor:          types.ColorFromRGB(245, 247, 250), // el fill #f5f7fa
		SurfaceColor:     types.ColorFromRGB(255, 255, 255), // White
		BorderColor:      types.ColorFromRGB(220, 223, 230), // el border-base #dcdfe6
		DividerColor:     types.ColorFromRGB(228, 231, 237), // el border-light #e4e7ed
		ErrorColor:       types.ColorFromRGB(245, 108, 108), // el danger #F56C6C
		SuccessColor:     types.ColorFromRGB(103, 194, 58),  // el #67C23A
		WarningColor:     types.ColorFromRGB(230, 162, 60),  // el #E6A23C
		InfoColor:        types.ColorFromRGB(144, 147, 153), // el #909399
		DisabledColor:    types.ColorFromRGB(168, 171, 178), // el placeholder #a8abb2
		TextRegular:      types.ColorFromRGB(96, 98, 102),   // el #606266
		PlaceholderColor: types.ColorFromRGB(168, 171, 178), // el #a8abb2
		BorderLight:      types.ColorFromRGB(228, 231, 237), // el #e4e7ed
		BorderLighter:    types.ColorFromRGB(235, 238, 245), // el #ebeef5
		FillColor:        types.ColorFromRGB(245, 247, 250), // el #f5f7fa
		White:            types.ColorWhite,
		Black:            types.ColorBlack,

		// Checkbox 主题
		Checkbox: CheckboxTheme{
			ActiveColor:       types.ColorFromRGB(64, 158, 255),  // el #409EFF
			InactiveBgColor:   types.ColorFromRGB(255, 255, 255),
			BorderColor:       types.ColorFromRGB(220, 223, 230), // el #dcdfe6
			HoverBorderColor:  types.ColorFromRGB(64, 158, 255),  // el 主色
			CheckColor:        types.ColorWhite,
			LabelColor:        types.ColorFromRGB(96, 98, 102),   // el #606266
			BoxSize:           18,
			BorderRadius:      3,
			HoverBgColor:      types.ColorFromRGB(240, 242, 245),
			CheckedHoverColor: types.ColorFromRGB(102, 177, 255), // el hover #66b1ff
		},

		// Button 主题
		Button: ButtonTheme{
			DefaultColor:  types.ColorFromRGB(64, 158, 255),  // el #409EFF
			PrimaryColor:  types.ColorFromRGB(64, 158, 255),
			SuccessColor:  types.ColorFromRGB(103, 194, 58),  // el #67C23A
			DangerColor:   types.ColorFromRGB(245, 108, 108), // el #F56C6C
			TextColor:     types.ColorWhite,
			HoverColor:    types.ColorFromRGB(102, 177, 255), // el hover #66b1ff
			DisabledColor: types.ColorFromRGB(168, 171, 178),
			MinWidth:      64,
			MinHeight:     32,
			BorderRadius:  4,
		},

		// Input 主题
		Input: InputTheme{
			TextColor:        types.ColorFromRGB(96, 98, 102),   // el #606266
			BGColor:          types.ColorFromRGB(255, 255, 255),
			BorderColor:      types.ColorFromRGB(220, 223, 230), // el #dcdfe6
			FocusBorderColor: types.ColorFromRGB(64, 158, 255),  // el 主色 #409EFF
			PlaceholderColor: types.ColorFromRGB(168, 171, 178), // el #a8abb2
			CursorColor:      types.ColorFromRGB(33, 33, 33),
			CursorWidth:      1.5,
			BorderRadius:     4,
		},

		// ProgressBar 主题
		ProgressBar: ProgressBarTheme{
			Color:      types.ColorFromRGB(64, 158, 255),
			BGColor:    types.ColorFromRGB(235, 238, 245),
			LabelColor: types.ColorFromRGB(96, 98, 102),
			Height:     8,
			Radius:     4,
		},

		// Slider 主题
		Slider: SliderTheme{
			ActiveColor:   types.ColorFromRGB(64, 158, 255),
			InactiveColor: types.ColorFromRGB(220, 223, 230),
			ThumbColor:    types.ColorFromRGB(64, 158, 255),
			LabelColor:    types.ColorFromRGB(96, 98, 102),
			ThumbRadius:   8,
			TrackHeight:   4,
		},

		// Card 主题
		Card: CardTheme{
			Color:       types.ColorWhite,
			Radius:      8,
			Elevation:   1,
			BorderColor: nil,
			BorderWidth: 0,
			ShadowColor: types.ColorFromRGBA(0, 0, 0, 30),
		},

		// Switch 主题
		Switch: SwitchTheme{
			ActiveColor:   types.ColorFromRGB(64, 158, 255),  // el primary #409EFF
			InactiveColor: types.ColorFromRGB(220, 223, 230), // el #dcdfe6
			ThumbColor:    types.ColorWhite,
			TrackHeight:   24,
			ThumbRadius:   10,
		},

		// Radio 主题
		Radio: RadioTheme{
			ActiveColor:      types.ColorFromRGB(64, 158, 255),
			InactiveColor:    types.ColorFromRGB(255, 255, 255),
			BorderColor:      types.ColorFromRGB(220, 223, 230),
			HoverBorderColor: types.ColorFromRGB(64, 158, 255),
			LabelColor:       types.ColorFromRGB(96, 98, 102),
			DotColor:         types.ColorWhite,
			Size:             18,
		},

		// Tag 主题（Element Plus 官方配色）
		Tag: TagTheme{
			PrimaryColor: types.ColorFromRGB(64, 158, 255),
			SuccessColor: types.ColorFromRGB(103, 194, 58),
			InfoColor:    types.ColorFromRGB(144, 147, 153),
			WarningColor: types.ColorFromRGB(230, 162, 60),
			DangerColor:  types.ColorFromRGB(245, 108, 108),
			BorderRadius: 4,
			FontSize:     12,
		},

		// Tooltip 主题
		Tooltip: TooltipTheme{
			DarkBg:      types.ColorFromRGB(48, 49, 51),
			DarkText:    types.ColorWhite,
			LightBg:     types.ColorWhite,
			LightText:   types.ColorFromRGB(48, 49, 51),
			LightBorder: types.ColorFromRGB(228, 231, 237),
			FontSize:    12,
			Radius:      4,
			PaddingH:    11,
			PaddingV:    8,
		},

		// Popover 主题
		Popover: PopoverTheme{
			Bg:         types.ColorWhite,
			Border:     types.ColorFromRGB(235, 238, 245),
			TitleColor: types.ColorFromRGB(48, 49, 51),
			TextColor:  types.ColorFromRGB(96, 98, 102),
			TitleSize:  16,
			FontSize:   14,
			Radius:     4,
			PaddingH:   12,
			PaddingV:   12,
		},

		// CodeEditor 主题（GitHub Light 语法配色）
		CodeEditor: CodeEditorTheme{
			Background:     types.ColorFromRGB(255, 255, 255),
			GutterBg:       types.ColorFromRGB(0xF6, 0xF8, 0xFA),
			GutterText:     types.ColorFromRGB(0x9D, 0xA5, 0xB4),
			GutterActiveBg: types.ColorFromRGB(0xE8, 0xEE, 0xF8),
			CurrentLineBg:  types.ColorFromRGB(0xF0, 0xF4, 0xFC),
			Selection:      types.ColorFromRGBA(0x40, 0x9E, 0xFF, 70),
			MinimapBg:      types.ColorFromRGB(0xFB, 0xFB, 0xFC),
			Keyword:       types.ColorFromRGB(0xD7, 0x3A, 0x49), // 红
			Type:          types.ColorFromRGB(0x6F, 0x42, 0xC1), // 紫
			String:        types.ColorFromRGB(0x03, 0x2F, 0x62), // 深蓝
			Comment:       types.ColorFromRGB(0x6A, 0x73, 0x7D), // 灰
			Number:        types.ColorFromRGB(0x00, 0x5C, 0xC5), // 蓝
			Func:          types.ColorFromRGB(0x6F, 0x42, 0xC1), // 紫
			Text:          types.ColorFromRGB(0x24, 0x29, 0x2E), // 深灰
		},

		// StructEditor 主题（代码表格化）
		StructEditor: StructEditorTheme{
			Background:  types.ColorFromRGB(255, 255, 255),
			CellBg:      types.ColorFromRGB(255, 255, 255),
			LineColor:   types.ColorFromRGB(0xE4, 0xE7, 0xED),
			HeaderBg:    types.ColorFromRGB(0xF5, 0xF7, 0xFA),
			GutterBg:    types.ColorFromRGB(0xF6, 0xF8, 0xFA),
			GutterText:  types.ColorFromRGB(0x9D, 0xA5, 0xB4),
			TextColor:   types.ColorFromRGB(0x30, 0x31, 0x33),
			SelectionBg: types.ColorFromRGB(0xE8, 0xF0, 0xFE),
			HeaderText:  types.ColorFromRGB(0x60, 0x62, 0x66),
			FuncRowBg:   types.ColorFromRGB(0xEC, 0xF2, 0xFC),
			MinimapBg:   types.ColorFromRGB(0xFB, 0xFB, 0xFC),
		},
	}
}

// DarkTheme 深色主题（VS Code Dark+ 风格）。SetTheme(DarkTheme()) 即可整体切深色。
// 通用色 + 代码/表格编辑器配色完整深色；仍有少量组件硬编码色未深色化（按需补）。
func DarkTheme() Theme {
	t := DefaultTheme()
	// 通用色
	t.PrimaryColor = types.ColorFromRGB(0x4E, 0xA1, 0xF7)
	t.TextColor = types.ColorFromRGB(0xD4, 0xD4, 0xD4)
	t.SecondaryText = types.ColorFromRGB(0x85, 0x85, 0x85)
	t.BGColor = types.ColorFromRGB(0x1E, 0x1E, 0x1E)
	t.SurfaceColor = types.ColorFromRGB(0x25, 0x25, 0x26)
	t.BorderColor = types.ColorFromRGB(0x3C, 0x3C, 0x3C)
	t.DividerColor = types.ColorFromRGB(0x33, 0x33, 0x34)
	t.TextRegular = types.ColorFromRGB(0xCC, 0xCC, 0xCC)
	t.PlaceholderColor = types.ColorFromRGB(0x6E, 0x6E, 0x6E)
	t.BorderLight = types.ColorFromRGB(0x3C, 0x3C, 0x3C)
	t.BorderLighter = types.ColorFromRGB(0x33, 0x33, 0x34)
	t.FillColor = types.ColorFromRGB(0x2A, 0x2A, 0x2B)
	t.Input.TextColor = types.ColorFromRGB(0xD4, 0xD4, 0xD4)
	t.Input.BGColor = types.ColorFromRGB(0x2A, 0x2A, 0x2B)
	t.Input.BorderColor = types.ColorFromRGB(0x3C, 0x3C, 0x3C)

	// CodeEditor 深色（VS Code Dark+ 语法配色）
	t.CodeEditor = CodeEditorTheme{
		Background:     types.ColorFromRGB(0x1E, 0x1E, 0x1E),
		GutterBg:       types.ColorFromRGB(0x1E, 0x1E, 0x1E),
		GutterText:     types.ColorFromRGB(0x85, 0x85, 0x85),
		GutterActiveBg: types.ColorFromRGB(0x2A, 0x2A, 0x2B),
		CurrentLineBg:  types.ColorFromRGB(0x2A, 0x2A, 0x2B),
		Selection:      types.ColorFromRGBA(0x26, 0x4F, 0x78, 180),
		MinimapBg:      types.ColorFromRGB(0x25, 0x25, 0x26),
		Keyword:        types.ColorFromRGB(0x56, 0x9C, 0xD6), // 蓝
		Type:           types.ColorFromRGB(0x4E, 0xC9, 0xB0), // 青
		String:         types.ColorFromRGB(0xCE, 0x91, 0x78), // 橙
		Comment:        types.ColorFromRGB(0x6A, 0x99, 0x55), // 绿
		Number:         types.ColorFromRGB(0xB5, 0xCE, 0xA8), // 浅绿
		Func:           types.ColorFromRGB(0xDC, 0xDC, 0xAA), // 黄
		Text:           types.ColorFromRGB(0xD4, 0xD4, 0xD4),
	}

	// StructEditor 深色
	t.StructEditor = StructEditorTheme{
		Background:  types.ColorFromRGB(0x1E, 0x1E, 0x1E),
		CellBg:      types.ColorFromRGB(0x2A, 0x2A, 0x2B),
		LineColor:   types.ColorFromRGB(0x3C, 0x3C, 0x3C),
		HeaderBg:    types.ColorFromRGB(0x2D, 0x2D, 0x30),
		GutterBg:    types.ColorFromRGB(0x25, 0x25, 0x26),
		GutterText:  types.ColorFromRGB(0x85, 0x85, 0x85),
		TextColor:   types.ColorFromRGB(0xD4, 0xD4, 0xD4),
		SelectionBg: types.ColorFromRGB(0x09, 0x3A, 0x5C),
		HeaderText:  types.ColorFromRGB(0xC0, 0xC0, 0xC0),
		FuncRowBg:   types.ColorFromRGB(0x2D, 0x3A, 0x4A),
		MinimapBg:   types.ColorFromRGB(0x25, 0x25, 0x26),
	}

	// 组件子主题深色化（背景类→深、文字类→浅、边框→深灰；主色/勾点白保留）
	surf := types.ColorFromRGB(0x25, 0x25, 0x26)
	surf2 := types.ColorFromRGB(0x2A, 0x2A, 0x2B)
	bd := types.ColorFromRGB(0x3C, 0x3C, 0x3C)
	tx := types.ColorFromRGB(0xD4, 0xD4, 0xD4)
	txr := types.ColorFromRGB(0xCC, 0xCC, 0xCC)
	t.Checkbox.InactiveBgColor = surf2
	t.Checkbox.BorderColor = bd
	t.Checkbox.LabelColor = tx
	t.Checkbox.HoverBgColor = surf2
	t.Button.TextColor = types.ColorWhite
	t.Input.PlaceholderColor = types.ColorFromRGB(0x6E, 0x6E, 0x6E)
	t.Input.CursorColor = tx
	t.ProgressBar.BGColor = surf2
	t.ProgressBar.LabelColor = tx
	t.Slider.InactiveColor = bd
	t.Slider.LabelColor = tx
	cardBorder := bd
	t.Card.Color = surf
	t.Card.BorderColor = &cardBorder
	t.Card.ShadowColor = types.ColorFromRGBA(0, 0, 0, 90)
	t.Switch.InactiveColor = types.ColorFromRGB(0x48, 0x48, 0x4A)
	t.Radio.InactiveColor = surf2
	t.Radio.BorderColor = bd
	t.Radio.LabelColor = tx
	t.Tooltip.LightBg = surf2
	t.Tooltip.LightText = tx
	t.Tooltip.LightBorder = bd
	t.Popover.Bg = surf
	t.Popover.Border = bd
	t.Popover.TitleColor = tx
	t.Popover.TextColor = txr
	return t
}
