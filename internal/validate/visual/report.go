package visual

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────
// 验证报告类型
// ─────────────────────────────────────────────────────────────

// VisualReport 是视觉验证的汇总报告。
// 由 VisualSuite.Run() 生成，包含所有检查项的通过/失败信息。
type VisualReport struct {
	// FrameInfo 描述被检查的帧信息（如 "800x600"）。
	FrameInfo string
	// Total 是检查项总数。
	Total int
	// Passed 是通过的检查项数量。
	Passed int
	// Failed 是失败的检查项数量。
	Failed int
	// AllPassed 表示是否所有检查项都通过。
	AllPassed bool
	// Checks 是每个检查项的详细结果。
	Checks []CheckResult
}

// CheckResult 记录单个检查项的执行结果。
type CheckResult struct {
	// Name 是检查项名称。
	Name string
	// Passed 表示此检查是否通过。
	Passed bool
	// Errors 是检查发现的错误列表（为空表示通过）。
	Errors []VisualError
}

// Summary 返回单行摘要字符串。
func (r *VisualReport) Summary() string {
	if r.AllPassed {
		return fmt.Sprintf("✅ ALL PASS — %d/%d checks passed (%s)",
			r.Passed, r.Total, r.FrameInfo)
	}
	return fmt.Sprintf("❌ FAIL — %d/%d checks passed, %d failed (%s)",
		r.Passed, r.Total, r.Failed, r.FrameInfo)
}

// String 返回报告的多行文本表示。
func (r *VisualReport) String() string {
	var b strings.Builder

	b.WriteString("\n════════════════════════════════════════════════\n")
	b.WriteString("  Visual Validation Report\n")
	b.WriteString("════════════════════════════════════════════════\n")
	b.WriteString(fmt.Sprintf("  Frame: %s\n", r.FrameInfo))
	b.WriteString(fmt.Sprintf("  Total: %d | Passed: %d | Failed: %d\n",
		r.Total, r.Passed, r.Failed))
	b.WriteString("────────────────────────────────────────────\n")

	for _, cr := range r.Checks {
		if cr.Passed {
			b.WriteString(fmt.Sprintf("  ✅ PASS  %s\n", cr.Name))
		} else {
			b.WriteString(fmt.Sprintf("  ❌ FAIL  %s\n", cr.Name))
			for _, err := range cr.Errors {
				b.WriteString(fmt.Sprintf("       └─ %s\n", err.Error()))
			}
		}
	}

	b.WriteString("────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  %s\n", r.Summary()))
	b.WriteString("════════════════════════════════════════════════\n")

	return b.String()
}
