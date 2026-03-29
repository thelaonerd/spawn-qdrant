# BLUEPRINT.md

This document serves as an architectural template for CLI applications designed to spawn and manage $n$ container instances. It abstracts the core modularity and security concepts used to build robust local multi-container orchestration utilities.

## 1. Abstract / Overview

This blueprint describes the design of a generic CLI tool for deploying and managing multiple local containers. By utilizing dynamic port mapping, robust concurrency control, and system resource validation, the application safely scales to handle $n$ instances. The design employs an execution structure that bridges a command-line interface layer with runtime-agnostic internal abstractions.

## 2. Architectural Modularity (Directory Structure)

A separation of concerns should be established by dividing the application into two main layers:

### 2.1 Command Layer (`cmd/`)
The CLI entry points (e.g., implemented using a library like Cobra). This layer parses inputs and orchestrates the internal business logic.
*   **`root`**: Acts as the global entry point. Handles setup operations, such as initializing lock files to prevent concurrent execution conflicts.
*   **`spawn` / `start`**: Responsible for validating host resources against requested instance counts and sequentially creating containers.
*   **`check`**: Validates system requirements (e.g., minimum RAM/CPU limits) and reports capacity.
*   **`stop`**: Manages the graceful teardown of specific container instances or all running instances.
*   **`clean`**: Executes a destructive reset operation, safely backing up and subsequently removing related storage directories or volumes.

### 2.2 Internal Abstractions (`internal/`)
This layer encapsulates the specific mechanics of interacting with the host system and the container runtime, hiding these details from the CLI layer.
*   **`container/`**: A runtime-agnostic wrapper (supporting tools like Docker or Podman) to abstract away direct CLI/API invocations for container management.
*   **`system/`**: Responsible for host resource discovery. For instance, reading `/proc/meminfo` or using system APIs to calculate available RAM and CPU capacity. **Portability Note**: This module should be abstracted to handle different operating systems (e.g., Linux vs. macOS) via platform-specific implementations.
*   **`lock/`**: Implements concurrency control (e.g., via file-based locking) to ensure that simultaneous executions of the CLI do not cause race conditions during provisioning or teardown.
*   **`config/`**: Manages environment-based configurations (e.g., `.env` files) and sets application-wide defaults. To ensure portability, application-specific details (image names, default ports, prefixes) should be centralized here.

## 3. Core Orchestration Logic

When handling $n$ identical containers, specific orchestration logic must be applied to ensure system stability and predictable access.

*   **Sequential Provisioning**: Introducing explicit delays (e.g., a 30-second sleep interval) between container spawns mitigates CPU and I/O spikes that could overwhelm the host system during a mass-start event.
*   **Resource Management**: Before attempting to spawn $n$ containers, validate the host's capacity. Enforce a minimum resource requirement (e.g., MB of RAM per instance) to prevent out-of-memory (OOM) failures during startup or operation.
*   **Network Management**: Instances should be attached to a dedicated shared bridge network (e.g., `app_network`) to allow inter-container communication and isolation from other host services.
*   **Dynamic Port & Volume Mapping**: Formulas are needed to deterministically assign ports and storage paths based on the 1-based index $i$ of the instance (where $i$ ranges from 1 to $n$).
    *   **Suffixing**: Use 0-padded suffixes (e.g., `01`, `02`, ..., `10`) for container names and storage paths to ensure consistent sorting and identification.
    *   *Example Port Mapping*: `PORT_A = Base_Port_A + Step * (i - 1)`
    *   *Example Volume Mapping*: `STORAGE_PATH = ~/.app_storage_0i`

## 4. Security Best Practices

The tool requires privileged actions (like removing volumes or managing processes). The following practices are mandated to prevent privilege escalation or accidental data loss:

*   **Symlink Protection**: When dynamically generating host paths for container storage mounts, *always* validate that the path points to an actual directory and is not a symbolic link before performing destructive operations. Use `os.Lstat` to detect symlinks.
*   **Non-Interactive Safety (Timeouts)**: Background processes (like `tar` for backups or `rm` for deletions) must run with strict contextual timeouts (e.g., 5-10 minutes) using `context.WithTimeout`. This prevents the CLI from hanging indefinitely in non-interactive or CI/CD environments where `sudo` might be waiting for a password.
*   **Terminal (TTY) Awareness**: When invoking privileged child processes (such as `sudo`), only attach `os.Stdin` if an interactive terminal is detected (e.g., checking `isatty`). This ensures that scripts don't hang waiting for input that cannot be provided.
*   **Command Injection Prevention**: Always use argument separators (e.g., `--`) when passing variable or user-defined paths to underlying system commands to prevent them from being interpreted as flags (e.g., `rm -rf -- /path/to/remove`).
