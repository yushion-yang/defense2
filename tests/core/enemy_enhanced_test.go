package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
)

// ============================================================
// 控制减免(韧性)测试
// ============================================================

func TestCC_ApplyStun(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, Path: []gamemap.Point{{X: 0, Y: 0}}}
	ok := combat.ApplyStun(e, 2.0, "tower1")
	if !ok {
		t.Error("普通敌人应能被眩晕")
	}
	if math.Abs(e.StunTimer-2.0) > 1e-9 {
		t.Errorf("眩晕时间=%.2f, 期望2.0", e.StunTimer)
	}
}

func TestCC_StunImmune(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{IsStunImmune: true}}
	ok := combat.ApplyStun(e, 2.0, "tower1")
	if ok {
		t.Error("眩晕免疫敌人不应被眩晕")
	}
}

func TestCC_ControlImmune(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{IsControlImmune: true}}
	if combat.ApplyStun(e, 2.0, "") {
		t.Error("控制免疫应阻止眩晕")
	}
	if combat.ApplySlow(e, 0.5, 3.0, "") {
		t.Error("控制免疫应阻止减速")
	}
	// ApplyRoot 已移除（定身 CC 不再使用）
}

func TestCC_Tenacity(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{Tenacity: 0.5}}
	combat.ApplyStun(e, 4.0, "test")
	// 韧性0.5: 实际持续 = 4.0 * (1-0.5) = 2.0
	if math.Abs(e.StunTimer-2.0) > 1e-9 {
		t.Errorf("韧性0.5时眩晕=%.2f, 期望2.0", e.StunTimer)
	}
}

func TestCC_SlowCap(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, BaseSpeed: 100, Speed: 100}
	// factor=0.1 低于 MinSpeedRatio=0.2，应被 clamp 到 0.2
	combat.ApplySlow(e, 0.1, 3.0, "test")
	if math.Abs(e.SlowFactor-0.2) > 1e-9 {
		t.Errorf("减速倍率=%.2f, 期望0.2(MinSpeedRatio)", e.SlowFactor)
	}
	if math.Abs(e.Speed-20) > 1e-9 {
		t.Errorf("速度=%.2f, 期望20(BaseSpeed*MinSpeedRatio)", e.Speed)
	}
}

// ============================================================
// 敌人生命周期测试
// ============================================================

func TestLifecycle_RegisterAndResolve(t *testing.T) {
	e := &enemy.Enemy{ID: 1}
	enemy.InitLifecycle(e)

	called := false
	enemy.RegisterHandler(e, "death", enemy.LifecycleHandler{
		ID: "test-handler",
		Apply: func(en *enemy.Enemy, ctx interface{}) interface{} {
			called = true
			return "ok"
		},
	})

	results := enemy.ResolveEvent(e, "death", nil)
	if !called {
		t.Error("onDeath handler 应被调用")
	}
	if len(results) != 1 || results[0] != "ok" {
		t.Errorf("结果=%v, 期望[ok]", results)
	}
}

func TestLifecycle_DeduplicateByID(t *testing.T) {
	e := &enemy.Enemy{ID: 1}
	enemy.InitLifecycle(e)

	count := 0
	handler := enemy.LifecycleHandler{
		ID: "dup",
		Apply: func(en *enemy.Enemy, ctx interface{}) interface{} {
			count++
			return nil
		},
	}
	enemy.RegisterHandler(e, "death", handler)
	enemy.RegisterHandler(e, "death", handler) // 重复注册

	enemy.ResolveEvent(e, "death", nil)
	if count != 1 {
		t.Errorf("去重后应只调用1次, 实际%d", count)
	}
}

func TestLifecycle_RemoveHandler(t *testing.T) {
	e := &enemy.Enemy{ID: 1}
	enemy.InitLifecycle(e)

	enemy.RegisterHandler(e, "death", enemy.LifecycleHandler{
		ID:    "removable",
		Apply: func(en *enemy.Enemy, ctx interface{}) interface{} { return nil },
	})
	enemy.RemoveHandler(e, "death", "removable")

	results := enemy.ResolveEvent(e, "death", nil)
	if len(results) != 0 {
		t.Errorf("移除后应无结果, 实际%d", len(results))
	}
}

// ============================================================
// 敌人行为测试
// ============================================================

func TestBehavior_Berserk(t *testing.T) {
	e := &enemy.Enemy{
		HP: 60, MaxHP: 100,
		BaseSpeed: 100, Speed: 100,
		BerserkThreshold: 0.5, BerserkSpeedScale: 1.5,
	}
	// 60% HP, 不触发
	triggered := enemy.UpdateBerserk(e)
	if triggered {
		t.Error("60%HP不应触发狂暴")
	}

	// 降到40%
	e.HP = 40
	triggered = enemy.UpdateBerserk(e)
	if !triggered {
		t.Error("40%HP应触发狂暴(阈值50%)")
	}
	if math.Abs(e.BaseSpeed-150) > 1e-9 {
		t.Errorf("狂暴后BaseSpeed=%.1f, 期望150", e.BaseSpeed)
	}
	if !e.BerserkTriggered {
		t.Error("应标记为已触发")
	}

	// 再次调用不重复触发
	e.HP = 10
	triggered = enemy.UpdateBerserk(e)
	if triggered {
		t.Error("已触发后不应重复")
	}
}

func TestBehavior_Regeneration(t *testing.T) {
	e := &enemy.Enemy{HP: 50, MaxHP: 100, RegenPerSec: 10}
	healed := enemy.UpdateRegeneration(e, 1.0)
	if math.Abs(healed-10) > 1e-9 {
		t.Errorf("回血量=%.1f, 期望10", healed)
	}
	if math.Abs(e.HP-60) > 1e-9 {
		t.Errorf("回血后HP=%.1f, 期望60", e.HP)
	}

	// 不超过 MaxHP
	e.HP = 95
	healed = enemy.UpdateRegeneration(e, 1.0)
	if e.HP > e.MaxHP {
		t.Errorf("HP=%.1f 超过MaxHP=%.1f", e.HP, e.MaxHP)
	}
}

// ============================================================
// 敌人事件测试
// ============================================================

func TestEvent_HPPercent(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100}
	enemy.ApplyEnemyEvent(e, "hpPercent", 0.3)
	if math.Abs(e.MaxHP-130) > 1e-9 {
		t.Errorf("hpPercent后MaxHP=%.1f, 期望130", e.MaxHP)
	}
	if math.Abs(e.HP-130) > 1e-9 {
		t.Errorf("hpPercent后HP=%.1f, 期望130", e.HP)
	}
}

// ============================================================
// Buff模板测试
// ============================================================

func TestBuffTemplate_ApplyBerserk(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100}
	ok := enemy.ApplyBuffTemplate(e, "berserk")
	if !ok {
		t.Error("berserk模板应用应成功")
	}
	if e.BerserkThreshold <= 0 {
		t.Error("应设置BerserkThreshold")
	}
	if e.BerserkSpeedScale <= 1 {
		t.Error("应设置BerserkSpeedScale > 1")
	}
}

func TestBuffTemplate_ApplyFlags_Elite(t *testing.T) {
	// "elite" flag was removed from ApplyFlags — it is now a no-op.
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Speed: 50, BaseSpeed: 50, Reward: 10}
	enemy.ApplyFlags(e, []string{"elite"})
	if math.Abs(e.MaxHP-100) > 1e-9 {
		t.Errorf("elite flag (removed) MaxHP=%.1f, 期望100(不变)", e.MaxHP)
	}
	if e.Reward != 10 {
		t.Errorf("elite flag (removed) Reward=%d, 期望10(不变)", e.Reward)
	}
}

func TestBuffTemplate_ApplyFlags_Boss(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Reward: 10}
	enemy.ApplyFlags(e, []string{"boss"})
	if math.Abs(e.MaxHP-3000) > 1e-9 {
		t.Errorf("boss flag MaxHP=%.1f, 期望3000", e.MaxHP)
	}
	if e.Reward != 50 {
		t.Errorf("boss flag Reward=%d, 期望50", e.Reward)
	}
	if !e.Boss {
		t.Error("boss flag应设置Boss=true")
	}
}

func TestBuffTemplate_MapLegacyType(t *testing.T) {
	arch, flags, buffs := enemy.MapLegacyType("berserker")
	if arch == "" {
		t.Error("berserker应映射到有效原型")
	}
	_ = flags
	_ = buffs
	// 未知类型降级到 normal
	arch2, _, _ := enemy.MapLegacyType("unknown_type_xyz")
	if arch2 != "normal" {
		t.Errorf("未知类型应降级为normal, 实际=%s", arch2)
	}
}
