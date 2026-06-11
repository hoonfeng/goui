package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── ConfigProvider 局部主题作用域（el-config-provider）──────────────
// 包裹 Child：子树内组件用本 Provider 解析出的主题，子树外不受影响。
// 实现「组件/子树级配色差异化」——这块深色、那块换主色，互不干扰，可嵌套。
//
// 原理：Element 在 Layout/Paint 子树前把全局 themeScope 指向本主题、之后还原；
// 作用域内 elXxx()/CurrentTheme() 经 activeTheme() 读到它。
//
//	widget.NewConfigProvider(editor).WithTheme(widget.DarkTheme())   // 仅这个编辑器深色
//	widget.NewConfigProvider(panel).WithPrimaryColor(green)          // 仅这块绿主色
//	widget.NewConfigProvider(x).WithApply(func(t *widget.Theme){ ... }) // 任意自定义
type ConfigProvider struct {
	SingleChildWidget
	PrimaryColor *types.Color // 覆盖主色（联动 Button/Tag/Switch 等子主题）
	TextColor    *types.Color // 覆盖主文字色
	Theme        *Theme       // 完整主题覆盖（优先于上面两项）
	Apply        func(*Theme) // 在解析出的主题副本上自定义修改（最灵活）
}

func NewConfigProvider(child Widget) *ConfigProvider {
	return &ConfigProvider{SingleChildWidget: SingleChildWidget{Child: child}}
}
func (c *ConfigProvider) WithPrimaryColor(col types.Color) *ConfigProvider { c.PrimaryColor = &col; return c }
func (c *ConfigProvider) WithTextColor(col types.Color) *ConfigProvider    { c.TextColor = &col; return c }
func (c *ConfigProvider) WithTheme(t Theme) *ConfigProvider                { c.Theme = &t; return c }
func (c *ConfigProvider) WithApply(fn func(*Theme)) *ConfigProvider        { c.Apply = fn; return c }

// resolved 解析本 Provider 生效的主题（基于外层作用域，故可嵌套累积）。
func (c *ConfigProvider) resolved() Theme {
	t := *activeTheme() // 基于当前作用域（嵌套时叠加外层）
	if c.Theme != nil {
		t = *c.Theme
	}
	if c.PrimaryColor != nil {
		applyPrimaryColor(&t, *c.PrimaryColor)
	}
	if c.TextColor != nil {
		t.TextColor = *c.TextColor
	}
	if c.Apply != nil {
		c.Apply(&t)
	}
	return t
}

func (c *ConfigProvider) CreateElement() Element {
	return &configProviderElement{BaseElement: BaseElement{widget: c}, cp: c}
}

type configProviderElement struct {
	BaseElement
	cp    *ConfigProvider
	child Element
	theme Theme // 本帧解析出的主题（作用域期间被 themeScope 指向）
}

func (e *configProviderElement) Build() []Element {
	if e.cp.Child != nil {
		e.child = CreateElementFor(e.cp.Child)
		e.child.Mount(e, 0)
		e.children = []Element{e.child}
		return e.children
	}
	e.children = nil
	return nil
}

func (e *configProviderElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		e.theme = e.cp.resolved() // Layout 也入作用域（部分组件按 theme 算尺寸）
		prev := themeScope
		themeScope = &e.theme
		res := e.child.Layout(ctx)
		themeScope = prev
		e.child.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *configProviderElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child == nil {
		return
	}
	e.theme = e.cp.resolved()
	prev := themeScope
	themeScope = &e.theme // 进入作用域：子树 elXxx()/CurrentTheme() 读本主题
	e.child.Paint(cvs, offset)
	themeScope = prev // 还原：作用域外不受影响
}

func (e *configProviderElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*ConfigProvider); ok {
		e.cp = nc
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
