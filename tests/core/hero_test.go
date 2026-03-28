package core_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/hero"
	"defense2/internal/core/projectile"
)

func TestHeroIdleOrbit(t *testing.T) {
	h := hero.DefaultHero(300, 300)
	ep := enemy.NewPool(4)
	pp := projectile.NewPool(16)

	// 无敌人时应保持空闲巡逻
	h.Update(ep, pp, 0.5)
	if h.State != hero.StateIdle {
		t.Fatalf("无敌人时应为 Idle，实际 %d", h.State)
	}
	// 位置应在基地附近轨道上
	dx := h.X - h.BaseX
	dy := h.Y - h.BaseY
	dist := dx*dx + dy*dy
	if dist > h.OrbitRadius*h.OrbitRadius*1.5 {
		t.Fatalf("英雄应在轨道附近，实际距基地 %.0f", dist)
	}
}

func TestHeroEngageEnemy(t *testing.T) {
	h := hero.DefaultHero(300, 300)
	ep := enemy.NewPool(4)
	pp := projectile.NewPool(16)

	// 在交战范围内放一个敌人
	ep.Spawn(400, 300, 50, 60, 8, 1) // 100px 远，在 EngageRange(180) 内

	h.Update(ep, pp, 0.016)
	if h.State != hero.StateEngage {
		t.Fatalf("应切换到 Engage，实际 %d", h.State)
	}
}

func TestHeroReturnOnLeash(t *testing.T) {
	h := hero.DefaultHero(300, 300)
	h.State = hero.StateEngage
	// 手动拉远英雄超过牵引距离
	h.X = 300 + h.LeashRadius + 50
	h.Y = 300

	ep := enemy.NewPool(4)
	ep.Spawn(600, 300, 50, 60, 8, 1) // 敌人在远处
	pp := projectile.NewPool(16)

	h.Update(ep, pp, 0.016)
	if h.State != hero.StateReturn {
		t.Fatalf("超出牵引距离应切换到 Return，实际 %d", h.State)
	}
}

func TestHeroAwardXP(t *testing.T) {
	h := hero.DefaultHero(0, 0)
	h.AwardXP(25) // 超过 20 的升级阈值
	if h.Level != 2 {
		t.Fatalf("应升到 2 级，实际 %d", h.Level)
	}
	if h.Damage <= 15 {
		t.Fatalf("升级后伤害应增加，实际 %.0f", h.Damage)
	}
}
