package mascot

// StageSnapshot is a pure-value snapshot of stage game state.
// Populated by the scene layer, consumed by condition evaluators.
type StageSnapshot struct {
	Wave        int
	MaxWaves    int
	Lives       int
	MaxLives    int
	Gold        int
	EnemyCount  int
	TowerCount  int
	Kills       int
	MultiKill   int     // current kill streak
	ElapsedSecs float64 // seconds since stage start
	IsBossWave  bool
	WaveActive  bool

	// Performance (from PerfTracker)
	FPS        float64
	AvgFrameMs float64
	HeapMB     float64
	GCPauseUs  uint64

	// Game state
	Paused        bool // true when game is paused
	Victory       bool // true when stage ended in victory
	Defeat        bool // true when stage ended in defeat
	IsClassicMode bool // true in classic campaign mode (no mascot ability)

	// UI Context
	InteractMode       int      // 0=idle, 1=buildMenu, 3=towerSel, 7=wardenSel, 8=upgrade, 9=itemPanel
	SelectedTowerLabel string   // tower name when modeTowerSel
	SelectedTowerStyle string   // attack style name
	ChoiceAbilities    []string // ability labels during modeUpgrade
	WardenType         string   // current warden type key
	QualityLevel       int      // 0=High, 1=Medium, 2=Low
}

// GameContext is the complete context for condition evaluation.
type GameContext struct {
	SceneName   string
	HourOfDay   int     // 0-23, from system clock
	SessionSecs float64 // wall-clock seconds since game launch

	// Stage data (zero values when not in stage)
	InStage bool
	StageSnapshot

	// Player progress (from persistence)
	TotalWins   int
	TotalKills  int
	TotalGames  int
	MapsCleared int
}
