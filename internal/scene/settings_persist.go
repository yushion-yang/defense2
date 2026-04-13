// settings_persist.go — 设置持久化（音量/画质）。
// 读写 os.UserConfigDir()/defense2/settings.json。
package scene

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// SettingsData 可序列化的用户设置。
type SettingsData struct {
	SFXEnabled bool    `json:"sfxEnabled"`
	SFXVolume  float64 `json:"sfxVolume"`
	BGMVolume  float64 `json:"bgmVolume"`
	Quality    int     `json:"quality"`          // 0=High, 1=Medium, 2=Low
	Locale     string  `json:"locale,omitempty"` // "zh" (default), "en"
}

// DefaultSettings 返回默认设置。
func DefaultSettings() SettingsData {
	return SettingsData{
		SFXEnabled: true,
		SFXVolume:  0.8,
		BGMVolume:  0.5,
		Quality:    0, // High
	}
}

// settingsPath 返回设置文件的完整路径。
func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "defense2", "settings.json")
}

// LoadSettings 从磁盘加载设置，文件不存在或解析失败时返回默认值。
func LoadSettings() SettingsData {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return DefaultSettings()
	}
	var s SettingsData
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultSettings()
	}
	// 范围校验
	s.SFXVolume = clampF(s.SFXVolume, 0, 1)
	s.BGMVolume = clampF(s.BGMVolume, 0, 1)
	if s.Quality < 0 || s.Quality > 2 {
		s.Quality = 0
	}
	return s
}

// SaveSettings 将设置写入磁盘。
func SaveSettings(d SettingsData) {
	p := settingsPath()
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("创建设置目录失败: %v", err)
		return
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		log.Printf("序列化设置失败: %v", err)
		return
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		log.Printf("写入设置失败: %v", err)
	}
}
