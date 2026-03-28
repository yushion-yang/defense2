// stats.go — 综合会话统计。
// 跟踪单局游戏的全部关键指标，用于结算画面和成就系统。
package session

// Stats 单局游戏的综合统计数据。
type Stats struct {
	Kills        int     // 总击杀数
	Leaked       int     // 泄漏到终点的敌人数
	Damage       float64 // 总造成伤害
	TowersBuilt  int     // 建造的塔总数
	TowersSold   int     // 出售的塔总数
	WavesCleared int     // 已清除波次数
	GoldEarned   float64 // 总获得金币
	GoldSpent    float64 // 总消费金币
	PeakDPS      float64 // 峰值每秒伤害
	ElapsedTime  float64 // 已用时间(秒)

	// DPS 追踪（内部使用）
	dpsWindow  float64   // 当前DPS采样窗口内的伤害累计
	dpsTimer   float64   // DPS采样计时器
	dpsSamples []float64 // DPS快照（每秒一次）
}

// New 创建空的统计实例。
func New() *Stats {
	return &Stats{
		dpsSamples: make([]float64, 0, 120),
	}
}

// RecordKill 记录一次击杀。
func (s *Stats) RecordKill() {
	s.Kills++
}

// RecordLeak 记录一次泄漏。
func (s *Stats) RecordLeak() {
	s.Leaked++
}

// RecordDamage 记录一次伤害。
func (s *Stats) RecordDamage(amount float64) {
	s.Damage += amount
	s.dpsWindow += amount
}

// RecordTowerBuilt 记录建塔。
func (s *Stats) RecordTowerBuilt() {
	s.TowersBuilt++
}

// RecordTowerSold 记录卖塔。
func (s *Stats) RecordTowerSold() {
	s.TowersSold++
}

// RecordWaveCleared 记录波次清除。
func (s *Stats) RecordWaveCleared() {
	s.WavesCleared++
}

// RecordGoldEarned 记录金币收入。
func (s *Stats) RecordGoldEarned(amount float64) {
	s.GoldEarned += amount
}

// RecordGoldSpent 记录金币支出。
func (s *Stats) RecordGoldSpent(amount float64) {
	s.GoldSpent += amount
}

// Tick 每帧调用，更新DPS采样和时间。
// dt: 帧间隔(秒)。每1秒采样一次DPS。
func (s *Stats) Tick(dt float64) {
	s.ElapsedTime += dt
	s.dpsTimer += dt

	// 每秒采样一次DPS
	if s.dpsTimer >= 1.0 {
		currentDPS := s.dpsWindow / s.dpsTimer
		s.dpsSamples = append(s.dpsSamples, currentDPS)
		if currentDPS > s.PeakDPS {
			s.PeakDPS = currentDPS
		}
		s.dpsWindow = 0
		s.dpsTimer = 0
	}
}

// AverageDPS 计算平均DPS。
func (s *Stats) AverageDPS() float64 {
	if len(s.dpsSamples) == 0 {
		return 0
	}
	total := 0.0
	for _, v := range s.dpsSamples {
		total += v
	}
	return total / float64(len(s.dpsSamples))
}

// DPSSamples 返回DPS快照的副本（只读用途）。
func (s *Stats) DPSSamples() []float64 {
	cp := make([]float64, len(s.dpsSamples))
	copy(cp, s.dpsSamples)
	return cp
}
