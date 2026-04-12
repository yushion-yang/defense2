package config

import (
	"math"
	"testing"
)

func TestResolveEnemyAbilityAtWave(t *testing.T) {
	// 设置测试用 ability table
	enemyAbilityTable = map[string]*EnemyAbilityDef{
		"armorPlating": {Type: "armorPlating", Base: 10, Potential: 0.5, Param: 0},
		"evasion":      {Type: "evasion", Base: 0.3, Potential: 0.01, Param: 0},
		"damageCap":    {Type: "damageCap", Base: 60, Potential: 0, Param: 0},
	}
	defer func() { enemyAbilityTable = nil }()

	tests := []struct {
		name     string
		ref      EnemyAbilityRef
		wave     int
		wantBase float64
	}{
		{
			name:     "wave 0: base only",
			ref:      EnemyAbilityRef{Type: "armorPlating"},
			wave:     0,
			wantBase: 10,
		},
		{
			name:     "wave 10: base + potential * 10",
			ref:      EnemyAbilityRef{Type: "armorPlating"},
			wave:     10,
			wantBase: 10 + 0.5*10, // 15
		},
		{
			name:     "wave 20: base + potential * 20",
			ref:      EnemyAbilityRef{Type: "armorPlating"},
			wave:     20,
			wantBase: 10 + 0.5*20, // 20
		},
		{
			name:     "ref overrides potential",
			ref:      EnemyAbilityRef{Type: "armorPlating", Potential: 1.0},
			wave:     10,
			wantBase: 10 + 1.0*10, // 20
		},
		{
			name:     "ref overrides base and potential",
			ref:      EnemyAbilityRef{Type: "armorPlating", Base: 20, Potential: 2.0},
			wave:     5,
			wantBase: 20 + 2.0*5, // 30
		},
		{
			name:     "zero potential: no scaling",
			ref:      EnemyAbilityRef{Type: "damageCap"},
			wave:     30,
			wantBase: 60, // potential=0, no scaling
		},
		{
			name:     "evasion wave 10",
			ref:      EnemyAbilityRef{Type: "evasion"},
			wave:     10,
			wantBase: 0.3 + 0.01*10, // 0.4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := ResolveEnemyAbilityAtWave(tt.ref, tt.wave)
			if def == nil {
				t.Fatal("ResolveEnemyAbilityAtWave returned nil")
			}
			if math.Abs(def.Base-tt.wantBase) > 1e-9 {
				t.Errorf("Base = %v, want %v", def.Base, tt.wantBase)
			}
		})
	}
}

func TestResolveEffectivePotential(t *testing.T) {
	enemyAbilityTable = map[string]*EnemyAbilityDef{
		"armorPlating": {Type: "armorPlating", Base: 10, Potential: 0.5},
	}
	defer func() { enemyAbilityTable = nil }()

	tests := []struct {
		name string
		ref  EnemyAbilityRef
		want float64
	}{
		{"default potential", EnemyAbilityRef{Type: "armorPlating"}, 0.5},
		{"ref overrides", EnemyAbilityRef{Type: "armorPlating", Potential: 1.0}, 1.0},
		{"unknown type", EnemyAbilityRef{Type: "unknown"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveEffectivePotential(tt.ref)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("ResolveEffectivePotential = %v, want %v", got, tt.want)
			}
		})
	}
}
