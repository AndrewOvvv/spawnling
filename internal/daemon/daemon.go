package daemon

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type Daemon struct {
	socketPath string
}

func NewDaemon(socketPath string) *Daemon {
	return &Daemon{socketPath: socketPath}
}

// Run starts the daemon and blocks until SIGINT or SIGTERM.
func (d *Daemon) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// TODO: start gRPC server on d.socketPath

	<-ctx.Done()
	return d.shutdown()
}

func (d *Daemon) shutdown() error {
	// TODO: gracefully stop all running server instances
	return nil
}
