package process

import (
	"context"
	"io"

	"github.com/AndrewOvvv/spawnling/internal/runtime"
)

type Config struct {
	DataDir  string
	JavaPath string
}

type ProcessBackend struct {
	cfg Config
	fs  FileSystem
}

func NewProcessBackend(cfg Config, fs FileSystem) *ProcessBackend {
	return &ProcessBackend{cfg: cfg, fs: fs}
}

func (b *ProcessBackend) Create(ctx context.Context, spec runtime.InstanceSpec) error {
	panic("not implemented")
}

func (b *ProcessBackend) Start(ctx context.Context, name string) error {
	panic("not implemented")
}

func (b *ProcessBackend) Stop(ctx context.Context, name string) error {
	panic("not implemented")
}

func (b *ProcessBackend) Delete(ctx context.Context, name string) error {
	panic("not implemented")
}

func (b *ProcessBackend) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	panic("not implemented")
}

func (b *ProcessBackend) Exec(ctx context.Context, name string, cmd string) error {
	panic("not implemented")
}

func (b *ProcessBackend) Stats(ctx context.Context, name string) (runtime.Stats, error) {
	panic("not implemented")
}
