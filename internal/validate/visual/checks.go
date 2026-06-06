package visual

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

// ─────────────────────────────────────────────────────────────
// VisualCheck 定义
// ─────────────────────────────────────────────────────────────

// VisualCheck 是一个视觉检查项，包含名称和检查函数。
type VisualCheck struct {
	// Name 是检查项的名称，用于报告和日志。
	Name string
	// Check 是检查函数，接收 VisualFrame，返回检查错误列表。
	Check func(frame *VisualFrame) []VisualError
}

// VisualError 记录单个视觉检查错误。
type VisualError struct {
	// Type 是错误类型（如 "color-mismatch", "blank", "region-not-found"）。
	Type string
	// Message 是人类可读的错误描述。
	Message string
	// Rect 是错误发生的矩形区域（可选）。
	Rect image.Rectangle
	// Expected 是预期值（可选）。
	Expected interface{}
	// Actual 是实际值（可选）。
	Actual interface{}
}

// Error 实现 error 接口。
func (ve VisualError) Error() string {
	s := fmt.Sprintf("[%s] %s", ve.Type, ve.Message)
	if ve.Rect != image.ZR {
		s += fmt.Sprintf(" (rect: %v)", ve.Rect)
	}
	return s
}

// ─────────────────────────────────────────────────────────────
// VisualSuite — 视觉验证套件
// ─────────────────────────────────────────────────────────────

// VisualSuite 管理一组视觉检查项，统一执行并生成报告。
// 用法：
//
//	suite := visual.NewVisualSuite()
//	suite.AddCheck("非空白", visual.CheckNonBlank())
//	suite.AddCheck("按钮存在", visual.CheckRectHasNonWhite(
//	    image.Rect(0, 0, 100, 50), 10))
//	report := suite.Run(frame)
type VisualSuite struct {
	checks []VisualCheck
}

// NewVisualSuite 创建空的视觉验证套件。
func NewVisualSuite() *VisualSuite {
	return &VisualSuite{
		checks: make([]VisualCheck, 0),
	}
}

// AddCheck 添加一个视觉检查项。
func (vs *VisualSuite) AddCheck(name string, checkFunc func(frame *VisualFrame) []VisualError) {
	vs.checks = append(vs.checks, VisualCheck{
		Name:  name,
		Check: checkFunc,
	})
}

// Run 对所有已注册的检查项执行验证，返回报告。
func (vs *VisualSuite) Run(frame *VisualFrame) *VisualReport {
	report := &VisualReport{
		FrameInfo: fmt.Sprintf("%dx%d", frame.Width, frame.Height),
		Checks:    make([]CheckResult, 0, len(vs.checks)),
	}

	for _, vc := range vs.checks {
		errs := vc.Check(frame)
		passed := len(errs) == 0
		result := CheckResult{
			Name:   vc.Name,
			Passed: passed,
			Errors: errs,
		}
		report.Checks = append(report.Checks, result)
		if passed {
			report.Passed++
		} else {
			report.Failed++
		}
		report.Total++
	}

	report.AllPassed = report.Failed == 0
	return report
}

// ─────────────────────────────────────────────────────────────
// 内置检查函数
// ─────────────────────────────────────────────────────────────

// CheckNonBlank 检查画布是否非空白（有非白色像素）。
// 用于验证组件确实被绘制到画布上。
func CheckNonBlank() func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		pa := frame.Analyze()
		if pa.IsBlank() {
			return []VisualError{{
				Type:    "blank",
				Message: "Canvas is completely blank (all white pixels)",
			}}
		}
		return nil
	}
}

// CheckNonWhiteRatio 检查画布上非白色像素的比例是否超过阈值。
// minRatio 范围 0.0~1.0，表示非白色像素占总像素的最小比例。
// 例如 minRatio=0.05 表示至少 5% 的像素是非白色的。
func CheckNonWhiteRatio(minRatio float64) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		pa := frame.Analyze()
		totalPixels := frame.Width * frame.Height
		nonWhite := pa.CountNonWhitePixels(frame.Image.Bounds())
		ratio := float64(nonWhite) / float64(totalPixels)

		if ratio < minRatio {
			return []VisualError{{
				Type:     "low-content-ratio",
				Message:  fmt.Sprintf("Non-white pixel ratio %.1f%% is below minimum %.1f%%", ratio*100, minRatio*100),
				Expected: fmt.Sprintf("≥%.1f%%", minRatio*100),
				Actual:   fmt.Sprintf("%.1f%%", ratio*100),
			}}
		}
		return nil
	}
}

// CheckRectColor 检查指定矩形区域是否包含目标颜色至少 minCount 个像素。
// 用于验证特定组件是否在预期区域出现（如按钮的主色）。
func CheckRectColor(rect image.Rectangle, target color.RGBA, tolerance uint8, minCount int) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		pa := frame.Analyze()
		count := pa.CountColorInRect(target, tolerance, rect)

		if count < minCount {
			return []VisualError{{
				Type:     "color-not-found",
				Message:  fmt.Sprintf("Expected at least %d pixels of color %v in rect %v, found %d", minCount, target, rect, count),
				Rect:     rect,
				Expected: minCount,
				Actual:   count,
			}}
		}
		return nil
	}
}

// CheckColorAt 检查画布上指定坐标的颜色是否与预期相似。
func CheckColorAt(x, y int, expected color.RGBA, tolerance uint8) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		actual := frame.PixelAt(x, y)
		if !colorSimilar(actual, expected, tolerance) {
			return []VisualError{{
				Type:     "color-mismatch",
				Message:  fmt.Sprintf("Color at (%d,%d): expected %v, got %v", x, y, expected, actual),
				Rect:     image.Rect(x, y, x+1, y+1),
				Expected: expected,
				Actual:   actual,
			}}
		}
		return nil
	}
}

// CheckRectHasNonWhite 检查指定矩形区域是否有至少 minNonWhitePixels 个非白色像素。
// 用于验证组件是否在预期区域内绘制了内容。
func CheckRectHasNonWhite(rect image.Rectangle, minNonWhitePixels int) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		pa := frame.Analyze()
		total := rect.Dx() * rect.Dy()
		whiteCount := pa.CountColorInRect(
			color.RGBA{R: 255, G: 255, B: 255, A: 255},
			0, rect,
		)
		nonWhite := total - whiteCount

		if nonWhite < minNonWhitePixels {
			return []VisualError{{
				Type:     "insufficient-content",
				Message:  fmt.Sprintf("Rect %v has %d non-white pixels, expected at least %d", rect, nonWhite, minNonWhitePixels),
				Rect:     rect,
				Expected: minNonWhitePixels,
				Actual:   nonWhite,
			}}
		}
		return nil
	}
}

// CheckPixelCount 统计全画布上指定颜色的像素数量是否在 [min, max] 范围内。
// 用于验证颜色用量（如确认没有意外的颜色污染）。
func CheckPixelCount(target color.RGBA, tolerance uint8, min, max int) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		pa := frame.Analyze()
		count := pa.CountColor(target, tolerance)

		if count < min || count > max {
			return []VisualError{{
				Type:     "pixel-count-out-of-range",
				Message:  fmt.Sprintf("Color %v pixel count: %d (expected range [%d, %d])", target, count, min, max),
				Expected: fmt.Sprintf("[%d, %d]", min, max),
				Actual:   count,
			}}
		}
		return nil
	}
}

// CheckRegionExists 检查画布上是否存在非白色的连通区域（组件）。
// 通过查找颜色聚类判断是否有可见的绘制内容。适用于验证组件整体存在性。
func CheckRegionExists(minArea int) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		pa := frame.Analyze()
		// 查找所有非白色聚类
		// 使用一个宽松的容差来涵盖抗锯齿和渐变
		nonWhite := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		clusters := pa.FindColorClusters(nonWhite, 10, minArea)

		if len(clusters) == 0 {
			return []VisualError{{
				Type:    "no-regions",
				Message: fmt.Sprintf("No visible component regions found (minArea=%d)", minArea),
			}}
		}
		return nil
	}
}

// CheckHorizontalLine 检查指定水平扫描线的颜色。
// 用于验证分割线、背景填充等水平延伸的组件。
func CheckHorizontalLine(y int, expected color.RGBA, tolerance uint8) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		pa := frame.Analyze()
		if !pa.HorizontalLineColor(y, expected, tolerance) {
			return []VisualError{{
				Type:    "horizontal-line-mismatch",
				Message: fmt.Sprintf("Horizontal line at y=%d does not match color %v", y, expected),
				Rect:    image.Rect(0, y, frame.Width, y+1),
			}}
		}
		return nil
	}
}

// CheckVerticalLine 检查指定垂直扫描线的颜色。
func CheckVerticalLine(x int, expected color.RGBA, tolerance uint8) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		pa := frame.Analyze()
		if !pa.VerticalLineColor(x, expected, tolerance) {
			return []VisualError{{
				Type:    "vertical-line-mismatch",
				Message: fmt.Sprintf("Vertical line at x=%d does not match color %v", x, expected),
				Rect:    image.Rect(x, 0, x+1, frame.Height),
			}}
		}
		return nil
	}
}

// ─────────────────────────────────────────────────────────────
// 组合检查
// ─────────────────────────────────────────────────────────────

// ComposeChecks 将多个检查函数组合为一个。
func ComposeChecks(checks ...func(frame *VisualFrame) []VisualError) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		var all []VisualError
		for _, check := range checks {
			errs := check(frame)
			if len(errs) > 0 {
				all = append(all, errs...)
			}
		}
		return all
	}
}

// MustPass 包装一个检查函数，如果检查失败则 panic（用于测试断言）。
func MustPass(checkFunc func(frame *VisualFrame) []VisualError) func(frame *VisualFrame) []VisualError {
	return func(frame *VisualFrame) []VisualError {
		errs := checkFunc(frame)
		if len(errs) > 0 {
			var msgs []string
			for _, e := range errs {
				msgs = append(msgs, e.Error())
			}
			panic(fmt.Sprintf("Visual check MUST PASS but failed:\n%s", strings.Join(msgs, "\n")))
		}
		return nil
	}
}
