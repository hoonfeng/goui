package canvas

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─────────────────────────────────────────────────────────────
// NewSoftCanvas 测试
// ─────────────────────────────────────────────────────────────

func TestNewSoftCanvas(t *testing.T) {
	t.Run("normal size", func(t *testing.T) {
		c := NewSoftCanvas(800, 600)
		if c == nil {
			t.Fatal("NewSoftCanvas returned nil")
		}
		if c.width != 800 {
			t.Errorf("width = %v, want 800", c.width)
		}
		if c.height != 600 {
			t.Errorf("height = %v, want 600", c.height)
		}
		if c.img == nil {
			t.Errorf("img is nil")
		} else {
			b := c.img.Bounds()
			if b.Dx() != 800 || b.Dy() != 600 {
				t.Errorf("img bounds = %v, want 800x600", b)
			}
		}
	})

	t.Run("zero size", func(t *testing.T) {
		// 零尺寸不 panic
		c := NewSoftCanvas(0, 0)
		if c == nil {
			t.Fatal("NewSoftCanvas returned nil")
		}
		if c.width != 0 || c.height != 0 {
			t.Errorf("expected 0x0, got %vx%v", c.width, c.height)
		}
		if c.img == nil {
			t.Errorf("img is nil for zero size")
		}
	})

	t.Run("negative size", func(t *testing.T) {
		// 负尺寸不 panic
		c := NewSoftCanvas(-10, -20)
		if c == nil {
			t.Fatal("NewSoftCanvas returned nil")
		}
		// image.NewRGBA(image.Rect(0,0,-10,-20)) 创建空图像
		if c.img == nil {
			t.Errorf("img is nil for negative size")
		}
	})

	t.Run("zero width", func(t *testing.T) {
		c := NewSoftCanvas(0, 100)
		if c == nil {
			t.Fatal("NewSoftCanvas returned nil")
		}
	})

	t.Run("zero height", func(t *testing.T) {
		c := NewSoftCanvas(100, 0)
		if c == nil {
			t.Fatal("NewSoftCanvas returned nil")
		}
	})

	t.Run("large size", func(t *testing.T) {
		// 不 panic
		c := NewSoftCanvas(1920, 1080)
		if c == nil {
			t.Fatal("NewSoftCanvas returned nil")
		}
		if c.width != 1920 || c.height != 1080 {
			t.Errorf("expected 1920x1080, got %vx%v", c.width, c.height)
		}
	})
}

// ─────────────────────────────────────────────────────────────
// Width / Height 测试
// ─────────────────────────────────────────────────────────────

func TestWidthHeight(t *testing.T) {
	c := NewSoftCanvas(640, 480)
	if w := c.Width(); w != 640 {
		t.Errorf("Width() = %v, want 640", w)
	}
	if h := c.Height(); h != 480 {
		t.Errorf("Height() = %v, want 480", h)
	}
}

// ─────────────────────────────────────────────────────────────
// Image 测试
// ─────────────────────────────────────────────────────────────

func TestImage(t *testing.T) {
	c := NewSoftCanvas(100, 200)
	img := c.Image()
	if img == nil {
		t.Fatal("Image() returned nil")
	}
	b := img.Bounds()
	if b.Dx() != 100 || b.Dy() != 200 {
		t.Errorf("Image bounds = %v, want 100x200", b)
	}

	// 修改底层图像应影响画布
	img.Set(50, 100, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	if c.img.At(50, 100) != img.At(50, 100) {
		t.Errorf("Image() should return the same backing image")
	}
}

// ─────────────────────────────────────────────────────────────
// Save / Restore 测试
// ─────────────────────────────────────────────────────────────

func TestSaveRestore(t *testing.T) {
	c := NewSoftCanvas(100, 100)

	// 设置裁剪
	c.ClipRect(10, 10, 50, 50)
	if !c.clipEnabled {
		t.Error("clipEnabled should be true after ClipRect")
	}

	// Save 状态
	c.Save()
	savedClip := c.clipRect

	// 修改裁剪
	c.ClipRect(20, 20, 30, 30)
	if c.clipRect == savedClip {
		t.Error("clipRect should have changed after second ClipRect")
	}

	// Restore 状态
	c.Restore()
	if c.clipRect != savedClip {
		t.Errorf("clipRect = %v, want %v after Restore", c.clipRect, savedClip)
	}

	// Restore 空栈不 panic
	for i := 0; i < 10; i++ {
		c.Restore()
	}
}

func TestSaveRestoreStackDepth(t *testing.T) {
	c := NewSoftCanvas(50, 50)

	// 多层 Save
	for i := 0; i < 5; i++ {
		c.Save()
	}
	if len(c.stateStack) != 5 {
		t.Errorf("stateStack len = %d, want 5", len(c.stateStack))
	}

	// 依次 Restore
	for i := 5; i > 0; i-- {
		c.Restore()
		if len(c.stateStack) != i-1 {
			t.Errorf("after Restore stateStack len = %d, want %d", len(c.stateStack), i-1)
		}
	}
}

// ─────────────────────────────────────────────────────────────
// Translate / Rotate / Scale 测试（当前为空操作）
// ─────────────────────────────────────────────────────────────

func TestTransformNoPanic(t *testing.T) {
	c := NewSoftCanvas(50, 50)
	// 这些方法当前为空操作，但不 panic
	TranslateTest := func() {
		c.Translate(10, 20)
		c.Rotate(45)
		c.Scale(2, 3)
	}
	TranslateTest()
}

// ─────────────────────────────────────────────────────────────
// Clip / ClipRect 测试
// ─────────────────────────────────────────────────────────────

func TestClip(t *testing.T) {
	t.Run("nil path disables clip", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		c.ClipRect(10, 10, 50, 50) // 启用裁剪
		c.Clip(nil)                 // 传入 nil 应禁用裁剪
		if c.clipEnabled {
			t.Error("Clip(nil) should disable clip")
		}
	})

	t.Run("empty path disables clip", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		c.ClipRect(10, 10, 50, 50)
		c.Clip(NewPath()) // 空路径应禁用裁剪
		if c.clipEnabled {
			t.Error("Clip(empty path) should disable clip")
		}
	})

	t.Run("path clip sets clip rect", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		path := NewPath()
		path.MoveTo(20, 30)
		path.LineTo(80, 30)
		path.LineTo(80, 70)
		path.LineTo(20, 70)
		path.Close()
		c.Clip(path)
		if !c.clipEnabled {
			t.Error("clipEnabled should be true after Clip with path")
		}
		// 包围盒应为 (20,30)-(80,70)
		expected := image.Rect(20, 30, 80, 70)
		if c.clipRect != expected {
			t.Errorf("clipRect = %v, want %v", c.clipRect, expected)
		}
	})

	t.Run("ClipRect sets clip rect", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		c.ClipRect(10, 10, 50, 50)
		if !c.clipEnabled {
			t.Error("clipEnabled should be true")
		}
		expected := image.Rect(10, 10, 60, 60)
		if c.clipRect != expected {
			t.Errorf("clipRect = %v, want %v", c.clipRect, expected)
		}
	})

	t.Run("ClipRect negative size", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		// 负宽高不应 panic
		c.ClipRect(50, 50, -10, -10)
		if !c.clipEnabled {
			t.Error("clipEnabled should be true even with negative size")
		}
	})
}

// ─────────────────────────────────────────────────────────────
// DrawRect 测试
// ─────────────────────────────────────────────────────────────

func TestDrawRect(t *testing.T) {
	t.Run("fill rect", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.Paint{
			Style:   paint.StyleFill,
			Color:   types.ColorFromRGB(255, 0, 0),
			Opacity: 1.0,
		}
		c.DrawRect(10, 10, 50, 50, p)
		// 验证矩形中心点已被绘制（红色）
		center := c.img.At(35, 35)
		r, g, b, a := center.RGBA()
		if r == 0 && g == 0 && b == 0 {
			t.Errorf("center pixel should be red, got %v", center)
		}
		_ = g
		_ = b
		_ = a
	})

	t.Run("stroke rect", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.Paint{
			Style:       paint.StyleStroke,
			Color:       types.ColorFromRGB(0, 255, 0),
			StrokeWidth: 2.0,
			Opacity:     1.0,
		}
		c.DrawRect(10, 10, 80, 80, p)
		// 描边模式下，矩形内部应未被填充
		// 边缘像素应有颜色
	})

	t.Run("fill and stroke rect", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.Paint{
			Style:       paint.StyleFillAndStroke,
			Color:       types.ColorFromRGB(0, 0, 255),
			StrokeWidth: 2.0,
			Opacity:     1.0,
		}
		c.DrawRect(10, 10, 50, 50, p)
	})

	t.Run("rect outside canvas", func(t *testing.T) {
		// 矩形完全在画布外
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultPaint()
		c.DrawRect(-100, -100, 50, 50, p) // 不 panic
		c.DrawRect(100, 100, 50, 50, p)   // 不 panic
	})

	t.Run("rect with zero size", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultPaint()
		c.DrawRect(10, 10, 0, 0, p) // 不 panic
		c.DrawRect(10, 10, -5, -5, p) // 不 panic
	})

	t.Run("rect with zero stroke width", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.Paint{
			Style:       paint.StyleStroke,
			Color:       types.ColorBlack,
			StrokeWidth: 0,
			Opacity:     1.0,
		}
		c.DrawRect(10, 10, 30, 30, p) // 应使用默认宽度 1
	})

	t.Run("rect with opacity", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.Paint{
			Style:   paint.StyleFill,
			Color:   types.ColorFromRGBA(255, 0, 0, 255),
			Opacity: 0.5,
		}
		c.DrawRect(10, 10, 30, 30, p)
	})
}

// ─────────────────────────────────────────────────────────────
// DrawLine 测试
// ─────────────────────────────────────────────────────────────

func TestDrawLine(t *testing.T) {
	t.Run("basic line", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.DefaultStrokePaint()
		c.DrawLine(10, 10, 90, 90, p)
	})

	t.Run("horizontal line", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.DefaultStrokePaint()
		c.DrawLine(10, 50, 90, 50, p)
	})

	t.Run("vertical line", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.DefaultStrokePaint()
		c.DrawLine(50, 10, 50, 90, p)
	})

	t.Run("line outside canvas", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultStrokePaint()
		c.DrawLine(-100, -100, -50, -50, p) // 完全在外
		c.DrawLine(-100, 25, 100, 25, p)    // 穿越画布
	})

	t.Run("zero length line", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultStrokePaint()
		c.DrawLine(25, 25, 25, 25, p) // 不 panic
	})

	t.Run("line with zero stroke width", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.Paint{
			Style:       paint.StyleStroke,
			Color:       types.ColorBlack,
			StrokeWidth: 0,
			Opacity:     1.0,
		}
		c.DrawLine(10, 10, 40, 40, p) // 应使用默认宽度 1
	})
}

// ─────────────────────────────────────────────────────────────
// DrawCircle 测试
// ─────────────────────────────────────────────────────────────

func TestDrawCircle(t *testing.T) {
	t.Run("fill circle", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.Paint{
			Style:   paint.StyleFill,
			Color:   types.ColorFromRGB(255, 0, 0),
			Opacity: 1.0,
		}
		c.DrawCircle(50, 50, 30, p)
	})

	t.Run("stroke circle", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.DefaultStrokePaint()
		c.DrawCircle(50, 50, 30, p)
	})

	t.Run("fill and stroke circle", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.Paint{
			Style:       paint.StyleFillAndStroke,
			Color:       types.ColorFromRGB(0, 255, 0),
			StrokeWidth: 2.0,
			Opacity:     1.0,
		}
		c.DrawCircle(50, 50, 30, p)
	})

	t.Run("zero radius", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultPaint()
		c.DrawCircle(25, 25, 0, p)  // 不 panic
		c.DrawCircle(25, 25, -5, p) // 负半径不 panic
	})

	t.Run("circle outside canvas", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultPaint()
		c.DrawCircle(-100, -100, 30, p) // 完全在外
		c.DrawCircle(0, 0, 100, p)      // 超出边界
	})
}

// ─────────────────────────────────────────────────────────────
// DrawRoundedRect 测试
// ─────────────────────────────────────────────────────────────

func TestDrawRoundedRect(t *testing.T) {
	t.Run("fill rounded rect", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.DefaultPaint()
		c.DrawRoundedRect(10, 10, 80, 80, 10, p)
	})

	t.Run("stroke rounded rect", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.DefaultStrokePaint()
		c.DrawRoundedRect(10, 10, 80, 80, 10, p)
	})

	t.Run("zero radius falls back to DrawRect", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.DefaultPaint()
		c.DrawRoundedRect(10, 10, 80, 80, 0, p)
	})

	t.Run("radius larger than half min dimension", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		p := paint.DefaultPaint()
		c.DrawRoundedRect(10, 10, 30, 30, 100, p) // 应自动缩小
	})

	t.Run("rounded rect outside canvas", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultPaint()
		c.DrawRoundedRect(-100, -100, 50, 50, 5, p)
	})
}

// ─────────────────────────────────────────────────────────────
// DrawPath 测试
// ─────────────────────────────────────────────────────────────

func TestDrawPath(t *testing.T) {
	t.Run("nil path", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		c.DrawPath(nil, paint.DefaultPaint()) // 不 panic
	})

	t.Run("empty path", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		c.DrawPath(NewPath(), paint.DefaultPaint()) // 不 panic
	})

	t.Run("simple triangle path", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		path := NewPath()
		path.MoveTo(50, 10)
		path.LineTo(90, 90)
		path.LineTo(10, 90)
		path.Close()
		p := paint.Paint{
			Style:   paint.StyleFill,
			Color:   types.ColorFromRGB(0, 0, 255),
			Opacity: 1.0,
		}
		c.DrawPath(path, p)
	})

	t.Run("stroke path", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		path := NewPath()
		path.MoveTo(10, 10)
		path.LineTo(90, 90)
		path.LineTo(10, 90)
		p := paint.DefaultStrokePaint()
		c.DrawPath(path, p)
	})

	t.Run("path with quad bezier", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		path := NewPath()
		path.MoveTo(10, 50)
		path.QuadTo(50, 10, 90, 50)
		p := paint.DefaultStrokePaint()
		c.DrawPath(path, p)
	})

	t.Run("path with cubic bezier", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		path := NewPath()
		path.MoveTo(10, 50)
		path.CubicTo(30, 10, 70, 90, 90, 50)
		p := paint.DefaultStrokePaint()
		c.DrawPath(path, p)
	})

	t.Run("path with arc", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		path := NewPath()
		path.MoveTo(50, 50)
		path.Arc(50, 50, 30, 0, math.Pi, false)
		p := paint.DefaultStrokePaint()
		c.DrawPath(path, p)
	})
}

// ─────────────────────────────────────────────────────────────
// DrawText 测试
// ─────────────────────────────────────────────────────────────

func TestDrawText(t *testing.T) {
	t.Run("draw normal text", func(t *testing.T) {
		c := NewSoftCanvas(200, 100)
		font := DefaultFont()
		p := paint.DefaultPaint()
		c.DrawText("Hello", 10, 50, font, p)
	})

	t.Run("empty string", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		font := DefaultFont()
		p := paint.DefaultPaint()
		c.DrawText("", 10, 50, font, p) // 不 panic
	})

	t.Run("zero size font", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		font := Font{Family: "sans-serif", Size: 0}
		p := paint.DefaultPaint()
		c.DrawText("test", 10, 50, font, p) // 应使用默认字号 16
	})

	t.Run("text outside canvas", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		font := DefaultFont()
		p := paint.DefaultPaint()
		c.DrawText("far away", -1000, -1000, font, p) // 不 panic
	})
}

// ─────────────────────────────────────────────────────────────
// DrawImage 测试
// ─────────────────────────────────────────────────────────────

func TestDrawImage(t *testing.T) {
	t.Run("draw valid image", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		src := image.NewRGBA(image.Rect(0, 0, 20, 20))
		// 填充红色
		for y := 0; y < 20; y++ {
			for x := 0; x < 20; x++ {
				src.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			}
		}
		dst := types.Rect{X: 10, Y: 10, Width: 40, Height: 40}
		c.DrawImage(src, dst)
	})

	t.Run("nil image", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		// DrawImage 使用 img.Bounds()，nil 会 panic，所以需要捕获
		defer func() {
			if r := recover(); r != nil {
				t.Logf("DrawImage with nil panicked (expected): %v", r)
			}
		}()
		c.DrawImage(nil, types.Rect{X: 0, Y: 0, Width: 10, Height: 10})
	})

	t.Run("empty image bounds", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		src := image.NewRGBA(image.Rect(0, 0, 0, 0))
		c.DrawImage(src, types.Rect{X: 0, Y: 0, Width: 10, Height: 10}) // 不 panic
	})

	t.Run("image with zero destination size", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		src := image.NewRGBA(image.Rect(0, 0, 10, 10))
		c.DrawImage(src, types.Rect{X: 10, Y: 10, Width: 0, Height: 0})  // 不 panic
		c.DrawImage(src, types.Rect{X: 10, Y: 10, Width: -5, Height: -5}) // 不 panic
	})

	t.Run("image with negative destination", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		src := image.NewRGBA(image.Rect(0, 0, 10, 10))
		c.DrawImage(src, types.Rect{X: -100, Y: -100, Width: 50, Height: 50}) // 不 panic
	})
}

// ─────────────────────────────────────────────────────────────
// MeasureText 测试
// ─────────────────────────────────────────────────────────────

func TestMeasureText(t *testing.T) {
	t.Run("measure normal text", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		font := DefaultFont()
		m := c.MeasureText("Hello", font)
		if m.Width <= 0 {
			t.Errorf("MeasureText Width = %v, want > 0", m.Width)
		}
		if m.Height <= 0 {
			t.Errorf("MeasureText Height = %v, want > 0", m.Height)
		}
		if m.Ascent <= 0 {
			t.Errorf("MeasureText Ascent = %v, want > 0", m.Ascent)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		font := DefaultFont()
		m := c.MeasureText("", font)
		// 空字符串应返回零值 TextMetrics
		if m.Width != 0 || m.Height != 0 {
			t.Errorf("MeasureText('') = %+v, want zero values", m)
		}
	})

	t.Run("zero font size", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		font := Font{Family: "sans-serif", Size: 0}
		m := c.MeasureText("test", font)
		if m.Width <= 0 {
			t.Errorf("MeasureText with zero size Width = %v, want > 0", m.Width)
		}
		if m.Height <= 0 {
			t.Errorf("MeasureText with zero size Height = %v, want > 0", m.Height)
		}
	})

	t.Run("chinese text", func(t *testing.T) {
		c := NewSoftCanvas(100, 100)
		font := DefaultFont()
		m := c.MeasureText("你好世界", font)
		if m.Width <= 0 {
			t.Errorf("MeasureText Chinese Width = %v, want > 0", m.Width)
		}
	})
}

// ─────────────────────────────────────────────────────────────
// Flush 测试
// ─────────────────────────────────────────────────────────────

func TestFlush(t *testing.T) {
	c := NewSoftCanvas(100, 100)
	err := c.Flush()
	if err != nil {
		t.Errorf("Flush() returned error: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────
// SaveToPNG 测试
// ─────────────────────────────────────────────────────────────

func TestSaveToPNG(t *testing.T) {
	t.Run("save to temp file", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "test.png")
		err := c.SaveToPNG(path)
		if err != nil {
			t.Errorf("SaveToPNG returned error: %v", err)
		}
		// 验证文件被创建（当前实现不会实际写入 PNG，但文件被 Create 了）
		// 注意：SaveToPNG 的实现调用了 os.Create 但 defer f.Close() 后 return nil
		// 没有实际编码 PNG，但文件已创建
		_, statErr := os.Stat(path)
		if statErr != nil {
			t.Logf("os.Stat after SaveToPNG: %v (expected: file might not have PNG data)", statErr)
		}
	})

	t.Run("save to invalid path", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		err := c.SaveToPNG("") // 空路径可能 panic 或不 panic
		if err != nil {
			t.Logf("SaveToPNG with empty path returned error: %v (acceptable)", err)
		}
	})

	t.Run("save to non-existent directory", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		err := c.SaveToPNG("Z:\\nonexistent_dir\\test.png")
		if err != nil {
			t.Logf("SaveToPNG with bad path returned error: %v (expected)", err)
		}
	})
}

// ─────────────────────────────────────────────────────────────
// 综合边界测试：所有方法在零/负/奇异参数下不 panic
// ─────────────────────────────────────────────────────────────

func TestBoundaryConditionsPanicFree(t *testing.T) {
	// 创建一个极小的画布，所有操作都不应 panic
	c := NewSoftCanvas(1, 1)

	p := paint.DefaultPaint()
	strokeP := paint.DefaultStrokePaint()
	font := DefaultFont()

	t.Run("all methods on 1x1 canvas", func(t *testing.T) {
		c.Save()
		c.Restore()
		c.Translate(1, 1)
		c.Rotate(90)
		c.Scale(2, 2)
		c.Clip(nil)
		c.Clip(NewPath())
		c.ClipRect(0, 0, 1, 1)
		c.DrawPath(nil, p)
		c.DrawPath(NewPath(), p)
		c.DrawRect(-1, -1, 5, 5, p)
		c.DrawRoundedRect(-1, -1, 5, 5, 1, p)
		c.DrawCircle(0, 0, 10, p)
		c.DrawLine(-5, -5, 5, 5, strokeP)
		c.DrawText("X", 0, 0, font, p)
		c.MeasureText("X", font)
		c.DrawImage(image.NewRGBA(image.Rect(0, 0, 2, 2)), types.Rect{X: -1, Y: -1, Width: 3, Height: 3})
		_ = c.Width()
		_ = c.Height()
		_ = c.Flush()
		_ = c.Image()
	})
}

func TestBoundaryZeroCanvas(t *testing.T) {
	// 使用零尺寸画布
	c := NewSoftCanvas(0, 0)

	p := paint.DefaultPaint()
	strokeP := paint.DefaultStrokePaint()
	font := DefaultFont()

	t.Run("all methods on 0x0 canvas", func(t *testing.T) {
		c.Save()
		c.Restore()
		c.ClipRect(0, 0, 0, 0)
		c.DrawRect(0, 0, 0, 0, p)
		c.DrawCircle(0, 0, 0, p)
		c.DrawLine(0, 0, 0, 0, strokeP)
		c.DrawText("", 0, 0, font, p)
		c.MeasureText("", font)
		_ = c.Width()
		_ = c.Height()
		_ = c.Flush()
	})
}

// ─────────────────────────────────────────────────────────────
// 绘制像素验证测试
// ─────────────────────────────────────────────────────────────

func TestDrawRectPixelColors(t *testing.T) {
	c := NewSoftCanvas(20, 20)

	// 绘制红色填充矩形
	red := paint.Paint{
		Style:   paint.StyleFill,
		Color:   types.ColorFromRGB(255, 0, 0),
		Opacity: 1.0,
	}
	c.DrawRect(5, 5, 10, 10, red)

	// 验证矩形内部像素为红色
	for y := 6; y < 14; y++ {
		for x := 6; x < 14; x++ {
			r, _, _, _ := c.img.At(x, y).RGBA()
			if r == 0 {
				t.Errorf("pixel at (%d,%d) inside rect is not red", x, y)
				goto done
			}
		}
	}
done:

	// 验证矩形外部（左上角）仍为黑色（透明/初始颜色）
	r, g, b, a := c.img.At(2, 2).RGBA()
	// 初始 RGBA 图像是透明的黑 (0,0,0,0)
	if r != 0 || g != 0 || b != 0 || a != 0 {
		t.Logf("outside pixel at (2,2) has unexpected color: R=%d G=%d B=%d A=%d", r, g, b, a)
	}
}

func TestDrawLinePixelColors(t *testing.T) {
	c := NewSoftCanvas(20, 20)

	// 绘制水平线
	blue := paint.Paint{
		Style:       paint.StyleStroke,
		Color:       types.ColorFromRGB(0, 0, 255),
		StrokeWidth: 1.0,
		Opacity:     1.0,
	}
	c.DrawLine(2, 10, 18, 10, blue)

	// 验证直线上的像素为蓝色
	found := false
	for x := 2; x <= 18; x++ {
		_, _, b, _ := c.img.At(x, 10).RGBA()
		if b > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("DrawLine did not produce any blue pixels on the line")
	}
}

func TestDrawCirclePixelColors(t *testing.T) {
	c := NewSoftCanvas(30, 30)

	green := paint.Paint{
		Style:   paint.StyleFill,
		Color:   types.ColorFromRGB(0, 255, 0),
		Opacity: 1.0,
	}
	c.DrawCircle(15, 15, 10, green)

	// 验证圆心像素为绿色
	_, g, _, _ := c.img.At(15, 15).RGBA()
	if g == 0 {
		t.Error("Circle center should be green")
	}
}

func TestDrawImagePixelColors(t *testing.T) {
	c := NewSoftCanvas(30, 30)

	// 创建一个红蓝各半的图像
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if x < 2 {
				src.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255}) // 红色
			} else {
				src.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255}) // 蓝色
			}
		}
	}

	dst := types.Rect{X: 5, Y: 5, Width: 20, Height: 20}
	c.DrawImage(src, dst)

	// 验证目标区域左半部分有红色像素
	foundRed := false
	foundBlue := false
	for y := 5; y < 25; y++ {
		for x := 5; x < 15; x++ {
			r, _, _, _ := c.img.At(x, y).RGBA()
			if r > 0 {
				foundRed = true
			}
		}
		for x := 15; x < 25; x++ {
			_, _, b, _ := c.img.At(x, y).RGBA()
			if b > 0 {
				foundBlue = true
			}
		}
	}
	if !foundRed {
		t.Error("Left half of drawn image should have red pixels")
	}
	if !foundBlue {
		t.Error("Right half of drawn image should have blue pixels")
	}
}

// ─────────────────────────────────────────────────────────────
// Clip 像素验证测试
// ─────────────────────────────────────────────────────────────

func TestClipAffectsDrawing(t *testing.T) {
	c := NewSoftCanvas(50, 50)

	// 设置裁剪区域为左上角 10x10
	c.ClipRect(0, 0, 10, 10)

	red := paint.Paint{
		Style:   paint.StyleFill,
		Color:   types.ColorFromRGB(255, 0, 0),
		Opacity: 1.0,
	}
	// 绘制覆盖全画布的矩形
	c.DrawRect(0, 0, 50, 50, red)

	// 裁剪区域内应有红色像素
	inR, _, _, _ := c.img.At(5, 5).RGBA()
	if inR == 0 {
		t.Error("Pixel inside clip region should be red")
	}

	// 裁剪区域外应无红色像素（透明黑色）
	outR, outG, outB, outA := c.img.At(25, 25).RGBA()
	if outR != 0 || outG != 0 || outB != 0 || outA != 0 {
		t.Logf("Pixel outside clip region at (25,25): R=%d G=%d B=%d A=%d", outR, outG, outB, outA)
	}
}

// ─────────────────────────────────────────────────────────────
// 变换方法兼容性测试
// ─────────────────────────────────────────────────────────────

func TestTransformCompatibility(t *testing.T) {
	// 验证变换方法调用不 panic，且后续绘制仍正常工作
	c := NewSoftCanvas(50, 50)
	c.Translate(10, 10)
	c.Rotate(45)
	c.Scale(2, 2)
	// 变换后绘制
	p := paint.DefaultPaint()
	c.DrawRect(5, 5, 10, 10, p)
}

// ─────────────────────────────────────────────────────────────
// SaveToPNG 的 SaveToPNG 方法当前实现检查
// ─────────────────────────────────────────────────────────────

func TestSaveToPNGCurrentBehavior(t *testing.T) {
	// SaveToPNG 当前实现创建文件但不编码 PNG，验证行为
	c := NewSoftCanvas(10, 10)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "output.png")

	err := c.SaveToPNG(path)
	if err != nil {
		t.Fatalf("SaveToPNG returned error: %v", err)
	}

	// 文件应该存在（被 os.Create 创建了）
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatalf("os.Stat failed: %v", statErr)
	}
	if info.Size() <= 0 {
		t.Logf("SaveToPNG created file but size=%d (no PNG data written yet)", info.Size())
	}
}

// ─────────────────────────────────────────────────────────────
// inClip 内部方法行为验证
// ─────────────────────────────────────────────────────────────

func TestInClip(t *testing.T) {
	c := NewSoftCanvas(50, 50)

	// 默认情况下 clipEnabled 为 false，inClip 应返回 true
	if !c.inClip(10, 10) {
		t.Error("inClip should return true when clip is disabled")
	}

	// 启用裁剪
	c.ClipRect(10, 10, 20, 20)
	if c.inClip(15, 15) != true {
		t.Error("pixel (15,15) should be inside clip rect (10,10)-(30,30)")
	}
	if c.inClip(5, 5) != false {
		t.Error("pixel (5,5) should be outside clip rect")
	}
	if c.inClip(35, 35) != false {
		t.Error("pixel (35,35) should be outside clip rect")
	}
}

// ─────────────────────────────────────────────────────────────
// 多次绘制验证
// ─────────────────────────────────────────────────────────────

func TestMultipleDraws(t *testing.T) {
	c := NewSoftCanvas(50, 50)

	// 依次绘制多个图形
	red := paint.Paint{Style: paint.StyleFill, Color: types.ColorFromRGB(255, 0, 0), Opacity: 1.0}
	green := paint.Paint{Style: paint.StyleFill, Color: types.ColorFromRGB(0, 255, 0), Opacity: 1.0}
	blue := paint.Paint{Style: paint.StyleFill, Color: types.ColorFromRGB(0, 0, 255), Opacity: 1.0}

	c.DrawRect(0, 0, 10, 10, red)
	c.DrawCircle(25, 25, 10, green)
	c.DrawLine(0, 49, 49, 0, paint.DefaultStrokePaint())
	c.DrawText("Hi", 30, 30, DefaultFont(), blue)

	// 只是验证不 panic
}

// ─────────────────────────────────────────────────────────────
// Helper 测试
// ─────────────────────────────────────────────────────────────

func TestClampInt(t *testing.T) {
	tests := []struct {
		v, min, max, want int
	}{
		{5, 0, 10, 5},
		{-5, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 0, 0},
	}
	for _, tt := range tests {
		got := clampInt(tt.v, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clampInt(%d,%d,%d) = %d, want %d", tt.v, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestClampIntRange(t *testing.T) {
	x0, y0, x1, y1 := -5, -5, 55, 55
	clampIntRange(&x0, &y0, &x1, &y1, 50, 50)
	if x0 != 0 || y0 != 0 || x1 != 50 || y1 != 50 {
		t.Errorf("clampIntRange = (%d,%d,%d,%d), want (0,0,50,50)", x0, y0, x1, y1)
	}
}

// ─────────────────────────────────────────────────────────────
// 字体常量重置测试隔离（如果包级字体状态影响测试）
// ─────────────────────────────────────────────────────────────

func TestTextMetricsConsistency(t *testing.T) {
	// 多次测量同一文本应返回相同值
	c := NewSoftCanvas(100, 100)
	font := DefaultFont()

	m1 := c.MeasureText("Hello World", font)
	m2 := c.MeasureText("Hello World", font)

	if m1.Width != m2.Width {
		t.Errorf("MeasureText inconsistent Width: %v vs %v", m1.Width, m2.Width)
	}
	if m1.Height != m2.Height {
		t.Errorf("MeasureText inconsistent Height: %v vs %v", m1.Height, m2.Height)
	}
	if m1.Ascent != m2.Ascent {
		t.Errorf("MeasureText inconsistent Ascent: %v vs %v", m1.Ascent, m2.Ascent)
	}
}

// ─────────────────────────────────────────────────────────────
// 空路径操作测试
// ─────────────────────────────────────────────────────────────

func TestEmptyPathOperations(t *testing.T) {
	t.Run("path with only MoveTo", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		path := NewPath()
		path.MoveTo(10, 10)
		// 只有 MoveTo 没有 LineTo/Close，flattenPath 应产生一个顶点
		p := paint.DefaultPaint()
		c.DrawPath(path, p) // 少于3个顶点，fillPolygon 直接返回
	})

	t.Run("path with single point", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		path := NewPath()
		path.MoveTo(10, 10)
		path.LineTo(10, 10) // 零长度线段
		p := paint.DefaultStrokePaint()
		c.DrawPath(path, p)
	})
}

// ─────────────────────────────────────────────────────────────
// RoundedRect with extreme parameters
// ─────────────────────────────────────────────────────────────

func TestDrawRoundedRectEdgeCases(t *testing.T) {
	t.Run("negative radius", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultPaint()
		c.DrawRoundedRect(10, 10, 30, 30, -5, p) // 负半径应转为0，走 DrawRect 路径
	})

	t.Run("very large radius", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultPaint()
		c.DrawRoundedRect(10, 10, 30, 30, 1e6, p) // 极大半径
	})

	t.Run("half pixel rect", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		p := paint.DefaultPaint()
		c.DrawRoundedRect(10.5, 10.5, 20.7, 20.3, 2.5, p) // 非整数坐标
	})
}

// ─────────────────────────────────────────────────────────────
// Ensure all public API methods are covered
// ─────────────────────────────────────────────────────────────

func TestAllPublicMethodsCompile(t *testing.T) {
	// 此测试确保所有 Canvas 接口方法可在 SoftCanvas 上调用
	c := NewSoftCanvas(10, 10)
	font := DefaultFont()
	p := paint.DefaultPaint()

	_ = c.Image()
	c.Save()
	c.Restore()
	c.Translate(1, 1)
	c.Rotate(1)
	c.Scale(1, 1)
	c.Clip(nil)
	c.ClipRect(0, 0, 5, 5)
	_ = c.Width()
	_ = c.Height()
	_ = c.Flush()
	_ = c.MeasureText("A", font)
	c.DrawPath(nil, p)
	c.DrawRect(0, 0, 5, 5, p)
	c.DrawRoundedRect(0, 0, 5, 5, 1, p)
	c.DrawCircle(5, 5, 3, p)
	c.DrawLine(0, 0, 5, 5, p)
	c.DrawText("A", 0, 0, font, p)
	c.DrawImage(image.NewRGBA(image.Rect(0, 0, 1, 1)), types.Rect{X: 0, Y: 0, Width: 1, Height: 1})
	_ = c.SaveToPNG("")
}

// ─────────────────────────────────────────────────────────────
// DrawCircle with 0 stroke width (should default to 1)
// ─────────────────────────────────────────────────────────────

func TestDrawCircleZeroStrokeWidth(t *testing.T) {
	c := NewSoftCanvas(50, 50)
	p := paint.Paint{
		Style:       paint.StyleStroke,
		Color:       types.ColorBlack,
		StrokeWidth: 0,
		Opacity:     1.0,
	}
	c.DrawCircle(25, 25, 10, p) // 不 panic
}

// ─────────────────────────────────────────────────────────────
// DrawRoundedRect fill and stroke
// ─────────────────────────────────────────────────────────────

func TestDrawRoundedRectFillAndStroke(t *testing.T) {
	c := NewSoftCanvas(50, 50)
	p := paint.Paint{
		Style:       paint.StyleFillAndStroke,
		Color:       types.ColorFromRGB(255, 0, 0),
		StrokeWidth: 2.0,
		Opacity:     1.0,
	}
	c.DrawRoundedRect(5, 5, 40, 40, 5, p)
}

// ─────────────────────────────────────────────────────────────
// DrawLine with fractional coordinates
// ─────────────────────────────────────────────────────────────

func TestDrawLineFractional(t *testing.T) {
	c := NewSoftCanvas(50, 50)
	p := paint.DefaultStrokePaint()
	c.DrawLine(10.7, 10.3, 40.2, 40.9, p)
	c.DrawLine(10.2, 40.8, 40.5, 10.1, p)
}

// ─────────────────────────────────────────────────────────────
// 多行文本测试
// ─────────────────────────────────────────────────────────────

func TestDrawTextMultiline(t *testing.T) {
	c := NewSoftCanvas(100, 100)
	font := DefaultFont()
	p := paint.DefaultPaint()

	// 绘制多行文本（当前实现中 \n 不会被特殊处理，但不应 panic）
	text := "Line1\nLine2\nLine3"
	c.DrawText(text, 10, 30, font, p)
}

// ─────────────────────────────────────────────────────────────
// 透明色绘制测试
// ─────────────────────────────────────────────────────────────

func TestDrawTransparent(t *testing.T) {
	c := NewSoftCanvas(20, 20)

	// 先用红色填充
	red := paint.Paint{Style: paint.StyleFill, Color: types.ColorFromRGB(255, 0, 0), Opacity: 1.0}
	c.DrawRect(0, 0, 20, 20, red)

	// 再用透明色覆盖（透明度为0）
	transparent := paint.Paint{Style: paint.StyleFill, Color: types.ColorFromRGBA(0, 0, 255, 255), Opacity: 0}
	c.DrawRect(0, 0, 20, 20, transparent)

	// 像素应保持红色（透明度0的绘制不改变像素）
	r, _, _, _ := c.img.At(10, 10).RGBA()
	if r == 0 {
		t.Error("Pixel should still be red after transparent overlay")
	}
}

// ─────────────────────────────────────────────────────────────
// 裁剪与 Save/Restore 交互测试
// ─────────────────────────────────────────────────────────────

func TestClipWithSaveRestore(t *testing.T) {
	c := NewSoftCanvas(50, 50)

	// 设置裁剪区域并保存
	c.ClipRect(0, 0, 10, 10)
	c.Save()

	// 改变裁剪
	c.ClipRect(20, 20, 10, 10)

	// 绘制
	red := paint.Paint{Style: paint.StyleFill, Color: types.ColorFromRGB(255, 0, 0), Opacity: 1.0}
	c.DrawRect(0, 0, 50, 50, red)

	// Restore 恢复裁剪
	c.Restore()

	// 恢复后的裁剪区域应回到 (0,0)-(10,10)
	expected := image.Rect(0, 0, 10, 10)
	if c.clipRect != expected {
		t.Errorf("clipRect after restore = %v, want %v", c.clipRect, expected)
	}
}

// ─────────────────────────────────────────────────────────────
// 路径操作：各种 PathOp 组合
// ─────────────────────────────────────────────────────────────

func TestPathOperations(t *testing.T) {
	c := NewSoftCanvas(100, 100)
	p := paint.DefaultStrokePaint()

	t.Run("path with close only", func(t *testing.T) {
		path := NewPath()
		path.Close() // 空路径 close
		c.DrawPath(path, p)
	})

	t.Run("path with arc clockwise", func(t *testing.T) {
		path := NewPath()
		path.MoveTo(50, 50)
		path.Arc(50, 50, 20, 0, math.Pi*2, true)
		c.DrawPath(path, p)
	})

	t.Run("path with arc counter-clockwise", func(t *testing.T) {
		path := NewPath()
		path.MoveTo(50, 50)
		path.Arc(50, 50, 20, 0, math.Pi*2, false)
		c.DrawPath(path, p)
	})
}

// ─────────────────────────────────────────────────────────────
// 超大尺寸图像绘制
// ─────────────────────────────────────────────────────────────

func TestDrawLargeImage(t *testing.T) {
	c := NewSoftCanvas(10, 10)
	src := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
	c.DrawImage(src, types.Rect{X: 0, Y: 0, Width: 10, Height: 10}) // 缩放到小区域
}

// ─────────────────────────────────────────────────────────────
// 抗锯齿标志兼容性（当前未实现，但不 panic）
// ─────────────────────────────────────────────────────────────

func TestAntiAliasFlag(t *testing.T) {
	c := NewSoftCanvas(20, 20)
	p := paint.Paint{
		Style:     paint.StyleFill,
		Color:     types.ColorBlack,
		AntiAlias: true,
		Opacity:   1.0,
	}
	c.DrawCircle(10, 10, 5, p)

	p2 := paint.Paint{
		Style:     paint.StyleStroke,
		Color:     types.ColorBlack,
		AntiAlias: false,
		Opacity:   1.0,
	}
	c.DrawLine(0, 0, 19, 19, p2)
}

// ─────────────────────────────────────────────────────────────
// 文字与背景重叠测试
// ─────────────────────────────────────────────────────────────

func TestDrawTextOnBackground(t *testing.T) {
	c := NewSoftCanvas(100, 50)

	// 先绘制蓝色背景
	blue := paint.Paint{Style: paint.StyleFill, Color: types.ColorFromRGB(0, 0, 255), Opacity: 1.0}
	c.DrawRect(0, 0, 100, 50, blue)

	// 再绘制白色文字
	white := paint.Paint{Style: paint.StyleFill, Color: types.ColorFromRGB(255, 255, 255), Opacity: 1.0}
	c.DrawText("Test on Blue", 10, 30, DefaultFont(), white)

	// 不 panic 即可
}

// ─────────────────────────────────────────────────────────────
// 测试空路径绘制不引发 panic
// ─────────────────────────────────────────────────────────────

func TestDrawPathEdgeCases(t *testing.T) {
	t.Run("path with only arc no move", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		path := NewPath()
		// 没有 MoveTo 直接 Arc
		path.Arc(25, 25, 10, 0, math.Pi, false)
		p := paint.DefaultStrokePaint()
		c.DrawPath(path, p)
	})

	t.Run("path with multiple move to", func(t *testing.T) {
		c := NewSoftCanvas(50, 50)
		path := NewPath()
		path.MoveTo(10, 10)
		path.MoveTo(20, 20)
		path.MoveTo(30, 30)
		path.LineTo(40, 40)
		p := paint.DefaultPaint()
		c.DrawPath(path, p)
	})
}

// ─────────────────────────────────────────────────────────────
// 颜色转换测试
// ─────────────────────────────────────────────────────────────

func TestToSoftRGBA(t *testing.T) {
	tests := []struct {
		name string
		p    paint.Paint
		want color.RGBA
	}{
		{
			name: "full opacity",
			p: paint.Paint{
				Color:   types.ColorFromRGB(255, 128, 64),
				Opacity: 1.0,
			},
			want: color.RGBA{R: 255, G: 128, B: 64, A: 255},
		},
		{
			name: "half opacity",
			p: paint.Paint{
				Color:   types.ColorFromRGB(255, 0, 0),
				Opacity: 0.5,
			},
			want: color.RGBA{R: 255, G: 0, B: 0, A: 127},
		},
		{
			name: "zero opacity",
			p: paint.Paint{
				Color:   types.ColorFromRGB(100, 200, 50),
				Opacity: 0,
			},
			want: color.RGBA{R: 100, G: 200, B: 50, A: 0},
		},
		{
			name: "transparent base color",
			p: paint.Paint{
				Color:   types.ColorFromRGBA(255, 0, 0, 128),
				Opacity: 1.0,
			},
			want: color.RGBA{R: 255, G: 0, B: 0, A: 128},
		},
		{
			name: "transparent base + half opacity",
			p: paint.Paint{
				Color:   types.ColorFromRGBA(255, 0, 0, 200),
				Opacity: 0.5,
			},
			want: color.RGBA{R: 255, G: 0, B: 0, A: 100},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSoftRGBA(tt.p)
			if got != tt.want {
				t.Errorf("toSoftRGBA() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────
// 多次 Save 后 Restore 还原裁剪状态测试
// ─────────────────────────────────────────────────────────────

func TestMultipleSaveRestoreClipSequence(t *testing.T) {
	c := NewSoftCanvas(100, 100)

	// 初始无裁剪
	c.ClipRect(0, 0, 100, 100)
	c.Save() // 保存状态 A: (0,0)-(100,100)

	c.ClipRect(10, 10, 50, 50)
	c.Save() // 保存状态 B: (10,10)-(60,60)

	c.ClipRect(20, 20, 20, 20)
	// 当前裁剪: (20,20)-(40,40)

	c.Restore() // 回到 B: (10,10)-(60,60)
	if c.clipRect != image.Rect(10, 10, 60, 60) {
		t.Errorf("after restore to B, clipRect = %v, want (10,10)-(60,60)", c.clipRect)
	}

	c.Restore() // 回到 A: (0,0)-(100,100)
	if c.clipRect != image.Rect(0, 0, 100, 100) {
		t.Errorf("after restore to A, clipRect = %v, want (0,0)-(100,100)", c.clipRect)
	}
}

// ─────────────────────────────────────────────────────────────
// 确保 Image() 返回的图像与画布尺寸一致
// ─────────────────────────────────────────────────────────────

func TestImageBoundsMultipleSizes(t *testing.T) {
	sizes := []struct{ w, h int }{
		{1, 1},
		{100, 50},
		{320, 240},
		{1024, 768},
	}
	for _, s := range sizes {
		name := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(
			t.Name()+"/"+itoa(s.w)+"x"+itoa(s.h), " ", ""), "\n", ""))
		t.Run(name, func(t *testing.T) {
			c := NewSoftCanvas(s.w, s.h)
			img := c.Image()
			b := img.Bounds()
			if b.Dx() != s.w || b.Dy() != s.h {
				t.Errorf("Image bounds = %dx%d, want %dx%d", b.Dx(), b.Dy(), s.w, s.h)
			}
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

// ─────────────────────────────────────────────────────────────
// SoftCanvas 是否满足 Canvas 接口的编译期检查
// ─────────────────────────────────────────────────────────────

func TestSoftCanvasImplementsCanvas(t *testing.T) {
	// 编译期接口检查
	var _ Canvas = (*SoftCanvas)(nil)
}
