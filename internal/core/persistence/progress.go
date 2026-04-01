// progress.go — 玩家进度管理。
// 管理高分记录、关卡解锁、通关成就等持久化数据。
package persistence

import (
	"fmt"
	"strings"
)

// UnlockData 解锁进度数据。
type UnlockData struct {
	Maps    map[string]bool `json:"maps"`    // map_01: true
	Towers  map[string]bool `json:"towers"`  // basic: true
	Wardens map[string]bool `json:"wardens"` // prince: true
}

// NewUnlockData 创建初始解锁数据（默认解锁 map_01/basic/prince）。
func NewUnlockData() UnlockData {
	return UnlockData{
		Maps:    map[string]bool{"map_01": true},
		Towers:  map[string]bool{"basic": true},
		Wardens: map[string]bool{"prince": true},
	}
}

// unlockRule 解锁规则：通关某关后解锁新内容。
type unlockRule struct {
	Requires string   // 需要通关的地图
	Unlocks  []string // 解锁内容（前缀: "map:", "tower:", "warden:"）
}

// unlockRules 解锁规则表。
var unlockRules = []unlockRule{
	{"map_01", []string{"map:map_02", "tower:shotgun"}},
	{"map_02", []string{"map:map_03", "map:map_04", "warden:core", "tower:prism"}},
	{"map_04", []string{"map:map_05", "map:map_06", "warden:chain", "tower:cyclone"}},
	{"map_06", []string{"map:map_07", "map:map_08", "warden:skystrike", "warden:envoy", "tower:railgun"}},
}

// unlockRequirementText 返回锁定项的解锁条件文本。
// 查找 unlockRules 中哪条规则包含该项，返回需要通关的地图名。
var unlockRequirementText = map[string]string{
	"map:map_02":       "通关 蜿蜒峡谷 解锁",
	"tower:shotgun":    "通关 蜿蜒峡谷 解锁",
	"map:map_03":       "通关 交叉路口 解锁",
	"map:map_04":       "通关 交叉路口 解锁",
	"warden:core":      "通关 交叉路口 解锁",
	"tower:prism":      "通关 交叉路口 解锁",
	"map:map_05":       "通关 螺旋要塞 解锁",
	"map:map_06":       "通关 螺旋要塞 解锁",
	"warden:chain":     "通关 螺旋要塞 解锁",
	"tower:cyclone":    "通关 螺旋要塞 解锁",
	"map:map_07":       "通关 迷宫回廊 解锁",
	"map:map_08":       "通关 迷宫回廊 解锁",
	"warden:skystrike": "通关 迷宫回廊 解锁",
	"warden:envoy":     "通关 迷宫回廊 解锁",
	"tower:railgun":    "通关 迷宫回廊 解锁",
}

// UnlockRequirement 返回指定项的解锁条件描述。
// prefix 为 "map"/"tower"/"warden"，key 为具体 ID。
func UnlockRequirement(prefix, key string) string {
	full := prefix + ":" + key
	if text, ok := unlockRequirementText[full]; ok {
		return text
	}
	return ""
}

// Progress 玩家进度数据（序列化到存储中）。
type Progress struct {
	HighScores    map[string]int `json:"highScores"`    // 各关卡最高击杀数（mapID → kills）
	UnlockedMaps  []string       `json:"unlockedMaps"`  // 已解锁关卡 ID 列表（向后兼容）
	TotalKills    int            `json:"totalKills"`    // 累计击杀总数
	TotalWins     int            `json:"totalWins"`     // 累计胜利次数
	TotalGames    int            `json:"totalGames"`    // 累计游戏场次
	TutorialDone  bool           `json:"tutorialDone"`  // 教程是否已完成
	Unlocks       UnlockData     `json:"unlocks"`       // 解锁进度
}

// NewProgress 创建初始进度（默认解锁 map_01）。
func NewProgress() *Progress {
	return &Progress{
		HighScores:   make(map[string]int),
		UnlockedMaps: []string{"map_01"},
		Unlocks:      NewUnlockData(),
	}
}

const progressKey = "progress"

// ProgressManager 进度管理器，封装存储读写。
type ProgressManager struct {
	storage  Storage   // 底层存储
	progress *Progress // 内存中的进度缓存
}

// NewProgressManager 创建进度管理器，从存储加载已有进度。
func NewProgressManager(s Storage) *ProgressManager {
	pm := &ProgressManager{
		storage:  s,
		progress: NewProgress(),
	}
	// 尝试加载已有进度
	if s.Has(progressKey) {
		_ = s.Get(progressKey, pm.progress)
	}
	// 迁移旧数据：如果 Unlocks.Maps 为空但 UnlockedMaps 有值，则同步
	pm.migrateUnlocks()
	return pm
}

// migrateUnlocks 将旧版 UnlockedMaps 列表迁移到新版 Unlocks 结构。
func (pm *ProgressManager) migrateUnlocks() {
	p := pm.progress
	// 确保 Unlocks maps 已初始化
	if p.Unlocks.Maps == nil {
		p.Unlocks.Maps = map[string]bool{"map_01": true}
	}
	if p.Unlocks.Towers == nil {
		p.Unlocks.Towers = map[string]bool{"basic": true}
	}
	if p.Unlocks.Wardens == nil {
		p.Unlocks.Wardens = map[string]bool{"prince": true}
	}
	// 同步旧版 UnlockedMaps 到新版
	for _, mapID := range p.UnlockedMaps {
		p.Unlocks.Maps[mapID] = true
	}
	// 确保默认解锁项始终存在
	p.Unlocks.Maps["map_01"] = true
	p.Unlocks.Towers["basic"] = true
	p.Unlocks.Wardens["prince"] = true

	// 基于已通关地图重新应用解锁规则（恢复可能缺失的 tower/warden 解锁）
	for _, rule := range unlockRules {
		if pm.isMapCleared(rule.Requires) {
			for _, item := range rule.Unlocks {
				pm.applyUnlock(item)
			}
		}
	}
}

// isMapCleared 检查某地图是否曾通关（在 HighScores 中有任意 mode 的记录）。
func (pm *ProgressManager) isMapCleared(mapID string) bool {
	for key := range pm.progress.HighScores {
		// key 格式: "modeID_mapID"，也可能是旧版纯 "mapID"
		if key == mapID || strings.HasSuffix(key, "_"+mapID) {
			return true
		}
	}
	return false
}

// applyUnlock 应用单个解锁项（如 "map:map_02", "tower:shotgun"）。
func (pm *ProgressManager) applyUnlock(item string) {
	parts := strings.SplitN(item, ":", 2)
	if len(parts) != 2 {
		return
	}
	prefix, key := parts[0], parts[1]
	switch prefix {
	case "map":
		pm.progress.Unlocks.Maps[key] = true
		// 同步到旧版列表
		found := false
		for _, id := range pm.progress.UnlockedMaps {
			if id == key {
				found = true
				break
			}
		}
		if !found {
			pm.progress.UnlockedMaps = append(pm.progress.UnlockedMaps, key)
		}
	case "tower":
		pm.progress.Unlocks.Towers[key] = true
	case "warden":
		pm.progress.Unlocks.Wardens[key] = true
	}
}

// Progress 返回当前进度快照。
func (pm *ProgressManager) Progress() *Progress {
	return pm.progress
}

// RecordGameResult 记录一场游戏结果。
// modeID + mapID 组合键存储高分，兼容旧版纯 mapID 键。
// 返回本次解锁的新内容列表（用于 UI 提示）。
func (pm *ProgressManager) RecordGameResult(modeID, mapID string, kills int, won bool) []string {
	p := pm.progress
	p.TotalGames++
	p.TotalKills += kills
	var newUnlocks []string
	if won {
		p.TotalWins++
		// 使用 modeID_mapID 复合键存储高分
		scoreKey := modeID + "_" + mapID
		if kills > p.HighScores[scoreKey] {
			p.HighScores[scoreKey] = kills
		}
		// 解锁下一关（旧逻辑保持兼容）
		pm.unlockNext(mapID)
		// 应用解锁规则
		newUnlocks = pm.applyUnlockRules(mapID)
	}
	pm.save()
	return newUnlocks
}

// applyUnlockRules 根据通关地图应用解锁规则，返回新解锁项的显示名。
func (pm *ProgressManager) applyUnlockRules(clearedMapID string) []string {
	var newUnlocks []string
	for _, rule := range unlockRules {
		if rule.Requires != clearedMapID {
			continue
		}
		for _, item := range rule.Unlocks {
			if pm.isAlreadyUnlocked(item) {
				continue
			}
			pm.applyUnlock(item)
			newUnlocks = append(newUnlocks, unlockDisplayName(item))
		}
	}
	return newUnlocks
}

// isAlreadyUnlocked 检查某项是否已解锁。
func (pm *ProgressManager) isAlreadyUnlocked(item string) bool {
	parts := strings.SplitN(item, ":", 2)
	if len(parts) != 2 {
		return false
	}
	prefix, key := parts[0], parts[1]
	switch prefix {
	case "map":
		return pm.progress.Unlocks.Maps[key]
	case "tower":
		return pm.progress.Unlocks.Towers[key]
	case "warden":
		return pm.progress.Unlocks.Wardens[key]
	}
	return false
}

// unlockDisplayName 将解锁项 ID 转为显示名。
func unlockDisplayName(item string) string {
	names := map[string]string{
		"map:map_02":       "关卡: 交叉路口",
		"map:map_03":       "关卡: 铁壁防线",
		"map:map_04":       "关卡: 螺旋要塞",
		"map:map_05":       "关卡: 双线战场",
		"map:map_06":       "关卡: 迷宫回廊",
		"map:map_07":       "关卡: 极限窄道",
		"map:map_08":       "关卡: 竞技场",
		"tower:shotgun":    "塔: 霰弹塔",
		"tower:prism":      "塔: 棱镜塔",
		"tower:cyclone":    "塔: 旋风塔",
		"tower:railgun":    "塔: 磁轨炮",
		"warden:core":      "战灵: 核心",
		"warden:chain":     "战灵: 连锁",
		"warden:skystrike": "战灵: 天击",
		"warden:envoy":     "战灵: 金灵",
	}
	if name, ok := names[item]; ok {
		return name
	}
	return item
}

// LoadBestScore 加载最高分。优先查新键（modeID_mapID），回退旧键（mapID）。
func (pm *ProgressManager) LoadBestScore(modeID, mapID string) int {
	// 新键：modeID_mapID
	newKey := modeID + "_" + mapID
	if score, ok := pm.progress.HighScores[newKey]; ok {
		return score
	}
	// 旧键兼容：纯 mapID
	if score, ok := pm.progress.HighScores[mapID]; ok {
		return score
	}
	return 0
}

// SaveGameResult 保存游戏结果（RecordGameResult 的别名，语义更清晰）。
func (pm *ProgressManager) SaveGameResult(modeID, mapID string, kills int, won bool) []string {
	return pm.RecordGameResult(modeID, mapID, kills, won)
}

// SetTutorialDone 标记教程完成。
func (pm *ProgressManager) SetTutorialDone() {
	pm.progress.TutorialDone = true
	pm.save()
}

// IsMapUnlocked 检查关卡是否已解锁。
func (pm *ProgressManager) IsMapUnlocked(mapID string) bool {
	// 优先检查新版 Unlocks
	if pm.progress.Unlocks.Maps[mapID] {
		return true
	}
	// 向后兼容旧版列表
	for _, id := range pm.progress.UnlockedMaps {
		if id == mapID {
			return true
		}
	}
	return false
}

// IsTowerUnlocked 检查塔类型是否已解锁。
func (pm *ProgressManager) IsTowerUnlocked(towerKey string) bool {
	return pm.progress.Unlocks.Towers[towerKey]
}

// IsWardenUnlocked 检查战灵类型是否已解锁。
func (pm *ProgressManager) IsWardenUnlocked(wardenKey string) bool {
	// "none" 选项始终可用
	if wardenKey == "none" {
		return true
	}
	return pm.progress.Unlocks.Wardens[wardenKey]
}

// unlockNext 通关后解锁下一关（旧逻辑，保持向后兼容）。
func (pm *ProgressManager) unlockNext(mapID string) {
	// map_01 → map_02, map_02 → map_03, etc.
	if len(mapID) < 5 {
		return
	}
	prefix := mapID[:len(mapID)-2]
	numStr := mapID[len(mapID)-2:]
	num := 0
	for _, c := range numStr {
		num = num*10 + int(c-'0')
	}
	next := prefix + fmt.Sprintf("%02d", num+1)

	// 检查是否已解锁
	for _, id := range pm.progress.UnlockedMaps {
		if id == next {
			return
		}
	}
	pm.progress.UnlockedMaps = append(pm.progress.UnlockedMaps, next)
	pm.progress.Unlocks.Maps[next] = true
}

func (pm *ProgressManager) save() {
	_ = pm.storage.Set(progressKey, pm.progress)
}
