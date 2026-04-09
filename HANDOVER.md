# spawn-qdrant - Developer Handover Documentation

> **Project**: CLI tool for spawning and managing multiple Qdrant instances using Docker/Podman  
> **Target Audience**: Go developers with 1-2 years experience, familiar with Docker/containerization  
> **Technology Stack**: Go 1.25+, Cobra CLI, Docker/Podman

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Architecture Overview](#2-architecture-overview)
3. [End-to-End Workflow Documentation](#3-end-to-end-workflow-documentation)
   - Workflow 1: Spawn Instances
   - Workflow 2: Stop Instances
   - Workflow 3: Clean & Backup
4. [Directory Structure](#4-directory-structure)
5. [Key Patterns & Conventions](#5-key-patterns--conventions)
6. [Security Considerations](#6-security-considerations)
7. [Testing Guide](#7-testing-guide)
8. [Common Tasks](#8-common-tasks)
9. [Troubleshooting](#9-troubleshooting)

---

## 1. Project Overview

**spawn-qdrant** is a CLI utility that simplifies running multiple isolated Qdrant (vector database) instances on Linux using Docker or Podman.

### Key Features
- Spawn N isolated Qdrant instances with auto-incrementing ports
- Automatic runtime detection (Docker → Podman fallback)
- Smart RAM estimation before spawning
- Safe cleanup with backup functionality
- File-based locking to prevent concurrent operations

### Prerequisites
- Linux OS
- Docker or Podman installed
- Go 1.25+ (for development)

---

## 2. Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Main Entry                           │
│                         main.go                             │
└───────────────────────┬─────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────┐
│                      Cobra Commands                         │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ │
│  │  check  │ │  spawn  │ │   stop  │ │  clean  │ │ version │ │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘ │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
┌───────▼──────┐ ┌──────▼──────┐ ┌─────▼──────┐
│   internal/  │ │  internal/  │ │  internal/ │
│   container/  │ │    lock/     │ │   system/  │
│  (docker API) │ │(file locking)│ │  (RAM chk) │
└──────────────┘ └─────────────┘ └────────────┘
```

### Layered Architecture Pattern

The project follows a **Layered Architecture** with clear separation of concerns:

| Layer | Package | Responsibility |
|-------|---------|----------------|
| **Presentation** | `cmd/` | CLI commands, user interaction, flag parsing |
| **Business Logic** | `internal/*` | Container operations, resource checks, locking |
| **System/Infra** | `internal/container/`, `internal/system/` | External system interactions (Docker, OS) |

---

## 3. End-to-End Workflow Documentation

### Workflow 1: Spawn Instances (`spawn-qdrant spawn 3`)

**Business Purpose**: Creates N Qdrant container instances with isolated storage and incremental ports.

**Files Involved**:
- `cmd/spawn.go` - Command handler
- `internal/container/runtime.go` - Docker/Podman operations
- `internal/lock/lockfile.go` - Concurrency control
- `internal/system/resources.go` - RAM validation

#### Entry Point (`cmd/spawn.go`)

```go
var spawnCmd = &cobra.Command{
    Use:   "spawn [instance_count]",
    Short: "Spawn qdrant instances",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation details below...
    },
}
```

**Key Steps**:

1. **Signal Handling Setup** (Lines 37-39)
```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```
> Creates a context that listens for SIGINT/SIGTERM for graceful shutdown.

2. **Lock Acquisition** (Lines 42-46)
```go
if len(args) > 0 {
    if err := lock.Create(); err != nil {
        return err
    }
}
```
> Prevents concurrent spawn operations. Lock file: `~/.spawn-qdrant.lock`

3. **RAM Validation** (Lines 48-56)
```go
ramMB, err := system.GetAvailableRAM()
maxStartup, maxEfficient := system.EstimateInstances(ramMB)
```
> Checks `/proc/meminfo` for available RAM and calculates max instances.

4. **Image Check** (Lines 95-99)
```go
if err := container.EnsureImage("qdrant/qdrant"); err != nil {
    lock.Remove()
    return fmt.Errorf("failed to ensure qdrant image: %w", err)
}
```
> Pulls image if not present locally.

5. **Network Creation** (Line 103)
```go
_ = container.CreateNetwork("qdrant_network")
```
> Creates Docker network for inter-container communication.

6. **Instance Loop** (Lines 115-170)
```go
for i := 0; i < n; i++ {
    // Port calculation: startRest + (2 * i)
    restPort := startRest + (2 * i)
    grpcPort := startGrpc + (2 * i)
    
    container.RunQdrant(container.QdrantConfig{
        Name:       containerName,  // "qdrant-01", "qdrant-02", etc.
        Network:    "qdrant_network",
        RestPort:   restPort,      // 6333, 6335, 6337...
        GrpcPort:   grpcPort,      // 6334, 6336, 6338...
        StorageDir: storageDir,     // ~/.qdrant_storage01...
    })
    
    // 30-second delay between instances (except last)
    time.After(30 * time.Second)
}
```

#### Container Runtime Layer (`internal/container/runtime.go`)

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

**Key Pattern**: All commands use `--` separator to prevent argument injection:
```go
func runCommand(args ...string) error {
    cmd := exec.Command(string(containerRuntime), args...)
    return cmd.Run()
}
// Usage: runCommand("pull", "--", imageName)
```

#### Resource Estimation (`internal/system/resources.go`)

```go
func GetAvailableRAM() (uint64, error) {
    file, err := os.Open("/proc/meminfo")
    // Parse "MemAvailable:" line
    // Returns MB
}

func EstimateInstances(availableRAMMB uint64) (maxStartup uint64, maxEfficient uint64) {
    maxStartup = availableRAMMB / 256    // 256MB per instance (minimum)
    maxEfficient = availableRAMMB / 512  // 512MB per instance (recommended)
    return
}
```

---

### Workflow 2: Stop Instances (`spawn-qdrant stop all` or `spawn-qdrant stop 2`)

**Business Purpose**: Gracefully stops and removes Qdrant containers and network.

**Files Involved**:
- `cmd/stop.go`
- `internal/container/runtime.go`
- `internal/lock/lockfile.go`

#### Entry Point (`cmd/stop.go`)

```go
var stopCmd = &cobra.Command{
    Use:   "stop [all|n]",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        arg := args[0]
        if arg == "all" {
            return stopAll(cmd)
        }
        n, _ := strconv.Atoi(arg)
        return stopInstance(cmd, n)
    },
}
```

#### Stop All Flow (`stopAll` function)

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

#### Stop Single Instance (`stopInstance` function)

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

---

### Workflow 3: Clean & Backup (`spawn-qdrant clean`)

**Business Purpose**: Destructive cleanup with backup - stops containers, backs up data to tar.gz, deletes storage.

**Files Involved**:
- `cmd/clean.go`
- `cmd/stop.go` (reuses `stopAll`)

#### Entry Point (`cmd/clean.go`)

```go
var cleanCmd = &cobra.Command{
    Use:   "clean",
    Short: "Stop instances, backup storage, and clean up",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Interactive confirmation (if TTY)
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
    },
}
```

#### Security: Symlink Validation

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

#### Security: TTY Detection

```go
func isatty(f *os.File) bool {
    var termios syscall.Termios
    _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, f.Fd(), 
                 uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
    return err == 0
}
```
> Used to determine if interactive prompt should be shown.

---

## 4. Directory Structure

```
spawn-qdrant/
├── main.go                          # Entry point, exit code handling
├── go.mod                           # Go module definition
├── go.sum                           # Dependency checksums
│
├── cmd/                             # Cobra CLI commands
│   ├── root.go                      # Root command, config init, logging helpers
│   ├── spawn.go                     # Spawn instances command
│   ├── stop.go                      # Stop instances command
│   ├── clean.go                     # Clean/backup command
│   ├── check.go                     # RAM check command
│   ├── version.go                   # Version command
│   ├── completion.go                # Shell completion generator
│   └── clean_test.go                # Tests for clean command
│
└── internal/                        # Private application code
    ├── container/
    │   └── runtime.go               # Docker/Podman abstraction
    ├── lock/
    │   └── lockfile.go              # File-based locking
    ├── system/
    │   ├── resources.go             # RAM checking
    │   └── resources_test.go        # Resource calculation tests
    └── config/
        └── config.go                # Configuration loading (mostly unused)
```

---

## 5. Key Patterns & Conventions

### 5.1 Command Pattern (Cobra)

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

### 5.2 Error Handling Pattern

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Cleanup on error (important pattern)
if err := riskyOperation(); err != nil {
    lock.Remove()  // Always cleanup resources on failure
    return err
}
```

### 5.3 Context Cancellation Pattern

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()

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

### 5.4 Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Commands | `[action]Cmd` | `spawnCmd`, `stopCmd` |
| Results/DTOs | `[Action]Result` | `SpawnResult`, `CheckResult` |
| Config Structs | PascalCase | `QdrantConfig` |
| Private funcs | camelCase | `stopAll`, `stopInstance` |
| Package names | lowercase, no underscore | `container`, `lock` |
| Interface files | `pkg.go` in package | `runtime.go`, `lockfile.go` |

### 5.5 Container Runtime Abstraction

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

### 5.6 Configuration Hierarchy

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
```

---

## 6. Security Considerations

### 6.1 Argument Injection Prevention

Always use `--` separator before user-provided arguments:

```go
// BAD - vulnerable to injection
tarArgs := []string{"tar", "-czf", backupFile, userPath}

// GOOD - uses separator
tarArgs := []string{"tar", "-czf", backupFile, "--", userPath}
```

### 6.2 Path Traversal Prevention

```go
// Security check in RunQdrant
if strings.Contains(cfg.StorageDir, ":") {
    return fmt.Errorf("invalid storage directory: path cannot contain ':'")
}
```

### 6.3 Symlink Attack Prevention

```go
// In filterStorageDirs()
if info.Mode()&os.ModeSymlink != 0 {
    logInfo(cmd, "Warning: %s is a symbolic link, skipping", match)
    continue
}
```

### 6.4 Timeout Protection

```go
// Prevent hanging in CI/non-interactive environments
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()

tarCmd := exec.CommandContext(ctx, "sudo", tarArgs...)
```

### 6.5 File Permissions

```go
// Lock file: only owner can read/write
f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

// Storage directories: owner+group read/execute
os.MkdirAll(storageDir, 0755)
```

---

## 7. Testing Guide

### 7.1 Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/system/...
```

### 7.2 Test Patterns

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
    // Execute: call function
    // Assert: check results
}
```

### 7.3 Integration Testing

For container operations (not in current test suite), you would mock:

```go
// Example pattern for mocking exec.Command
type CommandRunner interface {
    Run(args ...string) error
    Output(args ...string) (string, error)
}

type RealRunner struct{}
func (r RealRunner) Run(args ...string) error {
    cmd := exec.Command(string(containerRuntime), args...)
    return cmd.Run()
}
```

---

## 8. Common Tasks

### 8.1 Adding a New Command

```go
// cmd/newcommand.go
package cmd

import "github.com/spf13/cobra"

var newCmd = &cobra.Command{
    Use:   "newcommand [arg]",
    Short: "Description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(newCmd)
}
```

### 8.2 Adding a New Container Operation

```go
// internal/container/runtime.go

func NewOperation(name string) error {
    return runCommand("operation", "--", name)
}
```

### 8.3 Adding Configuration Options

```go
// In root.go init()
rootCmd.PersistentFlags().String("new-option", "default", "Description")
viper.BindPFlag("new-option", rootCmd.PersistentFlags().Lookup("new-option"))

// Usage in commands
value := viper.GetString("new-option")
```

### 8.4 Building the Binary

```bash
# Development build
go build -o spawn-qdrant main.go

# Production build with version info
go build -ldflags "-X github.com/thelaonerd/spawn-qdrant/cmd.Version=1.0.0 \
  -X github.com/thelaonerd/spawn-qdrant/cmd.Commit=abc123 \
  -X github.com/thelaonerd/spawn-qdrant/cmd.BuildDate=$(date -u +%Y-%m-%d)" \
  -o spawn-qdrant main.go
```

---

## 9. Troubleshooting

### 9.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "Lock file exists" | Previous run crashed | `rm ~/.spawn-qdrant.lock` |
| "neither docker nor podman" | Runtime not installed | Install Docker or Podman |
| Permission denied during clean | Storage owned by root | Run with sudo or check sudoers |
| Port already in use | Conflicting services | Check with `docker ps` or change ports |
| "Insufficient RAM" | Too many instances requested | Run `check` command first |

### 9.2 Exit Codes

Defined in `main.go`:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Generic failure |
| 64 | Usage error (invalid args, flags) |
| 65 | Data error (RAM, ports) |
| 71 | System error (sudo, missing tools, locks) |
| 130 | Cancelled/interrupted |

### 9.3 Debug Mode

```bash
# Verbose output
spawn-qdrant -v spawn 2

# JSON output for scripting
spawn-qdrant -o json check
```

---

## Summary

### Key Takeaways for New Developers

1. **Always use `--` before user paths** in external commands (security)
2. **Cleanup resources on error** (locks, partial containers)
3. **Respect context cancellation** for graceful shutdowns
4. **Use Cobra patterns** consistently for new commands
5. **Run tests before committing** - `go test ./...`

### Architecture Strengths

- Clean separation between CLI and business logic
- Runtime-agnostic (Docker/Podman)
- Security-conscious (symlink checks, argument separators)
- Signal-aware (graceful shutdown)
- Well-tested critical paths

### Areas for Potential Extension

- Add `logs` command to view container logs
- Add `status` command to show running instances
- Support custom Qdrant versions (currently hardcoded)
- Add health check after spawn
- Support Podman-specific networking options
