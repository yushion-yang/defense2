package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/tower"
)

// ============================================================
// 分支特化测试
// ============================================================

func TestBranch_Apply(t *testing.T) {
	tw := &tower.Tower{Damage: 100, Range: 200, BaseDamage: 100, BaseRange: 200}
	ok := tower.ApplyBranch(tw, "damage", tower.BranchConfig{DamageBonus: 0.5})
	if !ok {
		t.Error("首次分支应成功")
	}
	if tw.Branch != "damage" {
		t.Errorf("Branch=%s, 期望damage", tw.Branch)
	}
	// 伤害 +50%
	if math.Abs(tw.Damage-150) > 1e-9 {
		t.Errorf("伤害=%.1f, 期望150", tw.Damage)
	}
}

func TestBranch_OnlyOnce(t *testing.T) {
	tw := &tower.Tower{Damage: 100, Range: 200, Branch: "damage"}
	ok := tower.ApplyBranch(tw, "range", tower.BranchConfig{RangeBonus: 0.3})
	if ok {
		t.Error("已分支的塔不应再次分支")
	}
}

func TestBranch_RangeBonus(t *testing.T) {
	tw := &tower.Tower{Range: 200, BaseRange: 200}
	tower.ApplyBranch(tw, "range", tower.BranchConfig{RangeBonus: 0.3})
	// 200 * 1.3 = 260
	if math.Abs(tw.Range-260) > 1e-9 {
		t.Errorf("射程=%.1f, 期望260", tw.Range)
	}
}

// ============================================================
// 塔生命周期钩子测试
// ============================================================

func TestLifecycle_CreateHook(t *testing.T) {
	tower.ResetHooks()
	called := false
	tower.OnTowerCreate(func(tw *tower.Tower) { called = true })

	tw := &tower.Tower{Key: "test"}
	tower.ExecuteCreate(tw)
	if !called {
		t.Error("Create钩子应被调用")
	}
}

func TestLifecycle_DestroyHook(t *testing.T) {
	tower.ResetHooks()
	called := false
	tower.OnTowerDestroy(func(tw *tower.Tower) { called = true })

	tw := &tower.Tower{Key: "test"}
	tower.ExecuteDestroy(tw)
	if !called {
		t.Error("Destroy钩子应被调用")
	}
}

func TestLifecycle_UpgradeHook(t *testing.T) {
	tower.ResetHooks()
	called := false
	tower.OnTowerUpgrade(func(tw *tower.Tower) { called = true })

	tw := &tower.Tower{Key: "test"}
	tower.ExecuteUpgrade(tw)
	if !called {
		t.Error("Upgrade钩子应被调用")
	}
}

func TestLifecycle_PanicRecovery(t *testing.T) {
	tower.ResetHooks()
	tower.OnTowerCreate(func(tw *tower.Tower) { panic("boom") })

	secondCalled := false
	tower.OnTowerCreate(func(tw *tower.Tower) { secondCalled = true })

	tw := &tower.Tower{Key: "test"}
	// 不应panic
	tower.ExecuteCreate(tw)
	if !secondCalled {
		t.Error("panic不应阻止后续钩子执行")
	}
}

// ============================================================
// 冷却-通道状态机测试
// ============================================================

func TestCooldownChannel_Phases(t *testing.T) {
	state := tower.InitCooldownChannel()
	tickCount := 0

	cfg := &tower.CooldownChannelConfig{
		Cooldown:    2.0,
		Duration:    1.0,
		TickRate:    0.5,
		CanActivate: func() bool { return true },
		OnActivate:  func() {},
		OnTick:      func() interface{} { tickCount++; return tickCount },
	}

	// 冷却阶段：1秒后还没就绪
	r1 := tower.TickCooldownChannel(state, cfg, 1.0)
	if r1.Active {
		t.Error("1秒时不应激活(冷却2秒)")
	}
	if r1.Progress > 0.6 {
		t.Errorf("1秒进度=%.2f, 应约0.5", r1.Progress)
	}

	// 冷却完成，进入通道
	r2 := tower.TickCooldownChannel(state, cfg, 1.5)
	if !r2.Active {
		t.Error("2.5秒时应进入通道阶段")
	}

	// 通道中tick
	r3 := tower.TickCooldownChannel(state, cfg, 0.6)
	if len(r3.Ticks) == 0 {
		t.Error("通道中0.6秒应有tick结果")
	}

	// 通道结束
	r4 := tower.TickCooldownChannel(state, cfg, 1.0)
	if r4.Active {
		t.Error("通道持续1秒后应结束")
	}
}

// ============================================================
// 维度元数据测试
// ============================================================

func TestDimensionMeta_ScaleMultiply(t *testing.T) {
	val := tower.ScaleValue("radius", 50, 1.5)
	// multiply: 50 * 1.5 = 75
	if math.Abs(val-75) > 1e-9 {
		t.Errorf("radius缩放=%.1f, 期望75", val)
	}
}

func TestDimensionMeta_ScaleAddCapped(t *testing.T) {
	val := tower.ScaleValue("chance", 0.3, 3.0)
	// add_capped: min(0.3 * 3.0, cap=0.6) = 0.6
	if math.Abs(val-0.6) > 1e-9 {
		t.Errorf("chance缩放=%.2f, 期望0.6(cap)", val)
	}
}

func TestDimensionMeta_ScaleInverse(t *testing.T) {
	val := tower.ScaleValue("cooldown", 10, 2.0)
	// inverse: 10 / 2.0 = 5, floor=0.5
	if math.Abs(val-5) > 1e-9 {
		t.Errorf("cooldown缩放=%.1f, 期望5", val)
	}

	val2 := tower.ScaleValue("cooldown", 0.8, 100)
	// 0.8/100 = 0.008, floored to 0.5
	if math.Abs(val2-0.5) > 1e-9 {
		t.Errorf("cooldown下限=%.2f, 期望0.5", val2)
	}
}

func TestDimensionMeta_ScaleNone(t *testing.T) {
	val := tower.ScaleValue("type", 42, 999)
	// none: 原值不变
	if math.Abs(val-42) > 1e-9 {
		t.Errorf("type不应缩放, 值=%.1f, 期望42", val)
	}
}

func TestDimensionMeta_ScaleAbility(t *testing.T) {
	ab := map[string]float64{
		"radius":   50,
		"cooldown": 10,
		"type":     1,
	}
	scaled := tower.ScaleAbility(ab, 2.0)

	if math.Abs(scaled["radius"]-100) > 1e-9 {
		t.Errorf("radius=%.1f, 期望100", scaled["radius"])
	}
	if math.Abs(scaled["cooldown"]-5) > 1e-9 {
		t.Errorf("cooldown=%.1f, 期望5", scaled["cooldown"])
	}
	if math.Abs(scaled["type"]-1) > 1e-9 {
		t.Errorf("type=%.1f, 期望1(不变)", scaled["type"])
	}

	// 原始不应被修改
	if ab["radius"] != 50 {
		t.Error("原始map不应被修改")
	}
}

// ============================================================
// 关键词查询测试
// ============================================================

func TestKeywordQuery_HasAbility(t *testing.T) {
	abilities := []string{"splash", "crit", "bounce"}
	if !tower.TowerHasAbility(abilities, "splash") {
		t.Error("应找到splash")
	}
	if tower.TowerHasAbility(abilities, "stun") {
		t.Error("不应找到stun")
	}
}

func TestKeywordQuery_GetAbility(t *testing.T) {
	abilities := []string{"splash", "crit"}
	entry := tower.GetTowerAbility(abilities, "splash")
	if entry == nil {
		t.Fatal("应找到splash条目")
	}
	if entry.Type != "splash" {
		t.Errorf("Type=%s, 期望splash", entry.Type)
	}
}

func TestKeywordQuery_GetAll(t *testing.T) {
	abilities := []string{"splash", "crit"}
	legacy := map[string]interface{}{
		"bounceConfig": map[string]interface{}{"maxBounces": 3},
	}
	all := tower.GetTowerAbilities(abilities, legacy)
	// splash + crit + bounce = 3
	if len(all) != 3 {
		t.Errorf("总能力数=%d, 期望3", len(all))
	}

	// 检查去重
	abilities2 := []string{"splash", "bounce"}
	all2 := tower.GetTowerAbilities(abilities2, legacy)
	if len(all2) != 2 {
		t.Errorf("去重后=%d, 期望2", len(all2))
	}
}
