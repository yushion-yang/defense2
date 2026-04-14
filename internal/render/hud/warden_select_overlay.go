// warden_select_overlay.go — 战灵选择覆盖层（Stage 内使用）。
// 从 warden_select.go 提取渲染逻辑，作为 HUD 覆盖层组件。
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
	Locked      bool   // 是否锁定
	LockReason  string // 解锁条件文本
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
	woBtnH    = 44.0
	woBtnGap  = 20.0
)

// ── WardenSelectOverlay ──────────────────────────

// WardenSelectOverlay 战灵选择覆盖层（Stage 内弹出）。
type WardenSelectOverlay struct {
	Active       bool
	OnSelect     func(key string)               // 选择回调
	SpriteFunc   func(key string) *ebiten.Image // 战灵精灵获取（由 stage 注入）
	options      []WardenOption                  // 由外部注入的战灵选项列表
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

// Show 激活覆盖层。options 为外部构建的战灵选择数据。
func (o *WardenSelectOverlay) Show(options []WardenOption, onSelect func(string)) {
	o.Active = true
	o.options = options
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
		if idx < len(o.options) && o.options[idx].Locked {
			// 点击锁定战灵：显示解锁条件
			ShowToast(o.options[idx].LockReason)
		} else {
			o.selectedIdx = idx
		}
	}
	// 确认按钮（锁定状态不可确认）
	if o.hitTestBtn(mx, my, 0) {
		if o.selectedIdx < len(o.options) && !o.options[o.selectedIdx].Locked {
			o.confirm()
		}
	}
	// 跳过按钮
	if o.hitTestBtn(mx, my, 1) {
		o.selectedIdx = len(o.options) - 1 // "none"
		o.confirm()
	}
}

func (o *WardenSelectOverlay) confirm() {
	opt := o.options[o.selectedIdx]
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
	for i := range o.options {
		y := woListY + float64(i)*(woListH+woListGap)
		if mx >= woListX && mx <= woListX+woListW && my >= y && my <= y+woListH {
			return i
		}
	}
	return -1
}

func (o *WardenSelectOverlay) hitTestBtn(mx, my float64, btnIdx int) bool {
	sw := float64(theme.CanvasW)
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
	sw := float64(theme.CanvasW)

	// 半透明遮罩
	ui.Overlay(screen, theme.HUDGameOverlay.A)

	// 标题
	ui.Label(screen, i18n.T("hud.wardensel.title"), 0, 20, sw, ui.LabelStyle{
		Font: theme.FontOverlayTitle, Color: theme.TextTitle, Bold: true, Align: ui.AlignCenter,
	})
	ui.Label(screen, i18n.T("hud.wardensel.subtitle"), 0, 50, sw, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted, Align: ui.AlignCenter,
	})

	// 左侧列表
	for i, opt := range o.options {
		x := float32(woListX)
		y := float32(woListY + float64(i)*(woListH+woListGap))
		w := float32(woListW)
		h := float32(woListH)
		selected := i == o.selectedIdx
		hovered := i == o.hoverIdx
		locked := opt.Locked

		bg := theme.PanelBg
		if locked {
			bg = color.RGBA{R: 20, G: 25, B: 40, A: 220}
		} else if selected {
			bg = opt.Color
			bg.A = 180
		} else if hovered {
			bg = color.RGBA{R: 40, G: 50, B: 75, A: 230}
		}

		// 列表项背景 + 选中边框
		borderClr := color.Color(nil)
		if selected && !locked {
			borderClr = opt.Color
		}
		ui.Panel(screen, x, y, w, h, ui.PanelStyle{
			BgColor: bg, BorderColor: borderClr, Radius: 8, BorderWidth: 2,
		})

		nameClr := color.Color(theme.TextBody)
		if locked {
			nameClr = theme.TextLocked
		} else if selected {
			nameClr = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}

		if locked {
			ui.Label(screen, opt.Name, float64(x)+12, float64(y)+8, 150, ui.LabelStyle{
				Font: theme.FontLG, Color: nameClr, Bold: true,
			})
			ui.Label(screen, i18n.T("hud.wardensel.locked"), float64(x)+12, float64(y)+26, 150, ui.LabelStyle{
				Font: theme.FontXS, Color: theme.TextLocked,
			})
		} else {
			ui.Label(screen, opt.Name, float64(x)+12, float64(y)+8, 150, ui.LabelStyle{
				Font: theme.FontLG, Color: nameClr, Bold: true,
			})
			if opt.Category != "-" {
				catClr := color.Color(theme.TextMuted)
				if opt.Category == i18n.T("hud.wardensel.cat_mobile") {
					catClr = color.RGBA{R: 100, G: 200, B: 130, A: 200}
				} else {
					catClr = color.RGBA{R: 200, G: 160, B: 100, A: 200}
				}
				ui.Label(screen, opt.Category, float64(x)+12, float64(y)+26, 150, ui.LabelStyle{
					Font: theme.FontXS, Color: catClr,
				})
			}
		}
	}

	// 右侧详情面板
	opt := o.options[o.selectedIdx]
	o.drawDetail(screen, opt)

	// 底部按钮
	totalBtnW := woBtnW*2 + woBtnGap
	btnStartX := (sw - totalBtnW) / 2

	confirmClr := color.Color(theme.TonePrimary)
	if o.hoverConfirm {
		confirmClr = color.RGBA{R: 60, G: 180, B: 100, A: 255}
	}
	confirmLabel := i18n.TF("hud.wardensel.confirm", opt.Name)
	ui.Button(screen, float32(btnStartX), float32(woBtnY), float32(woBtnW), float32(woBtnH), confirmLabel, ui.ButtonStyle{
		BgColor: confirmClr, TextColor: theme.TextTitle, FontSize: theme.FontLG, Radius: 14, Bold: true,
	})

	skipClr := color.Color(theme.BtnSecondary)
	if o.hoverSkip {
		skipClr = theme.BtnMuted
	}
	skipX := btnStartX + woBtnW + woBtnGap
	ui.Button(screen, float32(skipX), float32(woBtnY), float32(woBtnW), float32(woBtnH), i18n.T("hud.wardensel.skip"), ui.ButtonStyle{
		BgColor: skipClr, TextColor: theme.TextMuted, FontSize: theme.FontMD, Radius: 14,
	})
}

func (o *WardenSelectOverlay) drawDetail(screen *ebiten.Image, opt WardenOption) {
	x := float32(woDetailX)
	y := float32(woDetailY)
	w := float32(woDetailW)
	h := float32(woDetailH)

	// 详情面板背景+边框
	ui.Panel(screen, x, y, w, h, ui.PanelStyle{
		BgColor: theme.PanelBg, BorderColor: theme.PanelBorder, Radius: 12, BorderWidth: 1,
	})

	if opt.Key == "none" {
		ui.Label(screen, i18n.T("hud.wardensel.none_desc"), float64(x), float64(y)+float64(h)/2-10, float64(w), ui.LabelStyle{
			Font: theme.FontLG, Color: theme.TextMuted, Align: ui.AlignCenter,
		})
		return
	}

	if opt.Locked {
		ui.Label(screen, opt.Name, float64(x), float64(y)+float64(h)/2-20, float64(w), ui.LabelStyle{
			Font: theme.FontOverlayName, Color: theme.TextLocked, Bold: true, Align: ui.AlignCenter,
		})
		ui.Label(screen, i18n.T("hud.wardensel.locked"), float64(x), float64(y)+float64(h)/2+10, float64(w), ui.LabelStyle{
			Font: theme.FontLG, Color: theme.TextLocked, Align: ui.AlignCenter,
		})
		if opt.LockReason != "" {
			ui.Label(screen, opt.LockReason, float64(x), float64(y)+float64(h)/2+34, float64(w), ui.LabelStyle{
				Font: theme.FontMD, Color: theme.TextLocked, Align: ui.AlignCenter,
			})
		}
		return
	}

	px := float64(x) + 20
	py := float64(y) + 16

	// 战灵精灵预览（右上角）
	if o.SpriteFunc != nil {
		if img := o.SpriteFunc(opt.Key); img != nil {
			previewX := float64(x) + float64(w) - 84
			previewY := float64(y) + 20
			draw.Sprite(screen, img, previewX, previewY, 64) //nolint:hud
		}
	}

	ui.Label(screen, i18n.TF("hud.wardensel.detail_title", opt.Name), px, py, float64(w)-40-80, ui.LabelStyle{
		Font: theme.FontDetailTitle, Color: theme.TextTitle, Bold: true,
	})
	py += 24

	contentW := float64(w) - 40 - 80 // 留出右侧精灵预览空间
	nLines := ui.Paragraph(screen, opt.Description, px, py, contentW, ui.ParagraphStyle{
		Font: theme.FontMD, Color: theme.TextBody, LineGap: 16 - theme.FontMD,
	})
	py += float64(nLines) * 16
	py += 8

	ui.Divider(screen, float32(px), float32(py), w-40, theme.PanelBorder)
	py += 12

	descW := float64(w) - 56 // 描述区可用宽度（留左右 padding）
	if opt.AttackName != "" {
		ui.Label(screen, ">> "+opt.AttackName, px, py, descW, ui.LabelStyle{
			Font: theme.FontMD, Color: theme.TextTitle, Bold: true,
		})
		py += 16
		nLines = ui.Paragraph(screen, opt.AttackDesc, px+16, py, descW, ui.ParagraphStyle{
			Font: theme.FontSM, Color: theme.TextBody, LineGap: 14 - theme.FontSM,
		})
		py += float64(nLines) * 14
		py += 4
	}

	if opt.SpecialName != "" {
		ui.Label(screen, ">> "+opt.SpecialName, px, py, descW, ui.LabelStyle{
			Font: theme.FontMD, Color: color.RGBA{R: 255, G: 180, B: 60, A: 255}, Bold: true,
		})
		py += 16
		nLines = ui.Paragraph(screen, opt.SpecialDesc, px+16, py, descW, ui.ParagraphStyle{
			Font: theme.FontSM, Color: theme.TextBody, LineGap: 14 - theme.FontSM,
		})
		py += float64(nLines) * 14
		py += 4
	}

	for _, tip := range opt.Tips {
		ui.Label(screen, tip, px+16, py, descW-16, ui.LabelStyle{
			Font: theme.FontXS, Color: theme.TextMuted,
		})
		py += 14
	}

	py += 10
	ui.Divider(screen, float32(px), float32(py), w-40, theme.PanelBorder)
	py += 12

	ui.Label(screen, i18n.T("hud.wardensel.base_stats"), px, py, descW, ui.LabelStyle{
		Font: theme.FontMD, Color: theme.TextTitle, Bold: true,
	})
	py += 20

	attrClr := theme.TextBody
	valClr := theme.TextTitle
	colW := 150.0

	attrs := []struct{ iconName, label, value string }{
		{"stat-damage", i18n.T("hud.stat.damage"), opt.Damage},
		{"stat-atkspd", i18n.T("hud.wardensel.interval"), opt.Interval},
		{"stat-movspd", i18n.T("hud.wardensel.speed"), opt.Speed},
		{"stat-splash", i18n.T("hud.stat.range"), opt.AoE},
		{"stat-atkspd", i18n.T("hud.wardensel.duration"), opt.Duration},
		{"burn", i18n.T("hud.wardensel.dot"), opt.DoT},
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
				draw.Sprite(screen, img, ax+5, ay+5, 10) //nolint:hud
			}
		}
		ui.Label(screen, a.label, ax+14, ay, 30, ui.LabelStyle{
			Font: theme.FontSM, Color: attrClr,
		})
		ui.Label(screen, a.value, ax+44, ay, 100, ui.LabelStyle{
			Font: theme.FontSM, Color: valClr, Bold: true,
		})
	}

	py += 40
	ui.Label(screen, i18n.T("hud.wardensel.growth"), px, py, descW, ui.LabelStyle{
		Font: theme.FontMD, Color: theme.TextTitle, Bold: true,
	})
	py += 18
	growthClr := color.RGBA{R: 76, G: 175, B: 80, A: 255}
	growthText := ""
	if opt.GrowthKill != "-" && opt.GrowthKill != "" {
		growthText += i18n.TF("hud.wardensel.growth_kill", opt.GrowthKill)
	}
	if opt.GrowthWave != "-" && opt.GrowthWave != "" {
		if growthText != "" {
			growthText += "    "
		}
		growthText += i18n.TF("hud.wardensel.growth_wave", opt.GrowthWave)
	}
	if growthText != "" {
		ui.Label(screen, growthText, px, py, descW, ui.LabelStyle{
			Font: theme.FontSM, Color: growthClr,
		})
	}
}
