package widget

import "github.com/user/goui/internal/types"

// StructEditor 的配色全部取自全局 Theme.StructEditor（SetTheme 换肤即生效），不再硬编码。
// 每帧渲染时实时读取，故换主题后下次重绘自动更新。

func seLineColor() types.Color { return CurrentTheme().StructEditor.LineColor }
func seHeaderBG() types.Color  { return CurrentTheme().StructEditor.HeaderBg }
func seGutterBG() types.Color  { return CurrentTheme().StructEditor.GutterBg }
func seGutterNum() types.Color { return CurrentTheme().StructEditor.GutterText }
func seTextColor() types.Color { return CurrentTheme().StructEditor.TextColor }
func seSelBG() types.Color     { return CurrentTheme().StructEditor.SelectionBg }
func seHeadText() types.Color   { return CurrentTheme().StructEditor.HeaderText }
func seFuncRowBG() types.Color  { return CurrentTheme().StructEditor.FuncRowBg }
func seBackground() types.Color { return CurrentTheme().StructEditor.Background }
func seCellBg() types.Color     { return CurrentTheme().StructEditor.CellBg }
func seMinimapBG() types.Color  { return CurrentTheme().StructEditor.MinimapBg }

// withAlpha 返回改了 alpha 的颜色（缩略图视口框半透明用）。
func withAlpha(c types.Color, a uint8) types.Color { c.A = a; return c }
