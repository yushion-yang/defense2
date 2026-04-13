# BGM Redesign — MIDI + SoundFont

日期: 2026-04-13

## 背景

现有 3 首 BGM（menu/battle/boss）由纯波形合成引擎生成，标记为 placeholder。
音色单薄（sine/square/triangle），编曲简单，整体质感不匹配游戏的视觉和深度。

## 目标

1. 音色升级：从纯波形合成 → SoundFont 采样乐器（钢琴、弦乐、铜管、合成器、鼓组）
2. 编曲升级：更专业的和弦进行、旋律、动态变化、段落对比
3. 风格定位：电子+管弦混合风（现代感 + 史诗感）
4. Go 端零改动：输出仍为 44100Hz 16-bit WAV，文件名/常量/播放逻辑不变

## 技术方案

### 工具链

- **FluidSynth**（C 库）：开源 SoundFont 渲染器，macOS 通过 `brew install fluid-synth` 安装
- **pyfluidsynth**（Python 绑定）：`pip install pyfluidsynth`
- **SoundFont**：GeneralUser GS v1.471（~30MB，免费，GM 兼容，128 乐器 + 鼓组）
- **保留现有框架**：`composer.py` 的 Track/Song/DrumPattern 结构复用

### 新增文件

```
scripts/audio_synth/
├── midi_renderer.py      # FluidSynth 渲染器封装
│   - MidiRenderer 类：加载 SF2、渲染 Track → numpy array
│   - 支持 GM program number 选择乐器
│   - 支持 velocity/expression/reverb/chorus 控制
│   - 输出 stereo 44100Hz float64 信号
├── instruments_sf2.py    # SF2 乐器映射常量
│   - GM Program Number 映射（Piano=0, Strings=48, Brass=61...）
│   - 预设组合（PIANO_WARM, STRINGS_STACCATO, BRASS_SECTION...）
│   - 鼓组映射（GM Drum Map: kick=36, snare=38, hihat=42...）
├── soundfonts/
│   └── .gitkeep          # SF2 文件不入库（>30MB），README 说明下载方式
├── soundfonts/README.md  # 下载指引
└── songs/                # 三首曲目重写
    ├── menu.py
    ├── battle.py
    └── boss.py
```

### MidiRenderer 核心接口

```python
class MidiRenderer:
    def __init__(self, sf2_path: str, sample_rate: int = 44100):
        """加载 SoundFont，初始化 FluidSynth synth。"""

    def render_track(self, track: Track, bpm: float,
                     program: int, channel: int = 0,
                     velocity: int = 100,
                     reverb: float = 0.3,
                     chorus: float = 0.1) -> np.ndarray:
        """将 Track 的 note events 通过 FluidSynth 渲染为音频。

        流程：
        1. 设定 channel 的 program（乐器）
        2. 按时间顺序发送 noteon/noteoff 事件
        3. 收集 FluidSynth 输出的 PCM 采样
        4. 返回 stereo numpy array
        """

    def render_drum_track(self, track: Track, bpm: float,
                          velocity: int = 100) -> np.ndarray:
        """GM Channel 10 鼓组渲染。note pitch 映射到 GM drum map。"""
```

### Song 渲染流程

```
Song.render()
  → 遍历 tracks
    → 判断 instrument 类型
      → SF2 乐器 → MidiRenderer.render_track()
      → SF2 鼓组 → MidiRenderer.render_drum_track()
      → 旧波形乐器 → 原有 synth_note() (fallback)
  → mix_signals() 混合所有轨道
  → normalize() → save_wav()
```

向后兼容：旧的波形乐器 preset 仍可使用，SF2 和波形可以混用。

## 三首曲目设计

### 1. bgm-menu — "Echoes of Command"（指挥回响）

| 属性 | 值 |
|------|-----|
| 调性 | D major → Bm（温暖→神秘） |
| BPM | 92 |
| 时长 | ~80s |
| 拍号 | 4/4 |

**乐器编排**:
| 轨道 | GM Program | 角色 |
|------|-----------|------|
| Piano | 0 (Acoustic Grand) | 主旋律琶音 |
| String Pad | 48 (String Ensemble 1) | 长音铺底 |
| Synth Pad | 89 (Warm Pad) | 层叠氛围 |
| Celesta | 8 (Celesta) | 装饰音点缀 |
| Brush Kit | GM Drum (brush) | 轻打击 |

**结构**:
```
Intro (8 bar) — 钢琴独奏 + 渐入弦乐 pad
A (8 bar)    — 主题旋律，钢琴 + celesta 点缀
B (8 bar)    — 转 Bm，弦乐主导，略带忧郁
A' (8 bar)   — 回 D major，全乐器，加入轻鼓
Outro (4 bar) — 渐弱，自然接循环头
```

**和弦进行**:
- A 段: D - A/C# - Bm - G - D/F# - Em - A - D
- B 段: Bm - F#m - G - D - Em - Bm - A - A

### 2. bgm-battle — "Frontline Protocol"（前线协议）

| 属性 | 值 |
|------|-----|
| 调性 | Am → Dm → Em |
| BPM | 128 |
| 时长 | ~90s |
| 拍号 | 4/4 |

**乐器编排**:
| 轨道 | GM Program | 角色 |
|------|-----------|------|
| Synth Bass | 38 (Synth Bass 1) | octave 脉冲底盘 |
| Strings Staccato | 48 (String Ensemble 1) | 切分紧迫节奏 |
| Brass Section | 61 (Brass Section) | 重拍强调 |
| Synth Lead | 81 (Lead 2 - Sawtooth) | 主旋律 |
| Arp Synth | 84 (Lead 5 - Charang) | B段上行 pattern |
| Drum Kit | GM Drum (Standard) | 完整鼓组 |

**结构**:
```
Intro (4 bar)    — 鼓组 + bass buildup
A (8 bar)        — Am, 紧凑战斗主题，brass stab
B (8 bar)        — Dm, synth lead 旋律 + arp 加入
A' (8 bar)       — Em, 变奏提升，弦乐加厚
Bridge (4 bar)   — breakdown → rebuild，接循环
```

**和弦进行**:
- A 段: Am - F - C - G - Am - F - Dm - E
- B 段: Dm - Bb - F - C - Dm - Bb - Am - A (→Em)
- A' 段: Em - C - G - D - Em - C - Am - B

### 3. bgm-boss — "Titan's Descent"（巨灵降临）

| 属性 | 值 |
|------|-----|
| 调性 | Em → C#m |
| BPM | 148 |
| 时长 | ~80s |
| 拍号 | 4/4 |

**乐器编排**:
| 轨道 | GM Program | 角色 |
|------|-----------|------|
| Heavy Bass | 38 (Synth Bass 1) | 下行 riff |
| String Tutti | 48 (String Ensemble 1) | tremolo + stab |
| Brass Fanfare | 61 (Brass Section) | Boss 主题动机 |
| Timpani | GM Drum (47=Timpani) | 史诗打击 |
| Distorted Lead | 30 (Distortion Guitar) | 攻击性旋律 |
| Choir Pad | 52 (Choir Aahs) | 高潮人声质感 |
| Drum Kit | GM Drum (Standard) | 密集打击 |

**结构**:
```
Impact Intro (2 bar) — 全乐器冲击 + timpani roll
A (8 bar)            — Em, brass fanfare + 弦乐 tremolo
B (4 bar)            — 短暂喘息，bass solo + 稀疏打击
A' (8 bar)           — C#m, 半音下行压迫，全力推进
C (8 bar)            — 高潮段，choir 加入，lead 最高音区
Loop tail (2 bar)    — crash → 接循环
```

**和弦进行**:
- A 段: Em - C - D - B - Em - C - Am - B
- A' 段: C#m - A - B - G# - C#m - A - F#m - G#
- C 段: Em - D - C - B - Am - G - F# - B → Em

## 依赖安装

```bash
# macOS
brew install fluid-synth
pip install pyfluidsynth numpy

# SoundFont 下载（不入 git）
# GeneralUser GS v1.471: https://schristiancollins.com/generaluser.php
# 放到 scripts/audio_synth/soundfonts/GeneralUser_GS.sf2
```

## 生成命令

```bash
cd scripts/audio_synth
python generate_all.py          # 生成 3 首 BGM → assets/audio/
python generate_all.py --song menu  # 只生成 menu
```

## Go 端影响

**零改动**。输出文件名（bgm-menu.wav / bgm-battle.wav / bgm-boss.wav）、
格式（44100Hz WAV）、播放接口（PlayBGM/StopBGM）全部不变。

## 回退方案

如果 FluidSynth 环境有问题，旧的波形合成 songs 保留为 `songs/menu_legacy.py` 等，
`generate_all.py --legacy` 可回退到纯波形输出。
