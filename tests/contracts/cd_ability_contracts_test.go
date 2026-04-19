// cd_ability_contracts_test.go — CD 类型能力契约测试。
//
// 验证 onTick + cooldown 组合能力的三条新路径：
//  1. modifyStat 在 onTick 中通过短时效 buff/debuff 生效（不再 warning 跳过）
//  2. teleport 在 onTick 中调用 PushBack 生效（不再 warning 跳过）
//  3. 预制 CD 能力在 ability-descriptors.json 中存在且可正确解析
//
// 这些测试在实现前必须 FAIL（红灯）。
package contracts_test

import (
	"testing"

	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/core/tower/descriptor"
)

// ── 辅助 ────────────────────────────────────────────────────

func cdTestTower() *tower.Tower {
	t := &tower.Tower{
		X: 200, Y: 200, Range: 300, Damage: 50,
		Active: true, InstanceKey: "test_cd",
		Strength: strength.NewStrengthData(),
		Buffs:    buff.NewDefaultBuffList(),
	}
	return t
}

func cdTestEnemyPool() *enemy.Pool {
	pool := enemy.NewPool(10)
	e := pool.Spawn(220, 200, 100, 60, 2, "normal", nil)
	e.Path = []gamemap.Point{
		{X: 0, Y: 200},
		{X: 100, Y: 200},
		{X: 200, Y: 200},
		{X: 300, Y: 200},
		{X: 400, Y: 200},
	}
	e.Buffs = buff.NewDefaultBuffList()
	// 清除出生动画，否则 querier 跳过 spawning 敌人
	e.SpawnTimer = 0
	return pool
}

// ── 契约 1: modifyStat + onTick 通过 buff 生效 ─────────────

// TestContract_ModifyStatOnTick_AppliesAsBuff 验证 modifyStat 效果在
// onTick 管线中以短时效 buff 形式施加到自身塔，而非 warning 跳过。
func TestContract_ModifyStatOnTick_AppliesAsBuff(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_modstat_tick",
		Label: "Test ModifyStat Tick",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger: descriptor.TriggerOnTick,
				Conditions: []descriptor.Condition{
					descriptor.NewCooldownCondition(2.0),
				},
				Selector: descriptor.SelfTowerSelector{},
				Effects: []descriptor.Effect{
					descriptor.ModifyStatEffect{Stat: "damage", Multiplier: 1.5},
				},
			},
		},
	}

	ability := descriptor.NewDescriptorAbility(desc)
	ticker, ok := ability.(tower.Ticker)
	if !ok {
		t.Fatal("modifyStat + onTick ability should implement Ticker")
	}

	tw := cdTestTower()
	ctx := &tower.TickContext{DT: 0.5}

	// 第 1 tick: CooldownCondition 首次通过 → modifyStat 应以 buff 施加
	tr := ticker.OnTick(tw, ctx)
	if tr == nil {
		t.Fatal("tick 1: expected non-nil TickResult")
	}

	// 核心断言：塔的 BuffList 中应有 damage aura buff
	if !tw.Buffs.Has(buff.IDAuraDamageAmp) {
		t.Errorf("tick 1: tower should have %s buff after modifyStat onTick, but doesn't", buff.IDAuraDamageAmp)
	}
}

// ── 契约 2: teleport + onTick 调用 PushBack ────────────────

// TestContract_TeleportOnTick_PushesEnemyBack 验证 teleport 效果在
// onTick 管线中实际调用 PushBack 回推敌人，而非 warning 跳过。
func TestContract_TeleportOnTick_PushesEnemyBack(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_teleport_tick",
		Label: "Test Teleport Tick",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger: descriptor.TriggerOnTick,
				Conditions: []descriptor.Condition{
					descriptor.NewCooldownCondition(3.0),
				},
				Selector: descriptor.AllInRangeSelector{},
				Effects: []descriptor.Effect{
					descriptor.TeleportEffect{Distance: descriptor.FixedScaler{Value: 80}},
				},
			},
		},
	}

	ability := descriptor.NewDescriptorAbility(desc)
	ticker, ok := ability.(tower.Ticker)
	if !ok {
		t.Fatal("teleport + onTick ability should implement Ticker")
	}

	tw := cdTestTower()
	pool := cdTestEnemyPool()
	towerPool := tower.NewPool(5)
	ctx := &tower.TickContext{DT: 0.5, Enemies: pool, Towers: towerPool}

	// 记录敌人初始 X
	var initialX float64
	pool.EachActive(func(e *enemy.Enemy) {
		initialX = e.X
	})

	// 第 1 tick: CooldownCondition 首次通过 → teleport 应回推敌人
	ticker.OnTick(tw, ctx)

	// 核心断言：敌人的 X 应该比初始位置小（被往回推了）
	var finalX float64
	pool.EachActive(func(e *enemy.Enemy) {
		finalX = e.X
	})

	if finalX >= initialX {
		t.Errorf("teleport onTick should push enemy back: initialX=%.1f, finalX=%.1f (expected finalX < initialX)", initialX, finalX)
	}
}

// ── 契约 3: 预制 CD 能力存在且可解析 ─────────────────────────

// TestContract_PrebuiltCDAbilities_Exist 验证 ability-descriptors.json
// 中包含 onTick + cooldown 的战斗类 CD 能力。
func TestContract_PrebuiltCDAbilities_Exist(t *testing.T) {
	table := descriptor.GlobalDescriptorTable()
	if table == nil {
		t.Fatal("GlobalDescriptorTable returned nil")
	}

	// 期望存在的 CD 能力 ID
	cdAbilities := []string{
		"pulseStun",
		"pushField",
		"curseAura",
	}

	for _, id := range cdAbilities {
		desc, ok := table[id]
		if !ok || desc == nil {
			t.Errorf("prebuilt CD ability %q not found in descriptor table", id)
			continue
		}

		// 验证至少有一条 pipeline 使用 onTick + cooldown
		found := false
		for _, p := range desc.Pipelines {
			if p.Trigger != descriptor.TriggerOnTick {
				continue
			}
			for _, c := range p.Conditions {
				if _, isCooldown := c.(*descriptor.CooldownCondition); isCooldown {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("prebuilt CD ability %q should have onTick + cooldown pipeline", id)
		}
	}
}
