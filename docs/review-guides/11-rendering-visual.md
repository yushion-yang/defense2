# 11 渲染与视觉审核指导

## 审核目标

通过读渲染代码发现视觉 bug——不需要运行游戏或看截图。
检查精灵渲染、动画状态、HP bar、状态效果叠加、HUD 面板布局、文本格式。

## 必读文件

| 文件 | 审核内容 |
|------|---------|
| `internal/render/draw_tower.go` | 塔精灵、fallback 圆形、建塔/卖塔动画、射程圈、蓄能光效 |
| `internal/render/draw_enemy.go` | 敌人精灵、HP bar、dying 动画、状态效果叠加、**怪物能力视觉**（盾牌/脚环/飘字/触发特效） |
| `internal/scene/stage.go` drawEnemyAbilityVFX/drawStrengthDrainLinks | 范围圈、连接线、冲刺速度线 |
| `internal/render/draw_projectile.go` | 弹射物拖尾、穿透视觉 |
| `internal/render/draw_beam.go` | 光束多层 glow + 端点光斑 |
| `internal/render/draw_warden.go` | 战灵精灵、拖尾、射击线、轨道 |
| `internal/render/hud/info_panel.go` | 塔信息面板（属性+能力详情） |
| `internal/render/hud/build_menu.go` | 建塔面板 |
| `internal/render/hud/top_bar.go` | 顶栏（金币/生命/波次/按钮） |
| `internal/render/hud/wave_announce.go` | 波次入场公告 |
| `internal/render/hud/choice_panel.go` | 能力 3 选 1 面板 |
| `internal/render/hud/action_bar.go` | 底部操作栏 |

## 检查项

### A. 精灵与 Fallback

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | PNG 缺失时 fallback | 读 draw_tower.go 的 `img == nil` 分支 | 渲染 fallback 圆形+炮管，不 panic |
| A2 | Fallback 炮管朝向 | 读 fallback 分支是否用 `t.Angle` | 应旋转，不应永远朝上 |
| A3 | 精灵缩放 | 读 `GeoM.Scale` 调用 | scale > 0，不会出现负数翻转 |
| A4 | 战灵 sprite vs fallback 一致性 | 对比两条渲染路径 | bobY 应用方式应一致 |

### B. 动画状态

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | 塔 Animator 共享问题 | 读 animators map 的 key | 应按 InstanceKey 不按 sprKey（同类多塔会互相重置帧） |
| B2 | HitFlash 衰减 | 读 TickStatusEffects 中 HitFlash 递减 | 必然归零，不会卡在非零值 |
| B3 | DyingTimer/DyingDuration 除零 | 读 `progress = 1 - DyingTimer/DyingDuration` | DyingDuration=0 时需守卫 |
| B4 | BuildAnim/SellAnim 边界 | 读 progress 计算 | progress 应 clamp 到 [0,1]，防止负 scale |
| B5 | 敌人出生帧 hit flash | 读 HitFlash 设置处 | Age < 0.1 时应跳过 |

### C. HP Bar

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | HP=0 时 fill width | 读 ratio 计算 | clamp 到 0，不负 |
| C2 | DisplayHP 拖尾 | 读 trailRatio | dying 敌人应跳过拖尾，或 trail ≥ actual ratio |
| C3 | HP > MaxHP 情况 | 读 ratio 计算 | 治疗溢出时 clamp 到 1 |

### D. 状态效果视觉

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | 5 种效果可区分 | 读每种 overlay 的颜色/形状 | slow(蓝)/stun(黄星)/burn(橙)/bleed(红)/root(棕) 各不同 |
| D2 | 状态点区分 | 读 HP bar 上方的状态点渲染 | stun 和 root 不应共用同色同形 |
| D3 | burn 叠加位置 | 读 FilledCircle 的 cy 偏移 | 应在敌人中心，不应偏下看不见 |

### E. HUD 面板

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| E1 | 全中文 | 搜索所有 DrawText 的字面量 | 无英文残留（AoE→范围、HP:→血量:、Lv.→级） |
| E2 | 格式化字符串 | 搜索 fmt.Sprintf / Sprintf 模式 | 无 `{0}`、`%!s`、`<nil>` 占位符 |
| E3 | 三段色规则 | 读 ScaleText/AttrRow 调用 | base=白, scaled>0=绿, scaled<0=红 |
| E4 | 面板高度溢出 | 读 panelY 计算 | InfoPanel 底部应留出 ActionBarH 高度 |
| E5 | 按钮遮挡 | 读 TopBar 按钮布局 | 倒计时文字不应挡住按钮 |
| E6 | 波次倒计时精度 | 读 countdown 显示逻辑 | 应用 Ceil() 不用 int()+1 |
| E7 | ChoicePanel 关闭行为 | 读 click-outside 处理 | 是否应该可关闭？能力选择应否为强制？ |

### F. 怪物能力视觉（新）

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| F1 | 盾牌叠加 | 读 draw_enemy.go 盾牌块 | projectileBlock→shield-white, armorPlating→shield-blue, damageCap→shield-orange |
| F2 | 免疫脚环 | 读 draw_enemy.go 脚环块 | ccImmune→红色（用 hasAbility 而非 IsControlImmune），slowImmune→青色 |
| F3 | 触发特效衰减 | 读 enemy.go TickStatusEffects | BlockFlash/DodgeFlash/ArmorSpark/DamageCapHit/PurgeFlash 全部每帧递减 |
| F4 | 飘字系统 | 读 draw_enemy.go 飘字渲染 | FloatText 上浮+淡出，MISS/CAP/免伤/免疫 各有颜色 |
| F5 | 相位 alpha | 读 draw_enemy.go PhaseActive | body alpha 35% |
| F6 | 沉默隐藏 | 读 draw_enemy.go AbilitySilenced 守卫 | 所有常驻视觉（盾牌/脚环/范围圈）在沉默时隐藏 |
| F7 | 净化白色微光 | 读 draw_enemy.go ControlImmuneTimer | 净化免疫期显示白色微光圈（不显示红色脚环） |

### G. Z-Order

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| G1 | Dying 敌人渲染顺序 | 读 pool.Each 遍历顺序 | dying 应在 active 之下（先渲 dying 再渲 active） |
| G2 | HUD 在特效之上 | 读 Draw 函数调用顺序 | HUD 最后绘制，不被粒子/光效遮挡 |

## 已知问题模式（历史 bug）

- **塔变白圈**: PNG 文件缺失 → fallback 渲染。检查 `assets/sprites/towers/` 是否缺文件
- **怪物出生就播 hit 帧**: HitFlash 在出生帧被设置。已修复（Age < 0.1 保护）
- **战灵出生闪现 (0,0)**: TrailHistory 全 0 初始化。已修复（(0,0) 守卫）

## 跨系统关联

- draw_*.go 读取 core/ 包的 struct 字段 → 字段语义变化需同步更新渲染
- HUD 面板读取 config.GlobalAbilityTable() → 能力配置变化需同步 HUD
- 精灵加载在 render/sprite/ → PNG 命名约定变化需同步 draw_tower/draw_enemy
