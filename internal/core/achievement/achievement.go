// achievement.go — Achievement tracking and persistence.
// Tracks 15 achievements across 4 tiers, with per-session counters
// for in-game checks and persistent unlock state via Storage.
package achievement

import (
	"defense2/internal/core/persistence"
	"sync"
)

// Tier represents the rarity/difficulty tier of an achievement.
type Tier int

const (
	TierBronze  Tier = iota // easy, introductory
	TierSilver              // moderate skill
	TierGold                // significant skill
	TierDiamond             // mastery
)

// Achievement defines a single achievement.
type Achievement struct {
	ID          string
	Name        string // display name (Chinese)
	Description string // unlock condition description
	Tier        Tier
	Threshold   int // 数值阈值（如 10=建造10塔, 100=击杀100, 0=无数值条件）
}

// All is the full list of achievements in display order.
// Threshold: 数值条件（0=无数值条件，由特殊逻辑判断）。
var All = []Achievement{
	{"first_win", "初次胜利", "通关任意地图", TierBronze, 0},
	{"builder_10", "塔防新手", "累计建造10座塔", TierBronze, 10},
	{"first_boss", "首个Boss", "击杀首个Boss", TierBronze, 0},
	{"perfect_star", "完美主义", "任意关卡3星通关", TierSilver, 0},
	{"killstreak_20", "连杀达人", "单局20连杀", TierSilver, 20},
	{"item_master", "道具大师", "单局使用10个道具", TierSilver, 10},
	{"all_towers", "全能战士", "单局建造5种不同塔", TierSilver, 5},
	{"rich", "富甲一方", "单局持有1000金币", TierSilver, 1000},
	{"all_3star", "全图三星", "所有地图3星通关", TierGold, 0},
	{"centurion", "百杀", "单局击杀100敌人", TierGold, 100},
	{"no_leak_hard", "零泄漏", "Hard难度无泄漏通关", TierGold, 0},
	{"speedrun", "速通", "10分钟内通关", TierGold, 600},
	{"extreme_master", "大师", "Extreme难度通关", TierDiamond, 0},
	{"extreme_perfect", "完美大师", "Extreme难度3星通关", TierDiamond, 0},
	{"endless_50", "不灭传说", "Endless模式坚持50波", TierDiamond, 50},
}

// ThresholdOf returns the Threshold for a given achievement ID (0 if not found).
func ThresholdOf(id string) int {
	for _, a := range All {
		if a.ID == id {
			return a.Threshold
		}
	}
	return 0
}

// storageKey is the persistence key for achievement data.
const storageKey = "achievements"

// Tracker tracks achievement progress and unlocks.
type Tracker struct {
	mu      sync.Mutex
	storage persistence.Storage // nil = no persistence

	Unlocked map[string]bool `json:"unlocked"`

	// Persistent cumulative counters
	TotalTowersBuilt int `json:"total_towers_built"`

	// Session counters (reset per game, not persisted)
	SessionKills      int
	SessionMaxStreak  int
	SessionItemsUsed  int
	SessionTowerTypes map[string]bool
	SessionMaxGold    int
}

// NewTracker creates a tracker with optional persistence storage.
func NewTracker(store persistence.Storage) *Tracker {
	t := &Tracker{
		storage:           store,
		Unlocked:          make(map[string]bool),
		SessionTowerTypes: make(map[string]bool),
	}
	t.Load()
	return t
}

// Unlock marks an achievement as unlocked. Returns true if newly unlocked.
// Saves to disk immediately on new unlock.
func (t *Tracker) Unlock(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Unlocked[id] {
		return false
	}
	t.Unlocked[id] = true
	t.saveLocked()
	return true
}

// IsUnlocked checks if an achievement is unlocked.
func (t *Tracker) IsUnlocked(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Unlocked[id]
}

// UnlockedCount returns number of unlocked achievements.
func (t *Tracker) UnlockedCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.Unlocked)
}

// ResetSession resets per-game counters for a new stage.
func (t *Tracker) ResetSession() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.SessionKills = 0
	t.SessionMaxStreak = 0
	t.SessionItemsUsed = 0
	t.SessionTowerTypes = make(map[string]bool)
	t.SessionMaxGold = 0
}

// IncrTowersBuilt increments the persistent total towers built counter.
// Saves to disk.
func (t *Tracker) IncrTowersBuilt() {
	t.mu.Lock()
	t.TotalTowersBuilt++
	t.saveLocked()
	t.mu.Unlock()
}

// Save persists the tracker state to storage.
func (t *Tracker) Save() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.saveLocked()
}

// saveLocked writes state to storage (caller must hold mu).
func (t *Tracker) saveLocked() {
	if t.storage == nil {
		return
	}
	// Persist only the durable fields via a slim struct.
	data := struct {
		Unlocked         map[string]bool `json:"unlocked"`
		TotalTowersBuilt int             `json:"total_towers_built"`
	}{
		Unlocked:         t.Unlocked,
		TotalTowersBuilt: t.TotalTowersBuilt,
	}
	_ = t.storage.Set(storageKey, &data)
}

// Load restores tracker state from storage.
func (t *Tracker) Load() {
	if t.storage == nil {
		return
	}
	var data struct {
		Unlocked         map[string]bool `json:"unlocked"`
		TotalTowersBuilt int             `json:"total_towers_built"`
	}
	if err := t.storage.Get(storageKey, &data); err != nil {
		return // no saved data yet
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if data.Unlocked != nil {
		t.Unlocked = data.Unlocked
	}
	t.TotalTowersBuilt = data.TotalTowersBuilt
}

// NameByID returns the display name for an achievement ID, or "" if not found.
func NameByID(id string) string {
	for _, a := range All {
		if a.ID == id {
			return a.Name
		}
	}
	return ""
}
