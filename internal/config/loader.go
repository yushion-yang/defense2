package config

import (
	"embed"
	"encoding/json"
	"fmt"
)

// dataFS holds the embedded config filesystem. Must be set via SetDataFS before loading.
var dataFS *embed.FS

// SetDataFS sets the embedded filesystem for config loading.
func SetDataFS(fs *embed.FS) {
	dataFS = fs
}

// LoadMap loads a map config by its ID (e.g. "map_01").
func LoadMap(id string) (*MapConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load map %s: dataFS not initialized", id)
	}
	path := fmt.Sprintf("config/levels/%s.json", id)
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load map %s: %w", id, err)
	}
	var m MapConfig
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse map %s: %w", id, err)
	}
	return &m, nil
}
