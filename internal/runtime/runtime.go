package runtime

import (
	"context"
	"io"
	"time"
)

type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusCrashed  Status = "crashed"
)

type ServerType string

const (
	ServerTypeVanilla ServerType = "vanilla"
	ServerTypePaper   ServerType = "paper"
	ServerTypeFabric  ServerType = "fabric"
	ServerTypeForge   ServerType = "forge"
	ServerTypeQuilt   ServerType = "quilt"
)

type InstanceSpec struct {
	Name        string
	Type        ServerType
	Version     string
	Port        int
	MaxMemoryMB int
	MinMemoryMB int
	JVMArgs     []string
	AutoRestart bool
}

type Stats struct {
	Status    Status
	CPUPct    float64
	MemoryMB  int
	Players   []string
	MaxPlayer int
	Uptime    time.Duration
	PID       int
}

// Runtime abstracts how server processes are created and managed.
// ProcessBackend is the only implementation in v1; K8sBackend is planned for v2.
type Runtime interface {
	Create(ctx context.Context, spec InstanceSpec) error
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
	Logs(ctx context.Context, name string) (io.ReadCloser, error)
	Exec(ctx context.Context, name string, cmd string) error
	Stats(ctx context.Context, name string) (Stats, error)
}
