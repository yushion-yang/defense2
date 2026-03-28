// probes.go — 运行时探针。
// 零开销的计数探针，用于验证代码路径是否被执行。
// 生产环境中不使用时无额外开销（sync.Map 空查找）。
package debug

import (
	"sync"
	"sync/atomic"
)

// probeHits 探针命中计数（原子操作，零开销设计）。
var probeHits sync.Map

// probeCounter 单个探针的原子计数器。
type probeCounter struct {
	count int64
}

// Probe 记录一次探针命中（原子操作，并发安全）。
func Probe(id string) {
	v, _ := probeHits.LoadOrStore(id, &probeCounter{})
	c := v.(*probeCounter)
	atomic.AddInt64(&c.count, 1)
}

// GetProbeCount 获取指定探针的命中次数。
func GetProbeCount(id string) int64 {
	v, ok := probeHits.Load(id)
	if !ok {
		return 0
	}
	return atomic.LoadInt64(&v.(*probeCounter).count)
}

// GetProbeReport 返回所有探针的命中报告。
func GetProbeReport() map[string]int64 {
	report := make(map[string]int64)
	probeHits.Range(func(key, value interface{}) bool {
		id := key.(string)
		c := value.(*probeCounter)
		report[id] = atomic.LoadInt64(&c.count)
		return true
	})
	return report
}

// ResetProbes 重置所有探针计数（测试用）。
func ResetProbes() {
	probeHits.Range(func(key, _ interface{}) bool {
		probeHits.Delete(key)
		return true
	})
}

// GetUnhitProbes 返回期望列表中从未命中的探针 ID。
// 用于验证所有关键代码路径都被覆盖。
func GetUnhitProbes(expected []string) []string {
	var unhit []string
	for _, id := range expected {
		if GetProbeCount(id) == 0 {
			unhit = append(unhit, id)
		}
	}
	return unhit
}
