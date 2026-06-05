# Architecture

## Overview

Spawnling is split into two binaries: a **daemon** (`spawnlingd`) that manages server
processes and holds all state, and a **CLI** (`spawnling`) that talks to it over gRPC.
This separation means a future web UI or TUI can plug in without touching the core logic.

```
┌──────────────────────────────────────────────┐
│                 spawnling CLI                 │
│  create / start / stop / status / logs /      │
│  backup / tunnel / console ...               │
└────────────────────┬─────────────────────────┘
                     │ gRPC over unix socket
                     │ /tmp/spawnling.sock
                     ▼
┌──────────────────────────────────────────────┐
│               spawnling daemon                │
│                                              │
│  ┌──────────────┐ ┌────────────┐ ┌────────┐  │
│  │   Instance   │ │  Monitor  │ │ Tunnel │  │
│  │   Manager   │ │  Service  │ │ Service│  │
│  └──────┬───────┘ └─────┬──────┘ └────────┘  │
│         │               │                    │
│  ┌──────▼───────────────▼──────────────────┐ │
│  │          Runtime (Go interface)          │ │
│  └──────────────────┬───────────────────────┘ │
│                     │                        │
│            ┌────────▼────────┐               │
│            │  ProcessBackend │               │
│            │   (v1, default) │               │
│            └─────────────────┘               │
│                                              │
│  State: ~/.spawnling/instances/<name>/        │
└──────────────────────────────────────────────┘
```

---

## Components

### Instance Manager

Owns the lifecycle of server instances: create, delete, list, configure.
Persists state to `~/.spawnling/instances/<name>/config.json`.
Downloads server JARs from official sources (Paper API, Mojang version manifest).
Manages `server.properties` and JVM arguments.

### Monitor Service

Runs inside the daemon. Polls each running instance on a fixed interval:

| Source | What it provides | Interval |
|---|---|---|
| Process alive check | running / stopped / crashed | 5s |
| `/proc/<pid>/stat` (Linux) | CPU%, RSS | 5s |
| Minecraft Query (UDP) | player list, TPS | 10s |

Aggregates results into an in-memory state snapshot.
Pushes updates to CLI subscribers via gRPC server-side streams.

### Tunnel Service

Wraps the `playit-cli` agent binary. Downloads and caches it on first use.
Manages one tunnel process per server instance — not a daemon-global singleton.
Exposes tunnel address and status through the gRPC API.
Custom `frp` server is supported as an alternative backend via config.

### Runtime

A Go interface that abstracts how server processes are created and managed.
Only `ProcessBackend` is implemented in v1. `K8sBackend` is planned for v2.

```go
type Runtime interface {
    Create(ctx context.Context, spec InstanceSpec) error
    Start(ctx context.Context, name string) error
    Stop(ctx context.Context, name string) error
    Delete(ctx context.Context, name string) error
    Logs(ctx context.Context, name string) (io.ReadCloser, error)
    Exec(ctx context.Context, name string, cmd string) error
    Stats(ctx context.Context, name string) (Stats, error)
}
```

### ProcessBackend

Each server runs as a child JVM process spawned by the daemon via `os/exec`.
The daemon holds stdin/stdout/stderr pipes for the lifetime of the process.

```
daemon
  ├── os/exec → JVM process
  │     ├── stdin  ← send commands directly (no RCON needed for local)
  │     └── stdout/stderr → log ring buffer → gRPC log stream
  ├── RCON (TCP, localhost) → query player list, TPS
  └── /proc/<pid>/stat → CPU%, RSS
```

Auto-restart: a goroutine watches `cmd.Wait()`; if the process exits unexpectedly
and restart is enabled in config, the backend respawns it.

---

## gRPC API

A single `.proto` file defines the full contract between CLI and daemon.
Transport: unix socket `/tmp/spawnling.sock` (localhost only, per NFR-3.1).

| Service | Description |
|---|---|
| `InstanceService` | CRUD, start, stop, status |
| `LogService` | Server-side streaming log tail |
| `ConsoleService` | Bidirectional stream for interactive console |
| `TunnelService` | Start, stop, status for the tunnel |
| `BackupService` | Trigger, list, restore backups |

For the future web UI: a grpc-gateway will expose the same services as HTTP/JSON
on `localhost:7420` alongside the unix socket.

---

## Repository Structure

```
spawnling/
├── cmd/
│   ├── spawnling/        ← CLI entrypoint (cobra commands)
│   └── spawnlingd/       ← daemon entrypoint + manual DI wiring
├── internal/
│   ├── daemon/           ← daemon startup, shutdown, signal handling
│   ├── runtime/
│   │   ├── runtime.go    ← Runtime interface, InstanceSpec, Stats types
│   │   └── process/      ← ProcessBackend
│   ├── monitor/          ← MonitorService
│   ├── tunnel/           ← TunnelService (playit-cli wrapper)
│   ├── backup/           ← BackupService
│   ├── instance/         ← InstanceManager (config, download, state)
│   └── rcon/             ← Minecraft RCON client
├── proto/
│   └── spawnling.proto   ← gRPC service definitions
└── docs/
```

---

## Coding Conventions

### Dependency injection via `deps.go`

Every package declares its external dependencies as interfaces in a dedicated `deps.go`
file. The package implementation depends only on these interfaces, never on concrete types
from other packages.

```go
// internal/instance/deps.go
package instance

type RuntimeBackend interface {
    Create(ctx context.Context, spec InstanceSpec) error
    Start(ctx context.Context, name string) error
    Stop(ctx context.Context, name string) error
    Delete(ctx context.Context, name string) error
}

type ConfigStore interface {
    Load(name string) (*Config, error)
    Save(name string, cfg *Config) error
    Delete(name string) error
    List() ([]string, error)
}
```

This makes every package independently testable — swap any dependency for a mock
without touching production code.

### Constructor naming

Constructors are named after what they construct, not just `New`:

```go
func NewInstanceManager(rt RuntimeBackend, store ConfigStore) *Manager
func NewMonitorService(rt RuntimeBackend, state StateStore) *MonitorService
func NewProcessBackend(cfg Config) *ProcessBackend
```

### Wiring

All concrete dependencies are assembled in `cmd/spawnlingd/main.go`.
No DI framework — explicit manual wiring is intentional: the full dependency graph
is readable in one place.

```go
// cmd/spawnlingd/main.go
store   := sqlite.NewConfigStore(dbPath)
backend := process.NewProcessBackend(cfg)
manager := instance.NewInstanceManager(backend, store)
monitor := monitor.NewMonitorService(backend, store)
// ...
```

### Testing

Tests use the **external package** pattern (`package instance_test`).
They test the public API from a consumer's perspective — no access to unexported fields.

Mocks are generated by [mockery](https://github.com/vektra/mockery) from the interfaces
in `deps.go`. Each package keeps its mocks in a `mocks/` subdirectory.

```go
// internal/instance/manager_test.go
package instance_test

import (
    "testing"
    "github.com/AndrewOvvv/spawnling/internal/instance"
    "github.com/AndrewOvvv/spawnling/internal/instance/mocks"
)

func TestCreate(t *testing.T) {
    rt    := mocks.NewMockRuntimeBackend(t)
    store := mocks.NewMockConfigStore(t)
    rt.On("Create", mock.Anything, mock.Anything).Return(nil)

    mgr := instance.NewInstanceManager(rt, store)
    err := mgr.Create(context.Background(), instance.Spec{Name: "test"})

    assert.NoError(t, err)
    rt.AssertExpectations(t)
}
```

Generate mocks with:
```bash
go generate ./...
```

Each `deps.go` carries the corresponding `//go:generate` directives.

---

## K8s Backend (v2, deferred)

When implemented: no sidecar. The daemon talks directly to the Kubernetes API.

| Source | What it provides |
|---|---|
| K8s API (client-go) | pod status, events |
| K8s Metrics Server | CPU%, RAM |
| client-go log stream | stdout (equivalent to `kubectl logs -f`) |
| RCON TCP via K8s Service | player list, commands |

Each server instance maps to one `Deployment` + one `Service`.
The `K8sBackend` implements the same `Runtime` interface — no changes to
`InstanceManager`, `MonitorService`, or the gRPC API.
