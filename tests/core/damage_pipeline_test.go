package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
)

// makePipelineEnemy 创建测试用敌人
func makePipelineEnemy(hp, maxHP float64) *enemy.Enemy {
	return &enemy.Enemy{
		HP:     hp,
		MaxHP:  maxHP,
		Active: true,
		Path:   []gamemap.Point{{X: 0, Y: 0}},
	}
}

// ============================================================
// 伤害类型测试
// ============================================================

func TestDamageType_BypassRules(t *testing.T) {
	tests := []struct {
		typ        string
		reduction  bool
		invincible bool
	}{
		{combat.DmgPhysical, false, false},
		{combat.DmgMagic, false, false},
		{combat.DmgTrue, true, false},
		{combat.DmgPure, true, true},
	}
	for _, tt := range tests {
		if combat.IgnoresReduction(tt.typ) != tt.reduction {
			t.Errorf("%s: IgnoresReduction=%v, 期望%v", tt.typ, !tt.reduction, tt.reduction)
		}
		if combat.IgnoresInvincible(tt.typ) != tt.invincible {
			t.Errorf("%s: IgnoresInvincible=%v, 期望%v", tt.typ, !tt.invincible, tt.invincible)
		}
	}
}

func TestDamageTypeColor(t *testing.T) {
	c := combat.DamageTypeColor(combat.DmgPhysical)
	if c.R != 0xef {
		t.Errorf("物理伤害颜色R=%d, 期望0xef", c.R)
	}
	// 未知类型降级到物理
	c2 := combat.DamageTypeColor("unknown")
	if c2.R != 0xef {
		t.Errorf("未知类型应降级到物理颜色")
	}
}

// ============================================================
// 伤害管线测试
// ============================================================

func TestPipeline_BasicDamage(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 30,
	})
	if r.Blocked {
		t.Fatal("基础伤害不应被阻挡")
	}
	if math.Abs(r.HPDamage-30) > 1e-9 {
		t.Errorf("HP伤害=%.1f, 期望30", r.HPDamage)
	}
	if math.Abs(e.HP-70) > 1e-9 {
		t.Errorf("敌人HP=%.1f, 期望70", e.HP)
	}
}

func TestPipeline_Kill(t *testing.T) {
	e := makePipelineEnemy(50, 100)
	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 60,
	})
	if !r.Killed {
		t.Error("应该击杀")
	}
	if e.HP != 0 {
		t.Errorf("死亡后HP应为0, 实际%.1f", e.HP)
	}
}

// ── 步骤1: 免疫检查 ──

func TestPipeline_Invincible(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	e.IsInvincible = true

	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 50,
	})
	if !r.Blocked || r.BlockedReason != "invincible" {
		t.Error("无敌状态应阻挡物理伤害")
	}
	if e.HP != 100 {
		t.Errorf("无敌时HP不应变化, 实际%.1f", e.HP)
	}
}

func TestPipeline_PureBypassesInvincible(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	e.IsInvincible = true

	r := combat.ProcessDamage(combat.DamageInput{
		Target:     e,
		RawDamage:  50,
		DamageType: combat.DmgPure,
	})
	if r.Blocked {
		t.Error("pure伤害应穿透无敌")
	}
	if math.Abs(e.HP-50) > 1e-9 {
		t.Errorf("pure穿透后HP=%.1f, 期望50", e.HP)
	}
}

func TestPipeline_Untargetable(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	e.IsUntargetable = true

	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 50,
	})
	if !r.Blocked || r.BlockedReason != "untargetable" {
		t.Error("不可选中应阻挡伤害")
	}
}

// ── 步骤2: Boss百分比HP上限 ──

func TestPipeline_BossPercentCap(t *testing.T) {
	e := makePipelineEnemy(10000, 10000)
	e.Boss = true

	r := combat.ProcessDamage(combat.DamageInput{
		Target:      e,
		RawDamage:   9999,
		IsPercentHP: true,
		PercentCap:  0.05, // 5% = 500
	})
	// 伤害应被限制到 500
	if math.Abs(r.HPDamage-500) > 1e-9 {
		t.Errorf("Boss百分比上限后HP伤害=%.1f, 期望500", r.HPDamage)
	}
}

func TestPipeline_NonBossNoPercentCap(t *testing.T) {
	e := makePipelineEnemy(10000, 10000)
	// 非Boss

	r := combat.ProcessDamage(combat.DamageInput{
		Target:      e,
		RawDamage:   9999,
		IsPercentHP: true,
		PercentCap:  0.05,
	})
	// 非Boss不限制
	if math.Abs(r.HPDamage-9999) > 1e-9 {
		t.Errorf("非Boss不应有百分比上限, HP伤害=%.1f", r.HPDamage)
	}
}

// ── 步骤3+4: 增伤/减伤 ──

func TestPipeline_AttackerDamageUp(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	r := combat.ProcessDamage(combat.DamageInput{
		Target:           e,
		RawDamage:        10,
		AttackerDamageUp: 1.5, // +50%
	})
	if math.Abs(r.AfterAttackerMod-15) > 1e-9 {
		t.Errorf("增伤后=%.1f, 期望15", r.AfterAttackerMod)
	}
}

func TestPipeline_TrueDamageIgnoresModifiers(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	r := combat.ProcessDamage(combat.DamageInput{
		Target:           e,
		RawDamage:        10,
		DamageType:       combat.DmgTrue,
		AttackerDamageUp: 2.0,
		TargetDamageDown: 0.5,
	})
	// true 伤害忽略增减益
	if math.Abs(r.HPDamage-10) > 1e-9 {
		t.Errorf("true伤害应忽略增减益, HP伤害=%.1f, 期望10", r.HPDamage)
	}
}

// ── 步骤4.5: 伤害上限 ──

func TestPipeline_DamageCap(t *testing.T) {
	e := makePipelineEnemy(1000, 1000)
	e.DamageCap = 60 // 铁甲怪

	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 200,
	})
	if math.Abs(r.AfterDamageCap-60) > 1e-9 {
		t.Errorf("damageCap后=%.1f, 期望60", r.AfterDamageCap)
	}
}

func TestPipeline_DamageCapPercent(t *testing.T) {
	e := makePipelineEnemy(1000, 1000)
	e.DamageCapPercent = 0.08 // 巨人 8%maxHP = 80

	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 200,
	})
	if math.Abs(r.AfterDamageCap-80) > 1e-9 {
		t.Errorf("damageCapPercent后=%.1f, 期望80", r.AfterDamageCap)
	}
}

func TestPipeline_DamageCapDisabledBySilence(t *testing.T) {
	e := makePipelineEnemy(1000, 1000)
	e.DamageCap = 60
	e.Silenced = true

	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 200,
	})
	if math.Abs(r.AfterDamageCap-200) > 1e-9 {
		t.Errorf("沉默时damageCap应失效, 伤害=%.1f, 期望200", r.AfterDamageCap)
	}
}

// ── 步骤7: 阈值触发 ──

func TestPipeline_Thresholds(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	enemy.AddThreshold(e, "berserk", 0.5)
	enemy.AddThreshold(e, "phase2", 0.3)

	// 打到60HP (60%)，不触发
	r1 := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 40,
	})
	if len(r1.Thresholds) != 0 {
		t.Errorf("60%%HP不应触发阈值, 触发了%d个", len(r1.Thresholds))
	}

	// 打到30HP (30%)，同时触发 berserk(50%) 和 phase2(30%)
	r2 := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 30,
	})
	if len(r2.Thresholds) != 2 {
		t.Errorf("30%%HP应同时触发berserk和phase2, 实际触发%d个: %v", len(r2.Thresholds), r2.Thresholds)
	}

	// 再打一次，不再重复触发
	r3 := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 10,
	})
	if len(r3.Thresholds) != 0 {
		t.Errorf("阈值不应重复触发, 实际触发%v", r3.Thresholds)
	}
}

// ── 步骤8: 最低伤害保底 ──

func TestPipeline_MinimumDamage(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	e.DamageCap = 0.5 // 上限0.5

	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 100,
	})
	// 最低保底1点
	if r.HPDamage < 1 {
		t.Errorf("最低伤害应为1, 实际%.2f", r.HPDamage)
	}
}

// ── QuickDamage 快捷函数 ──

func TestQuickDamage(t *testing.T) {
	e := makePipelineEnemy(100, 100)
	dmg, killed := combat.QuickDamage(e, 30, combat.DmgPhysical)
	if math.Abs(dmg-30) > 1e-9 {
		t.Errorf("QuickDamage=%.1f, 期望30", dmg)
	}
	if killed {
		t.Error("30伤害不应击杀100HP敌人")
	}
}

