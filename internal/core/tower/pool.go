package tower

import "defense2/internal/core/game"

// Pool is a fixed-size tower pool.
type Pool struct {
	towers []Tower
	Count  int
}

// NewPool creates a tower pool with the given capacity.
func NewPool(cap int) *Pool {
	return &Pool{
		towers: make([]Tower, cap),
	}
}

// DefaultPool creates a pool with default capacity.
func DefaultPool() *Pool {
	return NewPool(game.MaxTowers)
}

// Place activates a tower slot at the given grid position.
func (p *Pool) Place(row, col int, cx, cy float64, def TowerDef) *Tower {
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
			t.Faction = def.Faction
			t.Key = def.Key
			t.Label = def.Label
			t.Abilities = def.Abilities
			t.Color = def.Color
			t.Active = true
			p.Count++
			return t
		}
	}
	return nil
}

// Remove deactivates a tower (sell).
func (p *Pool) Remove(t *Tower) {
	if t.Active {
		t.Active = false
		p.Count--
	}
}

// Each iterates over all active towers.
func (p *Pool) Each(fn func(t *Tower)) {
	for i := range p.towers {
		if p.towers[i].Active {
			fn(&p.towers[i])
		}
	}
}

// At returns the tower at grid position (row, col), or nil.
func (p *Pool) At(row, col int) *Tower {
	for i := range p.towers {
		if p.towers[i].Active && p.towers[i].Row == row && p.towers[i].Col == col {
			return &p.towers[i]
		}
	}
	return nil
}

// TowerDef defines the stats for placing a tower.
type TowerDef struct {
	Key         string
	Label       string
	Faction     string
	Range       float64
	Damage      float64
	AttackSpeed float64 // attacks per second
	Cost        int
	Abilities   []string
	Color       [3]uint8
}

// BaseTowerDefs returns the 4 base tower types for P3.
func BaseTowerDefs() []TowerDef {
	return []TowerDef{
		{
			Key: "basic", Label: "Arrow", Faction: "base",
			Range: 150, Damage: 10, AttackSpeed: 1.5, Cost: 50,
			Color: [3]uint8{80, 140, 220},
		},
		{
			Key: "splash", Label: "Cannon", Faction: "base",
			Range: 120, Damage: 20, AttackSpeed: 0.8, Cost: 80,
			Abilities: []string{"splash"},
			Color:     [3]uint8{200, 120, 60},
		},
		{
			Key: "slow", Label: "Frost", Faction: "base",
			Range: 130, Damage: 5, AttackSpeed: 1.2, Cost: 60,
			Abilities: []string{"onHitSlow"},
			Color:     [3]uint8{100, 180, 220},
		},
		{
			Key: "sniper", Label: "Sniper", Faction: "base",
			Range: 220, Damage: 35, AttackSpeed: 0.5, Cost: 100,
			Abilities: []string{"crit"},
			Color:     [3]uint8{180, 60, 180},
		},
	}
}
