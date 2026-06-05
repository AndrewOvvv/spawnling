package main

import (
	"context"
	"log"

	"github.com/AndrewOvvv/spawnling/internal/daemon"
)

const socketPath = "/tmp/spawnling.sock"

func main() {
	// Manual DI wiring — all concrete dependencies assembled here.
	//
	// cfg    := config.Load()
	// fs     := fs.NewOSFileSystem()
	// store  := store.NewJSONConfigStore(cfg.DataDir)
	// rt     := process.NewProcessBackend(process.Config{...}, fs)
	// mgr    := instance.NewInstanceManager(rt, store)
	// mon    := monitor.NewMonitorService(rt)
	// tun    := tunnel.NewTunnelService(playit.NewAgentRunner(cfg.DataDir))
	// bak    := backup.NewBackupService(backup.NewArchiveStore(cfg.DataDir))
	// d      := daemon.NewDaemon(socketPath, mgr, mon, tun, bak)

	d := daemon.NewDaemon(socketPath)
	if err := d.Run(context.Background()); err != nil {
		log.Fatalf("daemon: %v", err)
	}
}
