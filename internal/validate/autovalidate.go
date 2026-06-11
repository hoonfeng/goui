// Package validate 提供组件验证套件（Validation Suite）。
//
// 本包定义了 Suite 框架，用于统一管理组件验证的三个阶段：
//   Phase 1 — Build 结构验证：递归检查 Element 树的构建一致性
//   Phase 2 — 布局验证：执行 Layout 后检查尺寸合理性
//   Phase 3 — Element 一致性验证：调用各 Element 的 Validate()
//
// 使用方式：
//
//	s := validate.NewSuite()
//	s.Register("Container", buildCheck, layoutCheck, stateCheck)
//	report := s.RunAll(&validate.SuiteContext{
//	    Root: rootElement,
//	    Constraints: &constraint,
//	})
package validate

import (
	"fmt"
	"math"
	"strings"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/widget"
)

// ─────────────────────────────────────────────────────────────
// 核心类型
// ─────────────────────────────────────────────────────────────

// CheckFunc 验证单个 Element，返回错误字符串列表。
// 返回 nil 或空切片表示通过。
type CheckFunc func(el widget.Element) []string

// ComponentValidator 为特定组件类型注册三个阶段的验证函数。
type ComponentValidator struct {
	// WidgetType 是组件类型标识，如 "Container"、"Text"、"Button"。
	// 使用 "*" 表示匹配所有类型（全局检查）。
	WidgetType string

	// BuildCheck 对应 Phase 1 — Build 结构验证。
	// nil 表示此组件跳过该阶段。
	BuildCheck CheckFunc

	// LayoutCheck 对应 Phase 2 — 布局验证。
	// nil 表示此组件跳过该阶段。
	LayoutCheck CheckFunc

	// StateCheck 对应 Phase 3 — Element 一致性验证。
	// nil 表示此组件跳过该阶段。
	StateCheck CheckFunc
}

// Suite 验证套件，管理组件验证函数的注册和统一运行。
type Suite struct {
	validators []ComponentValidator
	scenes     []Scene
}

// Scene 定义单个验证场景。
// 每个场景是一个具名的 Widget 构建函数，ExecuteAll 会为其自动创建 Element 树并执行三阶段验证。
type Scene struct {
	// Name 是场景名称，用于报告和日志。
	Name string
	// Builder 是创建 Widget 树的函数。
	Builder func() widget.Widget
}

// ScenarioResult 记录单个场景的验证结果。
type ScenarioResult struct {
	// Name 是场景名称。
	Name string
	// Pass 表示该场景是否全部三个阶段都通过。
	Pass bool
	// Report 是该场景的详细验证报告。
	Report *SuiteReport
	// Error 记录场景执行过程中的严重错误（如 nil 返回值、panic 等）。
	Error string
}

// ExecuteAllReport 是 ExecuteAll 的汇总报告。
type ExecuteAllReport struct {
	// Total 是总场景数。
	Total int
	// Passed 是通过的场景数。
	Passed int
	// Failed 是失败的场景数。
	Failed int
	// Scenarios 是每个场景的详细结果。
	Scenarios []ScenarioResult
}

// ─────────────────────────────────────────────────────────────
// 结果类型
// ─────────────────────────────────────────────────────────────

// CheckItem 记录单个 Element 的检查结果。
type CheckItem struct {
	// Path 是 Element 在树中的路径描述。
	Path string
	// Errors 是发现的错误列表，为空表示通过。
	Errors []string
}

// SuiteContext 提供给 RunAll 的上下文。
type SuiteContext struct {
	// Root 是 Element 树的根节点。
	Root widget.Element
	// Constraints 是 Phase 2 执行 Layout 时使用的布局约束。
	// 如果为 nil，则跳过 Phase 2 的 Layout 执行步骤。
	Constraints *layout.BoxConstraints
}

// SuiteReport 是验证全过程的汇总报告。
type SuiteReport struct {
	// Phase 1 — Build 结构验证
	Phase1Pass   bool
	Phase1Total  int
	Phase1Failed int
	Phase1Items  []CheckItem

	// Phase 2 — 布局验证
	Phase2Pass   bool
	Phase2Total  int
	Phase2Failed int
	Phase2Items  []CheckItem

	// Phase 3 — Element 一致性验证
	Phase3Pass   bool
	Phase3Errors []string
}

// ─────────────────────────────────────────────────────────────
// Suite 方法
// ─────────────────────────────────────────────────────────────

// NewSuite 创建新的验证套件。
func NewSuite() *Suite {
	return &Suite{
		validators: make([]ComponentValidator, 0),
	}
}

// Register 注册一个组件类型的三个阶段验证函数。
//
//	widgetType：组件类型标识，如 "Container"、"Text"。
//	  使用 "*" 表示匹配所有类型（注册全局检查）。
//	buildCheck / layoutCheck / stateCheck：对应三个阶段的验证函数。
//	  任意一个可以为 nil（表示跳过该阶段）。
func (s *Suite) Register(widgetType string, buildCheck, layoutCheck, stateCheck CheckFunc) {
	// 去重：如果已存在同类型，替换
	for i, v := range s.validators {
		if v.WidgetType == widgetType {
			s.validators[i] = ComponentValidator{
				WidgetType:  widgetType,
				BuildCheck:  buildCheck,
				LayoutCheck: layoutCheck,
				StateCheck:  stateCheck,
			}
			return
		}
	}
	s.validators = append(s.validators, ComponentValidator{
		WidgetType:  widgetType,
		BuildCheck:  buildCheck,
		LayoutCheck: layoutCheck,
		StateCheck:  stateCheck,
	})
}

// RegisterScene 注册一个验证场景。
// name 是场景名称，builder 是创建 Widget 树的函数。
// ExecuteAll 会为每个注册的场景自动创建 Element 树、构建、布局并执行三阶段验证。
func (s *Suite) RegisterScene(name string, builder func() widget.Widget) {
	s.scenes = append(s.scenes, Scene{Name: name, Builder: builder})
}

// Scenes 返回所有已注册的场景列表。
// 用于外部遍历场景以执行额外的验证（如视觉验证）。
func (s *Suite) Scenes() []Scene {
	scenes := make([]Scene, len(s.scenes))
	copy(scenes, s.scenes)
	return scenes
}

// ExecuteAll 遍历所有已注册场景，对每个场景自动执行完整的验证流程：
//
//	1. 调用场景的 Builder 创建 Widget
//	2. 通过 CreateElementFor 创建根 Element
//	3. 挂载并递归 Build 构建 Element 树
//	4. 使用标准约束 800x600 执行 Layout
//	5. 运行三阶段验证（Build 结构检查 → 布局验证 → Element 一致性验证）
//
// 返回汇总报告，包含每个场景的通过/失败信息和详细错误。
func (s *Suite) ExecuteAll() *ExecuteAllReport {
	report := &ExecuteAllReport{
		Scenarios: make([]ScenarioResult, 0, len(s.scenes)),
	}
	constraints := layout.Tight(800, 600)

	for _, scene := range s.scenes {
		result := s.executeScene(scene, &constraints)
		report.Scenarios = append(report.Scenarios, result)
		report.Total++
		if result.Pass {
			report.Passed++
		} else {
			report.Failed++
		}
	}
	return report
}

// executeScene 执行单个场景的完整验证流程。
func (s *Suite) executeScene(scene Scene, constraints *layout.BoxConstraints) ScenarioResult {
	result := ScenarioResult{Name: scene.Name}

	// 1. 构建 Widget
	w := scene.Builder()
	if w == nil {
		result.Error = "Builder returned nil Widget"
		return result
	}

	// 2. 创建 Element
	root := widget.CreateElementFor(w)
	if root == nil {
		result.Error = "CreateElementFor returned nil"
		return result
	}

	// 3. 挂载并递归构建 Element 树
	root.Mount(nil, 0)
	BuildElementTree(root)

	// 4. 执行三阶段验证
	ctx := &SuiteContext{
		Root:        root,
		Constraints: constraints,
	}
	rep := s.RunAll(ctx)
	result.Report = rep
	result.Pass = rep.Phase1Pass && rep.Phase2Pass && rep.Phase3Pass

	return result
}

// RunAll 在 ctx 上运行全部三个阶段，返回汇总报告。
//
// 执行流程：
//  1. Phase 1 — 遍历 Element 树，对每个 Element 匹配注册的 BuildCheck 并执行。
//  2. Phase 2 — 先对根节点执行 Layout（若 Constraints 非 nil），
//     然后遍历 Element 树匹配注册的 LayoutCheck 并执行。
//  3. Phase 3 — 调用 widget.ValidateElementTree 收集所有 Element 的 Validate() 错误。
func (s *Suite) RunAll(ctx *SuiteContext) *SuiteReport {
	rep := &SuiteReport{}

	// Phase 1: Build 结构验证
	rep.Phase1Pass, rep.Phase1Total, rep.Phase1Failed, rep.Phase1Items = s.runPhase1(ctx.Root)

	// Phase 2: 布局验证（先执行 Layout）
	if ctx.Constraints != nil {
		ctx.Root.Layout(&layout.LayoutContext{Constraints: *ctx.Constraints})
	}
	rep.Phase2Pass, rep.Phase2Total, rep.Phase2Failed, rep.Phase2Items = s.runPhase2(ctx.Root)

	// Phase 3: Element 一致性验证
	rep.Phase3Errors = s.runPhase3(ctx.Root)
	rep.Phase3Pass = len(rep.Phase3Errors) == 0

	return rep
}

// ─────────────────────────────────────────────────────────────
// 内部：阶段执行
// ─────────────────────────────────────────────────────────────

// findValidator 查找匹配 widgetType 的验证器。
// 优先精确匹配，其次匹配 "*"（全局）。
func (s *Suite) findValidator(widgetType string) *ComponentValidator {
	for i := range s.validators {
		if s.validators[i].WidgetType == widgetType {
			return &s.validators[i]
		}
	}
	// 回退到全局匹配
	for i := range s.validators {
		if s.validators[i].WidgetType == "*" {
			return &s.validators[i]
		}
	}
	return nil
}

// elementTypeName 返回 Element 关联的 Widget 类型名（不含包路径）。
func elementTypeName(el widget.Element) string {
	s := fmt.Sprintf("%T", el.Widget())
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.TrimPrefix(s, "*")
	return s
}

// elementShortName 返回 Element 的简短类型名。
func elementShortName(el widget.Element) string {
	s := fmt.Sprintf("%T", el)
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.TrimPrefix(s, "*")
	if len(s) > 28 {
		s = s[:28] + ".."
	}
	return s
}

func (s *Suite) runPhase1(root widget.Element) (pass bool, total, failed int, items []CheckItem) {
	items = make([]CheckItem, 0)
	total = 0
	failed = 0

	var traverse func(el widget.Element, path string)
	traverse = func(el widget.Element, path string) {
		if el == nil {
			return
		}

		typeName := elementTypeName(el)
		currentPath := elementShortName(el) + "@" + pathSuffix(path)

		// 查找匹配的验证器
		validator := s.findValidator(typeName)

		var errs []string
		if validator != nil && validator.BuildCheck != nil {
			errs = validator.BuildCheck(el)
		}

		total++
		item := CheckItem{Path: currentPath}
		if len(errs) > 0 {
			failed++
			item.Errors = errs
		}
		items = append(items, item)

		newPath := currentPath
		for i, child := range el.Children() {
			if child != nil {
				traverse(child, fmt.Sprintf("%s[%d]", newPath, i))
			}
		}
	}

	traverse(root, "")

	pass = failed == 0
	return
}

func (s *Suite) runPhase2(root widget.Element) (pass bool, total, failed int, items []CheckItem) {
	items = make([]CheckItem, 0)
	total = 0
	failed = 0

	const infThreshold = float64(1 << 30)

	var traverse func(el widget.Element, depth int)
	traverse = func(el widget.Element, depth int) {
		if el == nil {
			return
		}

		typeName := elementTypeName(el)
		validator := s.findValidator(typeName)

		var errs []string
		if validator != nil && validator.LayoutCheck != nil {
			errs = validator.LayoutCheck(el)
		} else {
			// 默认全局检查：尺寸合理性
			sz := el.Size()
			if sz.Width < 0 || sz.Height < 0 {
				errs = append(errs, fmt.Sprintf("尺寸为负数: %v", sz))
			}
			if sz.Width >= infThreshold || sz.Height >= infThreshold {
				errs = append(errs, fmt.Sprintf("尺寸为 INF: %v", sz))
			}
			if sz.Width != sz.Width || sz.Height != sz.Height {
				errs = append(errs, fmt.Sprintf("尺寸为 NaN: %v", sz))
			}
		}

		total++
		item := CheckItem{Path: fmt.Sprintf("%s[depth=%d]", elementShortName(el), depth)}
		if len(errs) > 0 {
			failed++
			item.Errors = errs
		}
		items = append(items, item)

		for _, child := range el.Children() {
			if child != nil {
				traverse(child, depth+1)
			}
		}
	}

	traverse(root, 0)

	pass = failed == 0
	return
}

func (s *Suite) runPhase3(root widget.Element) []string {
	// 使用 widget 包内建的 ValidateElementTree
	validateErrs := widget.ValidateElementTree(root)
	if len(validateErrs) == 0 {
		return nil
	}
	errStrs := make([]string, len(validateErrs))
	for i, err := range validateErrs {
		errStrs[i] = err.Error()
	}
	return errStrs
}

// ─────────────────────────────────────────────────────────────
// 工具函数
// ─────────────────────────────────────────────────────────────

func pathSuffix(p string) string {
	if p == "" {
		return "root"
	}
	parts := strings.Split(p, "@")
	if len(parts) < 2 {
		return p
	}
	return parts[len(parts)-1]
}

// ShortType 返回 Element 类型的简短描述（用于日志/报告）。
func ShortType(el widget.Element) string {
	return elementShortName(el)
}

// ─────────────────────────────────────────────────────────────
// 内置 CheckFunc 工厂
// ─────────────────────────────────────────────────────────────

// BuildCheckStandard 返回标准的 Build 结构检查函数。
// 检查项：
//   - Widget() 非 nil
//   - 所有子 Element 的 Parent() 指向自身
//   - 无 nil 子 Element
func BuildCheckStandard() CheckFunc {
	return func(el widget.Element) []string {
		var errs []string
		if el.Widget() == nil {
			errs = append(errs, "Widget() 为 nil")
		}
		for i, child := range el.Children() {
			if child == nil {
				errs = append(errs, fmt.Sprintf("Children()[%d] 为 nil", i))
				continue
			}
			if child.Parent() != el {
				errs = append(errs, fmt.Sprintf("Children()[%d].Parent() 不指向自身", i))
			}
		}
		return errs
	}
}

// LayoutCheckStandard 返回标准的布局尺寸检查函数。
// 检查项：
//   - 宽高非负
//   - 宽高非 INF
//   - 宽高非 NaN
func LayoutCheckStandard() CheckFunc {
	return func(el widget.Element) []string {
		var errs []string
		sz := el.Size()
		if sz.Width < 0 || sz.Height < 0 {
			errs = append(errs, fmt.Sprintf("尺寸为负数: %v", sz))
		}
		if math.IsInf(sz.Width, 0) || math.IsInf(sz.Height, 0) {
			errs = append(errs, fmt.Sprintf("尺寸为 INF: %v", sz))
		}
		if math.IsNaN(sz.Width) || math.IsNaN(sz.Height) {
			errs = append(errs, fmt.Sprintf("尺寸为 NaN: %v", sz))
		}
		return errs
	}
}

// ─────────────────────────────────────────────────────────────
// 辅助：类型统计
// ─────────────────────────────────────────────────────────────

// CountTypes 统计树中各 Widget 类型出现次数。
func CountTypes(el widget.Element) map[string]int {
	counts := make(map[string]int)
	var traverse func(el widget.Element)
	traverse = func(el widget.Element) {
		if el == nil {
			return
		}
		typeName := elementTypeName(el)
		counts[typeName]++
		for _, child := range el.Children() {
			if child != nil {
				traverse(child)
			}
		}
	}
	traverse(el)
	return counts
}

// BuildElementTree 递归构建 Element 树（调用每个节点的 Build）。
func BuildElementTree(el widget.Element) {
	children := el.Build()
	for _, child := range children {
		if child != nil {
			BuildElementTree(child)
		}
	}
}

// DebugPrintTree 返回 Element 树的文本表示。
func DebugPrintTree(el widget.Element) string {
	var b strings.Builder
	var traverse func(el widget.Element, indent string)
	traverse = func(el widget.Element, indent string) {
		if el == nil {
			return
		}
		typeName := elementTypeName(el)
		sz := el.Size()
		b.WriteString(fmt.Sprintf("%s%s sz=%v\n", indent, typeName, sz))
		for _, child := range el.Children() {
			if child != nil {
				traverse(child, indent+"  ")
			}
		}
	}
	traverse(el, "")
	return b.String()
}
