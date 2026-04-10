// vfx_config.go — VFX 特效目录配置加载。
// 从 config/visuals/vfx.json 加载特效元数据（名称、中文标签、描述），
// 供 VFX Preview 场景和其他需要遍历特效的系统使用。
package config

import (
	"encoding/json"
	"fmt"
)

// VFXEffect 单个特效条目。
type VFXEffect struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// VFXCategory 特效类别。
type VFXCategory struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Label   string      `json:"label"`
	Effects []VFXEffect `json:"effects"`
}

// VFXCatalog 完整特效目录。
type VFXCatalog struct {
	Categories []VFXCategory `json:"categories"`
}

var globalVFXCatalog *VFXCatalog

// GlobalVFXCatalog 返回全局 VFX 目录。LoadVFXCatalog 成功后可用。
func GlobalVFXCatalog() *VFXCatalog {
	return globalVFXCatalog
}

// LoadVFXCatalog 从 config/visuals/vfx.json 加载特效目录。
func LoadVFXCatalog() (*VFXCatalog, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load vfx catalog: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/visuals/vfx.json")
	if err != nil {
		return nil, fmt.Errorf("load vfx catalog: %w", err)
	}

	var cat VFXCatalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parse vfx catalog: %w", err)
	}
	globalVFXCatalog = &cat
	return &cat, nil
}
