package core_test

import (
	"testing"

	_ "defense2/internal/core/warden/types" // 注册 envoy 行为

	"defense2/internal/core/enemy"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

func TestWardenCalcStrength(t *testing.T) {
	w := warden.NewWarden(1, "测试使者", "envoy")
	w.SelfStrength = 10

	tp := tower.NewPool(4)
	def := tower.BaseTowerDefs()[0]
	tp.Place(0, 0, 100, 100, def)
	// 给塔设置默认战力(100)，overflow = max(0, 100-100) = 0
	tp.Each(func(t2 *tower.Tower) {
		t2.Strength = strength.NewStrengthData()
	})

	w.CalcStrength(tp)
	if w.PerceivedStrength != 10 {
		t.Fatalf("预期强度 10 (selfStrength + 0 overflow)，实际 %.0f", w.PerceivedStrength)
	}

	// 放一座有高战力的塔（战力150，overflow = 50）
	highDef := tower.TowerDef{Key: "test", Label: "T", Damage: 30, Range: 100, AttackSpeed: 1, Cost: 50}
	tp.Place(1, 1, 200, 200, highDef)
	tp.Each(func(t2 *tower.Tower) {
		if t2.Key == "test" {
			sd := strength.NewStrengthData()
			sd.AddPermanent(50) // effective = 150, overflow = 50
			t2.Strength = sd
		}
	})

	w.CalcStrength(tp)
	// selfStrength(10) + overflow(0 + 50) = 60
	if w.PerceivedStrength != 60 {
		t.Fatalf("预期强度 60 (10 + 0 + 50)，实际 %.0f", w.PerceivedStrength)
	}
}

func TestWardenOnKillAndWave(t *testing.T) {
	w := warden.NewWarden(1, "测试", "envoy")
	w.OnKill()
	w.OnKill()
	w.OnWaveClear()
	if w.SelfStrength != 9 { // 2*2 + 5
		t.Fatalf("预期自身强度 9，实际 %.0f", w.SelfStrength)
	}
}

func TestEnvoyBehaviorInit(t *testing.T) {
	w := warden.NewWarden(1, "使者", "envoy")
	if w.State == nil {
		t.Fatal("envoy 应初始化 State")
	}
}

func TestEnvoyPossess(t *testing.T) {
	w := warden.NewWarden(1, "使者", "envoy")

	tp := tower.NewPool(4)
	def := tower.TowerDef{Key: "t", Label: "T", Damage: 20, Range: 200, AttackSpeed: 1, Cost: 50}
	placed := tp.Place(0, 0, 100, 100, def)

	ep := enemy.NewPool(4)
	ep.Spawn(150, 100, 50, 60, 1, "normal", nil) // 在塔范围内

	ctx := &warden.TickContext{
		Enemies: ep,
		Towers:  tp,
		DT:      0.016,
	}

	// 第一次 tick：应附身到塔
	w.Tick(ctx)

	if placed.Damage != 25 { // 20 + 5 (DamageBonus)
		t.Fatalf("附身后塔伤害应为 25，实际 %.0f", placed.Damage)
	}
}
