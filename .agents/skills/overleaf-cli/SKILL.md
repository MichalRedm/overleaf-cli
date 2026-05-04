---
name: overleaf-cli
description: Agentic skill to synchronize local projects with Overleaf using the overleaf-cli utility. Supports push, watch, compile, and project management.
---

# Overleaf CLI Skill

This skill allows the agent to synchronize a local project directory with an Overleaf instance (self-hosted or cloud) using the `overleaf-cli` utility.

## Prerequisites

- **Binary**: `overleaf-cli` must be installed in the system PATH.
- **Zero-Config Root Detection**: The tool automatically detects the project root by searching upwards for a `.overleaf/` directory. This makes the `--src` parameter optional when run from within a project.
- **Configuration**: A `.overleaf/config.json` file. It is automatically detected if it resides in the current directory, project root, or relative to the `--src` path.
- **Incremental Sync**: Uses `.overleaf/state.json` to track local file hashes and minimize network traffic.
- **Hybrid Mode**: Supports both Docker-based (local) and Web API-based (remote) synchronization.

## Usage

### 0. Installation (if not in PATH)
```powershell
overleaf-cli install
```

### 1. Project Initialization
```powershell
overleaf-cli init
```
*Note: Legacy `overleaf_config.json` files are automatically migrated to `.overleaf/config.json`.*

### 2. Synchronization
```powershell
# Push local changes to Overleaf (Incremental)
# --src is optional if run within a project folder
overleaf-cli push

# Force re-upload of all files (bypass state tracking)
overleaf-cli push --force

# Mirror local to remote (delete remote orphans)
overleaf-cli push --delete

# Explicitly specify source directory
overleaf-cli push --src <local_dir>
```

### 3. Background Watch
```powershell
# Automatically sync on file save
# --src is optional if run within a project folder
overleaf-cli watch --delete
```

### 4. Compilation & PDF
```powershell
# Trigger compilation
overleaf-cli compile

# View logs
overleaf-cli logs

# Download PDF
overleaf-cli pdf --out <filename.pdf>
```

### 5. Project Management
```powershell
# Create a new project
overleaf-cli project create --name "My Project"

# Delete the current project
overleaf-cli project rm
```

## Handling Non-Standard Authentication

If a standard email/password login is not supported by the Overleaf instance (e.g., SAML, OAuth, SSO), the agent must:

1. **Research the Login Flow**: Use `playwright-cli` to explore the instance's login page and identify the authentication mechanism.
2. **Implement an Auth Script**: Create a script (e.g., in `scripts/auth.py`) that performs the login and outputs the **raw session cookie value** to `stdout`.
    - **CRITICAL**: The script must output ONLY the cookie value. All logging, progress info, or errors MUST be directed to `stderr`.
    - **Wait for Project**: Ensure the script waits for the final redirect to a project URL (`/project/**`) before extracting the cookie to ensure the session is fully established.
3. **Configure CLI**: Set `auth_type` to `custom` and `auth_command` to run the script (e.g., `python scripts/auth.py`).
4. **Environment Variables**: The CLI will automatically pass `OVERLEAF_EMAIL`, `OVERLEAF_PASSWORD`, and `OVERLEAF_URL` to the script.

## Entity Discovery (Native)

On instances where the standard REST API is restricted (e.g., returns paths without IDs), the CLI implements **Native Discovery**:
- **Native Websocket**: The CLI automatically establishes a Socket.io connection to fetch the project tree directly from the server. This is the most reliable method for non-standard university instances.

## Best Practices
- **Auto-Login**: Provide `email` and `password` in the config for seamless session management. Custom authentication flows will also trigger auto-login if the session expires.
- **Git Hygiene**: Add `.overleaf/` to your `.gitignore` to prevent leaking credentials and local state.
- **Troubleshooting Sync**: If files are not uploading as expected, use the `--force` flag to reset the synchronization state.
- **Docker vs Web API**: Set `use_docker: true` only if you have direct access to the `sharelatex` Docker container. Otherwise, set to `false` to use the standard Web API.

## Troubleshooting

- **"Failed to retrieve entities"**: This usually means the binary is outdated or the session is invalid.
    1. **Update Binary**: Run `go install` from the `overleaf-cli` source directory to ensure you have the latest native websocket discovery fixes.
    2. **Check Auth**: Verify your custom auth script (if used) outputs a valid cookie.
    3. **Clear Cache**: Remove the `cookie` value from `.overleaf/config.json` to force a fresh login.
- **Stale State**: If the remote and local state are out of sync, delete `.overleaf/state.json` and run `push --force`.
