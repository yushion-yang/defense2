# Go/Ebitengine 优势演进蓝图

> 生成日期: 2026-03-29 | 分支: claude/migration-44items
>
> **定位**: 长期演进蓝图。覆盖所有 JS Canvas 2D 做不到或做不好的能力方向。
>
> **平台策略**: 移动优先（Android/iOS），桌面为开发环境。
>
> **规模策略**: 保持现有战场规模（256 敌人 / 64 塔 / 1024 弹射物），Go 优势用在视觉和平台能力上。

## 核心原则

只做 JS Canvas 2D **做不到或做不好**的事情。不重复功能迁移（那是 `migration-checklist.md` 的工作）。

## 演进路线

```
Phase 1: 视觉超越 ——— Kage Shader + GPU 粒子系统
Phase 2: 移动原生 ——— 触控体验 + 平台集成 + 资源策略
Phase 3: 智能体验 ——— 并发计算 + 程序化内容
Phase 4: 性能封顶 ——— 批渲染 + 内存/GC 极限优化 + 发热控制
```

每个 Phase 独立可交付，前者为后者铺基础。Phase 1 最大感知提升，Phase 4 是发布前的打磨。

**与现有 roadmap 的关系**: `visual-upgrade-roadmap.md` 的 P1-P3 内容被吸收进 Phase 1/4，本文档是上位规划。

---

## Phase 1: 视觉超越 (大部分完成)

**目标**: 让玩家第一眼就感受到"这不是浏览器游戏"。利用 Kage Shader 和 GPU 做 Canvas 2D 根本不可能的效果。

> **已完成**: 1.2 后处理(vignette/color_grade/radial_blur/ripple/lighting) / 1.3 GPU粒子系统(2048池+8预设) / 1.4 动态光照(4点光源)
> **阻塞**: 1.1 Bloom (DrawRectShader v2.9.9 crash)

### 1.1 Bloom 后处理管线（阻塞）

Canvas 2D 没有 render-to-texture 能力，无法做后处理。Go 版可以：

- **实现**: 主场景渲染到 offscreen -> 提取高亮（亮度>阈值）-> 2 pass 高斯模糊 -> additive blend 回主画面
- **应用场景**:
  - 弹射物飞行时的发光拖尾
  - Boss 出场全屏 bloom 脉冲
  - 战灵技能释放时局部 bloom
  - 塔升级/建造完成时的闪光
- **Shader**: `bloom_extract.kage`（提取）+ `blur_h.kage`/`blur_v.kage`（两 pass 高斯）
- **性能预算**: 3 张额外 offscreen image（1/4 分辨率），移动端可降为 1/8

### 1.2 屏幕后处理效果

一旦有了 render-to-texture 管线，多种后处理几乎免费：

| 效果 | 触发时机 | 实现 |
|------|---------|------|
| 径向模糊 | Boss 出场/死亡 | 从中心采样偏移的 Kage shader |
| 色调映射 | 关卡氛围（昼夜、受伤红闪） | 颜色矩阵变换 shader |
| 波纹扭曲 | 冰冻技能命中 | UV 偏移 shader（正弦波） |
| 暗角 Vignette | 常驻 | 边缘暗化 shader |
| HitStop | Boss 受击 | 暂停 2-3 帧 + 轻微缩放 |

### 1.3 GPU 粒子系统

JS Canvas 每个粒子一次 `fillRect/arc` 调用，>200 个就卡。Go 可以：

- **架构**: 固定大小粒子池（2048 cap）+ `DrawTriangles` 批量渲染
- **粒子类型**（按游戏场景）:
  - 敌人死亡: 像素碎片爆散（8-16 粒子，重力下落）
  - 枪口火花: 射击时 3-5 个粒子沿射击方向散射
  - 元素效果: 火焰持续上升粒子、冰雪飘落、电弧闪烁
  - 金币收集: 闪光粒子从击杀点飞向 HUD 金币栏
  - 战灵光环: 环绕战灵的微小光点
- **实现**: 每粒子 = position + velocity + life + color + size，Update 纯数组遍历，Draw 用 `DrawTriangles` 一次提交
- **移动端策略**: 粒子数减半，生命周期缩短

### 1.4 动态光照（简化版）

完整法线贴图在移动端过重，采用简化方案：

- 每个塔/弹射物/战灵携带一个"光源描述"（位置、颜色、半径、强度）
- 绘制时在其位置叠加 radial gradient overlay（半透明圆形 additive blend）
- Kage shader 做全局 ambient + 点光源叠加
- **移动端**: 限制同屏光源数（最多 8 个），超出丢弃最远的

---

## Phase 2: 移动原生

**目标**: 利用 Ebitengine 的原生编译能力（Gomobile），提供浏览器 H5 游戏无法企及的移动体验。

### 2.1 触控体验优化

JS 版是鼠标优先，触控是二等公民。Go 版反过来：

| 手势 | 功能 | JS 版问题 -> Go 解决 |
|------|------|---------------------|
| 单指点击 | 选塔/建塔/开波 | JS: 300ms 点击延迟 -> Go: 原生触控无延迟 |
| 长按 | 显示塔详情/敌人详情 | JS: 与右键冲突 -> Go: 原生长按识别 |
| 双指缩放 | 地图缩放（大地图） | JS: Canvas 缩放模糊 -> Go: GPU 重采样清晰 |
| 双指拖拽 | 地图平移 | JS: 触发浏览器默认行为 -> Go: 完全控制 |
| 快速滑动 | 技能释放方向 | JS: 无 -> Go: 原生手势识别 |
| 多指 | 同时操作塔+释放技能 | JS: multitouch 事件混乱 -> Go: Ebitengine TouchIDs 可靠 |

- **触控区域自适应**: 按钮/HUD 元素在移动端自动放大到 44px 最小触控区域（Apple HIG 标准）
- **震动反馈**（Android/iOS 原生调用）: 建塔成功、Boss 出场、关键击杀时触发设备震动
- **现有基础**: `internal/input/` 已有统一输入层，在其上扩展手势识别

### 2.2 平台集成

原生 App 能做而浏览器做不到的事：

| 能力 | 说明 | 实现方式 |
|------|------|---------|
| 触觉反馈 | 建塔/Boss/击杀震动 | Gomobile 调 Android Vibrator / iOS UIImpactFeedbackGenerator |
| 本地通知 | "体力恢复了" "限时活动" | Android NotificationManager / iOS UNUserNotificationCenter |
| 横竖屏锁定 | 强制横屏 | AndroidManifest screenOrientation / iOS plist |
| 安全区适配 | iPhone 刘海/药丸 | Ebitengine `DeviceScaleFactor` + safe area insets |
| 省电模式检测 | 低电量时降帧到 30fps | 读系统电量 API，动态调整 TPS |
| 后台暂停 | App 切后台自动暂停 | Ebitengine 已支持 `SetScreenClearedEveryFrame` |
| 云存档 | Google Play Games / Game Center | 平台 SDK 调用保存/加载 |

### 2.3 资源策略

移动端内存和存储敏感，需要专门策略：

- **纹理压缩**: PNG -> ASTC/ETC2（Android）/ ASTC/PVRTC（iOS），GPU 直接解码，内存占用降 4-8x
- **按需加载**: 非当前关卡的资源不预加载（当前 `go:embed` 全量嵌入）
- **分辨率适配**:
  - 手机: 逻辑分辨率 960x432（1200x540 的 80%）
  - 平板: 保持 1200x540
  - 低端设备: 720x324（60%）
- **帧率策略**: 默认 60fps，低端设备/省电模式自动降为 30fps
- **内存预算**: 目标 <150MB RSS（当前 `go:embed` 全量嵌入约 50MB 音频 + 20MB 图片）

### 2.4 Android/iOS 发布工程

| 项目 | Android | iOS |
|------|---------|-----|
| 构建 | `gomobile bind` + AAR | `gomobile bind` + Framework |
| 签名 | Gradle keystore | Xcode provisioning profile |
| 商店 | Google Play Console | App Store Connect |
| 审核要点 | 64-bit 必须、target API level | App Review Guidelines |
| CI | GitHub Actions + Gradle | GitHub Actions + Fastlane |
| 热更新 | 配置 JSON 从 CDN 拉取 | 同上（资源级热更，不触发审核） |

---

## Phase 3: 智能体验

**目标**: 利用 Go 的并发能力和强类型系统，做 JS 单线程很难实现的游戏智能和内容生成。

### 3.1 并发计算卸载

当前游戏完全单线程。以下计算可以安全地移到后台 goroutine：

| 计算任务 | 当前状态 | 并发方案 | 延迟容忍度 |
|----------|---------|---------|-----------|
| 寻路预计算 | 敌人沿固定路径 | 地图加载时 goroutine 预计算所有路径变体（反转/分支） | 1-2s 启动时间 |
| AI 决策树 | 无 | 战灵 AI 评估在后台 goroutine，结果通过 channel 传回 | 1-2 帧延迟可接受 |
| 波次预生成 | Spawner 实时计算 | 下一波敌人配置在当前波开始时异步准备 | 几秒提前量 |
| 统计聚合 | 每帧累加 | Session 统计的聚合/排序移到后台 | 非实时 |
| 配置热重载 | 无 | 文件监控 goroutine（开发模式），检测到变化通过 channel 通知主循环 | 100ms |

**安全约束**:
- 游戏主循环（Update）永远在单 goroutine 运行，不加锁
- 后台 goroutine 只做计算，结果通过 buffered channel 传回
- 主循环在帧开头检查 channel，有数据就消费，无数据不阻塞（`select` + `default`）

### 3.2 程序化内容生成

JS 版所有关卡/波次/事件都是手工配置。Go 版可以引入程序化生成：

**波次生成器增强**:
- 当前: 5 阶段递进 + 每 5 波 Boss（硬编码模式）
- 升级: 基于约束的波次生成引擎
  - 输入: 当前波次号、难度系数、玩家当前防线强度评估
  - 约束: 总 HP 预算、类型多样性最低要求、Boss 间隔规则
  - 输出: 一波的完整敌人配置
  - **自适应难度**: 跟踪玩家的泄漏率，动态微调 HP/速度系数

**地图生成**（无尽模式）:
- 给定网格尺寸 + 入口/出口位置 -> 自动生成路径 + 建塔位
- 算法: 随机游走 + 约束求解（路径长度下限、建塔位最少数量）
- 在后台 goroutine 预生成下一张地图，玩家通关时无缝切换

**事件组合**:
- 从事件模板池中按规则组合，避免冲突（如"全护盾" + "护盾破碎"不共存）
- 根据玩家历史偏好调整事件出现概率

### 3.3 战灵 AI 增强

当前战灵 AI 是简单状态机（idle/orbit/dash/attack）。可以用 Go 的强类型做更丰富的行为树：

- **行为树框架**: `internal/core/ai/` — Node 接口（Sequence/Selector/Decorator/Action），JSON 可配置
- **战灵决策**:
  - 扫描威胁（快到终点的敌人 > 高 HP 精英 > 普通群体）
  - 技能时机（等敌人聚团时放 AoE，单体高 HP 时放单体技能）
  - 走位（远离高伤害敌人的攻击范围，靠近需要保护的路径段）
- **可配置性**: 每种战灵类型有不同的行为树 JSON 配置

### 3.4 Replay 系统

JS 难以做到精确回放（浮点精度、随机数状态、事件时序）。Go 可以：

- **录制**: 每帧记录玩家操作（建塔/卖塔/技能/开波）+ 帧号 + 随机种子
- **回放**: 相同种子 + 相同操作序列 -> 确定性回放
- **存储**: 一局 10 分钟约 10-50KB（只存操作，不存状态）
- **用途**: 分享精彩回放、Bug 复现、AI 训练数据

---

## Phase 4: 性能封顶

**目标**: 发布前的深度优化。让游戏在中低端移动设备上也能流畅 60fps，控制发热和耗电。

### 4.1 批量渲染（Sprite Batch）

当前每个实体一次 `DrawImage` 调用。256 敌人 + 64 塔 + 1024 弹射物 = 1344+ draw calls/帧。

- **纹理图集**: 将同类精灵打包为单张大图（2048x2048）
  - 塔图集、敌人图集、弹射物图集、粒子图集
  - 构建时用脚本打包，运行时 `SubImage` 切片
- **DrawTriangles 批处理**: 同一图集的所有实体合并为一次 `DrawTriangles` 调用
  - 理论上 1344 draw calls -> 4-5 calls
  - 每帧构造顶点数组（位置/UV/颜色），一次提交 GPU
- **移动端收益**: 减少 GPU 状态切换，显著降低 draw call 开销

### 4.2 离屏预渲染

- **地图背景**: 只在地图加载/变化时渲染一次到 offscreen image
- **HUD 缓存**: TopBar/InfoPanel/WavePanel 等只在数据变化时重绘（dirty flag）
- **静态文本缓存**: 频繁绘制的文本（"Wave 3"、金币数字）预渲染为 image
- **预估收益**: HUD 每帧节省 50-100 次 draw/text 调用

### 4.3 GC 压力控制

Go GC 在移动端更敏感（GC pause 导致掉帧）：

| 当前分配热点 | 优化方案 |
|-------------|---------|
| 每帧 `fmt.Sprintf` 格式化数字 | 预分配 `[]byte` buffer + strconv |
| `ebiten.DrawImageOptions{}` 字面量 | 池化或复用 `sync.Pool` |
| Slice append（弹跳列表等） | 固定容量预分配，不用 append |
| 闭包/回调捕获 | 改为方法调用，避免闭包分配 |
| Map 迭代（事件分发等） | 改为 slice 存储，按索引遍历 |

- **目标**: Update 循环零分配（`go test -benchmem` 验证）
- **工具**: `runtime.ReadMemStats` 每秒采样，debug overlay 显示 GC 次数/暂停时间

### 4.4 热量与耗电控制

移动游戏的隐形杀手——设备发热导致降频导致卡顿：

- **自适应帧率**:
  - 默认 60fps
  - 检测到温度升高（通过帧时间突然变长推断）-> 降到 45fps
  - 省电模式 / 低电量 -> 降到 30fps
  - 游戏暂停 / 后台 -> 降到 10fps
- **渲染质量分级**:
  - High: 全效果（Bloom + 粒子 2048 + 光照）
  - Medium: 粒子 1024，无光照，Bloom 1/8 分辨率
  - Low: 无 Bloom，粒子 512，无后处理
  - 首次启动自动检测设备性能，后续可手动切换
- **CPU 预算监控**:
  - Update/Draw 各自计时，超过 12ms 触发警告
  - debug overlay 实时显示 Update/Draw/Idle 占比

### 4.5 启动优化

- **分阶段加载**:
  - Phase A（<1s）: 显示 Logo + 加载进度条
  - Phase B（并发）: goroutine 并行加载 音频/精灵/配置
  - Phase C: 进入标题画面
- **懒加载**: 非核心资源（高级技能特效、后期关卡素材）在首次使用时加载
- **当前问题**: `go:embed` 全量嵌入，二进制约 70MB，需评估是否改为运行时从 assets 目录加载

---

## 全景概览

| Phase | 主题 | 核心交付物 | JS 做不到的原因 |
|-------|------|-----------|----------------|
| 1 | 视觉超越 | ~~后处理 + 粒子 + 光照~~ ✅ / Bloom 阻塞 | Canvas 2D 无 shader/render-to-texture |
| 2 | 移动原生 | 触控优化 + 震动 + 省电 + 商店发布 | 浏览器沙箱无原生 API |
| 3 | 智能体验 | 并发 AI + 程序化生成 + Replay | JS 单线程 + 浮点不确定 |
| 4 | 性能封顶 | 批渲染 + GC 控制 + 热量管理 | JS GC 不可控 + 无 GPU 批处理 |

## 依赖关系

```
Phase 1 (Shader/粒子)
   ├── 1.1 Bloom 管线 ← 基础，1.2/1.4 依赖它
   ├── 1.2 后处理效果 ← 依赖 1.1 的 render-to-texture
   ├── 1.3 粒子系统 ← 独立，可与 1.1 并行
   └── 1.4 动态光照 ← 依赖 1.1 的 offscreen 管线

Phase 2 (移动原生) ← 不依赖 Phase 1，可并行
   ├── 2.1 触控优化 ← 独立
   ├── 2.2 平台集成 ← 依赖 Gomobile 构建链
   ├── 2.3 资源策略 ← 独立
   └── 2.4 发布工程 ← 依赖 2.2

Phase 3 (智能体验) ← 不依赖 Phase 1/2
   ├── 3.1 并发卸载 ← 独立
   ├── 3.2 程序化生成 ← 可用 3.1 的并发能力
   ├── 3.3 战灵 AI ← 可用 3.1 的并发能力
   └── 3.4 Replay ← 独立

Phase 4 (性能封顶) ← 建议在 Phase 1-3 完成后
   ├── 4.1 批渲染 ← Phase 1 的粒子系统受益最大
   ├── 4.2 离屏预渲染 ← 独立
   ├── 4.3 GC 控制 ← 独立
   ├── 4.4 热量控制 ← 依赖 Phase 2 的移动端基础
   └── 4.5 启动优化 ← 依赖 Phase 2 的资源策略
```

## 备注

- Phase 1 和 Phase 2 可以并行推进（无依赖）
- Phase 3 理论上也可以与 1/2 并行，但建议在核心游戏循环稳定后再做
- Phase 4 作为发布前打磨，需要 1-3 的功能稳定后才有优化基准
