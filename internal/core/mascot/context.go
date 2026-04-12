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
