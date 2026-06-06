package process_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AndrewOvvv/spawnling/internal/runtime"
	"github.com/AndrewOvvv/spawnling/internal/runtime/process"
	"github.com/AndrewOvvv/spawnling/internal/runtime/process/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── FileSystem stub ───────────────────────────────────────────────────────────

type memFS struct {
	mu   sync.RWMutex
	data map[string][]byte
	dirs map[string]bool
}

func newMemFS() *memFS {
	return &memFS{
		data: make(map[string][]byte),
		dirs: make(map[string]bool),
	}
}

func (f *memFS) MkdirAll(path string, _ fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dirs[path] = true
	return nil
}

func (f *memFS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	f.data[path] = cp
	return nil
}

func (f *memFS) ReadFile(path string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	d, ok := f.data[path]
	if !ok {
		return nil, fmt.Errorf("memFS: %q not found", path)
	}
	return d, nil
}

func (f *memFS) Exists(path string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, file := f.data[path]
	_, dir := f.dirs[path]
	return file || dir, nil
}

func (f *memFS) RemoveAll(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k := range f.data {
		if strings.HasPrefix(k, path) {
			delete(f.data, k)
		}
	}
	for k := range f.dirs {
		if strings.HasPrefix(k, path) {
			delete(f.dirs, k)
		}
	}
	return nil
}

// ── syncBuf: buffered stdout for fakeHandle ────────────────────────────────
//
// io.Pipe is synchronous — WriteOutput blocks until pipeOutput reads, which
// creates a scheduler race: Exit() can arrive before writeLine() is called.
// syncBuf decouples the writer and reader: Write returns immediately once the
// bytes are in the internal channel, so WriteOutput never races with the
// log pipeline. Uses a channel of byte slices to avoid sync.Cond lost-wakeup.

type syncBuf struct {
	ch     chan []byte
	done   chan struct{}
	once   sync.Once
}

func newSyncBuf() *syncBuf {
	return &syncBuf{
		ch:   make(chan []byte, 4096),
		done: make(chan struct{}),
	}
}

func (sb *syncBuf) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	select {
	case sb.ch <- cp:
		return len(p), nil
	case <-sb.done:
		return 0, io.ErrClosedPipe
	}
}

func (sb *syncBuf) Read(p []byte) (int, error) {
	select {
	case chunk, ok := <-sb.ch:
		if !ok {
			return 0, io.EOF
		}
		n := copy(p, chunk)
		if n < len(chunk) {
			// put back the remainder
			remainder := make([]byte, len(chunk)-n)
			copy(remainder, chunk[n:])
			select {
			case sb.ch <- remainder:
			default:
			}
		}
		return n, nil
	case <-sb.done:
		// drain any remaining data first
		select {
		case chunk := <-sb.ch:
			n := copy(p, chunk)
			return n, nil
		default:
			return 0, io.EOF
		}
	}
}

func (sb *syncBuf) Close() error {
	sb.once.Do(func() { close(sb.done) })
	return nil
}

// ── ProcessHandle stub ────────────────────────────────────────────────────────

type fakeHandle struct {
	stdinR   *io.PipeReader
	stdinW   *io.PipeWriter
	stdout   *syncBuf
	waitCh   chan error
	pid      int
	exitOnce sync.Once
}

func newFakeHandle() *fakeHandle {
	sr, sw := io.Pipe()
	return &fakeHandle{
		stdinR: sr,
		stdinW: sw,
		stdout: newSyncBuf(),
		waitCh: make(chan error, 1),
		pid:    99,
	}
}

func (h *fakeHandle) Stdin() io.WriteCloser { return h.stdinW }
func (h *fakeHandle) Stdout() io.ReadCloser { return h.stdout }
func (h *fakeHandle) Wait() error           { return <-h.waitCh }
func (h *fakeHandle) Kill() error {
	h.exitOnce.Do(func() {
		h.stdout.Close()
		select {
		case h.waitCh <- errors.New("killed"):
		default:
		}
	})
	return nil
}
func (h *fakeHandle) Pid() int { return h.pid }

// Exit simulates the JVM process exiting normally. Idempotent.
func (h *fakeHandle) Exit() {
	h.exitOnce.Do(func() {
		h.stdout.Close()
		h.waitCh <- nil
	})
}

// WriteOutput simulates the server printing a line to stdout.
// Returns immediately — does not block on the log pipeline consuming the data.
func (h *fakeHandle) WriteOutput(line string) { fmt.Fprintln(h.stdout, line) }

// ReadStdin reads the next chunk written to the server's stdin.
func (h *fakeHandle) ReadStdin(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 512)
	n, err := h.stdinR.Read(buf)
	require.NoError(t, err)
	return string(buf[:n])
}

// ── ProcessSpawner stub ───────────────────────────────────────────────────────

type fakeSpawner struct {
	handle *fakeHandle
	err    error
	mu     sync.Mutex
	called []spawnCall
}

type spawnCall struct{ javaPath string; args []string; dir string }

func (s *fakeSpawner) Spawn(javaPath string, args []string, dir string) (process.ProcessHandle, error) {
	s.mu.Lock()
	s.called = append(s.called, spawnCall{javaPath, args, dir})
	s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	return s.handle, nil
}

// ── JARProvider stub ──────────────────────────────────────────────────────────

type noopJARs struct{}

func (noopJARs) Ensure(_ context.Context, _ runtime.InstanceSpec, _ string) error { return nil }

// ── ProcStats stub ────────────────────────────────────────────────────────────

type fakeStats struct{ cpu float64; mem int }

func (s *fakeStats) Read(_ int) (float64, int, error) { return s.cpu, s.mem, nil }

// ── Test fixtures ─────────────────────────────────────────────────────────────

const dataDir = "/instances"

func testConfig() process.Config {
	return process.Config{DataDir: dataDir, JavaPath: "java"}
}

func testSpec(name string) runtime.InstanceSpec {
	return runtime.InstanceSpec{
		Name:        name,
		Type:        runtime.ServerTypePaper,
		Version:     "1.21.4",
		Port:        25565,
		MaxMemoryMB: 2048,
		MinMemoryMB: 512,
	}
}

func newBackend(t *testing.T, mfs *memFS, spawner *fakeSpawner) *process.ProcessBackend {
	t.Helper()
	return process.NewProcessBackend(testConfig(), mfs, spawner, noopJARs{}, &fakeStats{cpu: 1.5, mem: 512}, process.DefaultLogStreamFactory{})
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCreate_CreatesDirectoryStructure(t *testing.T) {
	mfs := newMemFS()
	b := newBackend(t, mfs, nil)

	err := b.Create(context.Background(), testSpec("srv"))

	require.NoError(t, err)
	ok, _ := mfs.Exists(dataDir + "/srv/server")
	assert.True(t, ok, "server/ dir should be created")
	ok, _ = mfs.Exists(dataDir + "/srv/backups")
	assert.True(t, ok, "backups/ dir should be created")
}

func TestCreate_WritesSpecAndProperties(t *testing.T) {
	mfs := newMemFS()
	b := newBackend(t, mfs, nil)

	err := b.Create(context.Background(), testSpec("srv"))

	require.NoError(t, err)
	_, err = mfs.ReadFile(dataDir + "/srv/spec.json")
	assert.NoError(t, err, "spec.json should be written")
	props, err := mfs.ReadFile(dataDir + "/srv/server/server.properties")
	require.NoError(t, err)
	assert.Contains(t, string(props), "server-port=25565")
}

func TestStart_SpawnsJVM(t *testing.T) {
	mfs := newMemFS()
	handle := newFakeHandle()
	spawner := &fakeSpawner{handle: handle}
	b := newBackend(t, mfs, spawner)

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Start(context.Background(), "srv"))
	t.Cleanup(func() { handle.Exit() })

	spawner.mu.Lock()
	defer spawner.mu.Unlock()
	require.Len(t, spawner.called, 1)
	assert.Equal(t, "java", spawner.called[0].javaPath)
	assert.Contains(t, spawner.called[0].args, "-Xmx2048M")
	assert.Contains(t, spawner.called[0].args, "-Xms512M")
	assert.Contains(t, spawner.called[0].args, "--nogui")
}

func TestStart_WritesEula(t *testing.T) {
	mfs := newMemFS()
	handle := newFakeHandle()
	b := newBackend(t, mfs, &fakeSpawner{handle: handle})

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Start(context.Background(), "srv"))
	t.Cleanup(func() { handle.Exit() })

	eula, err := mfs.ReadFile(dataDir + "/srv/server/eula.txt")
	require.NoError(t, err)
	assert.Equal(t, "eula=true\n", string(eula))
}

func TestStart_ErrorWhenAlreadyRunning(t *testing.T) {
	mfs := newMemFS()
	handle := newFakeHandle()
	b := newBackend(t, mfs, &fakeSpawner{handle: handle})

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Start(context.Background(), "srv"))
	t.Cleanup(func() { handle.Exit() })

	err := b.Start(context.Background(), "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestStop_SendsStopCommand(t *testing.T) {
	mfs := newMemFS()
	handle := newFakeHandle()
	b := newBackend(t, mfs, &fakeSpawner{handle: handle})

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Start(context.Background(), "srv"))

	// Stop must not block; simulate server exiting after receiving "stop"
	stopDone := make(chan error, 1)
	go func() { stopDone <- b.Stop(context.Background(), "srv") }()

	cmd := handle.ReadStdin(t)
	assert.Equal(t, "stop\n", cmd)

	handle.Exit() // server acknowledges stop

	require.NoError(t, <-stopDone)
}

func TestStop_ErrorWhenNotRunning(t *testing.T) {
	b := newBackend(t, newMemFS(), &fakeSpawner{})
	err := b.Stop(context.Background(), "ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestExec_WritesCommandToStdin(t *testing.T) {
	mfs := newMemFS()
	handle := newFakeHandle()
	b := newBackend(t, mfs, &fakeSpawner{handle: handle})

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Start(context.Background(), "srv"))
	t.Cleanup(func() { handle.Exit() })

	// io.Pipe is synchronous: read and write must be concurrent.
	stdinCh := make(chan string, 1)
	go func() {
		buf := make([]byte, 512)
		n, _ := handle.stdinR.Read(buf)
		stdinCh <- string(buf[:n])
	}()

	require.NoError(t, b.Exec(context.Background(), "srv", "say hello"))

	select {
	case got := <-stdinCh:
		assert.Equal(t, "say hello\n", got)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for stdin read")
	}
}

func TestExec_ErrorWhenNotRunning(t *testing.T) {
	b := newBackend(t, newMemFS(), &fakeSpawner{})
	err := b.Exec(context.Background(), "ghost", "list")
	require.Error(t, err)
}

func TestLogs_DelegatesToLogStream(t *testing.T) {
	mfs := newMemFS()
	handle := newFakeHandle()
	t.Cleanup(func() { handle.Exit() })

	// Inject mock LogStream that returns a known sentinel reader.
	// Verifies that Logs() calls LogStream.Reader() and returns its result.
	sentinel := io.NopCloser(strings.NewReader("sentinel"))

	mockStream := mocks.NewMockLogStream(t)
	mockStream.EXPECT().Reader(mock.Anything).Return(sentinel)
	mockStream.EXPECT().WriteLine(mock.Anything).Maybe()
	mockStream.EXPECT().Close().Maybe()

	mockFactory := mocks.NewMockLogStreamFactory(t)
	mockFactory.EXPECT().NewLogStream().Return(mockStream)

	b := process.NewProcessBackend(testConfig(), mfs, &fakeSpawner{handle: handle}, noopJARs{}, &fakeStats{}, mockFactory)

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Start(context.Background(), "srv"))

	rc, err := b.Logs(context.Background(), "srv")

	require.NoError(t, err)
	data, _ := io.ReadAll(rc)
	assert.Equal(t, "sentinel", string(data))
}

func TestLogs_ErrorWhenNotRunning(t *testing.T) {
	b := newBackend(t, newMemFS(), nil)
	_, err := b.Logs(context.Background(), "ghost")
	require.Error(t, err)
}

func TestDelete_RemovesDirectory(t *testing.T) {
	mfs := newMemFS()
	b := newBackend(t, mfs, nil)

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Delete(context.Background(), "srv"))

	ok, _ := mfs.Exists(dataDir + "/srv")
	assert.False(t, ok)
}

func TestDelete_ErrorWhenRunning(t *testing.T) {
	mfs := newMemFS()
	handle := newFakeHandle()
	b := newBackend(t, mfs, &fakeSpawner{handle: handle})

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Start(context.Background(), "srv"))
	t.Cleanup(func() { handle.Exit() })

	err := b.Delete(context.Background(), "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stop")
}

func TestStats_RunningInstance(t *testing.T) {
	mfs := newMemFS()
	handle := newFakeHandle()
	b := newBackend(t, mfs, &fakeSpawner{handle: handle})

	require.NoError(t, b.Create(context.Background(), testSpec("srv")))
	require.NoError(t, b.Start(context.Background(), "srv"))
	t.Cleanup(func() { handle.Exit() })

	stats, err := b.Stats(context.Background(), "srv")

	require.NoError(t, err)
	assert.Equal(t, runtime.StatusRunning, stats.Status)
	assert.Equal(t, 1.5, stats.CPUPct)
	assert.Equal(t, 512, stats.MemoryMB)
	assert.Equal(t, 99, stats.PID)
}

func TestStats_StoppedInstance(t *testing.T) {
	mfs := newMemFS()
	b := newBackend(t, mfs, nil)
	require.NoError(t, b.Create(context.Background(), testSpec("srv")))

	stats, err := b.Stats(context.Background(), "srv")

	require.NoError(t, err)
	assert.Equal(t, runtime.StatusStopped, stats.Status)
}

func TestStats_UnknownInstance(t *testing.T) {
	b := newBackend(t, newMemFS(), nil)
	_, err := b.Stats(context.Background(), "ghost")
	require.Error(t, err)
}
