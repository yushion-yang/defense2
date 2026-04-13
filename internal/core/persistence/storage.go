// storage.go — 持久化存储。
// 提供基于 JSON 文件的本地存储，功能类似浏览器 localStorage。
// 数据保存在用户目录下的 .defense2/ 文件夹中。
// WASM 环境下会自动回退到内存存储（不持久化）。
package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Storage 持久化存储接口。
type Storage interface {
	Get(key string, target any) error  // 读取并反序列化
	Set(key string, value any) error   // 序列化并写入
	Has(key string) bool                       // 键是否存在
	Delete(key string) error                   // 删除
}

// FileStorage 基于 JSON 文件的本地持久化存储。
type FileStorage struct {
	dir string     // 存储目录路径
	mu  sync.Mutex // 并发安全锁
}

// NewFileStorage 创建文件存储，目录不存在时自动创建。
func NewFileStorage(dir string) (*FileStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}
	return &FileStorage{dir: dir}, nil
}

// DefaultStorage 创建默认位置（~/.defense2/）的存储。
func DefaultStorage() (Storage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &MemoryStorage{data: make(map[string][]byte)}, nil
	}
	dir := filepath.Join(home, ".defense2")
	return NewFileStorage(dir)
}

func (s *FileStorage) path(key string) string {
	return filepath.Join(s.dir, key+".json")
}

// Get 读取并反序列化键值。
func (s *FileStorage) Get(key string, target any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path(key))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// Set 序列化并原子写入键值（write-to-temp-then-rename 防崩溃丢数据）。
func (s *FileStorage) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	target := s.path(key)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

// Has 检查键是否存在。
func (s *FileStorage) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := os.Stat(s.path(key))
	return err == nil
}

// Delete 删除键。
func (s *FileStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.path(key))
}

// MemoryStorage 纯内存存储（WASM 回退用，不持久化）。
type MemoryStorage struct {
	data map[string][]byte // 键 → JSON 字节数据
	mu   sync.Mutex        // 并发安全锁
}

// NewMemoryStorage 创建内存存储实例。
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{data: make(map[string][]byte)}
}

// Get 从内存读取。
func (s *MemoryStorage) Get(key string, target any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.data[key]
	if !ok {
		return fmt.Errorf("key %s not found", key)
	}
	return json.Unmarshal(data, target)
}

// Set 写入内存。
func (s *MemoryStorage) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.data[key] = data
	return nil
}

// Has 检查内存中是否存在。
func (s *MemoryStorage) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[key]
	return ok
}

// Delete 从内存删除。
func (s *MemoryStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}
