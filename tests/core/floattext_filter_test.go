package core_test

import (
	"testing"

	"defense2/internal/core/game"
	"defense2/internal/render"
)

func TestDamageTextHighQualityShowsAll(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()
	game.CurrentQuality = game.QualityHigh

	// Small damage should still create a float text
	render.SpawnDamageText(100, 100, 1.0, false, false)
	// We can't easily check if it was created without reading pool internals,
	// but at minimum this should not panic
}

func TestDamageTextMediumFilterSmall(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()
	game.CurrentQuality = game.QualityMedium

	// This should be filtered (< 3)
	render.SpawnDamageText(100, 100, 2.0, false, false)
	// Not filtered (>= 3)
	render.SpawnDamageText(100, 100, 5.0, false, false)
	// Crit always shown
	render.SpawnDamageText(100, 100, 1.0, true, false)
	// Boss always shown
	render.SpawnDamageText(100, 100, 1.0, false, true)
}

func TestDamageTextLowFilterMost(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()
	game.CurrentQuality = game.QualityLow

	// Small non-crit non-boss filtered
	render.SpawnDamageText(100, 100, 10.0, false, false)
	// Large damage shown
	render.SpawnDamageText(100, 100, 25.0, false, false)
	// Crit always shown
	render.SpawnDamageText(100, 100, 1.0, true, false)
	// Boss always shown
	render.SpawnDamageText(100, 100, 1.0, false, true)
}
