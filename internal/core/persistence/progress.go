// progress.go — 玩家进度管理。
// 管理高分记录、关卡解锁、通关成就等持久化数据。
package persistence

import "fmt"

// Progress 玩家进度数据（序列化到存储中）。
type Progress struct {
	HighScores    map[string]int `json:"highScores"`    // 各关卡最高击杀数（mapID → kills）
	UnlockedMaps  []string       `json:"unlockedMaps"`  // 已解锁关卡 ID 列表
	TotalKills    int            `json:"totalKills"`    // 累计击杀总数
	TotalWins     int            `json:"totalWins"`     // 累计胜利次数
	TotalGames    int            `json:"totalGames"`    // 累计游戏场次
	TutorialDone  bool           `json:"tutorialDone"`  // 教程是否已完成
}

// NewProgress 创建初始进度（默认解锁 map_01）。
func NewProgress() *Progress {
	return &Progress{
		HighScores:   make(map[string]int),
		UnlockedMaps: []string{"map_01"},
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
	return pm
}

// Progress 返回当前进度快照。
func (pm *ProgressManager) Progress() *Progress {
	return pm.progress
}

// RecordGameResult 记录一场游戏结果。
// modeID + mapID 组合键存储高分，兼容旧版纯 mapID 键。
func (pm *ProgressManager) RecordGameResult(modeID, mapID string, kills int, won bool) {
	p := pm.progress
	p.TotalGames++
	p.TotalKills += kills
	if won {
		p.TotalWins++
		// 使用 modeID_mapID 复合键存储高分
		scoreKey := modeID + "_" + mapID
		if kills > p.HighScores[scoreKey] {
			p.HighScores[scoreKey] = kills
		}
		// 解锁下一关
		pm.unlockNext(mapID)
	}
	pm.save()
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
func (pm *ProgressManager) SaveGameResult(modeID, mapID string, kills int, won bool) {
	pm.RecordGameResult(modeID, mapID, kills, won)
}

// SetTutorialDone 标记教程完成。
func (pm *ProgressManager) SetTutorialDone() {
	pm.progress.TutorialDone = true
	pm.save()
}

// IsMapUnlocked 检查关卡是否已解锁。
func (pm *ProgressManager) IsMapUnlocked(mapID string) bool {
	for _, id := range pm.progress.UnlockedMaps {
		if id == mapID {
			return true
		}
	}
	return false
}

// unlockNext 通关后解锁下一关。
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
}

func (pm *ProgressManager) save() {
	_ = pm.storage.Set(progressKey, pm.progress)
}
