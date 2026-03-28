// mobile.go — Android 移动端入口。
// 通过 ebitenmobile 编译为 .aar 库供 Android 项目集成。
// 使用 init() 在加载时自动启动游戏，导出 Dummy() 满足 ebitenmobile 要求。
package mobile

import (
	"github.com/hajimehoshi/ebiten/v2/mobile"

	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/scene"
)

func init() {
	config.SetDataFS(&defense2.DataFS)
	config.SetAssetFS(&defense2.AssetFS)
	mobile.SetGame(scene.NewGame())
}

// Dummy ebitenmobile 要求导出的占位函数。
func Dummy() {}
