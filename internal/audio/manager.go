// manager.go — 音效管理器。
// 统一管理游戏音效的加载、缓存和播放。
// 使用 Ebitengine 的 audio 包，支持 WAV 格式，多实例并发播放。
package audio

import (
	"bytes"
	"io"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = 44100 // 采样率（Hz）

// Manager 音效管理器。
type Manager struct {
	context *audio.Context            // Ebitengine 音频上下文
	cache   map[string][]byte         // 音效 PCM 数据缓存（名称 → 解码后数据）
	volume  float64                   // 主音量（0.0 ~ 1.0）
	mu      sync.Mutex                // 并发安全锁
}

// NewManager 创建音效管理器。
func NewManager() *Manager {
	return &Manager{
		context: audio.NewContext(sampleRate),
		cache:   make(map[string][]byte),
		volume:  0.8,
	}
}

// LoadWAV 从字节数据加载 WAV 音效并缓存。
func (m *Manager) LoadWAV(name string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stream, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		return err
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		return err
	}
	m.cache[name] = pcm
	return nil
}

// Play 播放已缓存的音效。未缓存的音效静默忽略。
func (m *Manager) Play(name string) {
	m.mu.Lock()
	pcm, ok := m.cache[name]
	vol := m.volume
	m.mu.Unlock()

	if !ok || vol <= 0 {
		return
	}

	player := m.context.NewPlayerFromBytes(pcm)
	player.SetVolume(vol)
	player.Play()
}

// SetVolume 设置主音量（0.0 ~ 1.0）。
func (m *Manager) SetVolume(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	m.volume = v
}

// Volume 返回当前主音量。
func (m *Manager) Volume() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.volume
}

// IsLoaded 检查音效是否已缓存。
func (m *Manager) IsLoaded(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.cache[name]
	return ok
}

// Count 返回已缓存音效数量。
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.cache)
}

// SFX 预定义音效事件名称常量。
const (
	SFXBuild     = "build"     // 建塔
	SFXSell      = "sell"      // 卖塔
	SFXShoot     = "shoot"     // 射击
	SFXHit       = "hit"       // 命中
	SFXKill      = "kill"      // 击杀
	SFXWaveClear = "waveClear" // 波次通过
	SFXVictory   = "victory"   // 胜利
	SFXDefeat    = "defeat"    // 失败
	SFXLevelUp   = "levelUp"   // 英雄升级
)

// PlaySafe 安全播放音效，出错时仅打印日志不崩溃。
func (m *Manager) PlaySafe(name string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("音效播放异常 %s: %v", name, r)
		}
	}()
	m.Play(name)
}
