// loader.go — 音效批量加载器。
// 从嵌入式文件系统扫描 assets/audio/ 下的 WAV 文件并预加载到音效管理器。
// 文件名（去除扩展名）作为音效名称，如 "build.wav" → "build"。
package audio

import (
	"embed"
	"log"
	"path/filepath"
	"strings"
)

// LoadAllFromFS 从嵌入式文件系统批量加载所有 WAV 音效。
// 同时保存 FS 引用供 BGM 按需加载使用。
func (m *Manager) LoadAllFromFS(fs *embed.FS) {
	if fs == nil {
		return
	}
	m.assetFS = fs

	entries, err := fs.ReadDir("assets/audio")
	if err != nil {
		log.Printf("读取音效目录失败: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".wav") {
			continue
		}

		sfxName := strings.TrimSuffix(name, filepath.Ext(name))
		// kebab-case → camelCase（与 JS 版约定一致）
		sfxName = kebabToCamel(sfxName)

		data, err := fs.ReadFile("assets/audio/" + name)
		if err != nil {
			log.Printf("读取音效 %s 失败: %v", name, err)
			continue
		}

		if err := m.LoadWAV(sfxName, data); err != nil {
			log.Printf("解码音效 %s 失败: %v", name, err)
		}
	}

	log.Printf("已加载 %d 个音效", m.Count())
}

// WAVEntry 单个 WAV 文件的加载条目。
type WAVEntry struct {
	FileName string // 原始文件名 (e.g. "tower-build.wav")
	SFXName  string // camelCase 音效名 (e.g. "towerBuild")
}

// ScanWAVEntries 扫描嵌入 FS 中的 WAV 文件列表（不加载数据）。
// 同时保存 FS 引用供后续 LoadWAVEntry / BGM 使用。
func (m *Manager) ScanWAVEntries(fs *embed.FS) []WAVEntry {
	if fs == nil {
		return nil
	}
	m.assetFS = fs
	entries, err := fs.ReadDir("assets/audio")
	if err != nil {
		log.Printf("读取音效目录失败: %v", err)
		return nil
	}
	var result []WAVEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".wav") {
			continue
		}
		sfxName := strings.TrimSuffix(name, filepath.Ext(name))
		sfxName = kebabToCamel(sfxName)
		result = append(result, WAVEntry{FileName: name, SFXName: sfxName})
	}
	return result
}

// LoadWAVEntry 加载单个 WAV 条目。调用前必须先调用 ScanWAVEntries 设置 FS。
func (m *Manager) LoadWAVEntry(entry WAVEntry) error {
	data, err := m.assetFS.ReadFile("assets/audio/" + entry.FileName)
	if err != nil {
		return err
	}
	return m.LoadWAV(entry.SFXName, data)
}

// kebabToCamel 将 kebab-case 转为 camelCase。
// 如 "tower-build" → "towerBuild"。
func kebabToCamel(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) <= 1 {
		return s
	}
	result := parts[0]
	for _, p := range parts[1:] {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}
