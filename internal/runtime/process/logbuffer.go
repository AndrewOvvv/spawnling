package process

import (
	"bytes"
	"context"
	"io"
	"sync"
)

const ringCapacity = 500

// logBuffer is a thread-safe ring buffer that broadcasts new lines to subscribers.
type logBuffer struct {
	mu      sync.Mutex
	ring    []string
	head    int
	count   int
	subs    map[int]chan string
	nextSub int
	closed  bool
}

func newLogBuffer() *logBuffer {
	return &logBuffer{
		ring: make([]string, ringCapacity),
		subs: make(map[int]chan string),
	}
}

func (b *logBuffer) writeLine(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.ring[b.head] = line
	b.head = (b.head + 1) % len(b.ring)
	if b.count < len(b.ring) {
		b.count++
	}
	for _, ch := range b.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

func (b *logBuffer) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for id, ch := range b.subs {
		close(ch)
		delete(b.subs, id)
	}
}

// subscribe registers a subscriber and returns:
//   - history: lines already in the buffer at subscribe time
//   - live: channel that receives new lines until closed (on buffer.close())
//   - unsub: call to unregister before the buffer is closed (no-op after close)
func (b *logBuffer) subscribe() (history []string, live <-chan string, unsub func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count > 0 {
		n := len(b.ring)
		start := (b.head - b.count + n) % n
		history = make([]string, b.count)
		for i := range history {
			history[i] = b.ring[(start+i)%n]
		}
	}

	ch := make(chan string, 256)

	if b.closed {
		close(ch)
		return history, ch, func() {}
	}

	id := b.nextSub
	b.nextSub++
	b.subs[id] = ch

	unsub = func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[id]; ok {
			delete(b.subs, id)
			close(ch)
		}
	}
	return
}

// reader returns an io.ReadCloser that yields history then live lines.
// Closes when ctx is cancelled or the buffer is closed.
func (b *logBuffer) reader(ctx context.Context) io.ReadCloser {
	history, live, unsub := b.subscribe()
	return &logReader{
		ctx:     ctx,
		history: history,
		live:    live,
		unsub:   unsub,
	}
}

// logReader implements io.ReadCloser over a history slice + live channel.
type logReader struct {
	ctx     context.Context
	history []string
	histIdx int
	live    <-chan string
	unsub   func()
	buf     bytes.Buffer
	once    sync.Once
}

func (r *logReader) Read(p []byte) (int, error) {
	for r.buf.Len() == 0 {
		// First drain history without blocking.
		if r.histIdx < len(r.history) {
			r.buf.WriteString(r.history[r.histIdx])
			r.buf.WriteByte('\n')
			r.histIdx++
			break
		}
		// Then block on live channel or ctx cancellation.
		select {
		case <-r.ctx.Done():
			return 0, io.EOF
		case line, ok := <-r.live:
			if !ok {
				return 0, io.EOF
			}
			r.buf.WriteString(line)
			r.buf.WriteByte('\n')
		}
	}
	return r.buf.Read(p)
}

func (r *logReader) Close() error {
	r.once.Do(r.unsub)
	return nil
}
