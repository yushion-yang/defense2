package event

// ── 塔事件载荷 ──────────────────────────────────

// TowerBuiltPayload 塔建造完成事件载荷。
type TowerBuiltPayload struct {
	TowerKey string // 塔类型 ID（如 "sentinel", "prism"）
	Cost     int    // 建造花费
}

// TowerUpgradedPayload 塔升级完成事件载荷。
type TowerUpgradedPayload struct {
	TowerKey string // 塔类型 ID
	Spent    int    // 本次升级花费
}

// TowerSoldPayload 塔出售事件载荷。
type TowerSoldPayload struct {
	TowerKey string // 塔类型 ID
	Refund   int    // 退还金额
}

// ── 敌人事件载荷 ─────────────────────────────────

// EnemyKilledPayload 敌人击杀事件载荷。
type EnemyKilledPayload struct {
	IsBoss    bool   // 是否 Boss
	KillerID  string // 击杀来源（"projectile"/"warden"）
	GoldValue int    // 击杀金币（预计算）
}

// EnemyLeakedPayload 敌人泄漏事件载荷。
type EnemyLeakedPayload struct{}

// ── 波次事件载荷 ─────────────────────────────────

// WaveStartedPayload 波次开始事件载荷。
type WaveStartedPayload struct {
	Wave   int  // 波次序号
	IsBoss bool // 是否 Boss 波（wave%5==0）
}

// WaveClearedPayload 波次清除事件载荷。
// 注：session.OnWaveCleared 有返回值，仍保持直调；此载荷仅用于通知订阅者。
type WaveClearedPayload struct {
	Wave    int  // 已清除的波次序号
	Perfect bool // 是否完美（无泄漏）
}
