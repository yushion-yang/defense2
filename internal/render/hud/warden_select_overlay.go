// warden_select_overlay.go — 战灵选择覆盖层（Stage 内使用）。
// 从 warden_select.go 提取渲染逻辑，作为 HUD 覆盖层组件。
package hud

import (
	"fmt"
	"image/color"
	"strings"

	"defense2/internal/config"
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

// wardenColors 按 key 映射战灵主题色（不在 JSON 中的视觉属性）。
var wardenColors = map[string]color.RGBA{
	"prince":    {R: 255, G: 140, B: 30, A: 255},
	"core":      {R: 60, G: 140, B: 255, A: 255},
	"chain":     {R: 160, G: 80, B: 255, A: 255},
	"skystrike": {R: 80, G: 200, B: 255, A: 255},
	"envoy":     {R: 255, G: 200, B: 60, A: 255},
}

// wardenOrder 战灵在选择列表中的顺序。
var wardenOrder = []string{"prince", "core", "chain", "skystrike", "envoy"}

// categoryName 将 config category 转为显示名。
func categoryName(cat string) string {
	switch cat {
	case "mobile":
		return "移动型"
	case "indirect":
		return "间接型"
	default:
		return cat
	}
}

// BuildWardenOptions 从 wardens.json 配置构建选择列表。
func BuildWardenOptions() []WardenOption {
	cfgs, err := config.LoadWardenConfigs()
	if err != nil {
		return []WardenOption{{Key: "none", Name: "纯塔挑战", Category: "-", Description: "不选择战灵，纯靠塔防御。", Color: color.RGBA{R: 120, G: 120, B: 130, A: 255}}}
	}

	var opts []WardenOption
	for _, key := range wardenOrder {
		c, ok := cfgs[key]
		if !ok {
			continue
		}
		clr := wardenColors[key]
		growthKill := "-"
		if c.GrowthOnKill > 0 {
			growthKill = fmt.Sprintf("+%.0f 强度", c.GrowthOnKill)
		}
		growthWave := "-"
		if c.GrowthOnWaveClear > 0 {
			growthWave = fmt.Sprintf("+%.0f 强度", c.GrowthOnWaveClear)
		}

		// 用配置基础值替换描述中的占位符
		params := buildStaticParams(c)
		opts = append(opts, WardenOption{
			Key:         c.Key,
			Name:        c.Name,
			Category:    categoryName(c.Category),
			Description: c.Description,
			Color:       clr,
			AttackName:  c.AttackName,
			AttackDesc:  replaceParams(c.AttackDesc, params),
			SpecialName: c.SpecialName,
			SpecialDesc: replaceParams(c.SpecialDesc, params),
			Tips:        []string{c.StrengthDesc, c.CounterTip},
			Damage:      fmt.Sprintf("%.0f", c.Damage),
			Interval:    fmt.Sprintf("%.1fs", c.AttackInterval),
			Speed:       fmt.Sprintf("%.0f", c.MoveSpeed),
			AoE:         "-",
			Duration:    "-",
			DoT:         "-",
			GrowthKill:  growthKill,
			GrowthWave:  growthWave,
		})
	}

	// "不选" 选项
	opts = append(opts, WardenOption{
		Key:         "none",
		Name:        "纯塔挑战",
		Category:    "-",
		Description: "不选择战灵，纯靠塔防御。",
		Color:       color.RGBA{R: 120, G: 120, B: 130, A: 255},
	})
	return opts
}

// buildStaticParams 从配置构建占位符参数（选择阶段用基础值，不含强度缩放）。
func buildStaticParams(c config.WardenConfig) map[string]string {
	p := map[string]string{
		"attackInterval": fmt.Sprintf("%.1f", c.AttackInterval),
		"damage":         fmt.Sprintf("%.0f", c.Damage),
		"moveSpeed":      fmt.Sprintf("%.0f", c.MoveSpeed),
		"range":          fmt.Sprintf("%.0f", c.Range),
	}
	// 按 key 补充类型特有参数的默认值（与 Go Init 中的硬编码一致）
	switch c.Key {
	case "prince":
		p["fireballInterval"] = "4"
		p["fireballDmg"] = fmt.Sprintf("%.0f", c.Damage*2) // 200%
		p["trailDuration"] = "2"
		p["trailDps"] = fmt.Sprintf("%.0f", c.Damage*0.5) // 50%
	case "core":
		p["aoeThreshold"] = "4"
		p["execHpPct"] = "20"
	case "chain":
		p["chainRange"] = "150"
		p["bonusPerTower"] = "10"
	case "skystrike":
		p["specialInterval"] = "1"
		p["multiTargets"] = "3"
		p["multiDmg"] = fmt.Sprintf("%.0f", c.Damage*2) // 200%
		p["burstHits"] = "5"
		p["burstDmg"] = fmt.Sprintf("%.0f", c.Damage*1) // 100%
		p["hpTargets"] = "3"
		p["hpPct"] = "10"
	case "envoy":
		p["buffInterval"] = "10"
		p["buffDuration"] = "6"
		p["buffThreshold"] = "100"
		p["buffBonus"] = "0"
		p["permGrant"] = "5"
	}
	return p
}

// replaceParams 替换字符串中的 {key} 占位符。
func replaceParams(s string, params map[string]string) string {
	for k, v := range params {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// WardenOptions 延迟初始化的选项列表（首次访问时从配置构建）。
var wardenOptionsCache []WardenOption

// GetWardenOptions 返回战灵选项列表（懒加载）。
func GetWardenOptions() []WardenOption {
	if wardenOptionsCache == nil {
		wardenOptionsCache = BuildWardenOptions()
	}
	return wardenOptionsCache
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
		o.selectedIdx = len(GetWardenOptions()) - 1 // "none"
		o.confirm()
	}
}

func (o *WardenSelectOverlay) confirm() {
	opt := GetWardenOptions()[o.selectedIdx]
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
	for i := range GetWardenOptions() {
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
	for i, opt := range GetWardenOptions() {
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
	opt := GetWardenOptions()[o.selectedIdx]
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

	contentW := float64(w) - 40 - 80 // 留出右侧精灵预览空间
	for _, line := range wrapText(fm, opt.Description, contentW, theme.FontMD) {
		fm.DrawText(screen, line, px, py, theme.FontMD, theme.TextBody)
		py += 16
	}
	py += 8

	draw.Line(screen, float32(px), float32(py), float32(px)+w-40, float32(py), 1, theme.PanelBorder, false)
	py += 12

	descW := float64(w) - 56 // 描述区可用宽度（留左右 padding）
	if opt.AttackName != "" {
		fm.DrawBoldText(screen, ">> "+opt.AttackName, px, py, theme.FontMD, theme.TextTitle)
		py += 16
		for _, line := range wrapText(fm, opt.AttackDesc, descW, theme.FontSM) {
			fm.DrawText(screen, line, px+16, py, theme.FontSM, theme.TextBody)
			py += 14
		}
		py += 4
	}

	if opt.SpecialName != "" {
		fm.DrawBoldText(screen, ">> "+opt.SpecialName, px, py, theme.FontMD, color.RGBA{R: 255, G: 180, B: 60, A: 255})
		py += 16
		for _, line := range wrapText(fm, opt.SpecialDesc, descW, theme.FontSM) {
			fm.DrawText(screen, line, px+16, py, theme.FontSM, theme.TextBody)
			py += 14
		}
		py += 4
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
