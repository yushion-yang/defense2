// manager.go — 音效管理器。
// 统一管理游戏音效的加载、缓存和播放。
// 使用 Ebitengine 的 audio 包，支持 WAV 格式，多实例并发播放。
package audio

import (
	"bytes"
	"io"
	"log"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = 44100 // 采样率（Hz）

// audioContext 全局唯一的 Ebitengine 音频上下文（Ebitengine 限制只能创建一次）。
var (
	audioContext     *audio.Context
	audioContextOnce sync.Once
)

func getAudioContext() *audio.Context {
	audioContextOnce.Do(func() {
		audioContext = audio.NewContext(sampleRate)
	})
	return audioContext
}

// Manager 音效管理器。
type Manager struct {
	context  *audio.Context       // Ebitengine 音频上下文
	cache    map[string][]byte    // 音效 PCM 数据缓存（名称 → 解码后数据）
	volume   float64              // 主音量（0.0 ~ 1.0）
	throttle map[string]time.Time // 每个音效的上次播放时间（per-sound 节流）
	mu       sync.Mutex           // 并发安全锁
}

// NewManager 创建音效管理器。
func NewManager() *Manager {
	return &Manager{
		context:  getAudioContext(),
		cache:    make(map[string][]byte),
		throttle: make(map[string]time.Time),
		volume:   0.8,
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
// 名称必须与 WAV 文件名经 kebabToCamel 转换后一致。
const (
	SFXBuild            = "build"            // build.wav — 建塔
	SFXTowerSell        = "towerSell"        // tower-sell.wav — 卖塔
	SFXShot             = "shot"             // shot.wav — 射击
	SFXHit              = "hit"              // hit.wav — 命中
	SFXEnemyDeath       = "enemyDeath"       // enemy-death.wav — 击杀
	SFXEnemyDeathElite  = "enemyDeathElite"  // enemy-death-elite.wav — 精英击杀
	SFXEnemyDeathBoss   = "enemyDeathBoss"   // enemy-death-boss.wav — Boss击杀
	SFXEnemyLeak        = "enemyLeak"        // enemy-leak.wav — 敌人泄漏
	SFXWaveStart        = "waveStart"        // wave-start.wav — 波次开始
	SFXWaveClear        = "waveClear"        // wave-clear.wav — 波次通过
	SFXWaveClearPerfect = "waveClearPerfect" // wave-clear-perfect.wav — 完美通过
	SFXVictory          = "victory"          // victory.wav — 胜利
	SFXDefeat           = "defeat"           // defeat.wav — 失败
	SFXGoldEarn         = "goldEarn"         // gold-earn.wav — 获得金币
	SFXUIClick          = "uiClick"          // ui-click.wav — UI点击
	SFXExplode          = "explode"          // explode.wav — 爆炸
	SFXUpgrade          = "upgrade"          // upgrade.wav — 升级塔
	SFXSpeedToggle      = "speedToggle"      // speed-toggle.wav — 变速
	SFXUIOpen           = "uiOpen"           // ui-open.wav — 打开面板
	SFXUIClose          = "uiClose"          // ui-close.wav — 关闭面板
	SFXBossEnter        = "bossEnter"        // boss-enter.wav — Boss出场
	SFXHitFlesh         = "hitFlesh"         // hit-flesh.wav — 命中普通敌人
	SFXHitHeavy         = "hitHeavy"         // hit-heavy.wav — 命中Boss/Tank
	SFXHitShield        = "hitShield"        // hit-shield.wav — 命中护盾
	SFXShieldBreak      = "shieldBreak"      // shield-break.wav — 护盾击碎
	SFXChoiceAppear     = "choiceAppear"     // choice-appear.wav — 事件弹窗出现
	SFXChoiceSelect     = "choiceSelect"     // choice-select.wav — 事件选择
)

// FireSFXForStyle 根据攻击方式返回射击音效名称。
// 如果对应风格的 WAV 不存在，降级到通用 SFXShot。
func FireSFXForStyle(style string) string {
	name := "fire" + ucFirst(style) // 如 "fireProjectile", "fireLaser"
	// 降级：如果没有对应文件就用通用射击音效
	if name == "fire" {
		return SFXShot
	}
	return name
}

// HitSFXForStyle 根据攻击方式返回命中音效名称。
// 降级到 SFXHitFlesh。
func HitSFXForStyle(style string) string {
	name := "hit" + ucFirst(style) // 如 "hitProjectile", "hitLaser"
	if name == "hit" {
		return SFXHitFlesh
	}
	return name
}

// ucFirst 首字母大写（简单 ASCII）。
func ucFirst(s string) string {
	if s == "" {
		return ""
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	return string(b)
}

// PlayThrottled 带 per-sound 节流的播放。
// intervalMs=0 时等同于 Play（无节流）。
// 同一 name 在 intervalMs 毫秒内只播放一次，后续调用静默跳过。
func (m *Manager) PlayThrottled(name string, intervalMs int) {
	if intervalMs > 0 {
		now := time.Now()
		m.mu.Lock()
		last, ok := m.throttle[name]
		if ok && now.Sub(last) < time.Duration(intervalMs)*time.Millisecond {
			m.mu.Unlock()
			return
		}
		m.throttle[name] = now
		m.mu.Unlock()
	}
	m.Play(name)
}

// PlaySafe 安全播放音效，出错时仅打印日志不崩溃。
func (m *Manager) PlaySafe(name string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("音效播放异常 %s: %v", name, r)
		}
	}()
	m.Play(name)
}
