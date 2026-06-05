package instance

import (
	"context"

	"github.com/AndrewOvvv/spawnling/internal/runtime"
)

//go:generate mockery --name=RuntimeBackend --output=./mocks
//go:generate mockery --name=ConfigStore --output=./mocks

// RuntimeBackend is the subset of runtime.Runtime used by the instance manager.
type RuntimeBackend interface {
	Create(ctx context.Context, spec runtime.InstanceSpec) error
	Delete(ctx context.Context, name string) error
}

// ConfigStore persists instance configuration.
type ConfigStore interface {
	Load(name string) (*Config, error)
	Save(name string, cfg *Config) error
	Delete(name string) error
	List() ([]string, error)
}
