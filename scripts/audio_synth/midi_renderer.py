# midi_renderer.py — FluidSynth + SoundFont 渲染器。
#
# 职责：将 composer.Track 的 note events 通过 FluidSynth 渲染为高品质音频。
# 替代旧的纯波形 synth_note() 合成，使用 GM SoundFont 采样音色。
#
# 依赖：
#   - brew install fluid-synth
#   - pip install pyfluidsynth numpy
#   - TimGM6mb.sf2 或其他 GM SoundFont

from __future__ import annotations

import os
import numpy as np

# macOS Homebrew 安装的 FluidSynth 需要设置 HOMEBREW_PREFIX
# 让 pyfluidsynth 的 load_libfluidsynth() 能找到 dylib
if not os.getenv('HOMEBREW_PREFIX'):
    for prefix in ['/opt/homebrew', '/usr/local']:
        lib_path = os.path.join(prefix, 'lib', 'libfluidsynth.dylib')
        if os.path.exists(lib_path):
            os.environ['HOMEBREW_PREFIX'] = prefix
            break

import fluidsynth

from .synth import SAMPLE_RATE, mix_signals, normalize, save_wav, note_freq

# ── 默认 SoundFont 路径 ──────────────────────────────
_SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_SF2 = os.path.join(_SCRIPT_DIR, 'soundfonts', 'TimGM6mb.sf2')


class MidiRenderer:
    """FluidSynth 渲染器，将 Track note events 渲染为 numpy 音频数组。

    用法:
        renderer = MidiRenderer()
        audio = renderer.render_track(track, bpm=128, preset=PIANO_WARM, channel=0)
        renderer.close()
    """

    def __init__(self, sf2_path: str | None = None, sample_rate: int = SAMPLE_RATE):
        self.sr = sample_rate
        self.sf2_path = sf2_path or DEFAULT_SF2

        if not os.path.exists(self.sf2_path):
            raise FileNotFoundError(
                f"SoundFont not found: {self.sf2_path}\n"
                f"请下载 GM SoundFont 到 scripts/audio_synth/soundfonts/ 目录"
            )

        # 初始化 FluidSynth（不输出到音频设备，只用于离线渲染）
        self.synth = fluidsynth.Synth(samplerate=float(self.sr))
        self.sfid = self.synth.sfload(self.sf2_path)

    def close(self):
        """释放 FluidSynth 资源。"""
        if self.synth:
            self.synth.delete()
            self.synth = None

    def __del__(self):
        self.close()

    def _setup_channel(self, channel: int, preset: dict):
        """为通道设置乐器 program + 效果参数。"""
        bank = preset.get('bank', 0)
        program = preset.get('program', 0)
        self.synth.program_select(channel, self.sfid, bank, program)

        # FluidSynth CC: reverb=91, chorus=93
        reverb = int(preset.get('reverb', 0.3) * 127)
        chorus_val = int(preset.get('chorus', 0.0) * 127)
        self.synth.cc(channel, 91, reverb)
        self.synth.cc(channel, 93, chorus_val)

    def _note_to_midi(self, note_name: str) -> int:
        """音名转 MIDI 编号（如 'C4' → 60）。"""
        if not note_name or len(note_name) < 2:
            return 60
        names = {'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
        letter = note_name[0].upper()
        semitone = names.get(letter, 0)
        rest = note_name[1:]
        if rest.startswith('#'):
            semitone += 1
            rest = rest[1:]
        elif rest.startswith('b'):
            semitone -= 1
            rest = rest[1:]
        try:
            octave = int(rest)
        except ValueError:
            octave = 4
        return (octave + 1) * 12 + semitone

    def render_track(self, track, bpm: float, preset: dict,
                     channel: int = 0) -> np.ndarray:
        """渲染单个旋律 Track 为 stereo 音频。

        参数:
            track: composer.Track 实例（使用其 events 列表）
            bpm: 每分钟节拍数
            preset: instruments_sf2 中的乐器预设 dict
            channel: MIDI 通道（0-15，避开 9=鼓组）

        返回:
            float64 numpy array（mono，已从 stereo 混合）
        """
        if not track.events:
            return np.array([], dtype=np.float64)

        beat_dur = 60.0 / bpm
        velocity = preset.get('velocity', 100)

        # 计算总时长（最后一个 note 结束 + 1 秒 release）
        max_end = 0.0
        for beat_pos, _note, dur_beats in track.events:
            end = (beat_pos + dur_beats) * beat_dur
            max_end = max(max_end, end)
        total_seconds = max_end + 1.0
        total_samples = int(total_seconds * self.sr)

        self._setup_channel(channel, preset)

        # 按时间排序事件，转为 (sample_offset, midi_note, duration_samples) 列表
        events = []
        for beat_pos, note_name, dur_beats in track.events:
            offset = int(beat_pos * beat_dur * self.sr)
            dur_samples = int(dur_beats * beat_dur * self.sr)
            midi = self._note_to_midi(note_name)
            events.append((offset, midi, dur_samples))
        events.sort(key=lambda e: e[0])

        # 逐块渲染：遍历所有事件，在每个时间点发送 noteon/noteoff
        # 收集 FluidSynth 输出的 PCM 数据
        output = np.zeros(total_samples * 2, dtype=np.float64)  # stereo interleaved
        cursor = 0  # 当前已渲染到的 sample 位置

        # 构建 on/off 事件时间线
        timeline = []
        for offset, midi, dur in events:
            timeline.append((offset, 'on', midi, velocity))
            timeline.append((offset + dur, 'off', midi, 0))
        timeline.sort(key=lambda e: (e[0], 0 if e[1] == 'off' else 1))

        for sample_pos, action, midi, vel in timeline:
            # 渲染到此时间点
            if sample_pos > cursor:
                chunk_size = sample_pos - cursor
                buf = self.synth.get_samples(chunk_size)
                arr = np.frombuffer(buf, dtype=np.int16).astype(np.float64) / 32768.0
                start = cursor * 2
                end = start + len(arr)
                if end <= len(output):
                    output[start:end] = arr
                cursor = sample_pos

            if action == 'on':
                self.synth.noteon(channel, midi, vel)
            else:
                self.synth.noteoff(channel, midi)

        # 渲染剩余（release tail）
        remaining = total_samples - cursor
        if remaining > 0:
            buf = self.synth.get_samples(remaining)
            arr = np.frombuffer(buf, dtype=np.int16).astype(np.float64) / 32768.0
            start = cursor * 2
            end = start + len(arr)
            if end <= len(output):
                output[start:end] = arr

        # 关闭所有音符
        self.synth.cc(channel, 123, 0)  # All Notes Off

        # stereo → mono (L+R 平均)
        left = output[0::2]
        right = output[1::2]
        mono = (left + right) * 0.5

        return mono[:int(total_seconds * self.sr)] * track.volume

    def render_drum_track(self, track, bpm: float, preset: dict) -> np.ndarray:
        """渲染鼓组 Track。

        鼓组使用 GM Channel 9。Track 中的 note pitch 直接映射到 GM Drum Map。
        例如 'C2'=36=kick, 'D2'=38=snare, 'F#2'=42=hihat_closed。

        注意：鼓组 Track 的 note 字段用于指定鼓声类型（MIDI 音高），
        而非音名。应使用 instruments_sf2 的 DRUM_* 常量。
        """
        if not track.events:
            return np.array([], dtype=np.float64)

        beat_dur = 60.0 / bpm
        velocity = preset.get('velocity', 100)
        channel = 9  # GM 鼓组通道

        max_end = 0.0
        for beat_pos, _note, dur_beats in track.events:
            end = (beat_pos + dur_beats) * beat_dur
            max_end = max(max_end, end)
        total_seconds = max_end + 1.0
        total_samples = int(total_seconds * self.sr)

        self._setup_channel(channel, preset)

        # 构建时间线
        timeline = []
        for beat_pos, note_name, dur_beats in track.events:
            offset = int(beat_pos * beat_dur * self.sr)
            dur_samples = int(dur_beats * beat_dur * self.sr)
            # note_name 可能是 MIDI 编号(int)、纯数字字符串("36")或音名("C2")
            if isinstance(note_name, int):
                midi = note_name
            elif note_name.isdigit():
                midi = int(note_name)
            else:
                midi = self._note_to_midi(note_name)
            timeline.append((offset, 'on', midi, velocity))
            timeline.append((offset + dur_samples, 'off', midi, 0))
        timeline.sort(key=lambda e: (e[0], 0 if e[1] == 'off' else 1))

        output = np.zeros(total_samples * 2, dtype=np.float64)
        cursor = 0

        for sample_pos, action, midi, vel in timeline:
            if sample_pos > cursor:
                chunk_size = sample_pos - cursor
                buf = self.synth.get_samples(chunk_size)
                arr = np.frombuffer(buf, dtype=np.int16).astype(np.float64) / 32768.0
                start = cursor * 2
                end = start + len(arr)
                if end <= len(output):
                    output[start:end] = arr
                cursor = sample_pos

            if action == 'on':
                self.synth.noteon(channel, midi, vel)
            else:
                self.synth.noteoff(channel, midi)

        remaining = total_samples - cursor
        if remaining > 0:
            buf = self.synth.get_samples(remaining)
            arr = np.frombuffer(buf, dtype=np.int16).astype(np.float64) / 32768.0
            start = cursor * 2
            end = start + len(arr)
            if end <= len(output):
                output[start:end] = arr

        self.synth.cc(channel, 123, 0)

        left = output[0::2]
        right = output[1::2]
        mono = (left + right) * 0.5

        return mono[:int(total_seconds * self.sr)] * track.volume


class SF2Song:
    """基于 SF2 的 Song，管理多个 Track + MidiRenderer 渲染。

    与旧的 composer.Song 接口兼容，但渲染走 FluidSynth。
    每个 Track 需要绑定一个 SF2 preset。
    """

    def __init__(self, bpm: float, beats_per_bar: int = 4,
                 sf2_path: str | None = None):
        self.bpm = bpm
        self.beats_per_bar = beats_per_bar
        self.tracks: list[tuple] = []  # (Track, preset, channel)
        self.sf2_path = sf2_path
        self._next_channel = 0

    def add_track(self, track, preset: dict, channel: int | None = None):
        """添加轨道。

        参数:
            track: composer.Track 实例
            preset: instruments_sf2 中的预设 dict
            channel: MIDI 通道，None 则自动分配（跳过 9=鼓组）
        """
        is_drum = preset.get('type') == 'sf2_drum'

        if channel is None:
            if is_drum:
                channel = 9
            else:
                if self._next_channel == 9:
                    self._next_channel = 10
                channel = self._next_channel
                self._next_channel += 1

        self.tracks.append((track, preset, channel))
        return track

    def render(self, sr: int = SAMPLE_RATE) -> np.ndarray:
        """渲染所有轨道，混合并归一化。"""
        if not self.tracks:
            return np.array([], dtype=np.float64)

        renderer = MidiRenderer(sf2_path=self.sf2_path, sample_rate=sr)
        rendered = []

        try:
            for track, preset, channel in self.tracks:
                is_drum = preset.get('type') == 'sf2_drum'
                if is_drum:
                    audio = renderer.render_drum_track(track, self.bpm, preset)
                else:
                    audio = renderer.render_track(
                        track, self.bpm, preset, channel=channel
                    )
                rendered.append(audio)
        finally:
            renderer.close()

        mixed = mix_signals(*rendered)
        return normalize(mixed)

    def save(self, filename: str, sr: int = SAMPLE_RATE) -> None:
        """渲染并保存为 WAV。"""
        signal = self.render(sr)
        save_wav(filename, signal, sr)
        print(f"  Saved: {filename} ({len(signal)/sr:.1f}s, {len(signal)} samples)")
