# spawn-qdrant - Developer Handover Documentation

> **Project**: CLI tool for spawning and managing multiple Qdrant instances using Docker/Podman  
> **Target Audience**: Go developers with 1-2 years experience, familiar with Docker/containerization  
> **Technology Stack**: Go 1.25+, Cobra CLI, Docker/Podman  
> **Architecture**: Layered architecture with clear separation of concerns

---

## Table of Contents

1. [Quick Start for New Developers](#1-quick-start-for-new-developers)
2. [Project Overview](#2-project-overview)
3. [Architecture Overview](#3-architecture-overview)
4. [Data Flow & Component Interaction](#4-data-flow--component-interaction)
5. [End-to-End Workflow Documentation](#5-end-to-end-workflow-documentation)
6. [Directory Structure](#6-directory-structure)
7. [Key Patterns & Conventions](#7-key-patterns--conventions)
8. [Security Considerations](#8-security-considerations)
9. [Testing Guide](#9-testing-guide)
10. [Common Tasks](#10-common-tasks)
11. [Troubleshooting](#11-troubleshooting)

---

## 1. Quick Start for New Developers

### 1.1 Clone and Build

```bash
# Clone the repository
git clone <repository-url>
cd spawn-qdrant

# Download dependencies
go mod download

# Build the binary
go build -o spawn-qdrant main.go

# Run tests
go test ./...
```

### 1.2 First Commands to Try

```bash
# Check available RAM and estimated instances
./spawn-qdrant check

# Spawn 2 instances (dry run - just see what would happen)
./spawn-qdrant spawn 2

# View help
./spawn-qdrant --help
./spawn-qdrant spawn --help
```

### 1.3 Development Workflow

1. **Make changes** to relevant files in `cmd/` or `internal/`
2. **Run tests**: `go test ./...`
3. **Build**: `go build -o spawn-qdrant main.go`
4. **Test manually**: `./spawn-qdrant check`
5. **Commit**: Follow conventional commit messages

---

## 2. Project Overview

**spawn-qdrant** is a CLI utility that simplifies running multiple isolated Qdrant (vector database) instances on Linux using Docker or Podman. Qdrant is a vector similarity search engine - this tool makes it easy to run multiple isolated instances for development, testing, or multi-tenant scenarios.

### Key Features

| Feature | Description |
|---------|-------------|
| **Multi-Instance Spawn** | Create N isolated Qdrant instances with auto-incrementing ports |
| **Runtime Detection** | Automatically uses Docker, falls back to Podman |
| **Resource Safety** | Pre-flight RAM estimation prevents OOM conditions |
| **Safe Cleanup** | Backup before delete, with interactive confirmation |
| **Concurrency Control** | File-based locking prevents operation conflicts |
| **Signal Handling** | Graceful shutdown on SIGINT/SIGTERM |

### Prerequisites

- **OS**: Linux (tested on Ubuntu/Debian)
- **Runtime**: Docker (preferred) or Podman installed and in PATH
- **Privileges**: Passwordless sudo for `clean` command (backs up/deletes root-owned files)
- **Go**: Version 1.25+ (for development only)

### Container Architecture

Each spawned instance gets:
- **Container name**: `qdrant-01`, `qdrant-02`, etc.
- **REST API port**: 6333, 6335, 6337... (auto-increment by 2)
- **gRPC port**: 6334, 6336, 6338... (auto-increment by 2)
- **Storage directory**: `~/.qdrant_storage01`, `~/.qdrant_storage02`, etc.
- **Network**: All instances attach to `qdrant_network` (Docker bridge)
- **Restart policy**: `unless-stopped`

---

## 3. Architecture Overview

### 3.1 High-Level Architecture

The project follows a **Layered Architecture** pattern with three distinct layers:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         LAYER 1: PRESENTATION                        │
│                              cmd/ package                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │  check   │ │  spawn   │ │   stop   │ │  clean   │ │ version  │  │
│  │   cmd    │ │   cmd    │ │   cmd    │ │   cmd    │ │   cmd    │  │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘  │
└───────┼────────────┼────────────┼────────────┼────────────┼────────┘
        │            │            │            │            │
        └────────────┴────────────┴────────────┴────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       LAYER 2: BUSINESS LOGIC                          │
│                           internal/ packages                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │   lock/      │  │   system/    │  │  container/  │  │  config/  │ │
│  │ File locking │  │  RAM checks  │  │Docker/Podman │  │  Loading  │ │
│  │   Create()   │  │GetAvailable()│  │   Run()      │  │   Init    │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └─────┬─────┘ │
└─────────┼─────────────────┼─────────────────┼────────────────┼───────┘
          │                 │                 │                │
          └─────────────────┴─────────────────┴────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      LAYER 3: INFRASTRUCTURE                         │
│                      External Systems / OS                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │    Docker    │  │   Podman     │  │ Linux Kernel │  │  Files   │ │
│  │   Engine     │  │   (fallback) │  │ /proc, sysfs │  │  ~/.qdrant* │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 Layer Responsibilities

| Layer | Package | Responsibility | Key Files |
|-------|---------|----------------|-----------|
| **Presentation** | `cmd/` | CLI commands, flag parsing, user interaction, cobra setup | `root.go`, `spawn.go`, `stop.go`, `clean.go`, `check.go` |
| **Business Logic** | `internal/lock/` | Concurrency control via file locking | `lockfile.go` |
| **Business Logic** | `internal/system/` | Resource checking, RAM estimation | `resources.go` |
| **Business Logic** | `internal/container/` | Container lifecycle operations | `runtime.go` |
| **Infrastructure** | Docker/Podman | Container runtime execution | External binary |
| **Infrastructure** | Linux OS | Memory info, signals, filesystem | `/proc/meminfo` |

### 3.3 Component Interaction Model

The application uses **dependency-free** architecture - business logic packages don't import each other directly. Instead, they receive dependencies through function parameters:

```go
// cmd/spawn.go - Orchestrator pattern
func spawnWorkflow() error {
    // 1. Lock acquisition
    if err := lock.Create(); err != nil {
        return err
    }
    defer lock.Remove() // Cleanup pattern

    // 2. Resource check
    ramMB, _ := system.GetAvailableRAM()
    
    // 3. Container operations
    container.EnsureImage("qdrant/qdrant")
    container.CreateNetwork("qdrant_network")
    
    // 4. Instance creation loop
    for i := 0; i < count; i++ {
        container.RunQdrant(config)
    }
}
```

**Key Principle**: Commands in `cmd/` orchestrate calls to `internal/` packages. Packages in `internal/` are independent and focused on single responsibilities.

---

## 4. Data Flow & Component Interaction

### 4.1 Spawn Command Data Flow

This diagram shows how data flows when user runs `spawn-qdrant spawn 2`:

```
┌────────┐                                                    
│  User  │ CLI Input: "spawn 2"                               
└───┬────┘                                                    
    │                                                         
    ▼                                                         
┌──────────────────────────────────────────────────────────┐
│  LAYER 1: PRESENTATION (cmd/)                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  root.go                                            │  │
│  │  - Parse flags: --rest-port, --grpc-port           │  │
│  │  - Bind to viper config                             │  │
│  │  - Route to spawn subcommand                        │  │
│  └─────────────────────────────────────────────────────┘  │
│                          │                                │
│                          ▼                                │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  spawn.go                                           │  │
│  │  - Validate instance_count                          │  │
│  │  - Setup signal handling (SIGINT/SIGTERM)             │  │
│  │  - Call internal packages                           │  │
│  └─────────────────────────────────────────────────────┘  │
└──────────────────────────┬───────────────────────────────┘
                           │ Function calls
                           ▼
┌──────────────────────────────────────────────────────────┐
│  LAYER 2: BUSINESS LOGIC (internal/)                    │
│                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌──────────────┐ │
│  │  lock/      │    │  system/    │    │  container/  │ │
│  │  Create()   │    │GetAvailable │    │  EnsureImage │ │
│  │  ├─▶ File   │    │  RAM()      │    │  ├─▶ docker   │ │
│  │  │  ~/.spawn │    │  ├─▶ /proc  │    │  │   inspect  │ │
│  │  │  -qdrant  │    │  │  /meminfo │    │  │            │ │
│  │  │  .lock    │    │  │           │    │  └─▶ docker   │ │
│  │  │           │    │  └─▶ Return │    │     pull      │ │
│  │  └─▶ bool    │    │     uint64   │    │               │ │
│  │     (success)│    │              │    │  CreateNetwork│ │
│  │              │    │  Estimate()  │    │  ├─▶ docker   │ │
│  │  Remove()    │    │  ├─▶ Calc    │    │  │   network   │ │
│  │  ├─▶ os.     │    │  │  startup/  │    │  │   create    │ │
│  │  │  Remove() │    │  │  efficient │    │  │             │ │
│  │  │           │    │  │           │    │  └─▶ Return   │ │
│  │  └─▶ error   │    │  └─▶ Return   │    │     string    │ │
│  │     (nil ok) │    │     (max S, E)│    │               │ │
│  └─────────────┘    └─────────────┘    │  RunQdrant()  │ │
│                                          │  ├─▶ docker   │ │
│                                          │  │   run       │ │
│                                          │  │   [config]  │ │
│                                          │  │             │ │
│                                          │  └─▶ Return   │ │
│                                          │     error     │ │
│                                          └──────────────┘ │
└──────────────────────────┬───────────────────────────────┘
                           │ System calls / exec.Command
                           ▼
┌──────────────────────────────────────────────────────────┐
│  LAYER 3: INFRASTRUCTURE                                │
│                                                          │
│  ┌──────────┐  ┌────────────┐  ┌─────────────────────┐  │
│  │  Docker  │  │   Podman   │  │    Linux OS         │  │
│  │  Binary  │  │  (fallback)│  │                     │  │
│  │          │  │            │  │  ┌───────────────┐  │  │
│  │ Commands:│  │  Commands: │  │  │/proc/meminfo  │  │  │
│  │  - run   │  │   - run    │  │  │  (RAM data)   │  │  │
│  │  - stop  │  │   - stop   │  │  └───────────────┘  │  │
│  │  - rm    │  │   - rm     │  │                     │  │
│  │  - pull  │  │   - pull   │  │  ┌───────────────┐  │  │
│  │  - ps    │  │   - ps     │  │  │ Signal:       │  │  │
│  │  - network│  │   - network│  │  │ SIGINT/TERM   │  │  │
│  │    create│  │     create │  │  │               │  │  │
│  │          │  │            │  │  │ Context:      │  │  │
│  │ Output:  │  │  Output:   │  │  │ ctx.Done()    │  │  │
│  │ Containers│  │  Containers│  │  └───────────────┘  │  │
│  │ Networks │  │  Networks  │  │                     │  │
│  └──────────┘  └────────────┘  └─────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### 4.2 State Management

The application maintains minimal state:

| State Type | Location | Purpose |
|------------|----------|---------|
| **Lock File** | `~/.spawn-qdrant.lock` | Prevents concurrent spawn operations |
| **Config File** | `~/.spawn-qdrant.yaml` | User preferences (rarely used) |
| **Storage Dirs** | `~/.qdrant_storageNN` | Container data volumes |
| **Backup Dir** | `~/qdrant_backup/` | Archived storage from `clean` |
| **Docker State** | Docker daemon | Container and network lifecycle |

**State Flow**:
1. **Lock acquired** → Operations begin
2. **Containers created** → Docker daemon manages state
3. **Lock released** → On success or error (via defer)
4. **Cleanup** → `stop` or `clean` commands remove containers and lock

---

## 5. End-to-End Workflow Documentation

### 5.1 Workflow 1: Spawn Instances (`spawn-qdrant spawn 3`)

**Business Purpose**: Creates N Qdrant container instances with isolated storage and incremental ports.

**Entry Point**: `cmd/spawn.go`

**Step-by-Step Execution Flow**:

#### Phase 1: Initialization

**Lines 33-44: Signal Handling Setup**
```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```
> Creates a context that listens for SIGINT/SIGTERM for graceful shutdown. If user presses Ctrl+C during spawn, context cancellation triggers cleanup.

**Lines 46-58: Argument Parsing**
```go
n := 1 // Default
if len(args) > 0 {
    parsed, err := strconv.Atoi(args[0])
    if err != nil || parsed < 1 {
        return fmt.Errorf("instance_count must be a positive integer")
    }
    n = parsed
}
```
> Validates input: must be positive integer. Provides helpful error message on invalid input.

**Lines 61-70: Lock Acquisition**
```go
if err := lock.Create(); err != nil {
    return fmt.Errorf("failed to acquire lock: %w", err)
}
defer func() {
    if cleanupLock {
        lock.Remove()
    }
}()
```
> **Critical**: Prevents concurrent spawn operations. Creates file at `~/.spawn-qdrant.lock` with O_EXCL flag (fails if exists). Cleanup deferred to ensure lock always removed, even on panic.

#### Phase 2: Resource Validation

**Lines 72-85: RAM Check**
```go
ramMB, err := system.GetAvailableRAM()
if err != nil {
    return fmt.Errorf("failed to get available RAM: %w", err)
}

maxStartup, maxEfficient := system.EstimateInstances(ramMB)
if uint64(n) > maxEfficient {
    return fmt.Errorf("insufficient RAM for %d instances (max efficient: %d, max startup: %d)", 
        n, maxEfficient, maxStartup)
}
```
> Reads `/proc/meminfo` for `MemAvailable`, calculates capacity (256MB/startup, 512MB/efficient). Prevents OOM conditions.

#### Phase 3: Container Preparation

**Lines 95-101: Image Check**
```go
if err := container.EnsureImage("qdrant/qdrant"); err != nil {
    lock.Remove() // Manual cleanup before return
    return fmt.Errorf("failed to ensure qdrant image: %w", err)
}
```
> Checks if image exists locally. If not, runs `docker pull qdrant/qdrant`. Lock manually removed on error (before defer executes).

**Line 103: Network Creation**
```go
_ = container.CreateNetwork("qdrant_network")
```
> Idempotent network creation. Safe to call multiple times (Docker handles duplicates).

#### Phase 4: Instance Creation Loop

**Lines 115-170: Main Loop**
```go
for i := 0; i < n; i++ {
    // Port calculation
    restPort := startRest + (2 * i)  // 6333, 6335, 6337...
    grpcPort := startGrpc + (2 * i)  // 6334, 6336, 6338...
    suffix := fmt.Sprintf("%02d", i+1)  // "01", "02", "03"...
    
    // Context cancellation check
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Create container
    containerName := fmt.Sprintf("qdrant-%s", suffix)
    storageDir := filepath.Join(homeDir, fmt.Sprintf(".qdrant_storage%s", suffix))
    
    err := container.RunQdrant(container.QdrantConfig{
        Name:       containerName,
        Network:    "qdrant_network",
        RestPort:   restPort,
        GrpcPort:   grpcPort,
        StorageDir: storageDir,
    })
    
    // Inter-instance delay (except last)
    if i < n-1 {
        select {
        case <-time.After(30 * time.Second):
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```
> **Key behaviors**:
- Ports auto-increment by 2 (avoids port conflicts)
- Names zero-padded (qdrant-01, not qdrant-1)
- 30-second delay between instances (allows Qdrant to initialize)
- Respects context cancellation at each checkpoint

#### Container Runtime Layer

**Location**: `internal/container/runtime.go`

**Runtime Detection** (Auto-detects Docker or Podman):
```go
func InitRuntime() error {
    if isCommandAvailable("docker") {
        containerRuntime = Docker
        return nil
    }
    if isCommandAvailable("podman") {
        containerRuntime = Podman
        return nil
    }
    return fmt.Errorf("neither docker nor podman is installed")
}
```

**Container Creation**:
```go
func RunQdrant(cfg QdrantConfig) error {
    // Security: Prevent path injection
    if strings.Contains(cfg.StorageDir, ":") {
        return fmt.Errorf("invalid storage directory: path cannot contain ':'")
    }

    return runCommand("run", "-d",
        "--name", cfg.Name,
        "--net", cfg.Network,
        "--restart", "unless-stopped",
        "-p", fmt.Sprintf("%d:6333", cfg.RestPort),
        "-p", fmt.Sprintf("%d:6334", cfg.GrpcPort),
        "-v", fmt.Sprintf("%s:/qdrant/storage", cfg.StorageDir),
        "qdrant/qdrant",
    )
}
```

**Command Execution Pattern**:
```go
func runCommand(args ...string) error {
    cmd := exec.Command(string(containerRuntime), args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

### 5.2 Workflow 2: Stop Instances (`spawn-qdrant stop all` or `spawn-qdrant stop 2`)

**Business Purpose**: Gracefully stops and removes Qdrant containers and network.

**Entry Point**: `cmd/stop.go`

**Stop All Flow** (`stopAll` function):
```go
func stopAll(cmd *cobra.Command) error {
    // 1. List containers with "qdrant-" prefix
    targets, err := container.ListContainerNames("qdrant-")
    
    // 2. Stop and remove each container
    for _, name := range targets {
        container.StopAndRemoveContainer(name)
    }
    
    // 3. Remove network
    _ = container.RemoveNetwork("qdrant_network")
    
    // 4. Remove lock file
    return lock.Remove()
}
```

**Stop Single Instance** (`stopInstance` function):
```go
func stopInstance(cmd *cobra.Command, n int) error {
    name := fmt.Sprintf("qdrant-%02d", n)
    stopAndRemove(cmd, name)
    
    // Check if any instances remain
    anyRemaining, _ := container.HasRunningContainers("name=qdrant-")
    if !anyRemaining {
        _ = container.RemoveNetwork("qdrant_network")
        _ = lock.Remove()  // Remove lock if last instance
    }
}
```

### 5.3 Workflow 3: Clean & Backup (`spawn-qdrant clean`)

**Business Purpose**: Destructive cleanup with backup - stops containers, backs up data to tar.gz, deletes storage.

**Entry Point**: `cmd/clean.go`

**Execution Flow**:
```go
func cleanWorkflow(cmd *cobra.Command) error {
    // 1. Interactive confirmation (if TTY and not --force)
    if !viper.GetBool("force") && isatty(os.Stdin) {
        // Prompt user: "Are you sure?"
    }
    
    // 2. Stop all instances
    stopAll(cmd)
    
    // 3. Create backup
    backupFile := filepath.Join(homeDir, "qdrant_backup", 
                 fmt.Sprintf("backup_%s.tar.gz", timestamp))
    
    // 4. Validate and filter storage directories
    validatedMatches := filterStorageDirs(cmd, matches)
    
    // 5. Backup with sudo (files owned by root)
    tarCmd := exec.CommandContext(ctx, "sudo", "tar", "-czf", backupFile, "--", dirs...)
    
    // 6. Delete with sudo
    rmCmd := exec.CommandContext(rmCtx, "sudo", "rm", "-rf", "--", dirs...)
}
```

**Security: Symlink Validation**:
```go
func filterStorageDirs(cmd *cobra.Command, matches []string) []string {
    var validated []string
    for _, match := range matches {
        info, err := os.Lstat(match)
        
        // Reject symlinks (prevents privilege escalation)
        if info.Mode()&os.ModeSymlink != 0 {
            logInfo(cmd, "Warning: %s is a symbolic link, skipping", match)
            continue
        }
        
        // Must be a directory
        if !info.IsDir() {
            continue
        }
        validated = append(validated, match)
    }
    return validated
}
```

---

## 6. Directory Structure

```
spawn-qdrant/
├── main.go                          # Entry point, exit code handling
├── go.mod                           # Go module definition
├── go.sum                           # Dependency checksums
├── README.md                        # User-facing documentation
├── HANDOVER.md                      # This developer handover doc
├── architecture-diagram.drawio      # Architecture visualization
│
├── cmd/                             # Cobra CLI commands (Presentation Layer)
│   ├── root.go                      # Root command, config init, logging helpers
│   │                                # - initConfig(): viper setup
│   │                                # - logInfo(), logWarn(): Consistent logging
│   ├── spawn.go                     # Spawn instances command (complex workflow)
│   ├── stop.go                      # Stop instances command
│   ├── clean.go                     # Clean/backup command (most complex)
│   ├── check.go                     # RAM check command
│   ├── version.go                   # Version command
│   ├── completion.go                # Shell completion generator
│   └── clean_test.go                # Tests for clean command
│
└── internal/                        # Private application code (Business Logic Layer)
    ├── container/
    │   └── runtime.go               # Docker/Podman abstraction
    │                                  # - InitRuntime(): Auto-detect
    │                                  # - RunQdrant(): Create container
    │                                  # - StopAndRemoveContainer(): Cleanup
    │                                  # - EnsureImage(): Pull if needed
    │                                  # - CreateNetwork(), RemoveNetwork()
    ├── lock/
    │   └── lockfile.go              # File-based locking
    │                                  # - Create(): Atomic lock acquisition
    │                                  # - Remove(): Safe lock release
    │                                  # - Exists(): Check status
    ├── system/
    │   ├── resources.go             # RAM checking and estimation
    │   │                              # - GetAvailableRAM(): Parse /proc/meminfo
    │   │                              # - EstimateInstances(): Calculate capacity
    │   └── resources_test.go        # Unit tests for calculations
    └── config/
        └── config.go                # Configuration loading (viper integration)
                                     # - Currently minimal usage
```

---

## 7. Key Patterns & Conventions

### 7.1 Command Pattern (Cobra)

Each command follows this structure:

```go
var commandNameCmd = &cobra.Command{
    Use:   "command-name [args]",
    Short: "Brief description",
    Long:  `Detailed multiline description`,
    Args:  cobra.ExactArgs(1),  // or MaximumNArgs, RangeArgs, etc.
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation returns error for consistent error handling
        return nil
    },
}

func init() {
    rootCmd.AddCommand(commandNameCmd)
    // Add flags here
    commandNameCmd.Flags().BoolVarP(&force, "force", "f", false, "Description")
}
```

### 7.2 Error Handling Pattern

**Always wrap errors with context**:
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

**Cleanup on error** (critical pattern):
```go
if err := riskyOperation(); err != nil {
    lock.Remove()  // Always cleanup resources on failure
    return err
}
```

**Deferred cleanup** (for successful path):
```go
lock.Create()
defer lock.Remove()  // Runs even if panic occurs
```

### 7.3 Context Cancellation Pattern

**Setup**:
```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```

**Checkpoint pattern** (respect cancellation):
```go
for i := 0; i < n; i++ {
    select {
    case <-ctx.Done():
        // User interrupted - cleanup and exit
        if i == 0 {
            lock.Remove()
        }
        return ctx.Err()
    default:
        // Continue operation
    }
    
    // Long operation with cancellation support
    select {
    case <-time.After(30 * time.Second):
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 7.4 Container Runtime Abstraction

The project abstracts Docker/Podman for easy switching:

```go
type Runtime string

const (
    Docker Runtime = "docker"
    Podman Runtime = "podman"
)

var containerRuntime Runtime

// All operations use containerRuntime variable
func runCommand(args ...string) error {
    cmd := exec.Command(string(containerRuntime), args...)
    return cmd.Run()
}
```

**Why this matters**: Easy to extend for other container runtimes (containerd, cri-o, etc.)

### 7.5 Configuration Hierarchy

Configuration precedence (highest to lowest):
1. Command-line flags (`--rest-port`)
2. Environment variables (`SPAWN_QDRANT_REST_PORT`)
3. Config file (`~/.spawn-qdrant.yaml`)
4. Defaults (hardcoded)

```go
// In root.go init()
viper.SetEnvPrefix("SPAWN_QDRANT")
viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
viper.AutomaticEnv()

// Usage in commands
value := viper.GetString("rest-port")  // Reads from any source
```

### 7.6 Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Commands | `[action]Cmd` | `spawnCmd`, `stopCmd` |
| Results/DTOs | PascalCase | `QdrantConfig` |
| Private funcs | camelCase | `stopAll`, `stopInstance` |
| Package names | lowercase, no underscore | `container`, `lock` |
| Constants | PascalCase | `Docker`, `Podman` |
| Variables | camelCase | `containerRuntime`, `ramMB` |

---

## 8. Security Considerations

### 8.1 Argument Injection Prevention

**Always use `--` separator** before user-provided arguments:

```go
// BAD - vulnerable to injection
exec.Command("tar", "-czf", backupFile, userPath)

// GOOD - uses separator
exec.Command("tar", "-czf", backupFile, "--", userPath)
```

**Examples in codebase**:
```go
// runtime.go
docker pull -- qdrant/qdrant

// clean.go
sudo tar -czf backup.tar.gz -- ~/.qdrant_storage01
sudo rm -rf -- ~/.qdrant_storage01
```

### 8.2 Path Traversal Prevention

**Validation in RunQdrant**:
```go
if strings.Contains(cfg.StorageDir, ":") {
    return fmt.Errorf("invalid storage directory: path cannot contain ':'")
}
```

**Why**: Colons are used in PATH separators and some injection techniques.

### 8.3 Symlink Attack Prevention

```go
// In filterStorageDirs()
if info.Mode()&os.ModeSymlink != 0 {
    logInfo(cmd, "Warning: %s is a symbolic link, skipping", match)
    continue
}
```

**Attack scenario**: Attacker creates symlink `~/.qdrant_storage01 -> /etc/critical-files`, clean command could accidentally delete system files.

### 8.4 Timeout Protection

```go
// Prevent hanging in CI/non-interactive environments
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()

tarCmd := exec.CommandContext(ctx, "sudo", tarArgs...)
```

### 8.5 File Permissions

```go
// Lock file: only owner can read/write (0600 = rw-------)
f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

// Storage directories: owner+group read/execute (0755 = rwxr-xr-x)
os.MkdirAll(storageDir, 0755)
```

### 8.6 Privilege Escalation Awareness

The `clean` command requires `sudo` because:
1. Docker containers run as root by default
2. Container creates files owned by root in `~/.qdrant_storageNN`
3. Normal user cannot delete root-owned files
4. Solution: `sudo rm -rf -- dirs...`

**Safety**: Always validate paths before sudo operations (symlink check, path traversal check).

---

## 9. Testing Guide

### 9.1 Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/system/...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

### 9.2 Test Patterns

**Table-Driven Tests** (from `resources_test.go`):
```go
func TestEstimateInstances(t *testing.T) {
    tests := []struct {
        ramMB         uint64
        wantStartup   uint64
        wantEfficient uint64
    }{
        {0, 0, 0},
        {256, 1, 0},
        {512, 2, 1},
        {1024, 4, 2},
        {4096, 16, 8},
    }
    for _, tt := range tests {
        gotStartup, gotEfficient := EstimateInstances(tt.ramMB)
        if gotStartup != tt.wantStartup || gotEfficient != tt.wantEfficient {
            t.Errorf("EstimateInstances(%d) = (%d, %d), want (%d, %d)", 
                tt.ramMB, gotStartup, gotEfficient, tt.wantStartup, tt.wantEfficient)
        }
    }
}
```

**File System Tests** (from `clean_test.go`):
```go
func TestFilterStorageDirs(t *testing.T) {
    // Create temp directory
    tmpDir, _ := os.MkdirTemp("", "spawn-qdrant-test")
    defer os.RemoveAll(tmpDir)  // Cleanup after test
    
    // Setup: create test files, symlinks
    regularDir := filepath.Join(tmpDir, "regular")
    os.Mkdir(regularDir, 0755)
    
    symlinkDir := filepath.Join(tmpDir, "symlink")
    os.Symlink("/etc", symlinkDir)
    
    // Execute: call function
    matches := []string{regularDir, symlinkDir}
    validated := filterStorageDirs(nil, matches)
    
    // Assert: check results
    if len(validated) != 1 || validated[0] != regularDir {
        t.Errorf("Expected only regular dir, got %v", validated)
    }
}
```

### 9.3 Integration Testing Strategy

**Challenge**: Testing container operations requires Docker/Podman.

**Approach 1: Mocking** (for unit tests)
```go
type ContainerRunner interface {
    RunQdrant(cfg QdrantConfig) error
    ListContainers() ([]string, error)
}

type MockRunner struct {
    Containers []string
}

func (m *MockRunner) RunQdrant(cfg QdrantConfig) error {
    m.Containers = append(m.Containers, cfg.Name)
    return nil
}
```

**Approach 2: Build Tags** (for integration tests)
```go
// +build integration

func TestIntegrationSpawn(t *testing.T) {
    if !container.IsRuntimeAvailable() {
        t.Skip("Docker/Podman not available")
    }
    // Run actual container operations
}
```

Run with: `go test -tags=integration ./...`

### 9.4 Test Checklist Before Committing

- [ ] `go test ./...` passes
- [ ] `go build` succeeds without warnings
- [ ] `go vet ./...` clean
- [ ] `gofmt -d .` shows no formatting issues
- [ ] New functions have corresponding test coverage
- [ ] Edge cases handled (0 instances, max RAM, etc.)

---

## 10. Common Tasks

### 10.1 Adding a New Command

**Step 1**: Create `cmd/newcommand.go`

```go
package cmd

import "github.com/spf13/cobra"

var newCmd = &cobra.Command{
    Use:   "newcommand [arg]",
    Short: "Brief description",
    Long:  `Detailed description`,
    Args:  cobra.ExactArgs(1),  // or other validation
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(newCmd)
    
    // Add flags
    newCmd.Flags().String("option", "default", "Description")
    viper.BindPFlag("option", newCmd.Flags().Lookup("option"))
}
```

**Step 2**: Test manually
```bash
go build -o spawn-qdrant main.go
./spawn-qdrant newcommand test
```

**Step 3**: Add tests in `cmd/newcommand_test.go`

### 10.2 Adding a New Container Operation

**Location**: `internal/container/runtime.go`

```go
func NewOperation(name string) error {
    return runCommand("operation", "--", name)
}
```

**Pattern**: Always use `--` separator, return wrapped errors.

### 10.3 Adding Configuration Options

**Location**: `cmd/root.go` in `init()`

```go
rootCmd.PersistentFlags().String("new-option", "default", "Description")
viper.BindPFlag("new-option", rootCmd.PersistentFlags().Lookup("new-option"))
```

**Usage**:
```go
// In any command
value := viper.GetString("new-option")
```

### 10.4 Building Release Binary

```bash
# Production build with version info
go build -ldflags "-X github.com/thelaonerd/spawn-qdrant/cmd.Version=1.0.0 \
  -X github.com/thelaonerd/spawn-qdrant/cmd.Commit=$(git rev-parse --short HEAD) \
  -X github.com/thelaonerd/spawn-qdrant/cmd.BuildDate=$(date -u +%Y-%m-%d)" \
  -o spawn-qdrant main.go

# Verify
./spawn-qdrant version
```

---

## 11. Troubleshooting

### 11.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| **"Lock file exists"** | Previous run crashed | `rm ~/.spawn-qdrant.lock` |
| **"neither docker nor podman"** | Runtime not installed or not in PATH | Install Docker/Podman, verify `docker ps` works |
| **"Permission denied during clean"** | Storage owned by root (from containers) | Run with sudo or ensure passwordless sudo configured |
| **"Port already in use"** | Conflicting services or previous containers | `docker ps` to check, `docker rm` to remove conflicts |
| **"Insufficient RAM"** | Requested instances exceed available memory | Run `check` command first to see limits |
| **"docker: command not found"** | Docker not in PATH for non-login shells | Use full path or add to PATH in ~/.bashrc |

### 11.2 Exit Codes

Defined in `main.go`:

| Code | Meaning | When Returned |
|------|---------|---------------|
| 0 | Success | Command completed successfully |
| 1 | Generic failure | Unexpected error during execution |
| 64 | Usage error | Invalid arguments, missing required flags |
| 65 | Data error | RAM insufficient, port conflicts |
| 71 | System error | Sudo failed, missing tools, lock issues |
| 130 | Cancelled | User interrupted with Ctrl+C/SIGINT |

**Usage in scripts**:
```bash
./spawn-qdrant spawn 5
if [ $? -eq 65 ]; then
    echo "Not enough RAM, try fewer instances"
fi
```

### 11.3 Debug Mode

```bash
# Verbose output (if implemented in command)
./spawn-qdrant -v spawn 2

# Or use Go's debug output
DEBUG=1 ./spawn-qdrant spawn 2

# Check logs
journalctl -u docker  # If using systemd

# Inspect container logs
docker logs qdrant-01
```

### 11.4 Recovery Procedures

**Scenario 1: Lock file stuck**
```bash
# Check if process is actually running
ps aux | grep spawn-qdrant

# If not running, safe to remove
rm ~/.spawn-qdrant.lock
```

**Scenario 2: Partial spawn (some containers created)**
```bash
# Check what exists
docker ps -a | grep qdrant

# Stop and clean up
./spawn-qdrant stop all
# or manually:
docker rm -f qdrant-01 qdrant-02
```

**Scenario 3: Network conflict**
```bash
# Check existing networks
docker network ls | grep qdrant

# Remove if stuck
docker network rm qdrant_network
```

---

## Summary

### Key Takeaways for New Developers

1. **Architecture is layered**: `cmd/` (presentation) → `internal/` (business logic) → Docker/Host (infrastructure)

2. **Always use `--` before user paths** in external commands (security against injection)

3. **Cleanup resources on error**: Use defer for locks, manual cleanup before early returns

4. **Respect context cancellation**: Check `ctx.Done()` in loops and blocking operations

5. **Follow Cobra patterns**: Each command is a `*cobra.Command` with `init()` registration

6. **Test before committing**: `go test ./...`, `go vet`, `gofmt`

### Architecture Strengths

- **Clean separation**: CLI and business logic are decoupled
- **Runtime-agnostic**: Docker/Podman abstraction
- **Security-conscious**: Symlink checks, argument separators, path validation
- **Signal-aware**: Graceful shutdown on interruption
- **Well-tested**: Critical paths (RAM calc, path filtering) have unit tests

### Areas for Potential Extension

| Feature | Complexity | Notes |
|---------|------------|-------|
| Add `logs` command | Low | `docker logs` wrapper |
| Add `status` command | Low | List running instances with health |
| Custom Qdrant versions | Medium | Flag `--version v1.2.3` |
| Health check after spawn | Medium | HTTP check on REST port |
| Podman-specific networking | Medium | Rootless podman networking differs |
| Docker Compose export | High | Generate docker-compose.yml |
| Remote deployment | High | SSH to remote Docker daemon |

---

**Document Version**: 1.0  
**Last Updated**: 2026-04-09  
**Author**: Developer Handover Team
