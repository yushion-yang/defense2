// buff_templates.go — Legacy helpers for flag application and type mapping.
// The old BuffTemplate system has been removed; wave buffs are now applied
// via direct field setting in spawner.go, and archetype abilities are
// configured in config/enemies/abilities.json.
package enemy

// ApplyFlags applies flag effects to an enemy.
// Supported flags:
//   - "boss": HP*30
func ApplyFlags(e *Enemy, flags []string) {
	for _, flag := range flags {
		switch flag {
		case "boss":
			e.MaxHP *= 30
			e.HP *= 30
			e.Boss = true
		}
	}
}

// MapLegacyType maps old enemy type names to the new archetype system.
// Returns (archetype, flags, buffIDs) for backward compatibility.
func MapLegacyType(typeName string) (archetype string, flags []string, buffIDs []string) {
	switch typeName {
	case "normal":
		return "normal", nil, nil
	case "fast", "runner":
		return "runner", nil, nil
	case "tank", "armored":
		return "tank", nil, nil
	case "flying":
		return "flying", nil, nil
	case "healer":
		return "healer", nil, []string{"healAura"}
	case "berserker":
		return "berserker", nil, []string{"berserk"}
	case "regenerator":
		return "regenerator", nil, []string{"regen"}
	case "splitter":
		return "splitter", nil, []string{"deathSplit"}
	case "summoner":
		return "summoner", nil, []string{"spawnMinions"}
	case "reflector":
		return "reflector", nil, []string{"reflect"}
	case "boss":
		return "normal", []string{"boss"}, nil
	default:
		return "normal", nil, nil
	}
}
