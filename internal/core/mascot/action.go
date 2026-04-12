package mascot

// ActionType identifies a mascot assistance ability.
type ActionType string

const (
	ActionKillWeakEnemy ActionType = "kill_weak_enemy"
)

// MascotAction is a gameplay action requested by the mascot.
type MascotAction struct {
	Type ActionType
}
