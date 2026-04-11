package mascot

import (
	"encoding/json"
	"fmt"
)

// ParseDialogs decodes a JSON array of Dialog objects.
func ParseDialogs(data []byte) ([]Dialog, error) {
	var dialogs []Dialog
	if err := json.Unmarshal(data, &dialogs); err != nil {
		return nil, fmt.Errorf("parse mascot dialogs: %w", err)
	}
	return dialogs, nil
}

// AssetReader reads embedded files.
type AssetReader interface {
	ReadFile(name string) ([]byte, error)
}

// LoadAllDialogs loads all dialog JSON files from config/mascot/.
func LoadAllDialogs(fs AssetReader) ([]Dialog, error) {
	files := []string{
		"config/mascot/dialogs-title.json",
		"config/mascot/dialogs-stage.json",
		"config/mascot/dialogs-select.json",
		"config/mascot/dialogs-common.json",
	}
	var all []Dialog
	for _, f := range files {
		data, err := fs.ReadFile(f)
		if err != nil {
			continue // file not yet created is OK
		}
		dialogs, err := ParseDialogs(data)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", f, err)
		}
		all = append(all, dialogs...)
	}
	return all, nil
}
