package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/buff"
)

// ============================================================
// 堆叠模式测试
// ============================================================

func TestStackMode_Override(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("stun", "tower1", 1.0, 2.0))
	bl.AddSimple(buff.NewBuff("stun", "tower2", 1.0, 3.0))

	// Override 模式：新 buff 替换旧的
	if bl.ActiveCount() != 1 {
		t.Errorf("Override模式应只有1个活跃buff，实际%d", bl.ActiveCount())
	}
	val := bl.GetEffective("stun")
	if val != 1.0 {
		t.Errorf("生效值=%.1f, 期望1.0", val)
	}
}

func TestStackMode_Strongest(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("slow", "tower1", 0.3, 5.0))
	bl.AddSimple(buff.NewBuff("slow", "tower2", 0.5, 5.0))

	// Strongest 模式：取最强值
	val := bl.GetEffective("slow")
	if math.Abs(val-0.5) > 1e-9 {
		t.Errorf("slow生效值=%.1f, 期望0.5", val)
	}
}

func TestStackMode_Strongest_Cap(t *testing.T) {
	bl := buff.NewBuffList()
	// 默认 slow cap = 0.8（最多减速80%，速度不低于20%）
	bl.AddSimple(buff.NewBuff("slow", "tower1", 0.9, 5.0))

	val := bl.GetEffective("slow")
	if math.Abs(val-0.8) > 1e-9 {
		t.Errorf("slow应被cap到0.8, 实际%.1f", val)
	}
}

func TestStackMode_Additive(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("speedUp", "aura1", 0.2, 5.0))
	bl.AddSimple(buff.NewBuff("speedUp", "aura2", 0.3, 5.0))

	val := bl.GetEffective("speedUp")
	if math.Abs(val-0.5) > 1e-9 {
		t.Errorf("speedUp生效值=%.1f, 期望0.5", val)
	}
}

func TestStackMode_Additive_Cap(t *testing.T) {
	bl := buff.NewBuffList()
	// speedUp cap = 1.4
	bl.AddSimple(buff.NewBuff("speedUp", "a", 0.8, 5.0))
	bl.AddSimple(buff.NewBuff("speedUp", "b", 0.8, 5.0))

	val := bl.GetEffective("speedUp")
	if math.Abs(val-1.4) > 1e-9 {
		t.Errorf("speedUp应被cap到1.4, 实际%.1f", val)
	}
}

func TestStackMode_Multiplicative(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("damageUp", "skill1", 1.2, 5.0))
	bl.AddSimple(buff.NewBuff("damageUp", "skill2", 1.3, 5.0))

	val := bl.GetEffective("damageUp")
	expected := 1.2 * 1.3
	if math.Abs(val-expected) > 1e-9 {
		t.Errorf("damageUp生效值=%.3f, 期望%.3f", val, expected)
	}
}

func TestStackMode_Multiplicative_Floor(t *testing.T) {
	bl := buff.NewBuffList()
	// damageDown floor = 0.2
	bl.AddSimple(buff.NewBuff("damageDown", "a", 0.1, 5.0))
	bl.AddSimple(buff.NewBuff("damageDown", "b", 0.1, 5.0))

	val := bl.GetEffective("damageDown")
	if math.Abs(val-0.2) > 1e-9 {
		t.Errorf("damageDown应被floor到0.2, 实际%.3f", val)
	}
}

func TestStackMode_Independent(t *testing.T) {
	bl := buff.NewBuffList()
	b1 := buff.NewBuff("shield", "tower1", 100, 10.0)
	b2 := buff.NewBuff("shield", "tower2", 200, 10.0)
	bl.AddSimple(b1)
	bl.AddSimple(b2)

	// Independent 模式：两个 buff 都存在
	if bl.ActiveCount() != 2 {
		t.Errorf("Independent模式应有2个活跃buff，实际%d", bl.ActiveCount())
	}
	val := bl.GetEffective("shield")
	if math.Abs(val-300) > 1e-9 {
		t.Errorf("shield总值=%.0f, 期望300", val)
	}
}

func TestStackMode_IndependentPerSource(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("dot", "tower1", 10, 3.0))
	bl.AddSimple(buff.NewBuff("dot", "tower2", 15, 3.0))

	// 不同来源独立
	if bl.ActiveCount() != 2 {
		t.Errorf("不同来源应有2个dot, 实际%d", bl.ActiveCount())
	}

	// 同来源刷新
	bl.AddSimple(buff.NewBuff("dot", "tower1", 20, 5.0))
	if bl.ActiveCount() != 2 {
		t.Errorf("同来源刷新后应仍有2个, 实际%d", bl.ActiveCount())
	}
}

// ============================================================
// 回调测试
// ============================================================

func TestBuff_OnApplyCallback(t *testing.T) {
	bl := buff.NewBuffList()
	applied := false
	b := buff.NewBuff("stun", "test", 1.0, 2.0)
	b.OnApply = func(target interface{}) { applied = true }

	bl.Add(b, "dummy_target")
	if !applied {
		t.Error("OnApply 回调未触发")
	}
}

func TestBuff_OnExpireCallback(t *testing.T) {
	bl := buff.NewBuffList()
	expired := false
	b := buff.NewBuff("stun", "test", 1.0, 1.0)
	b.OnExpire = func(target interface{}) { expired = true }

	bl.Add(b, nil)
	bl.Tick(2.0, nil) // 超过持续时间

	if !expired {
		t.Error("OnExpire 回调未触发")
	}
}

func TestBuff_OnTickCallback(t *testing.T) {
	bl := buff.NewBuffList()
	tickCount := 0
	b := buff.NewBuff("dot", "test", 10, 5.0)
	b.OnTick = func(target interface{}) { tickCount++ }
	b.TickInterval = 1.0

	bl.Add(b, nil)

	// 模拟3.5秒
	for i := 0; i < 7; i++ {
		bl.Tick(0.5, nil)
	}

	if tickCount != 3 {
		t.Errorf("OnTick 应触发3次（3.5秒/1秒间隔），实际%d", tickCount)
	}
}

// ============================================================
// 操作测试
// ============================================================

func TestBuff_RemoveByID(t *testing.T) {
	bl := buff.NewBuffList()
	b := buff.NewBuff("shield", "test", 100, 10.0)
	bl.AddSimple(b)

	removed := bl.RemoveByID(b.ID, nil)
	if !removed {
		t.Error("RemoveByID 应返回 true")
	}
	if bl.ActiveCount() != 0 {
		t.Errorf("移除后应有0个活跃buff，实际%d", bl.ActiveCount())
	}
}

func TestBuff_RemoveBySource(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("shield", "tower1", 100, 10.0))
	bl.AddSimple(buff.NewBuff("shield", "tower1", 200, 10.0))
	bl.AddSimple(buff.NewBuff("shield", "tower2", 300, 10.0))

	count := bl.RemoveBySource("tower1", nil)
	if count != 2 {
		t.Errorf("应移除2个来自tower1的buff，实际%d", count)
	}
	if bl.ActiveCount() != 1 {
		t.Errorf("移除后应有1个活跃buff，实际%d", bl.ActiveCount())
	}
}

func TestBuff_PurgeDispellable(t *testing.T) {
	bl := buff.NewBuffList()

	b1 := buff.NewBuff("shield", "a", 100, 10.0)
	b1.Dispellable = true
	bl.AddSimple(b1)

	b2 := buff.NewBuff("shield", "b", 200, 10.0)
	b2.Dispellable = false
	bl.AddSimple(b2)

	count := bl.PurgeDispellable(nil, nil)
	if count != 1 {
		t.Errorf("应净化1个可驱散buff，实际%d", count)
	}
	if bl.ActiveCount() != 1 {
		t.Errorf("净化后应有1个不可驱散buff，实际%d", bl.ActiveCount())
	}
}

func TestBuff_ClearAll(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("shield", "a", 100, 10.0))
	bl.AddSimple(buff.NewBuff("stun", "b", 1, 2.0))

	expireCount := 0
	for i := range bl.Buffs {
		bl.Buffs[i].OnExpire = func(target interface{}) { expireCount++ }
	}

	bl.ClearAll(nil)
	if bl.ActiveCount() != 0 {
		t.Errorf("ClearAll后应有0个活跃buff，实际%d", bl.ActiveCount())
	}
	if expireCount != 2 {
		t.Errorf("ClearAll应触发2个OnExpire，实际%d", expireCount)
	}
}

// ============================================================
// 状态查询测试
// ============================================================

func TestBuff_ImmunityChecks(t *testing.T) {
	bl := buff.NewBuffList()

	if bl.IsStunImmune() {
		t.Error("空列表不应有眩晕免疫")
	}

	bl.AddSimple(buff.NewBuff("stunImmune", "passive", 1, -1))
	if !bl.IsStunImmune() {
		t.Error("添加stunImmune后应有眩晕免疫")
	}

	bl2 := buff.NewBuffList()
	bl2.AddSimple(buff.NewBuff("controlImmune", "boss", 1, -1))
	if !bl2.IsStunImmune() {
		t.Error("controlImmune应包含眩晕免疫")
	}
	if !bl2.IsSlowImmune() {
		t.Error("controlImmune应包含减速免疫")
	}
	if !bl2.IsRootImmune() {
		t.Error("controlImmune应包含定身免疫")
	}
}

func TestBuff_Duration(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("stun", "test", 1, 2.0))

	bl.TickSimple(1.0)
	if !bl.HasType("stun") {
		t.Error("1秒后stun应仍然存在")
	}

	bl.TickSimple(1.5)
	if bl.HasType("stun") {
		t.Error("2.5秒后stun应已过期")
	}
}

func TestBuff_Permanent(t *testing.T) {
	bl := buff.NewBuffList()
	bl.AddSimple(buff.NewBuff("slowImmune", "passive", 1, -1)) // duration <= 0 = 永久

	bl.TickSimple(100)
	if !bl.HasType("slowImmune") {
		t.Error("永久buff在100秒后应仍然存在")
	}
}
