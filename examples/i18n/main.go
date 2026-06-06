// 多语言（i18n）演示：点击按钮在 英文 / 中文 / 阿拉伯语 之间切换，
// 观察文本翻译、数字本地化、复数选择，以及阿拉伯语下的 RTL 行镜像。
//
// 运行方式（需要 CGO + libSkiaSharp.dll + fonts/ 在工作目录）:
//
//	$env:CGO_ENABLED='1'; go run ./examples/i18n/
package main

import (
	"runtime"
	"time"

	"github.com/user/goui/internal/app"
	"github.com/user/goui/internal/i18n"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
	_ "github.com/user/goui/internal/window/win32"
)

func init() {
	i18n.Register("en", map[string]string{
		"title": "Internationalization Demo", "greeting": "Hello, %s!",
		"count_label": "Items:", "num_label": "Number:", "switch": "Switch Language",
		"date_label": "Date:", "time_label": "Time:",
		"para": "This is a longer paragraph that wraps across several lines, so you can see how text aligns to the writing direction of each language.",
	})
	i18n.Register("zh-CN", map[string]string{
		"title": "多语言演示", "greeting": "你好，%s！",
		"count_label": "数量：", "num_label": "数字：", "switch": "切换语言",
		"date_label": "日期：", "time_label": "时间：",
		"para": "这是一段较长的文字，会自动换行成多行，用来展示在不同书写方向下文本的对齐方式。",
	})
	i18n.Register("ar", map[string]string{
		"title": "عرض التدويل", "greeting": "مرحبا، %s!",
		"count_label": "العناصر:", "num_label": "رقم:", "switch": "تبديل اللغة",
		"date_label": "التاريخ:", "time_label": "الوقت:",
		"para": "هذه فقرة أطول تلتف عبر عدة أسطر، حتى تتمكن من رؤية كيفية محاذاة النص حسب اتجاه الكتابة لكل لغة.",
	})
	i18n.SetFallback("en")
	i18n.SetLocale("en")
}

type I18nDemo struct{ widget.StatefulWidget }

func (d *I18nDemo) CreateState() widget.State { return &i18nState{} }

type i18nState struct {
	widget.BaseState
	idx int
}

var locales = []i18n.Locale{"en", "zh-CN", "ar"}

func labelRow(label, value string) widget.Widget {
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 6, 0, 6)},
		widget.HBox(
			widget.NewText(label, types.ColorFromRGB(120, 120, 120)),
			widget.SpacerDiv(),
			widget.NewText(value, types.ColorFromRGB(33, 33, 33)),
		),
	)
}

func (s *i18nState) Build(ctx widget.BuildContext) widget.Widget {
	plural := i18n.Plural(3, map[string]string{"one": "{n} item", "other": "{n} items"})
	return widget.Div(
		widget.Style{Padding: types.EdgeInsets(36)},
		widget.H2(i18n.T("title")),
		widget.NewText(i18n.Tf("greeting", "goui"), types.ColorFromRGB(33, 33, 33)),
		labelRow(i18n.T("count_label"), plural),
		labelRow(i18n.T("num_label"), i18n.FormatNumber(1234567)),
		labelRow(i18n.T("date_label"), i18n.FormatDate(time.Now())),
		labelRow(i18n.T("time_label"), i18n.FormatTime(time.Now())),
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 14, 0, 0)},
			// 多行长文本：阿拉伯语下自动右对齐（RTL），中英文左对齐。
			widget.NewText(i18n.T("para"), types.ColorFromRGB(90, 90, 90)),
		),
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 16, 0, 0)},
			// SetLocale 会触发全树重建并重新布局，界面自动刷新为新语言，无需手动 SetState。
			widget.NewButton(i18n.T("switch"), func() {
				s.idx = (s.idx + 1) % len(locales)
				i18n.SetLocale(locales[s.idx])
			}),
		),
	)
}

func main() {
	runtime.LockOSThread()
	application := app.NewApplication()
	application.SetRootWidget(&I18nDemo{})
	cfg := app.DefaultConfig()
	cfg.Title = "goui — i18n Demo"
	application.Run(cfg)
}
