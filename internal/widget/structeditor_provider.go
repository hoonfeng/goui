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

func init() {
	RegisterProvider(goLangProvider{}, "golang")
	RegisterProvider(eyLangProvider{}, "易语言", "estr")
}
