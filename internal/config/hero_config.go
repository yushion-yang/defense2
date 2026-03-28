// hero_config.go — 英雄配置加载。
// 从 hero.json 加载英雄基础属性。
package config

import (
	"encoding/json"
	"fmt"
)

// HeroJSON 英雄配置的 JSON 原始结构。
type HeroJSON struct {
	Label            string  `json:"label"`            // 显示名称
	BodyRadius       float64 `json:"bodyRadius"`       // 碰撞半径
	AuraRadius       float64 `json:"auraRadius"`       // 光环半径
	LeashRadius      float64 `json:"leashRadius"`      // 牵引半径
	MoveSpeed        float64 `json:"moveSpeed"`        // 移动速度
	Damage           float64 `json:"damage"`           // 基础伤害
	FireRate         float64 `json:"fireRate"`          // 射击间隔（秒/次）
	AttackRange      float64 `json:"attackRange"`      // 攻击范围
	EngagementRange  float64 `json:"engagementRange"`  // 交战触发范围
	PreferredDistance float64 `json:"preferredDistance"` // 首选战斗距离
	ProjectileSpeed  float64 `json:"projectileSpeed"`  // 弹射物速度
	MaxLevel         int     `json:"maxLevel"`          // 最高等级
	XPPerKill        int     `json:"xpPerKill"`         // 击杀经验
	XPPerWave        int     `json:"xpPerWave"`         // 通波经验
}

// LoadHeroConfig 加载英雄配置。
func LoadHeroConfig() (*HeroJSON, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load hero: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/hero/hero.json")
	if err != nil {
		return nil, fmt.Errorf("load hero: %w", err)
	}
	var h HeroJSON
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parse hero: %w", err)
	}
	return &h, nil
}
