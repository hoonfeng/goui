// 设置面板 —— 复刻参考 SettingsPanel（模态对话框 + 顶部 tab）。v1 实现「模型」tab（LLM 服务商/接口/
// Key/模型，存到用户配置目录，agent 据此建 Provider），其余 tab 占位待接入。帮助→打开设置 / Ctrl+,。
//
//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// appSettings 持久化设置（API Key 敏感，存用户配置目录，权限 0600，不入库）。
type appSettings struct {
	Provider    string `json:"provider"`
	BaseURL     string `json:"baseURL"`
	APIKey      string `json:"apiKey"`
	Model       string `json:"model"`
	Temperature string `json:"temperature"` // 字符串：留空=用服务端默认（区分于显式 0）
	MaxTokens   int    `json:"maxTokens"`   // 0=不下发
	LastProject      string   `json:"lastProject"`      // 兼容旧版单文件夹（迁移用）
	WorkspaceFolders []string `json:"workspaceFolders"` // 工作区文件夹列表（VS Code 多根，启动时恢复）
	// Agent 行为（默认值；对话输入区开关可临时覆盖本轮）
	AutoReview    bool `json:"autoReview"`
	Autonomous    bool `json:"autonomous"`
	AutoCollapse  bool `json:"autoCollapse"`
	MaxIterations int  `json:"maxIterations"`
	// 终端
	DefaultShell string `json:"defaultShell"` // cmd / powershell / gitbash
	TermFontSize int    `json:"termFontSize"` // 0=默认 13
	// 外观（深色固定；这里调编辑器）
	EditorFontSize int  `json:"editorFontSize"` // 0=默认 14
	HideMinimap    bool `json:"hideMinimap"`    // 反向存：零值=显示 minimap（默认）
}

var (
	theSettings        appSettings // 生效设置
	editingSettings    appSettings // 对话框编辑副本
	settingsLoaded     bool        // settings.json 是否存在（决定启动时是否覆盖 chat 内置默认）
	editingInstructions string     // 「指令」tab 编辑副本（存 <工作区>/.pair/rules.md）
)

// instructionsPath 项目的指令文件（随项目存 .pair/，agent 经 ProjectRules 注入）。
func instructionsPath() string {
	return filepath.Join(currentRoot(), ".pair", "rules.md")
}

func loadInstructions() string {
	data, _ := os.ReadFile(instructionsPath())
	return string(data)
}

func saveInstructions(s string) {
	p := instructionsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(s), 0o644)
}

// applyAgentSettings 把持久化设置应用到运行态（启动时若有存档 + 保存后调用）。
func applyAgentSettings() {
	theChatState.autoReview = theSettings.AutoReview
	theChatState.autonomous = theSettings.Autonomous
	theChatState.autoCollapse = theSettings.AutoCollapse
	if theSettings.DefaultShell != "" {
		theTerminal.shell = theSettings.DefaultShell
	}
}

// configDir 全局配置目录：安装目录（exe 所在）下的 config/ 子区。go run 的临时 exe→回退 cwd/config。
// 全局配置（settings.json 含 key、mcp.json）放这里；项目级（rules.md、memory）随项目放 .pair/。
func configDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		low := strings.ToLower(dir)
		if !strings.Contains(low, "go-build") && !strings.Contains(low, `\temp\`) && !strings.Contains(low, "/tmp/") {
			return filepath.Join(dir, "config")
		}
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, "config")
}

func settingsPath() string { return filepath.Join(configDir(), "settings.json") }

func loadSettings() {
	if data, err := os.ReadFile(settingsPath()); err == nil {
		_ = json.Unmarshal(data, &theSettings)
		settingsLoaded = true
		applyAgentSettings() // 有存档 → 用存档的 Agent 默认覆盖 chat 内置默认
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

// settingsTemperature 解析温度：留空/非法→-1（不下发，用服务端默认）。
func settingsTemperature() float64 {
	s := strings.TrimSpace(theSettings.Temperature)
	if s == "" {
		return -1
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return -1
}

// providerPresets 服务商表：每个含 id/显示名/baseURL + 该服务商的模型列表（下拉选项）。
// custom（自定义）无固定列表 → 模型改为手填输入框。
var providerPresets = []struct {
	name, label, base string
	models            []string
}{
	{"deepseek", "DeepSeek", "https://api.deepseek.com/v1", []string{"deepseek-chat", "deepseek-reasoner"}},
	{"openai", "OpenAI", "https://api.openai.com/v1", []string{"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini", "o3", "o4-mini"}},
	{"dashscope", "通义千问", "https://dashscope.aliyuncs.com/compatible-mode/v1", []string{"qwen-max", "qwen-plus", "qwen-turbo", "qwen2.5-coder-32b-instruct"}},
	{"zhipu", "智谱 GLM", "https://open.bigmodel.cn/api/paas/v4", []string{"glm-4-plus", "glm-4", "glm-4-air", "glm-4-flash"}},
	{"moonshot", "Moonshot (Kimi)", "https://api.moonshot.cn/v1", []string{"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"}},
	{"openrouter", "OpenRouter", "https://openrouter.ai/api/v1", []string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o", "google/gemini-pro-1.5", "deepseek/deepseek-chat"}},
	{"custom", "自定义 (OpenAI 兼容)", "", nil},
}

// providerByID 取服务商表项（找不到→custom 兜底，即最后一项）。
func providerByID(id string) (name, label, base string, models []string) {
	for _, p := range providerPresets {
		if p.name == id {
			return p.name, p.label, p.base, p.models
		}
	}
	last := providerPresets[len(providerPresets)-1]
	return last.name, last.label, last.base, last.models
}

// defaultModelFor 服务商的默认模型（列表首个；custom 为空）。
func defaultModelFor(id string) string {
	if _, _, _, models := providerByID(id); len(models) > 0 {
		return models[0]
	}
	return ""
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
	if editingSettings.Provider == "" { // 默认值：首次未配置时给 DeepSeek + 默认模型 + 对应接口地址
		editingSettings.Provider = "deepseek"
	}
	if editingSettings.BaseURL == "" {
		_, _, editingSettings.BaseURL, _ = providerByID(editingSettings.Provider)
	}
	if editingSettings.Model == "" {
		editingSettings.Model = defaultModelFor(editingSettings.Provider)
	}
	editingInstructions = loadInstructions()
	theSettingsBody.tab = "model"
	theSettingsBody.resetTok++
	var id int
	dlg := widget.NewDialog("设置", &SettingsBody{}).WithWidth(580).WithTransition("fade").WithFooter(
		widget.NewButton("取消", func() { widget.HideOverlay(id) }).WithColor(*ghBgTertiary).WithTextColor(ghText),
		widget.NewButton("保存", func() {
			theSettings = editingSettings
			saveSettings()
			saveInstructions(editingInstructions) // 项目指令 → .pair/rules.md
			settingsLoaded = true
			applyAgentSettings() // 保存即应用 Agent 设置到对话
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
		// 固定高度内容区（超出滚动）：各 tab 内容高度不一，固定高度→对话框尺寸恒定，
		// 切换标签不再因高度变化重新居中"跳动"，入场也不会因内容沉降"弹两次"。
		// 统一左右内边距(10)：内容左缘与上方标签文字对齐、左右留白对称（各 tab 不再各自设 padding）。
		widget.Div(widget.Style{Height: 420, FlexDirection: "column", AlignItems: "stretch",
			Padding: types.EdgeInsetsLTRB(10, 0, 10, 0)},
			expand(widget.NewScrollView(b.content())),
		),
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
	switch b.tab {
	case "model":
		return b.modelTab()
	case "agent":
		return b.agentTab()
	case "instructions":
		return b.instructionsTab()
	case "appearance":
		return b.appearanceTab()
	case "terminal":
		return b.terminalTab()
	case "mcp":
		return b.mcpTab()
	}
	return widget.Div(
		widget.Style{Height: 180, FlexDirection: "column", AlignItems: "center", JustifyContent: "center"},
		widget.Lucide("settings", widget.IconSize(26), widget.IconColor(ghTextMuted)),
		widget.Div(widget.Style{Height: 8}),
		label("该设置项待接入", ghTextMuted, 12),
	)
}

// agentTab Agent 行为设置：审批/自主/收起开关 + 最大迭代步数。
func (b *settingsBodyState) agentTab() widget.Widget {
	iterVal := ""
	if editingSettings.MaxIterations > 0 {
		iterVal = itoa(editingSettings.MaxIterations)
	}
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		label("审批与自主", ghTextMuted, 11),
		widget.Div(widget.Style{Height: 6}),
		settingsToggle("自动审核（写类工具免逐次确认）", editingSettings.AutoReview, func() {
			editingSettings.AutoReview = !editingSettings.AutoReview
			b.SetState()
		}),
		settingsToggle("自主模式（先列计划，连续完成所有步骤）", editingSettings.Autonomous, func() {
			editingSettings.Autonomous = !editingSettings.Autonomous
			b.SetState()
		}),
		settingsToggle("完成后自动收起上一轮对话", editingSettings.AutoCollapse, func() {
			editingSettings.AutoCollapse = !editingSettings.AutoCollapse
			b.SetState()
		}),
		settingsField("最大迭代步数（默认 30；自主模式翻倍）", settingsInput("30", iterVal, b.resetTok, func(t string) {
			editingSettings.MaxIterations, _ = strconv.Atoi(strings.TrimSpace(t))
		})),
		widget.Div(widget.Style{Height: 6}),
		label("提示：这些是默认值，保存即生效；对话输入区的开关可临时切换本轮。", ghTextMuted, 10),
	)
}

// instructionsTab 项目指令编辑（存 .pair/rules.md，agent 经 ProjectRules 注入系统提示）。
func (b *settingsBodyState) instructionsTab() widget.Widget {
	ta := widget.NewTextarea("写给 agent 的项目规则/约定，例如：代码风格、命名、禁忌、构建与测试命令、目录约定…", 9,
		func(t string) { editingInstructions = t })
	ta.Text = editingInstructions
	ta.ResetToken = b.resetTok
	ta.Color = ghText
	ta.CursorColor = ghText
	ta.PlaceholderColor = ghTextMuted
	ta.BGColor = *ghBgPrimary
	ta.BorderColor = *ghBorder
	ta.FocusBorderColor = *ghAccent
	ta.HoverBorderColor = *ghBorder
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		label("项目指令（agent 每轮读取并遵守）", ghTextMuted, 11),
		widget.Div(widget.Style{Height: 6}),
		ta,
		widget.Div(widget.Style{Height: 4}),
		label("保存到当前工作区 .pair/rules.md；与项目根的 AGENTS.md/CLAUDE.md 一并注入系统提示。", ghTextMuted, 10),
	)
}

// appearanceTab 外观设置：主题（深色固定）+ 编辑器字号 + Minimap 开关。
func (b *settingsBodyState) appearanceTab() widget.Widget {
	fsz := ""
	if editingSettings.EditorFontSize > 0 {
		fsz = itoa(editingSettings.EditorFontSize)
	}
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		label("主题", ghTextMuted, 11),
		widget.Div(widget.Style{Height: 6}),
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
			&widget.Button{
				SingleChildWidget: widget.SingleChildWidget{Child: label("深色", cWhite, 11)},
				OnClick:           func() {}, Color: *ghAccentEmph, MinHeight: 24, Padding: types.EdgeInsetsLTRB(10, 0, 10, 0),
			},
			widget.Div(widget.Style{Width: 8}),
			label("（目前固定深色主题）", ghTextMuted, 10),
		),
		settingsToggle("显示编辑器 Minimap（右侧缩略图）", !editingSettings.HideMinimap, func() {
			editingSettings.HideMinimap = !editingSettings.HideMinimap
			b.SetState()
		}),
		settingsField("编辑器字号（默认 14）", settingsInput("14", fsz, b.resetTok, func(t string) {
			editingSettings.EditorFontSize, _ = strconv.Atoi(strings.TrimSpace(t))
		})),
		widget.Div(widget.Style{Height: 6}),
		label("提示：保存后重开文件生效（或切换标签）。", ghTextMuted, 10),
	)
}

// terminalTab 终端设置：默认 shell（终端启动时用；运行中仍可点徽标临时切换）。
func (b *settingsBodyState) terminalTab() widget.Widget {
	cur := editingSettings.DefaultShell
	if cur == "" {
		cur = "cmd"
	}
	termFsz := ""
	if editingSettings.TermFontSize > 0 {
		termFsz = itoa(editingSettings.TermFontSize)
	}
	var btns []widget.Widget
	for i, s := range []struct{ id, label string }{{"cmd", "CMD"}, {"powershell", "PowerShell"}, {"gitbash", "Git Bash"}} {
		ss := s
		if i > 0 {
			btns = append(btns, widget.Div(widget.Style{Width: 6}))
		}
		tc, bg := ghText, *ghBgTertiary
		if cur == ss.id {
			tc, bg = cWhite, *ghAccentEmph
		}
		btns = append(btns, &widget.Button{
			SingleChildWidget: widget.SingleChildWidget{Child: label(ss.label, tc, 11)},
			OnClick:           func() { editingSettings.DefaultShell = ss.id; b.SetState() },
			Color:             bg, MinHeight: 24, Padding: types.EdgeInsetsLTRB(10, 0, 10, 0),
		})
	}
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		label("默认 Shell（终端启动时使用）", ghTextMuted, 11),
		widget.Div(widget.Style{Height: 6}),
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, btns),
		settingsField("终端字号（默认 13）", settingsInput("13", termFsz, b.resetTok, func(t string) {
			editingSettings.TermFontSize, _ = strconv.Atoi(strings.TrimSpace(t))
		})),
		widget.Div(widget.Style{Height: 8}),
		label("运行中也可点终端输入行左侧的徽标临时切换 shell。Git Bash 需 git 的 bash 在 PATH。", ghTextMuted, 10),
	)
}

// mcpTab MCP 服务器配置说明（companion 启动对话时读 mcp.json）。
func (b *settingsBodyState) mcpTab() widget.Widget {
	path := filepath.Join(filepath.Dir(settingsPath()), "mcp.json")
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		label("MCP 服务器", ghTextMuted, 11),
		widget.Div(widget.Style{Height: 6}),
		label("对话开始时读取此文件、连接外部 MCP 服务器并注册其工具：", ghText, 11),
		widget.Div(widget.Style{Height: 6}),
		widget.Div(
			widget.Style{BackgroundColor: ghBgPrimary, BorderColor: ghBorder, BorderWidth: 1, BorderRadius: 5,
				Padding: types.EdgeInsets(8)},
			monoLabel(path, ghText, 11),
		),
		widget.Div(widget.Style{Height: 10}),
		label(`格式（同 Claude Desktop）：`, ghText, 11),
		widget.Div(
			widget.Style{BackgroundColor: ghBgPrimary, BorderColor: ghBorder, BorderWidth: 1, BorderRadius: 5,
				Padding: types.EdgeInsets(8)},
			monoLabel(`{"mcpServers":{"名字":{"command":"npx","args":["-y","包名"]}}}`, ghTextMuted, 10),
		),
		widget.Div(widget.Style{Height: 8}),
		label("编辑后重开对话生效；起不来的服务器会自动跳过、不影响其它工具。", ghTextMuted, 10),
	)
}

// settingsToggle 一行开关：左标签 + 右「开/关」按钮。
func settingsToggle(lbl string, on bool, toggle func()) widget.Widget {
	state, tc, bg := "关", ghText, *ghBgTertiary
	if on {
		state, tc, bg = "开", cWhite, *ghAccentEmph
	}
	return widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "center", Padding: types.EdgeInsetsLTRB(0, 12, 0, 0)},
		expand(label(lbl, ghText, 12)),
		&widget.Button{
			SingleChildWidget: widget.SingleChildWidget{Child: label(state, tc, 11)},
			OnClick:           toggle,
			Color:             bg, MinHeight: 22, MinWidth: 44, Padding: types.EdgeInsetsLTRB(10, 0, 10, 0),
		},
	)
}

const settingsCtlW = 536 // 设置控件（下拉/输入）宽度，对齐内容区（556 - 左右内边距 10）

func (b *settingsBodyState) modelTab() widget.Widget {
	// 服务商下拉
	provOpts := make([]widget.SelectOption, 0, len(providerPresets))
	for _, p := range providerPresets {
		provOpts = append(provOpts, widget.SelectOption{Label: p.label, Value: p.name})
	}
	provSel := widget.NewSelect(provOpts).WithValue(editingSettings.Provider).WithWidth(settingsCtlW).
		WithOnChanged(func(v string) {
			editingSettings.Provider = v
			if v != "custom" { // 选服务商→自动填 baseURL + 默认模型（默认值）
				_, _, base, _ := providerByID(v)
				editingSettings.BaseURL = base
				editingSettings.Model = defaultModelFor(v)
			}
			b.resetTok++
			b.SetState()
		})

	// 模型：标准服务商=下拉（该商的模型列表）；custom=手填输入
	var modelCtl widget.Widget
	if editingSettings.Provider == "custom" {
		modelCtl = settingsInput("自定义模型 ID", editingSettings.Model, b.resetTok, func(t string) { editingSettings.Model = t })
	} else {
		_, _, _, models := providerByID(editingSettings.Provider)
		mOpts := make([]widget.SelectOption, 0, len(models))
		for _, m := range models {
			mOpts = append(mOpts, widget.SelectOption{Label: m, Value: m})
		}
		modelCtl = widget.NewSelect(mOpts).WithValue(editingSettings.Model).WithWidth(settingsCtlW).
			WithOnChanged(func(v string) { editingSettings.Model = v; b.SetState() })
	}

	maxTokStr := ""
	if editingSettings.MaxTokens > 0 {
		maxTokStr = itoa(editingSettings.MaxTokens)
	}
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		settingsField("服务商 (Provider)", provSel),
		settingsField("模型 (Model)", modelCtl),
		settingsField("接口地址 (Base URL)", settingsInput("https://...", editingSettings.BaseURL, b.resetTok, func(t string) { editingSettings.BaseURL = t })),
		settingsField("API Key", settingsInput("sk-...", editingSettings.APIKey, b.resetTok, func(t string) { editingSettings.APIKey = t })),
		settingsField("温度 Temperature（留空=服务端默认，0~2）", settingsInput("留空=默认", editingSettings.Temperature, b.resetTok, func(t string) { editingSettings.Temperature = t })),
		settingsField("最大输出 Token（0=不下发）", settingsInput("0", maxTokStr, b.resetTok, func(t string) { editingSettings.MaxTokens, _ = strconv.Atoi(strings.TrimSpace(t)) })),
		widget.Div(widget.Style{Height: 4}),
		label("提示：选服务商自动填接口地址与默认模型；留空 Key 则回退环境变量（DEEPSEEK_API_KEY 等）。", ghTextMuted, 10),
	)
}

func settingsField(lbl string, in widget.Widget) widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch", Padding: types.EdgeInsetsLTRB(0, 12, 0, 0)},
		label(lbl, ghText, 12), // 与 settingsToggle 标签同字号(12)，统一视觉
		widget.Div(widget.Style{Height: 5}),
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
