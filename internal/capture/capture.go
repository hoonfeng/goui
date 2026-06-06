// Package capture 提供截图捕获工具（Screenshot Capture Utility）。
//
// 本包用于捕获窗口客户区的像素内容，返回 *image.RGBA 格式的图片数据，
// 可直接与 internal/validate/visual 包配合使用，对实际渲染的窗口内容
// 进行像素级分析。
//
// 支持使用 OpenGL glReadPixels 捕获渲染表面。
//
// 典型使用流程：
//
//	// 使用 OpenGL 捕获（需要在 GL 线程中调用）
//	c := capture.NewGL(width, height)
//	img, err := c.Capture()
//
//	// 分析：转换为 VisualFrame 后使用 PixelAnalyzer
//	vf := &visual.VisualFrame{Image: img, Width: img.Bounds().Dx(), Height: img.Bounds().Dy()}
//	pa := vf.Analyze()
//	if pa.IsBlank() { t.Error("window is blank") }
//
//	// 或直接保存为 PNG
//	path, err := capture.SaveImage(img, "screenshots/", "capture")
package capture

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

// Capturer 是截图捕获器的抽象接口。
type Capturer interface {
	// Capture 捕获完整客户区或渲染表面的截图。
	// 返回 RGBA 格式的图像数据。
	Capture() (*image.RGBA, error)

	// CaptureRect 捕获指定矩形区域的截图。
	// rect 相对于被捕获区域的左上角 (0,0)。
	CaptureRect(rect image.Rectangle) (*image.RGBA, error)

	// Size 返回被捕获区域的尺寸（像素）。
	Size() (width, height int)
}

// ─────────────────────────────────────────────────────────────
// 工厂函数
// ─────────────────────────────────────────────────────────────

// NewGL 创建一个使用 OpenGL glReadPixels 的截图捕获器。
// 必须在 OpenGL 上下文当前的线程中调用 Capture()。
// width 和 height 指定视口尺寸（像素）。
func NewGL(width, height int) Capturer {
	return newGLCapturer(width, height)
}

// ─────────────────────────────────────────────────────────────
// 工具函数
// ─────────────────────────────────────────────────────────────

// CaptureResult 封装一次截图操作的结果。
type CaptureResult struct {
	Image     *image.RGBA // 捕获的像素数据
	Width     int         // 图像宽度
	Height    int         // 图像高度
	Timestamp time.Time   // 捕获时间
}

// NewCaptureResult 从 *image.RGBA 创建 CaptureResult。
func NewCaptureResult(img *image.RGBA) *CaptureResult {
	bounds := img.Bounds()
	return &CaptureResult{
		Image:     img,
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
		Timestamp: time.Now(),
	}
}

// SaveImage 将 RGBA 图像保存为 PNG 文件。
// 参数：
//   - img: 要保存的图像数据
//   - dir: 输出目录（自动创建）
//   - name: 文件名（不含扩展名），为空时自动生成时间戳文件名
//
// 返回保存的完整文件路径。
func SaveImage(img *image.RGBA, dir, name string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("capture: cannot create directory %s: %w", dir, err)
	}

	if name == "" {
		name = fmt.Sprintf("capture_%s", time.Now().Format("20060102_150405.000"))
	}

	filename := filepath.Join(dir, name+".png")
	f, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("capture: cannot create file %s: %w", filename, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return "", fmt.Errorf("capture: PNG encode failed: %w", err)
	}

	return filename, nil
}

// SaveResult 将 CaptureResult 保存为 PNG 文件。
func SaveResult(cr *CaptureResult, dir, name string) (string, error) {
	return SaveImage(cr.Image, dir, name)
}

// Compare 比较两个 RGBA 图像，返回差异信息。
// tolerance 是逐通道容差 (0-255)，0 表示精确匹配。
func Compare(got, want *image.RGBA, tolerance uint8) *DiffReport {
	bounds := got.Bounds()
	wantBounds := want.Bounds()

	cmpW := bounds.Dx()
	if wantBounds.Dx() < cmpW {
		cmpW = wantBounds.Dx()
	}
	cmpH := bounds.Dy()
	if wantBounds.Dy() < cmpH {
		cmpH = wantBounds.Dy()
	}

	totalPixels := cmpW * cmpH
	if totalPixels == 0 {
		return &DiffReport{
			Passed:      false,
			Message:     "empty image",
			DiffPercent: 100,
		}
	}

	diffPixels := 0
	maxDiff := 0.0

	for y := 0; y < cmpH; y++ {
		for x := 0; x < cmpW; x++ {
			gc := got.RGBAAt(x, y)
			wc := want.RGBAAt(x, y)

			dr := absDiff(gc.R, wc.R)
			dg := absDiff(gc.G, wc.G)
			db := absDiff(gc.B, wc.B)
			da := absDiff(gc.A, wc.A)

			if dr > tolerance || dg > tolerance || db > tolerance || da > tolerance {
				diffPixels++
				d := float64(dr)
				if float64(dg) > d {
					d = float64(dg)
				}
				if float64(db) > d {
					d = float64(db)
				}
				if float64(da) > d {
					d = float64(da)
				}
				if d > maxDiff {
					maxDiff = d
				}
			}
		}
	}

	diffPercent := float64(diffPixels) / float64(totalPixels) * 100
	passed := diffPercent < 5.0 // 默认 5% 差异以内通过

	return &DiffReport{
		Passed:      passed,
		DiffPixels:  diffPixels,
		TotalPixels: totalPixels,
		DiffPercent: diffPercent,
		MaxDiff:     maxDiff,
		Message:     fmt.Sprintf("%d/%d pixels differ (%.2f%%)", diffPixels, totalPixels, diffPercent),
	}
}

// DiffReport 记录两帧图像的比较结果。
type DiffReport struct {
	Passed      bool    // 是否通过比较
	DiffPixels  int     // 差异像素数
	TotalPixels int     // 总像素数
	DiffPercent float64 // 差异百分比 (0-100)
	MaxDiff     float64 // 最大通道差异 (0-255)
	Message     string  // 描述信息
}

// ─────────────────────────────────────────────────────────────
// 内部辅助
// ─────────────────────────────────────────────────────────────

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
