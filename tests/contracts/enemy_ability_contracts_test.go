// enemy_ability_contracts_test.go — 敌人能力系统契约测试。
// 验证 abilities.json 完整性、原型引用一致性、Spawn 字段拷贝。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
)

// phaseVal reads phaseShift buff params from enemy's BuffList.
// duration=true returns Value (phaseDuration), false returns Value2 (phaseCooldown).
func phaseVal(e *enemy.Enemy, duration bool) float64 {
	if e.Buffs == nil {
		return 0
	}
	b, ok := e.Buffs.Get("phaseShift")
	if !ok {
		return 0
	}
	if duration {
		return b.Value
	}
	return b.Value2
}

// validAbilityCategories 能力配置中允许的类别集合。
var validAbilityCategories = map[string]bool{
	"defense":  true,
	"passive":  true,
	"resist":   true,
	"movement": true,
	"offense":  true,
	"support":  true,
	"death":    true,
}

// ═══════════════════════════════════════
// 能力配置完整性
// ═══════════════════════════════════════

func TestEnemyAbilityConfigComplete(t *testing.T) {
	abilities, err := config.LoadEnemyAbilities()
	if err != nil {
		t.Fatalf("加载能力配置失败: %v", err)
	}
	if len(abilities) < 10 {
		t.Fatalf("期望至少 10 个能力，实际 %d", len(abilities))
	}

	for id, def := range abilities {
		if def.Type == "" {
			t.Errorf("能力 %q: type 为空", id)
		}
		if def.Label == "" {
			t.Errorf("能力 %q: label 为空", id)
		}
		if def.Category == "" {
			t.Errorf("能力 %q: category 为空", id)
		}
		if !validAbilityCategories[def.Category] {
			t.Errorf("能力 %q: category=%q 不在合法列表中", id, def.Category)
		}
		if def.Description == "" {
			t.Errorf("能力 %q: description 为空", id)
		}
	}
}

// ═══════════════════════════════════════
// 能力类别覆盖
// ═══════════════════════════════════════

func TestEnemyAbilityCategories(t *testing.T) {
	abilities, err := config.LoadEnemyAbilities()
	if err != nil {
		t.Fatalf("加载能力配置失败: %v", err)
	}

	catCount := make(map[string]int)
	for _, def := range abilities {
		catCount[def.Category]++
	}

	// 统计原型实际引用的类别
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("加载原型失败: %v", err)
	}
	usedCategories := make(map[string]bool)
	for _, arch := range archs {
		for _, ref := range arch.Abilities {
			if def, ok := abilities[ref.Type]; ok {
				usedCategories[def.Category] = true
			}
		}
	}

	// 每个被引用的类别至少有 1 个能力
	for cat := range usedCategories {
		if catCount[cat] == 0 {
			t.Errorf("类别 %q 被原型引用但在能力表中无定义", cat)
		}
	}

	// 验证至少覆盖 5 个类别（实际 7 个）
	if len(catCount) < 5 {
		t.Errorf("能力类别数=%d，期望 ≥5", len(catCount))
	}
}

// ═══════════════════════════════════════
// 原型能力引用一致性
// ═══════════════════════════════════════

func TestEnemyAbilityArchetypeRefs(t *testing.T) {
	abilities, err := config.LoadEnemyAbilities()
	if err != nil {
		t.Fatalf("加载能力配置失败: %v", err)
	}

	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("加载原型失败: %v", err)
	}

	for id, arch := range archs {
		for _, ref := range arch.Abilities {
			if _, ok := abilities[ref.Type]; !ok {
				t.Errorf("原型 %q 引用了不存在的能力 %q", id, ref.Type)
			}
		}
	}
}

// ═══════════════════════════════════════
// silenceable 字段
// ═══════════════════════════════════════

func TestEnemyAbilitySilenceableField(t *testing.T) {
	abilities, err := config.LoadEnemyAbilities()
	if err != nil {
		t.Fatalf("加载能力配置失败: %v", err)
	}

	// purge 必须不可沉默（boss 专用，不应被玩家禁用）
	purge, ok := abilities["purge"]
	if !ok {
		t.Fatal("缺少 purge 能力")
	}
	if purge.Silenceable {
		t.Error("purge 应为 silenceable=false（boss 专用）")
	}

	// 至少 10 个能力可被沉默
	silenceableCount := 0
	for _, def := range abilities {
		if def.Silenceable {
			silenceableCount++
		}
	}
	if silenceableCount < 10 {
		t.Errorf("可沉默能力数=%d，期望 ≥10", silenceableCount)
	}
}

// ═══════════════════════════════════════
// Spawn 字段拷贝
// ═══════════════════════════════════════

func TestEnemyAbilitySpawnCopy(t *testing.T) {
	pool := enemy.NewPool(8)

	cfg := enemy.DefaultSpawnConfig()
	cfg.ProjectileBlockChance = 1
	cfg.ArmorFlat = 5
	cfg.EvasionChance = 0.3
	cfg.DamageCap = 60
	cfg.DamageCapPercent = 0.05
	cfg.DashSpeedBoost = 0.5
	cfg.DashDuration = 2
	cfg.DashCooldown = 5
	cfg.PhaseDuration = 2
	cfg.PhaseCooldown = 4
	cfg.StrDrainRatio = 0.5
	cfg.StrDrainInterval = 10
	cfg.StrDrainDuration = 6
	cfg.DeathSpawnCount = 3
	cfg.DeathSpawnArch = "normal"
	cfg.PurgeInterval = 4
	cfg.PurgeImmuneDur = 2

	e := pool.Spawn(100, 100, 500, 60, 1, "test", cfg)
	if e == nil {
		t.Fatal("Spawn 返回 nil")
	}

	checks := []struct {
		name string
		got  float64
		want float64
	}{
		{"ProjectileBlockChance", e.ProjectileBlockChance, 1},
		{"ArmorFlat", e.ArmorFlat, 5},
		{"EvasionChance", e.EvasionChance, 0.3},
		{"DamageCap", e.DamageCap, 60},
		{"DamageCapPercent", e.DamageCapPercent, 0.05},
		{"DashSpeedBoost", e.DashSpeedBoost, 0.5},
		{"DashDuration", e.DashDuration, 2},
		{"DashCooldown", e.DashCooldown, 5},
		{"PhaseShift(duration)", phaseVal(e, true), 2},
		{"PhaseShift(cooldown)", phaseVal(e, false), 4},
		{"StrDrainRatio", e.StrDrainRatio, 0.5},
		{"StrDrainInterval", e.StrDrainInterval, 10},
		{"StrDrainDuration", e.StrDrainDuration, 6},
		{"PurgeInterval", e.PurgeInterval, 4},
		{"PurgeImmuneDur", e.PurgeImmuneDur, 2},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got=%.4f, want=%.4f", c.name, c.got, c.want)
		}
	}

	if e.DeathSpawnCount != 3 {
		t.Errorf("DeathSpawnCount: got=%d, want=3", e.DeathSpawnCount)
	}
	if e.DeathSpawnArch != "normal" {
		t.Errorf("DeathSpawnArch: got=%q, want=%q", e.DeathSpawnArch, "normal")
	}
}

func TestEnemyAbilityIDsCopied(t *testing.T) {
	pool := enemy.NewPool(8)

	cfg := enemy.DefaultSpawnConfig()
	cfg.AbilityIDs = []string{"ccImmune", "armorPlating"}

	e := pool.Spawn(100, 100, 500, 60, 1, "test", cfg)
	if e == nil {
		t.Fatal("Spawn 返回 nil")
	}

	if len(e.AbilityIDs) != 2 {
		t.Fatalf("AbilityIDs 长度: got=%d, want=2", len(e.AbilityIDs))
	}
	if e.AbilityIDs[0] != "ccImmune" {
		t.Errorf("AbilityIDs[0]: got=%q, want=%q", e.AbilityIDs[0], "ccImmune")
	}
	if e.AbilityIDs[1] != "armorPlating" {
		t.Errorf("AbilityIDs[1]: got=%q, want=%q", e.AbilityIDs[1], "armorPlating")
	}
}
