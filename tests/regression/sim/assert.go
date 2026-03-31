// assert.go — Domain-specific test assertion helpers for Sim.
package sim

import (
	"fmt"
	"testing"
)

// cmp evaluates "actual op expected" for float64.
func cmp(actual float64, op string, expected float64) bool {
	switch op {
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	case ">":
		return actual > expected
	case ">=":
		return actual >= expected
	case "<":
		return actual < expected
	case "<=":
		return actual <= expected
	default:
		panic("unknown operator: " + op)
	}
}

func cmpInt(actual int, op string, expected int) bool {
	return cmp(float64(actual), op, float64(expected))
}

func failMsg(t *testing.T, label string, actual interface{}, op string, expected interface{}) {
	t.Helper()
	t.Fatalf("%s: got %v, expected %s %v", label, actual, op, expected)
}

// AssertEnemyHP checks enemy HP at index.
func (s *Sim) AssertEnemyHP(t *testing.T, idx int, op string, expected float64) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !cmp(e.HP, op, expected) {
		failMsg(t, fmt.Sprintf("enemy[%d].HP", idx), e.HP, op, expected)
	}
}

// AssertEnemySpeed checks enemy Speed at index.
func (s *Sim) AssertEnemySpeed(t *testing.T, idx int, op string, expected float64) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !cmp(e.Speed, op, expected) {
		failMsg(t, fmt.Sprintf("enemy[%d].Speed", idx), e.Speed, op, expected)
	}
}

// AssertEnemyAlive checks enemy is still active and not dying.
func (s *Sim) AssertEnemyAlive(t *testing.T, idx int) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !e.Active || e.IsDying() {
		t.Fatalf("enemy[%d]: expected alive, Active=%v Dying=%v", idx, e.Active, e.IsDying())
	}
}

// AssertEnemyDead checks enemy is dead (inactive or dying).
func (s *Sim) AssertEnemyDead(t *testing.T, idx int) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if e.Active && !e.IsDying() {
		t.Fatalf("enemy[%d]: expected dead, Active=%v HP=%.1f", idx, e.Active, e.HP)
	}
}

// AssertEnemyDying checks enemy is in dying animation.
func (s *Sim) AssertEnemyDying(t *testing.T, idx int) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !e.IsDying() {
		t.Fatalf("enemy[%d]: expected dying, DyingTimer=%.3f", idx, e.DyingTimer)
	}
}

// AssertEnemyReachedEnd checks enemy reached path end.
func (s *Sim) AssertEnemyReachedEnd(t *testing.T, idx int) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !e.ReachedEnd {
		t.Fatalf("enemy[%d]: expected ReachedEnd, X=%.1f Y=%.1f", idx, e.X, e.Y)
	}
}

// AssertEnemyCount checks active enemy count.
func (s *Sim) AssertEnemyCount(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Enemies.Count, op, expected) {
		failMsg(t, "enemyCount", s.Enemies.Count, op, expected)
	}
}

// AssertProjectileCount checks active projectile count.
func (s *Sim) AssertProjectileCount(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Projectiles.Count, op, expected) {
		failMsg(t, "projectileCount", s.Projectiles.Count, op, expected)
	}
}

// AssertGold checks current gold.
func (s *Sim) AssertGold(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Gold, op, expected) {
		failMsg(t, "gold", s.Gold, op, expected)
	}
}

// AssertLives checks current lives.
func (s *Sim) AssertLives(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Lives, op, expected) {
		failMsg(t, "lives", s.Lives, op, expected)
	}
}

// AssertKills checks total kills.
func (s *Sim) AssertKills(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Kills, op, expected) {
		failMsg(t, "kills", s.Kills, op, expected)
	}
}

// AssertLeaked checks total leaked enemies.
func (s *Sim) AssertLeaked(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Leaked, op, expected) {
		failMsg(t, "leaked", s.Leaked, op, expected)
	}
}
