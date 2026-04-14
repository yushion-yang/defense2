// progress.go — 玩家进度管理。
// 管理高分记录、关卡解锁、通关成就等持久化数据。
package persistence

import (
	"fmt"
	"log"
	"strings"

	"defense2/internal/i18n"
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
	{"map_06", []string{"map:map_07", "map:map_08", "warden:skystrike", "warden:envoy"}},
}

// unlockRequirementText 返回锁定项的解锁条件文本。
// 查找 unlockRules 中哪条规则包含该项，返回需要通关的地图名。
// unlockRequirementMap maps locked items to their required map IDs.
// The display text is resolved at runtime via i18n.
var unlockRequirementMap = map[string]string{
	"map:map_02":       "map_01",
	"tower:shotgun":    "map_01",
	"map:map_03":       "map_02",
	"map:map_04":       "map_02",
	"warden:core":      "map_02",
	"tower:prism":      "map_02",
	"map:map_05":       "map_04",
	"map:map_06":       "map_04",
	"warden:chain":     "map_04",
	"tower:cyclone":    "map_04",
	"map:map_07":       "map_06",
	"map:map_08":       "map_06",
	"warden:skystrike": "map_06",
	"warden:envoy":     "map_06",
}

// UnlockRequirement 返回指定项的解锁条件描述。
// prefix 为 "map"/"tower"/"warden"，key 为具体 ID。
func UnlockRequirement(prefix, key string) string {
	full := prefix + ":" + key
	if reqMap, ok := unlockRequirementMap[full]; ok {
		mapName := i18n.T("progress.map." + reqMap + ".name")
		return i18n.TF("progress.unlock_requirement", mapName)
	}
	return ""
}

// MapRecord 每地图每难度的成绩记录。
type MapRecord struct {
	Stars     int  `json:"stars"`     // 0-3
	BestScore int  `json:"bestScore"` // Mode.GetScore()
	BestKills int  `json:"bestKills"` // 击杀数
	Cleared   bool `json:"cleared"`   // 是否通关过
}

// BestiaryData 图鉴统计数据。
type BestiaryData struct {
	EnemyKills   map[string]int `json:"enemyKills"`   // archetypeID -> 总击杀数
	AbilityPicks map[string]int `json:"abilityPicks"` // abilityType -> 选择次数
	WardenGames  map[string]int `json:"wardenGames"`  // wardenKey -> 使用局数
}

// Progress 玩家进度数据（序列化到存储中）。
type Progress struct {
	HighScores   map[string]int  `json:"highScores"`   // 各关卡最高击杀数（mapID → kills）
	UnlockedMaps []string        `json:"unlockedMaps"` // 已解锁关卡 ID 列表（向后兼容）
	TotalKills   int             `json:"totalKills"`   // 累计击杀总数
	TotalWins    int             `json:"totalWins"`    // 累计胜利次数
	TotalGames   int             `json:"totalGames"`   // 累计游戏场次
	TutorialDone bool            `json:"tutorialDone"` // 教程是否已完成
	Unlocks      UnlockData      `json:"unlocks"`      // 解锁进度
	MascotShown  map[string]bool `json:"mascotShown"`  // 吉祥物 Once 对话已展示 ID 集合

	// ── V1.0 新增 ──
	MapRecords    map[string]*MapRecord `json:"mapRecords"`    // "diffID_mapID" -> 成绩
	Bestiary      BestiaryData          `json:"bestiary"`      // 图鉴统计
	FirstRunDone  bool                  `json:"firstRunDone"`  // 首次语言选择完成
	TotalPlayTime float64               `json:"totalPlayTime"` // 总游玩时长(秒)
}

// NewProgress 创建初始进度（默认解锁 map_01）。
func NewProgress() *Progress {
	return &Progress{
		HighScores:   make(map[string]int),
		UnlockedMaps: []string{"map_01"},
		Unlocks:      NewUnlockData(),
		MascotShown:  make(map[string]bool),
		MapRecords:   make(map[string]*MapRecord),
		Bestiary: BestiaryData{
			EnemyKills:   make(map[string]int),
			AbilityPicks: make(map[string]int),
			WardenGames:  make(map[string]int),
		},
	}
}

const progressKey = "progress"

// ProgressManager 进度管理器，封装存储读写。
type ProgressManager struct {
	storage  Storage   // 底层存储
	progress *Progress // 内存中的进度缓存
}

// DefaultProgressManager 返回使用默认存储的进度管理器（简化初始化模式）。
func DefaultProgressManager() *ProgressManager {
	store, err := DefaultStorage()
	if err != nil {
		log.Printf("[persistence] storage init failed: %v", err)
	}
	return NewProgressManager(store)
}

// NewProgressManager 创建进度管理器，从存储加载已有进度。
func NewProgressManager(s Storage) *ProgressManager {
	pm := &ProgressManager{
		storage:  s,
		progress: NewProgress(),
	}
	// 尝试加载已有进度
	if s.Has(progressKey) {
		if err := s.Get(progressKey, pm.progress); err != nil {
			log.Printf("[persistence] load progress failed: %v", err)
		}
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
	// 确保 V1.0 新增 map 字段非 nil（旧存档兼容）
	if p.MapRecords == nil {
		p.MapRecords = make(map[string]*MapRecord)
	}
	if p.Bestiary.EnemyKills == nil {
		p.Bestiary.EnemyKills = make(map[string]int)
	}
	if p.Bestiary.AbilityPicks == nil {
		p.Bestiary.AbilityPicks = make(map[string]int)
	}
	if p.Bestiary.WardenGames == nil {
		p.Bestiary.WardenGames = make(map[string]int)
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

// GameResultParams 游戏结算参数。
type GameResultParams struct {
	ModeID       string
	MapID        string
	DifficultyID string
	WardenKey    string
	Kills        int
	Score        int
	Stars        int
	Won          bool
	ElapsedSecs  float64
	EnemyKills   map[string]int // archetypeID -> kills this game
	AbilityPicks []string       // abilities picked this game
}

// RecordGameResult 记录一场游戏结果。
// modeID + mapID 组合键存储高分，兼容旧版纯 mapID 键。
// 返回本次解锁的新内容列表（用于 UI 提示）。
func (pm *ProgressManager) RecordGameResult(modeID, mapID string, kills int, won bool) []string {
	return pm.RecordGameResultFull(GameResultParams{
		ModeID: modeID,
		MapID:  mapID,
		Kills:  kills,
		Won:    won,
	})
}

// RecordGameResultFull 记录完整游戏结果（含星级、分数、图鉴数据）。
func (pm *ProgressManager) RecordGameResultFull(params GameResultParams) []string {
	p := pm.progress
	p.TotalGames++
	p.TotalKills += params.Kills
	p.TotalPlayTime += params.ElapsedSecs

	// 图鉴: 敌人击杀
	for archID, count := range params.EnemyKills {
		p.Bestiary.EnemyKills[archID] += count
	}
	// 图鉴: 能力选择
	for _, abilityType := range params.AbilityPicks {
		p.Bestiary.AbilityPicks[abilityType]++
	}
	// 图鉴: 战灵使用
	if params.WardenKey != "" && params.WardenKey != "none" {
		p.Bestiary.WardenGames[params.WardenKey]++
	}

	var newUnlocks []string
	if params.Won {
		p.TotalWins++
		// 旧版高分（向后兼容）
		scoreKey := params.ModeID + "_" + params.MapID
		if params.Kills > p.HighScores[scoreKey] {
			p.HighScores[scoreKey] = params.Kills
		}
		// 新版 MapRecord
		if params.DifficultyID != "" {
			recordKey := params.DifficultyID + "_" + params.MapID
			rec := p.MapRecords[recordKey]
			if rec == nil {
				rec = &MapRecord{}
				p.MapRecords[recordKey] = rec
			}
			rec.Cleared = true
			if params.Stars > rec.Stars {
				rec.Stars = params.Stars
			}
			if params.Score > rec.BestScore {
				rec.BestScore = params.Score
			}
			if params.Kills > rec.BestKills {
				rec.BestKills = params.Kills
			}
		}
		// 解锁
		pm.unlockNext(params.MapID)
		newUnlocks = pm.applyUnlockRules(params.MapID)
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
	parts := strings.SplitN(item, ":", 2)
	if len(parts) != 2 {
		return item
	}
	prefix, key := parts[0], parts[1]
	switch prefix {
	case "map":
		return i18n.TF("progress.unlock.map", i18n.T("progress.map."+key+".name"))
	case "tower":
		return i18n.TF("progress.unlock.tower", i18n.T("progress.tower."+key+".name"))
	case "warden":
		return i18n.TF("progress.unlock.warden", i18n.T("progress.warden."+key+".name"))
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

// MascotShownIDs returns the set of mascot dialog IDs already shown.
func (pm *ProgressManager) MascotShownIDs() map[string]bool {
	if pm.progress.MascotShown == nil {
		pm.progress.MascotShown = make(map[string]bool)
	}
	return pm.progress.MascotShown
}

// SaveMascotShown persists the set of mascot dialog IDs that have been shown.
func (pm *ProgressManager) SaveMascotShown(ids map[string]bool) {
	pm.progress.MascotShown = ids
	pm.save()
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

// GetMapRecord 获取指定难度+地图的成绩记录，无记录时返回 nil。
func (pm *ProgressManager) GetMapRecord(diffID, mapID string) *MapRecord {
	return pm.progress.MapRecords[diffID+"_"+mapID]
}

// GetBestiary 返回图鉴统计数据的只读副本。
func (pm *ProgressManager) GetBestiary() BestiaryData {
	return pm.progress.Bestiary
}

// SetFirstRunDone 标记首次运行（语言选择）已完成。
func (pm *ProgressManager) SetFirstRunDone() {
	pm.progress.FirstRunDone = true
	pm.save()
}

// IsFirstRunDone 检查首次运行是否已完成。
func (pm *ProgressManager) IsFirstRunDone() bool {
	return pm.progress.FirstRunDone
}

func (pm *ProgressManager) save() {
	if err := pm.storage.Set(progressKey, pm.progress); err != nil {
		log.Printf("warning: failed to save progress: %v", err)
	}
}
