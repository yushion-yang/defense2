// autoplay.go — 自动对局 / 基准测试。
// 提供自动放塔、伤害统计、DPS 采样等基准测试工具。
package gametest

// BenchmarkStats 基准测试统计。
type BenchmarkStats struct {
	TotalDamage  float64   // 累计总伤害
	TotalKills   int       // 累计击杀数
	Leaks        int       // 累计泄漏数
	GoldEarned   float64   // 累计获得金币
	GoldSpent    float64   // 累计消耗金币
	PeakDPS      float64   // 峰值 DPS
	WavesCleared int       // 已通过波次数
	DPSSnapshots []float64 // DPS 采样序列（每秒一个）
	dpsWindow    float64   // 当前采样窗口内累计伤害
	dpsTimer     float64   // 距上次采样的累计时间（秒）
}

// dpsSampleInterval DPS 采样间隔（秒）
const dpsSampleInterval = 1.0

// NewBenchmarkStats 创建空的基准测试统计。
func NewBenchmarkStats() *BenchmarkStats {
	return &BenchmarkStats{
		DPSSnapshots: make([]float64, 0, 128),
	}
}

// RecordDamage 记录一次伤害。
func (b *BenchmarkStats) RecordDamage(amount float64) {
	b.TotalDamage += amount
	b.dpsWindow += amount
}

// RecordKill 记录一次击杀。
func (b *BenchmarkStats) RecordKill() {
	b.TotalKills++
}

// RecordLeak 记录一次泄漏。
func (b *BenchmarkStats) RecordLeak() {
	b.Leaks++
}

// Tick 每帧调用，驱动 DPS 采样（每 1 秒采样一次）。
func (b *BenchmarkStats) Tick(dt float64) {
	b.dpsTimer += dt
	if b.dpsTimer >= dpsSampleInterval {
		// 计算本窗口 DPS
		dps := b.dpsWindow / b.dpsTimer
		b.DPSSnapshots = append(b.DPSSnapshots, dps)
		if dps > b.PeakDPS {
			b.PeakDPS = dps
		}
		// 重置窗口
		b.dpsWindow = 0
		b.dpsTimer = 0
	}
}

// FinalizeBenchmarkResult 生成 JSON 可序列化的基准测试报告。
func FinalizeBenchmarkResult(stats *BenchmarkStats, scenarioID, faction string) map[string]interface{} {
	avgDPS := 0.0
	if len(stats.DPSSnapshots) > 0 {
		sum := 0.0
		for _, d := range stats.DPSSnapshots {
			sum += d
		}
		avgDPS = sum / float64(len(stats.DPSSnapshots))
	}

	return map[string]interface{}{
		"scenarioID":   scenarioID,
		"faction":      faction,
		"totalDamage":  stats.TotalDamage,
		"totalKills":   stats.TotalKills,
		"leaks":        stats.Leaks,
		"goldEarned":   stats.GoldEarned,
		"goldSpent":    stats.GoldSpent,
		"peakDPS":      stats.PeakDPS,
		"avgDPS":       avgDPS,
		"wavesCleared": stats.WavesCleared,
		"dpsSnapshots": stats.DPSSnapshots,
	}
}

// AutoPlaceTowers 自动放塔：将预设塔均匀分配到可用槽位。
// 返回 [(槽位索引, 塔键名)] 对列表。
func AutoPlaceTowers(slots [][2]float64, presetKeys []string) [][2]interface{} {
	if len(slots) == 0 || len(presetKeys) == 0 {
		return nil
	}

	result := make([][2]interface{}, 0, len(slots))
	for i, slot := range slots {
		// 轮询分配塔类型
		towerKey := presetKeys[i%len(presetKeys)]
		_ = slot // 槽位坐标供调用方使用
		result = append(result, [2]interface{}{i, towerKey})
	}
	return result
}
