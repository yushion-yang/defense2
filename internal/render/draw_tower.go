// draw_tower.go — 塔渲染。
// 优先使用 PNG 精灵渲染塔，回退到彩色方块。支持放塔预览。
package render

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/core/tower"
	"defense2/internal/render/sprite"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TowerRenderer 管理塔的 PNG 精灵渲染。
type TowerRenderer struct {
	cache   *sprite.Cache // 图像缓存，避免重复解码
	assetFS AssetReader   // 嵌入式资源文件读取器
}

// AssetReader 读取嵌入式资源文件的接口。
type AssetReader interface {
	ReadFile(name string) ([]byte, error)
}

// NewTowerRenderer 创建塔渲染器。
func NewTowerRenderer(assetFS AssetReader) *TowerRenderer {
	return &TowerRenderer{
		cache:   sprite.NewCache(),
		assetFS: assetFS,
	}
}

const towerSpriteSize = 40 // 塔 PNG 精灵尺寸（像素）

// DrawTowers 渲染所有已放置的塔。
func (tr *TowerRenderer) DrawTowers(screen *ebiten.Image, pool *tower.Pool) {
	pool.Each(func(t *tower.Tower) {
		cx := float32(t.X)
		cy := float32(t.Y)

		// 尝试加载 PNG 精灵
		img := tr.loadTowerImage(t)
		if img != nil {
			// 以塔中心为原点绘制精灵
			opts := &ebiten.DrawImageOptions{}
			w, h := img.Bounds().Dx(), img.Bounds().Dy()
			opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			opts.GeoM.Translate(float64(cx), float64(cy))
			screen.DrawImage(img, opts)
		} else {
			// 回退：彩色方块
			size := float32(20)
			clr := color.RGBA{R: t.Color[0], G: t.Color[1], B: t.Color[2], A: 255}
			if clr.R == 0 && clr.G == 0 && clr.B == 0 {
				clr = color.RGBA{R: 80, G: 140, B: 220, A: 255}
			}
			vector.DrawFilledRect(screen, cx-size/2, cy-size/2, size, size, clr, false)
		}

		// 射程圆（半透明）
		rangeClr := color.RGBA{R: t.Color[0], G: t.Color[1], B: t.Color[2], A: 30}
		drawCircleOutline(screen, cx, cy, float32(t.Range), 1, rangeClr)
	})
}

// loadTowerImage 尝试加载塔的 PNG 精灵。
// 路径约定：assets/towers/core/tower-{key}.png
func (tr *TowerRenderer) loadTowerImage(t *tower.Tower) *ebiten.Image {
	if tr.assetFS == nil {
		return nil
	}
	path := fmt.Sprintf("assets/towers/core/tower-%s.png", t.Key)
	cached := tr.cache.Get(path, towerSpriteSize, towerSpriteSize)
	if cached != nil {
		return cached
	}

	data, err := tr.assetFS.ReadFile(path)
	if err != nil {
		return nil // PNG 不存在，回退到方块
	}
	img, err := tr.cache.GetOrParse(path, data, towerSpriteSize, towerSpriteSize)
	if err != nil {
		return nil
	}
	return img
}

// DrawTowerRangePreview 绘制放塔预览（射程圆 + 方块影子）。
func DrawTowerRangePreview(screen *ebiten.Image, cx, cy float32, r float64, valid bool) {
	clr := color.RGBA{R: 100, G: 200, B: 100, A: 80}
	if !valid {
		clr = color.RGBA{R: 200, G: 80, B: 80, A: 80}
	}
	drawCircleOutline(screen, cx, cy, float32(r), 1.5, clr)

	size := float32(20)
	fillClr := color.RGBA{R: 100, G: 200, B: 100, A: 120}
	if !valid {
		fillClr = color.RGBA{R: 200, G: 80, B: 80, A: 120}
	}
	vector.DrawFilledRect(screen, cx-size/2, cy-size/2, size, size, fillClr, false)
}

// drawCircleOutline 用 48 段线段近似绘制圆形轮廓。
func drawCircleOutline(screen *ebiten.Image, cx, cy, r, width float32, clr color.RGBA) {
	const segments = 48
	for i := 0; i < segments; i++ {
		a1 := float64(i) * 2 * math.Pi / segments
		a2 := float64(i+1) * 2 * math.Pi / segments
		x1 := cx + r*float32(math.Cos(a1))
		y1 := cy + r*float32(math.Sin(a1))
		x2 := cx + r*float32(math.Cos(a2))
		y2 := cy + r*float32(math.Sin(a2))
		vector.StrokeLine(screen, x1, y1, x2, y2, width, clr, false)
	}
}
