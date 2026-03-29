# Security Action Plan

This document outlines potential security issues identified in the `spawn-qdrant` codebase, categorized by criticality, along with recommended actions.

## High Criticality [RESOLVED]

### 1. Privilege Escalation / Arbitrary File Read via Sudo & Globbing (`cmd/clean.go`)
*   **Status:** FIXED
*   **Action taken:** 
    *   Added `filterStorageDirs` using `os.Lstat` to reject symlinks.
    *   Added `--` separator to `tar` and `rm` commands.

## Medium Criticality [RESOLVED/ADDRESSED]

### 2. Unauthenticated Network Exposure (`cmd/spawn.go`)
*   **Status:** ADDRESSED (Won't Fix)
*   **Reasoning:** Environment is confirmed to be a private network; `0.0.0.0` binding is retained for accessibility within the private subnet as per user requirement.

### 3. TTY Hangs in Non-Interactive Environments (`cmd/clean.go`)
*   **Status:** FIXED
*   **Action taken:**
    *   Implemented `context.WithTimeout` (10m for backup, 5m for cleanup).
    *   Added `isatty` check for `os.Stdin` attachment.
    *   Added logging before `sudo` execution.

## Low Criticality / Defense in Depth [RESOLVED]

### 4. Local Denial of Service via Lock File (`internal/lock/lockfile.go`)
*   **Status:** FIXED
*   **Action taken:**
    *   Switched to atomic file creation using `os.O_CREATE | os.O_EXCL`.
    *   Set strict file permissions to `0600` (owner read/write only).

### 5. Parameter Injection Surface in Container Abstraction (`internal/container/runtime.go`)
*   **Status:** FIXED
*   **Action taken:**
    *   Unexported generic command runners (`runCommand`, `runCommandOutput`).
    *   Introduced specialized, strictly-typed functions for all container operations.
    *   Updated all call sites to use the new typed API.
