// loader.go — 动画帧加载器。
// 从嵌入式文件系统加载多帧 PNG，自动检测帧文件并组装为 Animator。
// 向下兼容：如果没有动画帧文件，回退到单帧静态 PNG。
package anim

import (
	"bytes"
	"fmt"
	"image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// AssetReader 嵌入式资源文件读取接口。
type AssetReader interface {
	ReadFile(name string) ([]byte, error)
}

// TowerAnimConfig 塔动画配置。
var TowerAnimConfig = map[string]struct {
	FPS  float64
	Loop bool
}{
	"idle":   {FPS: 2, Loop: true},
	"attack": {FPS: 8, Loop: false},
}

// EnemyAnimConfig 敌人动画配置。
var EnemyAnimConfig = map[string]struct {
	FPS  float64
	Loop bool
}{
	"walk": {FPS: 6, Loop: true},
	"hit":  {FPS: 10, Loop: false},
}

// LoadTowerAnimator 加载塔的动画帧。
// 路径约定：assets/towers/{key}/tower-{key}-{state}-{frame}.png
// 回退：assets/towers/{key}/tower-{key}.png（单帧）
func LoadTowerAnimator(fs AssetReader, key string) *Animator {
	a := NewAnimator()

	for state, cfg := range TowerAnimConfig {
		frames := loadFrames(fs, fmt.Sprintf("assets/towers/%s/tower-%s-%s", key, key, state))
		if len(frames) > 0 {
			a.AddAnim(state, frames, cfg.FPS, cfg.Loop)
		}
	}

	// 如果没有任何动画帧，尝试加载单帧静态 PNG 作为 idle
	if len(a.Anims) == 0 {
		img := loadSinglePNG(fs, fmt.Sprintf("assets/towers/%s/tower-%s.png", key, key))
		if img != nil {
			a.AddAnim("idle", []*ebiten.Image{img}, 1, true)
		}
	}

	return a
}

// LoadEnemyAnimator 加载敌人的动画帧。
// 路径约定：assets/enemies/{archetype}-{state}-{frame}.png
// 回退：assets/enemies/{archetype}.png（单帧）
func LoadEnemyAnimator(fs AssetReader, archetype string) *Animator {
	a := NewAnimator()

	for state, cfg := range EnemyAnimConfig {
		frames := loadFrames(fs, fmt.Sprintf("assets/enemies/%s-%s", archetype, state))
		if len(frames) > 0 {
			a.AddAnim(state, frames, cfg.FPS, cfg.Loop)
		}
	}

	// 回退到单帧静态 PNG
	if len(a.Anims) == 0 {
		img := loadSinglePNG(fs, fmt.Sprintf("assets/enemies/%s.png", archetype))
		if img != nil {
			a.AddAnim("walk", []*ebiten.Image{img}, 1, true)
		}
	}

	return a
}

// LoadEnemyAnimLib 加载敌人的动画帧库（共享帧数据，不含播放状态）。
// 与 LoadEnemyAnimator 路径约定相同，但返回 AnimLib 而非 Animator。
func LoadEnemyAnimLib(fs AssetReader, archetype string) *AnimLib {
	lib := NewAnimLib()

	for state, cfg := range EnemyAnimConfig {
		frames := loadFrames(fs, fmt.Sprintf("assets/enemies/%s-%s", archetype, state))
		if len(frames) > 0 {
			lib.Anims[state] = &Animation{Frames: frames, FPS: cfg.FPS, Loop: cfg.Loop}
		}
	}

	// 回退到单帧静态 PNG
	if len(lib.Anims) == 0 {
		img := loadSinglePNG(fs, fmt.Sprintf("assets/enemies/%s.png", archetype))
		if img != nil {
			lib.Anims["walk"] = &Animation{Frames: []*ebiten.Image{img}, FPS: 1, Loop: true}
		}
	}

	return lib
}

// WardenAnimConfig 战灵动画配置。
var WardenAnimConfig = map[string]struct {
	FPS  float64
	Loop bool
}{
	"idle":   {FPS: 4, Loop: true},
	"attack": {FPS: 8, Loop: false},
}

// LoadWardenAnimator 加载战灵的动画帧。
// 路径约定：assets/wardens/warden-{type}-{state}-{frame}.png
// 回退：assets/wardens/warden-{type}.png（单帧作为 idle）
func LoadWardenAnimator(fs AssetReader, typ string) *Animator {
	a := NewAnimator()

	for state, cfg := range WardenAnimConfig {
		frames := loadFrames(fs, fmt.Sprintf("assets/wardens/warden-%s-%s", typ, state))
		if len(frames) > 0 {
			a.AddAnim(state, frames, cfg.FPS, cfg.Loop)
		}
	}

	// 如果没有任何动画帧，尝试加载单帧静态 PNG 作为 idle
	if len(a.Anims) == 0 {
		img := loadSinglePNG(fs, fmt.Sprintf("assets/wardens/warden-%s.png", typ))
		if img != nil {
			a.AddAnim("idle", []*ebiten.Image{img}, 1, true)
		}
	}

	return a
}

// loadFrames 尝试加载 {prefix}-0.png, {prefix}-1.png, ... 直到文件不存在。
func loadFrames(fs AssetReader, prefix string) []*ebiten.Image {
	var frames []*ebiten.Image
	for i := 0; i < 16; i++ { // 最多 16 帧
		path := fmt.Sprintf("%s-%d.png", prefix, i)
		img := loadSinglePNG(fs, path)
		if img == nil {
			break
		}
		frames = append(frames, img)
	}
	return frames
}

// loadSinglePNG 加载单个 PNG 文件，失败返回 nil（静默）。
func loadSinglePNG(fs AssetReader, path string) *ebiten.Image {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("anim: PNG 解码失败 %s: %v", path, err)
		return nil
	}
	return ebiten.NewImageFromImage(img)
}
