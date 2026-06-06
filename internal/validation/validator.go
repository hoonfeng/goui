// Package validation 提供通用的验证框架。
//
// 本包定义了 Validatable 接口和 ValidationContext，用于在 goui 框架中
// 对控件、布局、尺寸等进行一致性验证。与 internal/widget/validate.go 中
// 的 Widget 专用验证不同，本包提供的是跨模块可复用的通用验证基础设施。
package validation

import (
	"fmt"
	"math"

	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// ─────────────────────────────────────────────────────────────
// ValidationContext
// ─────────────────────────────────────────────────────────────

// ValidationContext 携带验证所需的上下文信息。
// 包含当前的布局约束、尺寸值以及验证过程中累积的错误列表。
type ValidationContext struct {
	// Constraints 是当前验证对象的布局约束。
	Constraints layout.BoxConstraints

	// Size 是当前验证对象的尺寸。
	Size types.Size

	// Errors 是验证过程中累积的错误列表。
	Errors []error
}

// AddError 向上下文中添加一个验证错误。
func (vc *ValidationContext) AddError(err error) {
	if err != nil {
		vc.Errors = append(vc.Errors, err)
	}
}

// HasErrors 返回是否已记录任何验证错误。
func (vc *ValidationContext) HasErrors() bool {
	return len(vc.Errors) > 0
}

// ErrorCount 返回已记录的错误数量。
func (vc *ValidationContext) ErrorCount() int {
	return len(vc.Errors)
}

// ErrorStrings 返回所有错误的字符串表示。
func (vc *ValidationContext) ErrorStrings() []string {
	strs := make([]string, len(vc.Errors))
	for i, err := range vc.Errors {
		strs[i] = err.Error()
	}
	return strs
}

// NewValidationContext 创建新的验证上下文，包含指定的约束和尺寸。
func NewValidationContext(constraints layout.BoxConstraints, size types.Size) ValidationContext {
	return ValidationContext{
		Constraints: constraints,
		Size:        size,
		Errors:      nil,
	}
}

// ─────────────────────────────────────────────────────────────
// Validatable 接口
// ─────────────────────────────────────────────────────────────

// Validatable 表示可自我验证的对象。
// Validate 方法接收验证上下文，检查对象状态一致性，
// 通过上下文的 AddError 方法记录错误，并返回首个严重错误（若有）。
type Validatable interface {
	Validate(ctx *ValidationContext) error
}

// ─────────────────────────────────────────────────────────────
// 基础验证工具函数
// ─────────────────────────────────────────────────────────────

// CheckNonNilChild 验证子对象不为 nil。
// name 参数用于标识被验证的对象（如 "Container.Child"）。
// child 是被检查的接口值。
func CheckNonNilChild(name string, child interface{}) error {
	if child == nil {
		return fmt.Errorf("%s: child is nil, expected non-nil", name)
	}
	return nil
}

// CheckSizePositive 验证尺寸的宽和高都为正数（> 0）。
// name 参数用于标识被验证的对象。
// size 是被检查的尺寸值。
// 当宽或高为 0、负数、正无穷、NaN 时返回错误。
func CheckSizePositive(name string, size types.Size) error {
	if size.Width <= 0 {
		return fmt.Errorf("%s: width must be positive (>0), got %v", name, size.Width)
	}
	if size.Height <= 0 {
		return fmt.Errorf("%s: height must be positive (>0), got %v", name, size.Height)
	}
	if math.IsInf(size.Width, 0) || math.IsNaN(size.Width) {
		return fmt.Errorf("%s: width is invalid (Inf/NaN), got %v", name, size.Width)
	}
	if math.IsInf(size.Height, 0) || math.IsNaN(size.Height) {
		return fmt.Errorf("%s: height is invalid (Inf/NaN), got %v", name, size.Height)
	}
	return nil
}

// CheckSizeNonNegative 验证尺寸的宽和高都非负（>= 0）。
// name 参数用于标识被验证的对象。
// size 是被检查的尺寸值。
// 此函数允许零尺寸，适用于组件初始状态或尚未分配尺寸的场景。
func CheckSizeNonNegative(name string, size types.Size) error {
	if size.Width < 0 || math.IsInf(size.Width, 0) || math.IsNaN(size.Width) {
		return fmt.Errorf("%s: width must be non-negative and finite, got %v", name, size.Width)
	}
	if size.Height < 0 || math.IsInf(size.Height, 0) || math.IsNaN(size.Height) {
		return fmt.Errorf("%s: height must be non-negative and finite, got %v", name, size.Height)
	}
	return nil
}
