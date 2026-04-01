// stage_types.go — StageScene 的类型定义和常量。
package scene

// stageState 游戏主场景的胜负状态。
type stageState int

const (
	statePlaying stageState = iota // 游戏进行中
	stateVictory                   // 玩家胜利
	stateDefeat                    // 玩家失败
)

// interactMode 交互状态机。
// 控制玩家当前的操作模式和 UI 显示。
//
// 状态流转图：
//
//	                ┌─────────────────────────┐
//	                │        modeIdle         │ ◄── 默认状态
//	                └──┬────┬────┬────┬───────┘
//	   点击"造塔"/B    │    │    │    │ 点击已有塔
//	                ▼    │    │    │    ▼
//	         ┌──────────┐│    │    │ ┌──────────────┐
//	         │modeBuild-││    │    │ │modeTowerSel  │
//	         │  Menu    ││    │    │ │（显示面板+射程）│
//	         └──┬───────┘│    │    │ └──┬───────────┘
//	选择塔型    │        │    │    │    │ 点空地/ESC
//	         ▼        │    │    │    ▼
//	   ┌───────────┐  │    │    │  → modeIdle
//	   │modeBuild- │  │    │    │
//	   │  Place    │  │    │    │
//	   │（放塔模式） │  │    │    │
//	   └───────────┘  │    │    │
//	放塔后保持/ESC退出│    │    │
//	                  │    │    │
//	           菜单按钮   ▼    │
//	            ┌───────────┐  │
//	            │modePaused │  │
//	            └───────────┘  │
type interactMode int

const (
	modeIdle         interactMode = iota // 空闲：观察游戏
	modeBuildMenu                        // 建塔面板打开：选择塔类型
	modeBuildPlace                       // 放塔模式：已选塔型，点击可建位放塔
	modeTowerSel                         // 塔选中：显示信息面板+射程
	modeSpawnMenu                        // 造怪菜单：选择敌人类型
	modeSpawnPlace                       // 造怪放置：点击地图放置敌人
	modePaused                           // 暂停菜单
	modeWardenSelect                     // 战灵选择覆盖层
	modeUpgrade                          // 能力选择覆盖层
	modeItemPanel                        // 道具面板打开
	modeItemDrag                         // 拖拽道具中
)

// GameStats 单局详细统计数据，从 StageScene 传递到 ResultScene。
type GameStats struct {
	TotalKills    int     // 总击杀数
	TotalWaves    int     // 到达波次
	MaxWave       int     // 总波次数
	GoldEarned    int     // 累计获得金币
	GoldSpent     int     // 累计消耗金币
	TowersBuilt   int     // 建塔次数
	TowersSold    int     // 卖塔次数
	ItemsUsed     int     // 道具使用次数
	MaxKillStreak int     // 最大连杀数
	TimePlayed    float64 // 游戏时间（秒）
	BossKills     int     // Boss 击杀数
	LeaksTotal    int     // 泄漏数
	BestTowerKey  string  // 击杀最多的塔类型 Key
	BestTowerName string  // 击杀最多的塔显示名
	BestTowerKills int    // 该塔的击杀数
}

// StageOptions 创建 StageScene 的配置选项。
type StageOptions struct {
	MapID        string
	WardenType   string
	ModeID       string // 游戏模式 ID（默认 "campaign"）
	DifficultyID string // 难度 ID（默认 "normal"）
	Gold         int    // 0 = 默认（由难度决定）
	Lives        int    // 0 = 默认 20
	Waves        int    // 0 = 地图默认; -1 = 无波次
	TestMode     bool   // 测试模式（启用调试面板）
	ScenarioID   string // 测试场景 ID
	EnemyFilter  string // ground-only/flying-only/elite-only/boss-only/all-static/mixed/stress/dummy/none
	ManualWave   bool   // 仅手动开波
}
