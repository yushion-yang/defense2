// report.go — 汇总报告生成器。
// 聚合所有对局记录，输出覆盖率报告和异常统计。
package autoplay

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// BalanceTestResult 单个平衡测试结果。
type BalanceTestResult struct {
	ID         string            `json:"id"`
	Passed     bool              `json:"passed"`
	Assertions []AssertionResult `json:"assertions"`
}

// BalanceSummary 平衡测试汇总。
type BalanceSummary struct {
	Total   int                 `json:"total"`
	Passed  int                 `json:"passed"`
	Failed  int                 `json:"failed"`
	Details []BalanceTestResult `json:"details"`
}

// CoverageReport 覆盖率汇总报告。
type CoverageReport struct {
	TotalSessions  int                 `json:"total_sessions"`
	Victories      int                 `json:"victories"`
	Defeats        int                 `json:"defeats"`
	Timeouts       int                 `json:"timeouts"`
	CoverageGaps   map[string][]string `json:"coverage_gaps"`
	AnomalySummary map[string]int      `json:"anomaly_summary"`
	TowerUsage     map[string]int      `json:"tower_usage"`
	ArchetypesSeen map[string]int      `json:"archetypes_seen"`
	Balance        *BalanceSummary     `json:"balance_summary,omitempty"`
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
	abilitiesSeen := make(map[string]bool)
	attackStylesSeen := make(map[string]bool)
	pipelineStepsSeen := make(map[string]bool)
	damageTypesSeen := make(map[string]bool)
	buffTypesSeen := make(map[string]bool)
	buffModesSeen := make(map[string]bool)
	enemyTemplatesSeen := make(map[string]bool)
	interactionModesSeen := make(map[string]bool)
	ccSeen := make(map[string]bool)

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
		for _, ab := range rec.Coverage.AbilitiesTriggered {
			abilitiesSeen[ab] = true
		}
		for _, as := range rec.Coverage.AttackStylesFired {
			attackStylesSeen[as] = true
		}
		// 遥测维度
		for _, v := range rec.PipelineSteps {
			pipelineStepsSeen[v] = true
		}
		for _, v := range rec.DamageTypes {
			damageTypesSeen[v] = true
		}
		for _, v := range rec.BuffTypesApplied {
			buffTypesSeen[v] = true
		}
		for _, v := range rec.BuffStackModes {
			buffModesSeen[v] = true
		}
		for _, v := range rec.EnemyBuffTemplates {
			enemyTemplatesSeen[v] = true
		}
		for _, v := range rec.InteractionModes {
			interactionModesSeen[v] = true
		}
		for _, v := range rec.CCApplied {
			ccSeen[v] = true
		}
	}

	// 平衡测试汇总
	r.Balance = buildBalanceSummary(records)

	// 检查覆盖缺口
	r.CoverageGaps["towers"] = findGaps(TowerKeys, towersSeen)
	r.CoverageGaps["archetypes"] = findGaps(EnemyArchetypes, archetypesSeen)

	allAttackStyles := []string{"projectile", "wideBeam", "scatter", "spin_aoe", "radial", "barrage"}
	r.CoverageGaps["attack_styles"] = findGaps(allAttackStyles, attackStylesSeen)

	// 与 tower/ability_ids.go 保持同步的完整能力列表（排除已禁用的 elementSwitch/periodicCast）
	allAbilities := []string{
		// 攻击类
		"splash", "crit", "bounce", "momentum", "executionBonus",
		"flatDamage", "distanceDamage", "multiTarget", "deathMark", "enhance",
		// CC 类
		"slowPower", "slowDuration", "stun", "stunChance", "stunDuration",
		// DoT 类
		"bleedDot", "burn", "poison", "weaken",
		// 光环类
		"damageUpAura", "attackSpeedAura", "rangeAura", "critAura", "soloBoost",
		// 区域类
		"poisonZone", "silenceZone", "curseZone", "weakenZone",
		// 经济/被动类
		"goldPassive",
		// 攻击方式覆盖类
		"scatter", "wideBeam", "spinAoe", "radial", "barrage",
	}
	r.CoverageGaps["abilities"] = findGaps(allAbilities, abilitiesSeen)

	// 遥测维度覆盖
	allPipeline := []string{"immunity_check", "boss_hp_cap", "attacker_buff", "target_debuff", "damage_cap", "hp_deduct", "threshold", "death_check"}
	r.CoverageGaps["pipeline_steps"] = findGaps(allPipeline, pipelineStepsSeen)

	allDmgTypes := []string{"physical", "magic", "true", "pure"}
	r.CoverageGaps["damage_types"] = findGaps(allDmgTypes, damageTypesSeen)

	allBuffTypes := []string{
		"slow", "stun", "knockup", "silence", "disarm",
		"speedUp", "damageUp", "damageDown", "fireRateUp",
		"invincible", "damageImmune", "controlImmune", "slowImmune", "stunImmune", "untargetable",
		"dot", "tenacity",
	}
	r.CoverageGaps["buff_types"] = findGaps(allBuffTypes, buffTypesSeen)

	allBuffModes := []string{"strongest", "additive", "multiplicative", "override", "independent", "independentPerSource"}
	r.CoverageGaps["buff_stack_modes"] = findGaps(allBuffModes, buffModesSeen)

	allTemplates := []string{
		"berserk", "regen", "healAura", "speedAura", "damageReduce",
		"empBurst", "blink", "deathSplit", "deathSlow",
		"reflect", "timewarp", "revive", "spawnMinions",
	}
	r.CoverageGaps["enemy_templates"] = findGaps(allTemplates, enemyTemplatesSeen)

	allIModes := []string{"idle", "buildMenu", "buildPlace", "towerSel", "spawnMenu", "spawnPlace", "paused", "wardenSelect"}
	r.CoverageGaps["interaction_modes"] = findGaps(allIModes, interactionModesSeen)

	allCC := []string{"slow", "stun"}
	r.CoverageGaps["cc_types"] = findGaps(allCC, ccSeen)

	return r
}

// buildBalanceSummary 从对局记录中提取 bal_* 场景的断言结果。
func buildBalanceSummary(records []*SessionRecord) *BalanceSummary {
	var bs BalanceSummary
	for _, rec := range records {
		if len(rec.SessionID) < 4 || rec.SessionID[:4] != "bal_" {
			continue
		}
		if len(rec.Assertions) == 0 {
			continue
		}
		bs.Total++
		allPassed := true
		for _, a := range rec.Assertions {
			if !a.Passed {
				allPassed = false
				break
			}
		}
		if allPassed {
			bs.Passed++
		} else {
			bs.Failed++
		}
		bs.Details = append(bs.Details, BalanceTestResult{
			ID:         rec.SessionID,
			Passed:     allPassed,
			Assertions: rec.Assertions,
		})
	}
	if bs.Total == 0 {
		return nil
	}
	return &bs
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

	fmt.Fprintf(w, "Archetypes: %d/12", 12-len(r.CoverageGaps["archetypes"]))
	if len(r.CoverageGaps["archetypes"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["archetypes"])
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "AttackStyle:%d/7", 7-len(r.CoverageGaps["attack_styles"]))
	if len(r.CoverageGaps["attack_styles"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["attack_styles"])
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Abilities:  %d/26", 26-len(r.CoverageGaps["abilities"]))
	if len(r.CoverageGaps["abilities"]) > 0 {
		fmt.Fprintf(w, "  -- MISSING: %v", r.CoverageGaps["abilities"])
	}
	fmt.Fprintln(w)

	// 遥测维度
	dims := []struct {
		label string
		key   string
		total int
	}{
		{"Pipeline:  ", "pipeline_steps", 8},
		{"DmgTypes:  ", "damage_types", 4},
		{"BuffTypes: ", "buff_types", 19},
		{"BuffModes: ", "buff_stack_modes", 6},
		{"Templates: ", "enemy_templates", 13},
		{"IModes:    ", "interaction_modes", 8},
		{"CC Types:  ", "cc_types", 3},
	}
	for _, d := range dims {
		gaps := r.CoverageGaps[d.key]
		fmt.Fprintf(w, "%s%d/%d", d.label, d.total-len(gaps), d.total)
		if len(gaps) > 0 {
			fmt.Fprintf(w, "  -- MISSING: %v", gaps)
		}
		fmt.Fprintln(w)
	}

	// 平衡测试结果
	if r.Balance != nil {
		fmt.Fprintf(w, "\nBalance Tests: %d/%d passed\n", r.Balance.Passed, r.Balance.Total)
		for _, d := range r.Balance.Details {
			status := "PASS"
			if !d.Passed {
				status = "FAIL"
			}
			fmt.Fprintf(w, "  [%s] %s", status, d.ID)
			if !d.Passed {
				for _, a := range d.Assertions {
					if !a.Passed {
						fmt.Fprintf(w, "  (%s)", a.Name)
					}
				}
			}
			fmt.Fprintln(w)
		}
	}

	if len(r.AnomalySummary) > 0 {
		fmt.Fprintf(w, "\nAnomalies:\n")
		types := make([]string, 0, len(r.AnomalySummary))
		for t := range r.AnomalySummary {
			types = append(types, t)
		}
		slices.Sort(types)
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
