package admin

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/profiler"
	"github.com/df-mc/dragonfly/server/world"
)

func registerTimings() {
	cmd.Register(cmd.New("timings", "View server performance metrics, TPS, and profiler timings", []string{"spark", "tps"},
		TimingsSummary{},
		TimingsReport{},
		TimingsPaste{},
		TimingsReset{},
	))
}

// TimingsSummary prints a quick overview of server TPS, memory, and top timings.
type TimingsSummary struct{}

func (c TimingsSummary) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c TimingsSummary) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	rep := profiler.GlobalProfiler.GenerateReport()
	o.Printf("Rockfly TPS: %.2f (1m), %.2f (5m), %.2f (15m) | Memory: %.1f MB | Goroutines: %d",
		rep.TPS1m, rep.TPS5m, rep.TPS15m, rep.HeapAllocMB, rep.Goroutines)
}

// TimingsReport outputs the detailed ASCII breakdown of all subsystems.
type TimingsReport struct {
	Report cmd.SubCommand `cmd:"report"`
}

func (c TimingsReport) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c TimingsReport) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	text := profiler.GlobalProfiler.FormatText()
	o.Printf("%s", text)
}

// TimingsPaste outputs the full detailed breakdown (PaperMC /timings paste alias).
type TimingsPaste struct {
	Paste cmd.SubCommand `cmd:"paste"`
}

func (c TimingsPaste) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c TimingsPaste) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	text := profiler.GlobalProfiler.FormatText()
	o.Printf("%s", text)
}

// TimingsReset clears profiler data.
type TimingsReset struct {
	Reset cmd.SubCommand `cmd:"reset"`
}

func (c TimingsReset) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c TimingsReset) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	profiler.GlobalProfiler.Reset()
	o.Printf("Performance profiler timers and counters have been reset.")
}
