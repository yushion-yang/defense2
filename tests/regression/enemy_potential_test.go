package regression_test

import (
	"math"
	"testing"

	"defense2/internal/core/enemy"
	"defense2/tests/regression/sim"
)

// TestRegression_EnemyAbilityPotential verifies that ability potential
// scales enemy attributes by wave: effective = base + potential * wave.
func TestRegression_EnemyAbilityPotential(t *testing.T) {
	tests := []struct {
		name      string
		abilityID string
		potential float64
		wave      int
		baseHP    float64
		setupCfg  func(cfg *enemy.SpawnConfig)
		check     func(t *testing.T, e *enemy.Enemy)
	}{
		{
			name:      "armorPlating: base=10, potential=0.5, wave=20 → armor=20",
			abilityID: "armorPlating",
			potential: 0.5,
			wave:      20,
			baseHP:    100,
			setupCfg: func(cfg *enemy.SpawnConfig) {
				cfg.ArmorFlat = 10 // base value from ability
			},
			check: func(t *testing.T, e *enemy.Enemy) {
				want := 10.0 + 0.5*20 // 20
				if math.Abs(e.ArmorFlat-want) > 1e-9 {
					t.Errorf("ArmorFlat = %v, want %v", e.ArmorFlat, want)
				}
			},
		},
		{
			name:      "damageCap: base=60, potential=2, wave=10 → cap=80",
			abilityID: "damageCap",
			potential: 2.0,
			wave:      10,
			baseHP:    100,
			setupCfg: func(cfg *enemy.SpawnConfig) {
				cfg.DamageCap = 60
			},
			check: func(t *testing.T, e *enemy.Enemy) {
				want := 60.0 + 2.0*10
				if math.Abs(e.DamageCap-want) > 1e-9 {
					t.Errorf("DamageCap = %v, want %v", e.DamageCap, want)
				}
			},
		},
		{
			name:      "evasion: base=0.3, potential=0.01, wave=30 → 0.6 (clamped to 1)",
			abilityID: "evasion",
			potential: 0.01,
			wave:      30,
			baseHP:    100,
			setupCfg: func(cfg *enemy.SpawnConfig) {
				cfg.EvasionChance = 0.3
			},
			check: func(t *testing.T, e *enemy.Enemy) {
				want := 0.3 + 0.01*30 // 0.6
				if math.Abs(e.EvasionChance-want) > 1e-9 {
					t.Errorf("EvasionChance = %v, want %v", e.EvasionChance, want)
				}
			},
		},
		{
			name:      "evasion clamped at 1.0",
			abilityID: "evasion",
			potential: 0.1,
			wave:      100,
			baseHP:    100,
			setupCfg: func(cfg *enemy.SpawnConfig) {
				cfg.EvasionChance = 0.3
			},
			check: func(t *testing.T, e *enemy.Enemy) {
				if e.EvasionChance > 1.0 {
					t.Errorf("EvasionChance = %v, should be clamped to 1.0", e.EvasionChance)
				}
			},
		},
		{
			name:      "wave 0: no scaling",
			abilityID: "armorPlating",
			potential: 0.5,
			wave:      0,
			baseHP:    100,
			setupCfg: func(cfg *enemy.SpawnConfig) {
				cfg.ArmorFlat = 10
			},
			check: func(t *testing.T, e *enemy.Enemy) {
				if math.Abs(e.ArmorFlat-10) > 1e-9 {
					t.Errorf("ArmorFlat = %v, want 10 (wave=0, no scaling)", e.ArmorFlat)
				}
			},
		},
		{
			name:      "zero potential: no scaling regardless of wave",
			abilityID: "armorPlating",
			potential: 0,
			wave:      20,
			baseHP:    100,
			setupCfg: func(cfg *enemy.SpawnConfig) {
				cfg.ArmorFlat = 10
			},
			check: func(t *testing.T, e *enemy.Enemy) {
				if math.Abs(e.ArmorFlat-10) > 1e-9 {
					t.Errorf("ArmorFlat = %v, want 10 (potential=0)", e.ArmorFlat)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &enemy.SpawnConfig{
				HpScale:    1,
				SpeedScale: 1,
				Radius:     8,
			}
			tt.setupCfg(cfg)
			if tt.potential != 0 {
				cfg.AbilityPotentials = []enemy.AbilityPotentialEntry{
					{Type: tt.abilityID, Potential: tt.potential},
				}
			}

			s := sim.New().
				WithStraightPath(500).
				WithEnemyCfg("test", tt.baseHP, 60, cfg).
				Build()

			e := s.SpawnedEnemies[0]
			// Apply ability potentials (simulating what spawner does at spawn time)
			enemy.ApplyAbilityPotentials(e, cfg, tt.wave)
			tt.check(t, e)
		})
	}
}
