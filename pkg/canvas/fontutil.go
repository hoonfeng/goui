package canvas

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gofont "golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// 共享字体缓存（按字号+字重缓存，全局唯一，减少重复加载）
var globalFontCache = make(map[string]gofont.Face)

// isMonoFamily 识别等宽字体族。终端/代码渲染需要真·等宽（每字同宽），
// 而 goui 默认主字体是阿里普惠体（比例字体）——用比例字体画终端网格会出现「全角」似的稀疏间距。
// 命中这些族名时，Skia 渲染改用真等宽字体（Consolas / 系统等宽）。
func isMonoFamily(family string) bool {
	switch strings.ToLower(strings.TrimSpace(family)) {
	case "consolas", "courier", "courier new", "monospace", "mono",
		"menlo", "monaco", "cascadia code", "cascadia mono", "dejavu sans mono", "sf mono":
		return true
	}
	return false
}

// cacheKey 生成字体缓存键
func cacheKey(size float64, weight FontWeight) string {
	return fmt.Sprintf("%.1f-%d", size, weight)
}

// GetFontFace 获取指定尺寸和字重的系统默认字体 face，支持中文等多语言文字。
// weight 参数决定加载哪种字重的字体文件：
//   - FontWeightBold (700) → 优先加载 Bold 字体文件
//   - 其他 → 按 Regular → Medium → Bold 顺序回退
//
// 项目 fonts/ 目录优先于系统字体，跨平台提供一致的阿里普惠体渲染。
func GetFontFace(size float64, weight FontWeight) gofont.Face {
	key := cacheKey(size, weight)
	if f, ok := globalFontCache[key]; ok {
		return f
	}

	face := loadSystemFont(size, weight)
	if face == nil {
		// 字重优先级回退：尝试 Normal 字重的缓存
		if weight != FontWeightNormal {
			normalKey := cacheKey(size, FontWeightNormal)
			if f, ok := globalFontCache[normalKey]; ok {
				globalFontCache[key] = f
				return f
			}
		}
		face = basicfont.Face7x13
	}
	globalFontCache[key] = face
	return face
}

// getProjectFontDir 获取项目 fonts 目录路径。
// 优先使用相对于可执行文件的路径，其次使用相对于当前工作目录的路径。
func getProjectFontDir() string {
	// 尝试相对于可执行文件路径
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, "fonts")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	// 尝试相对于当前工作目录
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "fonts")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

// loadSystemFont 尝试从系统加载字体文件。
// 优先从项目 fonts/ 目录加载阿里普惠体（Alibaba PuHuiTi），回退到系统字体。
// weight 参数决定优先加载哪种字重的字体文件。
func loadSystemFont(size float64, weight FontWeight) gofont.Face {
	// 1. 优先加载项目本地阿里普惠体字体（跨平台一致体验，支持中文）
	//    根据不同字重设置不同的搜索优先级
	if fontDir := getProjectFontDir(); fontDir != "" {
		var projectFonts []string
		switch weight {
		case FontWeightBold:
			// Bold: 优先粗体文件，回退到 Medium → Regular
			projectFonts = []string{
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-85-Bold.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-65-Medium.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-55-Regular.ttf"),
			}
		case FontWeightMedium:
			// Medium: 优先中等字重
			projectFonts = []string{
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-65-Medium.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-55-Regular.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-85-Bold.ttf"),
			}
		default: // Normal / 其他
			// 阿里普惠体 55-Regular 偏细，视觉上 Medium(65) 更接近标准 Regular
			// 因此将 Medium(65) 放在 Regular(55) 前面优先加载，使默认文字更清晰厚重
			projectFonts = []string{
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-65-Medium.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-55-Regular.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-85-Bold.ttf"),
			}
		}
		for _, p := range projectFonts {
			if _, err := os.Stat(p); err == nil {
				if f, err := loadFontFromFile(p, size); err == nil {
					return f
				}
			}
		}
	}

	// 2. 回退到系统路径
	paths := []string{
		// Windows 中文优先（注：扩展名为 .ttc 表示 TrueType Collection，需特殊处理）
		"C:/Windows/Fonts/msyh.ttc",
		"C:/Windows/Fonts/msyhbd.ttc",
		"C:/Windows/Fonts/msyhl.ttc",
		"C:/Windows/Fonts/simhei.ttf",
		"C:/Windows/Fonts/simsun.ttc",
		"C:/Windows/Fonts/SimsunExtG.ttf",
		"C:/Windows/Fonts/STFANGSO.TTF",
		"C:/Windows/Fonts/STKAITI.TTF",
		// Windows 通用
		"C:/Windows/Fonts/segoeui.ttf",
		"C:/Windows/Fonts/arial.ttf",
		// macOS
		"/System/Library/Fonts/PingFang.ttc",
		"/System/Library/Fonts/Helvetica.ttc",
		"/System/Library/Fonts/Helvetica.ttf",
		// Linux
		"/usr/share/fonts/truetype/wqy/wqy-microhei.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			if f, err := loadFontFromFile(p, size); err == nil {
				return f
			}
		}
	}
	return nil
}

// loadFontFromFile 从字体文件加载指定尺寸的 face。
// 自动检测 TrueType Collection (.ttc/.otc) 文件并使用 sfnt.ParseCollection，
// 单个字体文件 (.ttf/.otf) 使用 sfnt.Parse。
func loadFontFromFile(path string, size float64) (gofont.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ft *sfnt.Font

	// 检测是否为 TTC/OTC Collection 文件（以 ttcf 魔术字开头）
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".ttc" || ext == ".otc" || (len(data) >= 4 && string(data[0:4]) == "ttcf") {
		// 解析字体集合，取第一个字体
		collection, err := sfnt.ParseCollection(data)
		if err != nil {
			return nil, err
		}
		ft, err = collection.Font(0)
		if err != nil {
			return nil, err
		}
	} else {
		ft, err = sfnt.Parse(data)
		if err != nil {
			return nil, err
		}
	}

	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    size,
		DPI:     96,
		Hinting: gofont.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	return face, nil
}

// sharedMeasureOnce/sharedMeasureInst 维护一个全局共享、仅用于测量的 1x1 画布。
var (
	sharedMeasureOnce sync.Once
	sharedMeasureInst *SkiaCanvas
)

// sharedMeasureCanvas 返回全局共享、仅用于文本测量的 1x1 SkiaCanvas。
// 让 Layout 阶段（拿不到 Canvas 实例）的文本测量与实际 Skia 渲染同源，
// 避免 gofont 与 Skia 字宽（尤其中文）不一致导致组件尺寸/换行/定位错误。
func sharedMeasureCanvas() *SkiaCanvas {
	sharedMeasureOnce.Do(func() {
		sharedMeasureInst = NewSkiaCanvas(1, 1)
	})
	return sharedMeasureInst
}

// MeasureTextGlobal 测量文本在指定字体下的精确尺寸。
// 不依赖外部 Canvas 实例，可在 Layout 阶段直接调用；
// 内部委托全局共享的 Skia 测量画布，确保与渲染同源。
func MeasureTextGlobal(text string, font Font) TextMetrics {
	if text == "" {
		return TextMetrics{}
	}
	return sharedMeasureCanvas().MeasureText(text, font)
}

// faceVMetricsCache 按字号缓存真实 Skia 垂直度量（{ascent, descent, lineHeight}），
// 避免每次取度量都走一遍 CGO。
var (
	faceVMetricsMu    sync.Mutex
	faceVMetricsCache = map[float64][3]float64{}
)

// faceVMetrics 返回指定字号（默认字重）的**真实** Skia 垂直度量（来自 sk_font_get_metrics）：
// ascent（基线到顶，正）、descent（基线到底，正）、lineHeight（推荐行距）。按字号缓存。
// 全面取代 size×0.8 / size×0.2 / size×1.3 的私有经验估算——度量只用 Skia 自己的计算。
func faceVMetrics(size float64) (ascent, descent, lineHeight float64) {
	if size <= 0 {
		size = 16
	}
	faceVMetricsMu.Lock()
	if m, ok := faceVMetricsCache[size]; ok {
		faceVMetricsMu.Unlock()
		return m[0], m[1], m[2]
	}
	faceVMetricsMu.Unlock()

	a, d, lh := sharedMeasureCanvas().fontVMetrics(Font{Size: size, Weight: FontWeightNormal})

	faceVMetricsMu.Lock()
	faceVMetricsCache[size] = [3]float64{a, d, lh}
	faceVMetricsMu.Unlock()
	return a, d, lh
}

// GetFaceAscent 返回指定字号文字基线到顶部的距离（来自 Skia 真实度量）。
func GetFaceAscent(size float64) float64 { a, _, _ := faceVMetrics(size); return a }

// GetFaceDescent 返回指定字号文字基线到底部的距离（正值，来自 Skia 真实度量）。
func GetFaceDescent(size float64) float64 { _, d, _ := faceVMetrics(size); return d }

// GetFaceLineHeight 返回指定字号的推荐行高（来自 Skia 真实度量），用于多行文本布局。
func GetFaceLineHeight(size float64) float64 { _, _, lh := faceVMetrics(size); return lh }

// ── 主字体字形覆盖检测（用于绘制时的字体回退）──

var (
	coverageMu    sync.Mutex
	coverageCache = map[rune]bool{}
)

// RuneCoveredByPrimary 返回主字体（项目内阿里普惠体）是否包含该字符的字形。
// 绘制时据此回退：主字体缺失的字符（阿拉伯/emoji 等）改用系统 fallback 字体。
func RuneCoveredByPrimary(r rune) bool {
	if r < 0x80 { // ASCII 一定覆盖
		return true
	}
	coverageMu.Lock()
	defer coverageMu.Unlock()
	if v, ok := coverageCache[r]; ok {
		return v
	}
	// 复用 Skia 全局共享的主 Typeface 做覆盖检测（unichar→glyph，非 0 即含该字形），
	// 不再单独用 gofont 解析一份字库。
	tf := sharedMeasureCanvas().getOrCreateTypefaceForCoverage()
	if tf == nil {
		coverageCache[r] = true // 无法检测时假设覆盖，不触发回退
		return true
	}
	covered := tf.UnicharToGlyph(r) != 0
	coverageCache[r] = covered
	return covered
}

// FallbackFamily 按字符所属脚本返回一个系统 fallback 字体族名。
// 返回空串表示交给 Skia 默认字体处理。
func FallbackFamily(r rune) string {
	switch {
	case (r >= 0x0600 && r <= 0x06FF) || (r >= 0x0750 && r <= 0x077F) ||
		(r >= 0x08A0 && r <= 0x08FF) || (r >= 0xFB50 && r <= 0xFDFF) ||
		(r >= 0xFE70 && r <= 0xFEFF): // 阿拉伯
		return "Tahoma"
	case r >= 0x0590 && r <= 0x05FF: // 希伯来
		return "Tahoma"
	case (r >= 0x1F000 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) ||
		(r >= 0x1F300 && r <= 0x1F9FF): // emoji
		return "Segoe UI Emoji"
	case (r >= 0x2190 && r <= 0x21FF) || // 箭头
		(r >= 0x2300 && r <= 0x23FF) || // 技术符号(⎘⌨⏏ 等)
		(r >= 0x25A0 && r <= 0x25FF) || // 几何图形(▲▼◀▶▸▾ 等 UI 三角；阿里普惠体只含部分)
		(r >= 0x2B00 && r <= 0x2BFF): // 杂项符号与箭头
		return "Segoe UI Symbol"
	default:
		return "" // 交给 Skia 默认字体（系统）
	}
}
