// result.go — 结算场景。
// 游戏结束（胜利/失败）后显示统计信息，提供重玩或返回选关的选项。
package scene

import (
	"fmt"
	"image/color"

	"defense2/internal/core/game"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ResultData 结算数据。
type ResultData struct {
	MapID    string // 关卡 ID
	MapName  string // 关卡名称
	Won      bool   // 是否胜利
	Kills    int    // 击杀数
	Waves    int    // 通过波次数
	MaxWaves int    // 总波次数
	Gold     int    // 剩余金币
	Towers   int    // 放置的塔数
}

// ResultScene 结算场景。
type ResultScene struct {
	switcher Switcher   // 场景切换器
	data     ResultData // 结算数据
}

// NewResultScene 创建结算场景。
func NewResultScene(sw Switcher, data ResultData) *ResultScene {
	return &ResultScene{
		switcher: sw,
		data:     data,
	}
}

func (s *ResultScene) Update() error {
	// R: 重玩同一关
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		s.switcher.SwitchScene(NewStageSceneWithMap(s.switcher, s.data.MapID))
		return nil
	}
	// Enter/Click: 返回选关
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
		return nil
	}
	// ESC: 返回标题
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewTitleScene(s.switcher))
	}
	return nil
}

func (s *ResultScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 25, B: 35, A: 255})

	cx := game.ScreenWidth / 2
	d := s.data

	// 标题
	title := "DEFEAT"
	if d.Won {
		title = "VICTORY!"
	}
	ebitenutil.DebugPrintAt(screen, title, cx-25, 100)

	// 统计
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Map: %s", d.MapName), cx-60, 150)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Waves: %d/%d", d.Waves, d.MaxWaves), cx-60, 175)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Kills: %d", d.Kills), cx-60, 200)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Gold: %d", d.Gold), cx-60, 225)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Towers: %d", d.Towers), cx-60, 250)

	// 操作提示
	ebitenutil.DebugPrintAt(screen, "[R] Replay  [ENTER] Map Select  [ESC] Title", cx-130, 320)
}
