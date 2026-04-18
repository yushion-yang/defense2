// train.go — AI 自我对弈训练管线。
//
// 通过 --learn 标志启动，在无头模式下使用 NeuralStrategy 进行自我对弈训练。
// 核心闭环：NeuralStrategy(model) → play → trainer 更新权重 → 下一局更好。
//
// 与旧版的关键区别：
//   - 旧版：CompetentStrategy（硬编码）玩 → AIPlayer 旁观学习 → 权重无法反哺策略
//   - 新版：NeuralStrategy（神经网络）自己玩 → 自己学 → 权重直接驱动下一局决策
//
// 典型用法：
//
//	go run cmd/autoplay/main.go --learn --learn-games 50 --map map_c01
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

// runTraining 自我对弈训练管线主函数。
//
// 流程：
//  1. 加载初始模型权重（从嵌入式 JSON 或 DefaultModel）
//  2. 每局创建 NeuralStrategy + Trainer，用当前模型权重驱动决策
//  3. Trainer 在波次/游戏结束时通过 reward 反向传播更新权重
//  4. 每局结束后快照模型，区分胜局/败局收集
//  5. 最终取胜局权重平均（无胜局取全部平均），导出到 config/ai/weights.json
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

	log.Printf("[Train] Starting %d self-play games (map=%s, difficulty=%s, strategy=neural)",
		cfg.Games, cfg.MapID, cfg.Difficulty)

	// 加载初始模型（从嵌入式 JSON，加载失败用 DefaultModel）
	model := learning.LoadFromFS(config.GetDataFS())
	log.Printf("[Train] Initial model: v%d, ep%d, neural=%v",
		model.Version, model.Episodes, model.UseNeural())

	var winModels, allModels []*learning.Model
	wins, losses := 0, 0
	var g *scene.Game // 复用同一个 Game 实例避免 GPU 资源泄漏

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

		// 复用同一个 Game 实例，只切换 Scene — 避免 Ebitengine GPU 资源泄漏
		if !gameInitDone {
			g = scene.NewGame()
			gameInitDone = true
		}

		// 创建 Stage — 不启用 AIPlayer（NeuralStrategy 直接控制，避免 AIPlayer 偷选战灵）
		modeID := "classic"
		if !isClassicMap(mapID) {
			modeID = "casual"
		}
		stage := scene.NewStageSceneWithOpts(g, scene.StageOptions{
			MapID:           mapID,
			WardenType:      "",
			ModeID:          modeID,
			DifficultyID:    cfg.Difficulty,
			HPScaleOverride: cfg.HPScale,
		})

		// 创建 Trainer（在线学习，直接更新 model 权重）
		trainer := learning.NewTrainer(learning.TrainerConfig{
			Model:   model,
			Enabled: true,
		})

		// 使用 NeuralStrategy 自我对弈：模型既驱动决策又接收 reward
		strategy := autoplay.NewNeuralStrategy(model,
			autoplay.WithNeuralSeed(sessionSeed),
			autoplay.WithNeuralTrainer(trainer),
		)
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

		// 通知训练器游戏结束（触发全局 reward 微调）。
		// 波次级 reward 已在 NeuralStrategy.onWaveChange 中处理，
		// 这里补充游戏级全局 reward（赢/输加成）。
		won := ctrl.Won()
		trainer.OnGameEnd(learning.GameResult{
			Won: won,
		})

		// 快照当前模型权重（Trainer 已在线更新）
		snapshot := model.Clone()
		if won {
			wins++
			winModels = append(winModels, snapshot)
		} else {
			losses++
		}
		allModels = append(allModels, snapshot)

		// 同时收集 AIPlayer 的学习模型（辅助信号源）
		aiModels := stage.LearningModels()
		for _, m := range aiModels {
			allModels = append(allModels, m.Clone())
			if won {
				winModels = append(winModels, m.Clone())
			}
		}

		// 重置 Trainer 状态准备下一局（模型权重保留）
		trainer.Reset()

		log.Printf("[Train] Game %d/%d: %s (v%d, ep%d) [W:%d L:%d]",
			i+1, cfg.Games, statusStr(won), model.Version, model.Episodes, wins, losses)
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

// statusStr 返回胜负状态字符串。
func statusStr(won bool) string {
	if won {
		return "WIN"
	}
	return "LOSS"
}

// isClassicMap 判断是否是经典模式地图（map_cXX 前缀）。
func isClassicMap(mapID string) bool {
	return len(mapID) >= 6 && mapID[:5] == "map_c" && mapID[5] >= '0' && mapID[5] <= '9'
}
