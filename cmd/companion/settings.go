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

// appSettings 持久化设置 —— 字段对齐参考 settings.ts（扁平存储；分组注释）。API Key 敏感，存安装目录 config/，不入库。
type appSettings struct {
	// 主模型（model）
	Provider         string `json:"provider"`
	BaseURL          string `json:"baseURL"`
	APIKey           string `json:"apiKey"`
	Model            string `json:"model"` // 兼容旧单模型字段（迁移→ExecuteModel）
	PlanModel        string `json:"planModel"`
	ExecuteModel     string `json:"executeModel"`
	ReviewModel      string `json:"reviewModel"`
	Temperature      string `json:"temperature"`      // 字符串：留空=用服务端默认（区分显式 0）
	ThinkingMode     string `json:"thinkingMode"`     // non-thinking / thinking / thinking_max
	MaxTokens        int    `json:"maxTokens"`        // 0=不下发
	ContextMaxTokens int    `json:"contextMaxTokens"` // 上下文窗口上限
	// 压缩模型（compressModel）
	CompressEnabled      bool   `json:"compressEnabled"`
	CompressProvider     string `json:"compressProvider"`
	CompressAPIKey       string `json:"compressApiKey"`
	CompressBaseURL      string `json:"compressBaseURL"`
	CompressModel        string `json:"compressModel"`
	CompressThinkingMode string `json:"compressThinkingMode"`
	// 工作区
	LastProject      string   `json:"lastProject"`
	WorkspaceFolders []string `json:"workspaceFolders"`
	// Agent 行为
	AutoReview         bool   `json:"autoReview"`
	Autonomous         bool   `json:"autonomous"`
	AutoCollapse       bool   `json:"autoCollapse"`
	MaxIterations      int    `json:"maxIterations"`
	MaxParallel        int    `json:"maxParallelAgents"`
	ReviewRetries      int    `json:"maxReviewRetries"`
	AutoIterate        bool   `json:"autoIterateOnRejection"`
	RequireApproval    bool   `json:"requireHumanApprovalForDestructive"`
	Benchmark          bool   `json:"enableBenchmarking"`
	SystemInstructions string `json:"systemInstructions"`
	SearxngURL         string `json:"searxngUrl"`
	// 终端
	DefaultShell string `json:"defaultShell"` // auto / cmd / powershell / git-bash
	TermFontSize int    `json:"termFontSize"` // 0=默认 13
	TermEncoding string `json:"termEncoding"` // auto / utf-8 / gbk
	// 外观
	Theme          string `json:"theme"` // dark / light / high-contrast / solarized-light / dracula
	FontFamily     string `json:"fontFamily"`
	EditorFontSize int    `json:"editorFontSize"` // 0=默认 14
	HideMinimap    bool   `json:"hideMinimap"`    // 反向存：零值=显示 minimap
	// 思想（philosophy）
	PhilosophyEnabled  bool              `json:"philosophyEnabled"`
	PhilosophySelected []string          `json:"philosophySelected"`
	PhilosophyRoles    map[string]string `json:"philosophyRoles"`
	// MCP
	AutoConnectMCP bool `json:"autoConnectMCP"`
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
	theChatState.autoReview = !theSettings.RequireApproval // 破坏性操作需人工确认 ↔ 手动审核
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

// defaultSettings 默认值 —— 对齐参考 settings.ts 的 DEFAULTS。
func defaultSettings() appSettings {
	return appSettings{
		Provider: "deepseek", BaseURL: "https://api.deepseek.com/v1",
		PlanModel: "deepseek-v4-pro", ExecuteModel: "deepseek-v4-flash", ReviewModel: "deepseek-v4-pro",
		Temperature: "1.0", ThinkingMode: "thinking", MaxTokens: 131072, ContextMaxTokens: 1000000,
		CompressEnabled: true, CompressProvider: "deepseek", CompressBaseURL: "https://api.deepseek.com/v1",
		CompressModel: "deepseek-v4-flash", CompressThinkingMode: "non-thinking",
		MaxIterations: 50, MaxParallel: 3, ReviewRetries: 3, AutoIterate: true, RequireApproval: true, Benchmark: true,
		DefaultShell: "auto", TermEncoding: "auto",
		Theme: "dark", FontFamily: "'Cascadia Code', 'Fira Code', Consolas, monospace",
		PhilosophySelected: []string{"tao-te-ching", "huangdi-yinfu-jing", "sunzi-bingfa"},
		AutoConnectMCP:     true,
	}
}

func loadSettings() {
	theSettings = defaultSettings() // 先铺默认值，再用存档覆盖（缺字段保留默认）
	if data, err := os.ReadFile(settingsPath()); err == nil {
		_ = json.Unmarshal(data, &theSettings)
		settingsLoaded = true
	}
	if theSettings.ExecuteModel == "" && theSettings.Model != "" { // 迁移旧单模型字段
		theSettings.ExecuteModel = theSettings.Model
	}
	if settingsLoaded {
		applyAgentSettings()
	}
}

func saveSettings() {
	p := settingsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if data, err := json.MarshalIndent(theSettings, "", "  "); err == nil {
		_ = os.WriteFile(p, data, 0o600) // 含 API Key → 仅本人可读
	}
}

// mainModel 主循环用的模型：执行模型优先（参考的 executeModel），回退旧 Model 字段。
func mainModel() string {
	if theSettings.ExecuteModel != "" {
		return theSettings.ExecuteModel
	}
	return theSettings.Model
}

// settingsConfigured 是否已在设置里配好可用 Provider。
func settingsConfigured() bool {
	return theSettings.APIKey != "" && theSettings.BaseURL != "" && mainModel() != ""
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

// modelEntry 一个模型选项（id + 显示名）。
type modelEntry struct{ id, name string }

// providerPresets 服务商表 —— 1:1 复刻参考源 src/agent/llm/gateway.ts 的 BUILTIN_PROVIDERS
// （id/显示名/baseURL + 每商完整模型列表，原样照搬，勿改）。custom 无固定列表→模型手填。
var providerPresets = []struct {
	name, label, base string
	models            []modelEntry
}{
	{"deepseek", "DeepSeek", "https://api.deepseek.com/v1", []modelEntry{
		{"deepseek-v4-flash", "DeepSeek V4 Flash"},
		{"deepseek-v4-pro", "DeepSeek V4 Pro"},
	}},
	{"openai", "OpenAI", "https://api.openai.com/v1", []modelEntry{
		{"gpt-5.5", "GPT-5.5（旗舰）"},
		{"gpt-5.4", "GPT-5.4（编码）"},
		{"gpt-5.4-mini", "GPT-5.4 Mini"},
		{"gpt-4.1", "GPT-4.1"},
		{"gpt-4.1-mini", "GPT-4.1 Mini"},
		{"gpt-4o", "GPT-4o（多模态）"},
		{"o3", "o3（推理）"},
		{"o3-pro", "o3 Pro（深度推理）"},
		{"o4-mini", "o4 Mini（快速推理）"},
		{"o4-mini-deep-research", "o4 Mini Deep Research"},
	}},
	{"qwen", "通义千问 (Qwen)", "https://dashscope.aliyuncs.com/compatible-mode/v1", []modelEntry{
		{"qwen3.7-max", "Qwen3.7 Max（旗舰）"},
		{"qwen3.6-plus", "Qwen3.6 Plus（增强）"},
		{"qwen3.6-flash", "Qwen3.6 Flash（快速）"},
		{"qwen3-235b-a22b", "Qwen3 235B-A22B（MoE）"},
		{"qwen-turbo-latest", "Qwen Turbo（轻量）"},
	}},
	{"zhipu", "智谱 (GLM)", "https://open.bigmodel.cn/api/paas/v4", []modelEntry{
		{"glm-5.1", "GLM-5.1（旗舰）"},
		{"glm-5", "GLM-5（高智能）"},
		{"glm-5-turbo", "GLM-5 Turbo（增强）"},
		{"glm-4.7", "GLM-4.7（高智能）"},
		{"glm-4.7-flashx", "GLM-4.7 FlashX（高速）"},
		{"glm-4.7-flash", "GLM-4.7 Flash（免费）"},
		{"glm-4.6", "GLM-4.6（超强性能）"},
		{"glm-4.5-air", "GLM-4.5 Air（高性价比）"},
		{"glm-4-long", "GLM-4 Long（超长上下文）"},
		{"glm-5v-turbo", "GLM-5V Turbo（多模态）"},
	}},
	{"moonshot", "月之暗面 (Kimi)", "https://api.moonshot.cn/v1", []modelEntry{
		{"kimi-k2.6", "Kimi K2.6（旗舰）"},
		{"kimi-k2.5", "Kimi K2.5（增强）"},
		{"kimi-k2", "Kimi K2（基础）"},
		{"kimi-k2-thinking", "Kimi K2 Thinking（深度思考）"},
		{"moonshot-v1-128k", "Moonshot v1 128K"},
	}},
	{"ocat", "Ocat.run", "https://ocat.run/v1", []modelEntry{
		{"ocat-default", "Ocat 模型（动态加载）"},
	}},
	{"custom", "自定义 (OpenAI 兼容)", "http://localhost:11434/v1", nil},
}

// providerByID 取服务商表项（找不到→custom 兜底，即最后一项）。
func providerByID(id string) (name, label, base string, models []modelEntry) {
	for _, p := range providerPresets {
		if p.name == id {
			return p.name, p.label, p.base, p.models
		}
	}
	last := providerPresets[len(providerPresets)-1]
	return last.name, last.label, last.base, last.models
}

// defaultModelFor 服务商的默认模型 id（列表首个；custom 为空）。
func defaultModelFor(id string) string {
	if _, _, _, models := providerByID(id); len(models) > 0 {
		return models[0].id
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
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		label("Agent 行为", ghTextMuted, 11),
		settingsSlider("最大迭代次数: "+itoa(editingSettings.MaxIterations), float64(editingSettings.MaxIterations), 5, 200, 1, func(v float64) {
			editingSettings.MaxIterations = int(v)
			b.SetState()
		}),
		settingsSlider("并行 Agent 数: "+itoa(editingSettings.MaxParallel), float64(editingSettings.MaxParallel), 1, 8, 1, func(v float64) {
			editingSettings.MaxParallel = int(v)
			b.SetState()
		}),
		settingsSlider("审核重试次数: "+itoa(editingSettings.ReviewRetries), float64(editingSettings.ReviewRetries), 0, 10, 1, func(v float64) {
			editingSettings.ReviewRetries = int(v)
			b.SetState()
		}),
		settingsToggle("审核驳回自动迭代修正", editingSettings.AutoIterate, func() {
			editingSettings.AutoIterate = !editingSettings.AutoIterate
			b.SetState()
		}),
		settingsToggle("破坏性操作需人工确认", editingSettings.RequireApproval, func() {
			editingSettings.RequireApproval = !editingSettings.RequireApproval
			b.SetState()
		}),
		settingsToggle("任务完成后自动评测评分", editingSettings.Benchmark, func() {
			editingSettings.Benchmark = !editingSettings.Benchmark
			b.SetState()
		}),
		label("关闭后 Agent 完成任务将跳过基准评测环节，减少等待时间。", ghTextMuted, 10),
		// ── 网络服务 ──
		widget.Div(widget.Style{Height: 14}),
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("globe", widget.IconSize(13), widget.IconColor(ghTextMuted)),
			widget.Div(widget.Style{Width: 6}),
			label("网络服务", ghTextMuted, 11),
		),
		settingsField("SearXNG 搜索地址（留空使用 DuckDuckGo）", settingsInput("http://192.168.1.100:8888", editingSettings.SearxngURL, b.resetTok, func(t string) {
			editingSettings.SearxngURL = t
		})),
		label("配置后 web_search 将使用自托管 SearXNG 实例进行搜索，支持更大结果量与自定义搜索引擎。", ghTextMuted, 10),
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

// providerSelect 服务商下拉。onPick(v) 在切换时由调用方处理 baseURL/模型联动。
func providerSelect(value string, onPick func(string)) widget.Widget {
	opts := make([]widget.SelectOption, 0, len(providerPresets))
	for _, p := range providerPresets {
		opts = append(opts, widget.SelectOption{Label: p.label, Value: p.name})
	}
	return widget.NewSelect(opts).WithValue(value).WithWidth(settingsCtlW).WithOnChanged(onPick)
}

func (b *settingsBodyState) modelTab() widget.Widget {
	prov := editingSettings.Provider
	temp := parseTempOr(editingSettings.Temperature, 1.0)
	maxTokStr := ""
	if editingSettings.MaxTokens > 0 {
		maxTokStr = itoa(editingSettings.MaxTokens)
	}
	provSel := providerSelect(prov, func(v string) {
		editingSettings.Provider = v
		if v != "custom" { // 选服务商→自动填 baseURL + 三角色默认模型
			_, _, base, _ := providerByID(v)
			dm := defaultModelFor(v)
			editingSettings.BaseURL = base
			editingSettings.PlanModel, editingSettings.ExecuteModel, editingSettings.ReviewModel = dm, dm, dm
		}
		b.resetTok++
		b.SetState()
	})
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		label("模型配置", ghTextMuted, 11),
		settingsField("服务提供商", provSel),
		settingsField("API Key", settingsInput("sk-...", editingSettings.APIKey, b.resetTok, func(t string) { editingSettings.APIKey = t })),
		settingsField("API 地址", settingsInput("https://...", editingSettings.BaseURL, b.resetTok, func(t string) { editingSettings.BaseURL = t })),
		settingsField("规划模型", modelSelectFor(prov, editingSettings.PlanModel, b.resetTok, func(v string) { editingSettings.PlanModel = v; b.SetState() })),
		settingsField("执行模型", modelSelectFor(prov, editingSettings.ExecuteModel, b.resetTok, func(v string) { editingSettings.ExecuteModel = v; b.SetState() })),
		settingsField("审核模型", modelSelectFor(prov, editingSettings.ReviewModel, b.resetTok, func(v string) { editingSettings.ReviewModel = v; b.SetState() })),
		settingsSlider("温度: "+strconv.FormatFloat(temp, 'f', 1, 64), temp, 0, 2, 0.1, func(v float64) {
			editingSettings.Temperature = strconv.FormatFloat(v, 'f', 1, 64)
			b.SetState()
		}),
		settingsField("思考模式", thinkingSelect(editingSettings.ThinkingMode, func(v string) { editingSettings.ThinkingMode = v; b.SetState() })),
		settingsField("最大 Token", settingsInput("131072", maxTokStr, b.resetTok, func(t string) { editingSettings.MaxTokens, _ = strconv.Atoi(strings.TrimSpace(t)) })),
		settingsSlider("上下文窗口: "+fmtContext(editingSettings.ContextMaxTokens), float64(editingSettings.ContextMaxTokens), 32000, 1000000, 32000, func(v float64) {
			editingSettings.ContextMaxTokens = int(v)
			b.SetState()
		}),
		label("超过此值自动压缩上下文 · DeepSeek: 1M", ghTextMuted, 10),
		// ── 压缩模型段 ──
		widget.Div(widget.Style{Height: 16}),
		label("压缩模型（上下文动态剪枝，独立配置）", ghTextMuted, 11),
		settingsToggle("启用压缩模型（关闭则使用规则式压缩）", editingSettings.CompressEnabled, func() {
			editingSettings.CompressEnabled = !editingSettings.CompressEnabled
			b.SetState()
		}),
		b.compressSection(),
	)
}

// compressSection 压缩模型字段（仅启用时显示）。
func (b *settingsBodyState) compressSection() widget.Widget {
	if !editingSettings.CompressEnabled {
		return widget.Div(widget.Style{})
	}
	cp := editingSettings.CompressProvider
	if cp == "" {
		cp = "deepseek"
	}
	provSel := providerSelect(cp, func(v string) {
		editingSettings.CompressProvider = v
		if v != "custom" {
			_, _, base, _ := providerByID(v)
			editingSettings.CompressBaseURL = base
			editingSettings.CompressModel = defaultModelFor(v)
		}
		b.resetTok++
		b.SetState()
	})
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		settingsField("服务提供商", provSel),
		settingsField("API Key", settingsInput("留空则复用主模型 Key", editingSettings.CompressAPIKey, b.resetTok, func(t string) { editingSettings.CompressAPIKey = t })),
		settingsField("API 地址", settingsInput("https://...", editingSettings.CompressBaseURL, b.resetTok, func(t string) { editingSettings.CompressBaseURL = t })),
		settingsField("模型", modelSelectFor(cp, editingSettings.CompressModel, b.resetTok, func(v string) { editingSettings.CompressModel = v; b.SetState() })),
		settingsField("思考模式", thinkingSelect(editingSettings.CompressThinkingMode, func(v string) { editingSettings.CompressThinkingMode = v; b.SetState() })),
		widget.Div(widget.Style{Height: 4}),
		label("提示：压缩模型建议用轻量模型并关闭思考模式，以降低延迟和成本。", ghTextMuted, 10),
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

// settingsSlider 深色滑块（field 标签含实时值）。
func settingsSlider(lbl string, val, min, max, step float64, onChange func(float64)) widget.Widget {
	sl := widget.NewSlider(val, onChange).WithRange(min, max).WithStep(step)
	sl.ActiveColor, sl.ThumbColor = *ghAccentEmph, *ghAccentEmph
	sl.InactiveColor = *ghBgTertiary
	sl.LabelColor = ghText
	return settingsField(lbl, sl)
}

// modelSelectFor 某服务商的模型下拉（custom→手填）；保留已存的非列表内值可见可选。
func modelSelectFor(provider, value string, tok int, onChange func(string)) widget.Widget {
	if provider == "custom" {
		return settingsInput("自定义模型 ID", value, tok, onChange)
	}
	_, _, _, models := providerByID(provider)
	opts := make([]widget.SelectOption, 0, len(models)+1)
	inList := false
	for _, m := range models {
		opts = append(opts, widget.SelectOption{Label: m.name, Value: m.id})
		if m.id == value {
			inList = true
		}
	}
	if value != "" && !inList {
		opts = append([]widget.SelectOption{{Label: value, Value: value}}, opts...)
	}
	return widget.NewSelect(opts).WithValue(value).WithWidth(settingsCtlW).WithOnChanged(onChange)
}

// thinkingSelect 思考模式下拉（关闭 / 开启推荐 / 最大推理深度）。
func thinkingSelect(value string, onChange func(string)) widget.Widget {
	return widget.NewSelect([]widget.SelectOption{
		{Label: "关闭", Value: "non-thinking"},
		{Label: "开启（推荐）", Value: "thinking"},
		{Label: "最大推理深度", Value: "thinking_max"},
	}).WithValue(value).WithWidth(settingsCtlW).WithOnChanged(onChange)
}

func parseTempOr(s string, def float64) float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return v
	}
	return def
}

func fmtContext(n int) string {
	if n >= 1000000 {
		return itoa(n/1000000) + "M"
	}
	if n >= 1000 {
		return itoa(n/1000) + "K"
	}
	return itoa(n)
}
