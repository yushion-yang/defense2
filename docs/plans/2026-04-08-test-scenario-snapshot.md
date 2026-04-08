# Test Scenario Snapshot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Save/load tower layouts + scene config as JSON files, enabling reusable custom test scenarios.

**Architecture:** New `config/scenarios/` directory with `//go:embed`, `ScenarioData` struct in `internal/config/scenario_config.go`, `PlaceFromSnapshot` in tower pool, save/restore in stage, custom category in test select.

**Tech Stack:** Go, embed.FS, encoding/json

---

### Task 1: ScenarioData types + LoadScenarios

**Files:**
- Create: `internal/config/scenario_config.go`
- Test: `internal/config/scenario_config_test.go`

**Step 1: Write the failing test**

```go
// internal/config/scenario_config_test.go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScenarioData(t *testing.T) {
	raw := []byte(`{
		"id": "test-save",
		"name": "Test Save",
		"description": "unit test",
		"mapID": "map_test",
		"gold": 9999,
		"lives": 999,
		"waves": 10,
		"enemyFilter": "mixed",
		"manualWave": true,
		"towers": [{
			"row": 3, "col": 5,
			"key": "prism",
			"abilitySlots": ["scatter","","auraDamage","","",""],
			"damageTier": "A", "speedTier": "B", "rangeTier": "S",
			"baseDamage": 12.0, "potentialDamage": 8.0,
			"baseSpeed": 1.5, "potentialSpeed": 0.5,
			"baseRange": 120.0, "potentialRange": 30.0
		}]
	}`)
	sd, err := ParseScenarioData(raw)
	require.NoError(t, err)
	assert.Equal(t, "test-save", sd.ID)
	assert.Equal(t, "map_test", sd.MapID)
	assert.Equal(t, 9999, sd.Gold)
	assert.True(t, sd.ManualWave)
	require.Len(t, sd.Towers, 1)
	tw := sd.Towers[0]
	assert.Equal(t, 3, tw.Row)
	assert.Equal(t, 5, tw.Col)
	assert.Equal(t, "prism", tw.Key)
	assert.Equal(t, [6]string{"scatter", "", "auraDamage", "", "", ""}, tw.AbilitySlots)
	assert.Equal(t, "A", tw.DamageTier)
	assert.InDelta(t, 12.0, tw.BaseDamage, 0.01)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestParseScenarioData -v`
Expected: FAIL (ParseScenarioData not defined)

**Step 3: Write implementation**

```go
// internal/config/scenario_config.go
package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// ScenarioData describes a reusable test scenario (tower layout + scene config).
type ScenarioData struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	MapID       string          `json:"mapID"`
	Gold        int             `json:"gold"`
	Lives       int             `json:"lives"`
	Waves       int             `json:"waves"`
	EnemyFilter string          `json:"enemyFilter"`
	ManualWave  bool            `json:"manualWave"`
	Towers      []TowerSnapshot `json:"towers"`
}

// TowerSnapshot captures a placed tower's full state for scenario restore.
type TowerSnapshot struct {
	Row            int        `json:"row"`
	Col            int        `json:"col"`
	Key            string     `json:"key"`
	AbilitySlots   [6]string  `json:"abilitySlots"`
	DamageTier     string     `json:"damageTier"`
	SpeedTier      string     `json:"speedTier"`
	RangeTier      string     `json:"rangeTier"`
	BaseDamage     float64    `json:"baseDamage"`
	PotentialDamage float64   `json:"potentialDamage"`
	BaseSpeed      float64    `json:"baseSpeed"`
	PotentialSpeed float64    `json:"potentialSpeed"`
	BaseRange      float64    `json:"baseRange"`
	PotentialRange float64    `json:"potentialRange"`
}

// ParseScenarioData parses a single scenario JSON.
func ParseScenarioData(data []byte) (*ScenarioData, error) {
	var sd ScenarioData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parse scenario: %w", err)
	}
	return &sd, nil
}

// LoadScenarios loads all scenario JSON files from config/scenarios/.
func LoadScenarios() ([]*ScenarioData, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load scenarios: dataFS not initialized")
	}
	dir := "config/scenarios"
	entries, err := fs.ReadDir(dataFS, dir)
	if err != nil {
		return nil, nil // directory doesn't exist = no custom scenarios
	}
	var results []*ScenarioData
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := dataFS.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		sd, err := ParseScenarioData(data)
		if err != nil {
			continue
		}
		results = append(results, sd)
	}
	return results, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestParseScenarioData -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/scenario_config.go internal/config/scenario_config_test.go
git commit -m "feat: add ScenarioData types and LoadScenarios"
```

---

### Task 2: Embed config/scenarios/ in data.go

**Files:**
- Create: `config/scenarios/.gitkeep` (placeholder so embed pattern has at least one match)
- Modify: `data.go:9` (add scenarios to DataFS embed)

**Step 1: Create directory with placeholder**

```bash
mkdir -p config/scenarios
touch config/scenarios/.gitkeep
```

**Step 2: Update data.go embed directive**

Change line 9 from:
```go
//go:embed config/levels/*.json config/towers/*.json config/enemies/*.json config/wardens/*.json config/abilities/*.json config/settings.json config/level-list.json
```
To:
```go
//go:embed config/levels/*.json config/towers/*.json config/enemies/*.json config/wardens/*.json config/abilities/*.json config/settings.json config/level-list.json config/scenarios
```

Note: Use `config/scenarios` (directory pattern) rather than `config/scenarios/*.json` to avoid "no matching files" error when no JSON files exist yet. The `embed` directive with a directory name embeds the whole directory tree including `.gitkeep`.

**Step 3: Verify build**

Run: `go build ./cmd/game/`
Expected: BUILD SUCCESS

**Step 4: Commit**

```bash
git add config/scenarios/.gitkeep data.go
git commit -m "feat: embed config/scenarios/ directory in DataFS"
```

---

### Task 3: PlaceFromSnapshot in tower pool

**Files:**
- Modify: `internal/core/tower/pool.go` (add PlaceFromSnapshot method)
- Test: `internal/core/tower/pool_test.go`

**Step 1: Write the failing test**

```go
// In pool_test.go (add test)
func TestPlaceFromSnapshot(t *testing.T) {
	p := NewPool(4)
	def := TowerDef{
		Key: "prism", Label: "Prism", Cost: 70,
		AttackStyleID: StyleWideBeam, ProjectileSpeed: 300,
		CfgBaseDamage: 10, PotentialDamage: 5,
		CfgBaseSpeed: 1.0, PotentialSpeed: 0.5,
		CfgBaseRange: 100, PotentialRange: 20,
	}
	snap := config.TowerSnapshot{
		Row: 3, Col: 5,
		Key: "prism",
		AbilitySlots: [6]string{"scatter", "", "auraDamage", "", "", ""},
		DamageTier: "A", SpeedTier: "B", RangeTier: "S",
		BaseDamage: 12.0, PotentialDamage: 8.0,
		BaseSpeed: 1.5, PotentialSpeed: 0.5,
		BaseRange: 120.0, PotentialRange: 30.0,
	}
	tw := p.PlaceFromSnapshot(3, 5, 200.0, 300.0, def, snap)
	require.NotNil(t, tw)
	assert.Equal(t, 3, tw.Row)
	assert.Equal(t, 5, tw.Col)
	assert.Equal(t, "prism", tw.Key)
	assert.Equal(t, "A", tw.DamageTier)
	assert.InDelta(t, 12.0, tw.BaseDamage, 0.01)
	assert.InDelta(t, 8.0, tw.PotentialDamage, 0.01)
	assert.Equal(t, 1, p.Count)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/core/tower/ -run TestPlaceFromSnapshot -v`
Expected: FAIL (PlaceFromSnapshot not defined)

**Step 3: Write implementation**

Add to `internal/core/tower/pool.go`:

```go
// PlaceFromSnapshot places a tower from a saved snapshot, skipping random rolls.
// Uses the snapshot's base/potential/tier values directly instead of RollTowerStats.
func (p *Pool) PlaceFromSnapshot(row, col int, cx, cy float64, def TowerDef, snap config.TowerSnapshot) *Tower {
	for i := range p.towers {
		if !p.towers[i].Active {
			t := &p.towers[i]
			t.Row = row
			t.Col = col
			t.X = cx
			t.Y = cy
			t.Range = def.Range
			t.Damage = def.Damage
			t.AttackSpeed = def.AttackSpeed
			t.FireTimer = 0
			t.Cost = def.Cost
			t.Key = def.Key
			t.InstanceKey = fmt.Sprintf("%s_%d_%d", def.Key, row, col)
			t.Label = def.Label
			t.Abilities = def.Abilities
			t.Color = def.Color
			t.Active = true
			t.AttackStyleID = def.AttackStyleID
			t.ProjectileSpeed = def.ProjectileSpeed
			t.ChargeProgress = 0
			t.ChargeReady = false
			t.SpinAngle = 0
			t.SpinActive = 0
			t.AuraPulse = 0
			t.Branch = ""
			t.Level = 1
			t.SpriteKey = spriteKeyForStyle(def.AttackStyleID)
			t.AbilitySlots = [6]string{}
			t.UnlockOrder = RollUnlockOrder()
			t.Target = nil
			t.Kills = 0
			t.StackTarget = 0
			t.StackCount = 0
			t.LastPercentHpTarget = 0
			t.GoldCooldown = 0
			t.Angle = 0
			t.FireAnim = 0
			t.CritBonus = 0
			t.Faction = ""

			// Apply snapshot attributes (skip random roll)
			t.BaseDamage = snap.BaseDamage
			t.PotentialDamage = snap.PotentialDamage
			t.BaseSpeed = snap.BaseSpeed
			t.PotentialSpeed = snap.PotentialSpeed
			t.BaseRange = snap.BaseRange
			t.PotentialRange = snap.PotentialRange
			t.DamageTier = snap.DamageTier
			t.SpeedTier = snap.SpeedTier
			t.RangeTier = snap.RangeTier

			t.BuildAnim = 0
			t.SellAnim = 0
			t.Selling = false
			t.Strength = strength.NewStrengthData()
			t.Buffs = nil
			t.PendingChoices = nil
			p.Count++
			return t
		}
	}
	return nil
}
```

Note: Import `"defense2/internal/config"` at the top of pool.go.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/core/tower/ -run TestPlaceFromSnapshot -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/tower/pool.go internal/core/tower/pool_test.go
git commit -m "feat: add PlaceFromSnapshot to tower pool"
```

---

### Task 4: Save scenario from debug panel

**Files:**
- Modify: `internal/scene/stage.go` (add saveScenario + debug button)

**Step 1: Add saveScenario method**

Add to `stage.go` (near `debugActions`):

```go
// saveScenario exports the current tower layout + scene config to a JSON file.
func (s *StageScene) saveScenario() {
	var towers []config.TowerSnapshot
	s.towers.Each(func(t *tower.Tower) {
		towers = append(towers, config.TowerSnapshot{
			Row:             t.Row,
			Col:             t.Col,
			Key:             t.Key,
			AbilitySlots:    t.AbilitySlots,
			DamageTier:      t.DamageTier,
			SpeedTier:       t.SpeedTier,
			RangeTier:       t.RangeTier,
			BaseDamage:      t.BaseDamage,
			PotentialDamage: t.PotentialDamage,
			BaseSpeed:       t.BaseSpeed,
			PotentialSpeed:  t.PotentialSpeed,
			BaseRange:       t.BaseRange,
			PotentialRange:  t.PotentialRange,
		})
	})

	sd := config.ScenarioData{
		ID:          fmt.Sprintf("scenario-%s", time.Now().Format("20060102-150405")),
		Name:        fmt.Sprintf("Snapshot %s", time.Now().Format("15:04:05")),
		Description: fmt.Sprintf("%d towers on %s", len(towers), s.initOpts.MapID),
		MapID:       s.initOpts.MapID,
		Gold:        s.initOpts.Gold,
		Lives:       s.initOpts.Lives,
		Waves:       s.initOpts.Waves,
		EnemyFilter: s.initOpts.EnemyFilter,
		ManualWave:  s.initOpts.ManualWave,
		Towers:      towers,
	}

	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		hud.ShowToast("Save failed: " + err.Error())
		return
	}

	path := filepath.Join("config", "scenarios", sd.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		hud.ShowToast("Save failed: " + err.Error())
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		hud.ShowToast("Save failed: " + err.Error())
		return
	}

	hud.ShowToast(fmt.Sprintf("Saved: %s", path))
	fmt.Printf("Scenario saved: %s\n", path)
}
```

**Step 2: Add debug button**

In `debugActions()`, before the final `return actions`, add a new section:

```go
	// ── 场景快照 ──
	actions = append(actions,
		hud.DebugAction{Label: "场景快照", IsSection: true},
		hud.DebugAction{Label: "Save Scenario", Action: func() { s.saveScenario() }},
	)
```

**Step 3: Verify build and manual test**

Run: `go build ./cmd/game/`
Expected: BUILD SUCCESS

**Step 4: Commit**

```bash
git add internal/scene/stage.go
git commit -m "feat: add Save Scenario button to debug panel"
```

---

### Task 5: Restore scenario on stage init

**Files:**
- Modify: `internal/scene/stage.go` (add restoreScenario, call it from NewStageSceneWithOpts)

**Step 1: Add restoreScenario method**

```go
// restoreScenario places towers from a saved scenario snapshot.
func (s *StageScene) restoreScenario(sd *config.ScenarioData) {
	defMap := make(map[string]tower.TowerDef)
	for _, d := range s.towerDefs {
		defMap[d.Key] = d
	}
	for _, snap := range sd.Towers {
		def, ok := defMap[snap.Key]
		if !ok {
			fmt.Printf("restoreScenario: unknown tower key %q, skip\n", snap.Key)
			continue
		}
		center := s.gameMap.CellCenter(snap.Row, snap.Col)
		t := s.towers.PlaceFromSnapshot(snap.Row, snap.Col, center.X, center.Y, def, snap)
		if t == nil {
			fmt.Printf("restoreScenario: pool full, cannot place %s at (%d,%d)\n", snap.Key, snap.Row, snap.Col)
			continue
		}
		// Restore abilities (order matters: attack mode first changes style/sprite)
		for _, abilityType := range snap.AbilitySlots {
			if abilityType != "" {
				t.AddAbility(abilityType)
			}
		}
		t.RecalcStats()
		s.gold -= def.Cost
	}
}
```

**Step 2: Call from NewStageSceneWithOpts**

In `NewStageSceneWithOpts`, after the existing `if opts.EnemyFilter == "all-static"` block (around line 340), add:

```go
	// Restore saved scenario towers
	if opts.ScenarioID != "" {
		if scenarios, err := config.LoadScenarios(); err == nil {
			for _, sd := range scenarios {
				if sd.ID == opts.ScenarioID {
					s.restoreScenario(sd)
					break
				}
			}
		}
	}
```

Note: This only triggers for custom scenarios (those from JSON files). Hardcoded test scenarios have IDs like "tower-core" which won't match any JSON file, so no conflict.

**Step 3: Verify build**

Run: `go build ./cmd/game/`
Expected: BUILD SUCCESS

**Step 4: Commit**

```bash
git add internal/scene/stage.go
git commit -m "feat: restore tower layout from scenario JSON on stage init"
```

---

### Task 6: Custom category in TestSelectScene

**Files:**
- Modify: `internal/scene/test_select.go`

**Step 1: Add "custom" category to testCategories**

```go
var testCategories = []testCategory{
	{"all", "全部"},
	{"tower", "炮塔测试"},
	{"enemy", "怪物测试"},
	{"combo", "综合测试"},
	{"ability", "能力测试"},
	{"dps", "DPS测试"},
	{"bench", "基准测试"},
	{"custom", "自定义"},   // <-- NEW
}
```

**Step 2: Add customScenarios field and load on init**

Add field to `TestSelectScene`:

```go
type TestSelectScene struct {
	switcher       Switcher
	selectedIdx    int
	hoverIdx       int
	activeTab      int
	hoverTab       int
	hoverStart     bool
	filtered       []int
	customLoaded   bool          // true after first load attempt
	customScenarios []testScenario // scenarios loaded from JSON
}
```

**Step 3: Load custom scenarios in NewTestSelectScene**

After creating the scene, before `s.updateFilter()`:

```go
func NewTestSelectScene(sw Switcher) *TestSelectScene {
	s := &TestSelectScene{
		switcher:    sw,
		selectedIdx: -1,
		hoverIdx:    -1,
		hoverTab:    -1,
	}
	s.loadCustomScenarios()
	s.updateFilter()
	return s
}

func (s *TestSelectScene) loadCustomScenarios() {
	scenarios, err := config.LoadScenarios()
	if err != nil || len(scenarios) == 0 {
		return
	}
	for _, sd := range scenarios {
		s.customScenarios = append(s.customScenarios, testScenario{
			ID:          sd.ID,
			Name:        sd.Name,
			Icon:        "multishot",
			Description: sd.Description,
			Category:    "custom",
			MapID:       sd.MapID,
			Gold:        sd.Gold,
			Lives:       sd.Lives,
			Waves:       sd.Waves,
			Color:       color.RGBA{R: 180, G: 140, B: 220, A: 255},
			EnemyFilter: sd.EnemyFilter,
			ManualWave:  sd.ManualWave,
		})
	}
	s.customLoaded = true
}
```

**Step 4: Update updateFilter to include custom scenarios**

The custom scenarios use indices offset from `len(testScenarios)`. Update `updateFilter`:

```go
func (s *TestSelectScene) updateFilter() {
	cat := testCategories[s.activeTab].ID
	s.filtered = nil
	for i, sc := range testScenarios {
		if cat == "all" || sc.Category == cat {
			s.filtered = append(s.filtered, i)
		}
	}
	// Append custom scenarios with offset indices
	base := len(testScenarios)
	for i, sc := range s.customScenarios {
		if cat == "all" || sc.Category == cat {
			s.filtered = append(s.filtered, base+i)
		}
	}
	s.selectedIdx = -1
}
```

**Step 5: Helper to get scenario by index (handles both builtin and custom)**

```go
func (s *TestSelectScene) scenarioAt(idx int) *testScenario {
	if idx < 0 {
		return nil
	}
	if idx < len(testScenarios) {
		return &testScenarios[idx]
	}
	ci := idx - len(testScenarios)
	if ci < len(s.customScenarios) {
		return &s.customScenarios[ci]
	}
	return nil
}
```

**Step 6: Update startScenario and Draw to use scenarioAt**

Replace `testScenarios[s.selectedIdx]` with `s.scenarioAt(s.selectedIdx)` in `startScenario()` and `Draw()`.

In `startScenario()`:
```go
func (s *TestSelectScene) startScenario() {
	sc := s.scenarioAt(s.selectedIdx)
	if sc == nil {
		return
	}
	// ... rest unchanged
}
```

In `Draw()`, the card loop:
```go
for i, scIdx := range s.filtered {
	sc := s.scenarioAt(scIdx)
	if sc == nil {
		continue
	}
	// ... rest unchanged, use *sc instead of testScenarios[scIdx]
}
```

And the selection check:
```go
selected := scIdx == s.selectedIdx
```

**Step 7: Verify build**

Run: `go build ./cmd/game/`
Expected: BUILD SUCCESS

**Step 8: Commit**

```bash
git add internal/scene/test_select.go
git commit -m "feat: add custom category to TestSelectScene for JSON scenarios"
```

---

### Task 7: Integration test — round-trip save/load

**Files:**
- Create: `tests/contracts/scenario_roundtrip_test.go`

**Step 1: Write round-trip test**

```go
package contracts

import (
	"encoding/json"
	"testing"

	"defense2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScenarioRoundTrip(t *testing.T) {
	original := config.ScenarioData{
		ID:          "roundtrip-test",
		Name:        "Round Trip",
		Description: "test",
		MapID:       "map_test",
		Gold:        9999,
		Lives:       999,
		Waves:       10,
		EnemyFilter: "mixed",
		ManualWave:  true,
		Towers: []config.TowerSnapshot{
			{
				Row: 3, Col: 5, Key: "prism",
				AbilitySlots:    [6]string{"scatter", "", "auraDamage", "", "", ""},
				DamageTier:      "A",
				SpeedTier:       "B",
				RangeTier:       "S",
				BaseDamage:      12.0,
				PotentialDamage: 8.0,
				BaseSpeed:       1.5,
				PotentialSpeed:  0.5,
				BaseRange:       120.0,
				PotentialRange:  30.0,
			},
			{
				Row: 5, Col: 7, Key: "sentinel",
				AbilitySlots:   [6]string{"enhance", "slow", "", "", "", ""},
				DamageTier:     "B",
				SpeedTier:      "A",
				RangeTier:      "B",
				BaseDamage:     8.0,
				PotentialDamage: 6.0,
				BaseSpeed:      2.0,
				PotentialSpeed: 0.8,
				BaseRange:      100.0,
				PotentialRange: 15.0,
			},
		},
	}

	// Marshal
	data, err := json.MarshalIndent(original, "", "  ")
	require.NoError(t, err)

	// Parse back
	parsed, err := config.ParseScenarioData(data)
	require.NoError(t, err)

	assert.Equal(t, original.ID, parsed.ID)
	assert.Equal(t, original.MapID, parsed.MapID)
	assert.Equal(t, original.Gold, parsed.Gold)
	assert.Equal(t, original.ManualWave, parsed.ManualWave)
	require.Len(t, parsed.Towers, 2)

	// Verify tower 1
	assert.Equal(t, original.Towers[0].Key, parsed.Towers[0].Key)
	assert.Equal(t, original.Towers[0].AbilitySlots, parsed.Towers[0].AbilitySlots)
	assert.InDelta(t, original.Towers[0].BaseDamage, parsed.Towers[0].BaseDamage, 0.001)

	// Verify tower 2
	assert.Equal(t, original.Towers[1].Key, parsed.Towers[1].Key)
	assert.Equal(t, original.Towers[1].AbilitySlots, parsed.Towers[1].AbilitySlots)
}
```

**Step 2: Run test**

Run: `go test ./tests/contracts/ -run TestScenarioRoundTrip -v`
Expected: PASS

**Step 3: Commit**

```bash
git add tests/contracts/scenario_roundtrip_test.go
git commit -m "test: add scenario round-trip contract test"
```

---

### Task 8: Final verification

**Step 1: Full test suite**

Run: `make test`
Expected: ALL PASS

**Step 2: Build verification**

Run: `go build ./cmd/game/`
Expected: BUILD SUCCESS

**Step 3: Manual smoke test**

1. `make run`
2. Enter test mode > select any scenario
3. Place a few towers, select abilities
4. Open debug panel (F key) > click "Save Scenario"
5. Verify console prints saved path
6. Verify `config/scenarios/scenario-*.json` exists with correct content
7. Restart game (need recompile with `make run` to embed new JSON)
8. Enter test mode > switch to "自定义" tab
9. Verify saved scenario appears
10. Select it > Start > verify towers are restored with correct positions and abilities

**Step 4: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "feat: complete test scenario snapshot save/load"
```
