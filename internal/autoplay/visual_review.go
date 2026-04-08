// visual_review.go — 视觉审查清单 + review manifest 生成。
//
// 设计理念:
//   - 每张截图配一组具体的 Yes/No 检查问题（不是"泛泛找 bug"）
//   - 问题来自已知 bug record，针对性强
//   - 输出 visual_review.md，可直接喂给 Claude Vision 或人工快速过
//   - 截图按类别分组，每类只需几张代表性截图
package autoplay

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScreenshotCategory 截图类别。
type ScreenshotCategory string

const (
	CatTowerSprite  ScreenshotCategory = "tower"   // 塔外观
	CatTowerAttack  ScreenshotCategory = "attack"  // 塔攻击特效
	CatEnemySprite  ScreenshotCategory = "enemy"   // 敌人外观
	CatStatusEffect ScreenshotCategory = "status"  // 状态效果
	CatHUD          ScreenshotCategory = "hud"     // HUD 面板
	CatWarden       ScreenshotCategory = "warden"  // 战灵
	CatScene        ScreenshotCategory = "scene"   // 场景全局
)

// CheckItem 单条检查项。
type CheckItem struct {
	Question string // 具体的 Yes/No 问题
	BugRef   string // 对应的 bug 来源（可选）
}

// ReviewEntry 一张截图的审查条目。
type ReviewEntry struct {
	Filename string             // 截图文件名
	Category ScreenshotCategory // 类别
	Context  string             // 截图时机描述（给审查者的上下文）
	Checks   []CheckItem        // 检查项列表
}

// categoryChecks 每个类别的通用检查项（所有该类截图都检查）。
var categoryChecks = map[ScreenshotCategory][]CheckItem{
	CatTowerSprite: {
		{Question: "塔是否渲染为正常的精灵图像（而非纯色圆形/白色圆形 fallback）？", BugRef: "炮塔变白圈"},
		{Question: "塔的精灵是否清晰、无明显缺失部件？"},
	},
	CatTowerAttack: {
		{Question: "塔是否有可见的攻击动画或特效（粒子、光束、弹射物等）？", BugRef: "旋风塔无攻击展示"},
		{Question: "弹射物/光束是否从塔的位置发出、指向敌人？"},
		{Question: "塔周围是否有异常的白色圆环 artifact？", BugRef: "旋刃白圈"},
		{Question: "是否能看到伤害浮字（数字）出现在敌人身上？", BugRef: "电磁炮无伤害浮字"},
	},
	CatEnemySprite: {
		{Question: "敌人是否渲染为正常的精灵图像（而非占位符）？"},
		{Question: "敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？", BugRef: "怪物出现就播 hit 帧"},
		{Question: "敌人周围是否有异常的白色方框/光晕？", BugRef: "hit帧白方框"},
		{Question: "敌人的血条是否正常显示在头顶？"},
	},
	CatStatusEffect: {
		{Question: "状态效果的视觉标记是否可见（颜色叠加/粒子/图标）？"},
		{Question: "受影响的敌人是否与未受影响的敌人有明显视觉区分？"},
	},
	CatHUD: {
		{Question: "面板文字是否全部为中文（无英文残留如 AoE/DoT/HP:/Lv.）？", BugRef: "HUD 英文残留"},
		{Question: "面板布局是否完整、无文字被截断或重叠？", BugRef: "造塔按钮被挡"},
		{Question: "数值颜色是否符合三段规则（白=基础, 绿=增益, 红=减益）？", BugRef: "属性颜色不对"},
		{Question: "是否存在占位符文本（如 {0}、N/A、undefined）？", BugRef: "hover 占位符"},
	},
	CatWarden: {
		{Question: "战灵精灵是否正常渲染（非占位符）？"},
		{Question: "战灵的朝向是否与其移动方向一致？", BugRef: "战灵朝向不对"},
		{Question: "战灵是否有可见的攻击动画/特效？"},
	},
	CatScene: {
		{Question: "整体画面是否正常渲染（无全黑/全白/严重撕裂）？"},
		{Question: "UI 元素（顶栏、按钮）是否可见且布局正常？"},
	},
}

// specificChecks 特定截图的额外检查项（按文件名前缀匹配）。
var specificChecks = map[string][]CheckItem{
	"attack_spin_aoe": {
		{Question: "旋风/范围攻击是否有可见的旋转或扩散特效？", BugRef: "旋风塔无攻击展示"},
	},
	"attack_scatter": {
		{Question: "散弹是否发射了多颗弹丸（而非单颗）？", BugRef: "散弹子弹数不增加"},
	},
	"attack_charge": {
		{Question: "蓄能过程是否有视觉反馈（充能条/光效）？"},
	},
	"attack_laser": {
		{Question: "激光束是否从塔延伸到目标敌人？"},
	},
	"hud_towerSel": {
		{Question: "能力描述是否格式正确（base+scaled=total, 无多余的 0）？", BugRef: "能力描述加 0"},
		{Question: "是否只显示该塔已获取的能力（无冗余空槽位）？", BugRef: "HUD 冗余槽位"},
	},
	"hud_buildMenu": {
		{Question: "各塔的图标是否为正常精灵（非旧图标/占位符）？"},
		{Question: "价格和名称标签是否清晰可见？"},
	},
	"warden_active": {
		{Question: "战灵是否已离开原点、位于地图上的合理位置？", BugRef: "战灵出生闪现"},
	},
	"boss_appear": {
		{Question: "Boss 敌人是否体型明显大于普通敌人？"},
		{Question: "Boss 是否有独特的视觉标识（颜色/光环）？"},
	},
}

// BuildReviewEntries 从已截图文件列表构建审查条目。
func BuildReviewEntries(capturedFiles []string) []ReviewEntry {
	var entries []ReviewEntry
	for _, fname := range capturedFiles {
		cat := classifyScreenshot(fname)
		entry := ReviewEntry{
			Filename: fname,
			Category: cat,
			Context:  describeContext(fname),
			Checks:   make([]CheckItem, 0),
		}

		// 添加类别通用检查项
		if checks, ok := categoryChecks[cat]; ok {
			entry.Checks = append(entry.Checks, checks...)
		}

		// 添加特定文件的额外检查项
		base := strings.TrimSuffix(fname, filepath.Ext(fname))
		for prefix, checks := range specificChecks {
			if strings.HasPrefix(base, prefix) {
				entry.Checks = append(entry.Checks, checks...)
				break
			}
		}

		// 跳过无检查项的截图（如 start.png 不需要特别审查）
		if len(entry.Checks) == 0 {
			continue
		}

		entries = append(entries, entry)
	}
	return entries
}

// WriteVisualReview 生成 visual_review.md 文件。
func WriteVisualReview(entries []ReviewEntry, dir string) error {
	if len(entries) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	var b strings.Builder
	b.WriteString("# Visual Review Manifest\n\n")
	b.WriteString("> 使用方法: 将本文件和同目录的截图一起发给 Claude Vision，\n")
	b.WriteString("> 让它逐张按检查项回答 Yes/No 并标记发现的问题。\n\n")
	b.WriteString(fmt.Sprintf("共 %d 张截图待审查。\n\n", len(entries)))

	// 按类别分组
	grouped := make(map[ScreenshotCategory][]ReviewEntry)
	catOrder := []ScreenshotCategory{CatTowerSprite, CatTowerAttack, CatEnemySprite, CatStatusEffect, CatHUD, CatWarden, CatScene}
	for _, e := range entries {
		grouped[e.Category] = append(grouped[e.Category], e)
	}

	catNames := map[ScreenshotCategory]string{
		CatTowerSprite:  "塔外观",
		CatTowerAttack:  "塔攻击特效",
		CatEnemySprite:  "敌人外观",
		CatStatusEffect: "状态效果",
		CatHUD:          "HUD 面板",
		CatWarden:       "战灵",
		CatScene:        "场景全局",
	}

	for _, cat := range catOrder {
		group, ok := grouped[cat]
		if !ok || len(group) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("## %s (%d)\n\n", catNames[cat], len(group)))

		for _, entry := range group {
			b.WriteString(fmt.Sprintf("### %s\n", entry.Filename))
			b.WriteString(fmt.Sprintf("![%s](%s)\n\n", entry.Filename, entry.Filename))
			if entry.Context != "" {
				b.WriteString(fmt.Sprintf("**场景**: %s\n\n", entry.Context))
			}
			b.WriteString("**检查项**:\n")
			for i, c := range entry.Checks {
				ref := ""
				if c.BugRef != "" {
					ref = fmt.Sprintf(" _(ref: %s)_", c.BugRef)
				}
				b.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, c.Question, ref))
			}
			b.WriteString("\n")
		}
	}

	path := filepath.Join(dir, "visual_review.md")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// classifyScreenshot 根据文件名推断类别。
func classifyScreenshot(fname string) ScreenshotCategory {
	base := strings.TrimSuffix(fname, filepath.Ext(fname))
	switch {
	case strings.HasPrefix(base, "tower_"):
		return CatTowerSprite
	case strings.HasPrefix(base, "attack_"):
		return CatTowerAttack
	case strings.HasPrefix(base, "enemy_"), strings.HasPrefix(base, "boss_"):
		return CatEnemySprite
	case strings.HasPrefix(base, "status_"):
		return CatStatusEffect
	case strings.HasPrefix(base, "hud_"):
		return CatHUD
	case strings.HasPrefix(base, "warden"):
		return CatWarden
	default:
		return CatScene
	}
}

// describeContext 为截图提供上下文描述。
func describeContext(fname string) string {
	base := strings.TrimSuffix(fname, filepath.Ext(fname))
	switch {
	case strings.HasPrefix(base, "tower_"):
		key := strings.TrimPrefix(base, "tower_")
		return fmt.Sprintf("塔类型 '%s' 首次建造时的外观", key)
	case strings.HasPrefix(base, "attack_"):
		style := strings.TrimPrefix(base, "attack_")
		return fmt.Sprintf("攻击方式 '%s' 首次开火时的战斗画面", style)
	case strings.HasPrefix(base, "enemy_"):
		arch := strings.TrimPrefix(base, "enemy_")
		return fmt.Sprintf("敌人原型 '%s' 首次出现时的行走状态", arch)
	case strings.HasPrefix(base, "boss_appear"):
		return "Boss 敌人首次出现的画面"
	case strings.HasPrefix(base, "status_"):
		fx := strings.TrimPrefix(base, "status_")
		return fmt.Sprintf("状态效果 '%s' 首次施加时的画面", fx)
	case strings.HasPrefix(base, "hud_towerSel"):
		return "选中塔后的信息面板（属性+能力详情）"
	case strings.HasPrefix(base, "hud_buildMenu"):
		return "建造面板打开时的塔列表"
	case strings.HasPrefix(base, "hud_wardenSelect"):
		return "战灵选择覆盖层"
	case strings.HasPrefix(base, "hud_paused"):
		return "暂停菜单"
	case strings.HasPrefix(base, "warden_active"):
		return "战灵激活后的首次画面"
	case strings.HasPrefix(base, "warden_combat"):
		return "战灵正在战斗中（有敌人在附近）"
	case base == "result":
		return "游戏结算画面"
	case base == "start":
		return "游戏开始画面"
	default:
		return ""
	}
}
