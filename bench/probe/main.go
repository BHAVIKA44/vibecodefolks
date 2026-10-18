package main

import (
	"encoding/json"
	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
	"os"
	"runtime"
	"time"
)

func main() {
	dir := os.Args[1]
	measure := func(f func() error) map[string]float64 {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		t := time.Now()
		if err := f(); err != nil {
			panic(err)
		}
		duration := time.Since(t)
		runtime.ReadMemStats(&after)
		return map[string]float64{"ms": float64(duration.Nanoseconds()) / 1e6, "allocated_bytes": float64(after.TotalAlloc - before.TotalAlloc), "allocations": float64(after.Mallocs - before.Mallocs)}
	}
	result := map[string]any{}
	result["discovery"] = measure(func() error { _, err := analysis.Discover(dir); return err })
	result["analysis"] = measure(func() error { _, err := analysis.Analyze(dir); return err })
	result["evidence"] = measure(func() error { _, err := analysis.Collect(dir); return err })
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
