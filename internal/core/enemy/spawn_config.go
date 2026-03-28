// spawn_config.go — 敌人生成配置。
// SpawnConfig 携带原型数据，使 enemy 包独立于 config 包。
package enemy

// SpawnConfig 敌人生成时的原型配置参数。
// 由调用者从 config.EnemyArchetype 转换而来，传入 Pool.Spawn。
type SpawnConfig struct {
	Label       string  // 中文显示名称
	HpScale     float64 // 血量倍率（应用于 baseHP）
	SpeedScale  float64 // 速度倍率（应用于 baseSpeed）
	Radius      float64 // 碰撞半径（像素绝对值）
	Boss        bool    // 是否为 Boss
	ShieldScale float64 // 护盾倍率（基于最终 HP，0 = 无护盾）
	Reward      int     // 击杀奖励金币
}

// DefaultSpawnConfig 返回默认生成配置（普通敌人）。
func DefaultSpawnConfig() *SpawnConfig {
	return &SpawnConfig{
		HpScale:    1,
		SpeedScale: 1,
		Radius:     8,
	}
}
