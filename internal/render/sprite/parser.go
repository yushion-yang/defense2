// parser.go — PNG 图像解析器。
// 将 PNG 字节数据解码为 ebiten.Image。
package sprite

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

// Parse 将 PNG 字节数据解码为 ebiten.Image。
// w, h 参数保留以兼容缓存接口，但不再用于缩放（PNG 已预渲染为目标尺寸）。
func Parse(data []byte, w, h int) (*ebiten.Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("png decode: %w", err)
	}
	return ebiten.NewImageFromImage(img), nil
}
