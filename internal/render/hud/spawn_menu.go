// spawn_menu.go — 造怪选择菜单（仅测试模式）。
// 网格展示所有敌人原型，带精灵图标、名称、hover 详情 tooltip。
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// SpawnEntry 造怪菜单中的单个敌人条目（纯值类型）。
type SpawnEntry struct {
	Name       string  // 原型名
	Label      string  // 中文显示名
	HpScale    float64 // HP 倍率
	SpeedScale float64 // 速度倍率
	Radius     float64 // 半径
	Reward     int     // 击杀奖励
	Boss       bool    // 是否 Boss
}

// SpawnMenuData 造怪菜单数据。
type SpawnMenuData struct {
	Entries    []SpawnEntry
	HoverIdx   int                            // 鼠标悬停索引（-1=无）
	SpriteFunc func(name string) *ebiten.Image // 敌人精灵获取
}

const (
	spawnCols    = 5
	spawnCardW   = float32(120)
	spawnCardH   = float32(44)
	spawnCardGap = float32(4)
	spawnCardR   = float32(6)
	spawnPad     = float32(12)
)

// spawnPanelGeom 计算面板几何参数。
func spawnPanelGeom(count int) (panelX, panelY, panelW, panelH, startY float32) {
	rows := (count + spawnCols - 1) / spawnCols
	panelW = spawnPad*2 + float32(spawnCols)*spawnCardW + float32(spawnCols-1)*spawnCardGap
	panelH = spawnPad*2 + float32(rows)*spawnCardH + float32(rows-1)*spawnCardGap + 28
	panelX = (float32(game.ScreenWidth) - panelW) / 2
	panelY = (float32(game.ScreenHeight) - panelH) / 2
	startY = panelY + spawnPad + 28
	return
}

// spawnCardPos 返回第 i 个卡片的左上角坐标。
func spawnCardPos(i int, panelX, startY float32) (cx, cy float32) {
	col := i % spawnCols
	row := i / spawnCols
	cx = panelX + spawnPad + float32(col)*(spawnCardW+spawnCardGap)
	cy = startY + float32(row)*(spawnCardH+spawnCardGap)
	return
}

// DrawSpawnMenu 渲染造怪选择菜单（居中弹出面板）。
func DrawSpawnMenu(screen *ebiten.Image, d SpawnMenuData) {
	fm := render.GlobalFont()
	if fm == nil || len(d.Entries) == 0 {
		return
	}

	panelX, panelY, panelW, panelH, startY := spawnPanelGeom(len(d.Entries))

	// 半透明遮罩
	draw.RoundRect(screen, 0, 0, float32(game.ScreenWidth), float32(game.ScreenHeight), 0, color.RGBA{A: 100})

	// 面板背景
	draw.RoundRect(screen, panelX, panelY, panelW, panelH, 12, color.RGBA{R: 15, G: 22, B: 40, A: 245})
	draw.StrokeRoundRect(screen, panelX, panelY, panelW, panelH, 12, 1, color.RGBA{R: 60, G: 80, B: 120, A: 200})

	// 标题
	fm.DrawCenteredBoldText(screen, "选择敌人类型", float64(panelX)+float64(panelW)/2, float64(panelY)+float64(spawnPad), theme.FontLG, color.White)

	// 右上角关闭提示
	fm.DrawRightText(screen, "Esc 关闭", float64(panelX)+float64(panelW)-float64(spawnPad), float64(panelY)+float64(spawnPad)+2, theme.FontXS, color.RGBA{R: 160, G: 175, B: 200, A: 220})

	// 卡片网格
	cardNormal := color.RGBA{R: 25, G: 35, B: 58, A: 240}
	cardHover := color.RGBA{R: 40, G: 55, B: 90, A: 255}
	cardBorder := color.RGBA{R: 80, G: 120, B: 180, A: 200}

	for i, entry := range d.Entries {
		cx, cy := spawnCardPos(i, panelX, startY)

		bg := cardNormal
		if i == d.HoverIdx {
			bg = cardHover
		}
		draw.RoundRect(screen, cx, cy, spawnCardW, spawnCardH, spawnCardR, bg)
		if i == d.HoverIdx {
			draw.StrokeRoundRect(screen, cx, cy, spawnCardW, spawnCardH, spawnCardR, 1, cardBorder)
		}

		// 精灵图标
		if d.SpriteFunc != nil {
			if img := d.SpriteFunc(entry.Name); img != nil {
				draw.Sprite(screen, img, float64(cx)+18, float64(cy)+float64(spawnCardH)/2, 28)
			}
		}

		// 名称（优先中文 Label）
		nameX := float64(cx) + 36
		nameY := float64(cy) + 6
		displayName := entry.Name
		if entry.Label != "" {
			displayName = entry.Label
		}
		fm.DrawBoldText(screen, displayName, nameX, nameY, theme.FontSM, color.White)

		// 简略属性
		tag := ""
		if entry.Boss {
			tag = "首领"
		} else if entry.HpScale >= 4 {
			tag = "精英"
		}
		if tag != "" {
			fm.DrawText(screen, tag, nameX, nameY+14, theme.FontXS, color.RGBA{R: 250, G: 190, B: 80, A: 240})
		}
		hpTxt := fmt.Sprintf("血量:%.0f", entry.HpScale*100)
		fm.DrawText(screen, hpTxt, nameX+40, nameY+14, theme.FontXS, color.RGBA{R: 200, G: 200, B: 210, A: 230})
	}

	// Hover tooltip（在面板下方）
	if d.HoverIdx >= 0 && d.HoverIdx < len(d.Entries) {
		drawSpawnTooltip(screen, fm, d.Entries[d.HoverIdx], panelX, panelY+panelH+4, panelW)
	}
}

// drawSpawnTooltip 渲染 hover 详情 tooltip。
func drawSpawnTooltip(screen *ebiten.Image, fm *render.FontManager, e SpawnEntry, x, y, maxW float32) {
	tipW := float32(260)
	tipH := float32(68)
	tipX := x + (maxW-tipW)/2
	tipY := y

	draw.RoundRect(screen, tipX, tipY, tipW, tipH, 8, color.RGBA{R: 12, G: 18, B: 35, A: 245})
	draw.StrokeRoundRect(screen, tipX, tipY, tipW, tipH, 8, 1, color.RGBA{R: 60, G: 80, B: 120, A: 180})

	tx := float64(tipX) + 10
	ty := float64(tipY) + 8

	// 名称 + 标签
	displayName := e.Name
	if e.Label != "" {
		displayName = e.Label
	}
	fm.DrawBoldText(screen, displayName, tx, ty, theme.FontMD, color.White)
	if e.Boss {
		fm.DrawText(screen, "首领", tx+100, ty+2, theme.FontSM, color.RGBA{R: 239, G: 68, B: 68, A: 255})
	} else if e.HpScale >= 4 {
		fm.DrawText(screen, "精英", tx+100, ty+2, theme.FontSM, color.RGBA{R: 180, G: 130, B: 255, A: 255})
	}
	ty += 18

	// 属性行
	im := render.GlobalIcons()
	attrX := tx

	if im != nil {
		if img := im.Get("stat-damage"); img != nil {
			draw.Sprite(screen, img, attrX+5, ty+5, 10)
			attrX += 14
		}
	}
	fm.DrawText(screen, fmt.Sprintf("血量:%.0f", e.HpScale*100), attrX, ty, theme.FontSM, color.RGBA{R: 239, G: 68, B: 68, A: 255})
	attrX += 60

	if im != nil {
		if img := im.Get("stat-movspd"); img != nil {
			draw.Sprite(screen, img, attrX+5, ty+5, 10)
			attrX += 14
		}
	}
	fm.DrawText(screen, fmt.Sprintf("速度:%.1f", e.SpeedScale), attrX, ty, theme.FontSM, color.RGBA{R: 74, G: 222, B: 128, A: 255})
	attrX += 60

	fm.DrawText(screen, fmt.Sprintf("半径:%.0f", e.Radius), attrX, ty, theme.FontSM, color.RGBA{R: 160, G: 160, B: 180, A: 200})
	ty += 16

	fm.DrawText(screen, fmt.Sprintf("击杀奖励: %d 金币", e.Reward), tx, ty, theme.FontXS, color.RGBA{R: 250, G: 190, B: 60, A: 200})
}

// SpawnMenuHitTest 检测点击了哪个敌人卡片，返回索引或 -1。
func SpawnMenuHitTest(px, py float32, count int) int {
	if count == 0 {
		return -1
	}
	panelX, _, _, _, startY := spawnPanelGeom(count)

	for i := 0; i < count; i++ {
		cx, cy := spawnCardPos(i, panelX, startY)
		if px >= cx && px <= cx+spawnCardW && py >= cy && py <= cy+spawnCardH {
			return i
		}
	}
	return -1
}

// SpawnMenuHoverTest 检测鼠标悬停在哪个卡片上。
func SpawnMenuHoverTest(mx, my float32, count int) int {
	return SpawnMenuHitTest(mx, my, count)
}
