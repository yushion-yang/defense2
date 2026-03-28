# Visual Upgrade Roadmap — 超越 JS Canvas 版

> 目标：充分发挥 Ebitengine GPU 能力，让 Go 版视觉效果**超越**原 JS Canvas 2D 版。

## 一、Kage Shader 效果（Canvas 2D 做不到）

### 1. Bloom 发光
- 塔攻击时弹射物发光拖尾
- Boss 出场全屏 bloom 脉冲
- 战灵技能释放时局部 bloom
- **实现**：render-to-texture → 降采样 → 高斯模糊 → additive blend 回主画面

### 2. 屏幕后处理
- **径向模糊**：Boss 出场/死亡时从中心向外模糊
- **色调映射**：昼夜循环（暖色→冷色渐变）、受伤时屏幕红闪
- **波纹扭曲**：冰冻塔命中时局部波纹效果
- **暗角（Vignette）**：画面边缘自然变暗，聚焦中心战场

### 3. 粒子系统（Shader 加速）
- 敌人死亡：像素碎片爆散
- 塔射击：枪口火花
- 火焰塔：持续火焰粒子
- 冰冻塔：雪花飘落
- 金币掉落：闪光粒子飞向 HUD 金币栏
- **实现**：用 instanced quad + shader 批量渲染数千粒子，零 GC 压力

### 4. 动态光照
- 每个塔有点光源，颜色匹配塔类型（蓝=冰，红=火，黄=电）
- 弹射物飞行时照亮周围地面
- Boss 自带脉冲光环
- **实现**：法线贴图 + 多光源 shader，或简化版 radial gradient overlay

## 二、高级渲染技巧

### 5. 拖尾效果（Motion Trail）
- 弹射物飞行留下渐隐尾迹（存最近 N 帧位置，逐帧降 alpha）
- 小王子冲刺拖尾（当前是直线，升级为渐变半透明带）
- 战斗机甲巡逻轨迹光弧
- **实现**：环形缓冲区存历史位置 + 逐段 alpha 衰减绘制

### 6. 精灵动画
- 塔攻击时旋转/后座动画（DrawImageOptions.GeoM.Rotate）
- 敌人行走摆动（sin 波微偏移）
- Boss 呼吸缩放（周期性 Scale 变化）
- 战灵 idle 悬浮动画（上下浮动 + 缩放脉冲）
- **实现**：纯 GeoM 变换，零额外资源

### 7. 精灵表（Sprite Sheet）
- 将多帧动画打包为单张大图
- 塔的攻击动画（3-4 帧循环）
- 敌人的行走动画
- 爆炸特效序列帧
- **实现**：`ebiten.Image.SubImage()` 切片 + 帧计数器

### 8. 地图视觉深度
- 路径边缘加投影（路径下方偏移 2px 的深色副本）
- 塔底部阴影（半透明椭圆）
- 建塔槽位凹陷效果（内阴影圆环）
- 远景视差层（轻微的背景星空/云层随时间缓慢移动）

## 三、HUD / UI 提升

### 9. 过渡动画
- 场景切换：淡入淡出（两帧 render-to-texture + alpha 交叉混合）
- 波次开始：屏幕上方滑入 "Wave 3" 大字，1s 后滑出
- 建塔成功：塔位闪烁 + 放大回弹动画
- 卖塔：金币数字飘升动画

### 10. 伤害数字
- 命中时在敌人头顶弹出伤害数字（白色/暴击红色/治疗绿色）
- 数字向上漂浮 + 缩放 + alpha 衰减
- **实现**：简单的浮动文本池，每帧更新 Y 和 alpha

### 11. 小地图 / 战场概览
- 右下角半透明小地图，显示敌人位置（红点）和塔位置（蓝点）
- 点击小地图可以快速定位

### 12. 技能释放 UI
- 战灵技能释放时，屏幕边缘闪光提示
- 天降打击：目标区域先出现瞄准圈（0.5s 预警），再落下
- 能量串联：塔之间画出能量连线（紫色脉冲线段）

## 四、音视觉联动

### 13. 屏幕震动
- Boss 死亡 / 天降打击 / 大范围 AoE 时短暂屏幕抖动
- **实现**：Draw 时 GeoM.Translate 加随机偏移（±3px，衰减 0.3s）

### 14. 节奏感
- 攻速快的塔配合短促音效 + 微闪光
- Boss 受击时全屏微顿（hitStop，暂停 2-3 帧）
- 连杀提示（3/5/10 连杀文字 + 音效升调）

## 五、性能优化

### 15. 批量渲染
- 将同类精灵合并为单次 DrawImage 调用（Sprite Batch）
- 敌人/弹射物数量大时避免逐个 Draw
- **实现**：`ebiten.DrawTriangles` + 纹理图集

### 16. 离屏预渲染
- 地图背景只渲染一次到 offscreen image，之后每帧直接 blit
- HUD 面板在数据变化时才重绘（当前每帧重绘）
- **实现**：`ebiten.NewImage` 缓存 + dirty flag

### 17. 对象池零分配
- 粒子系统、伤害数字、拖尾点全用固定大小池
- 保持现有 projectile/enemy 池模式的一致性

## 优先级建议

```
P0 (立竿见影):
  - 拖尾效果（弹射物 motion trail）
  - 精灵动画（塔攻击旋转、敌人行走摆动）
  - 伤害数字飘字
  - 屏幕震动

P1 (显著提升):
  - Bloom shader（弹射物 + 技能发光）
  - 粒子系统（死亡爆散、枪口火花）
  - 过渡动画（场景切换、波次提示）
  - 能量连线视效

P2 (锦上添花):
  - 动态光照
  - 暗角效果
  - 远景视差
  - 小地图

P3 (性能储备):
  - Sprite Batch
  - 离屏预渲染 HUD
```

## Kage Shader 示例参考

```go
// bloom_extract.kage — 提取高亮区域
package main

func Fragment(dst vec4, src vec2, color vec4) vec4 {
    c := imageSrc0At(src)
    brightness := dot(c.rgb, vec3(0.2126, 0.7152, 0.0722))
    if brightness > 0.7 {
        return c
    }
    return vec4(0)
}
```

```go
// screen_shake.go — 屏幕震动
type ScreenShake struct {
    Timer    float64
    Duration float64
    Strength float64
}

func (s *ScreenShake) Apply(opts *ebiten.DrawImageOptions) {
    if s.Timer <= 0 { return }
    ratio := s.Timer / s.Duration
    dx := (rand.Float64()*2 - 1) * s.Strength * ratio
    dy := (rand.Float64()*2 - 1) * s.Strength * ratio
    opts.GeoM.Translate(dx, dy)
}
```
