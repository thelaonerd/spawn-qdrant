# Security Action Plan

This document outlines potential security issues identified in the `spawn-qdrant` codebase, categorized by criticality, along with recommended actions.

## High Criticality

### 1. Privilege Escalation / Arbitrary File Read via Sudo & Globbing (`cmd/clean.go`)
*   **Description:** The `clean` command uses `filepath.Glob("~/.qdrant_storage*")` to find directories, then passes these absolute paths directly to `sudo tar -czf` and `sudo rm -rf`. If a malicious local process or user creates a symlink in the user's home directory (e.g., `ln -s /etc/shadow ~/.qdrant_storage_shadow`), the `sudo tar` command will archive the sensitive files as root.
*   **Suggested Action:**
    *   **Verify Paths:** Iterate through the results of `filepath.Glob` and use `os.Lstat` to verify that each path is a directory and **not** a symlink before passing it to `sudo`.
    *   **Prevent Parameter Injection:** Use `--` before passing variable arguments to shell commands (e.g., `exec.Command("sudo", "rm", "-rf", "--", path)`).
    *   *Note:* Sudo escalation is intentionally retained as Docker-created containers bind to volumes with root group and owner.

## Medium Criticality

### 2. Unauthenticated Network Exposure (`cmd/spawn.go`)
*   **Description:** When spawning instances, the port bindings are configured as `-p {REST}:6333 -p {GRPC}:6334`. By default, Docker/Podman will bind these to `0.0.0.0`, exposing the Qdrant instances to all network interfaces. Since no authentication is configured, they are open to anyone on the network.
*   **Suggested Action:** Explicitly bind the ports to localhost by changing the argument to `-p 127.0.0.1:{REST}:6333` and `-p 127.0.0.1:{GRPC}:6334`.

### 3. TTY Hangs in Non-Interactive Environments (`cmd/clean.go`)
*   **Description:** The cleanup process forcibly executes `sudo` and attaches `os.Stdin` to allow for a password prompt. In non-interactive environments (like CI/CD) or if the user doesn't notice the prompt, the application will silently hang waiting for a password.
*   **Suggested Action:** Implement a timeout for the `sudo` command or check if the session is interactive before attaching `os.Stdin`. Provide clear logging immediately before the `sudo` command executes so the user knows a password might be required.

## Low Criticality / Defense in Depth

### 4. Local Denial of Service via Lock File (`internal/lock/lockfile.go`)
*   **Description:** The concurrency lock relies on checking for the existence of `~/.spawn-qdrant.lock`. Any local process running under the same user can create this file, change its permissions, or hold it open, preventing `spawn-qdrant` from executing.
*   **Suggested Action:** Rely on POSIX file locking (e.g., Go's `syscall.Flock`) rather than basic `os.Stat` existence checks. Ensure the lock file is created with strict permissions (`0600`).

### 5. Parameter Injection Surface in Container Abstraction (`internal/container/runtime.go`)
*   **Description:** The `RunCommand(args ...string)` function is highly generic and passes arguments straight to the underlying runtime binary. While current usage is safe, any future addition that passes raw string input could lead to parameter injection.
*   **Suggested Action:** Avoid generic command runners where possible. Wrap operations in specific, strictly-typed functions (e.g., `RunQdrantContainer(config ContainerConfig)`) that sanitize or tightly control the arguments passed to `exec.Command`.
