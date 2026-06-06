package visual

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// ─────────────────────────────────────────────────────────────
// 快照管理
// ─────────────────────────────────────────────────────────────

// SnapshotManager 管理黄金快照（Golden Snapshots）的保存和比较。
// 黄金快照是"预期正确"的渲染结果，用于回归测试中与当前渲染结果
// 进行逐像素对比，检测渲染变化。
//
// 目录结构：
//
//	testdata/
//	  snapshots/
//	    golden/
//	      Button.png          # 黄金快照
//	      Container.png
//	    diff/
//	      Button_diff.png     # 差异标记图
type SnapshotManager struct {
	// GoldenDir 存放黄金快照的目录。
	GoldenDir string
	// DiffDir 存放差异图的目录（可选）。
	DiffDir string
	// SaveOnFailure 当比较失败时是否自动保存差异图。
	SaveOnFailure bool
}

// NewSnapshotManager 创建快照管理器。
// goldenDir 是黄金快照存储目录，diffDir 是差异图存储目录（为空则不保存差异图）。
func NewSnapshotManager(goldenDir, diffDir string) *SnapshotManager {
	return &SnapshotManager{
		GoldenDir:     goldenDir,
		DiffDir:       diffDir,
		SaveOnFailure: true,
	}
}

// EnsureDirs 确保黄金和差异目录存在。
func (sm *SnapshotManager) EnsureDirs() error {
	if err := os.MkdirAll(sm.GoldenDir, 0755); err != nil {
		return fmt.Errorf("visual: cannot create golden dir %s: %w", sm.GoldenDir, err)
	}
	if sm.DiffDir != "" {
		if err := os.MkdirAll(sm.DiffDir, 0755); err != nil {
			return fmt.Errorf("visual: cannot create diff dir %s: %w", sm.DiffDir, err)
		}
	}
	return nil
}

// Save 将当前帧保存为 PNG 快照文件。
// name 是快照名称（不含扩展名），文件保存为 {name}.png。
func (sm *SnapshotManager) Save(name string, frame *VisualFrame) error {
	if err := sm.EnsureDirs(); err != nil {
		return err
	}

	filename := filepath.Join(sm.GoldenDir, name+".png")
	return SaveFrameToPNG(frame, filename)
}

// Compare 将当前帧与已保存的黄金快照进行比较。
// name 是快照名称（不含扩展名），tolerance 是逐像素容差 (0-255)。
// 返回比较是否通过、差异报告和错误。
//
// 行为：
//   - 如果黄金快照不存在，自动保存当前帧作为新黄金快照并返回 "通过"。
//   - 如果黄金快照存在，逐像素比较，超出容差的像素被视为差异。
func (sm *SnapshotManager) Compare(name string, frame *VisualFrame, tolerance uint8) (bool, *DiffReport, error) {
	if err := sm.EnsureDirs(); err != nil {
		return false, nil, err
	}

	goldenPath := filepath.Join(sm.GoldenDir, name+".png")

	// 检查黄金快照是否存在
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		// 不存在则自动创建
		if err := SaveFrameToPNG(frame, goldenPath); err != nil {
			return false, nil, fmt.Errorf("visual: cannot create golden snapshot %s: %w", goldenPath, err)
		}
		return true, &DiffReport{
			Status:  "created",
			Message: fmt.Sprintf("Golden snapshot created: %s", goldenPath),
		}, nil
	} else if err != nil {
		return false, nil, fmt.Errorf("visual: cannot stat golden snapshot %s: %w", goldenPath, err)
	}

	// 读取黄金快照
	goldenFile, err := os.Open(goldenPath)
	if err != nil {
		return false, nil, fmt.Errorf("visual: cannot open golden snapshot %s: %w", goldenPath, err)
	}
	defer goldenFile.Close()

	goldenImg, err := png.Decode(goldenFile)
	if err != nil {
		return false, nil, fmt.Errorf("visual: cannot decode golden snapshot %s: %w", goldenPath, err)
	}

	// 转换为 RGBA 以便逐像素比较
	goldenRGBA := toRGBA(goldenImg)

	// 执行比较
	diffPixels := make([]image.Point, 0)
	totalPixels := frame.Width * frame.Height
	maxDiff := 0.0

	bounds := goldenRGBA.Bounds()
	compareW := min(bounds.Dx(), frame.Width)
	compareH := min(bounds.Dy(), frame.Height)

	for y := 0; y < compareH; y++ {
		for x := 0; x < compareW; x++ {
			gc := goldenRGBA.RGBAAt(x, y)
			fc := frame.Image.RGBAAt(x, y)

			if !colorSimilar(gc, fc, tolerance) {
				diffPixels = append(diffPixels, image.Point{X: x, Y: y})
				// 计算最大通道差异
				dr := absDiffUint8(gc.R, fc.R)
				dg := absDiffUint8(gc.G, fc.G)
				db := absDiffUint8(gc.B, fc.B)
				da := absDiffUint8(gc.A, fc.A)
				d := float64(max(dr, dg, db, da))
				if d > maxDiff {
					maxDiff = d
				}
			}
		}
	}

	diffCount := len(diffPixels)
	passed := diffCount == 0
	similarity := 100.0
	if totalPixels > 0 {
		similarity = float64(totalPixels-diffCount) / float64(totalPixels) * 100.0
	}

	report := &DiffReport{
		Status:      "compared",
		Passed:      passed,
		DiffCount:   diffCount,
		TotalPixels: totalPixels,
		Similarity:  similarity,
		MaxDiff:     maxDiff,
		Message:     fmt.Sprintf("Compared with golden: %.1f%% similar, %d pixels differ", similarity, diffCount),
	}

	// 保存差异图
	if !passed && sm.SaveOnFailure && sm.DiffDir != "" {
		diffPath := filepath.Join(sm.DiffDir, name+"_diff.png")
		if err := saveDiffImage(goldenRGBA, frame.Image, diffPixels, diffPath); err != nil {
			return false, report, fmt.Errorf("visual: cannot save diff image: %w", err)
		}
		report.DiffImagePath = diffPath
		report.Message += fmt.Sprintf(", diff image saved: %s", diffPath)
	}

	return passed, report, nil
}

// ─────────────────────────────────────────────────────────────
// 帧保存工具
// ─────────────────────────────────────────────────────────────

// SaveFrameToPNG 将 VisualFrame 保存为 PNG 文件。
func SaveFrameToPNG(frame *VisualFrame, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("visual: cannot create directory %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("visual: cannot create file %s: %w", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, frame.Image); err != nil {
		return fmt.Errorf("visual: cannot encode PNG %s: %w", path, err)
	}

	return nil
}

// LoadFrameFromPNG 从 PNG 文件加载 VisualFrame。
func LoadFrameFromPNG(path string) (*VisualFrame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("visual: cannot open file %s: %w", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("visual: cannot decode PNG %s: %w", path, err)
	}

	rgba := toRGBA(img)
	bounds := rgba.Bounds()

	return &VisualFrame{
		Image:  rgba,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}, nil
}

// ─────────────────────────────────────────────────────────────
// 对比报告
// ─────────────────────────────────────────────────────────────

// DiffReport 记录两帧图像的比较结果。
type DiffReport struct {
	Status       string  // "created" | "compared"
	Passed       bool    // 是否通过比较
	DiffCount    int     // 差异像素数
	TotalPixels  int     // 总像素数
	Similarity   float64 // 相似度百分比 0-100
	MaxDiff      float64 // 最大通道差异 (0-255)
	Message      string  // 描述信息
	DiffImagePath string // 差异图路径（如有保存）
}

// ─────────────────────────────────────────────────────────────
// 内部辅助
// ─────────────────────────────────────────────────────────────

// toRGBA 将任意图像转换为 *image.RGBA。
func toRGBA(img image.Image) *image.RGBA {
	switch img := img.(type) {
	case *image.RGBA:
		return img
	default:
		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
		return rgba
	}
}

// saveDiffImage 生成差异图：将差异像素标记为红色半透明，背景为原图。
func saveDiffImage(golden, current *image.RGBA, diffPixels []image.Point, path string) error {
	bounds := golden.Bounds()
	diffImg := image.NewRGBA(bounds)

	// 复制当前帧作为底图
	copy(diffImg.Pix, current.Pix)

	// 标记差异像素为红色
	highlight := color.RGBA{R: 255, G: 0, B: 0, A: 180}
	for _, p := range diffPixels {
		if p.X >= bounds.Min.X && p.X < bounds.Max.X &&
			p.Y >= bounds.Min.Y && p.Y < bounds.Max.Y {
			// 混合红色标记
			base := diffImg.RGBAAt(p.X, p.Y)
			diffImg.Set(p.X, p.Y, blendColor(base, highlight))
		}
	}

	return savePNG(diffImg, path)
}

// blendColor 将 src 与 overlay 进行 alpha 混合。
func blendColor(base, overlay color.RGBA) color.RGBA {
	ao := uint32(overlay.A)
	ab := uint32(base.A)
	ia := 255 - ao

	outR := uint8((uint32(overlay.R)*ao + uint32(base.R)*ab*ia/255) / (ao + ab*ia/255))
	outG := uint8((uint32(overlay.G)*ao + uint32(base.G)*ab*ia/255) / (ao + ab*ia/255))
	outB := uint8((uint32(overlay.B)*ao + uint32(base.B)*ab*ia/255) / (ao + ab*ia/255))
	outA := uint8(ao + ab*ia/255)

	return color.RGBA{R: outR, G: outG, B: outB, A: outA}
}

// savePNG 将图像保存为 PNG。
func savePNG(img image.Image, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ─────────────────────────────────────────────────────────────
// 工具函数
// ─────────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxUint8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

// FormatPath 将路径中的反斜杠统一为正斜杠（跨平台）。
func FormatPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}
