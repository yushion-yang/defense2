// select.go — 选关场景。
// 显示所有可用地图，高亮已解锁的关卡，点击或按数字键进入游戏。
package scene

import (
	"fmt"
	"image/color"

	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/persistence"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// 可选关卡列表（按顺序排列）。
var mapIDs = []string{
	"map_01", "map_02", "map_03", "map_04",
	"map_05", "map_06", "map_07", "map_08",
}

// SelectScene 选关场景。
type SelectScene struct {
	switcher    Switcher                     // 场景切换器
	progressMgr *persistence.ProgressManager // 进度管理器
	selected    int                          // 当前高亮的关卡索引
	mapNames    []string                     // 各关卡显示名称
}

// NewSelectScene 创建选关场景。
func NewSelectScene(sw Switcher) *SelectScene {
	store, _ := persistence.DefaultStorage()
	pm := persistence.NewProgressManager(store)

	// 加载各关卡名称
	names := make([]string, len(mapIDs))
	for i, id := range mapIDs {
		cfg, err := config.LoadMap(id)
		if err != nil {
			names[i] = id
		} else {
			names[i] = cfg.Name
		}
	}

	return &SelectScene{
		switcher:    sw,
		progressMgr: pm,
		mapNames:    names,
	}
}

func (s *SelectScene) Update() error {
	// 上下选择
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if s.selected > 0 {
			s.selected--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if s.selected < len(mapIDs)-1 {
			s.selected++
		}
	}

	// 数字键快速选择
	for i := 0; i < len(mapIDs) && i < 8; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			s.selected = i
		}
	}

	// Enter 或点击进入
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.tryEnterMap(s.selected)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		_, my := ebiten.CursorPosition()
		idx := (my - 80) / 40
		if idx >= 0 && idx < len(mapIDs) {
			s.selected = idx
			s.tryEnterMap(idx)
		}
	}

	// ESC 返回标题
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewTitleScene(s.switcher))
	}

	return nil
}

func (s *SelectScene) tryEnterMap(idx int) {
	if idx < 0 || idx >= len(mapIDs) {
		return
	}
	id := mapIDs[idx]
	if !s.progressMgr.IsMapUnlocked(id) {
		return
	}
	s.switcher.SwitchScene(NewStageSceneWithMap(s.switcher, id))
}

func (s *SelectScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 30, B: 20, A: 255})

	ebitenutil.DebugPrintAt(screen, "SELECT MAP (Up/Down + Enter)", game.ScreenWidth/2-80, 30)

	for i, id := range mapIDs {
		y := 80 + i*40
		x := game.ScreenWidth/2 - 150

		unlocked := s.progressMgr.IsMapUnlocked(id)
		highScore := s.progressMgr.Progress().HighScores[id]

		// 选中高亮
		if i == s.selected {
			vector.DrawFilledRect(screen, float32(x-4), float32(y-2), 320, 30,
				color.RGBA{R: 60, G: 80, B: 60, A: 200}, false)
		}

		// 关卡名称
		label := fmt.Sprintf("%d. %s", i+1, s.mapNames[i])
		if !unlocked {
			label += " [LOCKED]"
		} else if highScore > 0 {
			label += fmt.Sprintf("  (Best: %d kills)", highScore)
		}

		clr := color.RGBA{R: 200, G: 200, B: 200, A: 255}
		if !unlocked {
			clr = color.RGBA{R: 100, G: 100, B: 100, A: 150}
		}
		_ = clr // DebugPrint 不支持颜色，用标记代替
		ebitenutil.DebugPrintAt(screen, label, x, y+4)
	}

	ebitenutil.DebugPrintAt(screen, "ESC: Back to title", 20, game.ScreenHeight-20)
}
