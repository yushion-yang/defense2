// report.go — 汇总报告生成器。
// 聚合所有对局记录，输出覆盖率报告和异常统计。
package autoplay

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// CoverageReport 覆盖率汇总报告。
type CoverageReport struct {
	TotalSessions int                `json:"total_sessions"`
	Victories     int                `json:"victories"`
	Defeats       int                `json:"defeats"`
	Timeouts      int                `json:"timeouts"`
	CoverageGaps  map[string][]string `json:"coverage_gaps"`
	AnomalySummary map[string]int     `json:"anomaly_summary"`
	TowerUsage    map[string]int     `json:"tower_usage"`
	ArchetypesSeen map[string]int    `json:"archetypes_seen"`
}

// GenerateReport 从多个对局记录生成汇总报告。
func GenerateReport(records []*SessionRecord) *CoverageReport {
	r := &CoverageReport{
		TotalSessions:  len(records),
		CoverageGaps:   make(map[string][]string),
		AnomalySummary: make(map[string]int),
		TowerUsage:     make(map[string]int),
		ArchetypesSeen: make(map[string]int),
	}

	towersSeen := make(map[string]bool)
	archetypesSeen := make(map[string]bool)
	eventsSeen := make(map[string]bool)
	abilitiesSeen := make(map[string]bool)
	attackStylesSeen := make(map[string]bool)
	skillsSeen := make(map[string]bool)

	for _, rec := range records {
		switch rec.Result {
		case "victory":
			r.Victories++
		case "defeat":
			r.Defeats++
		default:
			r.Timeouts++
		}

		for _, a := range rec.Anomalies {
			r.AnomalySummary[a.Type]++
		}

		for _, t := range rec.Coverage.TowersUsed {
			towersSeen[t] = true
			r.TowerUsage[t]++
		}
		for _, a := range rec.Coverage.EnemyArchetypesSeen {
			archetypesSeen[a] = true
			r.ArchetypesSeen[a]++
		}
		for _, e := range rec.Coverage.EventsChosen {
			eventsSeen[e] = true
		}
		for _, ab := range rec.Coverage.AbilitiesTriggered {
			abilitiesSeen[ab] = true
		}
		for _, as := range rec.Coverage.AttackStylesFired {
			attackStylesSeen[as] = true
		}
		for _, sk := range rec.Coverage.SkillsActivated {
			skillsSeen[sk] = true
		}
	}

	// 检查覆盖缺口
	r.CoverageGaps["towers"] = findGaps(TowerKeys, towersSeen)
	r.CoverageGaps["archetypes"] = findGaps(EnemyArchetypes, archetypesSeen)

	allEvents := []string{
		"bonusGold", "buildDiscount", "killRewardUp", "outputUp", "rangeUp", "speedUp", "enemySlow",
	}
	r.CoverageGaps["events"] = findGaps(allEvents, eventsSeen)
	r.CoverageGaps["skills"] = findGaps(SkillNames, skillsSeen)

	allAttackStyles := []string{"projectile", "laser", "wideBeam", "scatter", "charge", "spin_aoe", "aura_dot"}
	r.CoverageGaps["attack_styles"] = findGaps(allAttackStyles, attackStylesSeen)

	allAbilities := []string{
		"bounce", "chargeShot", "crit", "deathMark", "distanceDamage", "executionBonus",
		"flatDamage", "multiTarget", "percentHpDamage", "percentHpMinor", "splash", "stackDamage",
		"bleedDot", "buffPurge", "burn", "onHitSlow", "stun",
		"attackSpeedAura", "critAura", "damageUpAura", "rangeAura", "soloBoost",
		"curseZone", "poisonZone", "silenceZone",
		"goldPassive", "goldOnKill",
	}
	r.CoverageGaps["abilities"] = findGaps(allAbilities, abilitiesSeen)

	return r
}

// findGaps 找出未覆盖的项。
func findGaps(all []string, seen map[string]bool) []string {
	var gaps []string
	for _, item := range all {
		if !seen[item] {
			gaps = append(gaps, item)
		}
	}
	return gaps
}

// WriteText 输出可读的覆盖率报告。
func (r *CoverageReport) WriteText(w io.Writer) {
	fmt.Fprintf(w, "=== Coverage Report ===\n")
	fmt.Fprintf(w, "Sessions: %d (Win: %d, Lose: %d, Timeout: %d)\n\n",
		r.TotalSessions, r.Victories, r.Defeats, r.Timeouts)

	fmt.Fprintf(w, "Towers:     %d/%d", len(TowerKeys)-len(r.CoverageGaps["towers"]), len(TowerKeys))
	if len(r.CoverageGaps["towers"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["towers"])
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Archetypes: %d/13", 13-len(r.CoverageGaps["archetypes"]))
	if len(r.CoverageGaps["archetypes"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["archetypes"])
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Events:     %d/7", 7-len(r.CoverageGaps["events"]))
	if len(r.CoverageGaps["events"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["events"])
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Skills:     %d/9", 9-len(r.CoverageGaps["skills"]))
	if len(r.CoverageGaps["skills"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["skills"])
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "AttackStyle:%d/7", 7-len(r.CoverageGaps["attack_styles"]))
	if len(r.CoverageGaps["attack_styles"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["attack_styles"])
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Abilities:  %d/27", 27-len(r.CoverageGaps["abilities"]))
	if len(r.CoverageGaps["abilities"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["abilities"])
	}
	fmt.Fprintln(w)

	if len(r.AnomalySummary) > 0 {
		fmt.Fprintf(w, "\nAnomalies:\n")
		types := make([]string, 0, len(r.AnomalySummary))
		for t := range r.AnomalySummary {
			types = append(types, t)
		}
		sort.Strings(types)
		for _, t := range types {
			fmt.Fprintf(w, "  %s: %d\n", t, r.AnomalySummary[t])
		}
	}
}

// WriteJSON 将覆盖率报告写入 JSON 文件。
func (r *CoverageReport) WriteJSON(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// GenerateSummaryReport 从目录中读取所有 report.json 并生成汇总。
func GenerateSummaryReport(outputDir string) {
	var records []*SessionRecord

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		fmt.Printf("read output dir: %v\n", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		reportPath := filepath.Join(outputDir, entry.Name(), "report.json")
		data, err := os.ReadFile(reportPath)
		if err != nil {
			continue
		}
		var rec SessionRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		records = append(records, &rec)
	}

	if len(records) == 0 {
		fmt.Println("No session reports found.")
		return
	}

	report := GenerateReport(records)
	report.WriteText(os.Stdout)

	summaryPath := filepath.Join(outputDir, "coverage_summary.json")
	if err := report.WriteJSON(summaryPath); err != nil {
		fmt.Printf("write summary: %v\n", err)
	} else {
		fmt.Printf("\nSummary written to: %s\n", summaryPath)
	}
}
