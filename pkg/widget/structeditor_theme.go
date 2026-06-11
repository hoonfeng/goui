package widget

import "github.com/hoonfeng/goui/pkg/types"

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

// 多语言重构新增的取色别名（映射到现有 StructEditorTheme 字段，无需新增主题字段）。
func seText() types.Color        { return CurrentTheme().StructEditor.TextColor }
func seHeaderText() types.Color  { return CurrentTheme().StructEditor.HeaderText }
func seFuncRowText() types.Color { return CurrentTheme().StructEditor.TextColor } // 函数行文字暂用通用文字色
func seRowBG() types.Color       { return CurrentTheme().StructEditor.CellBg }    // 普通数据行底用单元格底色

// withAlpha 返回改了 alpha 的颜色（缩略图视口框半透明用）。
func withAlpha(c types.Color, a uint8) types.Color { c.A = a; return c }
