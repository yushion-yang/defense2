// draw_enemy.go — 敌人渲染。
// 优先使用 SVG 图像渲染敌人，回退到红色圆形。受伤时显示血条，状态效果显示微图标。
package render

import (
	"fmt"
	"image/color"

	"defense2/internal/core/enemy"
	"defense2/internal/render/svg"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// EnemyRenderer 管理敌人的 SVG 图像渲染。
type EnemyRenderer struct {
	cache   *svg.Cache  // SVG 解析结果缓存
	assetFS AssetReader // 嵌入式资源文件读取器
}

// NewEnemyRenderer 创建敌人渲染器。
func NewEnemyRenderer(assetFS AssetReader) *EnemyRenderer {
	return &EnemyRenderer{
		cache:   svg.NewCache(),
		assetFS: assetFS,
	}
}

const enemySpriteSize = 24 // 敌人 SVG 渲染尺寸（像素）

// DrawEnemies 渲染所有存活敌人。
func (er *EnemyRenderer) DrawEnemies(screen *ebiten.Image, pool *enemy.Pool) {
	pool.Each(func(e *enemy.Enemy) {
		cx := float32(e.X)
		cy := float32(e.Y)
		r := float32(e.Radius)

		// 尝试加载 SVG
		img := er.loadEnemyImage(e)
		if img != nil {
			opts := &ebiten.DrawImageOptions{}
			w, h := img.Bounds().Dx(), img.Bounds().Dy()
			opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			opts.GeoM.Translate(float64(cx), float64(cy))
			screen.DrawImage(img, opts)
		} else {
			// 回退：红色圆形
			bodyColor := color.RGBA{R: 200, G: 60, B: 60, A: 255}
			if e.Boss {
				bodyColor = color.RGBA{R: 220, G: 160, B: 40, A: 255} // Boss 金色
			}
			vector.DrawFilledCircle(screen, cx, cy, r, bodyColor, false)
		}

		// 护盾（蓝色外环）
		if e.ShieldHP > 0 {
			vector.DrawFilledCircle(screen, cx, cy, r+2,
				color.RGBA{R: 80, G: 140, B: 255, A: 80}, false)
		}

		// 血条（仅受伤时显示）
		if e.HP < e.MaxHP {
			barW := r * 2.5
			barH := float32(3)
			barX := cx - barW/2
			barY := cy - r - 6

			vector.DrawFilledRect(screen, barX, barY, barW, barH,
				color.RGBA{R: 60, G: 20, B: 20, A: 200}, false)
			fill := barW * float32(e.HP/e.MaxHP)
			if fill < 0 {
				fill = 0
			}
			vector.DrawFilledRect(screen, barX, barY, fill, barH,
				color.RGBA{R: 60, G: 200, B: 60, A: 255}, false)
		}

		// 状态效果微图标
		iconY := cy - r - 10
		iconX := cx + r + 2
		if e.SlowTimer > 0 {
			// 蓝点 = 减速
			vector.DrawFilledCircle(screen, iconX, iconY, 2,
				color.RGBA{R: 80, G: 160, B: 255, A: 255}, false)
			iconX += 5
		}
		if e.BleedTimer > 0 {
			// 红点 = 流血
			vector.DrawFilledCircle(screen, iconX, iconY, 2,
				color.RGBA{R: 255, G: 60, B: 60, A: 255}, false)
			iconX += 5
		}
		if e.BurnTimer > 0 {
			// 橙点 = 灼烧
			vector.DrawFilledCircle(screen, iconX, iconY, 2,
				color.RGBA{R: 255, G: 140, B: 40, A: 255}, false)
			iconX += 5
		}
		if e.StunTimer > 0 || e.RootTimer > 0 {
			// 白点 = 控制
			vector.DrawFilledCircle(screen, iconX, iconY, 2,
				color.RGBA{R: 255, G: 255, B: 255, A: 255}, false)
		}
	})
}

// loadEnemyImage 尝试加载敌人的 SVG 图像。
// 路径约定：assets/enemies/enemy-{archetype}.svg
func (er *EnemyRenderer) loadEnemyImage(e *enemy.Enemy) *ebiten.Image {
	if er.assetFS == nil || e.Archetype == "" || e.Archetype == "normal" {
		return nil
	}
	path := fmt.Sprintf("assets/enemies/enemy-%s.svg", e.Archetype)
	cached := er.cache.Get(path, enemySpriteSize, enemySpriteSize)
	if cached != nil {
		return cached
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := er.cache.GetOrParse(path, data, enemySpriteSize, enemySpriteSize)
	if err != nil {
		return nil
	}
	return img
}
