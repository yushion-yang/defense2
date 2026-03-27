package core_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
)

func TestPoolSpawnAndKill(t *testing.T) {
	p := enemy.NewPool(4)
	e1 := p.Spawn(100, 100, 10, 60, 8, 1)
	if e1 == nil {
		t.Fatal("spawn should succeed")
	}
	if p.Count != 1 {
		t.Fatalf("count should be 1, got %d", p.Count)
	}

	e2 := p.Spawn(200, 200, 20, 60, 8, 1)
	if e2 == nil || p.Count != 2 {
		t.Fatal("second spawn failed")
	}

	p.Kill(e1)
	if p.Count != 1 {
		t.Fatalf("count after kill should be 1, got %d", p.Count)
	}

	// Killed slot should be reusable
	e3 := p.Spawn(300, 300, 30, 60, 8, 1)
	if e3 == nil || p.Count != 2 {
		t.Fatal("respawn into killed slot failed")
	}
}

func TestPoolOverflow(t *testing.T) {
	p := enemy.NewPool(2)
	p.Spawn(0, 0, 10, 60, 8, 1)
	p.Spawn(0, 0, 10, 60, 8, 1)
	e := p.Spawn(0, 0, 10, 60, 8, 1)
	if e != nil {
		t.Fatal("overflow spawn should return nil")
	}
}

func TestPoolEach(t *testing.T) {
	p := enemy.NewPool(4)
	p.Spawn(0, 0, 10, 60, 8, 1)
	p.Spawn(0, 0, 10, 60, 8, 1)
	e3 := p.Spawn(0, 0, 10, 60, 8, 1)
	p.Kill(e3)

	count := 0
	p.Each(func(_ *enemy.Enemy) { count++ })
	if count != 2 {
		t.Fatalf("each should visit 2 active, got %d", count)
	}
}

func TestMoveAlongPath(t *testing.T) {
	waypoints := []gamemap.Point{
		{X: 0, Y: 0},   // spawn point
		{X: 100, Y: 0},  // first target
		{X: 100, Y: 100}, // second target
	}
	e := &enemy.Enemy{
		X: 0, Y: 0, Speed: 200, Active: true, PathIndex: 1, // start targeting waypoint[1]
	}

	// Move 0.25s at 200px/s → 50px toward (100, 0)
	reached := enemy.MoveAlongPath(e, waypoints, 0.25)
	if reached {
		t.Fatal("should not reach end yet")
	}
	if e.X != 50 || e.Y != 0 {
		t.Fatalf("expected (50, 0), got (%.1f, %.1f)", e.X, e.Y)
	}

	// Move 0.25s → snap to (100, 0), pathIndex becomes 2
	enemy.MoveAlongPath(e, waypoints, 0.25)
	if e.X != 100 || e.Y != 0 {
		t.Fatalf("expected (100, 0), got (%.1f, %.1f)", e.X, e.Y)
	}
	if e.PathIndex != 2 {
		t.Fatalf("expected pathIndex 2, got %d", e.PathIndex)
	}

	// Move 0.5s → snap to (100, 100) and reach end
	reached = enemy.MoveAlongPath(e, waypoints, 0.5)
	if !reached {
		t.Fatal("should reach end")
	}
	if !e.ReachedEnd {
		t.Fatal("ReachedEnd should be true")
	}
}

func TestMoveStunned(t *testing.T) {
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}}
	e := &enemy.Enemy{X: 0, Y: 0, Speed: 200, Active: true, PathIndex: 1, StunTimer: 1.0}

	enemy.MoveAlongPath(e, waypoints, 0.5)
	if e.X != 0 {
		t.Fatalf("stunned enemy should not move, X=%.1f", e.X)
	}
	if e.StunTimer != 0.5 {
		t.Fatalf("stun timer should decrease to 0.5, got %.1f", e.StunTimer)
	}
}
