package core_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"defense2/internal/render/sprite"
)

// createTestPNG 生成指定尺寸的最小 PNG（红色填充）。
func createTestPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	red := color.RGBA{R: 255, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, red)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

var testPNG = createTestPNG(32, 32)

func TestSpriteParse(t *testing.T) {
	img, err := sprite.Parse(testPNG, 32, 32)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w != 32 || h != 32 {
		t.Fatalf("预期 32x32，实际 %dx%d", w, h)
	}
}

func TestSpriteCache(t *testing.T) {
	c := sprite.NewCache()

	// 首次解码
	img1, err := c.GetOrParse("test", testPNG, 32, 32)
	if err != nil {
		t.Fatalf("首次解码失败: %v", err)
	}
	if c.Count() != 1 {
		t.Fatalf("缓存数量应为 1，实际 %d", c.Count())
	}

	// 缓存命中
	img2, err := c.GetOrParse("test", testPNG, 32, 32)
	if err != nil {
		t.Fatalf("缓存命中失败: %v", err)
	}
	if img1 != img2 {
		t.Fatal("缓存应返回同一实例")
	}

	// 不同尺寸不命中（使用不同 key 模拟）
	testPNG64 := createTestPNG(64, 64)
	_, err = c.GetOrParse("test", testPNG64, 64, 64)
	if err != nil {
		t.Fatalf("不同尺寸解码失败: %v", err)
	}
	if c.Count() != 2 {
		t.Fatalf("缓存数量应为 2，实际 %d", c.Count())
	}
}
