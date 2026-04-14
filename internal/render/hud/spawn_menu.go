// spawn_menu.go — 造怪选择菜单（仅测试模式）。
// 网格展示所有敌人原型，带精灵图标、名称、hover 详情 tooltip。
package hud

import (
	"image/color"

	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// SpawnEntry 造怪菜单中的单个敌人条目（纯值类型）。
type SpawnEntry struct {
	Name       string  // 原型名
	Label      string  // 中文显示名
	HpScale    float64 // HP 倍率
	SpeedScale float64 // 速度倍率
	Radius     float64 // 半径
	Boss       bool    // 是否 Boss
}

// SpawnMenuData 造怪菜单数据。
type SpawnMenuData struct {
	Entries    []SpawnEntry
	HoverIdx   int                             // 鼠标悬停索引（-1=无）
	SpriteFunc func(name string) *ebiten.Image // 敌人精灵获取
}

const (
	spawnCardGap = float32(4)
	spawnCardR   = float32(6)
	spawnPad     = float32(12)
)

// spawnPanelGeom 计算面板几何参数。
func spawnPanelGeom(count int) (panelX, panelY, panelW, panelH, startY float32) {
	rows := (count + theme.SpawnCols - 1) / theme.SpawnCols
	panelW = spawnPad*2 + float32(theme.SpawnCols)*theme.SpawnCardW + float32(theme.SpawnCols-1)*spawnCardGap
	panelH = spawnPad*2 + float32(rows)*theme.SpawnCardH + float32(rows-1)*spawnCardGap + 28
	panelX = (float32(theme.CanvasW) - panelW) / 2
	panelY = (float32(theme.CanvasH) - panelH) / 2
	startY = panelY + spawnPad + 28
	return
}

// spawnCardPos 返回第 i 个卡片的左上角坐标。
func spawnCardPos(i int, panelX, startY float32) (cx, cy float32) {
	col := i % theme.SpawnCols
	row := i / theme.SpawnCols
	cx = panelX + spawnPad + float32(col)*(theme.SpawnCardW+spawnCardGap)
	cy = startY + float32(row)*(theme.SpawnCardH+spawnCardGap)
	return
}

// DrawSpawnMenu 渲染造怪选择菜单（居中弹出面板）。
func DrawSpawnMenu(screen *ebiten.Image, d SpawnMenuData) {
	if len(d.Entries) == 0 {
		return
	}

	panelX, panelY, panelW, panelH, startY := spawnPanelGeom(len(d.Entries))

	// 半透明遮罩
	ui.Overlay(screen, 100)

	// 面板背景
	ui.Panel(screen, panelX, panelY, panelW, panelH, ui.PanelStyle{
		BgColor:     color.RGBA{R: 15, G: 22, B: 40, A: 245},
		BorderColor: color.RGBA{R: 60, G: 80, B: 120, A: 200},
		Radius:      12,
	})

	// 标题
	ui.Label(screen, i18n.T("hud.spawn.title"),
		float64(panelX), float64(panelY)+float64(spawnPad), float64(panelW),
		ui.LabelStyle{Font: theme.FontLG, Bold: true, Align: ui.AlignCenter})

	// 右上角关闭提示
	ui.Label(screen, i18n.T("hud.spawn.esc_close"),
		float64(panelX)+float64(spawnPad), float64(panelY)+float64(spawnPad)+2, float64(panelW)-float64(spawnPad)*2,
		ui.LabelStyle{Font: theme.FontXS, Color: color.RGBA{R: 160, G: 175, B: 200, A: 220}, Align: ui.AlignRight})

	// 卡片网格
	cardNormal := color.RGBA{R: 25, G: 35, B: 58, A: 240}
	cardHover := color.RGBA{R: 40, G: 55, B: 90, A: 255}
	cardBorder := color.RGBA{R: 80, G: 120, B: 180, A: 200}

	for i, entry := range d.Entries {
		cx, cy := spawnCardPos(i, panelX, startY)

		bg := cardNormal
		var border color.Color
		if i == d.HoverIdx {
			bg = cardHover
			border = cardBorder
		}
		ui.Panel(screen, cx, cy, theme.SpawnCardW, theme.SpawnCardH, ui.PanelStyle{
			BgColor:     bg,
			BorderColor: border,
			Radius:      spawnCardR,
		})

		// 精灵图标
		if d.SpriteFunc != nil {
			if img := d.SpriteFunc(entry.Name); img != nil {
				draw.Sprite(screen, img, float64(cx)+18, float64(cy)+float64(theme.SpawnCardH)/2, 28) //nolint:hud
			}
		}

		// 名称（优先中文 Label）
		nameX := float64(cx) + 36
		nameY := float64(cy) + 6
		displayName := entry.Name
		if entry.Label != "" {
			displayName = entry.Label
		}
		ui.Label(screen, displayName, nameX, nameY, 72,
			ui.LabelStyle{Font: theme.FontSM, Bold: true})

		// 简略属性
		if entry.Boss {
			ui.Label(screen, i18n.T("hud.spawn.boss"), nameX, nameY+14, 40,
				ui.LabelStyle{Font: theme.FontXS, Color: color.RGBA{R: 250, G: 190, B: 80, A: 240}})
		}
		hpTxt := i18n.TF("hud.spawn.hp", entry.HpScale*100)
		ui.Label(screen, hpTxt, nameX+40, nameY+14, 50,
			ui.LabelStyle{Font: theme.FontXS, Color: color.RGBA{R: 200, G: 200, B: 210, A: 230}})
	}

	// Hover tooltip（在面板下方）
	if d.HoverIdx >= 0 && d.HoverIdx < len(d.Entries) {
		drawSpawnTooltip(screen, d.Entries[d.HoverIdx], panelX, panelY+panelH+4, panelW)
	}
}

// drawSpawnTooltip 渲染 hover 详情 tooltip。
func drawSpawnTooltip(screen *ebiten.Image, e SpawnEntry, x, y, maxW float32) {
	tipW := float32(260)
	tipH := float32(68)
	tipX := x + (maxW-tipW)/2
	tipY := y

	ui.Panel(screen, tipX, tipY, tipW, tipH, ui.PanelStyle{
		BgColor:     color.RGBA{R: 12, G: 18, B: 35, A: 245},
		BorderColor: color.RGBA{R: 60, G: 80, B: 120, A: 180},
		Radius:      8,
	})

	tx := float64(tipX) + 10
	ty := float64(tipY) + 8

	// 名称 + 标签
	displayName := e.Name
	if e.Label != "" {
		displayName = e.Label
	}
	ui.Label(screen, displayName, tx, ty, 220,
		ui.LabelStyle{Font: theme.FontMD, Bold: true})
	if e.Boss {
		ui.Label(screen, i18n.T("hud.spawn.boss"), tx+100, ty+2, 80,
			ui.LabelStyle{Font: theme.FontSM, Color: color.RGBA{R: 239, G: 68, B: 68, A: 255}})
	}
	ty += 18

	// 属性行
	im := render.GlobalIcons()
	attrX := tx

	if im != nil {
		if img := im.Get("stat-damage"); img != nil {
			draw.Sprite(screen, img, attrX+5, ty+5, 10) //nolint:hud
			attrX += 14
		}
	}
	ui.Label(screen, i18n.TF("hud.spawn.hp", e.HpScale*100), attrX, ty, 60,
		ui.LabelStyle{Font: theme.FontSM, Color: color.RGBA{R: 239, G: 68, B: 68, A: 255}})
	attrX += 60

	if im != nil {
		if img := im.Get("stat-movspd"); img != nil {
			draw.Sprite(screen, img, attrX+5, ty+5, 10) //nolint:hud
			attrX += 14
		}
	}
	ui.Label(screen, i18n.TF("hud.spawn.speed", e.SpeedScale), attrX, ty, 60,
		ui.LabelStyle{Font: theme.FontSM, Color: color.RGBA{R: 74, G: 222, B: 128, A: 255}})
	attrX += 60

	ui.Label(screen, i18n.TF("hud.spawn.radius", e.Radius), attrX, ty, 80,
		ui.LabelStyle{Font: theme.FontSM, Color: color.RGBA{R: 160, G: 160, B: 180, A: 200}})
}

// SpawnMenuHitTest 检测点击了哪个敌人卡片，返回索引或 -1。
func SpawnMenuHitTest(px, py float32, count int) int {
	if count == 0 {
		return -1
	}
	panelX, panelY, panelW, panelH, startY := spawnPanelGeom(count)

	// 快速排除：不在面板区域内直接返回
	if px < panelX || px > panelX+panelW || py < panelY || py > panelY+panelH {
		return -1
	}

	for i := 0; i < count; i++ {
		cx, cy := spawnCardPos(i, panelX, startY)
		if px >= cx && px <= cx+theme.SpawnCardW && py >= cy && py <= cy+theme.SpawnCardH {
			return i
		}
	}
	return -1
}

// SpawnMenuHoverTest 检测鼠标悬停在哪个卡片上。
func SpawnMenuHoverTest(mx, my float32, count int) int {
	return SpawnMenuHitTest(mx, my, count)
}
