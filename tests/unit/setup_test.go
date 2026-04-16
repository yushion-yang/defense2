//go:build unittest

// setup_test.go — 单元测试初始化。
//
// 注入 embed.FS 并重置配置缓存，确保 gamemode 包的延迟加载能正确读取 gamemodes.json。
package unit

import (
	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/core/gamemode"
)

func init() {
	config.SetDataFS(&defense2.DataFS)
	// 重置缓存，让延迟加载在 dataFS 就绪后重新执行
	gamemode.ResetModeConfigCache()
	gamemode.ResetRegistry()
}
