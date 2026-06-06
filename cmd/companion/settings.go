// 设置面板 —— 复刻参考 SettingsPanel（模态对话框 + 顶部 tab）。v1 实现「模型」tab（LLM 服务商/接口/
// Key/模型，存到用户配置目录，agent 据此建 Provider），其余 tab 占位待接入。帮助→打开设置 / Ctrl+,。
//
//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// appSettings 持久化设置（API Key 敏感，存用户配置目录，权限 0600，不入库）。
type appSettings struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"baseURL"`
	APIKey   string `json:"apiKey"`
	Model    string `json:"model"`
}

var (
	theSettings     appSettings // 生效设置
	editingSettings appSettings // 对话框编辑副本
)

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "companion", "settings.json")
}

func loadSettings() {
	if data, err := os.ReadFile(settingsPath()); err == nil {
		_ = json.Unmarshal(data, &theSettings)
	}
}

func saveSettings() {
	p := settingsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if data, err := json.MarshalIndent(theSettings, "", "  "); err == nil {
		_ = os.WriteFile(p, data, 0o600) // 含 API Key → 仅本人可读
	}
}

// settingsConfigured 是否已在设置里配好可用 Provider。
func settingsConfigured() bool {
	return theSettings.APIKey != "" && theSettings.BaseURL != "" && theSettings.Model != ""
}

var providerPresets = []struct{ name, label, base, model string }{
	{"deepseek", "DeepSeek", "https://api.deepseek.com/v1", "deepseek-chat"},
	{"openai", "OpenAI", "https://api.openai.com/v1", "gpt-4o"},
	{"dashscope", "通义千问", "https://dashscope.aliyuncs.com/compatible-mode/v1", "qwen-plus"},
	{"moonshot", "Moonshot", "https://api.moonshot.cn/v1", "moonshot-v1-8k"},
	{"openrouter", "OpenRouter", "https://openrouter.ai/api/v1", "anthropic/claude-3.5-sonnet"},
	{"custom", "自定义", "", ""},
}

// ─── 设置对话框 ────────────────────────────────────────────────

var settingsTabs = []struct{ id, label string }{
	{"model", "模型"}, {"agent", "Agent"}, {"instructions", "指令"}, {"appearance", "外观"},
	{"terminal", "终端"}, {"mcp", "MCP"}, {"skills", "Skills"},
}

var theSettingsBody = &settingsBodyState{tab: "model"}

// SettingsBody 设置对话框主体（tab + 内容）。
type SettingsBody struct{ widget.StatefulWidget }

func (b *SettingsBody) CreateState() widget.State { return theSettingsBody }

type settingsBodyState struct {
	widget.BaseState
	tab      string
	resetTok int // 预设填充时 +1，强制输入框刷新显示新值
}

// openSettings 打开设置模态对话框（帮助→打开设置 / Ctrl+,）。
func openSettings() {
	editingSettings = theSettings
	theSettingsBody.tab = "model"
	theSettingsBody.resetTok++
	var id int
	dlg := widget.NewDialog("设置", &SettingsBody{}).WithWidth(580).WithFooter(
		widget.NewButton("取消", func() { widget.HideOverlay(id) }).WithColor(*ghBgTertiary).WithTextColor(ghText),
		widget.NewButton("保存", func() {
			theSettings = editingSettings
			saveSettings()
			widget.HideOverlay(id)
		}).WithColor(*ghAccentEmph).WithTextColor(cWhite),
	)
	id = widget.ShowDialog(dlg)
}

func (b *settingsBodyState) Build(ctx widget.BuildContext) widget.Widget {
	tabs := make([]widget.Widget, 0, len(settingsTabs))
	for _, t := range settingsTabs {
		tabs = append(tabs, b.tabBtn(t.id, t.label))
	}
	return widget.Div(
		widget.Style{Width: 556, FlexDirection: "column", AlignItems: "stretch"},
		widget.Div( // tab 条
			widget.Style{FlexDirection: "row", AlignItems: "center", BackgroundColor: ghBgSecondary,
				BorderColor: ghBorder, BorderWidth: 1, Padding: types.EdgeInsetsLTRB(0, 0, 0, 2)},
			tabs,
		),
		widget.Div(widget.Style{Height: 12}),
		b.content(),
	)
}

func (b *settingsBodyState) tabBtn(id, lbl string) widget.Widget {
	on := b.tab == id
	tc, bg := ghTextMuted, *ghBgSecondary // 非激活：与条背景同色（Button 零色会落主题蓝，须显式给）
	if on {
		tc, bg = ghText, *ghBgTertiary
	}
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: label(lbl, tc, 12)},
		OnClick:           func() { b.tab = id; b.SetState() },
		Color:             bg,
		MinHeight:         26,
		Padding:           types.EdgeInsetsLTRB(10, 0, 10, 0),
	}
}

func (b *settingsBodyState) content() widget.Widget {
	if b.tab == "model" {
		return b.modelTab()
	}
	return widget.Div(
		widget.Style{Height: 180, FlexDirection: "column", AlignItems: "center", JustifyContent: "center"},
		widget.Lucide("settings", widget.IconSize(26), widget.IconColor(ghTextMuted)),
		widget.Div(widget.Style{Height: 8}),
		label("该设置项待接入", ghTextMuted, 12),
	)
}

func (b *settingsBodyState) modelTab() widget.Widget {
	// 预设按钮（点选填充 接口地址 + 模型）
	presets := make([]widget.Widget, 0, len(providerPresets)*2)
	for i, p := range providerPresets {
		pp := p
		if i > 0 {
			presets = append(presets, widget.Div(widget.Style{Width: 6}))
		}
		on := editingSettings.Provider == pp.name
		tc, bg := ghText, *ghBgTertiary
		if on {
			tc, bg = cWhite, *ghAccentEmph
		}
		presets = append(presets, &widget.Button{
			SingleChildWidget: widget.SingleChildWidget{Child: label(pp.label, tc, 11)},
			OnClick: func() {
				editingSettings.Provider = pp.name
				if pp.name != "custom" {
					editingSettings.BaseURL, editingSettings.Model = pp.base, pp.model
				}
				b.resetTok++
				b.SetState()
			},
			Color: bg, MinHeight: 24, Padding: types.EdgeInsetsLTRB(9, 0, 9, 0),
		})
	}
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(2, 0, 2, 0)},
		label("服务商预设", ghTextMuted, 11),
		widget.Div(widget.Style{Height: 6}),
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, presets),
		settingsField("接口地址 (Base URL)", settingsInput("https://...", editingSettings.BaseURL, b.resetTok, func(t string) { editingSettings.BaseURL = t })),
		settingsField("API Key", settingsInput("sk-...", editingSettings.APIKey, b.resetTok, func(t string) { editingSettings.APIKey = t })),
		settingsField("模型 (Model)", settingsInput("deepseek-chat", editingSettings.Model, b.resetTok, func(t string) { editingSettings.Model = t })),
		widget.Div(widget.Style{Height: 4}),
		label("提示：保存后对话即用此配置；留空则回退到环境变量（DEEPSEEK_API_KEY 等）。", ghTextMuted, 10),
	)
}

func settingsField(lbl string, in widget.Widget) widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(0, 10, 0, 0)},
		label(lbl, ghText, 11),
		widget.Div(widget.Style{Height: 4}),
		in,
	)
}

func settingsInput(placeholder, val string, tok int, onChanged func(string)) widget.Widget {
	in := widget.NewInput(placeholder, onChanged)
	in.Text = val
	in.ResetToken = tok
	in.Color = ghText
	in.CursorColor = ghText
	in.BGColor = *ghBgPrimary
	in.BorderColor = *ghBorder
	in.FocusBorderColor = *ghAccent
	in.HoverBorderColor = *ghBorder
	return in
}
