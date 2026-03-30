// result.go — 结算场景。
// 游戏结束（胜利/失败）后显示统计信息，提供重玩或返回选关的选项。
// 包含分阶段动画：标题→星级→统计→按钮依次揭示。
package scene

import (
	"fmt"
	"image/color"
	"math"
	"strconv"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Animation phase state machine
// ---------------------------------------------------------------------------

type resultPhase int

const (
	resultTitlePhase   resultPhase = iota // 0.5s: title fades in + bounces
	resultStarsPhase                      // 0.6s: 1-3 stars appear one by one
	resultStatsPhase                      // ~1.8s: stats count up
	resultButtonsPhase                    // 0.3s: buttons slide up
	resultDone                            // interactive, waiting for tap
)

// Phase durations (seconds).
const (
	titlePhaseDur  = 0.5
	starStagger    = 0.2 // per star
	starPadding    = 0.2 // extra after last star
	statStagger    = 0.15
	statCountDur   = 0.4 // each stat counts over this duration
	buttonSlideDur = 0.3
)

// ---------------------------------------------------------------------------
// ResultData
// ---------------------------------------------------------------------------

// ResultData 结算数据。
type ResultData struct {
	MapID        string  // 关卡 ID
	MapName      string  // 关卡名称
	Won          bool    // 是否胜利
	Kills        int     // 击杀数
	Waves        int     // 通过波次数
	MaxWaves     int     // 总波次数
	Gold         int     // 剩余金币
	Towers       int     // 放置的塔数
	WardenType   string  // 使用的战灵类型
	ModeID       string  // 游戏模式 ID（用于重玩）
	DifficultyID string  // 难度 ID（用于重玩）
	Score        int     // 分数
	ElapsedSecs  float64 // 游戏用时（秒）
}

// ---------------------------------------------------------------------------
// ResultScene
// ---------------------------------------------------------------------------

// ResultScene 结算场景。
type ResultScene struct {
	switcher Switcher   // 场景切换器
	data     ResultData // 结算数据

	// Animation state
	phase       resultPhase
	phaseTimer  float64 // time elapsed in current phase
	totalTimer  float64 // total elapsed since scene start
	stars       int     // calculated star count (0 for defeat)
	interactive bool    // true after all animations complete
}

// NewResultScene 创建结算场景。
func NewResultScene(sw Switcher, data ResultData) *ResultScene {
	return &ResultScene{
		switcher: sw,
		data:     data,
		stars:    calcStars(data),
	}
}

// calcStars determines star rating based on performance.
// 3 stars = won + all waves cleared
// 2 stars = won + ≥80% waves cleared
// 1 star  = won
// 0 stars = defeat
func calcStars(d ResultData) int {
	if !d.Won {
		return 0
	}
	if d.MaxWaves > 0 && d.Waves >= d.MaxWaves {
		return 3
	}
	if d.MaxWaves > 0 && float64(d.Waves) >= float64(d.MaxWaves)*0.8 {
		return 2
	}
	return 1
}

// ---------------------------------------------------------------------------
// Update — advance animation state machine
// ---------------------------------------------------------------------------

func (s *ResultScene) Update() error {
	const dt = 1.0 / 60.0
	s.totalTimer += dt
	s.phaseTimer += dt

	switch s.phase {
	case resultTitlePhase:
		if s.phaseTimer >= titlePhaseDur {
			s.phaseTimer = 0
			if !s.data.Won || s.stars == 0 {
				// Skip stars for defeat
				s.phase = resultStatsPhase
			} else {
				s.phase = resultStarsPhase
			}
		}

	case resultStarsPhase:
		dur := float64(s.stars)*starStagger + starPadding
		if s.phaseTimer >= dur {
			s.phase = resultStatsPhase
			s.phaseTimer = 0
		}

	case resultStatsPhase:
		// 6 stats * stagger + counting duration
		dur := 5*statStagger + statCountDur + 0.1 // small buffer
		if s.phaseTimer >= dur {
			s.phase = resultButtonsPhase
			s.phaseTimer = 0
		}

	case resultButtonsPhase:
		if s.phaseTimer >= buttonSlideDur {
			s.phase = resultDone
			s.interactive = true
		}

	case resultDone:
		// nothing
	}

	// Only allow tap after animations complete
	if s.interactive && isTapJustPressed() {
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

// ---------------------------------------------------------------------------
// Button layout & hit detection (unchanged layout)
// ---------------------------------------------------------------------------

func (s *ResultScene) resultBtnLayout() (replayX, menuX, btnY float32, btnW, btnH float32) {
	cx := float64(game.ScreenWidth) / 2
	btnW = 160
	btnH = 40
	btnGap := float32(20)
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

// ---------------------------------------------------------------------------
// Draw — animated phased reveal
// ---------------------------------------------------------------------------

func (s *ResultScene) Draw(screen *ebiten.Image) {
	// Defeat: slightly redder background
	if !s.data.Won {
		screen.Fill(colorLerp(theme.ResultBg, color.RGBA{R: 30, G: 15, B: 20, A: 0xff}, 0.3))
	} else {
		screen.Fill(theme.ResultBg)
	}

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	cx := float64(game.ScreenWidth) / 2
	d := s.data

	// ── Title (phase 0+) ──
	s.drawTitle(screen, fm, cx, d)

	// ── Stars (phase 1+, victory only) ──
	if s.phase >= resultStarsPhase && d.Won && s.stars > 0 {
		s.drawStars(screen, fm, cx)
	}

	// ── Stats panel (phase 2+) ──
	if s.phase >= resultStatsPhase {
		s.drawStats(screen, fm, cx, d)
	}

	// ── Buttons (phase 3+) ──
	if s.phase >= resultButtonsPhase {
		s.drawButtons(screen, fm, cx)
	}
}

// ---------------------------------------------------------------------------
// drawTitle — victory: scale bounce; defeat: shake + fade
// ---------------------------------------------------------------------------

func (s *ResultScene) drawTitle(screen *ebiten.Image, fm *render.FontManager, cx float64, d ResultData) {
	titleY := 60.0
	subY := titleY + 50

	// Title progress: 0→1 over titlePhaseDur
	var titleProgress float64
	if s.phase == resultTitlePhase {
		titleProgress = clampF(s.phaseTimer/titlePhaseDur, 0, 1)
	} else {
		titleProgress = 1
	}

	titleAlpha := clampF(titleProgress*2, 0, 1) // fade in over first half

	if d.Won {
		// Victory: scale 0.5 → 1.1 → 1.0 (overshoot bounce)
		scale := victoryTitleScale(titleProgress)
		titleText := "防守成功"
		clr := colorWithAlpha(theme.HUDVictoryColor, titleAlpha)

		// Draw scaled via offscreen image approach — simpler: adjust font size
		scaledSize := theme.FontResultTitle * scale
		fm.DrawCenteredText(screen, titleText, cx, titleY-(scale-1)*20, scaledSize, clr)
	} else {
		// Defeat: shake wobble + fade in
		titleText := "游戏结束"
		clr := colorWithAlpha(theme.HUDDefeatColor, titleAlpha)

		// Shake offset that decays
		shakeX := 0.0
		if titleProgress < 0.7 {
			shakeAmp := 6.0 * (1 - titleProgress/0.7)
			shakeX = shakeAmp * math.Sin(titleProgress*40)
		}
		fm.DrawCenteredText(screen, titleText, cx+shakeX, titleY, theme.FontResultTitle, clr)
	}

	// Subtitle: fades in during last 0.2s of title phase, or fully visible after
	var subAlpha float64
	if s.phase == resultTitlePhase {
		subAlpha = clampF((titleProgress-0.6)/0.4, 0, 1) // start at 60% of title phase
	} else {
		subAlpha = 1
	}
	subTxt := fmt.Sprintf("地图: %s", d.MapName)
	fm.DrawCenteredText(screen, subTxt, cx, subY, theme.FontH1, colorWithAlpha(theme.TextMuted, subAlpha))
}

// victoryTitleScale returns the bounce-in scale factor (0.5 → 1.1 → 1.0).
func victoryTitleScale(t float64) float64 {
	if t <= 0 {
		return 0.5
	}
	if t >= 1 {
		return 1.0
	}
	// Two-phase: 0→0.7 = scale 0.5→1.1, 0.7→1.0 = scale 1.1→1.0
	if t < 0.7 {
		p := t / 0.7
		return 0.5 + 0.6*easeOutQuad(p) // 0.5 → 1.1
	}
	p := (t - 0.7) / 0.3
	return 1.1 - 0.1*easeOutQuad(p) // 1.1 → 1.0
}

// ---------------------------------------------------------------------------
// drawStars — pop-in stars below title
// ---------------------------------------------------------------------------

func (s *ResultScene) drawStars(screen *ebiten.Image, fm *render.FontManager, cx float64) {
	starY := 118.0
	starSize := 32.0
	starGap := 40.0

	totalW := float64(s.stars)*starGap - starGap + starSize
	startX := cx - totalW/2

	for i := 0; i < s.stars; i++ {
		// Each star staggers in
		delay := float64(i) * starStagger
		var progress float64
		if s.phase == resultStarsPhase {
			progress = clampF((s.phaseTimer-delay)/starStagger, 0, 1)
		} else {
			progress = 1
		}

		if progress <= 0 {
			continue
		}

		// Pop: scale 0 → 1.2 → 1.0
		scale := starPopScale(progress)
		alpha := clampF(progress*3, 0, 1) // fade in quickly

		sx := startX + float64(i)*starGap
		scaledSize := starSize * scale
		clr := colorWithAlpha(theme.ResGold, alpha)
		fm.DrawCenteredText(screen, "★", sx+starSize/2, starY, scaledSize, clr)
	}
}

// starPopScale returns pop-in scale (0 → 1.2 → 1.0).
func starPopScale(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1.0
	}
	if t < 0.6 {
		p := t / 0.6
		return 1.2 * easeOutQuad(p) // 0 → 1.2
	}
	p := (t - 0.6) / 0.4
	return 1.2 - 0.2*easeOutQuad(p) // 1.2 → 1.0
}

// ---------------------------------------------------------------------------
// drawStats — counting numbers with stagger
// ---------------------------------------------------------------------------

func (s *ResultScene) drawStats(screen *ebiten.Image, fm *render.FontManager, cx float64, d ResultData) {
	const (
		panelW = float32(500)
		panelH = float32(220)
		panelR = float32(16)
		padX   = 30.0
		padY   = 20.0
		labelH = 18.0
		valueH = 34.0
		rowH   = labelH + valueH
		cols   = 3
	)
	panelX := float32(cx) - panelW/2
	subY := 60.0 + 50
	panelTopY := float32(subY + 36)

	// Panel fade-in
	var panelAlpha float64
	if s.phase == resultStatsPhase {
		panelAlpha = clampF(s.phaseTimer/0.2, 0, 1) // quick fade in
	} else {
		panelAlpha = 1
	}

	draw.RoundRect(screen, panelX, panelTopY, panelW, panelH, panelR,
		colorWithAlpha(theme.ResultStatsBg, panelAlpha))
	draw.StrokeRoundRect(screen, panelX, panelTopY, panelW, panelH, panelR, 1,
		colorWithAlpha(theme.PanelBorder, panelAlpha))

	// Stat items
	type statItem struct {
		label  string
		target int
		format string // "d" for int, "waves" for x/y, "time" for Xs
		clr    color.Color
	}
	items := []statItem{
		{"波次", d.Waves, "waves", theme.ResWaves},
		{"击杀", d.Kills, "d", theme.HUDDefeatColor},
		{"分数", d.Score, "d", theme.TonePrimary},
		{"金币", d.Gold, "d", theme.ResGold},
		{"塔数", d.Towers, "d", theme.StatusSkill},
		{"用时", int(d.ElapsedSecs), "time", theme.StatusWarden},
	}

	colW := (float64(panelW) - padX*2) / cols
	baseX := float64(panelX) + padX
	baseY := float64(panelTopY) + padY

	for i, item := range items {
		col := i % cols
		row := i / cols
		ix := baseX + float64(col)*colW
		iy := baseY + float64(row)*rowH

		// Staggered count-up
		delay := float64(i) * statStagger
		var progress float64
		if s.phase == resultStatsPhase {
			elapsed := s.phaseTimer - delay
			progress = clampF(elapsed/statCountDur, 0, 1)
		} else {
			progress = 1
		}

		// For defeat: faster counting (no stagger)
		if !d.Won && s.phase == resultStatsPhase {
			progress = clampF(s.phaseTimer/0.3, 0, 1)
		}

		alpha := clampF(progress*2, 0, 1) // fade in during first half of counting

		// Label
		fm.DrawText(screen, item.label, ix, iy, theme.FontBody, colorWithAlpha(theme.TextMuted, alpha))

		// Value with count-up
		displayVal := countUp(item.target, progress)
		var valText string
		switch item.format {
		case "waves":
			displayWaves := countUp(d.Waves, progress)
			displayMax := countUp(d.MaxWaves, progress)
			valText = strconv.Itoa(displayWaves) + " / " + strconv.Itoa(displayMax)
		case "time":
			valText = strconv.Itoa(displayVal) + "s"
		default:
			valText = strconv.Itoa(displayVal)
		}

		fm.DrawText(screen, valText, ix, iy+labelH, 28, colorWithAlpha(item.clr, alpha))
	}
}

// ---------------------------------------------------------------------------
// drawButtons — slide up from bottom
// ---------------------------------------------------------------------------

func (s *ResultScene) drawButtons(screen *ebiten.Image, fm *render.FontManager, cx float64) {
	subY := 60.0 + 50
	panelTopY := subY + 36
	btnBaseY := panelTopY + 220 + 24

	btnW := float32(160)
	btnH := float32(40)
	btnR := float32(12)
	btnGap := float32(20)

	// Slide-up: start 50px below, ease to final position
	var slideOffset float64
	if s.phase == resultButtonsPhase {
		progress := clampF(s.phaseTimer/buttonSlideDur, 0, 1)
		slideOffset = 50 * (1 - easeOutQuad(progress))
	}

	alpha := 1.0
	if s.phase == resultButtonsPhase {
		alpha = clampF(s.phaseTimer/buttonSlideDur, 0, 1)
	}

	actualY := float32(btnBaseY + slideOffset)

	// Replay button (green)
	replayX := float32(cx) - btnW - btnGap/2
	draw.RoundRect(screen, replayX, actualY, btnW, btnH, btnR,
		colorWithAlpha(theme.TonePrimary, alpha))
	fm.DrawCenteredText(screen, "重玩",
		float64(replayX)+float64(btnW)/2, float64(actualY)+float64(btnH)/2-6,
		theme.FontH2, colorWithAlpha(color.White, alpha))

	// Menu button (gray)
	menuX := float32(cx) + btnGap/2
	draw.RoundRect(screen, menuX, actualY, btnW, btnH, btnR,
		colorWithAlpha(theme.ToneSecondary, alpha))
	fm.DrawCenteredText(screen, "选关",
		float64(menuX)+float64(btnW)/2, float64(actualY)+float64(btnH)/2-6,
		theme.FontH2, colorWithAlpha(color.White, alpha))
}

// ---------------------------------------------------------------------------
// Animation helpers
// ---------------------------------------------------------------------------

// easeOutQuad: decelerating ease-out (fast start, slow end).
func easeOutQuad(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

// countUp interpolates from 0 to target using ease-out.
func countUp(target int, progress float64) int {
	if progress >= 1 {
		return target
	}
	if progress <= 0 {
		return 0
	}
	return int(float64(target) * easeOutQuad(progress))
}

// clampF clamps a float64 to [lo, hi].
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// colorWithAlpha applies an alpha multiplier (0-1) to a color.
func colorWithAlpha(c color.Color, alpha float64) color.Color {
	r, g, b, a := c.RGBA()
	newA := uint8(clampF(float64(a>>8)*alpha, 0, 255))
	return color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: newA,
	}
}

// colorLerp linearly interpolates between two colors.
func colorLerp(a, b color.Color, t float64) color.Color {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	lerp := func(x, y uint32) uint8 {
		return uint8(clampF(float64(x>>8)*(1-t)+float64(y>>8)*t, 0, 255))
	}
	return color.RGBA{
		R: lerp(ar, br),
		G: lerp(ag, bg),
		B: lerp(ab, bb),
		A: lerp(aa, ba),
	}
}
