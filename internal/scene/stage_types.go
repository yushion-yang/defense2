// stage_types.go — StageScene 的类型定义和常量。
//
// 本文件集中定义 StageScene（游戏主场景）使用的所有枚举和数据结构：
//   - stageState:   游戏胜负状态（3 种）
//   - interactMode: 交互状态机（11 种模式），控制玩家当前能做什么、UI 显示什么
//   - GameStats:    单局统计数据，Stage → Result 传递
//   - StageOptions: 创建 StageScene 的配置选项
//
// 交互状态机是 StageScene 最核心的设计：
// 玩家的每次点击/按键都由 stage_input.go 根据当前 interactMode 分发到不同的处理逻辑。
// 模式之间的切换遵循严格的状态流转规则（见下方 ASCII 图）。
package scene

// stageState 游戏主场景的胜负状态。
// 由 pipeline 中的 endgame system 每帧检测并设置。
// statePlaying 时正常 tick；stateVictory/stateDefeat 时触发结算动画后跳转 ResultScene。
type stageState int

const (
	statePlaying stageState = iota // 游戏进行中（正常 tick）
	stateVictory                   // 玩家胜利（所有波次清完且无泄漏）
	stateDefeat                    // 玩家失败（生命值归零）
)

// interactMode 交互状态机——StageScene 输入系统的核心。
//
// 每种模式决定了：
//   - 哪些 UI 面板可见（建塔面板、塔信息面板、暂停菜单等）
//   - 点击/按键如何被分发处理（见 stage_input.go handleInput）
//   - 相机拖拽是否可用（modeTowerSel 下禁用，防止点击取消被误判为拖拽）
//   - ESC 键的行为（关闭面板 or 进入暂停）
//
// 状态流转图（主要路径）：
//
//	                ┌─────────────────────────┐
//	                │        modeIdle         │ ◄── 默认状态（观察游戏）
//	                └──┬────┬────┬────┬───────┘
//	   点击"造塔"/B    │    │    │    │ 点击已有塔
//	                ▼    │    │    │    ▼
//	         ┌──────────┐│    │    │ ┌──────────────┐
//	         │modeBuild-││    │    │ │modeTowerSel  │──→ modeUpgrade（选能力）
//	         │  Menu    ││    │    │ │（显示面板+射程）│
//	         └──┬───────┘│    │    │ └──┬───────────┘
//	选择塔型    │        │    │    │    │ 点空地/ESC
//	         ▼        │    │    │    ▼
//	   ┌───────────┐  │    │    │  → modeIdle
//	   │modeBuild- │  │    │    │
//	   │  Place    │  │    │    │ 点击道具/I
//	   │（放塔模式） │  │    │    │    ▼
//	   └───────────┘  │    │    │ ┌──────────────┐
//	放塔后→modeIdle   │    │    │ │modeItemPanel │──→ modeItemDrag
//	  ESC→modeIdle    │    │    │ └──────────────┘
//	                  │    │    │
//	           菜单/ESC/P  ▼    │
//	            ┌───────────┐  │ 波前选择
//	            │modePaused │  │    ▼
//	            └───────────┘  │ modeWardenSelect
type interactMode int

const (
	modeIdle         interactMode = iota // 空闲：观察游戏，可点击塔/空地/UI
	modeBuildMenu                        // 建塔面板打开：展示可建塔类型卡片供选择
	modeBuildPlace                       // 放塔模式：已选塔型，点击可建位放塔（放完回 idle）
	modeTowerSel                         // 塔选中：显示信息面板（属性/能力/buff）+ 射程环
	modeSpawnMenu                        // 造怪菜单（测试模式）：选择要生成的敌人原型
	modeSpawnPlace                       // 造怪放置（测试模式）：点击地图放置静怪或动怪
	modePaused                           // 暂停菜单：继续/设置/重开/退出四个选项
	modeWardenSelect                     // 战灵选择覆盖层：战斗开始前选择战灵（全屏遮罩）
	modeUpgrade                          // 能力选择覆盖层：ChoicePanel 3 选 1，吃掉所有输入
	modeItemPanel                        // 道具面板打开：展示背包中的道具卡片
	modeItemDrag                         // 拖拽道具中：跟踪指针位置，松手时应用到目标塔
)

// GameStats 单局详细统计数据，在战斗中由 StageScene 逐步累积，
// 局结束时传递给 ResultScene 用于展示结算面板和计算成就。
type GameStats struct {
	// ── 战斗统计 ──
	TotalKills    int // 总击杀数（含 Boss、分裂子体）
	BossKills     int // Boss 击杀数
	MaxKillStreak int // 最大连杀数（快速连续击杀）
	LeaksTotal    int // 泄漏数（敌人抵达终点，每次 -1 生命）

	// ── 进度 ──
	TotalWaves int     // 当前到达的波次编号
	MaxWave    int     // 该地图的总波次数
	TimePlayed float64 // 游戏时间（秒），暂停时不计

	// ── 经济 ──
	GoldEarned int // 累计获得金币（击杀奖励 + 波次奖金 + 完美奖金）
	GoldSpent  int // 累计消耗金币（建塔 + 升级 + 解锁槽位）

	// ── 操作统计 ──
	TowersBuilt int // 建塔次数
	TowersSold  int // 卖塔次数
	ItemsUsed   int // 道具使用次数

	// ── MVP 塔 ──（结算面板展示"最佳塔"）
	BestTowerKey   string // 击杀最多的塔类型 Key（如 "basic_0_1"）
	BestTowerName  string // 击杀最多的塔显示名（如 "哨兵"）
	BestTowerKills int    // 该塔的击杀数
}

// StageOptions 创建 StageScene 的配置选项。
// 由 CampaignSelectScene/TestSelectScene/autoplay 构建，传入 NewStageSceneWithOpts()。
// 零值字段使用合理默认值，方便 autoplay 快速构建测试场景。
type StageOptions struct {
	MapID        string // 地图 ID（如 "map_01"），必填
	WardenType   string // 战灵类型（如 "guardian"），空串=不使用战灵
	ModeID       string // 游戏模式 ID（默认 "casual"；可选 "endless"/"timed"/...）
	DifficultyID string // 难度 ID（默认 "normal"；可选 "easy"/"hard"/"extreme"）
	Gold         int    // 初始金币，0 = 由难度配置决定
	Lives        int    // 初始生命，0 = 由难度配置决定（easy=25/normal=20/hard=15/extreme=10）
	Waves        int    // 波次数，0 = 地图默认；-1 = 无波次（纯测试沙盒）
	TestMode     bool   // 测试模式：启用调试面板(D键)、造怪菜单、无限金币等
	ScenarioID   string // autoplay 测试场景 ID（如 "attack-style-coverage"）
	EnemyFilter  string // 敌人过滤器（ground-only/flying-only/elite-only/boss-only/stress/dummy 等）
	ManualWave   bool   // 仅手动开波（禁用自动波次推进，用于调试特定波次）
	AIEnabled    bool   // 启用 AI 玩家（合作模式，Phase 1: 右半区域自动造塔）
}
