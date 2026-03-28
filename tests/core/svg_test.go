package core_test

import (
	"testing"

	"defense2/internal/render/svg"
)

// 最小有效 SVG（红色圆形）。
var testSVG = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <circle cx="32" cy="32" r="28" fill="red"/>
</svg>`)

func TestSVGParse(t *testing.T) {
	img, err := svg.Parse(testSVG, 32, 32)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w != 32 || h != 32 {
		t.Fatalf("预期 32x32，实际 %dx%d", w, h)
	}
}

func TestSVGCache(t *testing.T) {
	c := svg.NewCache()

	// 首次解析
	img1, err := c.GetOrParse("test", testSVG, 32, 32)
	if err != nil {
		t.Fatalf("首次解析失败: %v", err)
	}
	if c.Count() != 1 {
		t.Fatalf("缓存数量应为 1，实际 %d", c.Count())
	}

	// 缓存命中
	img2, err := c.GetOrParse("test", testSVG, 32, 32)
	if err != nil {
		t.Fatalf("缓存命中失败: %v", err)
	}
	if img1 != img2 {
		t.Fatal("缓存应返回同一实例")
	}

	// 不同尺寸不命中
	_, err = c.GetOrParse("test", testSVG, 64, 64)
	if err != nil {
		t.Fatalf("不同尺寸解析失败: %v", err)
	}
	if c.Count() != 2 {
		t.Fatalf("缓存数量应为 2，实际 %d", c.Count())
	}
}
