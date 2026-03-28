// icon.go — 图标管理器。
// 从 AssetFS 懒加载 PNG 图标，全局单例模式（同 font.go）。
package render

import (
	"fmt"
	"sync"

	"defense2/internal/render/sprite"

	"github.com/hajimehoshi/ebiten/v2"
)

// IconManager 加载并缓存 assets/icons/ 下的 PNG 图标。
type IconManager struct {
	cache   *sprite.Cache
	assetFS AssetReader
}

const iconSourceSize = 64 // PNG 源文件尺寸

// Get 按名称返回图标（如 "stat-damage"、"slow"、"tower-freeze"）。
// 找不到或出错返回 nil。
func (im *IconManager) Get(name string) *ebiten.Image {
	if im == nil || im.assetFS == nil {
		return nil
	}
	path := fmt.Sprintf("assets/icons/%s.png", name)
	if img := im.cache.Get(path, iconSourceSize, iconSourceSize); img != nil {
		return img
	}
	data, err := im.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _ := im.cache.GetOrParse(path, data, iconSourceSize, iconSourceSize)
	return img
}

// ── 全局单例 ──

var (
	globalIM   *IconManager
	iconOnce   sync.Once
)

// InitGlobalIcons 初始化全局图标管理器（只执行一次）。
func InitGlobalIcons(assetFS AssetReader) {
	iconOnce.Do(func() {
		globalIM = &IconManager{
			cache:   sprite.NewCache(),
			assetFS: assetFS,
		}
	})
}

// GlobalIcons 返回全局图标管理器。未初始化时返回 nil。
func GlobalIcons() *IconManager {
	return globalIM
}
