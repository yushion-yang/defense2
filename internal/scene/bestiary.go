// bestiary.go — 图鉴场景。
// 展示敌人/能力/战灵的百科全书，含统计数据。
// 从 Title 场景进入，三个 Tab 页切换。
package scene

import (
	"image/color"
	"log"
	"sort"

	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/persistence"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── Tab 枚举 ─────────────────────────────────────

const (
	bestiaryTabEnemies   = 0
	bestiaryTabAbilities = 1
	bestiaryTabWardens   = 2
)

// ── 布局常量 ─────────────────────────────────────

const (
	bsCardW   = float32(110)
	bsCardH   = float32(80)
	bsCardGap = float32(10)
	bsTabW    = float32(80)
	bsTabH    = float32(28)
	bsTabGap  = float32(8)
	bsDetailH = float32(120) // 详情面板高度
	bsGridTop = float32(80)  // 卡片网格起始 Y
	bsBackW   = float32(80)
	bsBackH   = float32(28)
)

// ── BestiaryScene ────────────────────────────────

// BestiaryScene 图鉴场景。
type BestiaryScene struct {
	switcher    Switcher
	progressMgr *persistence.ProgressManager
	bestiary    persistence.BestiaryData

	tab      int // 当前 Tab
	selected int // 选中项索引，-1=无
	hover    int // 悬停项索引，-1=无
	scrollY  float32

	// 缓存的配置数据
	enemies   []*config.EnemyArchetype
	abilities []abilityEntry
	wardens   []wardenEntry

	bgGrad *draw.CachedGradient
}

type abilityEntry struct {
	Type     string
	Label    string
	Category string
	Icon     string
}

type wardenEntry struct {
	Key      string
	Name     string
	Category string
}

// NewBestiaryScene 创建图鉴场景。
func NewBestiaryScene(sw Switcher) *BestiaryScene {
	pm := persistence.DefaultProgressManager()
	bestiary := pm.GetBestiary()

	// 加载敌人列表（按 ID 排序）
	archetypes, errArch := config.LoadEnemyArchetypes()
	if errArch != nil {
		log.Printf("[bestiary] load archetypes failed: %v", errArch)
	}
	enemies := make([]*config.EnemyArchetype, 0, len(archetypes))
	for _, a := range archetypes {
		enemies = append(enemies, a)
	}
	sort.Slice(enemies, func(i, j int) bool { return enemies[i].ID < enemies[j].ID })

	// 加载能力列表
	abilTable := config.GlobalAbilityTable()
	abilities := make([]abilityEntry, 0, len(abilTable))
	for _, a := range abilTable {
		abilities = append(abilities, abilityEntry{
			Type:     a.Type,
			Label:    a.Label,
			Category: a.Category,
			Icon:     a.Icon,
		})
	}
	sort.Slice(abilities, func(i, j int) bool { return abilities[i].Type < abilities[j].Type })

	// 加载战灵列表
	wardens := make([]wardenEntry, 0, 5)
	for _, key := range []string{"prince", "core", "chain", "skystrike", "envoy"} {
		if wc := config.GlobalWardenConfig(key); wc != nil {
			wardens = append(wardens, wardenEntry{
				Key:      key,
				Name:     wc.Name,
				Category: wc.Category,
			})
		}
	}

	return &BestiaryScene{
		switcher:    sw,
		progressMgr: pm,
		bestiary:    bestiary,
		tab:         bestiaryTabEnemies,
		selected:    -1,
		hover:       -1,
		enemies:     enemies,
		abilities:   abilities,
		wardens:     wardens,
		bgGrad:      draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
}

// ── 坐标辅助 ─────────────────────────────────────

func bsTabGeom(idx, count int) (float32, float32, float32, float32) {
	totalW := float32(count)*bsTabW + float32(count-1)*bsTabGap
	startX := (float32(game.ScreenWidth) - totalW) / 2
	x := startX + float32(idx)*(bsTabW+bsTabGap)
	return x, 46, bsTabW, bsTabH
}

func bsBackGeom() (float32, float32, float32, float32) {
	return 20, 14, bsBackW, bsBackH
}

func (s *BestiaryScene) cardCount() int {
	switch s.tab {
	case bestiaryTabEnemies:
		return len(s.enemies)
	case bestiaryTabAbilities:
		return len(s.abilities)
	case bestiaryTabWardens:
		return len(s.wardens)
	}
	return 0
}

func bsCardGeom(idx int) (float32, float32, float32, float32) {
	availW := float64(game.ScreenWidth) - 40
	cellW := float64(bsCardW + bsCardGap)
	cols := int(availW / cellW)
	if cols < 1 {
		cols = 1
	}
	col := idx % cols
	row := idx / cols
	marginX := (float32(game.ScreenWidth) - float32(cols)*(bsCardW+bsCardGap) + bsCardGap) / 2
	x := marginX + float32(col)*(bsCardW+bsCardGap)
	y := bsGridTop + float32(row)*(bsCardH+bsCardGap)
	return x, y, bsCardW, bsCardH
}

// ── Update ───────────────────────────────────────

func (s *BestiaryScene) Update() error {
	mx, my := draw.CursorPos()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.JustPressedTouchIDs()) > 0 {
		// Tab 按钮
		for idx := 0; idx < 3; idx++ {
			tx, ty, tw, th := bsTabGeom(idx, 3)
			if mx >= float64(tx) && mx <= float64(tx+tw) && my >= float64(ty) && my <= float64(ty+th) {
				if s.tab != idx {
					s.tab = idx
					s.selected = -1
					s.scrollY = 0
					playUIClick(s.switcher)
				}
			}
		}

		// 返回按钮
		bx, by, bw, bh := bsBackGeom()
		if mx >= float64(bx) && mx <= float64(bx+bw) && my >= float64(by) && my <= float64(by+bh) {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewSelectScene(s.switcher))
			return nil
		}

		// 卡片点击
		count := s.cardCount()
		for idx := 0; idx < count; idx++ {
			cx, cy, cw, ch := bsCardGeom(idx)
			if mx >= float64(cx) && mx <= float64(cx+cw) && my >= float64(cy) && my <= float64(cy+ch) {
				if s.selected == idx {
					s.selected = -1
				} else {
					s.selected = idx
				}
				playUIClick(s.switcher)
			}
		}
	}

	// 悬停
	s.hover = -1
	count := s.cardCount()
	for idx := 0; idx < count; idx++ {
		cx, cy, cw, ch := bsCardGeom(idx)
		if mx >= float64(cx) && mx <= float64(cx+cw) && my >= float64(cy) && my <= float64(cy+ch) {
			s.hover = idx
		}
	}

	// Esc 返回
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
	}

	return nil
}

// ── Draw ─────────────────────────────────────────

func (s *BestiaryScene) Draw(screen *ebiten.Image) {
	s.bgGrad.Draw(screen, 0, 0)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	sw := float64(game.ScreenWidth)

	// 标题
	fm.DrawCenteredBoldText(screen, i18n.T("scene.bestiary.title"), sw/2, 20, 22, theme.TextTitle)

	// 返回按钮
	bx, by, bw, bh := bsBackGeom()
	ui.Button(screen, bx, by, bw, bh, i18n.T("settings.back"), ui.ButtonStyle{
		BgColor:   theme.BtnSecondary,
		TextColor: color.White,
		FontSize:  12,
		Radius:    6,
		Bold:      true,
	})

	// Tab 按钮
	tabLabels := [3]string{
		i18n.T("scene.bestiary.tab.enemies"),
		i18n.T("scene.bestiary.tab.abilities"),
		i18n.T("scene.bestiary.tab.wardens"),
	}
	for idx := 0; idx < 3; idx++ {
		tx, ty, tw, th := bsTabGeom(idx, 3)
		btnClr := theme.ToneSecondary
		if idx == s.tab {
			btnClr = theme.TonePrimary
		}
		ui.Button(screen, tx, ty, tw, th, tabLabels[idx], ui.ButtonStyle{
			BgColor:   btnClr,
			TextColor: color.White,
			FontSize:  12,
			Radius:    6,
			Bold:      idx == s.tab,
		})
	}

	// 卡片网格
	switch s.tab {
	case bestiaryTabEnemies:
		s.drawEnemyCards(screen, fm)
	case bestiaryTabAbilities:
		s.drawAbilityCards(screen, fm)
	case bestiaryTabWardens:
		s.drawWardenCards(screen, fm)
	}

	// 选中详情面板
	s.drawDetailPanel(screen, fm)
}

// ── 敌人卡片 ─────────────────────────────────────

func (s *BestiaryScene) drawEnemyCards(screen *ebiten.Image, fm *render.FontManager) {
	for idx, arch := range s.enemies {
		x, y, w, h := bsCardGeom(idx)
		selected := idx == s.selected
		hovered := idx == s.hover

		bgClr := theme.PanelBg
		if selected {
			bgClr = theme.TonePrimary
		} else if hovered {
			bgClr = theme.ToneSecondary
		}

		draw.RoundRect(screen, x, y, w, h, 8, bgClr)
		draw.StrokeRoundRect(screen, x, y, w, h, 8, 1, theme.PanelBorder)

		cx := float64(x) + float64(w)/2

		// 名称
		fm.DrawCenteredBoldText(screen, arch.Label, cx, float64(y)+14, 11, theme.TextTitle)

		// Boss 标记
		if arch.Boss {
			fm.DrawCenteredText(screen, i18n.T("game.tag.boss"), cx, float64(y)+30, 9, theme.ToneGold)
		}

		// 击杀统计
		kills := s.bestiary.EnemyKills[arch.ID]
		killStr := i18n.TF("scene.bestiary.kills", kills)
		fm.DrawCenteredText(screen, killStr, cx, float64(y)+float64(h)-18, 10, theme.TextMuted)
	}
}

// ── 能力卡片 ─────────────────────────────────────

func (s *BestiaryScene) drawAbilityCards(screen *ebiten.Image, fm *render.FontManager) {
	icons := render.GlobalIcons()
	for idx, ab := range s.abilities {
		x, y, w, h := bsCardGeom(idx)
		selected := idx == s.selected
		hovered := idx == s.hover

		bgClr := theme.PanelBg
		if selected {
			bgClr = theme.TonePrimary
		} else if hovered {
			bgClr = theme.ToneSecondary
		}

		draw.RoundRect(screen, x, y, w, h, 8, bgClr)
		draw.StrokeRoundRect(screen, x, y, w, h, 8, 1, theme.PanelBorder)

		cx := float64(x) + float64(w)/2

		// 图标
		if icons != nil {
			if img := icons.Get(ab.Icon); img != nil {
				draw.Sprite(screen, img, cx, float64(y)+22, 24)
			}
		}

		// 名称
		fm.DrawCenteredText(screen, ab.Label, cx, float64(y)+44, 10, theme.TextTitle)

		// 选用统计
		picks := s.bestiary.AbilityPicks[ab.Type]
		pickStr := i18n.TF("scene.bestiary.picks", picks)
		fm.DrawCenteredText(screen, pickStr, cx, float64(y)+float64(h)-14, 9, theme.TextMuted)
	}
}

// ── 战灵卡片 ─────────────────────────────────────

func (s *BestiaryScene) drawWardenCards(screen *ebiten.Image, fm *render.FontManager) {
	for idx, w := range s.wardens {
		x, y, cw, ch := bsCardGeom(idx)
		selected := idx == s.selected
		hovered := idx == s.hover

		bgClr := theme.PanelBg
		if selected {
			bgClr = theme.TonePrimary
		} else if hovered {
			bgClr = theme.ToneSecondary
		}

		draw.RoundRect(screen, x, y, cw, ch, 8, bgClr)
		draw.StrokeRoundRect(screen, x, y, cw, ch, 8, 1, theme.PanelBorder)

		cx := float64(x) + float64(cw)/2

		// 名称
		fm.DrawCenteredBoldText(screen, w.Name, cx, float64(y)+20, 12, theme.TextTitle)

		// 类别
		fm.DrawCenteredText(screen, w.Category, cx, float64(y)+38, 10, theme.TextMuted)

		// 使用统计
		games := s.bestiary.WardenGames[w.Key]
		gameStr := i18n.TF("scene.bestiary.games", games)
		fm.DrawCenteredText(screen, gameStr, cx, float64(y)+float64(ch)-14, 10, theme.TextMuted)
	}
}

// ── 详情面板 ─────────────────────────────────────

func (s *BestiaryScene) drawDetailPanel(screen *ebiten.Image, fm *render.FontManager) {
	if s.selected < 0 {
		return
	}

	panelY := float32(game.ScreenHeight) - bsDetailH - 8
	panelX := float32(20)
	panelW := float32(game.ScreenWidth) - 40

	draw.RoundRect(screen, panelX, panelY, panelW, bsDetailH, 10, theme.PanelBg)
	draw.StrokeRoundRect(screen, panelX, panelY, panelW, bsDetailH, 10, 1, theme.PanelBorder)

	px := float64(panelX) + 20
	py := float64(panelY) + 16

	switch s.tab {
	case bestiaryTabEnemies:
		if s.selected < len(s.enemies) {
			arch := s.enemies[s.selected]
			fm.DrawBoldText(screen, arch.Label, px, py, 16, theme.TextTitle)
			if arch.Boss {
				fm.DrawText(screen, " "+i18n.T("game.tag.boss"), px+fm.MeasureText(arch.Label, 16)+4, py, 12, theme.ToneGold)
			}
			fm.DrawText(screen, i18n.TF("scene.bestiary.hp", arch.HPScale), px, py+24, 12, theme.TextBody)
			fm.DrawText(screen, i18n.TF("scene.bestiary.speed", arch.SpeedScale), px+140, py+24, 12, theme.TextBody)
			fm.DrawText(screen, i18n.TF("scene.bestiary.reward", arch.RewardScale), px+280, py+24, 12, theme.TextBody)

			// 能力列表
			if len(arch.Abilities) > 0 {
				abilStr := ""
				for i, a := range arch.Abilities {
					if i > 0 {
						abilStr += ", "
					}
					abilStr += a.Type
				}
				fm.DrawText(screen, abilStr, px, py+48, 11, theme.TextMuted)
			}

			kills := s.bestiary.EnemyKills[arch.ID]
			fm.DrawText(screen, i18n.TF("scene.bestiary.kills", kills), px, py+72, 12, theme.ToneGold)
		}

	case bestiaryTabAbilities:
		if s.selected < len(s.abilities) {
			ab := s.abilities[s.selected]
			fm.DrawBoldText(screen, ab.Label, px, py, 16, theme.TextTitle)
			fm.DrawText(screen, i18n.TF("scene.bestiary.category", ab.Category), px, py+24, 12, theme.TextBody)
			picks := s.bestiary.AbilityPicks[ab.Type]
			fm.DrawText(screen, i18n.TF("scene.bestiary.picks", picks), px, py+48, 12, theme.ToneGold)
		}

	case bestiaryTabWardens:
		if s.selected < len(s.wardens) {
			w := s.wardens[s.selected]
			fm.DrawBoldText(screen, w.Name, px, py, 16, theme.TextTitle)
			fm.DrawText(screen, i18n.TF("scene.bestiary.category", w.Category), px, py+24, 12, theme.TextBody)

			// 从配置获取详细属性
			if wc := config.GlobalWardenConfig(w.Key); wc != nil {
				fm.DrawText(screen, wc.AttackName+": "+wc.AttackDesc, px, py+48, 11, theme.TextMuted)
				fm.DrawText(screen, wc.SpecialName+": "+wc.SpecialDesc, px, py+66, 11, theme.TextMuted)
			}

			games := s.bestiary.WardenGames[w.Key]
			fm.DrawText(screen, i18n.TF("scene.bestiary.games", games), px, py+90, 12, theme.ToneGold)
		}
	}
}
