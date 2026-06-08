package widget

import "strings"

// LanguageProvider 语言适配器：把表格框架彻底「去语言化」。框架只认这个接口，不写死 Go。
// 接新语言 = 注册一个 provider：① Schema 给表结构(纯配置) ② Generate 表格→代码(多为模板)
// ③ Parse 代码→表格(语言特定核心，简单语言可套通用解析器，复杂语言如 Go 复用 go/parser)。
type LanguageProvider interface {
	Name() string                          // 语言名（注册键，不区分大小写）
	Schema() *SESchema                     // 表结构（列定义）；nil→框架兜底 DefaultSchema()
	Parse(src string) (*SEProgram, error)  // 代码文本 → 表格模型
	Generate(p *SEProgram) string          // 表格模型 → 代码文本
}

var languageProviders = map[string]LanguageProvider{}

// RegisterProvider 注册（或覆盖）一种语言的表格化适配器。这是「接新语言」的对外入口。
// aliases 是额外的注册名（如 "golang" 之于 "go"）。
func RegisterProvider(p LanguageProvider, aliases ...string) {
	if p == nil {
		return
	}
	languageProviders[strings.ToLower(p.Name())] = p
	for _, a := range aliases {
		languageProviders[strings.ToLower(a)] = p
	}
}

// providerFor 取某语言的适配器（未注册→nil，调用方自行兜底）。
func providerFor(lang string) LanguageProvider {
	if p, ok := languageProviders[strings.ToLower(lang)]; ok {
		return p
	}
	return nil
}

// HasProvider 检查某语言是否已注册结构化编辑器适配器（供 companion 判断用哪种编辑器）。
func HasProvider(lang string) bool {
	return providerFor(lang) != nil
}

// ── 内置适配器：Go / 易语言（把现有 ParseGo/ToGo、ParseProgram/Serialize 包装成 provider）──

type goLangProvider struct{}

func (goLangProvider) Name() string                         { return "go" }
func (goLangProvider) Schema() *SESchema                    { return DefaultSchema() }
func (goLangProvider) Parse(src string) (*SEProgram, error) { return ParseGo(src) }
func (goLangProvider) Generate(p *SEProgram) string         { return p.ToGo() }

type eyLangProvider struct{}

func (eyLangProvider) Name() string                         { return "ey" }
func (eyLangProvider) Schema() *SESchema                    { return DefaultSchema() }
func (eyLangProvider) Parse(src string) (*SEProgram, error) { return ParseProgram(src), nil }
func (eyLangProvider) Generate(p *SEProgram) string         { return p.Serialize() }

// ── 自定义语言提供者（从配置文件加载） ──

// CustomProviderConfig 用户自定义语言提供者配置。
// 用于在不修改 Go 源码的情况下，通过 settings.json 将新语言映射到已有 provider。
type CustomProviderConfig struct {
	Name        string     `json:"name"`        // 语言名（如 "vue"、"kotlin"）
	Extensions  []string   `json:"extensions"`  // 文件扩展名列表（不含点，如 ["vue"]）
	UseProvider string     `json:"useProvider"` // 重用已有 provider 的名称（如 "js"、"go"）
	Schema      *SESchema  `json:"schema,omitempty"` // 可选：覆盖表结构列定义
}

// customProvider 包装一个已有 provider，可换名并可选覆盖 schema。
type customProvider struct {
	base   LanguageProvider
	name   string
	schema *SESchema
}

func (c *customProvider) Name() string                         { return c.name }
func (c *customProvider) Schema() *SESchema                    { return c.schema }
func (c *customProvider) Parse(src string) (*SEProgram, error) { return c.base.Parse(src) }
func (c *customProvider) Generate(p *SEProgram) string         { return c.base.Generate(p) }

// LoadCustomProviders 从配置加载自定义语言提供者，注册到全局注册表。
// 每个配置项把 UseProvider 指向的已有 provider 以新名称注册，并可覆盖表结构。
// configs 为空切片或 nil 时无操作。
func LoadCustomProviders(configs []CustomProviderConfig) {
	for _, cp := range configs {
		if cp.Name == "" || cp.UseProvider == "" {
			continue
		}
		base := providerFor(cp.UseProvider)
		if base == nil {
			continue
		}
		// 如果自定义项没提供 schema，继承 base 的 schema
		sch := cp.Schema
		if sch == nil {
			sch = base.Schema()
		}
		p := &customProvider{base: base, name: cp.Name, schema: sch}
		RegisterProvider(p, cp.Extensions...)
	}
}

func init() {
	RegisterProvider(goLangProvider{}, "golang")
	RegisterProvider(eyLangProvider{}, "易语言", "estr")
}
