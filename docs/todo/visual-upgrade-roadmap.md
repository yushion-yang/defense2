# Visual Upgrade Roadmap — 剩余项

> 已完成项归档见 `docs/archive/2026-03-31-visual-upgrade-roadmap-completed.md`

## 阻塞项（等外部条件）

### Bloom 发光
- 塔攻击时弹射物发光拖尾 / Boss 出场全屏 bloom 脉冲 / 战灵技能局部 bloom
- **阻塞原因**: DrawRectShader 在 Ebitengine v2.9.9 多图片操作有 runtime crash，`BloomEnabled` 默认 false
- **解除条件**: 升级 Ebitengine 到修复版本后启用

## 待资产/机制项

### 爆炸特效序列帧
- 需要爆炸动画 sprite sheet 资产（当前资产管线只生成 idle/walk/hit/attack 帧）
- 可通过 `scripts/generate-assets.mjs` 扩展 "explosion" 类型生成

### 治疗绿色伤害数字
- `SpawnDamageText` 已支持自定义颜色（通过 `SpawnText`），但无治疗机制触发
- 待实现治疗型战灵/塔能力时加入

## 性能储备项

### Sprite Batch 批量渲染
- 将同类精灵合并为单次 `DrawTriangles` 调用 + 纹理图集
- **当前状况**: 粒子系统已用 DrawTriangles 批渲染；敌人/弹射物同屏 <100，性能足够
- **触发条件**: 移动端实测出现渲染瓶颈时实施
