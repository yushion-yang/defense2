# 资源创作方案

项目已有两套资源生成管线，Claude 可以直接编写配置/代码来生成资源，无需外部工具。

---

## 现有管线

| 管线 | 语言 | 输入 | 输出 | 命令 |
|------|------|------|------|------|
| SFX 合成 | Python | `sfx_configs.py` 声学参数 | WAV 44100Hz | `python3 scripts/generate_sfx.py` |
| 精灵生成 | Node.js | `config/visuals/*.json` 视觉描述 | SVG → PNG 128×128 | `node scripts/generate-assets.mjs <type>` |

### SFX 管线能力

`sfx_engine.py` (348 行) 提供：
- **振荡器**: sine/saw/square/FM 合成
- **噪声**: 白噪声 + 高通/带通滤波
- **包络**: exp 衰减 / ADSR / 线性
- **效果**: 谐波叠加(metallic/bright/warm)、瞬态注入、混响、压缩
- **自动优化**: `generate_sfx.py` 含 6 维评分器(spectral/transient/dynamic/envelope/harmonics/psycho)，自动迭代改善低分维度

已有 126 个 WAV，覆盖攻击/命中/CC/UI/Boss/战灵/敌人能力。

### 精灵管线能力

`generate-assets.mjs` + `generators/*.mjs` 提供：
- JSON 声明式视觉描述 → SVG 代码生成 → sharp 转 PNG
- 支持多帧动画：塔 5 帧(idle×2+attack×3)、敌人 6 帧(walk×4+hit×2)
- 战灵当前单帧，框架已支持 idle/attack 多帧

---

## 待创作资源清单

### 1. BGM 音乐（3 首）

**现状**: 代码就绪(PlayBGM/StopBGM)，缺音乐文件。

**方案 A: Python 算法作曲（Claude 可直接做）**

扩展 `sfx_engine.py` 为 `bgm_engine.py`，用算法作曲生成循环 BGM：
- 和弦进行：I-V-vi-IV 等常见进行
- 节奏轨：kick/snare/hihat 鼓机模式
- 旋律轨：五声音阶随机行走 + 量化
- 低音轨：跟随和弦根音
- 合成器音色：基于现有 FM/saw/harmonics 振荡器

```python
# bgm_configs.py 示例
bgm('bgm-menu', bpm=90, key='C', scale='pentatonic', mood='calm',
    bars=16, instruments=['pad', 'arp', 'soft_kick'],
    reverb=0.4, stereo_width=0.6)

bgm('bgm-battle', bpm=128, key='Am', scale='minor', mood='tense',
    bars=32, instruments=['lead', 'bass', 'kick', 'snare', 'hihat'],
    reverb=0.25, drive=1.2)

bgm('bgm-boss', bpm=140, key='Dm', scale='harmonic_minor', mood='intense',
    bars=16, instruments=['heavy_lead', 'sub_bass', 'double_kick', 'crash'],
    reverb=0.2, drive=1.5)
```

输出：WAV 44100Hz，60-120 秒循环。文件放入 `assets/audio/`。

**优点**: 完全离线，Claude 可写代码+调参，风格可控
**缺点**: 音色受限于合成器，没有真实乐器质感

**方案 B: AI 音乐生成服务（需要用户参与）**

用 Suno/Udio 等服务生成，提供 prompt：
- Menu: "calm ambient electronic, tower defense game menu, loopable, 90bpm"
- Battle: "tense electronic battle music, strategic game, 128bpm, loopable"
- Boss: "intense boss battle electronic, dramatic, 140bpm, loopable"

下载后转 WAV 放入 `assets/audio/`。

**优点**: 音质高，有真实乐器
**缺点**: 需要用户操作外部服务

**推荐**: 先用方案 A 快速出可用版本，后续用方案 B 替换提升品质。

### 2. 战灵多帧精灵（20 张 PNG）

**现状**: 5 种战灵各 1 张静态 PNG。Animator 框架已就绪。

**方案: 扩展 generate-assets.mjs**

在 `config/visuals/wardens.json` 中为每种战灵添加多帧描述：

```json
{
  "prince": {
    "frames": {
      "idle-0": { "pose": "standing", "flame": "low" },
      "idle-1": { "pose": "standing", "flame": "high" },
      "attack-0": { "pose": "arm_raised", "flame": "burst" },
      "attack-1": { "pose": "arm_thrust", "flame": "max" }
    }
  }
}
```

`generators/warden.mjs` 扩展为支持多帧 SVG 生成，每帧微调姿态/特效：
- idle 帧：身体微摆 + 特效脉冲（火焰/能量/光环）
- attack 帧：手臂/武器动作 + 特效爆发

输出命名：`warden-prince-idle-0.png` ... `warden-prince-attack-1.png`
放入 `assets/wardens/`，Animator 自动检测。

**执行**: Claude 编写 JSON 描述 + generator 代码 → `node scripts/generate-assets.mjs wardens --force`

### 3. 道具图标（6 张 PNG）

**现状**: 复用 stat 图标，可用但不够专属。

**方案: Go 程序化生成**

写一个 `cmd/genicons/main.go`，用 Go 标准库 `image/draw` 生成 64×64 PNG：

| 道具 | 图案设计 |
|------|---------|
| 攻击磨石 | 红色菱形 + 剑形轮廓 |
| 攻击秘卷 | 深红色卷轴 + 剑纹 |
| 速射齿轮 | 黄色齿轮 + 闪电标记 |
| 速射秘卷 | 深黄色卷轴 + 闪电纹 |
| 瞄准镜片 | 蓝色圆形 + 十字准星 |
| 瞄准秘卷 | 深蓝色卷轴 + 准星纹 |

```go
// 用 image.NRGBA + 基础几何绘制
// 圆形、矩形、线条组合出简约图标
// 每种道具 ~30 行绘制代码
```

输出放入 `assets/icons/item-*.png`，ItemPanel 改用 `render.GlobalIcons().Get("item-xxx")`。

**执行**: Claude 写 Go 代码 → `go run cmd/genicons/main.go` → 6 张 PNG

### 4. 额外 SFX（按需）

如果某个游戏行为缺音效，直接在 `sfx_configs.py` 追加配置：

```python
# 示例：道具使用音效
sfx('item-use', category='ui', duration=0.15, drive=0.8, layers=[
    {'type': 'harmonics', 'freq': 800, 'freq_end': 1200, 'harmonics': 6,
     'h_curve': 'bright', 'vol': 0.5, 'env': 'exp', 'env_rate': 12},
    {'type': 'sine', 'freq': 1600, 'vol': 0.3, 'env': 'exp', 'env_rate': 16},
])
```

然后运行：`python3 scripts/generate_sfx.py item-use`

评分器自动检测质量，低于阈值则自动迭代优化。

---

## 执行方式

所有资源创作都可以通过以下方式让 Claude 完成：

1. **告诉 Claude 要什么** — "生成 3 首 BGM" / "给战灵加动画帧" / "做道具图标"
2. **Claude 编写配置/代码** — 修改 JSON/Python/Go 生成脚本
3. **Claude 运行生成命令** — `python3 scripts/...` 或 `go run cmd/...` 或 `node scripts/...`
4. **资源自动落入 assets/** — 代码侧无需改动（已预留接口）
5. **审听/审视** — 你 `make run` 看效果，不满意告诉 Claude 调参

无需安装额外软件，无需离开终端。

---

## 优先级

```
1. BGM (方案A算法作曲) — 氛围感提升最大
2. 道具图标 (Go 程序化) — 半小时搞定
3. 战灵多帧 (扩展 generate-assets) — 动画质感
4. 额外 SFX (sfx_configs 追加) — 按需补充
```
