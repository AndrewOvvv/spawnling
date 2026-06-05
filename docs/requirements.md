# Requirements

## Context

Spawnling is a CLI application (with future UI) for hosting, monitoring, and managing
Minecraft server instances on a local machine. The key differentiator is built-in
connectivity support for home users without a static IP or port forwarding — allowing
friends to join a locally hosted server the same way modern games support party-hosted
multiplayer sessions.

---

## Functional Requirements

### FR-1 · Server Lifecycle Management

| ID | Requirement |
|---|---|
| FR-1.1 | Download and install server software: Vanilla, Paper, Fabric, Forge, Quilt |
| FR-1.2 | Create named server instances with configurable parameters (version, type, port) |
| FR-1.3 | Start, stop, restart a server instance |
| FR-1.4 | Run multiple server instances simultaneously on the same machine |
| FR-1.5 | Delete a server instance (requires explicit confirmation) |
| FR-1.6 | List all instances with their current status |

### FR-2 · Server Configuration

| ID | Requirement |
|---|---|
| FR-2.1 | Read and write `server.properties` through the app |
| FR-2.2 | Configure JVM arguments: heap size (`-Xmx`, `-Xms`), GC flags |
| FR-2.3 | Select the Java executable to use per instance (support multiple JDK installs) |
| FR-2.4 | Accept and store the EULA agreement per instance |

### FR-3 · World Management

| ID | Requirement |
|---|---|
| FR-3.1 | List worlds associated with a server instance |
| FR-3.2 | Trigger a manual backup of a world (compressed archive) |
| FR-3.3 | Configure and run scheduled backups (cron-style interval) |
| FR-3.4 | Restore a world from a backup |
| FR-3.5 | Delete old backups automatically based on retention policy (keep last N) |

### FR-4 · Monitoring

| ID | Requirement |
|---|---|
| FR-4.1 | Report server status: running / stopped / crashed / starting |
| FR-4.2 | Report online player count and player list |
| FR-4.3 | Report resource usage: CPU %, RAM (used / allocated) |
| FR-4.4 | Stream server log output in real time |
| FR-4.5 | Detect crashes and optionally auto-restart the instance |
| FR-4.6 | Record uptime and restart history per instance |

### FR-5 · Console Access

| ID | Requirement |
|---|---|
| FR-5.1 | Send arbitrary commands to a running server's stdin |
| FR-5.2 | View the last N lines of server output (tail mode) |
| FR-5.3 | Attach to a live interactive console session for a running instance |

### FR-6 · Connectivity

This is split into two modes based on the user's network setup.

#### FR-6.1 · Direct mode (static IP / port forwarding)

| ID | Requirement |
|---|---|
| FR-6.1.1 | Detect the machine's public IP and display the connection address |
| FR-6.1.2 | Validate that the configured port is reachable from outside (basic probe) |
| FR-6.1.3 | Display a ready-to-share connection string: `ip:port` |

#### FR-6.2 · Tunnel mode (home users, no port forwarding)

The goal: allow friends to connect without the host configuring their router,
similar to party-hosted multiplayer in modern games.

Implementation approach: integrate with an open relay/tunnel service as the transport
layer. Candidates: **playit.gg** (Minecraft-specific, has an open agent protocol),
**frp** (self-hosted reverse proxy), **Cloudflare Tunnel** (free tier).
The app wraps the tunnel agent — the user does not install or configure it separately.

| ID | Requirement |
|---|---|
| FR-6.2.1 | Start a tunnel for a given server instance with a single command |
| FR-6.2.2 | Display a stable connection address once the tunnel is active |
| FR-6.2.3 | Stop the tunnel when the server stops or on explicit command |
| FR-6.2.4 | Show tunnel status alongside server status in the instance overview |
| FR-6.2.5 | Support at minimum one tunnel backend out of the box (e.g. playit.gg) |
| FR-6.2.6 | Allow configuring a custom frp server as an alternative backend |

### FR-7 · CLI Interface

| ID | Requirement |
|---|---|
| FR-7.1 | All operations above are available as CLI subcommands |
| FR-7.2 | Machine-readable output mode (`--json`) for scripting and future UI integration |
| FR-7.3 | A local daemon/agent process manages instances; CLI communicates with it |
| FR-7.4 | The daemon exposes a local REST or gRPC API (foundation for future web UI) |

---

## Non-functional Requirements

### NFR-1 · Performance

| ID | Requirement |
|---|---|
| NFR-1.1 | The Spawnling daemon idle overhead: < 1% CPU, < 64 MB RAM |
| NFR-1.2 | Log streaming latency: < 1 second from server output to CLI display |
| NFR-1.3 | CLI command response time (status, list): < 500 ms |

### NFR-2 · Portability

| ID | Requirement |
|---|---|
| NFR-2.1 | Primary target: Linux (amd64, arm64). Secondary: macOS, Windows |
| NFR-2.2 | Distributed as a single binary with no external runtime dependencies |
| NFR-2.3 | No dependency on Docker or a container runtime |

### NFR-3 · Security

| ID | Requirement |
|---|---|
| NFR-3.1 | The daemon API listens on localhost only by default |
| NFR-3.2 | Tunnel connections use the backend's encrypted transport (TLS) |
| NFR-3.3 | No credentials or tokens stored in plain text |
| NFR-3.4 | Server process runs as the current user, not root |

### NFR-4 · Reliability

| ID | Requirement |
|---|---|
| NFR-4.1 | Daemon survives individual server crashes; does not exit with the server process |
| NFR-4.2 | Graceful shutdown sends a `stop` command to the server before killing the process |
| NFR-4.3 | State (instance config, status) persists across daemon restarts |

### NFR-5 · Usability

| ID | Requirement |
|---|---|
| NFR-5.1 | A new server instance can be created and started in ≤ 3 commands |
| NFR-5.2 | Error messages are human-readable and suggest a corrective action |
| NFR-5.3 | Long-running operations (download, backup) show a progress indicator |
| NFR-5.4 | The app does not require the user to read documentation for basic operations |

---

## Out of Scope (for initial versions)

- Web UI / graphical dashboard (planned, not in scope for v1)
- Plugin/mod marketplace integration (Modrinth, CurseForge)
- Multi-machine / remote server management
- User account system or multi-user access control
- Bedrock Edition support
