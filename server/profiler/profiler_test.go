package profiler

import (
	"strings"
	"testing"
	"time"
)

func TestProfiler_RecordAndReport(t *testing.T) {
	prof := NewProfiler()

	prof.Record(SubsystemWorldTick, 5*time.Millisecond)
	prof.Record(SubsystemWorldTick, 15*time.Millisecond)
	prof.Record(SubsystemEntityTick, 2*time.Millisecond)
	prof.Record(SubsystemRedstone, 1*time.Millisecond)

	rep := prof.GenerateReport()

	if len(rep.Subsystems) != 3 {
		t.Fatalf("expected 3 subsystems, got %d", len(rep.Subsystems))
	}

	// WorldTick should be first (sorted by total duration descending: 20ms)
	wt := rep.Subsystems[0]
	if wt.Name != SubsystemWorldTick {
		t.Fatalf("expected first subsystem to be WorldTick, got %s", wt.Name)
	}
	if wt.Count != 2 {
		t.Fatalf("expected 2 WorldTick records, got %d", wt.Count)
	}
	if wt.TotalDuration != 20*time.Millisecond {
		t.Fatalf("expected 20ms total duration, got %v", wt.TotalDuration)
	}
	if wt.AvgDuration != 10*time.Millisecond {
		t.Fatalf("expected 10ms avg duration, got %v", wt.AvgDuration)
	}

	// JSON Export
	data, err := prof.ExportJSON()
	if err != nil {
		t.Fatalf("failed to export JSON: %v", err)
	}
	if !strings.Contains(string(data), "WorldTick") {
		t.Fatalf("expected JSON to contain WorldTick, got %s", string(data))
	}

	// Formatted Text Output
	text := prof.FormatText()
	if !strings.Contains(text, "Rockfly Spark / Timings Profiler Report") || !strings.Contains(text, "WorldTick") {
		t.Fatalf("expected format text to contain header and WorldTick, got:\n%s", text)
	}
}

func TestProfiler_TimeFunction(t *testing.T) {
	prof := NewProfiler()

	func() {
		defer prof.Time(SubsystemChunkGen)()
		time.Sleep(2 * time.Millisecond)
	}()

	rep := prof.GenerateReport()
	if len(rep.Subsystems) != 1 || rep.Subsystems[0].Name != SubsystemChunkGen {
		t.Fatalf("expected 1 ChunkGen subsystem")
	}
	if rep.Subsystems[0].TotalDuration < 2*time.Millisecond {
		t.Fatalf("expected duration >= 2ms, got %v", rep.Subsystems[0].TotalDuration)
	}
}

func TestProfiler_Reset(t *testing.T) {
	prof := NewProfiler()
	prof.Record(SubsystemNetwork, 10*time.Millisecond)

	rep := prof.GenerateReport()
	if len(rep.Subsystems) != 1 {
		t.Fatalf("expected 1 subsystem before reset")
	}

	prof.Reset()
	rep = prof.GenerateReport()
	if len(rep.Subsystems) != 0 {
		t.Fatalf("expected 0 subsystems after reset, got %d", len(rep.Subsystems))
	}
}

func TestProfiler_NilAndZeroSafety(t *testing.T) {
	var nilProf *Profiler
	nilProf.Record("Test", 10*time.Millisecond)
	fn := nilProf.Time("Test")
	fn()
	nilProf.RecordTick()
	nilProf.Reset()
	rep := nilProf.GenerateReport()
	if rep.TPS1m != 20.0 {
		t.Fatalf("expected 20.0 TPS from nil profiler")
	}
	jsonBytes, err := nilProf.ExportJSON()
	if err != nil || len(jsonBytes) == 0 {
		t.Fatalf("expected valid JSON from nil profiler")
	}
	text := nilProf.FormatText()
	if len(text) == 0 {
		t.Fatalf("expected non-empty formatted text from nil profiler")
	}

	zeroProf := &Profiler{}
	zeroProf.Record("SubsystemA", -5*time.Millisecond)
	zeroProf.RecordTick()
	rep2 := zeroProf.GenerateReport()
	if len(rep2.Subsystems) != 1 || rep2.Subsystems[0].TotalDuration != 0 {
		t.Fatalf("expected negative duration to be clamped to 0")
	}
	zeroProf.Reset()
}

func BenchmarkProfiler_Record(b *testing.B) {
	prof := NewProfiler()
	dur := 5 * time.Millisecond

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		prof.Record(SubsystemWorldTick, dur)
	}
}

func BenchmarkProfiler_TimeWrapper(b *testing.B) {
	prof := NewProfiler()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		stop := prof.Time(SubsystemEntityTick)
		stop()
	}
}
