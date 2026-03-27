package config

// MapConfig represents a level map loaded from JSON.
type MapConfig struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Cols       int        `json:"cols"`
	Rows       int        `json:"rows"`
	CellSize   int        `json:"cellSize"`
	Theme      string     `json:"theme"`
	Waves      int        `json:"waves"`
	Difficulty string     `json:"difficulty"`
	Grid       [][]int    `json:"grid"`
	PathOrder  [][2]int   `json:"pathOrder"`
	Entries    []MapEntry `json:"entries,omitempty"`
	PathOrders map[string][][2]int `json:"pathOrders,omitempty"`
}

// MapEntry represents a spawn entry point for multi-path maps.
type MapEntry struct {
	ID     string  `json:"id"`
	Cell   [2]int  `json:"cell"`
	Weight float64 `json:"weight"`
}

// Grid cell type constants.
const (
	CellEmpty    = 0
	CellPath     = 1
	CellBuildable = 2
	CellSpawn    = 4
	CellBase     = 5
	CellHeroBase = 6
)
