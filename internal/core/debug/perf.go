// perf.go — frame time + GC performance tracker.
// Records Update/Draw durations in a ring buffer and computes
// per-second aggregate stats (avg, p99, GC count, heap size).
// Designed for zero extra allocations on the hot path.
package debug

import (
	"runtime"
	"slices"
	"time"
)

const historySize = 300

// PerfTracker records per-frame Update/Draw durations and
// periodically computes aggregate statistics (once per second).
type PerfTracker struct {
	updateTimes [historySize]float64 // ring buffer, ms
	drawTimes   [historySize]float64 // ring buffer, ms
	cursor      int
	frameCount  int

	updateStart time.Time
	drawStart   time.Time

	// Public stats (updated once per second via computeStats).
	FPS          float64
	AvgUpdateMs  float64
	AvgDrawMs    float64
	P99UpdateMs  float64
	P99DrawMs    float64
	GCCount      uint32 // GCs in last second
	GCPauseUs    uint64 // total GC pause in last second (microseconds)
	HeapMB       float64
	AllocsPerSec uint64

	lastTick   time.Time
	lastGCNum  uint32
	lastAllocs uint64
	tickFrames int // frames counted since last tick
}

// NewPerfTracker creates a new tracker with zeroed stats.
func NewPerfTracker() *PerfTracker {
	return &PerfTracker{
		lastTick: time.Now(),
	}
}

// BeginUpdate marks the start of the Update phase.
func (pt *PerfTracker) BeginUpdate() {
	pt.updateStart = time.Now()
}

// EndUpdate records the Update phase duration.
func (pt *PerfTracker) EndUpdate() {
	elapsed := time.Since(pt.updateStart).Seconds() * 1000 // ms
	pt.updateTimes[pt.cursor] = elapsed
}

// BeginDraw marks the start of the Draw phase.
func (pt *PerfTracker) BeginDraw() {
	pt.drawStart = time.Now()
}

// EndDraw records the Draw phase duration, advances the ring cursor,
// and triggers per-second stats computation when appropriate.
func (pt *PerfTracker) EndDraw() {
	elapsed := time.Since(pt.drawStart).Seconds() * 1000 // ms
	pt.drawTimes[pt.cursor] = elapsed

	pt.cursor = (pt.cursor + 1) % historySize
	if pt.frameCount < historySize {
		pt.frameCount++
	}
	pt.tickFrames++

	if time.Since(pt.lastTick) >= time.Second {
		pt.computeStats()
	}
}

// computeStats aggregates ring buffer data and reads runtime.MemStats.
// Called approximately once per second.
func (pt *PerfTracker) computeStats() {
	n := pt.frameCount
	if n == 0 {
		return
	}

	// FPS = frames in this tick interval / elapsed seconds
	elapsed := time.Since(pt.lastTick).Seconds()
	if elapsed > 0 {
		pt.FPS = float64(pt.tickFrames) / elapsed
	}

	// Slice the valid portion of ring buffers
	updateSlice := pt.updateTimes[:n]
	drawSlice := pt.drawTimes[:n]

	pt.AvgUpdateMs = Avg(updateSlice)
	pt.AvgDrawMs = Avg(drawSlice)
	pt.P99UpdateMs = Percentile(updateSlice, 99)
	pt.P99DrawMs = Percentile(drawSlice, 99)

	// GC / Heap stats
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	gcNum := uint32(ms.NumGC)
	pt.GCCount = gcNum - pt.lastGCNum
	pt.lastGCNum = gcNum

	// Total GC pause: sum recent pause entries
	pt.GCPauseUs = 0
	if pt.GCCount > 0 {
		count := pt.GCCount
		if count > 256 {
			count = 256
		}
		for i := uint32(0); i < count; i++ {
			idx := (gcNum - 1 - i) % 256
			pt.GCPauseUs += ms.PauseNs[idx] / 1000 // ns -> us
		}
	}

	pt.HeapMB = float64(ms.HeapAlloc) / (1024 * 1024)
	pt.AllocsPerSec = ms.Mallocs - pt.lastAllocs
	pt.lastAllocs = ms.Mallocs

	pt.lastTick = time.Now()
	pt.tickFrames = 0
}

// Avg returns the arithmetic mean of a float64 slice. Returns 0 for empty slices.
func Avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// Percentile returns the pct-th percentile of vals (0-100).
// Makes a copy to avoid mutating the input. Returns 0 for empty slices.
func Percentile(vals []float64, pct int) float64 {
	n := len(vals)
	if n == 0 {
		return 0
	}
	cp := make([]float64, n)
	copy(cp, vals)
	slices.Sort(cp)
	idx := (pct * n) / 100
	if idx >= n {
		idx = n - 1
	}
	return cp[idx]
}
