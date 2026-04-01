package achievement

import (
	"defense2/internal/core/persistence"
	"testing"
)

func newTestTracker() *Tracker {
	return NewTracker(nil)
}

func TestUnlock_NewlyUnlocked(t *testing.T) {
	tr := newTestTracker()
	if !tr.Unlock("first_win") {
		t.Error("expected first unlock to return true")
	}
	if !tr.IsUnlocked("first_win") {
		t.Error("expected first_win to be unlocked")
	}
}

func TestUnlock_AlreadyUnlocked(t *testing.T) {
	tr := newTestTracker()
	tr.Unlock("first_win")
	if tr.Unlock("first_win") {
		t.Error("expected second unlock of same ID to return false")
	}
}

func TestUnlockedCount(t *testing.T) {
	tr := newTestTracker()
	if tr.UnlockedCount() != 0 {
		t.Errorf("expected 0, got %d", tr.UnlockedCount())
	}
	tr.Unlock("first_win")
	tr.Unlock("first_boss")
	if tr.UnlockedCount() != 2 {
		t.Errorf("expected 2, got %d", tr.UnlockedCount())
	}
}

func TestResetSession(t *testing.T) {
	tr := newTestTracker()
	tr.SessionKills = 50
	tr.SessionMaxStreak = 10
	tr.SessionItemsUsed = 5
	tr.SessionTowerTypes["laser"] = true
	tr.SessionMaxGold = 999

	tr.ResetSession()

	if tr.SessionKills != 0 {
		t.Errorf("expected SessionKills=0, got %d", tr.SessionKills)
	}
	if tr.SessionMaxStreak != 0 {
		t.Errorf("expected SessionMaxStreak=0, got %d", tr.SessionMaxStreak)
	}
	if tr.SessionItemsUsed != 0 {
		t.Errorf("expected SessionItemsUsed=0, got %d", tr.SessionItemsUsed)
	}
	if len(tr.SessionTowerTypes) != 0 {
		t.Errorf("expected empty SessionTowerTypes, got %d entries", len(tr.SessionTowerTypes))
	}
	if tr.SessionMaxGold != 0 {
		t.Errorf("expected SessionMaxGold=0, got %d", tr.SessionMaxGold)
	}
}

func TestResetSession_PreservesUnlocked(t *testing.T) {
	tr := newTestTracker()
	tr.Unlock("first_win")
	tr.ResetSession()
	if !tr.IsUnlocked("first_win") {
		t.Error("ResetSession should not clear unlocked achievements")
	}
}

func TestIncrTowersBuilt(t *testing.T) {
	tr := newTestTracker()
	tr.IncrTowersBuilt()
	tr.IncrTowersBuilt()
	tr.IncrTowersBuilt()
	if tr.TotalTowersBuilt != 3 {
		t.Errorf("expected TotalTowersBuilt=3, got %d", tr.TotalTowersBuilt)
	}
}

func TestNameByID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"first_win", "初次胜利"},
		{"centurion", "百杀"},
		{"nonexistent", ""},
	}
	for _, tc := range tests {
		got := NameByID(tc.id)
		if got != tc.want {
			t.Errorf("NameByID(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestAllAchievementsHaveUniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, a := range All {
		if seen[a.ID] {
			t.Errorf("duplicate achievement ID: %s", a.ID)
		}
		seen[a.ID] = true
	}
}

func TestPersistence(t *testing.T) {
	store, err := persistence.NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// Create tracker, unlock something, save
	tr1 := NewTracker(store)
	tr1.Unlock("first_win")
	tr1.IncrTowersBuilt()

	// Create new tracker with same storage — should load
	tr2 := NewTracker(store)
	if !tr2.IsUnlocked("first_win") {
		t.Error("expected first_win to persist across tracker instances")
	}
	if tr2.TotalTowersBuilt != 1 {
		t.Errorf("expected TotalTowersBuilt=1, got %d", tr2.TotalTowersBuilt)
	}
}
