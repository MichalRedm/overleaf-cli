---
name: overleaf-cli
description: Agentic skill to synchronize local projects with Overleaf using the overleaf-cli utility. Supports push, watch, compile, and project management.
---

# Overleaf CLI Skill

This skill allows the agent to synchronize a local project directory with an Overleaf instance (self-hosted or cloud) using the `overleaf-cli` utility.

## Prerequisites

- **Binary**: `overleaf-cli` must be installed in the system PATH.
- **Configuration**: An `overleaf_config.json` file in the project root or specified via `--config`.

## Usage

### 1. Project Initialization
```powershell
overleaf-cli init
```

### 2. Synchronization
```powershell
# Push local changes to Overleaf
overleaf-cli push --src <local_dir>

# Mirror local to remote (delete remote orphans)
overleaf-cli push --src <local_dir> --delete
```

### 3. Background Watch
```powershell
# Automatically sync on file save
overleaf-cli watch --src <local_dir> --delete
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

## Best Practices
- **Auto-Login**: Provide `email` and `password` in the config for seamless session management.
- **Root Folder**: Use `root_folder_id` in config to sync to a specific subfolder if needed.
