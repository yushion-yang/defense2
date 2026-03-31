// main.go — AutoPlay 自动对局入口。
// 独立二进制，最小化窗口模式运行自动测试。
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	defense2 "defense2"
	"defense2/internal/autoplay"
	"defense2/internal/config"
	"defense2/internal/scene"
)

func main() {
	runs := flag.Int("runs", 1, "number of sessions per strategy")
	strategies := flag.String("strategies", "random,greedy", "comma-separated strategy names")
	mapID := flag.String("map", "map_01", "map ID")
	difficulty := flag.String("difficulty", "normal", "difficulty ID")
	warden := flag.String("warden", "prince", "warden type")
	output := flag.String("output", "./autoplay-results", "output directory")
	sweep := flag.Bool("sweep", false, "run full pairwise sweep (~60 cases)")
	scenarioName := flag.String("scenario", "", "run single scenario by name")
	flag.Parse()

	// 注入嵌入式文件系统
	config.SetDataFS(&defense2.DataFS)
	config.SetAssetFS(&defense2.AssetFS)

	// 生成测试用例
	var cases []autoplay.TestCase
	switch {
	case *sweep:
		cases = autoplay.GenerateTestPlan()
		log.Printf("Sweep mode: %d test cases", len(cases))
	case *scenarioName != "":
		cases = []autoplay.TestCase{autoplay.ScenarioCase(*scenarioName, *mapID)}
	default:
		cases = autoplay.ParseCLICases(*runs, *strategies, *mapID, *difficulty, *warden)
	}

	if len(cases) == 0 {
		fmt.Println("No test cases generated.")
		os.Exit(1)
	}

	log.Printf("AutoPlay: %d sessions to run", len(cases))

	// 逐个运行（Ebitengine 单窗口限制）
	for i, tc := range cases {
		log.Printf("[%d/%d] Running session: %s (strategy=%s map=%s diff=%s warden=%s)",
			i+1, len(cases), tc.ID, tc.Strategy.Name(), tc.MapID, tc.Difficulty, tc.Warden)
		runSession(tc, *output)
	}

	// 生成汇总报告
	autoplay.GenerateSummaryReport(*output)
}

func runSession(tc autoplay.TestCase, outputDir string) {
	// 无头优化：1x1 最小化窗口 + 高 TPS（Draw 被 StageScene 跳过，GPU≈0）
	ebiten.SetWindowSize(1, 1)
	ebiten.SetWindowTitle("AutoPlay")
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(600) // 逻辑帧 600/s，实际游戏速度由 gameSpeed 控制

	// 创建 Game
	g := scene.NewGame()

	// 创建 StageScene
	modeID := tc.EffectiveModeID()
	opts := scene.StageOptions{
		MapID:        tc.MapID,
		WardenType:   tc.Warden,
		ModeID:       modeID,
		DifficultyID: tc.Difficulty,
		EnemyFilter:  tc.EnemyFilter,
	}
	stage := scene.NewStageSceneWithOpts(g, opts)

	// 创建控制器
	ctrl := autoplay.NewController(autoplay.ControllerConfig{
		Strategy:   tc.Strategy,
		OutputDir:  outputDir,
		SessionID:  tc.ID,
		MapID:      tc.MapID,
		Difficulty: tc.Difficulty,
		Warden:     tc.Warden,
	})
	stage.SetAutoPlayer(ctrl)

	// 切换到 stage 场景
	g.SwitchScene(stage)

	// 运行游戏循环（Termination 会让 RunGame 正常返回 nil）
	if err := ebiten.RunGame(g); err != nil {
		log.Printf("session %s error: %v", tc.ID, err)
	}
}
