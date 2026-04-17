// train.go — AI 学习系统离线训练管线。
//
// 通过 --learn 标志启动，在无头模式下批量运行合作局，
// 利用 AIPlayer 的在线学习系统收集训练后的权重，
// 最终取胜局权重平均值导出到 config/ai/weights.json。
//
// 典型用法：
//
//	go run cmd/autoplay/main.go --learn --learn-games 50 --map map_co01
//	go run cmd/autoplay/main.go --learn --learn-games 100 --seed 42
package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	defense2 "defense2"
	"defense2/internal/autoplay"
	"defense2/internal/config"
	"defense2/internal/core/aiplayer/learning"
	"defense2/internal/scene"
)

// trainConfig 训练管线配置。
type trainConfig struct {
	Games      int     // 训练局数
	MapID      string  // 地图 ID
	Difficulty string  // 难度
	Warden     string  // 战灵
	Seed       int64   // 主种子（0=随机）
	OutputPath string  // 权重导出路径
	HPScale    float64 // HP 缩放覆盖（0=使用难度默认值）
}

// runTraining 离线训练管线主函数。
//
// 流程：
//  1. 运行 N 局合作模式游戏，AIPlayer 启用在线学习
//  2. 每局结束后提取 AI 的学习模型权重
//  3. 区分胜局/败局，分别收集
//  4. 用胜局权重的平均值作为最终模型（无胜局则用全部局的平均）
//  5. 导出到 config/ai/weights.json
func runTraining(cfg trainConfig) {
	// 初始化嵌入式文件系统
	config.SetDataFS(&defense2.DataFS)
	config.SetAssetFS(&defense2.AssetFS)

	scene.HeadlessMode = true

	if cfg.Seed == 0 {
		cfg.Seed = time.Now().UnixNano()
		log.Printf("[Train] Random seed: %d", cfg.Seed)
	} else {
		log.Printf("[Train] Deterministic seed: %d", cfg.Seed)
	}

	if cfg.OutputPath == "" {
		cfg.OutputPath = "config/ai/weights.json"
	}
	if cfg.MapID == "" {
		cfg.MapID = "map_c01" // 默认使用经典模式地图（确定性，零随机）
	}
	if cfg.Difficulty == "" {
		cfg.Difficulty = "normal"
	}
	// 经典模式无战灵；其他模式默认 chain
	if cfg.Warden == "" && !isClassicMap(cfg.MapID) {
		cfg.Warden = "chain"
	}

	coopMaps := []string{"map_co01", "map_co02", "map_co03"}

	log.Printf("[Train] Starting %d games (map=%s, difficulty=%s, warden=%s)",
		cfg.Games, cfg.MapID, cfg.Difficulty, cfg.Warden)

	var winModels, allModels []*learning.Model
	wins, losses := 0, 0

	for i := 0; i < cfg.Games; i++ {
		sessionSeed := cfg.Seed + int64(i)
		autoplay.SeedAll(sessionSeed)

		// 每局随机选一个合作地图（增加训练多样性）
		rng := rand.New(rand.NewSource(sessionSeed))
		mapID := cfg.MapID
		if cfg.MapID == "random" {
			mapID = coopMaps[rng.Intn(len(coopMaps))]
		}

		// 检查地图是否支持 coop（如果是 coop 地图则设置人数）
		coopPlayerCount := 0
		if m, err := config.LoadMap(mapID); err == nil && m.Coop != nil {
			coopPlayerCount = m.Coop.PlayerCount
		}
		_ = coopPlayerCount // 训练模式暂不使用 coop 分区

		var g *scene.Game
		if !gameInitDone {
			g = scene.NewGame()
			gameInitDone = true
		} else {
			g = scene.NewGameLite()
		}

		// 创建带学习模式的 Stage
		// 训练模式使用经典战役地图：确定性塔/波次，零随机性
		// 策略成败完全取决于决策质量
		stage := scene.NewStageSceneWithOpts(g, scene.StageOptions{
			MapID:           mapID,
			WardenType:      "",
			ModeID:          "classic",
			DifficultyID:    cfg.Difficulty,
			AIEnabled:       true,
			LearningEnabled: true,
			HPScaleOverride: cfg.HPScale,
		})

		// 使用 competent 策略驱动全图（AI 学习系统通过 autoplay 挂钩观察并学习）
		strategy := autoplay.NewCompetentStrategy(autoplay.WithCompetentSeed(sessionSeed))
		ctrl := autoplay.NewController(autoplay.ControllerConfig{
			Strategy:   strategy,
			OutputDir:  os.TempDir(),
			SessionID:  fmt.Sprintf("train_%04d", i+1),
			MapID:      mapID,
			Difficulty: cfg.Difficulty,
			Warden:     cfg.Warden,
			Seed:       sessionSeed,
		})

		stage.SetAutoPlayer(ctrl)
		g.SwitchScene(stage)

		// 运行游戏直到结束
		for {
			if err := g.Update(); err != nil {
				if err == ebiten.Termination {
					break
				}
				log.Printf("[Train] Game %d error: %v", i+1, err)
				break
			}
		}

		// 提取学习模型
		models := stage.LearningModels()
		won := ctrl.Won()
		if won {
			wins++
			for _, m := range models {
				winModels = append(winModels, m.Clone())
			}
		} else {
			losses++
		}
		for _, m := range models {
			allModels = append(allModels, m.Clone())
		}

		status := "LOSS"
		if won {
			status = "WIN"
		}
		modelInfo := ""
		if len(models) > 0 {
			modelInfo = fmt.Sprintf(" (v%d, ep%d)", models[0].Version, models[0].Episodes)
		}
		log.Printf("[Train] Game %d/%d: %s%s [W:%d L:%d]",
			i+1, cfg.Games, status, modelInfo, wins, losses)
	}

	// 选择最终模型：优先用胜局权重平均，无胜局用全部权重平均
	var finalModel *learning.Model
	if len(winModels) > 0 {
		finalModel = learning.AverageModels(winModels)
		log.Printf("[Train] Averaging %d winning models", len(winModels))
	} else if len(allModels) > 0 {
		finalModel = learning.AverageModels(allModels)
		log.Printf("[Train] No wins — averaging all %d models", len(allModels))
	} else {
		finalModel = learning.DefaultModel()
		log.Printf("[Train] No models collected — using defaults")
	}

	// 导出权重
	data, err := learning.MarshalModel(finalModel)
	if err != nil {
		log.Fatalf("[Train] Marshal error: %v", err)
	}

	// 确保输出目录存在
	dir := filepath.Dir(cfg.OutputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("[Train] Create dir error: %v", err)
	}

	if err := os.WriteFile(cfg.OutputPath, data, 0o644); err != nil {
		log.Fatalf("[Train] Write error: %v", err)
	}

	log.Printf("[Train] Complete: %d wins / %d games (%.0f%% win rate)",
		wins, cfg.Games, float64(wins)/float64(cfg.Games)*100)
	log.Printf("[Train] Exported to %s (v%d, ep%d)",
		cfg.OutputPath, finalModel.Version, finalModel.Episodes)
}

// isClassicMap 判断是否是经典模式地图（map_cXX 前缀）。
func isClassicMap(mapID string) bool {
	return len(mapID) >= 6 && mapID[:5] == "map_c" && mapID[5] >= '0' && mapID[5] <= '9'
}
