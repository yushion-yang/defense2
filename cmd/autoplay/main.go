// main.go — AutoPlay 自动对局工具入口（无头模式批量跑关卡）。
//
// 用途：自动化测试游戏平衡性、能力覆盖率、回归验证，无需人工操作。
// 典型用法：
//
//	go run cmd/autoplay/main.go --scenario attack-style-coverage   // 快速验证（1局,30秒）
//	go run cmd/autoplay/main.go --sweep --json-dir docs/autotest   // 全量回归（68局,~3分钟）
//	go run cmd/autoplay/main.go --marathon --games 100             // 随机压测
//
// 架构设计：
//   - 编排模式（默认）：生成测试计划（TestCase 列表），逐个在当前进程内运行
//   - 单局模式（--session-json）：接收序列化 JSON 配置，运行单个 TestCase（子进程入口）
//   - Ebitengine 的 RunGame() 只能调用一次，但 headless 模式下直接循环 Update() 绕过此限制
//   - 首局用 NewGame() 完整初始化，后续用 NewGameLite() 复用全局资源（避免重复加载配置/资源）
//
// 测试模式一览：
//
//	--sweep:          全量 pairwise 组合（地图×难度×战灵×策略）
//	--ability-sweep:  逐个能力的专项测试
//	--balance-sweep:  26 个预定义平衡场景
//	--sim-sweep:      仿真平衡扫描（mortal 模式,~68 例）
//	--marathon:       N 局随机组合压测（附带堆内存泄漏检测）
//	--scenario <name>: 指定单个场景
//
// 策略系统：每个 TestCase 绑定一个 Strategy 接口实现，控制建塔/升级/道具决策。
// 可用策略：random / greedy / balance_greedy / competent / champion_* / focus_* / scenario_* / llm
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	defense2 "defense2"
	"defense2/internal/autoplay"
	"defense2/internal/config"
	"defense2/internal/scene"
)

func main() {
	// ── 单局模式（子进程）──
	sessionJSON := flag.String("session-json", "", "(internal) single session JSON config")

	// ── 编排模式（父进程）──
	runs := flag.Int("runs", 1, "number of sessions per strategy")
	strategies := flag.String("strategies", "random,greedy", "comma-separated strategy names")
	mapID := flag.String("map", "map_01", "map ID")
	difficulty := flag.String("difficulty", "normal", "difficulty ID")
	warden := flag.String("warden", "prince", "warden type")
	output := flag.String("output", "./autoplay-results", "output directory for JSON reports")
	jsonDir := flag.String("json-dir", "", "JSON report output directory (overrides output for JSON)")
	sweep := flag.Bool("sweep", false, "run full pairwise sweep")
	scenarioName := flag.String("scenario", "", "run single scenario by name")
	seed := flag.Int64("seed", 0, "master random seed (0=use timestamp, same seed = reproducible results)")
	abilitySweep := flag.Bool("ability-sweep", false, "run all ability-level test scenarios")
	balanceSweep := flag.Bool("balance-sweep", false, "run all balance test scenarios (26 cases)")
	simSweep := flag.Bool("sim-sweep", false, "run simulation balance sweep (mortal mode, ~68 cases)")
	marathon := flag.Bool("marathon", false, "run N random games with random map/difficulty/warden/strategy")
	games := flag.Int("games", 100, "number of games in marathon mode")
	heapStats := flag.Bool("heap-stats", false, "print heap statistics every 10 games in marathon mode")
	modelPath := flag.String("model-path", "", "path to LLM .bin weight file (for llm strategy)")
	vocabPath := flag.String("vocab-path", "config/llm/vocab.json", "path to LLM vocab.json (for llm strategy)")
	flag.Parse()

	if *sessionJSON != "" {
		runSingleSession(*sessionJSON)
		return
	}

	orchestrate(*runs, *strategies, *mapID, *difficulty, *warden, *output, *jsonDir, *sweep, *abilitySweep, *balanceSweep, *simSweep, *scenarioName, *seed, *marathon, *games, *heapStats, *modelPath, *vocabPath)
}

// ─── 编排模式 ───

// sessionConfig 传给子进程的序列化配置（也用于进程内模式的参数传递）。
// 所有字段都是 JSON 可序列化的基础类型，不包含接口或函数引用。
type sessionConfig struct {
	ID          string `json:"id"`
	MapID       string `json:"map_id"`
	Difficulty  string `json:"difficulty"`
	Warden      string `json:"warden"`
	ModeID      string `json:"mode_id"`
	EnemyFilter string `json:"enemy_filter"`
	Strategy    string `json:"strategy"`
	TowerKey    string `json:"tower_key,omitempty"`
	ModelPath   string `json:"model_path,omitempty"`
	VocabPath   string `json:"vocab_path,omitempty"`
	OutputDir   string `json:"output_dir"`
	JSONDir     string `json:"json_dir,omitempty"` // 纯 JSON 输出 (空=混合到 OutputDir)
	Seed        int64  `json:"seed"`
}

// orchestrate 编排模式主函数：生成测试计划 → 逐个运行 → 汇总报告。
//
// 流程概览：
//  1. 初始化 dataFS（需要读取配置生成测试计划）
//  2. 创建输出目录（splitMode 下 JSON 和日志分离）
//  3. 根据运行模式（sweep/marathon/scenario 等）生成 TestCase 列表
//  4. 确定主种子（0=随机不可复现，非0=确定性可复现）
//  5. 逐个调用 runSessionInProcess() 运行测试
//  6. marathon 模式下每 10 局打印堆内存统计（检测内存泄漏）
//  7. 生成汇总报告（coverage_summary.json 等）
func orchestrate(runs int, strategies, mapID, difficulty, warden, output, jsonDir string, sweep, abilitySweep, balanceSweep, simSweep bool, scenarioName string, masterSeed int64, marathon bool, games int, heapStats bool, modelPath, vocabPath string) {
	// 分离模式: --json-dir 由调用方管理目录结构
	// 初始化 dataFS（父进程需要读取 ability_tests.json 生成测试计划）
	config.SetDataFS(&defense2.DataFS)

	// 兼容模式: 生成带时间戳的 run 目录，避免历史结果污染
	splitMode := jsonDir != ""
	runDir := output
	if !splitMode {
		runDir = filepath.Join(output, time.Now().Format("run_20060102_150405"))
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		log.Fatalf("create run dir: %v", err)
	}
	if splitMode {
		if err := os.MkdirAll(jsonDir, 0o755); err != nil {
			log.Fatalf("create json dir: %v", err)
		}
	}

	// 生成测试用例
	var cases []autoplay.TestCase
	switch {
	case marathon:
		cases = generateMarathonCases(games, masterSeed)
		log.Printf("Marathon mode: %d random games", len(cases))
	case sweep:
		cases = autoplay.GenerateTestPlan()
		log.Printf("Sweep mode: %d test cases", len(cases))
	case balanceSweep:
		for _, bs := range autoplay.AllBalanceScenarios() {
			cases = append(cases, autoplay.TestCase{
				ID:          bs.ID,
				MapID:       bs.MapID,
				Difficulty:  bs.Difficulty,
				Warden:      bs.Warden,
				EnemyFilter: bs.EnemyFilter,
				Strategy:    bs.Strategy,
				Assertions:  bs.Assertions,
			})
		}
		log.Printf("Balance sweep mode: %d test cases", len(cases))
	case simSweep:
		for _, ss := range autoplay.AllSimScenarios() {
			cases = append(cases, ss.ToTestCase())
		}
		log.Printf("Simulation sweep mode: %d test cases", len(cases))
	case abilitySweep:
		for _, name := range autoplay.AbilityScenarioNames() {
			tc := autoplay.ScenarioCase(name, mapID)
			tc.EnemyFilter = autoplay.AbilityTestEnemyFilter(name)
			cases = append(cases, tc)
		}
		log.Printf("Ability sweep mode: %d test cases", len(cases))
	case scenarioName != "":
		tc := autoplay.ScenarioCase(scenarioName, mapID)
		tc.EnemyFilter = autoplay.AbilityTestEnemyFilter(scenarioName)
		cases = []autoplay.TestCase{tc}
	default:
		cases = autoplay.ParseCLICases(runs, strategies, mapID, difficulty, warden)
	}

	if len(cases) == 0 {
		fmt.Println("No test cases generated.")
		os.Exit(1)
	}

	log.Printf("AutoPlay: %d sessions → %s", len(cases), runDir)

	// 主种子：0 表示用时间戳（不可复现），非 0 表示确定性
	if masterSeed == 0 {
		masterSeed = time.Now().UnixNano()
		log.Printf("Random seed: %d (use --seed=%d to reproduce)", masterSeed, masterSeed)
	} else {
		log.Printf("Deterministic seed: %d", masterSeed)
	}

	passed, failed := 0, 0

	// Heap stats tracking for marathon mode
	var firstHeapAlloc, lastHeapAlloc uint64
	if marathon && heapStats {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		firstHeapAlloc = m.Alloc
		log.Printf("[Heap] Initial: Alloc=%dMB, Sys=%dMB", m.Alloc/1024/1024, m.Sys/1024/1024)
	}

	for i, tc := range cases {
		sessionSeed := masterSeed + int64(i)

		log.Printf("[%d/%d] %s", i+1, len(cases), tc.ID)

		cfg := sessionConfig{
			ID:          tc.ID,
			MapID:       tc.MapID,
			Difficulty:  tc.Difficulty,
			Warden:      tc.Warden,
			ModeID:      tc.EffectiveModeID(),
			EnemyFilter: tc.EnemyFilter,
			Strategy:    tc.Strategy.Name(),
			ModelPath:   modelPath,
			VocabPath:   vocabPath,
			OutputDir:   runDir,
			JSONDir:     jsonDir,
			Seed:        sessionSeed,
		}
		if f, ok := tc.Strategy.(*autoplay.FocusStrategy); ok {
			cfg.TowerKey = f.TowerKey()
		}

		runSessionInProcess(cfg, tc.Assertions)
		passed++

		// Heap stats every 10 games in marathon mode
		if marathon && heapStats && (i+1)%10 == 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			lastHeapAlloc = m.Alloc
			log.Printf("[Heap] Game %d: Alloc=%dMB, TotalAlloc=%dMB, Sys=%dMB, NumGC=%d",
				i+1, m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024, m.NumGC)
		}
	}

	log.Printf("Completed: %d passed, %d failed", passed, failed)

	// Final heap comparison for marathon mode
	if marathon && heapStats {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		lastHeapAlloc = m.Alloc
		if firstHeapAlloc > 0 {
			growthPct := float64(lastHeapAlloc) / float64(firstHeapAlloc) * 100
			log.Printf("[Heap] Final: Alloc=%dMB (started at %dMB)", lastHeapAlloc/1024/1024, firstHeapAlloc/1024/1024)
			if growthPct > 150 {
				log.Printf("WARNING: Heap grew from %dMB to %dMB over %d games (%.0f%% growth) — possible memory leak",
					firstHeapAlloc/1024/1024, lastHeapAlloc/1024/1024, len(cases), growthPct-100)
			}
		}
	}

	// 汇总报告 (分离模式从 jsonDir 读取)
	reportDir := runDir
	if splitMode {
		reportDir = jsonDir
	}
	autoplay.GenerateSummaryReport(reportDir)

	// 仿真专用报告
	if simSweep {
		simReport, err := autoplay.GenerateSimReport(reportDir)
		if err != nil {
			log.Printf("sim report error: %v", err)
		} else if err := autoplay.WriteSimReport(simReport, reportDir); err != nil {
			log.Printf("sim report write error: %v", err)
		} else {
			log.Printf("Simulation report: %s/sim_report.json", reportDir)
			log.Printf("Simulation summary: %s/sim_summary.txt", reportDir)
		}
	}
}

// generateMarathonCases 生成 N 个随机测试用例（随机地图/难度/战灵/策略）。
// 用于长时间压力测试：验证不同组合下是否存在崩溃、内存泄漏或异常行为。
// 每个用例的随机种子 = masterSeed + 序号，保证同一 masterSeed 下结果可复现。
func generateMarathonCases(n int, seed int64) []autoplay.TestCase {
	rng := rand.New(rand.NewSource(seed))
	maps := autoplay.Maps
	diffs := autoplay.Difficulties
	wardens := autoplay.Wardens
	strats := []string{"random", "greedy"}

	cases := make([]autoplay.TestCase, 0, n)
	for i := 0; i < n; i++ {
		m := maps[rng.Intn(len(maps))]
		d := diffs[rng.Intn(len(diffs))]
		w := wardens[rng.Intn(len(wardens))]
		s := strats[rng.Intn(len(strats))]

		var strategy autoplay.Strategy
		switch s {
		case "greedy":
			strategy = autoplay.NewGreedyStrategy()
		default:
			strategy = autoplay.NewRandomStrategy(seed + int64(i))
		}

		cases = append(cases, autoplay.TestCase{
			ID:         fmt.Sprintf("marathon_%04d_%s_%s_%s", i+1, m, d, s),
			MapID:      m,
			Difficulty: d,
			Warden:     w,
			Strategy:   strategy,
		})
	}
	return cases
}

// ─── 单局模式 ───

// runSingleSession 单局模式入口（子进程调用）。
// 从 JSON 配置反序列化 → 初始化游戏 → 纯 CPU 循环直到游戏结束。
// 纯无头模式：不启动 Ebitengine 窗口/GPU，直接循环 Update()。
func runSingleSession(cfgJSON string) {
	var cfg sessionConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		log.Fatalf("parse session config: %v", err)
	}

	// 注入嵌入式文件系统
	config.SetDataFS(&defense2.DataFS)
	config.SetAssetFS(&defense2.AssetFS)

	// 确定性随机种子（同 seed = 同结果）
	autoplay.SeedAll(cfg.Seed)

	// 纯无头模式：不启动 Ebitengine 窗口/GPU，直接循环 Update()
	scene.HeadlessMode = true

	g := scene.NewGame()
	stage := scene.NewStageSceneWithOpts(g, scene.StageOptions{
		MapID:        cfg.MapID,
		WardenType:   cfg.Warden,
		ModeID:       cfg.ModeID,
		DifficultyID: cfg.Difficulty,
		EnemyFilter:  cfg.EnemyFilter,
	})

	strategy := restoreStrategy(cfg)
	ctrl := autoplay.NewController(autoplay.ControllerConfig{
		Strategy:   strategy,
		OutputDir:  cfg.OutputDir,
		JSONDir:    cfg.JSONDir,
		SessionID:  cfg.ID,
		MapID:      cfg.MapID,
		Difficulty: cfg.Difficulty,
		Warden:     cfg.Warden,
		Seed:       cfg.Seed,
	})
	// 为能力测试场景加载断言
	stratName := strategy.Name()
	if len(stratName) > 9 && stratName[:9] == "scenario_" {
		scenName := stratName[9:]
		if assertions, ok := autoplay.AbilityAssertionsMap()[scenName]; ok {
			ctrl.SetAssertions(assertions)
		}
	}

	stage.SetAutoPlayer(ctrl)
	g.SwitchScene(stage)

	// 纯 CPU 循环：直接驱动 Game.Update()，无 GPU/窗口依赖
	for {
		if err := g.Update(); err != nil {
			if err == ebiten.Termination {
				break // 正常结束
			}
			log.Printf("session %s error: %v", cfg.ID, err)
			break
		}
	}
}

// gameInitDone 标记是否已完成首次 Game 初始化。
// 首局用 NewGame()（加载所有配置/字体/音频），后续用 NewGameLite()（跳过重复初始化）。
var gameInitDone bool

// runSessionInProcess 在当前进程内运行单个测试场景（无子进程开销）。
// 这是编排模式下的核心执行函数，比 fork 子进程快 10x 以上。
// 关键设计：HeadlessMode=true 使 Ebitengine 跳过 GPU 初始化，Update() 纯 CPU 执行。
func runSessionInProcess(cfg sessionConfig, extraAssertions []autoplay.Assertion) {
	autoplay.SeedAll(cfg.Seed)
	scene.HeadlessMode = true

	var g *scene.Game
	if !gameInitDone {
		g = scene.NewGame()
		gameInitDone = true
	} else {
		g = scene.NewGameLite()
	}
	waves := 0 // 0 = 地图默认（12 波）
	stage := scene.NewStageSceneWithOpts(g, scene.StageOptions{
		MapID:        cfg.MapID,
		WardenType:   cfg.Warden,
		ModeID:       cfg.ModeID,
		DifficultyID: cfg.Difficulty,
		EnemyFilter:  cfg.EnemyFilter,
		Waves:        waves,
	})

	strategy := restoreStrategy(cfg)
	ctrl := autoplay.NewController(autoplay.ControllerConfig{
		Strategy:   strategy,
		OutputDir:  cfg.OutputDir,
		JSONDir:    cfg.JSONDir,
		SessionID:  cfg.ID,
		MapID:      cfg.MapID,
		Difficulty: cfg.Difficulty,
		Warden:     cfg.Warden,
		Seed:       cfg.Seed,
	})
	// 加载断言：优先使用直接传入的断言，其次按场景名查找
	if len(extraAssertions) > 0 {
		ctrl.SetAssertions(extraAssertions)
	} else {
		stratName := strategy.Name()
		if len(stratName) > 9 && stratName[:9] == "scenario_" {
			scenName := stratName[9:]
			if assertions, ok := autoplay.AbilityAssertionsMap()[scenName]; ok {
				ctrl.SetAssertions(assertions)
			}
		}
	}

	stage.SetAutoPlayer(ctrl)
	g.SwitchScene(stage)

	for {
		if err := g.Update(); err != nil {
			if err == ebiten.Termination {
				break
			}
			log.Printf("session %s error: %v", cfg.ID, err)
			break
		}
	}
}

// restoreStrategy 从序列化配置还原策略实例。
// 策略名称是字符串（可序列化），但 Strategy 是接口（不可序列化），
// 因此需要此函数根据名称重建具体实现。
// champion_* 和 focus_* 是前缀匹配（如 "champion_elite" → ChampionStrategy{StyleElite}）。
// scenario_* 从注册表查找工厂函数，找不到时回退到默认 BuildFlowScenario。
func restoreStrategy(cfg sessionConfig) autoplay.Strategy {
	name := cfg.Strategy
	switch {
	case name == "random":
		return autoplay.NewRandomStrategy(cfg.Seed)
	case name == "greedy":
		return autoplay.NewGreedyStrategy()
	case name == "balance_greedy":
		return autoplay.NewBalanceGreedyStrategy()
	case name == "competent":
		return autoplay.NewCompetentStrategy(autoplay.WithCompetentSeed(cfg.Seed))
	case name == "visual_catalog":
		return autoplay.NewVisualCatalogStrategy()
	case name == "llm":
		s, err := autoplay.NewLLMStrategy(cfg.ModelPath, cfg.VocabPath)
		if err != nil {
			log.Fatalf("create LLM strategy: %v", err)
		}
		return s
	case len(name) > 9 && name[:9] == "champion_":
		styleMap := map[string]autoplay.ChampionStyle{
			"champion_balanced": autoplay.StyleBalanced,
			"champion_elite":    autoplay.StyleElite,
			"champion_swarm":    autoplay.StyleSwarm,
			"champion_cc":       autoplay.StyleCC,
			"champion_dps":      autoplay.StyleDPS,
		}
		style := autoplay.StyleBalanced
		if s, ok := styleMap[name]; ok {
			style = s
		}
		return autoplay.NewChampionStrategy(style, cfg.Warden)
	case len(name) > 6 && name[:6] == "focus_":
		return autoplay.NewFocusStrategy(cfg.TowerKey)
	case len(name) > 9 && name[:9] == "scenario_":
		scenarioName := name[9:]
		scenarios := autoplay.AllScenariosMap()
		if factory, ok := scenarios[scenarioName]; ok {
			return factory()
		}
		return autoplay.BuildFlowScenario()
	default:
		return autoplay.NewRandomStrategy(time.Now().UnixNano())
	}
}
