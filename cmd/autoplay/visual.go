//go:build !unittest

// visual.go — 可视化自动对局模式。
// 用 --visual 启动，开窗口跑一局让用户亲眼看 AI 打游戏。
// 参数和训练模式完全一致（经典模式 + competent 策略）。
package main

import (
	"log"
	"math/rand"
	"time"

	defense2 "defense2"
	"defense2/internal/autoplay"
	"defense2/internal/config"
	"defense2/internal/scene"

	"github.com/hajimehoshi/ebiten/v2"
)

func runVisualGame(mapID, difficulty, warden string, seed int64, hpScale float64) {
	config.SetDataFS(&defense2.DataFS)
	config.SetAssetFS(&defense2.AssetFS)

	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	autoplay.SeedAll(seed)
	log.Printf("[Visual] seed=%d map=%s difficulty=%s warden=%s hpScale=%.1f", seed, mapID, difficulty, warden, hpScale)

	// 确定模式
	modeID := "classic"
	if mapID == "" {
		mapID = "map_c01"
	}
	// 非经典地图用 casual 模式
	if len(mapID) < 6 || mapID[4] != 'c' {
		modeID = "casual"
	}
	if warden == "" {
		warden = "chain"
	}

	g := scene.NewGame()

	stage := scene.NewStageSceneWithOpts(g, scene.StageOptions{
		MapID:           mapID,
		WardenType:      warden,
		ModeID:          modeID,
		DifficultyID:    difficulty,
		HPScaleOverride: hpScale,
		VisualAutoPlay:  true,
	})

	rng := rand.New(rand.NewSource(seed))
	_ = rng
	strategy := autoplay.NewCompetentStrategy(autoplay.WithCompetentSeed(seed))
	ctrl := autoplay.NewController(autoplay.ControllerConfig{
		Strategy:   strategy,
		OutputDir:  ".",
		SessionID:  "visual",
		MapID:      mapID,
		Difficulty: difficulty,
		Warden:     warden,
		Seed:       seed,
	})

	stage.SetAutoPlayer(ctrl)
	g.SwitchScene(stage)

	log.Printf("[Visual] Starting visual game (close window to exit)")

	ebiten.SetWindowSize(1200, 540)
	ebiten.SetWindowTitle("AI Visual Play")
	if err := ebiten.RunGame(g); err != nil {
		log.Printf("[Visual] %v", err)
	}
}
