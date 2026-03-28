// result.go — 结算场景。
// 游戏结束（胜利/失败）后显示统计信息，提供重玩或返回选关的选项。
package scene

import (
	"fmt"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ResultData 结算数据。
type ResultData struct {
	MapID      string // 关卡 ID
	MapName    string // 关卡名称
	Won        bool   // 是否胜利
	Kills      int    // 击杀数
	Waves      int    // 通过波次数
	MaxWaves   int    // 总波次数
	Gold       int    // 剩余金币
	Towers     int    // 放置的塔数
	WardenType string // 使用的战灵类型
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
		s.switcher.SwitchScene(NewStageSceneWithOptions(s.switcher, s.data.MapID, s.data.WardenType))
		return nil
	}
	// Enter/Click: 返回选关
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
		return nil
	}
	// ESC: 返回选关
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
	}
	return nil
}

func (s *ResultScene) Draw(screen *ebiten.Image) {
	screen.Fill(theme.ResultBg)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	cx := float64(game.ScreenWidth) / 2
	d := s.data

	// ── 标题 ──
	titleY := 100.0
	if d.Won {
		fm.DrawCenteredText(screen, "VICTORY!", cx, titleY, theme.FontResultTitle, theme.HUDVictoryColor)
	} else {
		fm.DrawCenteredText(screen, "DEFEAT", cx, titleY, theme.FontResultTitle, theme.HUDDefeatColor)
	}

	// ── 统计面板 ──
	const (
		panelW   = float32(500)
		panelR   = float32(16)
		padX     = 24.0
		padY     = 16.0
		lineH    = 24.0
		numLines = 5
	)
	panelH := float32(padY*2 + lineH*numLines)
	panelX := float32(cx) - panelW/2
	panelTopY := float32(160)

	draw.RoundRect(screen, panelX, panelTopY, panelW, panelH, panelR, theme.ResultStatsBg)

	// 统计行
	textX := float64(panelX) + padX
	textY := float64(panelTopY) + padY
	stats := []string{
		fmt.Sprintf("地图: %s", d.MapName),
		fmt.Sprintf("波次: %d / %d", d.Waves, d.MaxWaves),
		fmt.Sprintf("击杀: %d", d.Kills),
		fmt.Sprintf("金币: %d", d.Gold),
		fmt.Sprintf("塔数: %d", d.Towers),
	}
	for _, line := range stats {
		fm.DrawText(screen, line, textX, textY, theme.FontLG, theme.TextBody)
		textY += lineH
	}

	// ── 操作提示 ──
	hintY := float64(panelTopY) + float64(panelH) + 30
	fm.DrawCenteredText(screen, "[R] 重玩  [ENTER] 选关  [ESC] 返回", cx, hintY, theme.FontMD, theme.TextMuted)
}
