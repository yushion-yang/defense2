// main.go — AutoPlay 自动对局入口。
// 支持两种运行模式：
//   - 单局模式 (--session-json): 运行单个 TestCase，由父进程调度
//   - 编排模式 (默认): 生成测试计划，为每局 fork 子进程
//
// Ebitengine 的 RunGame() 只能调用一次，因此多局必须用子进程隔离。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	defense2 "defense2"
	"defense2/internal/autoplay"
	"defense2/internal/config"
	"defense2/internal/core/game"
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
	modelPath := flag.String("model-path", "", "path to LLM .bin weight file (for llm strategy)")
	vocabPath := flag.String("vocab-path", "config/llm/vocab.json", "path to LLM vocab.json (for llm strategy)")
	flag.Parse()

	if *sessionJSON != "" {
		runSingleSession(*sessionJSON)
		return
	}

	orchestrate(*runs, *strategies, *mapID, *difficulty, *warden, *output, *jsonDir, *sweep, *abilitySweep, *scenarioName, *seed, *modelPath, *vocabPath)
}

// ─── 编排模式 ───

// sessionConfig 传给子进程的序列化配置。
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

func orchestrate(runs int, strategies, mapID, difficulty, warden, output, jsonDir string, sweep, abilitySweep bool, scenarioName string, masterSeed int64, modelPath, vocabPath string) {
	// 分离模式: --json-dir 由调用方管理目录结构
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
	case sweep:
		cases = autoplay.GenerateTestPlan()
		log.Printf("Sweep mode: %d test cases", len(cases))
	case abilitySweep:
		for _, name := range autoplay.AbilityScenarioNames() {
			cases = append(cases, autoplay.ScenarioCase(name, mapID))
		}
		log.Printf("Ability sweep mode: %d test cases", len(cases))
	case scenarioName != "":
		cases = []autoplay.TestCase{autoplay.ScenarioCase(scenarioName, mapID)}
	default:
		cases = autoplay.ParseCLICases(runs, strategies, mapID, difficulty, warden)
	}

	if len(cases) == 0 {
		fmt.Println("No test cases generated.")
		os.Exit(1)
	}

	log.Printf("AutoPlay: %d sessions → %s", len(cases), runDir)

	// 找到自身可执行文件路径
	self, err := os.Executable()
	if err != nil {
		log.Fatalf("find executable: %v", err)
	}

	// 主种子：0 表示用时间戳（不可复现），非 0 表示确定性
	if masterSeed == 0 {
		masterSeed = time.Now().UnixNano()
		log.Printf("Random seed: %d (use --seed=%d to reproduce)", masterSeed, masterSeed)
	} else {
		log.Printf("Deterministic seed: %d", masterSeed)
	}

	passed, failed := 0, 0
	for i, tc := range cases {
		// 每局派生一个确定性子种子：masterSeed + 序号
		sessionSeed := masterSeed + int64(i)

		log.Printf("[%d/%d] %s (strategy=%s map=%s diff=%s seed=%d)",
			i+1, len(cases), tc.ID, tc.Strategy.Name(), tc.MapID, tc.Difficulty, sessionSeed)

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
		// 特殊策略参数
		if f, ok := tc.Strategy.(*autoplay.FocusStrategy); ok {
			cfg.TowerKey = f.TowerKey()
		}

		cfgJSON, _ := json.Marshal(cfg)
		cmd := exec.Command(self, "--session-json", string(cfgJSON))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Printf("  FAIL: %v", err)
			failed++
		} else {
			passed++
		}
	}

	log.Printf("Completed: %d passed, %d failed", passed, failed)

	// 汇总报告 (分离模式从 jsonDir 读取)
	reportDir := runDir
	if splitMode {
		reportDir = jsonDir
	}
	autoplay.GenerateSummaryReport(reportDir)
}

// ─── 单局模式 ───

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

	// Turbo 模式: 跳过音效 + 每帧跑数千 tick（纯无头，无渲染）
	scene.HeadlessMode = true
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("AutoPlay: " + cfg.ID)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(ebiten.SyncWithFPS) // Update:Draw = 1:1，turbo 循环在 Update 内加速
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetRunnableOnUnfocused(true)

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

	if err := ebiten.RunGame(g); err != nil {
		log.Printf("session %s error: %v", cfg.ID, err)
	}
}

// restoreStrategy 从序列化配置还原策略实例。
func restoreStrategy(cfg sessionConfig) autoplay.Strategy {
	name := cfg.Strategy
	switch {
	case name == "random":
		return autoplay.NewRandomStrategy(cfg.Seed)
	case name == "greedy":
		return autoplay.NewGreedyStrategy()
	case name == "visual_catalog":
		return autoplay.NewVisualCatalogStrategy()
	case name == "llm":
		s, err := autoplay.NewLLMStrategy(cfg.ModelPath, cfg.VocabPath)
		if err != nil {
			log.Fatalf("create LLM strategy: %v", err)
		}
		return s
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
