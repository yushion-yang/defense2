// loader.go — 配置加载器。
// 从嵌入式文件系统读取 JSON 配置并反序列化为 Go 结构体。
package config

import (
	"embed"
	"encoding/json"
	"fmt"
)

// dataFS 持有嵌入式配置文件系统的引用。
var dataFS *embed.FS

// assetFS 持有嵌入式资源文件系统的引用（SVG、音频等）。
var assetFS *embed.FS

// SetDataFS 注入配置文件系统。
func SetDataFS(fs *embed.FS) {
	dataFS = fs
}

// SetAssetFS 注入资源文件系统。
func SetAssetFS(fs *embed.FS) {
	assetFS = fs
}

// GetAssetFS 返回资源文件系统引用。
func GetAssetFS() *embed.FS {
	return assetFS
}

// LoadMap 按地图 ID（如 "map_01"）加载地图配置。
func LoadMap(id string) (*MapConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load map %s: dataFS not initialized", id)
	}
	path := fmt.Sprintf("config/levels/%s.json", id)
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load map %s: %w", id, err)
	}
	var m MapConfig
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse map %s: %w", id, err)
	}
	return &m, nil
}
