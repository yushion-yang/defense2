// platform_config.go — 平台差异化配置。
//
// 不同发布平台（桌面/Web/移动）可使用不同的全局缩放参数，
// 在不修改核心逻辑的前提下调整游戏体验。
//
// 配置文件位于 config/platform/{platformID}.json，
// platformID 通过 ldflags 注入（默认 "default"）。
package config

import (
	"encoding/json"
	"fmt"
	"log"
)

// PlatformConfig 平台差异化缩放参数。
// 所有字段默认 1.0（无缩放），仅在 JSON 中显式设置的值生效。
type PlatformConfig struct {
	EnemyHPScale        float64 `json:"enemyHPScale"`        // 敌人血量缩放
	EnemySpeedScale     float64 `json:"enemySpeedScale"`     // 敌人速度缩放
	EnemyArmorScale     float64 `json:"enemyArmorScale"`     // 敌人护甲缩放
	EnemyDamageCapScale float64 `json:"enemyDamageCapScale"` // 敌人伤害上限缩放
}

// platformID 通过 ldflags 注入：
//
//	go build -ldflags="-X 'defense2/internal/config.platformID=web'"
//
// 默认 "default"（桌面端，所有缩放 1.0）。
var platformID = "default"

var globalPlatform *PlatformConfig

// GlobalPlatform 返回当前平台配置。LoadPlatform 成功后可用。
func GlobalPlatform() *PlatformConfig {
	if globalPlatform == nil {
		return defaultPlatform()
	}
	return globalPlatform
}

// LoadPlatform 从 config/platform/{platformID}.json 加载平台配置。
// 文件不存在时静默回退到默认值（所有缩放 1.0）。
func LoadPlatform() {
	cfg := defaultPlatform()

	if dataFS == nil {
		globalPlatform = cfg
		return
	}

	path := fmt.Sprintf("config/platform/%s.json", platformID)
	data, err := dataFS.ReadFile(path)
	if err != nil {
		log.Printf("[platform] %s not found, using defaults", path)
		globalPlatform = cfg
		return
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		log.Printf("[platform] parse %s error: %v, using defaults", path, err)
		globalPlatform = defaultPlatform()
		return
	}

	log.Printf("[platform] loaded %s: HP=%.2f Speed=%.2f Armor=%.2f DmgCap=%.2f",
		path, cfg.EnemyHPScale, cfg.EnemySpeedScale, cfg.EnemyArmorScale, cfg.EnemyDamageCapScale)
	globalPlatform = cfg
}

func defaultPlatform() *PlatformConfig {
	return &PlatformConfig{
		EnemyHPScale:        1.0,
		EnemySpeedScale:     1.0,
		EnemyArmorScale:     1.0,
		EnemyDamageCapScale: 1.0,
	}
}
