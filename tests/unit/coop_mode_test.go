//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/gamemode"
)

func TestCoopModeRegistered(t *testing.T) {
	m := gamemode.GetOrDefault("coop")
	if m.ID() != "coop" {
		t.Errorf("GetOrDefault('coop') returned %q, want 'coop'", m.ID())
	}
}

func TestCoopModeNoAutoStart(t *testing.T) {
	m := gamemode.GetOrDefault("coop")
	if m.ShouldAutoStart() {
		t.Error("coop should not auto start waves")
	}
}

func TestCoopModeWardenDisabled(t *testing.T) {
	m := gamemode.GetOrDefault("coop")
	if m.Ruleset().WardenEnabled() {
		t.Error("coop should not enable wardens (players share gold, no individual warden)")
	}
}

func TestCoopModeGoldShareEnabled(t *testing.T) {
	m := gamemode.GetOrDefault("coop")
	if !m.Ruleset().GoldShare() {
		t.Error("coop should enable gold sharing (all players receive equal gold)")
	}
}

func TestCoopModeIntermission(t *testing.T) {
	m := gamemode.GetOrDefault("coop")
	if m.IntermissionSecs() != 15.0 {
		t.Errorf("IntermissionSecs = %f, want 15.0", m.IntermissionSecs())
	}
}
