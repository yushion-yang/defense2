// hero_loader.go — 英雄配置→运行时英雄转换器。
// 从 JSON 配置创建英雄实体，未找到配置时回退到默认值。
package loader

import (
	"log"

	"defense2/internal/config"
	"defense2/internal/core/hero"
)

// LoadHero 从 JSON 配置创建英雄，失败时回退到默认英雄。
func LoadHero(baseX, baseY float64) *hero.Hero {
	cfg, err := config.LoadHeroConfig()
	if err != nil {
		log.Printf("英雄配置加载失败，使用默认: %v", err)
		return hero.DefaultHero(baseX, baseY)
	}

	h := hero.DefaultHero(baseX, baseY)

	// 用 JSON 配置覆盖默认值（非零字段才覆盖）
	if cfg.Damage > 0 {
		h.Damage = cfg.Damage
	}
	if cfg.FireRate > 0 {
		h.FireRate = cfg.FireRate
	}
	if cfg.AttackRange > 0 {
		h.AttackRange = cfg.AttackRange
	}
	if cfg.EngagementRange > 0 {
		h.EngageRange = cfg.EngagementRange
	}
	if cfg.LeashRadius > 0 {
		h.LeashRadius = cfg.LeashRadius
	}
	if cfg.MoveSpeed > 0 {
		h.MoveSpeed = cfg.MoveSpeed
	}
	if cfg.BodyRadius > 0 {
		h.BodyRadius = cfg.BodyRadius
	}

	return h
}
