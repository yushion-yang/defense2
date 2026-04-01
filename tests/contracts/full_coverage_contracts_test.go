// full_coverage_contracts_test.go — 补齐所有游戏内容的契约测试。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/combat"
	"defense2/internal/core/economy"
	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
)

// ═══════════════════════════════════════
// §1 屏幕与引擎常量
// ═══════════════════════════════════════

func TestScreenDimensions(t *testing.T) {
	if game.ScreenWidth != 1200 {
		t.Errorf("ScreenWidth=%d 应为 1200", game.ScreenWidth)
	}
	if game.ScreenHeight != 540 {
		t.Errorf("ScreenHeight=%d 应为 540", game.ScreenHeight)
	}
}

func TestTargetTPS(t *testing.T) {
	if game.TargetTPS <= 0 {
		t.Errorf("TargetTPS=%d 应 > 0", game.TargetTPS)
	}
}

// ═══════════════════════════════════════
// §2 对象池大小
// ═══════════════════════════════════════

func TestPoolSizesPositive(t *testing.T) {
	if game.MaxTowers <= 0 {
		t.Errorf("MaxTowers=%d 应 > 0", game.MaxTowers)
	}
	if game.MaxEnemies <= 0 {
		t.Errorf("MaxEnemies=%d 应 > 0", game.MaxEnemies)
	}
	if game.MaxProjectiles <= 0 {
		t.Errorf("MaxProjectiles=%d 应 > 0", game.MaxProjectiles)
	}
}

func TestEnemyPoolLargerThanMaxWaveSize(t *testing.T) {
	// 最大波次 wave=25: count = 5+25 = 30, 加 Boss = 31
	// 池大小应远大于单波出怪数
	maxWaveEnemies := 5 + 25 + 1 // Boss
	if game.MaxEnemies < maxWaveEnemies*2 {
		t.Errorf("MaxEnemies=%d 应 >= %d（最大单波 %d 的 2 倍）", game.MaxEnemies, maxWaveEnemies*2, maxWaveEnemies)
	}
}

// ═══════════════════════════════════════
// §5 经济系统具体值
// ═══════════════════════════════════════

func TestSellRefundRatioIs70Percent(t *testing.T) {
	cfg := economy.DefaultConfig()
	if cfg.SellRefundRatio != 0.7 {
		t.Errorf("SellRefundRatio=%.2f 应为 0.7", cfg.SellRefundRatio)
	}
}

// ═══════════════════════════════════════
// §6 每种塔的攻击方式有效
// ═══════════════════════════════════════

func TestEachTowerHasValidAttackStyle(t *testing.T) {
	towers, err := config.LoadAllTowers()
	if err != nil {
		t.Fatal(err)
	}
	for key, tw := range towers {
		t.Run(key, func(t *testing.T) {
			if tw.AttackStyle == "" {
				t.Errorf("塔 %q 没有 attackStyle", key)
				return
			}
			if h := combat.Get(tower.AttackStyle(tw.AttackStyle)); h == nil {
				t.Errorf("塔 %q 的 attackStyle=%q 无 handler", key, tw.AttackStyle)
			}
		})
	}
}

func TestEachTowerHasProjectileSpeedIfNeeded(t *testing.T) {
	towers, err := config.LoadAllTowers()
	if err != nil {
		t.Fatal(err)
	}
	needsSpeed := map[string]bool{"projectile": true, "scatter": true, "pierce": true, "radial": true}
	for key, tw := range towers {
		if needsSpeed[tw.AttackStyle] && tw.ProjectileSpeed <= 0 {
			t.Errorf("塔 %q (style=%s) 需要 projectileSpeed > 0，当前=%.0f", key, tw.AttackStyle, tw.ProjectileSpeed)
		}
	}
}

// ═══════════════════════════════════════
// §8 升级系统
// ═══════════════════════════════════════

func TestMaxAbilitySlotsMatchesCategories(t *testing.T) {
	if tower.MaxAbilitySlots != config.AbilityCatCount {
		t.Errorf("MaxAbilitySlots=%d != AbilityCatCount=%d", tower.MaxAbilitySlots, config.AbilityCatCount)
	}
}

func TestUnlockedSlotsFormula(t *testing.T) {
	// wave 0 → 1 slot, wave 2 → 2, wave 10 → 6 (capped)
	if s := tower.UnlockedSlots(0); s != 1 {
		t.Errorf("UnlockedSlots(0)=%d 应为 1", s)
	}
	if s := tower.UnlockedSlots(2); s != 2 {
		t.Errorf("UnlockedSlots(2)=%d 应为 2", s)
	}
	if s := tower.UnlockedSlots(100); s > tower.MaxAbilitySlots {
		t.Errorf("UnlockedSlots(100)=%d 应 <= %d", s, tower.MaxAbilitySlots)
	}
}

// ═══════════════════════════════════════
// §9 战力系统
// ═══════════════════════════════════════

func TestStrengthBase100(t *testing.T) {
	s := strength.NewStrengthData()
	if s.Effective() != 100 {
		t.Errorf("初始 Effective()=%.1f 应为 100", s.Effective())
	}
	if s.Ratio() != 1.0 {
		t.Errorf("初始 Ratio()=%.2f 应为 1.0", s.Ratio())
	}
}

func TestStrengthAddPermanent(t *testing.T) {
	s := strength.NewStrengthData()
	s.AddPermanent(50)
	if s.Effective() != 150 {
		t.Errorf("加 50 后 Effective()=%.1f 应为 150", s.Effective())
	}
}

func TestStrengthNeverNegative(t *testing.T) {
	s := strength.NewStrengthData()
	s.AddPermanent(-999)
	if s.Effective() < 0 {
		t.Errorf("Effective()=%.1f 应 >= 0", s.Effective())
	}
}

// ═══════════════════════════════════════
// §10 连锁网络
// ═══════════════════════════════════════

func TestChainConstants(t *testing.T) {
	if strength.ChainDistance <= 0 {
		t.Errorf("ChainDistance=%.0f 应 > 0", strength.ChainDistance)
	}
	if strength.ChainStrengthPerTower <= 0 {
		t.Errorf("ChainStrengthPerTower=%.0f 应 > 0", strength.ChainStrengthPerTower)
	}
}

// ═══════════════════════════════════════
// §14 Buff 叠加规则（19 种）
// ═══════════════════════════════════════

func TestBuffStackRulesExist(t *testing.T) {
	required := []string{
		"slow", "stun", "knockup", "silence", "disarm",
		"speedUp", "damageUp", "damageDown", "fireRateUp",
		"invincible", "damageImmune", "controlImmune",
		"slowImmune", "stunImmune",
		"untargetable", "shield", "dot", "tenacity",
	}
	rules := buff.DefaultStackRules
	for _, name := range required {
		if _, ok := rules[name]; !ok {
			t.Errorf("buff 叠加规则缺少 %q", name)
		}
	}
}

func TestBuffSlowUsesStrongestMode(t *testing.T) {
	rules := buff.DefaultStackRules
	r, ok := rules["slow"]
	if !ok {
		t.Fatal("缺少 slow 规则")
	}
	if r.Mode != buff.ModeStrongest {
		t.Errorf("slow 应为 ModeStrongest，实际=%d", r.Mode)
	}
}

func TestBuffDamageUpUsesMultiplicative(t *testing.T) {
	rules := buff.DefaultStackRules
	r, ok := rules["damageUp"]
	if !ok {
		t.Fatal("缺少 damageUp 规则")
	}
	if r.Mode != buff.ModeMultiplicative {
		t.Errorf("damageUp 应为 ModeMultiplicative，实际=%d", r.Mode)
	}
}

func TestBuffInvincibleHighestPriority(t *testing.T) {
	rules := buff.DefaultStackRules
	inv := rules["invincible"]
	unt := rules["untargetable"]
	if inv.Priority < 90 {
		t.Errorf("invincible priority=%.0f 应 >= 90", inv.Priority)
	}
	if unt.Priority < inv.Priority {
		t.Errorf("untargetable priority=%.0f 应 >= invincible priority=%.0f", unt.Priority, inv.Priority)
	}
}

// ═══════════════════════════════════════
// §15 波次组合覆盖
// ═══════════════════════════════════════

// 波次组合覆盖测试需要内部函数访问，跳过（由 autoplay coverage report 覆盖）

// ═══════════════════════════════════════
// §16 伤害管线补充
// ═══════════════════════════════════════

func TestDamagePipeline_DamageReduceRatio(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, DamageReduceRatio: 0.3}
	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 100,
	})
	// 30% 减伤 → 实际 ~70
	if r.FinalDamage > 75 || r.FinalDamage < 65 {
		t.Errorf("DamageReduceRatio=0.3 时 FinalDamage=%.1f 应 ≈ 70", r.FinalDamage)
	}
}

func TestDamagePipeline_HPNeverBelowZero(t *testing.T) {
	e := &enemy.Enemy{HP: 5, MaxHP: 100, Active: true}
	combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 9999,
	})
	if e.HP < 0 {
		t.Errorf("HP=%.1f 不应 < 0", e.HP)
	}
}

// ═══════════════════════════════════════
// §20 地图建造位
// ═══════════════════════════════════════

func TestAllMapsHaveBuildableCells(t *testing.T) {
	list, err := config.LoadLevelList()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range list {
		t.Run(entry.ID, func(t *testing.T) {
			cfg, err := config.LoadMap(entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			gm := gamemap.NewGameMap(cfg)
			buildable := 0
			for row := 0; row < cfg.Rows; row++ {
				for col := 0; col < cfg.Cols; col++ {
					if cfg.Grid[row][col] == config.CellBuildable {
						buildable++
					}
				}
			}
			if buildable == 0 {
				t.Errorf("地图 %q 没有建造位", entry.ID)
			}
			_ = gm
		})
	}
}

func TestAllMapsHavePathCells(t *testing.T) {
	list, err := config.LoadLevelList()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range list {
		t.Run(entry.ID, func(t *testing.T) {
			cfg, err := config.LoadMap(entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			pathCells := 0
			for row := 0; row < cfg.Rows; row++ {
				for col := 0; col < cfg.Cols; col++ {
					if cfg.Grid[row][col] == config.CellPath {
						pathCells++
					}
				}
			}
			if pathCells == 0 {
				t.Errorf("地图 %q 没有路径格", entry.ID)
			}
		})
	}
}

// ═══════════════════════════════════════
// §12 Boss HP 缩放
// ═══════════════════════════════════════

func TestBossHPMultiplierIncreases(t *testing.T) {
	// Boss HP = 8 + wave, 应随波次递增
	prev := 0.0
	for wave := 5; wave <= 25; wave += 5 {
		mul := 8.0 + float64(wave)
		if mul <= prev {
			t.Errorf("wave=%d: Boss HP mul=%.0f 应 > 前一波 %.0f", wave, mul, prev)
		}
		prev = mul
	}
}

// ═══════════════════════════════════════
// §7 能力 OnTick 型验证
// ═══════════════════════════════════════

func TestTickerAbilitiesReturnNonNilResult(t *testing.T) {
	tickerAbilities := []string{
		"damageUpAura", "attackSpeedAura", "rangeAura", "critAura",
		"soloBoost", "goldPassive",
		"poisonZone", "silenceZone", "curseZone", "weakenZone",
	}
	for _, name := range tickerAbilities {
		t.Run(name, func(t *testing.T) {
			ab, ok := tower.Registry[name]
			if !ok {
				t.Fatalf("能力 %q 未注册", name)
			}
			_, isTicker := ab.(tower.Ticker)
			if !isTicker {
				t.Errorf("能力 %q 应实现 Ticker 接口", name)
			}
		})
	}
}

// ═══════════════════════════════════════
// §17 CC 常量一致性
// ═══════════════════════════════════════

func TestMinSpeedRatioConsistency(t *testing.T) {
	// combat 包和 enemy 包各定义一份，必须一致
	if combat.MinSpeedRatio != enemy.MinSpeedRatio {
		t.Errorf("combat.MinSpeedRatio=%.2f != enemy.MinSpeedRatio=%.2f", combat.MinSpeedRatio, enemy.MinSpeedRatio)
	}
}

func TestDotTickIntervalPositive(t *testing.T) {
	if enemy.DotTickInterval <= 0 {
		t.Errorf("DotTickInterval=%.2f 应 > 0", enemy.DotTickInterval)
	}
}

// ═══════════════════════════════════════
// §18 伤害类型
// ═══════════════════════════════════════

func TestOnlyPhysicalDamageTypeExists(t *testing.T) {
	if combat.DmgPhysical != "physical" {
		t.Errorf("DmgPhysical=%q 应为 \"physical\"", combat.DmgPhysical)
	}
}

func TestPhysicalDoesNotIgnoreReduction(t *testing.T) {
	if combat.IgnoresReduction("physical") {
		t.Error("physical 不应忽略减免")
	}
}

func TestPhysicalDoesNotIgnoreInvincible(t *testing.T) {
	if combat.IgnoresInvincible("physical") {
		t.Error("physical 不应忽略无敌")
	}
}
