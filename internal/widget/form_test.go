package widget

import (
	"testing"

	"github.com/user/goui/internal/types"
)

// TestValidateRequired 必填：纯 required 用 Message，混合规则空值报默认。
func TestValidateRequired(t *testing.T) {
	if msg := validateValue("", "用户名", []FormRule{{Required: true}}); msg != "用户名不能为空" {
		t.Errorf("required 空=%q want 用户名不能为空", msg)
	}
	if msg := validateValue("", "X", []FormRule{{Required: true, Message: "请输入"}}); msg != "请输入" {
		t.Errorf("纯 required Message=%q want 请输入", msg)
	}
	// 混合 required+pattern：空值报默认必填，不借 pattern 的 Message
	if msg := validateValue("", "邮箱", []FormRule{{Required: true, Pattern: "@", Message: "格式错"}}); msg != "邮箱不能为空" {
		t.Errorf("混合 required 空=%q want 邮箱不能为空", msg)
	}
	if msg := validateValue("abc", "X", []FormRule{{Required: true}}); msg != "" {
		t.Errorf("required 非空应通过=%q", msg)
	}
}

// TestValidatePattern 正则。
func TestValidatePattern(t *testing.T) {
	rules := []FormRule{{Required: true, Pattern: `^[\w.\-]+@[\w.\-]+\.\w+$`, Message: "邮箱格式不正确"}}
	if msg := validateValue("notmail", "邮箱", rules); msg != "邮箱格式不正确" {
		t.Errorf("非法邮箱=%q want 邮箱格式不正确", msg)
	}
	if msg := validateValue("a@b.com", "邮箱", rules); msg != "" {
		t.Errorf("合法邮箱应通过=%q", msg)
	}
}

// TestValidateMinLen 最小长度。
func TestValidateMinLen(t *testing.T) {
	rules := []FormRule{{Required: true, MinLen: 6, Message: "密码至少 6 位"}}
	if msg := validateValue("123", "密码", rules); msg != "密码至少 6 位" {
		t.Errorf("短密码=%q want 密码至少 6 位", msg)
	}
	if msg := validateValue("123456", "密码", rules); msg != "" {
		t.Errorf("足够长应通过=%q", msg)
	}
}

// TestValidateOptional 非必填空值通过；MaxLen 超长报错。
func TestValidateOptional(t *testing.T) {
	if msg := validateValue("", "昵称", []FormRule{{MaxLen: 5}}); msg != "" {
		t.Errorf("非必填空应通过=%q", msg)
	}
	if msg := validateValue("toolongname", "昵称", []FormRule{{MaxLen: 5}}); msg == "" {
		t.Error("超长应报错")
	}
}

// TestFormItemValidate FormItemElement 校验设 errorMsg + 红边框，通过后恢复。
func TestFormItemValidate(t *testing.T) {
	item := &FormItem{Label: "用户名", Required: true}
	e := item.CreateElement().(*FormItemElement)
	e.Build() // 创建内部 Input

	e.value = ""
	if e.validate() {
		t.Error("空必填应校验失败")
	}
	if e.errorMsg != "用户名不能为空" {
		t.Errorf("errorMsg=%q", e.errorMsg)
	}
	red := types.ColorFromRGB(245, 108, 108)
	if e.input.BorderColor != red {
		t.Error("校验失败应设红边框")
	}

	e.value = "alice"
	if !e.validate() {
		t.Error("非空应通过")
	}
	if e.errorMsg != "" {
		t.Errorf("通过后 errorMsg 应空=%q", e.errorMsg)
	}
	if e.input.BorderColor == red {
		t.Error("通过后应恢复非红边框")
	}
}
