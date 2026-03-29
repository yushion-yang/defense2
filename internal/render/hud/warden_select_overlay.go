// warden_select_overlay.go — 战灵选择覆盖层（Stage 内使用）。
// 从 warden_select.go 提取渲染逻辑，作为 HUD 覆盖层组件。
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

// ── 战灵数据 ────────────────────────────────────

// WardenOption 描述一个可选战灵。
type WardenOption struct {
	Key         string
	Name        string
	Category    string // "移动型" / "间接型" / "-"
	Description string
	Color       color.RGBA
	AttackName  string
	AttackDesc  string
	SpecialName string
	SpecialDesc string
	Tips        []string
	Damage      string
	Interval    string
	Speed       string
	AoE         string
	Duration    string
	DoT         string
	GrowthKill  string
	GrowthWave  string
}

// WardenOptions 所有可选战灵（5 个战灵 + 1 个"不选"）。
var WardenOptions = []WardenOption{
	{
		Key: "prince", Name: "火灵", Category: "移动型",
		Description: "围绕敌群轨道飞行并攻击，定时从虚空召唤火球冲撞敌群。",
		Color:       color.RGBA{R: 255, G: 140, B: 30, A: 255},
		AttackName:  "轨道射击", AttackDesc: "围绕敌群轨道飞行，攻击最近敌人。",
		SpecialName: "虚空火球", SpecialDesc: "每4s召唤火球冲向敌群，穿透伤害30，留下火焰痕迹。",
		Tips:   []string{"强度 → 攻击力 + 火球伤害增强。", "适合密集小怪波次。"},
		Damage: "15", Interval: "1.2s", Speed: "350", AoE: "-", Duration: "-", DoT: "10/s",
		GrowthKill: "+1 强度", GrowthWave: "+5 强度",
	},
	{
		Key: "core", Name: "机甲", Category: "移动型",
		Description: "单个超强移动实体，在敌群周围巡逻，智能切换攻击模式。",
		Color:       color.RGBA{R: 60, G: 140, B: 255, A: 255},
		AttackName:  "巡逻射击", AttackDesc: "绕敌群中心巡逻，自动锁定高威胁目标。",
		SpecialName: "智能模式", SpecialDesc: "4+敌人=AoE / <30%HP=斩杀 / Boss=全力。",
		Tips:   []string{"强度 → 伤害 + 攻速增强。", "适合精英/Boss波次。"},
		Damage: "25", Interval: "1.2s", Speed: "360", AoE: "-", Duration: "-", DoT: "-",
		GrowthKill: "+1 强度", GrowthWave: "+10 强度",
	},
	{
		Key: "chain", Name: "聚能", Category: "间接型",
		Description: "启用全场塔串联（+10伤害/塔），定时发射能量弹。",
		Color:       color.RGBA{R: 160, G: 80, B: 255, A: 255},
		AttackName:  "能量弹", AttackDesc: "定时向随机敌人发射能量弹造成伤害。",
		SpecialName: "串联体", SpecialDesc: "被动：全场塔 +10 伤害加成。",
		Tips:   []string{"塔越多，收益越高。", "适合塔数量多的防线。"},
		Damage: "15", Interval: "2s", Speed: "-", AoE: "-", Duration: "-", DoT: "-",
		GrowthKill: "-", GrowthWave: "+8 强度",
	},
	{
		Key: "skystrike", Name: "水灵", Category: "间接型",
		Description: "定时对敌群最密集区域发动AoE水灵打击。",
		Color:       color.RGBA{R: 80, G: 200, B: 255, A: 255},
		AttackName:  "水灵打击", AttackDesc: "锁定敌群最密集位置，释放范围打击。",
		SpecialName: "智能瞄准", SpecialDesc: "自动选择敌人最多的区域。",
		Tips:   []string{"AoE 半径随强度增长。", "适合拥堵路径节点。"},
		Damage: "40", Interval: "5s", Speed: "-", AoE: "60", Duration: "-", DoT: "-",
		GrowthKill: "-", GrowthWave: "+10 强度",
	},
	{
		Key: "envoy", Name: "金灵", Category: "移动型",
		Description: "围绕敌群轨道飞行并攻击，定时为最佳塔施加增强 buff。",
		Color:       color.RGBA{R: 180, G: 120, B: 255, A: 255},
		AttackName:  "轨道射击", AttackDesc: "围绕敌群轨道飞行，攻击最近敌人。",
		SpecialName: "增强光环", SpecialDesc: "每5s为射程内敌人最多的塔施加+8战力buff(4s)。",
		Tips:   []string{"攻防兼备，适合需要塔增强的阵型。"},
		Damage: "12", Interval: "1.5s", Speed: "320", AoE: "-", Duration: "-", DoT: "-",
		GrowthKill: "-", GrowthWave: "+5 强度",
	},
	{
		Key: "none", Name: "纯塔挑战", Category: "-",
		Description: "不选择战灵，纯靠塔防御。",
		Color:       color.RGBA{R: 120, G: 120, B: 130, A: 255},
	},
}

// ── 布局常量 ────────────────────────────────────

const (
	woListX   = 40.0
	woListY   = 80.0
	woListW   = 180.0
	woListH   = 44.0
	woListGap = 4.0
	woDetailX = 250.0
	woDetailY = 80.0
	woDetailW = 910.0
	woDetailH = 380.0
	woBtnY    = 480.0
	woBtnW    = 200.0
	woBtnH    = 36.0
	woBtnGap  = 20.0
)

// ── WardenSelectOverlay ──────────────────────────

// WardenSelectOverlay 战灵选择覆盖层（Stage 内弹出）。
type WardenSelectOverlay struct {
	Active       bool
	OnSelect     func(key string)               // 选择回调
	SpriteFunc   func(key string) *ebiten.Image // 战灵精灵获取（由 stage 注入）
	selectedIdx  int
	hoverIdx     int
	hoverConfirm bool
	hoverSkip    bool
}

// NewWardenSelectOverlay 创建战灵选择覆盖层。
func NewWardenSelectOverlay() *WardenSelectOverlay {
	return &WardenSelectOverlay{
		selectedIdx: 0,
		hoverIdx:    -1,
	}
}

// Show 激活覆盖层。
func (o *WardenSelectOverlay) Show(onSelect func(string)) {
	o.Active = true
	o.OnSelect = onSelect
	o.selectedIdx = 0
	o.hoverIdx = -1
}

// Close 关闭覆盖层。
func (o *WardenSelectOverlay) Close() {
	o.Active = false
	o.OnSelect = nil
}

// Update 处理交互。由 stage 调用，传入鼠标坐标和是否有点击。
func (o *WardenSelectOverlay) Update(mx, my float64, clicked bool) {
	if !o.Active {
		return
	}

	// 悬停检测
	o.hoverIdx = o.hitTestList(mx, my)
	o.hoverConfirm = o.hitTestBtn(mx, my, 0)
	o.hoverSkip = o.hitTestBtn(mx, my, 1)

	if !clicked {
		return
	}

	// 列表选择
	if idx := o.hitTestList(mx, my); idx >= 0 {
		o.selectedIdx = idx
	}
	// 确认按钮
	if o.hitTestBtn(mx, my, 0) {
		o.confirm()
	}
	// 跳过按钮
	if o.hitTestBtn(mx, my, 1) {
		o.selectedIdx = len(WardenOptions) - 1 // "none"
		o.confirm()
	}
}

func (o *WardenSelectOverlay) confirm() {
	opt := WardenOptions[o.selectedIdx]
	key := opt.Key
	if key == "none" {
		key = ""
	}
	if o.OnSelect != nil {
		o.OnSelect(key)
	}
	o.Close()
}

func (o *WardenSelectOverlay) hitTestList(mx, my float64) int {
	for i := range WardenOptions {
		y := woListY + float64(i)*(woListH+woListGap)
		if mx >= woListX && mx <= woListX+woListW && my >= y && my <= y+woListH {
			return i
		}
	}
	return -1
}

func (o *WardenSelectOverlay) hitTestBtn(mx, my float64, btnIdx int) bool {
	sw := float64(game.ScreenWidth)
	totalW := woBtnW*2 + woBtnGap
	startX := (sw - totalW) / 2
	bx := startX + float64(btnIdx)*(woBtnW+woBtnGap)
	return mx >= bx && mx <= bx+woBtnW && my >= woBtnY && my <= woBtnY+woBtnH
}

// ── Draw ────────────────────────────────────────

// Draw 绘制覆盖层。
func (o *WardenSelectOverlay) Draw(screen *ebiten.Image) {
	if !o.Active {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// 半透明遮罩
	draw.FilledRect(screen, 0, 0, float32(sw), float32(sh), theme.HUDGameOverlay, false)

	// 标题
	fm.DrawCenteredBoldText(screen, "选择你的战灵", sw/2, 20, 22, theme.TextTitle)
	fm.DrawCenteredText(screen, "选择最适合的战灵 — 或不选，挑战纯塔模式", sw/2, 50, 11, theme.TextMuted)

	// 左侧列表
	for i, opt := range WardenOptions {
		x := float32(woListX)
		y := float32(woListY + float64(i)*(woListH+woListGap))
		w := float32(woListW)
		h := float32(woListH)
		selected := i == o.selectedIdx
		hovered := i == o.hoverIdx

		bg := theme.PanelBg
		if selected {
			bg = opt.Color
			bg.A = 180
		} else if hovered {
			bg = color.RGBA{R: 40, G: 50, B: 75, A: 230}
		}
		draw.RoundRect(screen, x, y, w, h, 8, bg)

		if selected {
			draw.StrokeRoundRect(screen, x, y, w, h, 8, 2, opt.Color)
		}

		nameClr := theme.TextBody
		if selected {
			nameClr = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		fm.DrawBoldText(screen, opt.Name, float64(x)+12, float64(y)+8, theme.FontLG, nameClr)

		if opt.Category != "-" {
			catClr := theme.TextMuted
			if opt.Category == "移动型" {
				catClr = color.RGBA{R: 100, G: 200, B: 130, A: 200}
			} else {
				catClr = color.RGBA{R: 200, G: 160, B: 100, A: 200}
			}
			fm.DrawText(screen, opt.Category, float64(x)+12, float64(y)+26, theme.FontXS, catClr)
		}
	}

	// 右侧详情面板
	opt := WardenOptions[o.selectedIdx]
	o.drawDetail(screen, fm, opt)

	// 底部按钮
	totalBtnW := woBtnW*2 + woBtnGap
	btnStartX := (sw - totalBtnW) / 2

	confirmClr := theme.TonePrimary
	if o.hoverConfirm {
		confirmClr = color.RGBA{R: 60, G: 180, B: 100, A: 255}
	}
	draw.RoundRect(screen, float32(btnStartX), float32(woBtnY), float32(woBtnW), float32(woBtnH), 14, confirmClr)
	confirmLabel := fmt.Sprintf("选择 %s", opt.Name)
	fm.DrawCenteredBoldText(screen, confirmLabel, btnStartX+woBtnW/2, woBtnY+9, theme.FontLG, theme.TextTitle)

	skipClr := theme.BtnSecondary
	if o.hoverSkip {
		skipClr = theme.BtnMuted
	}
	skipX := btnStartX + woBtnW + woBtnGap
	draw.RoundRect(screen, float32(skipX), float32(woBtnY), float32(woBtnW), float32(woBtnH), 14, skipClr)
	fm.DrawCenteredText(screen, "不选（纯塔挑战）", skipX+woBtnW/2, woBtnY+10, theme.FontMD, theme.TextMuted)
}

func (o *WardenSelectOverlay) drawDetail(screen *ebiten.Image, fm *render.FontManager, opt WardenOption) {
	x := float32(woDetailX)
	y := float32(woDetailY)
	w := float32(woDetailW)
	h := float32(woDetailH)

	draw.RoundRect(screen, x, y, w, h, 12, theme.PanelBg)
	draw.StrokeRoundRect(screen, x, y, w, h, 12, 1, theme.PanelBorder)

	if opt.Key == "none" {
		fm.DrawCenteredText(screen, "不使用战灵，纯塔防御模式", float64(x)+float64(w)/2, float64(y)+float64(h)/2-10, theme.FontLG, theme.TextMuted)
		return
	}

	px := float64(x) + 20
	py := float64(y) + 16

	// 战灵精灵预览（右上角）
	if o.SpriteFunc != nil {
		if img := o.SpriteFunc(opt.Key); img != nil {
			previewX := float64(x) + float64(w) - 84
			previewY := float64(y) + 20
			draw.Sprite(screen, img, previewX, previewY, 64)
		}
	}

	fm.DrawBoldText(screen, fmt.Sprintf("战灵 · %s", opt.Name), px, py, 18, theme.TextTitle)
	py += 24

	fm.DrawText(screen, opt.Description, px, py, theme.FontMD, theme.TextBody)
	py += 28

	draw.Line(screen, float32(px), float32(py), float32(px)+w-40, float32(py), 1, theme.PanelBorder, false)
	py += 12

	if opt.AttackName != "" {
		fm.DrawBoldText(screen, ">> "+opt.AttackName, px, py, theme.FontMD, theme.TextTitle)
		py += 16
		fm.DrawText(screen, opt.AttackDesc, px+16, py, theme.FontSM, theme.TextBody)
		py += 18
	}

	if opt.SpecialName != "" {
		fm.DrawBoldText(screen, ">> "+opt.SpecialName, px, py, theme.FontMD, color.RGBA{R: 255, G: 180, B: 60, A: 255})
		py += 16
		fm.DrawText(screen, opt.SpecialDesc, px+16, py, theme.FontSM, theme.TextBody)
		py += 18
	}

	for _, tip := range opt.Tips {
		fm.DrawText(screen, tip, px+16, py, theme.FontXS, theme.TextMuted)
		py += 14
	}

	py += 10
	draw.Line(screen, float32(px), float32(py), float32(px)+w-40, float32(py), 1, theme.PanelBorder, false)
	py += 12

	fm.DrawBoldText(screen, "Lv.1 属性", px, py, theme.FontMD, theme.TextTitle)
	py += 20

	attrClr := theme.TextBody
	valClr := theme.TextTitle
	colW := 150.0

	attrs := []struct{ iconName, label, value string }{
		{"stat-damage", "伤害", opt.Damage},
		{"stat-atkspd", "间隔", opt.Interval},
		{"stat-movspd", "速度", opt.Speed},
		{"stat-splash", "AoE", opt.AoE},
		{"stat-atkspd", "持续", opt.Duration},
		{"burn", "DoT", opt.DoT},
	}

	im := render.GlobalIcons()
	for i, a := range attrs {
		if a.value == "-" || a.value == "" {
			continue
		}
		col := i % 3
		row := i / 3
		ax := px + float64(col)*colW
		ay := py + float64(row)*16
		if im != nil {
			if img := im.Get(a.iconName); img != nil {
				draw.Sprite(screen, img, ax+5, ay+5, 10)
			}
		}
		fm.DrawText(screen, a.label, ax+14, ay, theme.FontSM, attrClr)
		fm.DrawBoldText(screen, a.value, ax+44, ay, theme.FontSM, valClr)
	}

	py += 40
	fm.DrawBoldText(screen, "成长", px, py, theme.FontMD, theme.TextTitle)
	py += 18
	growthClr := color.RGBA{R: 76, G: 175, B: 80, A: 255}
	growthText := ""
	if opt.GrowthKill != "-" && opt.GrowthKill != "" {
		growthText += fmt.Sprintf("击杀 %s", opt.GrowthKill)
	}
	if opt.GrowthWave != "-" && opt.GrowthWave != "" {
		if growthText != "" {
			growthText += "    "
		}
		growthText += fmt.Sprintf("通波 %s", opt.GrowthWave)
	}
	if growthText != "" {
		fm.DrawText(screen, growthText, px, py, theme.FontSM, growthClr)
	}
}
