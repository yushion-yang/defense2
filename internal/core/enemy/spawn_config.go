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
	Reward      int     // 击杀奖励金币

	// ── 行为配置 ──
	Behavior        string  // 行为类型标识（"healer"/"stealth"/"splitter"/"buffer"/"regenerator"/""）
	StealthDuration float64 // 隐身持续时间（秒）
	SplitCount      int     // 分裂子体数量
	SplitScale      float64 // 子体血量倍率
	SplitHPRatio    float64 // 子体 HP 占父体 MaxHP 的比例（默认 0.3）
	SplitSpeedScale float64 // 子体速度倍率（默认 1.4）
	HealScale       float64 // 治疗量倍率（占目标 MaxHP 比例）
	HealRadius      float64 // 治疗范围（像素）
	HealInterval    float64 // 治疗间隔（秒）
	AuraRange       float64 // 光环范围（像素）
	AuraSpeedUp     float64 // 光环移速加成比例

	// 传送（teleporter 原型）
	TeleportInterval float64 // 传送间隔（秒，0=不传送）
	TeleportSkip     int     // 每次传送跳过的路径段数

	// 移动类型
	MovementType string // "ground"(默认) / "flying"

	// 击杀奖励倍率
	RewardScale float64 // 原型奖励倍率（如 tank=1.35, runner=0.72）
}

// DefaultSpawnConfig 返回默认生成配置（普通敌人）。
func DefaultSpawnConfig() *SpawnConfig {
	return &SpawnConfig{
		HpScale:    1,
		SpeedScale: 1,
		Radius:     8,
		Reward:     15, // 默认击杀奖金
	}
}
