package visual

import (
	"image"
	"image/color"
	"testing"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// ─────────────────────────────────────────────────────────────
// 基础功能测试
// ─────────────────────────────────────────────────────────────

// TestNewVisualTestContext 测试 VisualTestContext 的创建
func TestNewVisualTestContext(t *testing.T) {
	ctx := NewVisualTestContext(800, 600)
	if ctx == nil {
		t.Fatal("NewVisualTestContext returned nil")
	}
	if ctx.Width != 800 {
		t.Errorf("Width = %d, want 800", ctx.Width)
	}
	if ctx.Height != 600 {
		t.Errorf("Height = %d, want 600", ctx.Height)
	}
	if ctx.Canvas == nil {
		t.Error("Canvas is nil")
	}
	if ctx.Pipeline == nil {
		t.Error("Pipeline is nil")
	}
}

// TestRenderNilWidget 测试渲染 nil Widget 时返回错误
func TestRenderNilWidget(t *testing.T) {
	ctx := NewVisualTestContext(100, 100)
	frame, err := ctx.Render(nil)
	if err == nil {
		t.Fatal("Expected error for nil widget, got nil")
	}
	if frame != nil {
		t.Fatal("Expected nil frame for nil widget")
	}
}

// TestBlankCanvas 测试空白画布检测
func TestBlankCanvas(t *testing.T) {
	ctx := NewVisualTestContext(100, 100)

	// 创建一个空白 container（无背景、无子控件）
	frame, err := ctx.Render(&widget.Container{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	pa := frame.Analyze()
	if !pa.IsBlank() {
		t.Error("Expected blank canvas for empty Container")
	}
}

// TestNonBlankCanvas 测试非空白画布检测（有绘制内容的组件）
func TestNonBlankCanvas(t *testing.T) {
	ctx := NewVisualTestContext(200, 100)

	// 创建一个有背景色的 Container
	frame, err := ctx.Render(&widget.Container{
		Background: &widget.PaintWidget{
			Color: types.ColorRef(100, 150, 200),
		},
		Width:  200,
		Height: 100,
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	pa := frame.Analyze()
	if pa.IsBlank() {
		t.Error("Expected non-blank canvas for Container with background")
	}
}

// TestColorAt 测试坐标颜色读取
func TestColorAt(t *testing.T) {
	ctx := NewVisualTestContext(50, 50)

	// 使用蓝色背景的 Container
	frame, err := ctx.Render(&widget.Container{
		Background: &widget.PaintWidget{
			Color: types.ColorRef(50, 100, 200),
		},
		Width:  50,
		Height: 50,
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// 画布中央应该是蓝色
	c := frame.PixelAt(25, 25)
	if c.R != 50 || c.G != 100 || c.B != 200 {
		t.Errorf("Center pixel color = (%d,%d,%d), want (50,100,200)", c.R, c.G, c.B)
	}
}

// TestColorSimilarity 测试颜色相似度比较函数
func TestColorSimilarity(t *testing.T) {
	tests := []struct {
		a, b      color.RGBA
		tolerance uint8
		similar   bool
	}{
		{color.RGBA{100, 100, 100, 255}, color.RGBA{100, 100, 100, 255}, 0, true},
		{color.RGBA{100, 100, 100, 255}, color.RGBA{105, 100, 100, 255}, 0, false},
		{color.RGBA{100, 100, 100, 255}, color.RGBA{105, 100, 100, 255}, 5, true},
		{color.RGBA{100, 100, 100, 255}, color.RGBA{106, 100, 100, 255}, 5, false},
		{color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255}, 255, true},
	}

	for _, tt := range tests {
		got := colorSimilar(tt.a, tt.b, tt.tolerance)
		if got != tt.similar {
			t.Errorf("colorSimilar(%v, %v, %d) = %v, want %v",
				tt.a, tt.b, tt.tolerance, got, tt.similar)
		}
	}
}

// TestCountColor 测试颜色像素统计
func TestCountColor(t *testing.T) {
	ctx := NewVisualTestContext(100, 100)

	// 50x50 红色方块 + 50x50 蓝色方块的 Canvas
	// 使用 Container 的 Background 渲染纯色
	frame, err := ctx.Render(&widget.Container{
		Background: &widget.PaintWidget{
			Color: types.ColorRef(255, 0, 0),
		},
		Width:  100,
		Height: 100,
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	pa := frame.Analyze()
	redCount := pa.CountColor(color.RGBA{R: 255, G: 0, B: 0, A: 255}, 5)
	if redCount < 9000 { // 100x100 = 10000 像素，粗略检查
		t.Errorf("Expected ~10000 red pixels, got %d", redCount)
	}
}

// TestFindColorRect 测试颜色区域查找
func TestFindColorRect(t *testing.T) {
	ctx := NewVisualTestContext(100, 100)

	frame, err := ctx.Render(&widget.Container{
		Background: &widget.PaintWidget{
			Color: types.ColorRef(200, 100, 50),
		},
		Width:  100,
		Height: 100,
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	pa := frame.Analyze()
	target := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	rect := pa.FindColorRect(target, 5)

	if rect.Empty() {
		t.Fatal("FindColorRect returned empty rect")
	}
	// 应该覆盖整个画布
	if rect.Min.X != 0 || rect.Min.Y != 0 {
		t.Errorf("Expected rect.Min=(0,0), got %v", rect.Min)
	}
}

// TestFindColorClusters 测试颜色聚类查找
func TestFindColorClusters(t *testing.T) {
	ctx := NewVisualTestContext(100, 100)

	frame, err := ctx.Render(&widget.Container{
		Background: &widget.PaintWidget{
			Color: types.ColorRef(50, 150, 250),
		},
		Width:  100,
		Height: 100,
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	pa := frame.Analyze()
	// 查找蓝色区域的聚类（容差10，最少面积100）
	target := color.RGBA{R: 50, G: 150, B: 250, A: 255}
	clusters := pa.FindColorClusters(target, 10, 100)

	if len(clusters) == 0 {
		t.Fatal("FindColorClusters returned no clusters")
	}

	// 第一个聚类应该覆盖大部分画布
	mainCluster := clusters[0]
	area := mainCluster.Dx() * mainCluster.Dy()
	if area < 5000 {
		t.Errorf("Main cluster area = %d, expected > 5000", area)
	}
}

// ─────────────────────────────────────────────────────────────
// VisualSuite 测试
// ─────────────────────────────────────────────────────────────

// TestVisualSuite 测试 VisualSuite 的核心功能
func TestVisualSuite(t *testing.T) {
	ctx := NewVisualTestContext(200, 100)

	frame, err := ctx.Render(&widget.Container{
		Background: &widget.PaintWidget{
			Color: types.ColorRef(66, 133, 244),
		},
		Width:  200,
		Height: 100,
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	suite := NewVisualSuite()
	suite.AddCheck("非空白", CheckNonBlank())
	suite.AddCheck("有足够非白色像素", CheckNonWhiteRatio(0.5))
	suite.AddCheck("包含蓝色像素", CheckRectColor(
		image.Rect(50, 25, 150, 75),
		color.RGBA{R: 66, G: 133, B: 244, A: 255},
		10, 100))

	report := suite.Run(frame)

	if !report.AllPassed {
		t.Errorf("VisualSuite report: %s", report.String())
	}
}

// TestVisualSuiteWithFailure 测试 VisualSuite 在失败时的行为
func TestVisualSuiteWithFailure(t *testing.T) {
	ctx := NewVisualTestContext(100, 100)

	// 空白 Container
	frame, err := ctx.Render(&widget.Container{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	suite := NewVisualSuite()
	suite.AddCheck("应失败-非空白", CheckNonBlank())
	suite.AddCheck("应失败-高比例", CheckNonWhiteRatio(0.5))

	report := suite.Run(frame)

	if report.AllPassed {
		t.Error("Expected VisualSuite to report failure for blank canvas")
	}
	if report.Passed != 0 {
		t.Errorf("Expected 0 passed checks, got %d", report.Passed)
	}
	if report.Failed != 2 {
		t.Errorf("Expected 2 failed checks, got %d", report.Failed)
	}
}

// ─────────────────────────────────────────────────────────────
// 渲染完整组件测试
// ─────────────────────────────────────────────────────────────

// TestButtonRenders 测试 Button 是否被正确渲染
func TestButtonRenders(t *testing.T) {
	ctx := NewVisualTestContext(400, 200)

	frame, err := ctx.Render(&widget.Button{
		Text:  "Click Me",
		Color: types.ColorFromRGB(66, 133, 244),
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	pa := frame.Analyze()
	if pa.IsBlank() {
		t.Fatal("Button rendered nothing on canvas")
	}

	// 按钮背景色应该是蓝色
	blueCount := pa.CountColor(
		color.RGBA{R: 66, G: 133, B: 244, A: 255},
		10,
	)
	if blueCount < 100 {
		t.Errorf("Too few blue pixels for Button: %d", blueCount)
	}
}

// TestTextRenders 测试 Text 组件是否渲染出文本（非空白）
func TestTextRenders(t *testing.T) {
	ctx := NewVisualTestContext(400, 200)

	frame, err := ctx.Render(&widget.Text{
		Text:  "Hello Visual Validation!",
		Font:  canvas.DefaultFont(),
		Color: types.ColorFromRGB(33, 33, 33),
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	pa := frame.Analyze()
	if pa.IsBlank() {
		t.Fatal("Text rendered nothing on canvas")
	}

	// 文本颜色是深色，应该有非白色像素
	if !pa.HasNonWhiteContent() {
		t.Error("Text has no non-white content")
	}
}

// TestCompositeWidget 测试复合组件渲染
func TestCompositeWidget(t *testing.T) {
	ctx := NewVisualTestContext(400, 300)

	// Container + Button 组合（使用明确尺寸简化）
	root := &widget.Container{
		SingleChildWidget: widget.SingleChildWidget{
			Child: &widget.Button{
				Text:  "Submit",
				Color: types.ColorFromRGB(76, 175, 80),
			},
		},
		Width:  400,
		Height: 300,
		Padding: types.EdgeInsets(16),
		Background: &widget.PaintWidget{
			Color: types.ColorRef(255, 255, 255),
		},
	}

	frame, err := ctx.Render(root)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	suite := NewVisualSuite()
	suite.AddCheck("非空白", CheckNonBlank())
	suite.AddCheck("有内容", CheckNonWhiteRatio(0.01))
	// 按钮区域应该有非白色绘制内容
	suite.AddCheck("按钮区域有内容", CheckRectHasNonWhite(
		image.Rect(16, 16, 384, 284), 50))

	report := suite.Run(frame)
	if !report.AllPassed {
		t.Errorf("Composite widget visual check failed: %s", report.String())
	}
}
