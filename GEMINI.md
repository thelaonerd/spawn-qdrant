# GEMINI.md - spawn-qdrant

This document provides a technical overview and architectural mandates for the `spawn-qdrant` project, serving as the primary reference for AI agents.

## Project Overview
`spawn-qdrant` is a Go-based CLI utility designed to automate the deployment and management of multiple Qdrant vector database instances on Linux using Docker or Podman.

## Architecture & Components

### 1. Command Layer (`cmd/`)
The CLI is built using the **Cobra** library.
- **Root (`root.go`)**: Entry point for all commands. Handles persistent pre-run logic like lock file initialization.
- **Spawn (`spawn.go`)**: 
  - Validates requested instance count against available system RAM.
  - Ensures the `qdrant/qdrant` image is present.
  - Orchestrates sequential container creation with a **30-second delay** between each to mitigate resource spikes.
  - Manages port assignments (REST starting at 6333, gRPC at 6334 by default).
- **Stop (`stop.go`)**: 
  - Allows stopping individual instances by index (e.g., `stop 1`) or all instances (`stop all`).
  - Cleans up the `qdrant_network` if no containers remain.
- **Clean (`clean.go`)**: 
  - Performs a destructive reset: stops all instances, creates a `tar.gz` backup of all storage directories in `~/qdrant_backup/`, and deletes the original `~/.qdrant_storage*` directories.
  - **Note**: Deletion requires `sudo` as Docker volumes are typically owned by root.

### 2. Internal Abstractions (`internal/`)
- **Container (`internal/container/`)**: Runtime-agnostic wrapper that detects and uses `docker` or `podman`.
- **System (`internal/system/`)**: Resource discovery. Reads `/proc/meminfo` to calculate host capacity.
- **Lock (`internal/lock/`)**: Concurrency control using `~/.spawn-qdrant.lock`.
- **Config (`internal/config/`)**: Environment-based configuration (via `.env`) and defaults.

## Core Logic Mandates

### Port & Naming Conventions
- **Naming**: Containers are named `qdrant-01`, `qdrant-02`, etc.
- **Ports**: 
  - Instance $i$ (1-based) uses:
    - `REST_PORT = Base_REST + 2 * (i - 1)`
    - `GRPC_PORT = Base_GRPC + 2 * (i - 1)`
  - Default bases: 6333 (REST), 6334 (gRPC).
- **Storage**: Mapped to `~/.qdrant_storage01`, `~/.qdrant_storage02`, etc.

### Resource Constraints
- **RAM Check**: 
  - Startup Minimum: 256MB per instance.
  - Efficient Operation: 512MB per instance.
  - The `spawn` command will fail if startup limits are exceeded and warn if efficient limits are breached.

### Deployment Details
- **Network**: All containers join the `qdrant_network` bridge network.
- **Restart Policy**: Containers are created with `--restart unless-stopped`.

## Development Guidelines
- **Go Version**: 1.25+.
- **Testing**: The project currently lacks automated tests. **Requirement**: Any new feature or bug fix must be accompanied by appropriate test cases (unit or integration).
- **OS**: Linux exclusive (due to `/proc/meminfo` and `sudo` usage).
- **Tooling**: Prefer `go fmt` and `go mod tidy` for maintaining code quality and dependencies.
