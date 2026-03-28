// cache.go — SVG 图像缓存。
// 以 "路径:宽x高" 为键缓存已解析的 ebiten.Image，避免重复光栅化。
package svg

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Cache SVG 解析结果缓存（键为 "path:WxH"）。
type Cache struct {
	images map[string]*ebiten.Image
}

// NewCache 创建空缓存。
func NewCache() *Cache {
	return &Cache{images: make(map[string]*ebiten.Image)}
}

// Get 从缓存获取图像，未命中返回 nil。
func (c *Cache) Get(key string, w, h int) *ebiten.Image {
	return c.images[cacheKey(key, w, h)]
}

// Put 将图像存入缓存。
func (c *Cache) Put(key string, w, h int, img *ebiten.Image) {
	c.images[cacheKey(key, w, h)] = img
}

// GetOrParse 从缓存获取，未命中则解析 SVG 数据并缓存。
func (c *Cache) GetOrParse(key string, data []byte, w, h int) (*ebiten.Image, error) {
	k := cacheKey(key, w, h)
	if img, ok := c.images[k]; ok {
		return img, nil
	}
	img, err := Parse(data, w, h)
	if err != nil {
		return nil, err
	}
	c.images[k] = img
	return img, nil
}

// Count 返回缓存中的图像数量。
func (c *Cache) Count() int {
	return len(c.images)
}

func cacheKey(path string, w, h int) string {
	return fmt.Sprintf("%s:%dx%d", path, w, h)
}
