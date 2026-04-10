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
	rules := map[string]buff.StackRule{
		"stun": {Mode: buff.Override},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "stun", Source: "tower1", Value: 1.0, Duration: 2.0, Remaining: 2.0})
	bl.Add(buff.Buff{ID: "stun", Source: "tower2", Value: 1.0, Duration: 3.0, Remaining: 3.0})

	// Override 模式：新 buff 替换旧的
	if bl.Count() != 1 {
		t.Errorf("Override模式应只有1个活跃buff，实际%d", bl.Count())
	}
	b, ok := bl.Get("stun")
	if !ok {
		t.Fatal("应能获取到 stun buff")
	}
	if b.Value != 1.0 {
		t.Errorf("生效值=%.1f, 期望1.0", b.Value)
	}
	if b.Remaining != 3.0 {
		t.Errorf("Remaining=%.1f, 期望3.0（最新覆盖）", b.Remaining)
	}
}

func TestStackMode_Strongest(t *testing.T) {
	rules := map[string]buff.StackRule{
		"slow": {Mode: buff.Strongest, Cap: 0.8},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "slow", Source: "tower1", Value: 0.3, Duration: 5.0, Remaining: 5.0})
	bl.Add(buff.Buff{ID: "slow", Source: "tower2", Value: 0.5, Duration: 5.0, Remaining: 5.0})

	// Strongest 模式：取最高 Value（0.5 > 0.3）
	b, ok := bl.Get("slow")
	if !ok {
		t.Fatal("应能获取到 slow buff")
	}
	if math.Abs(b.Value-0.5) > 1e-9 {
		t.Errorf("slow生效值=%.1f, 期望0.5", b.Value)
	}
}

func TestStackMode_Strongest_Cap(t *testing.T) {
	rules := map[string]buff.StackRule{
		"slow": {Mode: buff.Strongest, Cap: 0.8},
	}
	bl := buff.NewBuffList(rules)
	// Value 0.9 exceeds cap 0.8
	bl.Add(buff.Buff{ID: "slow", Source: "tower1", Value: 0.9, Duration: 5.0, Remaining: 5.0})

	b, ok := bl.Get("slow")
	if !ok {
		t.Fatal("应能获取到 slow buff")
	}
	if math.Abs(b.Value-0.8) > 1e-9 {
		t.Errorf("slow应被cap到0.8, 实际%.1f", b.Value)
	}
}

func TestStackMode_Additive(t *testing.T) {
	rules := map[string]buff.StackRule{
		"speedUp": {Mode: buff.Additive, Cap: 1.4},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "speedUp", Source: "aura1", Value: 0.2, Duration: 5.0, Remaining: 5.0})
	bl.Add(buff.Buff{ID: "speedUp", Source: "aura2", Value: 0.3, Duration: 5.0, Remaining: 5.0})

	b, ok := bl.Get("speedUp")
	if !ok {
		t.Fatal("应能获取到 speedUp buff")
	}
	if math.Abs(b.Value-0.5) > 1e-9 {
		t.Errorf("speedUp生效值=%.1f, 期望0.5", b.Value)
	}
}

func TestStackMode_Additive_Cap(t *testing.T) {
	rules := map[string]buff.StackRule{
		"speedUp": {Mode: buff.Additive, Cap: 1.4},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "speedUp", Source: "a", Value: 0.8, Duration: 5.0, Remaining: 5.0})
	bl.Add(buff.Buff{ID: "speedUp", Source: "b", Value: 0.8, Duration: 5.0, Remaining: 5.0})

	b, ok := bl.Get("speedUp")
	if !ok {
		t.Fatal("应能获取到 speedUp buff")
	}
	if math.Abs(b.Value-1.4) > 1e-9 {
		t.Errorf("speedUp应被cap到1.4, 实际%.1f", b.Value)
	}
}

func TestStackMode_Multiplicative(t *testing.T) {
	rules := map[string]buff.StackRule{
		"damageDown": {Mode: buff.Multiplicative, Floor: 0.2},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "damageDown", Source: "a", Value: 0.5, Duration: 5.0, Remaining: 5.0})
	bl.Add(buff.Buff{ID: "damageDown", Source: "b", Value: 0.6, Duration: 5.0, Remaining: 5.0})

	b, ok := bl.Get("damageDown")
	if !ok {
		t.Fatal("应能获取到 damageDown buff")
	}
	expected := 0.5 * 0.6 // Multiplicative: 0.3
	if math.Abs(b.Value-expected) > 1e-9 {
		t.Errorf("damageDown生效值=%.3f, 期望%.3f", b.Value, expected)
	}
}

func TestStackMode_Multiplicative_Floor(t *testing.T) {
	rules := map[string]buff.StackRule{
		"damageDown": {Mode: buff.Multiplicative, Floor: 0.2},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "damageDown", Source: "a", Value: 0.1, Duration: 5.0, Remaining: 5.0})
	bl.Add(buff.Buff{ID: "damageDown", Source: "b", Value: 0.1, Duration: 5.0, Remaining: 5.0})

	b, ok := bl.Get("damageDown")
	if !ok {
		t.Fatal("应能获取到 damageDown buff")
	}
	// 0.1 * 0.1 = 0.01 → floored to 0.2
	if math.Abs(b.Value-0.2) > 1e-9 {
		t.Errorf("damageDown应被floor到0.2, 实际%.3f", b.Value)
	}
}

func TestStackMode_Independent(t *testing.T) {
	rules := map[string]buff.StackRule{
		"shield": {Mode: buff.Independent},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "shield", Source: "tower1", Value: 100, Duration: 10.0, Remaining: 10.0})
	bl.Add(buff.Buff{ID: "shield", Source: "tower2", Value: 200, Duration: 10.0, Remaining: 10.0})

	// Independent 模式：两个 buff 都存在
	if bl.Count() != 2 {
		t.Errorf("Independent模式应有2个活跃buff，实际%d", bl.Count())
	}
}

func TestStackMode_IndependentPerSource(t *testing.T) {
	rules := map[string]buff.StackRule{
		"dot": {Mode: buff.IndependentPerSource},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "dot", Source: "tower1", Value: 10, Duration: 3.0, Remaining: 3.0})
	bl.Add(buff.Buff{ID: "dot", Source: "tower2", Value: 15, Duration: 3.0, Remaining: 3.0})

	// 不同来源独立
	if bl.Count() != 2 {
		t.Errorf("不同来源应有2个dot, 实际%d", bl.Count())
	}

	// 同来源刷新
	bl.Add(buff.Buff{ID: "dot", Source: "tower1", Value: 20, Duration: 5.0, Remaining: 5.0})
	if bl.Count() != 2 {
		t.Errorf("同来源刷新后应仍有2个, 实际%d", bl.Count())
	}
}

// ============================================================
// 操作测试
// ============================================================

func TestBuff_RemoveByID(t *testing.T) {
	bl := buff.NewBuffList(map[string]buff.StackRule{})
	bl.Add(buff.Buff{ID: "shield", Source: "test", Value: 100, Duration: 10.0, Remaining: 10.0})

	bl.RemoveByID("shield")
	if bl.Count() != 0 {
		t.Errorf("移除后应有0个活跃buff，实际%d", bl.Count())
	}
}

func TestBuff_RemoveBySource(t *testing.T) {
	rules := map[string]buff.StackRule{
		"shield": {Mode: buff.Independent},
	}
	bl := buff.NewBuffList(rules)
	bl.Add(buff.Buff{ID: "shield", Source: "tower1", Value: 100, Duration: 10.0, Remaining: 10.0})
	bl.Add(buff.Buff{ID: "shield", Source: "tower1", Value: 200, Duration: 10.0, Remaining: 10.0})
	bl.Add(buff.Buff{ID: "shield", Source: "tower2", Value: 300, Duration: 10.0, Remaining: 10.0})

	bl.Remove("shield", "tower1")
	// Remove removes all with matching ID+Source
	if bl.Count() != 1 {
		t.Errorf("移除后应有1个活跃buff，实际%d", bl.Count())
	}
}

func TestBuff_ClearByCategory(t *testing.T) {
	bl := buff.NewBuffList(map[string]buff.StackRule{})
	bl.Add(buff.Buff{ID: "stun", Category: buff.CatCC, Source: "a", Value: 1, Duration: 2.0, Remaining: 2.0})
	bl.Add(buff.Buff{ID: "bleed", Category: buff.CatDoT, Source: "b", Value: 10, Duration: 3.0, Remaining: 3.0})

	bl.ClearByCategory(buff.CatCC)
	if bl.Count() != 1 {
		t.Errorf("清除CC后应有1个buff，实际%d", bl.Count())
	}
	if !bl.Has("bleed") {
		t.Error("DoT buff 应保留")
	}
}

func TestBuff_ClearAll(t *testing.T) {
	bl := buff.NewBuffList(map[string]buff.StackRule{})
	bl.Add(buff.Buff{ID: "shield", Source: "a", Value: 100, Duration: 10.0, Remaining: 10.0})
	bl.Add(buff.Buff{ID: "stun", Source: "b", Value: 1, Duration: 2.0, Remaining: 2.0})

	bl.Clear()
	if bl.Count() != 0 {
		t.Errorf("Clear后应有0个活跃buff，实际%d", bl.Count())
	}
}

// ============================================================
// 状态查询测试
// ============================================================

func TestBuff_HasAndGet(t *testing.T) {
	bl := buff.NewBuffList(map[string]buff.StackRule{})

	if bl.Has("stun") {
		t.Error("空列表不应有 stun")
	}

	bl.Add(buff.Buff{ID: "stun", Source: "test", Value: 1, Duration: 2.0, Remaining: 2.0})
	if !bl.Has("stun") {
		t.Error("添加后应有 stun")
	}

	b, ok := bl.Get("stun")
	if !ok {
		t.Error("Get 应返回 true")
	}
	if b.Value != 1 {
		t.Errorf("Value=%.1f, 期望1.0", b.Value)
	}
}

func TestBuff_Duration(t *testing.T) {
	bl := buff.NewBuffList(map[string]buff.StackRule{})
	bl.Add(buff.Buff{ID: "stun", Source: "test", Value: 1, Duration: 2.0, Remaining: 2.0})

	bl.Tick(1.0)
	if !bl.Has("stun") {
		t.Error("1秒后stun应仍然存在")
	}

	bl.Tick(1.5)
	if bl.Has("stun") {
		t.Error("2.5秒后stun应已过期")
	}
}

func TestBuff_Permanent(t *testing.T) {
	bl := buff.NewBuffList(map[string]buff.StackRule{})
	bl.Add(buff.Buff{ID: "slowImmune", Source: "passive", Value: 1, Duration: -1, Remaining: -1})

	bl.Tick(100)
	if !bl.Has("slowImmune") {
		t.Error("永久buff在100秒后应仍然存在")
	}
}

// ============================================================
// SumByID 测试
// ============================================================

func TestBuffList_SumByID(t *testing.T) {
	rules := map[string]buff.StackRule{
		"aura:damage": {Mode: buff.Additive, Cap: 100},
	}
	bl := buff.NewBuffList(rules)

	// Empty → 0
	if got := bl.SumByID("aura:damage"); got != 0 {
		t.Errorf("empty SumByID = %f, want 0", got)
	}

	// Single buff
	bl.Add(buff.Buff{ID: "aura:damage", Value: 10, Duration: 1, Remaining: 1})
	if got := bl.SumByID("aura:damage"); got != 10 {
		t.Errorf("single SumByID = %f, want 10", got)
	}

	// Multiple additive
	bl.Add(buff.Buff{ID: "aura:damage", Value: 20, Duration: 1, Remaining: 1, Source: "b"})
	if got := bl.SumByID("aura:damage"); got != 30 {
		t.Errorf("multi SumByID = %f, want 30", got)
	}

	// Respects cap
	bl.Add(buff.Buff{ID: "aura:damage", Value: 80, Duration: 1, Remaining: 1, Source: "c"})
	if got := bl.SumByID("aura:damage"); got != 100 {
		t.Errorf("capped SumByID = %f, want 100 (cap)", got)
	}

	// Unmatched ID
	if got := bl.SumByID("nonexistent"); got != 0 {
		t.Errorf("unmatched SumByID = %f, want 0", got)
	}
}

// ============================================================
// DoT Tick 测试
// ============================================================

func TestBuff_TickDoT(t *testing.T) {
	bl := buff.NewBuffList(map[string]buff.StackRule{})
	// bleed: 10 DPS
	bl.Add(buff.Buff{ID: "bleed", Category: buff.CatDoT, Source: "t1", Value: 10, Duration: 3.0, Remaining: 3.0})

	dotInterval := 0.5

	// 0.3s: no tick
	dmg := bl.TickDoT(0.3, dotInterval)
	if dmg != 0 {
		t.Errorf("0.3s 不应触发 tick, dmg=%.1f", dmg)
	}

	// 0.2s more (total 0.5s): tick fires
	dmg = bl.TickDoT(0.2, dotInterval)
	expected := 10.0 * 0.5 // DPS * interval
	if math.Abs(dmg-expected) > 1e-9 {
		t.Errorf("dmg=%.1f, 期望%.1f", dmg, expected)
	}
}
