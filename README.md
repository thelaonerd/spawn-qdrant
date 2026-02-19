# spawn-qdrant

A CLI utility written in Go to easily spawn, manage, and clean up multiple Qdrant instances using Docker (or Podman) on Linux.

## Features

- **Multi-Instance Management**: Spawn $N$ isolated Qdrant instances with auto-incrementing ports.
- **Smart Resource Checks**: Automatically estimates how many instances your system can handle based on available RAM.
- **Auto-Network**: Connects all instances to a dedicated `qdrant_network`.
- **Safe Cleanup**: `clean` command stops instances, backs up data to `~/qdrant_backup`, and securely removes storage directories.
- **Runtime Agnostic**: Automatically detects and uses `docker` or `podman`.

## Prerequisites

- **OS**: Linux
- **Runtime**: Docker or Podman installed and running.
- **Language**: Go 1.25+ (for building from source).

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/thelaonerd/spawn-qdrant.git
   cd spawn-qdrant
   ```

2. Build the binary:
   ```bash
   go build -o spawn-qdrant main.go
   ```

3. (Optional) Move to your PATH:
   ```bash
   sudo mv spawn-qdrant /usr/local/bin/
   ```

## Configuration

The application uses defaults but can be configured via a `.env` file in the working directory:

```env
# Starting ports for the first instance
REST_PORT=6333
GRPC_PORT=6334
```

*Subsequent instances will increment ports by 2 (e.g., Instance 2: 6335/6336).*

## Usage

### 1. Spawn Instances (`spawn`)

**Estimate Capacity:**
Run without arguments to see how many instances your system can support.
```bash
spawn-qdrant spawn
```
*Output example:*
```
Available RAM: 12236 MB
Max instances (startup only, 256MB/each): 47
Max efficient instances (vector ops, 512MB/each): 23
```

**Spawn N Instances:**
```bash
spawn-qdrant spawn 2
```
*This starts `qdrant-01` and `qdrant-02`. Data is stored in `~/.qdrant_storage01` and `~/.qdrant_storage02`.*

> **Note**: The command checks if you have enough RAM. It will error if you exceed startup limits and warn if you exceed efficient limits.

### 2. Stop Instances (`stop`)

**Stop a specific instance:**
```bash
spawn-qdrant stop 2
# Stops and removes qdrant-02
```

**Stop all instances:**
```bash
spawn-qdrant stop all
# Stops and removes all qdrant-* containers and the network
```

### 3. Clean Up (`clean`)

This is a destructive command that helps reset your environment while keeping a backup.

```bash
spawn-qdrant clean
```

**What it does:**
1. Stops all running Qdrant instances.
2. Creates a `tar.gz` backup of all `~/.qdrant_storage*` directories into `~/qdrant_backup/`.
3. Deletes the original `~/.qdrant_storage*` directories.

> **Important**: This command uses `sudo` to delete the storage directories because Docker-created volumes are often owned by root. You may be prompted for your password.

### 4. Lock File Mechanism

The application uses a lock file at `~/.spawn-qdrant.lock` to prevent multiple concurrent spawn sessions which could conflict on ports or resources.

- **Lock Creation**: Created automatically when you run `spawn`.
- **Lock Removal**: Automatically removed when you run `stop all`, `clean`, or `stop` the last remaining instance.
- **Manual Removal**: If the application crashes and the lock file persists, you may need to manually remove it:
  ```bash
  rm ~/.spawn-qdrant.lock
  ```

### 5. Restart Policy

All spawned containers are created with `--restart unless-stopped`. This means they will automatically restart if the host machine reboots or if the Docker/Podman service restarts, unless they were explicitly stopped (e.g., via `spawn-qdrant stop`).

## Troubleshooting

- **"Insufficient RAM"**: The tool prevents spawning if `n * 256MB > Available RAM`. Try spawning fewer instances.
- **Permission Denied during Clean**: Ensure you have `sudo` privileges, as `clean` requires them to remove root-owned storage folders.
- **"Lock file exists"**: Another instance might be running. If not, remove `~/.spawn-qdrant.lock` manually.
- **Docker/Podman not found**: Ensure one of them is installed and in your system PATH.
