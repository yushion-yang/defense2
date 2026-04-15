// progress.go — 玩家进度管理。
// 管理高分记录、关卡解锁、通关成就等持久化数据。
// 所有解锁/星级/高分按 modeID 隔离，各模式独立进度。
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
	{"map_02", []string{"map:map_03", "warden:core", "tower:prism"}},
	{"map_03", []string{"map:map_04"}},
	{"map_04", []string{"map:map_05", "warden:chain", "tower:cyclone"}},
	{"map_05", []string{"map:map_06"}},
	{"map_06", []string{"map:map_07", "warden:skystrike", "warden:envoy"}},
	{"map_07", []string{"map:map_08"}},
}

// unlockRequirementMap 锁定项 → 需要通关的地图 ID。
var unlockRequirementMap = map[string]string{
	"map:map_02":       "map_01",
	"tower:shotgun":    "map_01",
	"map:map_03":       "map_02",
	"warden:core":      "map_02",
	"tower:prism":      "map_02",
	"map:map_04":       "map_03",
	"map:map_05":       "map_04",
	"warden:chain":     "map_04",
	"tower:cyclone":    "map_04",
	"map:map_06":       "map_05",
	"map:map_07":       "map_06",
	"warden:skystrike": "map_06",
	"warden:envoy":     "map_06",
	"map:map_08":       "map_07",
}

// UnlockRequirement 返回指定项的解锁条件描述。
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
// 解锁/星级/高分按 modeID 隔离，全局统计(击杀/胜场/图鉴)跨模式共享。
type Progress struct {
	// ── 按模式隔离的进度 ──
	ModeUnlocks map[string]*UnlockData `json:"modeUnlocks"` // modeID -> 独立解锁
	ModeRecords map[string]*MapRecord  `json:"modeRecords"` // "modeID_diffID_mapID" -> 成绩
	ModeScores  map[string]int         `json:"modeScores"`  // "modeID_mapID" -> 最高击杀

	// ── 全局统计（跨模式共享）──
	TotalKills    int             `json:"totalKills"`
	TotalWins     int             `json:"totalWins"`
	TotalGames    int             `json:"totalGames"`
	TotalPlayTime float64         `json:"totalPlayTime"`
	TutorialDone  bool            `json:"tutorialDone"`
	MascotShown   map[string]bool `json:"mascotShown"`
	Bestiary      BestiaryData    `json:"bestiary"`
	FirstRunDone  bool            `json:"firstRunDone"`

	// ── 旧字段（迁移兼容，不再主动写入）──
	HighScores   map[string]int        `json:"highScores"`
	UnlockedMaps []string              `json:"unlockedMaps"`
	Unlocks      UnlockData            `json:"unlocks"`
	MapRecords   map[string]*MapRecord `json:"mapRecords"`
}

// NewProgress 创建初始进度。
func NewProgress() *Progress {
	return &Progress{
		ModeUnlocks:  make(map[string]*UnlockData),
		ModeRecords:  make(map[string]*MapRecord),
		ModeScores:   make(map[string]int),
		HighScores:   make(map[string]int),
		UnlockedMaps: []string{"map_01"},
		Unlocks:      NewUnlockData(),
		MapRecords:   make(map[string]*MapRecord),
		MascotShown:  make(map[string]bool),
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
	storage  Storage
	progress *Progress
}

// DefaultProgressManager 返回使用默认存储的进度管理器。
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
	if s.Has(progressKey) {
		if err := s.Get(progressKey, pm.progress); err != nil {
			log.Printf("[persistence] load progress failed: %v", err)
		}
	}
	pm.migrateUnlocks()
	return pm
}

// modeUnlockData 返回指定模式的解锁数据，不存在则创建默认值。
func (pm *ProgressManager) modeUnlockData(modeID string) *UnlockData {
	if pm.progress.ModeUnlocks == nil {
		pm.progress.ModeUnlocks = make(map[string]*UnlockData)
	}
	ud, ok := pm.progress.ModeUnlocks[modeID]
	if !ok {
		d := NewUnlockData()
		ud = &d
		pm.progress.ModeUnlocks[modeID] = ud
	}
	// 保证子 map 非 nil
	if ud.Maps == nil {
		ud.Maps = map[string]bool{"map_01": true}
	}
	if ud.Towers == nil {
		ud.Towers = map[string]bool{"basic": true}
	}
	if ud.Wardens == nil {
		ud.Wardens = map[string]bool{"prince": true}
	}
	return ud
}

// migrateUnlocks 将旧版数据迁移到按模式隔离的新结构。
// 旧 Unlocks/HighScores/MapRecords 全部归入 "casual" 模式。
func (pm *ProgressManager) migrateUnlocks() {
	p := pm.progress

	// 确保新字段非 nil
	if p.ModeUnlocks == nil {
		p.ModeUnlocks = make(map[string]*UnlockData)
	}
	if p.ModeRecords == nil {
		p.ModeRecords = make(map[string]*MapRecord)
	}
	if p.ModeScores == nil {
		p.ModeScores = make(map[string]int)
	}
	if p.MascotShown == nil {
		p.MascotShown = make(map[string]bool)
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

	// ── 迁移旧 Unlocks → ModeUnlocks["casual"] ──
	campUnlock := pm.modeUnlockData("casual")

	// 旧 Unlocks.Maps
	if p.Unlocks.Maps != nil {
		for k, v := range p.Unlocks.Maps {
			if v {
				campUnlock.Maps[k] = true
			}
		}
	}
	// 旧 UnlockedMaps 列表
	for _, mapID := range p.UnlockedMaps {
		campUnlock.Maps[mapID] = true
	}
	// 旧 Unlocks.Towers / Wardens
	if p.Unlocks.Towers != nil {
		for k, v := range p.Unlocks.Towers {
			if v {
				campUnlock.Towers[k] = true
			}
		}
	}
	if p.Unlocks.Wardens != nil {
		for k, v := range p.Unlocks.Wardens {
			if v {
				campUnlock.Wardens[k] = true
			}
		}
	}

	// ── 迁移旧 HighScores → ModeScores ──
	if p.HighScores != nil {
		for key, score := range p.HighScores {
			// 旧 key 格式: "modeID_mapID" 或纯 "mapID"
			if _, exists := p.ModeScores[key]; !exists {
				if strings.Contains(key, "_") {
					// 已有 modeID 前缀，直接迁移
					p.ModeScores[key] = score
				} else {
					// 纯 mapID，归入 campaign
					p.ModeScores["campaign_"+key] = score
				}
			}
		}
	}

	// ── 迁移旧 MapRecords → ModeRecords ──
	if p.MapRecords != nil {
		for key, rec := range p.MapRecords {
			// 旧 key 格式: "diffID_mapID"，归入 campaign
			newKey := "campaign_" + key
			if _, exists := p.ModeRecords[newKey]; !exists {
				p.ModeRecords[newKey] = rec
			}
		}
	}

	// 确保默认项
	campUnlock.Maps["map_01"] = true
	campUnlock.Towers["basic"] = true
	campUnlock.Wardens["prince"] = true

	// 基于已通关地图重新应用解锁规则（恢复可能缺失的 tower/warden 解锁）
	for _, mode := range []string{"casual", "classic"} {
		for _, rule := range unlockRules {
			if pm.isMapCleared(mode, rule.Requires) {
				for _, item := range rule.Unlocks {
					pm.applyUnlockToMode(mode, item)
				}
			}
		}
	}
}

// ── 公开 API ──

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

// RecordGameResult 记录一场游戏结果（简化接口）。
func (pm *ProgressManager) RecordGameResult(modeID, mapID string, kills int, won bool) []string {
	return pm.RecordGameResultFull(GameResultParams{
		ModeID: modeID,
		MapID:  mapID,
		Kills:  kills,
		Won:    won,
	})
}

// RecordGameResultFull 记录完整游戏结果。
// 解锁/高分/星级全部写入对应 modeID 的隔离空间。
func (pm *ProgressManager) RecordGameResultFull(params GameResultParams) []string {
	p := pm.progress
	p.TotalGames++
	p.TotalKills += params.Kills
	p.TotalPlayTime += params.ElapsedSecs

	// 图鉴统计（全局共享）
	for archID, count := range params.EnemyKills {
		p.Bestiary.EnemyKills[archID] += count
	}
	for _, abilityType := range params.AbilityPicks {
		p.Bestiary.AbilityPicks[abilityType]++
	}
	if params.WardenKey != "" && params.WardenKey != "none" {
		p.Bestiary.WardenGames[params.WardenKey]++
	}

	var newUnlocks []string
	if params.Won {
		p.TotalWins++
		mode := params.ModeID

		// 高分（按模式隔离）
		scoreKey := mode + "_" + params.MapID
		if params.Kills > p.ModeScores[scoreKey] {
			p.ModeScores[scoreKey] = params.Kills
		}
		// 旧字段同步（向后兼容）
		if params.Kills > p.HighScores[scoreKey] {
			p.HighScores[scoreKey] = params.Kills
		}

		// 星级记录（按模式隔离）
		if params.DifficultyID != "" {
			recordKey := mode + "_" + params.DifficultyID + "_" + params.MapID
			rec := p.ModeRecords[recordKey]
			if rec == nil {
				rec = &MapRecord{}
				p.ModeRecords[recordKey] = rec
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

		// 解锁（按模式隔离）
		pm.unlockNext(mode, params.MapID)
		newUnlocks = pm.applyUnlockRules(mode, params.MapID)
	}
	pm.save()
	return newUnlocks
}

// IsMapUnlocked 检查关卡在指定模式下是否已解锁。
func (pm *ProgressManager) IsMapUnlocked(modeID, mapID string) bool {
	ud := pm.modeUnlockData(modeID)
	return ud.Maps[mapID]
}

// IsTowerUnlocked 检查塔类型是否已解锁（任意模式解锁即可用）。
func (pm *ProgressManager) IsTowerUnlocked(towerKey string) bool {
	for _, ud := range pm.progress.ModeUnlocks {
		if ud.Towers[towerKey] {
			return true
		}
	}
	return false
}

// IsWardenUnlocked 检查战灵类型是否已解锁（任意模式解锁即可用）。
func (pm *ProgressManager) IsWardenUnlocked(wardenKey string) bool {
	if wardenKey == "none" {
		return true
	}
	for _, ud := range pm.progress.ModeUnlocks {
		if ud.Wardens[wardenKey] {
			return true
		}
	}
	return false
}

// LoadBestScore 加载最高分（按模式隔离）。
func (pm *ProgressManager) LoadBestScore(modeID, mapID string) int {
	key := modeID + "_" + mapID
	if score, ok := pm.progress.ModeScores[key]; ok {
		return score
	}
	// 旧键兼容
	if score, ok := pm.progress.HighScores[key]; ok {
		return score
	}
	if score, ok := pm.progress.HighScores[mapID]; ok {
		return score
	}
	return 0
}

// GetMapRecord 获取指定模式+难度+地图的成绩记录。
func (pm *ProgressManager) GetMapRecord(modeID, diffID, mapID string) *MapRecord {
	return pm.progress.ModeRecords[modeID+"_"+diffID+"_"+mapID]
}

// SetTutorialDone 标记教程完成。
func (pm *ProgressManager) SetTutorialDone() {
	pm.progress.TutorialDone = true
	pm.save()
}

// MascotShownIDs 返回吉祥物已展示对话 ID 集合。
func (pm *ProgressManager) MascotShownIDs() map[string]bool {
	if pm.progress.MascotShown == nil {
		pm.progress.MascotShown = make(map[string]bool)
	}
	return pm.progress.MascotShown
}

// SaveMascotShown 持久化吉祥物已展示对话 ID。
func (pm *ProgressManager) SaveMascotShown(ids map[string]bool) {
	pm.progress.MascotShown = ids
	pm.save()
}

// Progress 返回当前进度快照。
func (pm *ProgressManager) Progress() *Progress {
	return pm.progress
}

// GetBestiary 返回图鉴统计数据。
func (pm *ProgressManager) GetBestiary() BestiaryData {
	return pm.progress.Bestiary
}

// SetFirstRunDone 标记首次运行已完成。
func (pm *ProgressManager) SetFirstRunDone() {
	pm.progress.FirstRunDone = true
	pm.save()
}

// IsFirstRunDone 检查首次运行是否已完成。
func (pm *ProgressManager) IsFirstRunDone() bool {
	return pm.progress.FirstRunDone
}

// ── 内部方法 ──

// unlockNext 通关后解锁下一关（按模式隔离）。
func (pm *ProgressManager) unlockNext(modeID, mapID string) {
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

	ud := pm.modeUnlockData(modeID)
	ud.Maps[next] = true
}

// applyUnlockRules 根据通关地图应用解锁规则，返回新解锁项的显示名。
func (pm *ProgressManager) applyUnlockRules(modeID, clearedMapID string) []string {
	var newUnlocks []string
	for _, rule := range unlockRules {
		if rule.Requires != clearedMapID {
			continue
		}
		for _, item := range rule.Unlocks {
			if pm.isAlreadyUnlocked(modeID, item) {
				continue
			}
			pm.applyUnlockToMode(modeID, item)
			newUnlocks = append(newUnlocks, unlockDisplayName(item))
		}
	}
	return newUnlocks
}

// isAlreadyUnlocked 检查某项在指定模式下是否已解锁。
func (pm *ProgressManager) isAlreadyUnlocked(modeID, item string) bool {
	parts := strings.SplitN(item, ":", 2)
	if len(parts) != 2 {
		return false
	}
	ud := pm.modeUnlockData(modeID)
	prefix, key := parts[0], parts[1]
	switch prefix {
	case "map":
		return ud.Maps[key]
	case "tower":
		return ud.Towers[key]
	case "warden":
		return ud.Wardens[key]
	}
	return false
}

// applyUnlockToMode 将解锁项应用到指定模式。
func (pm *ProgressManager) applyUnlockToMode(modeID, item string) {
	parts := strings.SplitN(item, ":", 2)
	if len(parts) != 2 {
		return
	}
	ud := pm.modeUnlockData(modeID)
	prefix, key := parts[0], parts[1]
	switch prefix {
	case "map":
		ud.Maps[key] = true
	case "tower":
		ud.Towers[key] = true
	case "warden":
		ud.Wardens[key] = true
	}
}

// isMapCleared 检查某地图在指定模式下是否曾通关。
func (pm *ProgressManager) isMapCleared(modeID, mapID string) bool {
	key := modeID + "_" + mapID
	if _, ok := pm.progress.ModeScores[key]; ok {
		return true
	}
	// 旧数据兼容
	if _, ok := pm.progress.HighScores[key]; ok {
		return true
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

func (pm *ProgressManager) save() {
	if err := pm.storage.Set(progressKey, pm.progress); err != nil {
		log.Printf("warning: failed to save progress: %v", err)
	}
}
