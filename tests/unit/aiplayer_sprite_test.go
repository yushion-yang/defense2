//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestSpriteIdle(t *testing.T) {
	sp := aiplayer.NewSprite(600, 300)
	if sp.State() != aiplayer.SpriteIdle {
		t.Errorf("initial state = %d, want SpriteIdle", sp.State())
	}
	if sp.X() != 600 || sp.Y() != 300 {
		t.Errorf("position = (%.0f,%.0f), want (600,300)", sp.X(), sp.Y())
	}
}

func TestSpriteMoveTo(t *testing.T) {
	sp := aiplayer.NewSprite(0, 0)
	sp.MoveTo(120, 0) // 120px, speed=120px/s → ~1s

	// Tick 30 帧 (0.5s) — 在移动中
	for i := 0; i < 30; i++ {
		sp.Tick(1.0 / 60.0)
	}
	if sp.State() != aiplayer.SpriteMoving {
		t.Errorf("after 0.5s state = %d, want SpriteMoving", sp.State())
	}
	if sp.X() < 50 || sp.X() > 70 {
		t.Errorf("after 0.5s X = %.1f, want ~60", sp.X())
	}

	// 再 Tick 60 帧 (1s) — 应到达
	for i := 0; i < 60; i++ {
		sp.Tick(1.0 / 60.0)
	}
	if sp.State() != aiplayer.SpriteIdle {
		t.Errorf("after arrival state = %d, want SpriteIdle", sp.State())
	}
	if sp.X() < 118 || sp.X() > 122 {
		t.Errorf("after arrival X = %.1f, want ~120", sp.X())
	}
}

func TestSpriteThinking(t *testing.T) {
	sp := aiplayer.NewSprite(100, 200)
	sp.SetThinking(0.5)
	if sp.State() != aiplayer.SpriteThinking {
		t.Errorf("state = %d, want SpriteThinking", sp.State())
	}
	sp.Tick(0.3)
	if sp.State() != aiplayer.SpriteThinking {
		t.Errorf("after 0.3s state = %d, want SpriteThinking", sp.State())
	}
	sp.Tick(0.3)
	if sp.State() != aiplayer.SpriteIdle {
		t.Errorf("after 0.6s state = %d, want SpriteIdle", sp.State())
	}
}
