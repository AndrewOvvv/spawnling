//go:build !linux

package process

// OSProcStats is a no-op on non-Linux platforms.
type OSProcStats struct{}

func (OSProcStats) Read(_ int) (float64, int, error) { return 0, 0, nil }
