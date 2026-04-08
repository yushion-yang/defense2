// boss_behavior.go — Boss 行为系统（待迁移到能力系统）。
// Boss 行为将通过能力装配实现，当前保留类型定义和空操作函数。
package enemy

// BossState Boss 行为状态（待迁移到能力系统）。
type BossState struct{}

// FullBossState 创建 Boss 状态（当前返回 nil，待能力系统实现）。
func FullBossState(_ int) *BossState { return nil }

// TickBossPhase Boss 阶段转换（待迁移）。
func TickBossPhase(_ *Enemy) {}

// TickBossSpawnMinions Boss 召唤小怪（待迁移）。返回 (数量, 原型名)。
func TickBossSpawnMinions(_ *Enemy, _ float64) (int, string) { return 0, "" }

// TickBossAura Boss 光环（待迁移）。
func TickBossAura(_ *Enemy, _ *Pool) {}
