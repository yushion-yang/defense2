package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ── AdaptToHitResult: 伤害映射 ───────────────────────────────

func TestAdaptToHitResult_FlatDamage(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 10, DamageMode: descriptor.DmgFlat},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.SeparateDamage != 10 {
		t.Errorf("SeparateDamage = %v, want 10", hr.SeparateDamage)
	}
	if hr.BonusDamage != 0 {
		t.Errorf("BonusDamage = %v, want 0", hr.BonusDamage)
	}
}

func TestAdaptToHitResult_RatioDamage(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 25, DamageMode: descriptor.DmgRatio},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.BonusDamage != 25 {
		t.Errorf("BonusDamage = %v, want 25", hr.BonusDamage)
	}
	if hr.SeparateDamage != 0 {
		t.Errorf("SeparateDamage = %v, want 0", hr.SeparateDamage)
	}
}

func TestAdaptToHitResult_HpPercentDamage(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 50, DamageMode: descriptor.DmgHpPercent},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.SeparateDamage != 50 {
		t.Errorf("SeparateDamage = %v, want 50", hr.SeparateDamage)
	}
}

func TestAdaptToHitResult_MultipleDamages(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 10, DamageMode: descriptor.DmgFlat},
		{Type: descriptor.EffTypeDamage, Damage: 5, DamageMode: descriptor.DmgFlat},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.SeparateDamage != 15 {
		t.Errorf("SeparateDamage = %v, want 15", hr.SeparateDamage)
	}
}

func TestAdaptToHitResult_MixedDamageModes(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 10, DamageMode: descriptor.DmgFlat},
		{Type: descriptor.EffTypeDamage, Damage: 20, DamageMode: descriptor.DmgRatio},
		{Type: descriptor.EffTypeDamage, Damage: 30, DamageMode: descriptor.DmgHpPercent},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.SeparateDamage != 40 {
		t.Errorf("SeparateDamage = %v, want 40 (10 flat + 30 hp%%)", hr.SeparateDamage)
	}
	if hr.BonusDamage != 20 {
		t.Errorf("BonusDamage = %v, want 20", hr.BonusDamage)
	}
}

// ── AdaptToHitResult: 暴击 ───────────────────────────────────

func TestAdaptToHitResult_Crit(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 50, DamageMode: descriptor.DmgRatio, IsCrit: true},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if !hr.IsCrit {
		t.Error("expected IsCrit = true")
	}
	if hr.BonusDamage != 50 {
		t.Errorf("BonusDamage = %v, want 50", hr.BonusDamage)
	}
}

func TestAdaptToHitResult_AnyCritSetsFlag(t *testing.T) {
	// 只要有一个效果是暴击，整体就标记暴击
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 10, DamageMode: descriptor.DmgFlat, IsCrit: false},
		{Type: descriptor.EffTypeDamage, Damage: 20, DamageMode: descriptor.DmgRatio, IsCrit: true},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if !hr.IsCrit {
		t.Error("expected IsCrit = true when any effect has crit")
	}
}

// ── AdaptToHitResult: CC ─────────────────────────────────────

func TestAdaptToHitResult_Slow(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeSlow, SlowFactor: 0.3, Duration: 1.5},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.Slow == nil {
		t.Fatal("expected Slow != nil")
	}
	if hr.Slow.Factor != 0.3 {
		t.Errorf("Slow.Factor = %v, want 0.3", hr.Slow.Factor)
	}
	if hr.Slow.Duration != 1.5 {
		t.Errorf("Slow.Duration = %v, want 1.5", hr.Slow.Duration)
	}
}

func TestAdaptToHitResult_Stun(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeStun, StunDur: 0.5},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.Stun == nil {
		t.Fatal("expected Stun != nil")
	}
	if hr.Stun.Duration != 0.5 {
		t.Errorf("Stun.Duration = %v, want 0.5", hr.Stun.Duration)
	}
}

// ── AdaptToHitResult: DoT ────────────────────────────────────

func TestAdaptToHitResult_Bleed(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDot, DotSubtype: "bleed", DotValue: 5, DotDuration: 3},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.Bleed == nil {
		t.Fatal("expected Bleed != nil")
	}
	if hr.Bleed.DPS != 5 {
		t.Errorf("Bleed.DPS = %v, want 5", hr.Bleed.DPS)
	}
	if hr.Bleed.Duration != 3 {
		t.Errorf("Bleed.Duration = %v, want 3", hr.Bleed.Duration)
	}
}

func TestAdaptToHitResult_Burn(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDot, DotSubtype: "burn", DotValue: 8, DotDuration: 2},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.Burn == nil {
		t.Fatal("expected Burn != nil")
	}
	if hr.Burn.DPS != 8 {
		t.Errorf("Burn.DPS = %v, want 8", hr.Burn.DPS)
	}
	if hr.Burn.Duration != 2 {
		t.Errorf("Burn.Duration = %v, want 2", hr.Burn.Duration)
	}
}

func TestAdaptToHitResult_Poison(t *testing.T) {
	// poison 现在映射到 HitResult.Poison（Batch 1a 修复）
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDot, DotSubtype: "poison", DotValue: 3, DotDuration: 5},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult for poison")
	}
	if hr.Poison == nil {
		t.Fatal("expected Poison != nil")
	}
	if hr.Poison.DPS != 3 {
		t.Errorf("Poison.DPS = %v, want 3", hr.Poison.DPS)
	}
	if hr.Poison.Duration != 5 {
		t.Errorf("Poison.Duration = %v, want 5", hr.Poison.Duration)
	}
}

// ── AdaptToHitResult: 组合效果 ───────────────────────────────

func TestAdaptToHitResult_Combined(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 10, DamageMode: descriptor.DmgFlat},
		{Type: descriptor.EffTypeSlow, SlowFactor: 0.3, Duration: 1.0},
		{Type: descriptor.EffTypeDot, DotSubtype: "burn", DotValue: 5, DotDuration: 2},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.SeparateDamage != 10 {
		t.Errorf("SeparateDamage = %v, want 10", hr.SeparateDamage)
	}
	if hr.Slow == nil {
		t.Error("expected Slow != nil")
	}
	if hr.Burn == nil {
		t.Error("expected Burn != nil")
	}
}

// ── AdaptToHitResult: 空输入 ─────────────────────────────────

func TestAdaptToHitResult_Nil(t *testing.T) {
	hr := descriptor.AdaptToHitResult(nil)
	if hr != nil {
		t.Errorf("expected nil for nil input, got %+v", hr)
	}
}

func TestAdaptToHitResult_Empty(t *testing.T) {
	hr := descriptor.AdaptToHitResult([]descriptor.EffectResult{})
	if hr != nil {
		t.Errorf("expected nil for empty input, got %+v", hr)
	}
}

// ── AdaptToHitResult: 跳过非 hit 效果 ───────────────────────

func TestAdaptToHitResult_SkipsTickEffects(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeGold, GoldAmount: 5},
		{Type: descriptor.EffTypeBuff, BuffStat: "damage", BuffBonus: 0.1},
		{Type: descriptor.EffTypeSelfBuff, BuffStat: "speed", BuffBonus: 0.2},
		{Type: descriptor.EffTypeModifyStat, StatMult: 1.5},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr != nil {
		t.Errorf("expected nil for tick-only effects, got %+v", hr)
	}
}

func TestAdaptToHitResult_WeakenAndSilence(t *testing.T) {
	// weaken 和 silence 现在映射到 HitResult（Batch 1a 修复）
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeWeaken, WeakenAmp: 0.3, WeakenDur: 2},
		{Type: descriptor.EffTypeSilence, Duration: 3},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult for weaken+silence")
	}
	if hr.Weaken == nil {
		t.Fatal("expected Weaken != nil")
	}
	if hr.Weaken.Amplify != 0.3 {
		t.Errorf("Weaken.Amplify = %v, want 0.3", hr.Weaken.Amplify)
	}
	if hr.Weaken.Duration != 2 {
		t.Errorf("Weaken.Duration = %v, want 2", hr.Weaken.Duration)
	}
	if !hr.Silence {
		t.Error("expected Silence = true")
	}
}

// ── AdaptToHitResult: Purge / Teleport ──────────────────────

func TestAdaptToHitResult_Purge(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypePurge, PurgeCount: 3},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult for purge")
	}
	if hr.Purge == nil {
		t.Fatal("expected Purge != nil")
	}
	if hr.Purge.Count != 3 {
		t.Errorf("Purge.Count = %d, want 3", hr.Purge.Count)
	}
}

func TestAdaptToHitResult_Teleport(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeTeleport, TeleportDist: 120.5},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult for teleport")
	}
	if hr.Teleport == nil {
		t.Fatal("expected Teleport != nil")
	}
	if hr.Teleport.Distance != 120.5 {
		t.Errorf("Teleport.Distance = %v, want 120.5", hr.Teleport.Distance)
	}
}

func TestAdaptToHitResult_PurgePlusTeleport(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypePurge, PurgeCount: 2},
		{Type: descriptor.EffTypeTeleport, TeleportDist: 80},
		{Type: descriptor.EffTypeDamage, Damage: 15, DamageMode: descriptor.DmgFlat},
	}
	hr := descriptor.AdaptToHitResult(results)
	if hr == nil {
		t.Fatal("expected non-nil HitResult")
	}
	if hr.Purge == nil || hr.Purge.Count != 2 {
		t.Errorf("Purge = %v, want Count=2", hr.Purge)
	}
	if hr.Teleport == nil || hr.Teleport.Distance != 80 {
		t.Errorf("Teleport = %v, want Distance=80", hr.Teleport)
	}
	if hr.SeparateDamage != 15 {
		t.Errorf("SeparateDamage = %v, want 15", hr.SeparateDamage)
	}
}

// ── AdaptToTickResult: 金币 ──────────────────────────────────

func TestAdaptToTickResult_Gold(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeGold, GoldAmount: 3.7},
	}
	tr := descriptor.AdaptToTickResult(results)
	if tr == nil {
		t.Fatal("expected non-nil TickResult")
	}
	if tr.GoldEarned != 3 {
		t.Errorf("GoldEarned = %v, want 3 (truncated from 3.7)", tr.GoldEarned)
	}
}

func TestAdaptToTickResult_MultipleGold(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeGold, GoldAmount: 2},
		{Type: descriptor.EffTypeGold, GoldAmount: 1},
	}
	tr := descriptor.AdaptToTickResult(results)
	if tr == nil {
		t.Fatal("expected non-nil TickResult")
	}
	if tr.GoldEarned != 3 {
		t.Errorf("GoldEarned = %v, want 3", tr.GoldEarned)
	}
}

// ── AdaptToTickResult: 空输入 ────────────────────────────────

func TestAdaptToTickResult_Nil(t *testing.T) {
	tr := descriptor.AdaptToTickResult(nil)
	if tr != nil {
		t.Errorf("expected nil for nil input, got %+v", tr)
	}
}

func TestAdaptToTickResult_Empty(t *testing.T) {
	tr := descriptor.AdaptToTickResult([]descriptor.EffectResult{})
	if tr != nil {
		t.Errorf("expected nil for empty input, got %+v", tr)
	}
}

func TestAdaptToTickResult_IgnoresNonTickEffects(t *testing.T) {
	// Damage 和 Slow 不是 tick 效果，应被忽略。
	// Buff 现在由 tick 适配器标记 hasEffect（由 applyTickBuffEffects 处理），
	// 所以只测试纯非 tick 类型（damage/slow）返回 nil。
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 10, DamageMode: descriptor.DmgFlat},
		{Type: descriptor.EffTypeSlow, SlowFactor: 0.5, Duration: 1},
	}
	tr := descriptor.AdaptToTickResult(results)
	if tr != nil {
		t.Errorf("expected nil for non-tick effects, got %+v", tr)
	}
}

func TestAdaptToTickResult_BuffSetsHasEffect(t *testing.T) {
	// Buff 类型现在由 tick 适配器识别（applyTickBuffEffects 直接处理），
	// 返回非 nil TickResult（GoldEarned=0）。
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeBuff, BuffStat: "range", BuffBonus: 0.1},
	}
	tr := descriptor.AdaptToTickResult(results)
	if tr == nil {
		t.Fatal("expected non-nil TickResult for buff effect")
	}
	if tr.GoldEarned != 0 {
		t.Errorf("GoldEarned = %v, want 0", tr.GoldEarned)
	}
}

// ── AdaptToTickResult: 混合效果只取 gold ─────────────────────

func TestAdaptToTickResult_MixedWithGold(t *testing.T) {
	results := []descriptor.EffectResult{
		{Type: descriptor.EffTypeDamage, Damage: 10, DamageMode: descriptor.DmgFlat},
		{Type: descriptor.EffTypeGold, GoldAmount: 5},
		{Type: descriptor.EffTypeSlow, SlowFactor: 0.3, Duration: 1},
	}
	tr := descriptor.AdaptToTickResult(results)
	if tr == nil {
		t.Fatal("expected non-nil TickResult")
	}
	if tr.GoldEarned != 5 {
		t.Errorf("GoldEarned = %v, want 5", tr.GoldEarned)
	}
}
