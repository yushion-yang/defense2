package core_test

import (
	"testing"

	_ "defense2/internal/core/warden/types" // 注册 envoy 行为

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

func TestWardenCalcStrength(t *testing.T) {
	w := warden.NewWarden(1, "测试金灵", "envoy")
	w.SelfStrength = 10

	tp := tower.NewPool(4)
	def := tower.TowerDef{Key: "basic", Label: "Arrow", Range: 150, Damage: 10, AttackSpeed: 1.5, Cost: 50}
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
	initial := w.SelfStrength
	killGrowth := w.GrowthOnKill
	waveGrowth := w.GrowthOnWaveClear
	w.OnKill()
	w.OnKill()
	w.OnWaveClear()
	expected := initial + killGrowth*2 + waveGrowth
	if w.SelfStrength != expected {
		t.Fatalf("预期自身强度 %.0f (%.0f+%.0f*2+%.0f)，实际 %.0f",
			expected, initial, killGrowth, waveGrowth, w.SelfStrength)
	}
}

func TestEnvoyBehaviorInit(t *testing.T) {
	w := warden.NewWarden(1, "金灵", "envoy")
	if w.State == nil {
		t.Fatal("envoy 应初始化 State")
	}
}

func TestEnvoyBuff(t *testing.T) {
	w := warden.NewWarden(1, "金灵", "envoy")
	// buff = max(0, strength - 100)，需要强度 > 100 才生效
	w.SelfStrength = 150 // 感知强度 = 150，buff = 50

	tp := tower.NewPool(4)
	def := tower.TowerDef{Key: "t", Label: "T", Damage: 20, Range: 200, AttackSpeed: 1, Cost: 50}
	placed := tp.Place(0, 0, 100, 100, def)

	ep := enemy.NewPool(4)
	ep.Spawn(150, 100, 50, 60, 1, "normal", nil) // 在塔范围内

	pp := projectile.NewPool(32)
	ctx := &warden.TickContext{
		Enemies:     ep,
		Towers:      tp,
		Projectiles: pp,
		DT:          1.0, // 大步长加速倒计时
	}

	// BuffInterval=5s, 多次 tick 直到 buff 触发
	for i := 0; i < 6; i++ {
		w.Tick(ctx)
	}

	// buff 施加后塔应有 StrengthData + 临时加成
	if placed.Strength == nil {
		t.Fatal("buff 施加后塔应有 StrengthData")
	}
	// buff 应已生效：塔战力 > 基础值 100（具体值因 CalcStrength 反馈循环会递增）
	if placed.Strength.Effective() <= 100 {
		t.Fatalf("buff 后塔战力应 > 100，实际 %.0f", placed.Strength.Effective())
	}
}
