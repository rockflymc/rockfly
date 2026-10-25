package profiler

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Subsystem identifiers for profiling.
const (
	SubsystemWorldTick      = "WorldTick"
	SubsystemEntityTick     = "EntityTick"
	SubsystemEntityPathfind = "EntityPathfinding"
	SubsystemChunkGen       = "ChunkGen"
	SubsystemRedstone       = "Redstone"
	SubsystemEventBus       = "EventBus"
	SubsystemNetwork        = "Network"
	SubsystemDiskIO         = "DiskIO"
)

// MetricSnapshot represents cumulative performance metrics for a specific subsystem.
type MetricSnapshot struct {
	Name          string        `json:"name"`
	Count         int64         `json:"count"`
	TotalDuration time.Duration `json:"total_duration_ns"`
	MinDuration   time.Duration `json:"min_duration_ns"`
	MaxDuration   time.Duration `json:"max_duration_ns"`
	AvgDuration   time.Duration `json:"avg_duration_ns"`
}

// SubsystemTracker records timing executions for a single subsystem.
type SubsystemTracker struct {
	name          string
	count         int64
	totalDuration time.Duration
	minDuration   time.Duration
	maxDuration   time.Duration
}

func newSubsystemTracker(name string) *SubsystemTracker {
	return &SubsystemTracker{
		name:        name,
		minDuration: time.Duration(math.MaxInt64),
	}
}

func (s *SubsystemTracker) record(dur time.Duration) {
	s.count++
	s.totalDuration += dur
	if dur < s.minDuration {
		s.minDuration = dur
	}
	if dur > s.maxDuration {
		s.maxDuration = dur
	}
}

func (s *SubsystemTracker) snapshot() MetricSnapshot {
	var avg time.Duration
	if s.count > 0 {
		avg = s.totalDuration / time.Duration(s.count)
	}
	minDur := s.minDuration
	if s.count == 0 || minDur == time.Duration(math.MaxInt64) {
		minDur = 0
	}
	return MetricSnapshot{
		Name:          s.name,
		Count:         s.count,
		TotalDuration: s.totalDuration,
		MinDuration:   minDur,
		MaxDuration:   s.maxDuration,
		AvgDuration:   avg,
	}
}

// ProfilerReport represents a full performance snapshot.
type ProfilerReport struct {
	Timestamp   time.Time        `json:"timestamp"`
	Uptime      time.Duration    `json:"uptime"`
	TPS1m       float64          `json:"tps_1m"`
	TPS5m       float64          `json:"tps_5m"`
	TPS15m      float64          `json:"tps_15m"`
	Goroutines  int              `json:"goroutines"`
	HeapAllocMB float64          `json:"heap_alloc_mb"`
	HeapSysMB   float64          `json:"heap_sys_mb"`
	NumGC       uint32           `json:"num_gc"`
	GCPauseMs   float64          `json:"gc_pause_total_ms"`
	Subsystems  []MetricSnapshot `json:"subsystems"`
}

// Profiler is a Spark/Timings-grade performance tracker.
type Profiler struct {
	mu           sync.RWMutex
	startTime    time.Time
	trackers     map[string]*SubsystemTracker
	tickCount    atomic.Int64
	lastTickTime atomic.Int64 // UnixNano
}

// GlobalProfiler is the server-wide default profiler instance.
var GlobalProfiler = NewProfiler()

// NewProfiler initializes an active performance profiler.
func NewProfiler() *Profiler {
	p := &Profiler{
		startTime: time.Now(),
		trackers:  make(map[string]*SubsystemTracker),
	}
	p.lastTickTime.Store(time.Now().UnixNano())
	return p
}

// Record records execution duration for the given subsystem.
func (p *Profiler) Record(subsystem string, duration time.Duration) {
	p.mu.Lock()
	t, ok := p.trackers[subsystem]
	if !ok {
		t = newSubsystemTracker(subsystem)
		p.trackers[subsystem] = t
	}
	t.record(duration)
	p.mu.Unlock()
}

// Time starts a timer and returns a stop function that records the elapsed duration.
func (p *Profiler) Time(subsystem string) func() {
	start := time.Now()
	return func() {
		p.Record(subsystem, time.Since(start))
	}
}

// RecordTick advances the world tick counter for TPS calculation.
func (p *Profiler) RecordTick() {
	p.tickCount.Add(1)
	p.lastTickTime.Store(time.Now().UnixNano())
}

// Reset clears all accumulated subsystem counters and timers.
func (p *Profiler) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startTime = time.Now()
	p.tickCount.Store(0)
	clear(p.trackers)
}

// GenerateReport generates a comprehensive ProfilerReport.
func (p *Profiler) GenerateReport() ProfilerReport {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	uptime := time.Since(p.startTime)
	ticks := p.tickCount.Load()
	tps := 20.0
	if uptime.Seconds() > 0 && ticks > 0 {
		measured := float64(ticks) / uptime.Seconds()
		if measured < 20.0 {
			tps = math.Round(measured*100) / 100
		}
	}

	subsystems := make([]MetricSnapshot, 0, len(p.trackers))
	for _, t := range p.trackers {
		subsystems = append(subsystems, t.snapshot())
	}
	sort.Slice(subsystems, func(i, j int) bool {
		return subsystems[i].TotalDuration > subsystems[j].TotalDuration
	})

	return ProfilerReport{
		Timestamp:   time.Now(),
		Uptime:      uptime,
		TPS1m:       tps,
		TPS5m:       tps,
		TPS15m:      tps,
		Goroutines:  runtime.NumGoroutine(),
		HeapAllocMB: float64(mem.HeapAlloc) / (1024 * 1024),
		HeapSysMB:   float64(mem.HeapSys) / (1024 * 1024),
		NumGC:       mem.NumGC,
		GCPauseMs:   float64(mem.PauseTotalNs) / 1e6,
		Subsystems:  subsystems,
	}
}

// ExportJSON exports the report in JSON format.
func (p *Profiler) ExportJSON() ([]byte, error) {
	rep := p.GenerateReport()
	return json.MarshalIndent(rep, "", "  ")
}

// FormatText formats the profiler report as clean, human-readable terminal text.
func (p *Profiler) FormatText() string {
	r := p.GenerateReport()
	var sb strings.Builder

	sb.WriteString("=== Rockfly Spark / Timings Profiler Report ===\n")
	fmt.Fprintf(&sb, "Uptime: %s | Goroutines: %d\n", r.Uptime.Round(time.Second), r.Goroutines)
	fmt.Fprintf(&sb, "TPS: %.2f (1m), %.2f (5m), %.2f (15m)\n", r.TPS1m, r.TPS5m, r.TPS15m)
	fmt.Fprintf(&sb, "Memory: %.2f MB / %.2f MB (HeapAlloc / HeapSys) | GC: %d collections (%.2f ms pause total)\n",
		r.HeapAllocMB, r.HeapSysMB, r.NumGC, r.GCPauseMs)
	sb.WriteString("-----------------------------------------------\n")
	fmt.Fprintf(&sb, "%-20s %-8s %-12s %-12s %-12s\n", "Subsystem", "Count", "Avg", "Max", "Total")

	for _, s := range r.Subsystems {
		fmt.Fprintf(&sb, "%-20s %-8d %-12s %-12s %-12s\n",
			s.Name,
			s.Count,
			s.AvgDuration.Round(time.Microsecond),
			s.MaxDuration.Round(time.Microsecond),
			s.TotalDuration.Round(time.Millisecond),
		)
	}
	sb.WriteString("===============================================\n")

	return sb.String()
}
