// result.go — 结算场景。
// 游戏结束（胜利/失败）后显示统计信息，提供重玩或返回选关的选项。
package scene

import (
	"fmt"
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ResultData 结算数据。
type ResultData struct {
	MapID        string // 关卡 ID
	MapName      string // 关卡名称
	Won          bool   // 是否胜利
	Kills        int    // 击杀数
	Waves        int    // 通过波次数
	MaxWaves     int    // 总波次数
	Gold         int    // 剩余金币
	Towers       int    // 放置的塔数
	WardenType   string // 使用的战灵类型
	ModeID       string // 游戏模式 ID（用于重玩）
	DifficultyID string // 难度 ID（用于重玩）
	Score        int    // 分数
	ElapsedSecs  float64 // 游戏用时（秒）
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
	if isTapJustPressed() {
		mx, my := draw.CursorPos()
		if s.hitReplayButton(mx, my) {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewStageSceneWithOpts(s.switcher, StageOptions{
				MapID:        s.data.MapID,
				WardenType:   s.data.WardenType,
				ModeID:       s.data.ModeID,
				DifficultyID: s.data.DifficultyID,
			}))
			return nil
		}
		if s.hitMenuButton(mx, my) {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewSelectScene(s.switcher))
			return nil
		}
	}
	return nil
}

// 按钮布局常量（与 Draw 一致）
func (s *ResultScene) resultBtnLayout() (replayX, menuX, btnY float32, btnW, btnH float32) {
	cx := float64(game.ScreenWidth) / 2
	btnW = 160
	btnH = 40
	btnGap := float32(20)
	// 与 Draw 中的 panelTopY + panelH + 24 对齐
	btnY = float32(60+50+36) + 220 + 24
	replayX = float32(cx) - btnW - btnGap/2
	menuX = float32(cx) + btnGap/2
	return
}

func (s *ResultScene) hitReplayButton(mx, my float64) bool {
	rx, _, by, bw, bh := s.resultBtnLayout()
	return mx >= float64(rx) && mx <= float64(rx+bw) && my >= float64(by) && my <= float64(by+bh)
}

func (s *ResultScene) hitMenuButton(mx, my float64) bool {
	_, mx2, by, bw, bh := s.resultBtnLayout()
	return mx >= float64(mx2) && mx <= float64(mx2+bw) && my >= float64(by) && my <= float64(by+bh)
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
	titleY := 60.0
	if d.Won {
		fm.DrawCenteredText(screen, "防守成功", cx, titleY, theme.FontResultTitle, theme.HUDVictoryColor)
	} else {
		fm.DrawCenteredText(screen, "游戏结束", cx, titleY, theme.FontResultTitle, theme.HUDDefeatColor)
	}

	// ── 副标题 ──
	subY := titleY + 50
	subTxt := fmt.Sprintf("地图: %s", d.MapName)
	fm.DrawCenteredText(screen, subTxt, cx, subY, theme.FontXL, theme.TextMuted)

	// ── 统计面板 ──
	const (
		panelW   = float32(500)
		panelH   = float32(220)
		panelR   = float32(16)
		padX     = 30.0
		padY     = 20.0
		labelH   = 18.0
		valueH   = 34.0
		rowH     = labelH + valueH
		cols     = 3
	)
	panelX := float32(cx) - panelW/2
	panelTopY := float32(subY + 36)

	draw.RoundRect(screen, panelX, panelTopY, panelW, panelH, panelR, theme.ResultStatsBg)
	draw.StrokeRoundRect(screen, panelX, panelTopY, panelW, panelH, panelR, 1, theme.PanelBorder)

	// 统计项（标签+大数字，3列2行网格）
	type statItem struct {
		label string
		value string
		clr   color.Color
	}
	items := []statItem{
		{"波次", fmt.Sprintf("%d / %d", d.Waves, d.MaxWaves), theme.ResWaves},
		{"击杀", fmt.Sprintf("%d", d.Kills), theme.HUDDefeatColor},
		{"分数", fmt.Sprintf("%d", d.Score), theme.TonePrimary},
		{"金币", fmt.Sprintf("%d", d.Gold), theme.ResGold},
		{"塔数", fmt.Sprintf("%d", d.Towers), theme.StatusSkill},
		{"用时", fmt.Sprintf("%ds", int(d.ElapsedSecs)), theme.StatusWarden},
	}

	colW := (float64(panelW) - padX*2) / cols
	baseX := float64(panelX) + padX
	baseY := float64(panelTopY) + padY

	for i, item := range items {
		col := i % cols
		row := i / cols
		ix := baseX + float64(col)*colW
		iy := baseY + float64(row)*rowH

		fm.DrawText(screen, item.label, ix, iy, theme.FontSM, theme.TextMuted)
		fm.DrawText(screen, item.value, ix, iy+labelH, 28, item.clr)
	}

	// ── 按钮区域 ──
	btnBaseY := float64(panelTopY) + float64(panelH) + 24
	btnW := float32(160)
	btnH := float32(40)
	btnR := float32(12)
	btnGap := float32(20)

	// 重玩按钮（绿色）
	replayX := float32(cx) - btnW - btnGap/2
	draw.RoundRect(screen, replayX, float32(btnBaseY), btnW, btnH, btnR, theme.TonePrimary)
	fm.DrawCenteredText(screen, "重玩",
		float64(replayX)+float64(btnW)/2, btnBaseY+float64(btnH)/2-6, theme.FontLG, color.White)

	// 返回按钮（灰色）
	menuX := float32(cx) + btnGap/2
	draw.RoundRect(screen, menuX, float32(btnBaseY), btnW, btnH, btnR, theme.ToneSecondary)
	fm.DrawCenteredText(screen, "选关",
		float64(menuX)+float64(btnW)/2, btnBaseY+float64(btnH)/2-6, theme.FontLG, color.White)
}
