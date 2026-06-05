# Spawnling

> Host, monitor, and manage Minecraft servers from your local machine — including behind NAT.

Spawnling is a CLI daemon written in Go. It handles the full lifecycle of Minecraft server
instances and solves the home-user connectivity problem: friends can join your locally hosted
server without you touching your router, the same way party-hosted multiplayer works in
modern games.

---

## Why Spawnling?

Running a Minecraft server at home is easy. Letting your friends connect is not.

| Scenario | Without Spawnling | With Spawnling |
|---|---|---|
| Static IP / VPS | Port-forward manually, share IP | Automatic, one command |
| Home machine, no static IP | UPnP hacks, DynDNS, ngrok | Built-in tunnel, stable address |
| Multiple servers | Juggle JARs, scripts, ports | Named instances, single tool |
| Monitoring | Log tailing, guesswork | Live status, player list, resource usage |
| Backups | Cron scripts | Scheduled, retention policy, one-command restore |

---

## Key Features

- **Multi-instance management** — run Paper, Fabric, Vanilla, Forge side by side
- **Tunnel mode** — stable connection address for home users without port forwarding
- **Live monitoring** — status, player list, CPU/RAM, TPS
- **World backups** — scheduled backups with configurable retention
- **Console access** — attach to a live server console or send one-off commands
- **Daemon + CLI** — lightweight background process, CLI talks to it; future UI-ready via local API

---

## Quick Start

```bash
# Install
go install github.com/spawnling/spawnling@latest

# Create and start a Paper 1.21 server
spawnling create my-server --type paper --version 1.21
spawnling start my-server

# Enable tunnel for friends to join from anywhere
spawnling tunnel start my-server
# → Connect at: mc.playit.gg/xxxxx

# Watch live status
spawnling status my-server
```

---

## Documentation

- [Requirements](requirements.md) — functional and non-functional requirements
- [Release Policy](release.md) — versioning, release cycle, changelog format
- [Branching Strategy](branching.md) — branch model and workflow
