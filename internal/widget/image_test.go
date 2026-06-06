package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"
)

// makeGIFBytes 生成 n 帧、每帧纯色的动画 GIF（测试用）。
func makeGIFBytes(t *testing.T, n int) []byte {
	t.Helper()
	pal := color.Palette{
		color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}, color.RGBA{0, 0, 0, 255},
	}
	g := &gif.GIF{}
	for i := 0; i < n; i++ {
		fr := image.NewPaletted(image.Rect(0, 0, 12, 12), pal)
		ci := uint8(i % len(pal))
		for p := range fr.Pix {
			fr.Pix[p] = ci
		}
		g.Image = append(g.Image, fr)
		g.Delay = append(g.Delay, 8) // 80ms/帧
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestNewImageFromBytesGIF 多帧 GIF → 合成完整帧 + delays。
func TestNewImageFromBytesGIF(t *testing.T) {
	img, err := NewImageFromBytes(makeGIFBytes(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	if len(img.frames) != 3 {
		t.Errorf("frames=%d want 3", len(img.frames))
	}
	if len(img.delays) != 3 {
		t.Errorf("delays=%d want 3", len(img.delays))
	}
	if b := img.frames[0].Bounds(); b.Dx() != 12 || b.Dy() != 12 {
		t.Errorf("合成帧尺寸=%v want 12x12", b)
	}
	if img.Source == nil {
		t.Error("Source 应为首帧、非 nil")
	}
}

// TestNewImageFromBytesPNG 单帧 PNG → 静态(无动画帧)。
func TestNewImageFromBytesPNG(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	img, err := NewImageFromBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(img.frames) != 0 {
		t.Errorf("PNG 不应有动画帧, got %d", len(img.frames))
	}
	if img.Source == nil {
		t.Error("PNG Source 应非 nil")
	}
}

// TestNewImageFromSVG SVG 字符串光栅化为非空图。
func TestNewImageFromSVG(t *testing.T) {
	img, err := NewImageFromSVG(`<svg width="20" height="20"><rect width="20" height="20" fill="#ff0000"/></svg>`, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if img.Source == nil {
		t.Fatal("SVG Source 应非 nil")
	}
	if b := img.Source.Bounds(); b.Dx() <= 0 || b.Dy() <= 0 {
		t.Errorf("SVG 渲染尺寸=%v 应>0", b)
	}
}
