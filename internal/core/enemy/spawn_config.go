// spawn_config.go — 敌人生成配置。
// SpawnConfig 携带原型数据，使 enemy 包独立于 config 包。
package enemy

// SpawnConfig 敌人生成时的原型配置参数。
// 由调用者从 config.EnemyArchetype 转换而来，传入 Pool.Spawn。
type SpawnConfig struct {
	Label       string  // 中文显示名称
	Sprite      string  // 精灵目录名（用于加载贴图）
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

	// 击杀奖励倍率
	RewardScale float64  // 原型奖励倍率（如 tank=1.35, runner=0.72）
	AbilityIDs  []string // 装配的能力 ID 列表

	// ── 能力系统字段 ──
	DamageCap         float64 // 坚韧(固定)
	DamageCapPercent  float64 // 坚韧(百分比)
	ProjectileBlockChance float64 // 弹幕盾
	ArmorFlat             float64 // 装甲固定减免
	EvasionChance         float64 // 闪避概率
	DashSpeedBoost        float64 // 受击冲刺速度提升
	DashDuration          float64 // 受击冲刺持续时间
	DashCooldown          float64 // 受击冲刺冷却
	PhaseDuration         float64 // 相位偏移免伤时间
	PhaseCooldown         float64 // 相位偏移冷却
	StrDrainRatio         float64 // 削强比例
	StrDrainInterval      float64 // 削强间隔
	StrDrainDuration      float64 // 削强持续时间
	DeathSpawnCount       int     // 死亡召唤数量
	DeathSpawnArch        string  // 死亡召唤原型
	PurgeInterval         float64 // 净化间隔
	PurgeImmuneDur        float64 // 净化免疫时间
	// 免疫
	CCImmune   bool // 全控制免疫
	SlowImmune bool // 减速免疫
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
