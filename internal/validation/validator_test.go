package validation

import (
	"errors"
	"math"
	"testing"

	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// ─────────────────────────────────────────────────────────────
// ValidationContext 测试
// ─────────────────────────────────────────────────────────────

func TestNewValidationContext(t *testing.T) {
	constraints := layout.Tight(100, 200)
	size := types.Size{Width: 100, Height: 200}
	vc := NewValidationContext(constraints, size)

	if vc.Constraints != constraints {
		t.Errorf("NewValidationContext Constraints = %v, want %v", vc.Constraints, constraints)
	}
	if vc.Size != size {
		t.Errorf("NewValidationContext Size = %v, want %v", vc.Size, size)
	}
	if vc.Errors != nil {
		t.Errorf("NewValidationContext Errors = %v, want nil", vc.Errors)
	}
}

func TestValidationContext_AddError(t *testing.T) {
	vc := NewValidationContext(layout.Unbounded(), types.Size{Width: 0, Height: 0})

	if vc.HasErrors() {
		t.Error("HasErrors() = true, want false before adding errors")
	}
	if vc.ErrorCount() != 0 {
		t.Errorf("ErrorCount = %d, want 0", vc.ErrorCount())
	}

	vc.AddError(errors.New("error 1"))
	if !vc.HasErrors() {
		t.Error("HasErrors() = false, want true after adding error")
	}
	if vc.ErrorCount() != 1 {
		t.Errorf("ErrorCount = %d, want 1", vc.ErrorCount())
	}

	vc.AddError(errors.New("error 2"))
	if vc.ErrorCount() != 2 {
		t.Errorf("ErrorCount = %d, want 2", vc.ErrorCount())
	}

	// 添加 nil 不应增加错误计数
	vc.AddError(nil)
	if vc.ErrorCount() != 2 {
		t.Errorf("ErrorCount = %d, want 2 (nil should not be added)", vc.ErrorCount())
	}
}

func TestValidationContext_ErrorStrings(t *testing.T) {
	vc := NewValidationContext(layout.Unbounded(), types.Size{})
	vc.AddError(errors.New("err a"))
	vc.AddError(errors.New("err b"))

	strs := vc.ErrorStrings()
	if len(strs) != 2 {
		t.Fatalf("ErrorStrings length = %d, want 2", len(strs))
	}
	if strs[0] != "err a" {
		t.Errorf("ErrorStrings[0] = %q, want %q", strs[0], "err a")
	}
	if strs[1] != "err b" {
		t.Errorf("ErrorStrings[1] = %q, want %q", strs[1], "err b")
	}
}

// ─────────────────────────────────────────────────────────────
// CheckNonNilChild 测试
// ─────────────────────────────────────────────────────────────

func TestCheckNonNilChild(t *testing.T) {
	t.Run("nil child returns error", func(t *testing.T) {
		err := CheckNonNilChild("Container.Child", nil)
		if err == nil {
			t.Fatal("CheckNonNilChild(nil) should return error")
		}
		if err.Error() != "Container.Child: child is nil, expected non-nil" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("non-nil child returns nil", func(t *testing.T) {
		child := &struct{}{}
		err := CheckNonNilChild("Test.Child", child)
		if err != nil {
			t.Errorf("CheckNonNilChild(non-nil) should return nil, got %v", err)
		}
	})

	t.Run("nil pointer with non-nil interface", func(t *testing.T) {
		// typed nil: 一个 nil 指针被赋给 interface 后，interface 本身非 nil
		type childType struct{ val int }
		var typedNil *childType = nil
		err := CheckNonNilChild("TypedNil", typedNil)
		if err != nil {
			t.Errorf("typed nil child should pass (interface non-nil), got %v", err)
		}
	})

	t.Run("string value", func(t *testing.T) {
		var s string = "hello"
		err := CheckNonNilChild("StringChild", s)
		if err != nil {
			t.Errorf("string value should pass, got %v", err)
		}
	})

	t.Run("int value", func(t *testing.T) {
		err := CheckNonNilChild("IntChild", 42)
		if err != nil {
			t.Errorf("int value should pass, got %v", err)
		}
	})
}

// ─────────────────────────────────────────────────────────────
// CheckSizePositive 测试
// ─────────────────────────────────────────────────────────────

func TestCheckSizePositive(t *testing.T) {
	t.Run("positive size OK", func(t *testing.T) {
		err := CheckSizePositive("Widget.Size", types.Size{Width: 100, Height: 200})
		if err != nil {
			t.Errorf("positive size should pass, got %v", err)
		}
	})

	t.Run("zero width", func(t *testing.T) {
		err := CheckSizePositive("Widget.Size", types.Size{Width: 0, Height: 200})
		if err == nil {
			t.Fatal("zero width should return error")
		}
		if err.Error() != "Widget.Size: width must be positive (>0), got 0" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("zero height", func(t *testing.T) {
		err := CheckSizePositive("Widget.Size", types.Size{Width: 100, Height: 0})
		if err == nil {
			t.Fatal("zero height should return error")
		}
		if err.Error() != "Widget.Size: height must be positive (>0), got 0" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("negative width", func(t *testing.T) {
		err := CheckSizePositive("Widget.Size", types.Size{Width: -1, Height: 200})
		if err == nil {
			t.Fatal("negative width should return error")
		}
	})

	t.Run("negative height", func(t *testing.T) {
		err := CheckSizePositive("Widget.Size", types.Size{Width: 100, Height: -5})
		if err == nil {
			t.Fatal("negative height should return error")
		}
	})

	t.Run("infinite width", func(t *testing.T) {
		err := CheckSizePositive("Widget.Size", types.Size{Width: math.Inf(1), Height: 200})
		if err == nil {
			t.Fatal("infinite width should return error")
		}
	})

	t.Run("NaN height", func(t *testing.T) {
		err := CheckSizePositive("Widget.Size", types.Size{Width: 100, Height: math.NaN()})
		if err == nil {
			t.Fatal("NaN height should return error")
		}
	})
}

// ─────────────────────────────────────────────────────────────
// CheckSizeNonNegative 测试
// ─────────────────────────────────────────────────────────────

func TestCheckSizeNonNegative(t *testing.T) {
	t.Run("zero size OK", func(t *testing.T) {
		err := CheckSizeNonNegative("Widget.Size", types.Size{Width: 0, Height: 0})
		if err != nil {
			t.Errorf("zero size should pass, got %v", err)
		}
	})

	t.Run("positive size OK", func(t *testing.T) {
		err := CheckSizeNonNegative("Widget.Size", types.Size{Width: 100, Height: 200})
		if err != nil {
			t.Errorf("positive size should pass, got %v", err)
		}
	})

	t.Run("negative width", func(t *testing.T) {
		err := CheckSizeNonNegative("Widget.Size", types.Size{Width: -1, Height: 0})
		if err == nil {
			t.Fatal("negative width should return error")
		}
	})

	t.Run("negative height", func(t *testing.T) {
		err := CheckSizeNonNegative("Widget.Size", types.Size{Width: 0, Height: -1})
		if err == nil {
			t.Fatal("negative height should return error")
		}
	})
}

// ─────────────────────────────────────────────────────────────
// Validatable 接口实现验证
// ─────────────────────────────────────────────────────────────

// testValidatable 实现 Validatable 接口，用于测试
type testValidatable struct {
	shouldFail bool
}

func (tv *testValidatable) Validate(ctx *ValidationContext) error {
	if tv.shouldFail {
		return errors.New("validation failed")
	}
	// 使用 CheckSizePositive 和 CheckNonNilChild 进行验证
	if err := CheckSizePositive("testValidatable", ctx.Size); err != nil {
		ctx.AddError(err)
		return err
	}
	if err := CheckNonNilChild("testValidatable.child", tv); err != nil {
		ctx.AddError(err)
		return err
	}
	return nil
}

func TestValidatableInterface(t *testing.T) {
	t.Run("Validatable passes validation", func(t *testing.T) {
		v := &testValidatable{shouldFail: false}
		vc := NewValidationContext(layout.Unbounded(), types.Size{Width: 100, Height: 50})

		err := v.Validate(&vc)
		if err != nil {
			t.Errorf("Validate should pass, got %v", err)
		}
		if vc.HasErrors() {
			t.Errorf("HasErrors = true, want false; errors: %v", vc.Errors)
		}
	})

	t.Run("Validatable fails with negative size", func(t *testing.T) {
		v := &testValidatable{shouldFail: false}
		vc := NewValidationContext(layout.Unbounded(), types.Size{Width: -1, Height: 50})

		err := v.Validate(&vc)
		if err == nil {
			t.Fatal("Validate should fail with negative width")
		}
		if !vc.HasErrors() {
			t.Error("HasErrors = false, want true")
		}
	})

	t.Run("Validatable fails when shouldFail", func(t *testing.T) {
		v := &testValidatable{shouldFail: true}
		vc := NewValidationContext(layout.Unbounded(), types.Size{Width: 100, Height: 50})

		err := v.Validate(&vc)
		if err == nil {
			t.Fatal("Validate should fail when shouldFail=true")
		}
		if err.Error() != "validation failed" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ─────────────────────────────────────────────────────────────
// Benchmark
// ─────────────────────────────────────────────────────────────

func BenchmarkValidationContext(b *testing.B) {
	vc := NewValidationContext(layout.Unbounded(), types.Size{Width: 100, Height: 200})
	v := &testValidatable{shouldFail: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vc.Errors = vc.Errors[:0]
		_ = v.Validate(&vc)
	}
}

func BenchmarkCheckSizePositive(b *testing.B) {
	size := types.Size{Width: 100, Height: 200}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CheckSizePositive("bench", size)
	}
}
