// sim_report.go — 仿真测试平衡报告生成器。
// 聚合多局仿真结果，按维度（难度/地图/战灵/能力）统计，输出 JSON + 文本摘要。
package autoplay

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── 报告数据结构 ──

// SimReport 仿真平衡报告。
type SimReport struct {
	TotalRuns    int                       `json:"total_runs"`
	ByDifficulty map[string]*SimDiffStats  `json:"by_difficulty"`
	ByMap        map[string]*SimMapStats   `json:"by_map"`
	ByWarden     map[string]*SimDiffStats  `json:"by_warden,omitempty"`
	ByAbility    map[string]*SimDiffStats  `json:"by_ability,omitempty"`
	BalanceCurve []SimBalancePoint         `json:"balance_curve"`
	Outliers     []SimOutlier              `json:"outliers"`
}

// SimDiffStats 按维度聚合的统计。
type SimDiffStats struct {
	Runs       int     `json:"runs"`
	Victories  int     `json:"victories"`
	Defeats    int     `json:"defeats"`
	Timeouts   int     `json:"timeouts"`
	AvgWaves   float64 `json:"avg_waves_survived"`
	AvgLives   float64 `json:"avg_final_lives"`
	WinRate    float64 `json:"win_rate"`
	Verdict    string  `json:"verdict"` // too_easy / balanced / too_hard
}

// SimMapStats 每地图跨难度统计。
type SimMapStats struct {
	ByDifficulty map[string]*SimDiffStats `json:"by_difficulty"`
	Verdict      string                   `json:"verdict"` // balanced / easy_outlier / hard_outlier
}

// SimBalancePoint 平衡曲线数据点。
type SimBalancePoint struct {
	MapID      string  `json:"map_id"`
	Difficulty string  `json:"difficulty"`
	WinRate    float64 `json:"win_rate"`
	AvgLives   float64 `json:"avg_lives"`
	AvgWave    float64 `json:"avg_wave_survived"`
}

// SimOutlier 异常点。
type SimOutlier struct {
	MapID      string `json:"map_id"`
	Difficulty string `json:"difficulty"`
	Dimension  string `json:"dimension,omitempty"` // warden/ability/economy
	Value      string `json:"value,omitempty"`      // 具体值（如战灵名）
	Issue      string `json:"issue"`
}

// ── 报告生成 ──

// GenerateSimReport 从 JSON 报告目录生成仿真平衡报告。
func GenerateSimReport(jsonDir string) (*SimReport, error) {
	entries, err := os.ReadDir(jsonDir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	report := &SimReport{
		ByDifficulty: make(map[string]*SimDiffStats),
		ByMap:        make(map[string]*SimMapStats),
		ByWarden:     make(map[string]*SimDiffStats),
		ByAbility:    make(map[string]*SimDiffStats),
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// 只处理 sim_ 前缀的目录
		if !strings.HasPrefix(entry.Name(), "sim_") {
			continue
		}

		reportPath := filepath.Join(jsonDir, entry.Name(), "report.json")
		data, err := os.ReadFile(reportPath)
		if err != nil {
			continue
		}

		var rec SessionRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}

		report.TotalRuns++
		bp := SimBalancePoint{
			MapID:      rec.MapID,
			Difficulty: rec.Difficulty,
			AvgLives:   float64(rec.FinalLives),
			AvgWave:    float64(rec.WavesSurvived),
		}
		if rec.Result == "victory" {
			bp.WinRate = 1.0
		}
		report.BalanceCurve = append(report.BalanceCurve, bp)

		// 按难度聚合
		addToStats(report.ByDifficulty, rec.Difficulty, &rec)

		// 按地图聚合
		if _, ok := report.ByMap[rec.MapID]; !ok {
			report.ByMap[rec.MapID] = &SimMapStats{
				ByDifficulty: make(map[string]*SimDiffStats),
			}
		}
		addToStats(report.ByMap[rec.MapID].ByDifficulty, rec.Difficulty, &rec)

		// 按维度聚合（从 session ID 解析）
		sid := entry.Name()
		if strings.HasPrefix(sid, "sim_warden_") {
			parts := strings.Split(sid, "_")
			if len(parts) >= 3 {
				warden := parts[2]
				addToStats(report.ByWarden, warden, &rec)
			}
		}
		if strings.HasPrefix(sid, "sim_abil_") {
			parts := strings.Split(sid, "_")
			if len(parts) >= 3 {
				cat := parts[2]
				addToStats(report.ByAbility, cat, &rec)
			}
		}
	}

	// 计算统计值和判定
	for _, ds := range report.ByDifficulty {
		finalizeStats(ds)
	}
	for _, ms := range report.ByMap {
		for _, ds := range ms.ByDifficulty {
			finalizeStats(ds)
		}
	}
	for _, ds := range report.ByWarden {
		finalizeStats(ds)
	}
	for _, ds := range report.ByAbility {
		finalizeStats(ds)
	}

	// 判定难度
	for diff, ds := range report.ByDifficulty {
		ds.Verdict = judgeDifficulty(diff, ds)
	}

	// 检测地图异常点
	report.Outliers = detectOutliers(report)

	return report, nil
}

func addToStats(m map[string]*SimDiffStats, key string, rec *SessionRecord) {
	if _, ok := m[key]; !ok {
		m[key] = &SimDiffStats{}
	}
	ds := m[key]
	ds.Runs++
	switch rec.Result {
	case "victory":
		ds.Victories++
	case "defeat":
		ds.Defeats++
	default:
		ds.Timeouts++
	}
	ds.AvgWaves += float64(rec.WavesSurvived)
	ds.AvgLives += float64(rec.FinalLives)
}

func finalizeStats(ds *SimDiffStats) {
	if ds.Runs == 0 {
		return
	}
	ds.AvgWaves /= float64(ds.Runs)
	ds.AvgLives /= float64(ds.Runs)
	ds.WinRate = float64(ds.Victories) / float64(ds.Runs)
}

func judgeDifficulty(diff string, ds *SimDiffStats) string {
	switch diff {
	case "easy":
		if ds.WinRate < 0.9 || ds.AvgLives < 10 {
			return "too_hard"
		}
		return "balanced"
	case "normal":
		if ds.WinRate < 0.7 {
			return "too_hard"
		}
		if ds.AvgLives > 15 {
			return "too_easy"
		}
		return "balanced"
	case "hard":
		if ds.WinRate > 0.7 {
			return "too_easy"
		}
		if ds.AvgWaves < 5 {
			return "too_hard"
		}
		return "balanced"
	case "extreme":
		if ds.WinRate > 0.3 {
			return "too_easy"
		}
		if ds.AvgWaves < 3 {
			return "too_hard"
		}
		return "balanced"
	}
	return "unknown"
}

func detectOutliers(report *SimReport) []SimOutlier {
	var outliers []SimOutlier

	// 检查每个地图在其难度下是否异常
	for mapID, ms := range report.ByMap {
		for diff, ds := range ms.ByDifficulty {
			// 与同难度全局均值比较
			global, ok := report.ByDifficulty[diff]
			if !ok || global.Runs < 2 {
				continue
			}
			// 剩余生命偏差
			if global.Runs > 0 && math.Abs(ds.AvgLives-global.AvgLives) > 5 {
				if ds.AvgLives > global.AvgLives+5 {
					outliers = append(outliers, SimOutlier{
						MapID: mapID, Difficulty: diff,
						Issue: fmt.Sprintf("too easy: avg lives %.1f vs global %.1f", ds.AvgLives, global.AvgLives),
					})
				} else {
					outliers = append(outliers, SimOutlier{
						MapID: mapID, Difficulty: diff,
						Issue: fmt.Sprintf("too hard: avg lives %.1f vs global %.1f", ds.AvgLives, global.AvgLives),
					})
				}
			}
			// 胜率偏差
			if math.Abs(ds.WinRate-global.WinRate) > 0.3 {
				outliers = append(outliers, SimOutlier{
					MapID: mapID, Difficulty: diff,
					Issue: fmt.Sprintf("win rate outlier: %.0f%% vs global %.0f%%", ds.WinRate*100, global.WinRate*100),
				})
			}
		}
	}

	// 战灵间差异检测
	if len(report.ByWarden) > 1 {
		var wardenLives []float64
		for _, ds := range report.ByWarden {
			wardenLives = append(wardenLives, ds.AvgLives)
		}
		sort.Float64s(wardenLives)
		spread := wardenLives[len(wardenLives)-1] - wardenLives[0]
		if spread > 8 {
			for name, ds := range report.ByWarden {
				if ds.AvgLives == wardenLives[0] {
					outliers = append(outliers, SimOutlier{
						Dimension: "warden", Value: name,
						Issue: fmt.Sprintf("weakest warden: avg lives %.1f (spread %.1f)", ds.AvgLives, spread),
					})
				}
			}
		}
	}

	return outliers
}

// ── 输出 ──

// WriteSimReport 写入 JSON 报告和文本摘要。
func WriteSimReport(report *SimReport, dir string) error {
	// JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sim_report.json"), data, 0o644); err != nil {
		return fmt.Errorf("write json: %w", err)
	}

	// 文本摘要
	summary := buildSimSummary(report)
	if err := os.WriteFile(filepath.Join(dir, "sim_summary.txt"), []byte(summary), 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}

	return nil
}

func buildSimSummary(r *SimReport) string {
	var b strings.Builder
	b.WriteString("=== Simulation Balance Report ===\n")
	b.WriteString(fmt.Sprintf("Total runs: %d\n\n", r.TotalRuns))

	// 难度曲线
	b.WriteString("Difficulty Curve:\n")
	for _, diff := range []string{"easy", "normal", "hard", "extreme"} {
		ds, ok := r.ByDifficulty[diff]
		if !ok {
			continue
		}
		icon := "  "
		switch ds.Verdict {
		case "balanced":
			icon = "OK"
		case "too_easy":
			icon = "!!"
		case "too_hard":
			icon = "!!"
		}
		b.WriteString(fmt.Sprintf("  %-8s %d/%d victory (%3.0f%%), avg lives %5.1f, avg wave %4.1f  [%s] %s\n",
			diff, ds.Victories, ds.Runs, ds.WinRate*100, ds.AvgLives, ds.AvgWaves, icon, ds.Verdict))
	}
	b.WriteString("\n")

	// 地图概览
	b.WriteString("Per-Map Results:\n")
	for _, mapID := range Maps {
		ms, ok := r.ByMap[mapID]
		if !ok {
			continue
		}
		b.WriteString(fmt.Sprintf("  %s:", mapID))
		for _, diff := range []string{"easy", "normal", "hard", "extreme"} {
			ds, ok := ms.ByDifficulty[diff]
			if !ok {
				b.WriteString("  ---")
				continue
			}
			if ds.WinRate > 0.5 {
				b.WriteString(fmt.Sprintf("  W(%.0f)", ds.AvgLives))
			} else {
				b.WriteString(fmt.Sprintf("  L(w%d)", int(ds.AvgWaves)))
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// 战灵对比
	if len(r.ByWarden) > 0 {
		b.WriteString("Warden Balance:\n")
		for _, w := range Wardens {
			ds, ok := r.ByWarden[w]
			if !ok {
				continue
			}
			b.WriteString(fmt.Sprintf("  %-10s win %.0f%%, avg lives %4.1f\n", w, ds.WinRate*100, ds.AvgLives))
		}
		b.WriteString("\n")
	}

	// 能力对比
	if len(r.ByAbility) > 0 {
		b.WriteString("Ability Category Balance:\n")
		for cat, ds := range r.ByAbility {
			b.WriteString(fmt.Sprintf("  %-8s win %.0f%%, avg lives %4.1f\n", cat, ds.WinRate*100, ds.AvgLives))
		}
		b.WriteString("\n")
	}

	// 异常点
	if len(r.Outliers) > 0 {
		b.WriteString("Outliers:\n")
		for _, o := range r.Outliers {
			prefix := ""
			if o.MapID != "" {
				prefix = fmt.Sprintf("  %s %s", o.MapID, o.Difficulty)
			} else if o.Dimension != "" {
				prefix = fmt.Sprintf("  [%s] %s", o.Dimension, o.Value)
			}
			b.WriteString(fmt.Sprintf("%s: %s\n", prefix, o.Issue))
		}
	} else {
		b.WriteString("No outliers detected.\n")
	}

	return b.String()
}
