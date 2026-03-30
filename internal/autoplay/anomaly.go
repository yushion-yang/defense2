// anomaly.go — 运行时异常检测器。
// 每帧检查游戏状态，识别敌人卡住、金币异常、FPS 下降等问题。
package autoplay

import "fmt"

// AnomalySeverity 异常严重程度。
type AnomalySeverity int

const (
	SeverityLow      AnomalySeverity = iota // 低
	SeverityMedium                          // 中
	SeverityHigh                            // 高
	SeverityCritical                        // 严重
)

// String 返回严重程度的字符串表示。
func (s AnomalySeverity) String() string {
	switch s {
	case SeverityLow:
		return "LOW"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityHigh:
		return "HIGH"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Anomaly 运行时异常记录。
type Anomaly struct {
	Tick     int             `json:"tick"`
	Type     string          `json:"type"`
	Detail   string          `json:"detail"`
	Severity AnomalySeverity `json:"severity"`
}

// AnomalyDetector 运行时异常检测器。
type AnomalyDetector struct {
	prevEnemyPos  map[int][2]float64 // enemyID -> (x,y)
	stuckCounters map[int]int        // enemyID -> 连续静止帧数
	prevGold      int
	prevLives     int
	slowFrames    int      // 连续慢帧计数
	anomalies     []Anomaly // 历史异常列表
	initialized   bool
}

// stuckThreshold 判定敌人卡住的帧数阈值。
const stuckThreshold = 60

// NewAnomalyDetector 创建异常检测器。
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		prevEnemyPos:  make(map[int][2]float64),
		stuckCounters: make(map[int]int),
	}
}

// Check 执行一帧的异常检测，返回本帧新发现的异常。
func (d *AnomalyDetector) Check(state *GameState, updateMs float64) []Anomaly {
	var found []Anomaly

	if !d.initialized {
		d.prevGold = state.Gold
		d.prevLives = state.Lives
		d.initialized = true
		return nil
	}

	// 1. 敌人卡住检测
	found = append(found, d.checkEnemyStuck(state)...)

	// 2. 金币为负
	if state.Gold < 0 {
		found = append(found, Anomaly{
			Tick:     state.Tick,
			Type:     "gold_negative",
			Detail:   fmt.Sprintf("gold=%d", state.Gold),
			Severity: SeverityCritical,
		})
	}

	// 3. 金币突增（>500/帧）
	goldDelta := state.Gold - d.prevGold
	if goldDelta > 500 {
		found = append(found, Anomaly{
			Tick:     state.Tick,
			Type:     "gold_spike",
			Detail:   fmt.Sprintf("gold changed %d->%d (delta=%d)", d.prevGold, state.Gold, goldDelta),
			Severity: SeverityHigh,
		})
	}

	// 4. 生命骤降（>5/帧）
	livesDelta := d.prevLives - state.Lives
	if livesDelta > 5 {
		found = append(found, Anomaly{
			Tick:     state.Tick,
			Type:     "lives_drop",
			Detail:   fmt.Sprintf("lives dropped %d->%d (delta=%d)", d.prevLives, state.Lives, livesDelta),
			Severity: SeverityMedium,
		})
	}

	// 5. FPS 下降（连续 10 帧 >50ms）
	if updateMs > 50 {
		d.slowFrames++
		if d.slowFrames >= 10 {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "fps_drop",
				Detail:   fmt.Sprintf("update took %.1fms for %d consecutive frames", updateMs, d.slowFrames),
				Severity: SeverityMedium,
			})
			d.slowFrames = 0 // 重置避免重复报告
		}
	} else {
		d.slowFrames = 0
	}

	// 6. 塔在非建造位置（tower_orphan）— 需要更多上下文，暂通过 BuildCells 间接检查

	// 7. 死敌行走（HP<=0 仍 Active 且非 Dying）
	for _, e := range state.Enemies {
		if e.Active && !e.Dying && e.HP <= 0 {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "dead_enemy_walking",
				Detail:   fmt.Sprintf("enemy#%d archetype=%s HP=%.1f active=true dying=false", e.ID, e.Archetype, e.HP),
				Severity: SeverityCritical,
			})
		}
	}

	// 更新前帧状态
	d.prevGold = state.Gold
	d.prevLives = state.Lives

	// 记录异常
	d.anomalies = append(d.anomalies, found...)
	return found
}

// checkEnemyStuck 检查敌人是否卡住（位置连续 60 帧不变，非 Dying）。
func (d *AnomalyDetector) checkEnemyStuck(state *GameState) []Anomaly {
	var found []Anomaly
	activeIDs := make(map[int]bool)

	for _, e := range state.Enemies {
		if !e.Active || e.Dying {
			continue
		}
		activeIDs[e.ID] = true

		prev, exists := d.prevEnemyPos[e.ID]
		if !exists {
			d.prevEnemyPos[e.ID] = [2]float64{e.X, e.Y}
			d.stuckCounters[e.ID] = 0
			continue
		}

		// 位置几乎不变（<0.1 像素）且速度非零（非 stunned/rooted）
		dx := e.X - prev[0]
		dy := e.Y - prev[1]
		if dx*dx+dy*dy < 0.01 && e.Speed > 0 {
			d.stuckCounters[e.ID]++
			if d.stuckCounters[e.ID] == stuckThreshold {
				found = append(found, Anomaly{
					Tick:     state.Tick,
					Type:     "enemy_stuck",
					Detail:   fmt.Sprintf("enemy#%d at (%.1f,%.1f) not moving for %d ticks", e.ID, e.X, e.Y, stuckThreshold),
					Severity: SeverityHigh,
				})
			}
		} else {
			d.stuckCounters[e.ID] = 0
		}
		d.prevEnemyPos[e.ID] = [2]float64{e.X, e.Y}
	}

	// 清理已消失的敌人
	for id := range d.prevEnemyPos {
		if !activeIDs[id] {
			delete(d.prevEnemyPos, id)
			delete(d.stuckCounters, id)
		}
	}

	return found
}

// Anomalies 返回所有已发现的异常。
func (d *AnomalyDetector) Anomalies() []Anomaly {
	return d.anomalies
}

// RecordPanic 记录 panic 恢复异常。
func (d *AnomalyDetector) RecordPanic(tick int, detail string) {
	a := Anomaly{
		Tick:     tick,
		Type:     "panic_recovered",
		Detail:   detail,
		Severity: SeverityCritical,
	}
	d.anomalies = append(d.anomalies, a)
}
