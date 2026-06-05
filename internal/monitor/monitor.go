package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/AndrewOvvv/spawnling/internal/runtime"
)

type MonitorService struct {
	rt       RuntimeStats
	mu       sync.RWMutex
	snapshots map[string]runtime.Stats
}

func NewMonitorService(rt RuntimeStats) *MonitorService {
	return &MonitorService{
		rt:        rt,
		snapshots: make(map[string]runtime.Stats),
	}
}

func (s *MonitorService) Start(ctx context.Context, instances []string) {
	go s.loop(ctx, instances)
}

func (s *MonitorService) Snapshot(name string) (runtime.Stats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.snapshots[name]
	return snap, ok
}

func (s *MonitorService) loop(ctx context.Context, instances []string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, name := range instances {
				stats, err := s.rt.Stats(ctx, name)
				if err != nil {
					continue
				}
				s.mu.Lock()
				s.snapshots[name] = stats
				s.mu.Unlock()
			}
		}
	}
}
