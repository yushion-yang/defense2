// autoplay_types.go — AutoPlayer 接口和数据类型定义。
// 定义在 scene 包中以避免 scene <-> autoplay 循环导入。
package scene

import "defense2/internal/core/telemetry"

// AutoPlayer 自动对局驱动接口。
// 由 autoplay.Controller 实现，注入到 StageScene 中。
type AutoPlayer interface {
	// OnUpdate 每帧调用，返回要执行的操作列表。
	OnUpdate(snap AutoPlaySnapshot) []AutoPlayAction
	// OnGameEnd 游戏结束时调用。
	OnGameEnd(snap AutoPlaySnapshot, won bool)
	// Done 返回 true 表示自动对局完成，应退出游戏循环。
	Done() bool
}

// AutoPlaySnapshot 游戏状态快照（只读），传给 AutoPlayer。
type AutoPlaySnapshot struct {
	Tick         int
	Gold         int
	Lives        int
	Wave         int
	MaxWaves     int
	WaveActive   bool
	GameOver     bool
	Victory      bool
	WardenReady  bool
	InteractMode int
	Enemies      []AutoPlayEnemy
	Towers       []AutoPlayTower
	BuildCells   []AutoPlayCell
	TowerDefs    []AutoPlayTowerDef

	// 扩展字段：用于深层异常检测
	WavesCleared    int     // 已清除波次数（用于能力解锁回归检测）
	TotalKills      int     // 累计击杀数
	TotalLeaked     int     // 累计泄漏数（通过 lives 变化推算）
	EnemyPoolCount  int     // 敌人池活跃数
	ProjectileCount int     // 弹射物池活跃数
	TowerCount      int     // 塔池活跃数
	WardenX         float64 // 战灵 X 坐标（未选择为 0）
	WardenY         float64 // 战灵 Y 坐标
	MapPixelW       float64 // 地图像素宽度
	MapPixelH       float64 // 地图像素高度
	GameSpeed       int     // 当前游戏速度倍率

	// 遥测数据快照
	Telemetry telemetry.TelemetrySnapshot

	// 地图静态数据（仅首帧填充，后续为 nil）
	MapInfo *AutoPlayMapInfo
}

// AutoPlayMapInfo 地图静态信息（路径、网格等）。
type AutoPlayMapInfo struct {
	Grid      [][]int          // 0=empty, 1=path, 2=buildable, 4=spawn, 5=base
	CellSize  int              // 单元格像素边长
	Rows      int              // 行数
	Cols      int              // 列数
	Waypoints []AutoPlayPoint  // 默认路径点序列
	Paths     []AutoPlayPath   // 多路径入口列表
	MultiPath bool             // 是否多路径地图
}

// AutoPlayPoint 像素坐标点。
type AutoPlayPoint struct {
	X, Y float64
}

// AutoPlayPath 多路径入口信息。
type AutoPlayPath struct {
	ID        string
	Waypoints []AutoPlayPoint
	Weight    float64
}

// AutoPlayEnemy 敌人快照。
type AutoPlayEnemy struct {
	ID        int
	X, Y      float64
	HP, MaxHP float64
	Speed     float64
	Archetype string
	Boss      bool
	Active    bool
	Dying     bool
	// 状态效果（视觉目录用）
	IsSlowed   bool
	IsStunned  bool
	IsBurning  bool
	IsBleeding bool
	IsRooted   bool
	IsHit      bool // HitFlash > 0（受击闪白）

	BaseSpeed       float64
	DamageAmplify   float64
	AbilitySilenced bool
	PhaseActive     bool
	ArmorFlat       float64
	EvasionChance   float64
	DamageCap       float64
	DamageCapPct    float64
	HealRadius      float64
	BuffRadius      float64
	SplitCount      int
	AbilityIDs      []string
	PathIndex       int // 当前路径点索引
	PathTotal       int // 路径总点数
}

// AutoPlayTower 已建塔快照。
type AutoPlayTower struct {
	Key         string
	Row, Col    int
	X, Y        float64
	Damage      float64
	Range       float64
	Cost        int
	Strength    int
	Abilities   []string // 已装载能力列表
	AttackStyle string   // 攻击方式 ID
	HasTarget   bool     // 是否正在锁定目标（视觉目录用）
	AttackSpeed float64
	BaseDamage  float64
	Kills       int // 累计击杀数
}

// AutoPlayCell 可建造位置。
type AutoPlayCell struct {
	Row, Col int
	X, Y     float64
}

// AutoPlayTowerDef 可用塔类型定义。
type AutoPlayTowerDef struct {
	Key    string
	Cost   int
	Range  float64
	Damage float64
	Index  int
}

// APActionType 自动操作类型。
type APActionType int

const (
	APActionBuild        APActionType = iota // 建造塔
	APActionUpgrade                          // 升级塔
	APActionSell                             // 出售塔
	APActionStartWave                        // 开始下一波
	APActionSelectWarden                     // 选择战灵
	APActionAddAbility                       // 给塔添加能力
)

// AutoPlayAction 自动操作指令。
type AutoPlayAction struct {
	Type        APActionType
	TowerKey    string // Build: 塔类型 Key
	Row, Col    int    // Build/Upgrade/Sell: 网格坐标
	WardenKey   string // SelectWarden: 战灵类型
	AbilityName string // AddAbility: 能力名称
}
