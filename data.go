// data.go — 嵌入式数据入口。
// 通过 go:embed 将所有配置、PNG 精灵和音频文件编译进二进制。
package defense2

import "embed"

// DataFS 配置文件系统（JSON 配置）。
//
//go:embed config/levels/*.json config/towers/*.json config/enemies/*.json config/events/*.json config/wardens/*.json config/abilities/*.json config/settings.json config/level-list.json
var DataFS embed.FS

// AssetFS 资源文件系统（PNG 精灵 + WAV 音频）。
//
//go:embed assets/towers/*/*.png assets/enemies/*.png assets/wardens/*.png assets/icons/*.png assets/audio/*.wav assets/fonts/*.ttf
var AssetFS embed.FS
