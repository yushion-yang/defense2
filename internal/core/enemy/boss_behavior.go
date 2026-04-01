// boss_behavior.go — Boss 专属行为系统。
// 基于 boss-templates.json 配置，为 Boss 添加阶段转换、召唤小怪、光环等行为。
package enemy

import "math"

// BossAbility Boss 可携带的行为类型。
type BossAbility int

const (
	BossAbilPhase        BossAbility = 1 << iota // HP 阈值阶段转换
	BossAbilSpawnMinions                          // 定时召唤小怪
	BossAbilAura                                  // 友军加速光环
	BossAbilReflect                               // 反伤
	BossAbilGoldSteal                             // 偷金
)

// BossState Boss 行为运行时状态（挂载在 Enemy 上）。
type BossState struct {
	Abilities BossAbility // 位掩码：启用了哪些行为

	// Phase
	PhaseThresholds []float64 // HP 比例阈值（降序，如 [0.75, 0.5, 0.25]）
	CurrentPhase    int       // 当前阶段（0=初始，每过一个阈值+1）

	// SpawnMinions
	SpawnInterval float64 // 召唤间隔（秒）
	SpawnCount    int     // 每次召唤数量
	SpawnTimer    float64 // 召唤冷却倒计时
	SpawnArch     string  // 召唤的原型

	// Aura
	AuraRadius    float64 // 光环范围
	AuraSpeedUp   float64 // 光环加速比例
	AuraArmorUp   float64 // 光环护甲加成（预留）

	// Reflect
	ReflectRatio  float64 // 反伤比例

	// GoldSteal
	GoldPerHit    int     // 每次被命中偷取金币数
}

// TickBossPhase 检查 Boss HP 阶段转换。
// 每过一个阈值，Boss 加速 10%（叠加）。返回 true 表示本帧触发了阶段转换。
func TickBossPhase(e *Enemy) bool {
	bs := e.BossData
	if bs == nil || bs.Abilities&BossAbilPhase == 0 {
		return false
	}
	if e.MaxHP <= 0 {
		return false
	}
	ratio := e.HP / e.MaxHP
	triggered := false
	for bs.CurrentPhase < len(bs.PhaseThresholds) {
		if ratio <= bs.PhaseThresholds[bs.CurrentPhase] {
			bs.CurrentPhase++
			// 每阶段加速 10%
			e.BaseSpeed *= 1.1
			if e.SlowTimer <= 0 {
				e.Speed = e.BaseSpeed
			}
			triggered = true
		} else {
			break
		}
	}
	return triggered
}

// TickBossSpawnMinions 处理 Boss 定时召唤小怪。
// 返回需要生成的子怪数量和原型（由调用方执行实际 Spawn）。
func TickBossSpawnMinions(e *Enemy, dt float64) (count int, archetype string) {
	bs := e.BossData
	if bs == nil || bs.Abilities&BossAbilSpawnMinions == 0 {
		return 0, ""
	}
	bs.SpawnTimer -= dt
	if bs.SpawnTimer > 0 {
		return 0, ""
	}
	bs.SpawnTimer = bs.SpawnInterval
	return bs.SpawnCount, bs.SpawnArch
}

// TickBossAura 处理 Boss 友军光环。
func TickBossAura(boss *Enemy, enemies []*Enemy) {
	bs := boss.BossData
	if bs == nil || bs.Abilities&BossAbilAura == 0 {
		return
	}
	radiusSq := bs.AuraRadius * bs.AuraRadius
	for _, e := range enemies {
		if !e.Active || e.IsDying() || e.ID == boss.ID {
			continue
		}
		dx := e.X - boss.X
		dy := e.Y - boss.Y
		if dx*dx+dy*dy > radiusSq {
			continue
		}
		boost := e.BaseSpeed * bs.AuraSpeedUp
		if e.SlowTimer <= 0 {
			e.Speed = math.Max(e.Speed, e.BaseSpeed+boost)
		}
	}
}

// CalcBossReflectDamage 计算 Boss 反伤值。
func CalcBossReflectDamage(e *Enemy, incomingDamage float64) float64 {
	bs := e.BossData
	if bs == nil || bs.Abilities&BossAbilReflect == 0 {
		return 0
	}
	return incomingDamage * bs.ReflectRatio
}

// BossGoldSteal 返回 Boss 被命中时应偷取的金币数。
func BossGoldSteal(e *Enemy) int {
	bs := e.BossData
	if bs == nil || bs.Abilities&BossAbilGoldSteal == 0 {
		return 0
	}
	return bs.GoldPerHit
}

// DefaultBossState 创建默认 Boss 行为状态（阶段转换 + 光环）。
func DefaultBossState() *BossState {
	return &BossState{
		Abilities:       BossAbilPhase | BossAbilAura,
		PhaseThresholds: []float64{0.75, 0.5, 0.25},
		AuraRadius:      120,
		AuraSpeedUp:     0.3,
	}
}

// FullBossState 创建完整 Boss 行为状态（所有行为启用）。
func FullBossState(wave int) *BossState {
	bs := DefaultBossState()
	// 10 波后 Boss 开始召唤小怪
	if wave >= 10 {
		bs.Abilities |= BossAbilSpawnMinions
		bs.SpawnInterval = 20
		bs.SpawnCount = 3 + wave/10
		bs.SpawnArch = "normal"
		bs.SpawnTimer = bs.SpawnInterval
	}
	// 20 波后 Boss 有反伤
	if wave >= 20 {
		bs.Abilities |= BossAbilReflect
		bs.ReflectRatio = 0.15
	}
	// 30 波后 Boss 偷金
	if wave >= 30 {
		bs.Abilities |= BossAbilGoldSteal
		bs.GoldPerHit = 1
	}
	return bs
}
