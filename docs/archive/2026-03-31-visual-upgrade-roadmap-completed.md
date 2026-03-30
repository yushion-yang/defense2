# Visual Upgrade Roadmap

> 已完成的部分已移除。参见 MEMORY.md 了解已实现的视觉系统。

## 一、Kage Shader 效果

### 1. Bloom 发光（已有 pipeline，待修复启用）
- 塔攻击时弹射物发光拖尾
- Boss 出场全屏 bloom 脉冲
- 战灵技能释放时局部 bloom
- **状态**：Pipeline 已实现（bloom_extract/blur/combine），但 DrawRectShader 在 Ebitengine v2.9.9 有 runtime crash，`BloomEnabled` 默认 false。待 Ebitengine 升级后启用。

### 2. 屏幕后处理 ✅
- ~~**波纹扭曲**：冰冻塔命中时局部波纹效果~~ — ripple.kage shader + RippleState(4 slot) + scatter 命中触发
- ~~**昼夜循环色调**：暖色→冷色渐变~~ — color_grade.kage 扩展 DayNight uniforms，根据波次进度渐变

### 3. 粒子系统 ✅
- ~~火焰塔：持续火焰粒子~~ — spin_aoe 塔有目标时每 6 帧 emit FireParticles(1)
- ~~冰冻塔：雪花飘落~~ — scatter 塔有目标时每 6 帧 emit IceParticles(1)
- **已有**：DeathBurst/MuzzleFlash/Fire/Ice/GoldCollect/ElectricSparks 预设

## 二、高级渲染技巧

### 4. 拖尾效果 ✅
- ~~小王子冲刺拖尾~~ — WardenState.TrailHistory[8] 环形缓冲 + 渐变半透明圆链
- ~~战斗机甲巡逻轨迹光弧~~ — 同上，所有战灵共享 trail 系统
- **已有**：弹射物 6 帧拖尾

### 5. 精灵动画 ✅
- ~~Boss 呼吸缩放~~ — Boss 1.0+0.04*sin(t*1.8), Elite 1.0+0.02*sin(t*2.2)
- ~~战灵 idle 悬浮动画~~ — bobY=sin(t*2.5)*3 + scale 1.0+0.015*sin(t*2.0)

### 6. 精灵表
- 爆炸特效序列帧（需资产，暂缓）

### 7. 地图视觉深度 ✅
- ~~路径边缘加投影~~ — ThickLine 偏下 2px alpha 30
- ~~塔底部阴影~~ — FilledCircle alpha 30 偏下 35%
- ~~建塔槽位凹陷效果~~ — 深色内圈 + 亮色外环
- ~~远景视差层~~ — DrawParallaxBG 40 颗缓慢飘移微光点

## 三、HUD / UI 提升

### 8. 过渡动画 ✅
- ~~波次开始提示~~ — wave_announce.go
- ~~建塔动画~~ — overshoot bounce + 白色扩散环
- ~~卖塔金币飘升~~ — SpawnGoldText(t.X, t.Y, refund)
- ~~场景淡入淡出~~ — 0.3s fade-to-black

### 9. 伤害数字 ✅
- ~~暴击红色~~ — HitCallback 扩展 crit bool，IsCrit 贯通到 SpawnDamageText
- 治疗绿色区分（待有治疗机制时实现）

### 10. 小地图 ✅
- ~~右下角半透明小地图~~ — hud/minimap.go 120x55，敌人红点/塔蓝点/战灵紫点/路径灰线

### 11. 技能释放 UI ✅
- ~~屏幕边缘闪光~~ — OnActivate 触发白色 HitFlash(0.1s)
- ~~天降打击瞄准圈~~ — Strike 落地前虚线圈扩散
- ~~能量串联连线~~ — ChainState.ChainLinks + 紫色线段渲染

## 四、音视觉联动

### 12. 节奏感 ✅
- ~~攻速闪光~~ — 射击时白色半透明圆闪光，alpha 随 FireAnim 衰减
- ~~连杀提示~~ — MULTI KILL x5 / MEGA KILL x10
- **已有**：Boss 受击 hitStop

## 五、性能优化

### 13. 批量渲染（暂缓）
- Sprite Batch（DrawTriangles + 纹理图集）
- **状态**：粒子系统已用 DrawTriangles 批渲染。敌人/弹射物同屏数量有限（<100），当前性能足够。需确认渲染瓶颈后再实施。

## 剩余未完成

| 项 | 原因 |
|---|---|
| Bloom | Ebitengine v2.9.9 DrawRectShader crash，等升级 |
| 爆炸序列帧 | 需要资产（sprite sheet） |
| 治疗绿色 | 需要治疗机制 |
| Sprite Batch | 性价比低，等性能瓶颈确认 |
