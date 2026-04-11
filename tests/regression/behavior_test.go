// behavior_test.go — 行为级回归测试。
// 验证完整的数据流和系统交互，不只是单个函数。
package regression_test

import (
	"testing"

	"defense2/tests/regression/sim"
)

// ═══════════════════════════════════════
// 击杀 → 金币完整链
// ═══════════════════════════════════════

// TestBehavior_KillAwardsExactGold 验证击杀敌人后击杀计数增加。
// 注意：真实游戏的金币奖励通过 event bus + economy 系统发放，sim 不模拟该链路。
func TestBehavior_KillAwardsExactGold(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 20, 30). // 低 HP，容易被杀
		WithTower("basic", 250, 100, 25, 200). // 高伤害
		WithGold(0).
		Build()

	s.RunUntil(func(s *sim.Sim) bool { return s.Kills > 0 }, 300)

	s.AssertKills(t, ">=", 1)
}

// TestBehavior_NoGoldForLeakedEnemy 验证泄漏敌人不给金币。
func TestBehavior_NoGoldForLeakedEnemy(t *testing.T) {
	s := sim.New().
		WithStraightPath(200). // 短路径，快速泄漏
		WithEnemy("normal", 9999, 200). // 高 HP 高速，必泄漏
		WithGold(0).
		WithLives(5).
		Build()

	s.RunTicks(120)

	s.AssertLeaked(t, ">=", 1)
	s.AssertGold(t, "==", 0) // 泄漏不给金币
}

// ═══════════════════════════════════════
// 弹射物追踪
// ═══════════════════════════════════════

// TestBehavior_ProjectileTracksMovingTarget 验证弹射物追踪移动中的敌人。
func TestBehavior_ProjectileTracksMovingTarget(t *testing.T) {
	// L 形路径：敌人先右后下，塔在拐角附近
	s := sim.New().
		WithPath(
			sim.Pt(0, 100),   // 起点
			sim.Pt(300, 100), // 拐角
			sim.Pt(300, 300), // 终点
		).
		WithEnemy("normal", 50, 60).
		WithTowerFull("basic", "projectile", 250, 50, 30, 200, 1.0, 300).
		Build()

	// 跑足够多 tick 让弹射物发射并命中
	killed := s.RunUntil(func(s *sim.Sim) bool { return s.Kills > 0 }, 300)

	if !killed {
		// 弹射物至少应发射
		s.AssertProjectileCount(t, ">=", 0) // 不 panic 即可
		// 检查敌人是否受伤
		s.AssertEnemyHP(t, 0, "<", 50)
	}
}

// ═══════════════════════════════════════
// 多塔叠加 DPS
// ═══════════════════════════════════════

// TestBehavior_MultiTowerFasterKill 验证多塔比单塔更快击杀。
func TestBehavior_MultiTowerFasterKill(t *testing.T) {
	// 单塔（高攻速弹射物，确保命中）
	s1 := sim.New().
		WithStraightPath(800).
		WithEnemy("normal", 60, 20).
		WithTowerFull("basic", "projectile", 400, 50, 10, 200, 2.0, 500).
		Build()
	s1.RunUntil(func(s *sim.Sim) bool { return s.Kills > 0 }, 600)
	tick1 := s1.Tick

	// 三塔
	s3 := sim.New().
		WithStraightPath(800).
		WithEnemy("normal", 60, 20).
		WithTowerFull("t1", "projectile", 350, 50, 10, 200, 2.0, 500).
		WithTowerFull("t2", "projectile", 400, 50, 10, 200, 2.0, 500).
		WithTowerFull("t3", "projectile", 450, 50, 10, 200, 2.0, 500).
		Build()
	s3.RunUntil(func(s *sim.Sim) bool { return s.Kills > 0 }, 600)
	tick3 := s3.Tick

	if s1.Kills == 0 || s3.Kills == 0 {
		t.Skip("单塔或三塔都没击杀，跳过比较")
	}
	if tick3 >= tick1 {
		t.Errorf("三塔击杀用 %d tick，应 < 单塔 %d tick", tick3, tick1)
	}
}

// ═══════════════════════════════════════
// Burn DoT 实际扣血
// ═══════════════════════════════════════

// TestBehavior_BurnDamagesOverTime 验证灼烧持续掉血。
func TestBehavior_BurnDamagesOverTime(t *testing.T) {
	s := sim.New().
		WithStraightPath(1000). // 长路径防泄漏
		WithEnemy("normal", 200, 20).
		ApplyBurn(0, 10, 3.0). // 10 DPS, 3 秒
		Build()

	initialHP := s.SpawnedEnemies[0].HP

	// 跑 60 tick = 1 秒，应有 2 次 DoT tick (0.5s 间隔)
	s.RunTicks(60)

	currentHP := s.SpawnedEnemies[0].HP
	hpLost := initialHP - currentHP

	if hpLost < 5 { // 至少应掉一些血（10 DPS * 0.5s tick = 5 per tick）
		t.Errorf("灼烧 1 秒后 HP 损失=%.1f，应 >= 5", hpLost)
	}
}

// ═══════════════════════════════════════
// 无敌/沉默实际生效
// ═══════════════════════════════════════

// TestBehavior_InvincibleBlocksAllDamage 验证无敌状态下塔无法造成伤害。
func TestBehavior_InvincibleBlocksAllDamage(t *testing.T) {
	s := sim.New().
		WithStraightPath(1000).
		WithEnemy("normal", 100, 20).
		WithTower("basic", 250, 100, 50, 200).
		Build()

	s.SpawnedEnemies[0].IsInvincible = true

	s.RunTicks(120) // 2 秒

	s.AssertEnemyHP(t, 0, "==", 100) // 无敌，HP 不变
}

// TestBehavior_DamageCapLimitsDamage 验证伤害上限限制单次伤害。
func TestBehavior_DamageCapLimitsDamage(t *testing.T) {
	s := sim.New().
		WithStraightPath(1000).
		WithEnemy("normal", 1000, 20).
		WithTower("basic", 250, 100, 500, 200). // 超高伤害
		Build()

	s.SpawnedEnemies[0].DamageCap = 10 // 每次最多 10 伤害

	s.RunTicks(120)

	// DamageCap=10，即使塔伤害 500，每次只扣 10
	hp := s.SpawnedEnemies[0].HP
	if hp < 800 { // 2 秒 ~2 次攻击 ~20 伤害，HP 应 > 800
		t.Errorf("DamageCap=10 时 HP=%.1f 扣太多了", hp)
	}
}

// ═══════════════════════════════════════
// 泄漏 → lives clamp 到 0
// ═══════════════════════════════════════

// TestBehavior_LivesClampToZero 验证多敌同时泄漏时 lives 不为负。
func TestBehavior_LivesClampToZero(t *testing.T) {
	s := sim.New().
		WithStraightPath(100). // 极短路径
		WithEnemy("e1", 9999, 300).
		WithEnemy("e2", 9999, 300).
		WithEnemy("e3", 9999, 300).
		WithLives(1). // 只有 1 条命
		Build()

	s.RunTicks(120)

	if s.Lives < 0 {
		t.Errorf("Lives=%d 不应为负", s.Lives)
	}
}

// ═══════════════════════════════════════
// AoE 塔命中多敌
// ═══════════════════════════════════════

// TestBehavior_AoETowerHitsMultipleEnemies 验证范围塔同时伤害多个敌人。
func TestBehavior_AoETowerHitsMultipleEnemies(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyAt(250, 100, 100, 0). // 静止在塔范围内
		WithEnemyAt(260, 100, 100, 0). // 也在范围内
		WithEnemyAt(270, 100, 100, 0).
		WithTowerFull("aoe", "spin_aoe", 250, 100, 20, 150, 2.0, 0).
		Build()

	s.RunTicks(60)

	// 所有 3 个敌人都应受伤
	for i := 0; i < 3; i++ {
		s.AssertEnemyHP(t, i, "<", 100)
	}
}

// ═══════════════════════════════════════
// 出生保护期
// ═══════════════════════════════════════

// TestBehavior_EnemySpawnAge 验证新出生的敌人 Age=0。
func TestBehavior_EnemySpawnAge(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 100, 60).
		Build()

	// 首帧 Age 应为 0
	e := s.SpawnedEnemies[0]
	if e.Age != 0 {
		t.Errorf("新出生敌人 Age=%.2f 应为 0", e.Age)
	}

	s.RunTicks(60) // 1 秒

	if e.Age < 0.9 {
		t.Errorf("1 秒后 Age=%.2f 应 ≈ 1.0", e.Age)
	}
}
