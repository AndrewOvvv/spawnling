package instance

import (
	"context"
	"fmt"

	"github.com/AndrewOvvv/spawnling/internal/runtime"
)

type Config struct {
	Spec runtime.InstanceSpec
}

type Manager struct {
	rt    RuntimeBackend
	store ConfigStore
}

func NewInstanceManager(rt RuntimeBackend, store ConfigStore) *Manager {
	return &Manager{rt: rt, store: store}
}

func (m *Manager) Create(ctx context.Context, spec runtime.InstanceSpec) error {
	exists, err := m.store.List()
	if err != nil {
		return err
	}
	for _, name := range exists {
		if name == spec.Name {
			return fmt.Errorf("instance %q already exists", spec.Name)
		}
	}
	if err := m.rt.Create(ctx, spec); err != nil {
		return err
	}
	return m.store.Save(spec.Name, &Config{Spec: spec})
}

func (m *Manager) Delete(ctx context.Context, name string) error {
	if err := m.rt.Delete(ctx, name); err != nil {
		return err
	}
	return m.store.Delete(name)
}

func (m *Manager) List() ([]string, error) {
	return m.store.List()
}

func (m *Manager) Get(name string) (*Config, error) {
	return m.store.Load(name)
}
