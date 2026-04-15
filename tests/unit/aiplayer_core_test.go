//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

// mockOps 实现 aiplayer.StageOps 接口。
type mockOps struct {
	builtCount    int
	upgradedCount int
}

func (m *mockOps) BuildTowerForAI(key string, row, col, ownerID int) bool {
	m.builtCount++
	return true
}

func (m *mockOps) UpgradeTowerForAI(row, col int) bool {
	m.upgradedCount++
	return true
}

func (m *mockOps) SellTowerForAI(row, col int) bool { return true }
func (m *mockOps) TowerCost(key string) int          { return 50 }
func (m *mockOps) StrengthBuyCost() int              { return 10 }
func (m *mockOps) StartWave() bool                   { return true }
func (m *mockOps) SelectWarden(key string) bool      { return true }

func TestAIPlayerGold(t *testing.T) {
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    120,
		Ops:          &mockOps{},
	})
	if ap.Gold() != 120 {
		t.Errorf("gold = %d, want 120", ap.Gold())
	}
	ap.AddGold(30)
	if ap.Gold() != 150 {
		t.Errorf("after AddGold(30) gold = %d, want 150", ap.Gold())
	}
}

func TestAIPlayerSpriteExists(t *testing.T) {
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    120,
		Ops:          &mockOps{},
		CellSize:     60,
		SpawnX:       900,
		SpawnY:       390,
	})
	if ap.SpriteX() < 700 {
		t.Errorf("sprite X = %.0f, should be in AI zone (>= 700)", ap.SpriteX())
	}
}

func TestAIPlayerTickDecides(t *testing.T) {
	ops := &mockOps{}
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    200,
		Ops:          ops,
		CellSize:     60,
	})

	snap := aiplayer.AISnapshot{
		Wave:     1,
		MaxWaves: 12,
		TowerDefs: []aiplayer.AITowerDef{
			{Key: "basic", Cost: 50, Damage: 10, Range: 100, Index: 0},
		},
		BuildCells: []aiplayer.AICell{
			{Row: 5, Col: 15, X: 930, Y: 330},
			{Row: 7, Col: 18, X: 1110, Y: 450},
		},
		MapCenterX:  720,
		MapCenterY:  390,
		WardenReady: true,
		WaveActive:  true, // 波进行中，不触发开波决策
	}

	for i := 0; i < 600; i++ {
		ap.Tick(1.0/60.0, snap)
	}

	if ops.builtCount == 0 {
		t.Error("AI never built a tower after 10s of ticking")
	}
}

func TestAIPlayerFiltersZone(t *testing.T) {
	ops := &mockOps{}
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    200,
		Ops:          ops,
		CellSize:     60,
	})

	snap := aiplayer.AISnapshot{
		Wave:     1,
		MaxWaves: 12,
		TowerDefs: []aiplayer.AITowerDef{
			{Key: "basic", Cost: 50, Damage: 10, Range: 100},
		},
		BuildCells: []aiplayer.AICell{
			{Row: 5, Col: 3, X: 210, Y: 330},
			{Row: 7, Col: 8, X: 510, Y: 450},
		},
		WardenReady: true,
		WaveActive:  true,
	}

	for i := 0; i < 300; i++ {
		ap.Tick(1.0/60.0, snap)
	}

	if ops.builtCount > 0 {
		t.Errorf("AI built %d towers in human zone, should be 0", ops.builtCount)
	}
}
