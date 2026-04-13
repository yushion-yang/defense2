// storage_js.go — WASM 环境下基于浏览器 localStorage 的持久化存储。
//
// 通过 syscall/js 调用 window.localStorage API，数据以 JSON 字符串存储，
// key 统一加 "defense2:" 前缀避免与其他应用冲突。
// 刷新页面后进度保留，清除浏览器数据才会丢失。

//go:build js

package persistence

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

const lsPrefix = "defense2:" // localStorage key 前缀

// LocalStorage 基于浏览器 localStorage 的持久化存储（仅 WASM）。
type LocalStorage struct {
	storage js.Value
}

// NewLocalStorage 创建 localStorage 存储实例。
// 若浏览器不支持或处于隐私模式导致 localStorage 不可用，返回 error。
func NewLocalStorage() (*LocalStorage, error) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return nil, fmt.Errorf("localStorage not available")
	}
	return &LocalStorage{storage: ls}, nil
}

// Get 从 localStorage 读取并反序列化。
func (s *LocalStorage) Get(key string, target any) error {
	val := s.storage.Call("getItem", lsPrefix+key)
	if val.IsNull() || val.IsUndefined() {
		return fmt.Errorf("key %s not found", key)
	}
	return json.Unmarshal([]byte(val.String()), target)
}

// Set 序列化并写入 localStorage。
func (s *LocalStorage) Set(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.storage.Call("setItem", lsPrefix+key, string(data))
	return nil
}

// Has 检查 key 是否存在。
func (s *LocalStorage) Has(key string) bool {
	val := s.storage.Call("getItem", lsPrefix+key)
	return !val.IsNull() && !val.IsUndefined()
}

// Delete 删除 key。
func (s *LocalStorage) Delete(key string) error {
	s.storage.Call("removeItem", lsPrefix+key)
	return nil
}

// lsSingleton localStorage 单例，确保多个场景共享同一实例。
var lsSingleton *LocalStorage

func init() {
	defaultStorageOverride = func() (Storage, error) {
		if lsSingleton != nil {
			return lsSingleton, nil
		}
		ls, err := NewLocalStorage()
		if err != nil {
			return nil, err
		}
		lsSingleton = ls
		return ls, nil
	}
}
