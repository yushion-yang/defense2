// system_contracts_test.go — 跨系统结构性契约测试。
// 验证配置→代码映射完整性、注册表完整性、系统间一致性。
// 只测结构不变量，不测可调数值。
package contracts_test

import (
	"testing"

	_ "defense2/internal/core/warden/types" // 触发 init 注册战灵

	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/economy"
	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemode"
	"defense2/internal/core/tower"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/core/warden"
)

func init() {
	// 加载能力配置并注册到 tower.Registry（需要 DataFS 已设置，由 config_rules_test.go 的 init 完成）
	if err := abilities.InitConfigAbilities(); err != nil {
		panic("加载能力配置失败: " + err.Error())
	}
}

// ═══════════════════════════════════════
// 能力系统契约
// ═══════════════════════════════════════

// TestAbilityConfigToCodeMapping 验证 abilities.json 中每种能力在代码中有实现。
// 历史 bug: JSON tag 错误导致能力系统整体失效。
func TestAbilityConfigToCodeMapping(t *testing.T) {
	table := config.GlobalAbilityTable()
	if len(table) == 0 {
		t.Fatal("能力配置表为空（GlobalAbilityTable 返回空 map）")
	}

	for name := range table {
		if _, ok := tower.Registry[name]; !ok {
			t.Errorf("能力 %q 在 abilities.json 中定义但未在 tower.Registry 中注册", name)
		}
	}
}

// TestAbilityCodeToConfigMapping 验证代码中注册的能力在配置中有定义。
func TestAbilityCodeToConfigMapping(t *testing.T) {
	table := config.GlobalAbilityTable()
	for name := range tower.Registry {
		if _, ok := table[name]; !ok {
			// 允许非 config 驱动的能力（如 scaling.go 中的 KillUpgrade 等）
			t.Logf("注意: 能力 %q 在代码中注册但不在 abilities.json 中（可能是代码驱动的能力）", name)
		}
	}
}

// TestAbilityHasRequiredFields 验证每种能力配置的必要字段。
func TestAbilityHasRequiredFields(t *testing.T) {
	table := config.GlobalAbilityTable()
	for name, def := range table {
		t.Run(name, func(t *testing.T) {
			if def.Label == "" {
				t.Errorf("能力 %q 缺少 label", name)
			}
			if def.Category == "" {
				t.Errorf("能力 %q 缺少 category", name)
			}
			if def.ScaleDim == "" && def.ParamDim == "" {
				t.Errorf("能力 %q 缺少 scaleDim 和 paramDim（至少需要一个维度标识）", name)
			}
		})
	}
}

// TestAbilityCategories 验证能力类别在合法范围内。
func TestAbilityCategories(t *testing.T) {
	validCats := map[string]bool{
		"attack": true, "cc": true, "damage": true,
		"buff": true, "dot": true, "zone": true,
		"combat": true, "control": true, "aura": true, // 兼容旧名
	}
	table := config.GlobalAbilityTable()
	for name, def := range table {
		if !validCats[def.Category] {
			t.Errorf("能力 %q 的 category=%q 不在合法列表中", name, def.Category)
		}
	}
}

// ═══════════════════════════════════════
// 攻击方式契约
// ═══════════════════════════════════════

// TestAttackStyleHandlerRegistered 验证每种塔使用的 attackStyle 有对应 handler。
// 历史 bug: 新增 attackStyle 忘注册 handler 导致 nil。
func TestAttackStyleHandlerRegistered(t *testing.T) {
	towers, err := config.LoadAllTowers()
	if err != nil {
		t.Fatalf("加载塔配置失败: %v", err)
	}
	for key, tw := range towers {
		style := tower.AttackStyle(tw.AttackStyle)
		if style == "" {
			continue
		}
		if h := combat.Get(style); h == nil {
			t.Errorf("塔 %q 的 attackStyle=%q 没有注册 handler", key, style)
		}
	}
}

// ═══════════════════════════════════════
// 经济系统契约
// ═══════════════════════════════════════

// TestEconomyDefaults 验证经济系统默认值在合理范围内。
func TestEconomyDefaults(t *testing.T) {
	cfg := economy.DefaultConfig()

	if cfg.KillReward <= 0 {
		t.Errorf("KillReward=%d 应 > 0", cfg.KillReward)
	}
	if cfg.SellRefundRatio <= 0 || cfg.SellRefundRatio > 1.0 {
		t.Errorf("SellRefundRatio=%.2f 应在 (0, 1.0]", cfg.SellRefundRatio)
	}
}

// TestSellRefundPositive 验证卖塔退款金额 > 0。
func TestSellRefundPositive(t *testing.T) {
	cfg := economy.DefaultConfig()
	refund := cfg.SellRefund(50)
	if refund <= 0 {
		t.Errorf("SellRefund(50)=%d 应 > 0", refund)
	}
	if refund >= 50 {
		t.Errorf("SellRefund(50)=%d 应 < 原价 50", refund)
	}
}

// ═══════════════════════════════════════
// 波次系统契约
// ═══════════════════════════════════════

// TestSpawnerEnemyCountPositive 验证每波至少出 1 个敌人。

// TestWaveBuffPoolTemplatesExist 已移除 — 旧 BuffTemplate 系统已被直接字段设置替代。

// ═══════════════════════════════════════
// 游戏模式契约
// ═══════════════════════════════════════

// TestGameModeRegistered 验证所有预期模式已注册。
func TestGameModeRegistered(t *testing.T) {
	modes := []string{"casual", "classic", "coop", "test", "autoplay"}
	for _, id := range modes {
		if m := gamemode.Get(id); m == nil {
			t.Errorf("游戏模式 %q 未注册", id)
		}
	}
}

// TestCampaignDefeatOnZeroLives 验证 casual 模式下 lives=0 判败。
func TestCampaignDefeatOnZeroLives(t *testing.T) {
	m := gamemode.Get("casual")
	if m == nil {
		t.Fatal("casual 模式未注册")
	}
	ctx := &gamemode.Context{Lives: 0}
	if !m.CheckDefeat(ctx) {
		t.Error("casual: lives=0 应判败")
	}
}

// TestCampaignVictoryOnAllWaves 验证 casual 模式下全波清完判胜。
func TestCampaignVictoryOnAllWaves(t *testing.T) {
	m := gamemode.Get("casual")
	if m == nil {
		t.Fatal("casual 模式未注册")
	}
	ctx := &gamemode.Context{Wave: 12, MaxWaves: 12, Spawning: false, Lives: 10}
	if !m.CheckVictory(ctx) {
		t.Error("casual: 12/12 波清完应判胜")
	}
}

// TestAutoplayNeverDefeats 验证 autoplay 模式永不失败。
func TestAutoplayNeverDefeats(t *testing.T) {
	m := gamemode.Get("autoplay")
	if m == nil {
		t.Fatal("autoplay 模式未注册")
	}
	ctx := &gamemode.Context{Lives: 0}
	if m.CheckDefeat(ctx) {
		t.Error("autoplay: 不应判败")
	}
}

// ═══════════════════════════════════════
// 战灵契约
// ═══════════════════════════════════════

// TestWardenTypesRegistered 验证所有 5 种战灵行为已注册。
func TestWardenTypesRegistered(t *testing.T) {
	types := []string{"prince", "core", "chain", "skystrike", "envoy"}
	for _, typ := range types {
		w := warden.NewWarden(1, typ, typ)
		if w.State == nil {
			t.Errorf("战灵 %q 的 Behavior 未注册（State 为 nil）", typ)
		}
	}
}

// TestWardenConfigLoads 验证战灵配置可加载。
func TestWardenConfigLoads(t *testing.T) {
	cfgs, err := config.LoadWardenConfigs()
	if err != nil {
		t.Fatalf("加载战灵配置失败: %v", err)
	}
	expected := []string{"prince", "core", "chain", "skystrike", "envoy"}
	for _, key := range expected {
		if _, ok := cfgs[key]; !ok {
			t.Errorf("战灵配置缺少 %q", key)
		}
	}
}

// TestWardenConfigRanges 验证战灵配置值在合理范围。
func TestWardenConfigRanges(t *testing.T) {
	cfgs, err := config.LoadWardenConfigs()
	if err != nil {
		t.Fatalf("加载战灵配置失败: %v", err)
	}
	for key, cfg := range cfgs {
		t.Run(key, func(t *testing.T) {
			if cfg.Damage <= 0 {
				t.Errorf("damage=%.1f 应 > 0", cfg.Damage)
			}
			if cfg.AttackInterval <= 0 {
				t.Errorf("attackInterval=%.2f 应 > 0（否则除零）", cfg.AttackInterval)
			}
			if cfg.Range <= 0 {
				t.Errorf("range=%.1f 应 > 0", cfg.Range)
			}
			if cfg.MoveSpeed <= 0 {
				t.Errorf("moveSpeed=%.1f 应 > 0", cfg.MoveSpeed)
			}
		})
	}
}

// ═══════════════════════════════════════
// 地图契约
// ═══════════════════════════════════════

// TestAllMapsLoadable 验证所有地图可加载。
func TestAllMapsLoadable(t *testing.T) {
	list, err := config.LoadLevelList("")
	if err != nil {
		t.Fatalf("加载地图列表失败: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("地图列表为空")
	}
	for _, entry := range list {
		t.Run(entry.ID, func(t *testing.T) {
			m, err := config.LoadMap(entry.ID)
			if err != nil {
				t.Errorf("地图 %q 加载失败: %v", entry.ID, err)
				return
			}
			if m.Rows <= 0 || m.Cols <= 0 {
				t.Errorf("地图 %q 尺寸无效: %dx%d", entry.ID, m.Rows, m.Cols)
			}
		})
	}
}

// ═══════════════════════════════════════
// 伤害管线契约
// ═══════════════════════════════════════

// TestDamagePipelineMinDamage 验证管线最低伤害保底。
func TestDamagePipelineMinDamage(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true}
	r := combat.ApplyDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 0.001, // 极小伤害
	})
	if r.FinalDamage < 1 {
		t.Errorf("极小伤害应保底 1 点，实际=%.2f", r.FinalDamage)
	}
}

// TestDamagePipelineZeroDamage 验证 0 伤害不造成扣血。
func TestDamagePipelineZeroDamage(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true}
	r := combat.ApplyDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 0,
	})
	if r.FinalDamage != 0 {
		t.Errorf("0 伤害不应造成扣血，实际=%.2f", r.FinalDamage)
	}
}

// TestDamagePipelineInvincibleBlocks 验证无敌状态阻挡伤害。
func TestDamagePipelineInvincibleBlocks(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{IsInvincible: true}}
	r := combat.ApplyDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 50,
	})
	if !r.Blocked {
		t.Error("无敌状态应阻挡伤害")
	}
	if e.HP != 100 {
		t.Errorf("无敌时 HP 不应变化，实际=%.1f", e.HP)
	}
}

// TestDamagePipelineKill 验证伤害超过 HP 时判杀。
func TestDamagePipelineKill(t *testing.T) {
	e := &enemy.Enemy{HP: 10, MaxHP: 100, Active: true}
	r := combat.ApplyDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 50,
	})
	if !r.Killed {
		t.Error("伤害超过 HP 应判杀")
	}
	if e.HP != 0 {
		t.Errorf("击杀后 HP 应为 0，实际=%.1f", e.HP)
	}
}
