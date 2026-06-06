package process

import (
	"context"
	"io"
	"io/fs"

	"github.com/AndrewOvvv/spawnling/internal/runtime"
)

//go:generate mockery --name=FileSystem --output=./mocks
//go:generate mockery --name=ProcessSpawner --output=./mocks
//go:generate mockery --name=ProcessHandle --output=./mocks
//go:generate mockery --name=JARProvider --output=./mocks
//go:generate mockery --name=ProcStats --output=./mocks

// FileSystem abstracts OS file I/O.
type FileSystem interface {
	MkdirAll(path string, perm fs.FileMode) error
	WriteFile(path string, data []byte, perm fs.FileMode) error
	ReadFile(path string) ([]byte, error)
	Exists(path string) (bool, error)
	RemoveAll(path string) error
}

// ProcessHandle represents a running server process.
type ProcessHandle interface {
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Wait() error
	Kill() error
	Pid() int
}

// ProcessSpawner starts a server process and returns a handle to it.
type ProcessSpawner interface {
	Spawn(javaPath string, args []string, dir string) (ProcessHandle, error)
}

// JARProvider ensures the server JAR is present at destPath,
// downloading it if necessary.
type JARProvider interface {
	Ensure(ctx context.Context, spec runtime.InstanceSpec, destPath string) error
}

// ProcStats reads CPU and memory usage for a given PID.
type ProcStats interface {
	Read(pid int) (cpuPct float64, memMB int, err error)
}
