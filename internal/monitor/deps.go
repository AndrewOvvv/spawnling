package monitor

import (
	"context"

	"github.com/AndrewOvvv/spawnling/internal/runtime"
)

//go:generate mockery --name=RuntimeStats --output=./mocks

// RuntimeStats is the subset of runtime.Runtime used by the monitor service.
type RuntimeStats interface {
	Stats(ctx context.Context, name string) (runtime.Stats, error)
}
