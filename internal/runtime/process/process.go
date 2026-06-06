package process

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AndrewOvvv/spawnling/internal/runtime"
)

const stopTimeout = 30 * time.Second

// Config holds ProcessBackend configuration.
type Config struct {
	DataDir  string // root directory for instance data, e.g. ~/.spawnling/instances
	JavaPath string // path to java binary; defaults to "java"
}

type serverProcess struct {
	spec      runtime.InstanceSpec
	handle    ProcessHandle
	logs      *logBuffer
	done      chan struct{} // closed when the process exits
	stopping  atomic.Bool  // set by Stop to suppress auto-restart
	startedAt time.Time
}

// ProcessBackend manages Minecraft server processes as child JVM processes.
type ProcessBackend struct {
	cfg     Config
	fs      FileSystem
	spawner ProcessSpawner
	jars    JARProvider
	stats   ProcStats
	mu      sync.RWMutex
	running map[string]*serverProcess
}

func NewProcessBackend(cfg Config, fs FileSystem, spawner ProcessSpawner, jars JARProvider, stats ProcStats) *ProcessBackend {
	if cfg.JavaPath == "" {
		cfg.JavaPath = "java"
	}
	return &ProcessBackend{
		cfg:     cfg,
		fs:      fs,
		spawner: spawner,
		jars:    jars,
		stats:   stats,
		running: make(map[string]*serverProcess),
	}
}

// Create sets up the instance directory and persists the spec.
// The server JAR is downloaded on the first Start, not here.
func (b *ProcessBackend) Create(ctx context.Context, spec runtime.InstanceSpec) error {
	dir := b.instanceDir(spec.Name)

	if err := b.fs.MkdirAll(filepath.Join(dir, "server"), fs.FileMode(0o755)); err != nil {
		return fmt.Errorf("process: mkdir server for %q: %w", spec.Name, err)
	}
	if err := b.fs.MkdirAll(filepath.Join(dir, "backups"), fs.FileMode(0o755)); err != nil {
		return fmt.Errorf("process: mkdir backups for %q: %w", spec.Name, err)
	}

	specData, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	if err := b.fs.WriteFile(filepath.Join(dir, "spec.json"), specData, fs.FileMode(0o644)); err != nil {
		return fmt.Errorf("process: write spec for %q: %w", spec.Name, err)
	}

	props := defaultServerProperties(spec)
	if err := b.fs.WriteFile(filepath.Join(dir, "server", "server.properties"), []byte(props), fs.FileMode(0o644)); err != nil {
		return fmt.Errorf("process: write server.properties for %q: %w", spec.Name, err)
	}
	return nil
}

// Start downloads the JAR if needed, then spawns the JVM process.
func (b *ProcessBackend) Start(ctx context.Context, name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.running[name]; ok {
		return fmt.Errorf("process: %q is already running", name)
	}

	dir := b.instanceDir(name)
	spec, err := b.loadSpec(dir)
	if err != nil {
		return fmt.Errorf("process: load spec for %q: %w", name, err)
	}

	jarPath := filepath.Join(dir, "server", "server.jar")
	if err := b.jars.Ensure(ctx, spec, jarPath); err != nil {
		return fmt.Errorf("process: ensure JAR for %q: %w", name, err)
	}

	eulaPath := filepath.Join(dir, "server", "eula.txt")
	if exists, _ := b.fs.Exists(eulaPath); !exists {
		if err := b.fs.WriteFile(eulaPath, []byte("eula=true\n"), fs.FileMode(0o644)); err != nil {
			return fmt.Errorf("process: write eula for %q: %w", name, err)
		}
	}

	handle, err := b.spawner.Spawn(b.cfg.JavaPath, buildJVMArgs(spec), filepath.Join(dir, "server"))
	if err != nil {
		return fmt.Errorf("process: spawn %q: %w", name, err)
	}

	sp := &serverProcess{
		spec:      spec,
		handle:    handle,
		logs:      newLogBuffer(),
		done:      make(chan struct{}),
		startedAt: time.Now(),
	}
	b.running[name] = sp

	go b.pipeOutput(sp, handle.Stdout())
	go b.watchExit(ctx, name, sp)

	return nil
}

// Stop sends the "stop" command to the server and waits for it to exit.
// If the server does not exit within stopTimeout, it is killed.
func (b *ProcessBackend) Stop(_ context.Context, name string) error {
	b.mu.RLock()
	sp, ok := b.running[name]
	b.mu.RUnlock()
	if !ok {
		return fmt.Errorf("process: %q is not running", name)
	}

	sp.stopping.Store(true)

	if _, err := fmt.Fprintf(sp.handle.Stdin(), "stop\n"); err != nil {
		return sp.handle.Kill()
	}

	select {
	case <-sp.done:
		return nil
	case <-time.After(stopTimeout):
		return sp.handle.Kill()
	}
}

// Delete removes the instance directory. The instance must not be running.
func (b *ProcessBackend) Delete(_ context.Context, name string) error {
	b.mu.RLock()
	_, running := b.running[name]
	b.mu.RUnlock()
	if running {
		return fmt.Errorf("process: stop %q before deleting", name)
	}
	if err := b.fs.RemoveAll(b.instanceDir(name)); err != nil {
		return fmt.Errorf("process: delete %q: %w", name, err)
	}
	return nil
}

// Logs returns a stream of log lines (buffered history + live output).
func (b *ProcessBackend) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	b.mu.RLock()
	sp, ok := b.running[name]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("process: %q is not running", name)
	}
	return sp.logs.reader(ctx), nil
}

// Exec writes a command to the server's stdin.
func (b *ProcessBackend) Exec(_ context.Context, name string, cmd string) error {
	b.mu.RLock()
	sp, ok := b.running[name]
	b.mu.RUnlock()
	if !ok {
		return fmt.Errorf("process: %q is not running", name)
	}
	_, err := fmt.Fprintf(sp.handle.Stdin(), "%s\n", cmd)
	return err
}

// Stats returns the current status and resource usage of an instance.
func (b *ProcessBackend) Stats(_ context.Context, name string) (runtime.Stats, error) {
	b.mu.RLock()
	sp, ok := b.running[name]
	b.mu.RUnlock()

	if !ok {
		if exists, _ := b.fs.Exists(filepath.Join(b.instanceDir(name), "spec.json")); exists {
			return runtime.Stats{Status: runtime.StatusStopped}, nil
		}
		return runtime.Stats{}, fmt.Errorf("process: instance %q not found", name)
	}

	cpuPct, memMB, err := b.stats.Read(sp.handle.Pid())
	if err != nil {
		cpuPct, memMB = 0, 0
	}

	return runtime.Stats{
		Status:   runtime.StatusRunning,
		CPUPct:   cpuPct,
		MemoryMB: memMB,
		Uptime:   time.Since(sp.startedAt),
		PID:      sp.handle.Pid(),
	}, nil
}

// ── internal ─────────────────────────────────────────────────────────────────

func (b *ProcessBackend) instanceDir(name string) string {
	return filepath.Join(b.cfg.DataDir, name)
}

func (b *ProcessBackend) loadSpec(dir string) (runtime.InstanceSpec, error) {
	data, err := b.fs.ReadFile(filepath.Join(dir, "spec.json"))
	if err != nil {
		return runtime.InstanceSpec{}, err
	}
	var spec runtime.InstanceSpec
	return spec, json.Unmarshal(data, &spec)
}

func (b *ProcessBackend) pipeOutput(sp *serverProcess, stdout io.ReadCloser) {
	defer stdout.Close()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		sp.logs.writeLine(scanner.Text())
	}
	sp.logs.close()
}

func (b *ProcessBackend) watchExit(ctx context.Context, name string, sp *serverProcess) {
	sp.handle.Wait() //nolint:errcheck

	close(sp.done)

	b.mu.Lock()
	delete(b.running, name)
	b.mu.Unlock()

	if sp.spec.AutoRestart && !sp.stopping.Load() {
		time.Sleep(2 * time.Second)
		b.Start(ctx, name) //nolint:errcheck
	}
}

func buildJVMArgs(spec runtime.InstanceSpec) []string {
	return []string{
		fmt.Sprintf("-Xmx%dM", spec.MaxMemoryMB),
		fmt.Sprintf("-Xms%dM", spec.MinMemoryMB),
		"-jar", "server.jar",
		"--nogui",
	}
}

func defaultServerProperties(spec runtime.InstanceSpec) string {
	return fmt.Sprintf(
		"server-port=%d\nmax-players=20\nonline-mode=true\n",
		spec.Port,
	)
}
