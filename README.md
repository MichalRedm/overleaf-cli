# Overleaf CLI

[![Release](https://github.com/MichalRedm/overleaf-cli/actions/workflows/release.yml/badge.svg)](https://github.com/MichalRedm/overleaf-cli/actions/workflows/release.yml)
[![CI](https://github.com/MichalRedm/overleaf-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/MichalRedm/overleaf-cli/actions/workflows/ci.yml)

Overleaf CLI is a robust, single-binary utility designed to synchronize local directory structures with self-hosted Overleaf (ShareLaTeX) Community Edition instances. It provides a seamless bridge between your favorite local LaTeX editors and the Overleaf web interface.

## Key Features

- **Bidirectional-ish Sync**: Authoritative local-to-remote mirroring (`push`) with optional remote orphan deletion.
- **Fast & Reliable**: Written in Go for high performance and minimal footprint.
- **Automatic Auth**: Log in once and let the tool manage your session cookies.
- **Docker Integration**: Directly queries MongoDB and accesses logs/PDFs via Docker for maximum reliability in self-hosted environments.
- **Background Watch Mode**: Automatically sync changes as you save them locally.
- **Project Lifecycle Management**: Create and delete projects via CLI (powered by Playwright).
- **Log Streaming**: Tail LaTeX compilation logs directly in your terminal.
- **Custom Authentication**: Support for non-standard login flows (SAML/OAuth) via external scripts.
- **Hybrid Synchronization**: Works with both local Docker instances and remote Overleaf instances via Web API.

## 🚀 Quick Start

### Installation

**Download Binary:**
Download the latest binary for your platform from the [Releases](https://github.com/MichalRedm/overleaf-cli/releases) page.

```bash
go install github.com/MichalRedm/overleaf-cli@latest
```

**After downloading or building:**
Run the following command to add `overleaf-cli` to your system PATH:
```bash
overleaf-cli install
```

### Configuration

1. **Initialize your project**:
   ```bash
   overleaf-cli init
   ```
   This will create an `overleaf_config.json` file. Provide your Overleaf URL, credentials (email/password), and Project ID.

2. **Sync Local Files**:
   ```bash
   overleaf-cli push --src ./my-project --delete
   ```

3. **Start Watch Mode**:
   ```bash
   overleaf-cli watch --src ./my-project
   ```

4. **Compile and Download PDF**:
   ```bash
   overleaf-cli compile
   overleaf-cli pdf --out report.pdf
   ```

### Custom Authentication

For instances using non-standard login (e.g., SAML, OAuth, SSO), you can use the `custom` authentication type. This allows you to point to an external script that handles the login and returns the session cookie.

1. Set `"auth_type": "custom"` and `"auth_command": "python scripts/auth_put.py"` in your config.
2. The command will receive `OVERLEAF_EMAIL`, `OVERLEAF_PASSWORD`, and `OVERLEAF_URL` as environment variables.
3. The command must print **ONLY the raw session cookie value** (e.g., the value of `overleaf.sid`) to `stdout`. 
   - **Important**: Any logging or debug messages must be redirected to `stderr`.
   - **Tip**: Ensure your script waits for the redirect to `/project/:id` before capturing the cookie to guarantee the session is valid.

See [scripts/auth_put.py](scripts/auth_put.py) for a reference implementation.

### Native Entity Discovery

On older or restricted Overleaf instances where the standard REST API doesn't return entity IDs, the CLI automatically falls back to **Native Discovery**:
- It establishes a temporary Socket.io connection to the server.
- It joins the project and retrieves the full directory structure (tree) with IDs.
- This process is fully automatic and requires no user configuration.
- If native discovery fails, you can provide an optional `discovery_command` in your config to use an external scraper.

## Hybrid Mode (Web API vs Docker)

The CLI supports two distinct operational modes based on your `overleaf_config.json`:

- **Docker Mode**: Best for local self-hosted instances. Requires access to the Docker socket. Allows direct database access for faster status checks and direct file system access for PDF retrieval.
- **Web API Mode**: Best for remote hosted instances or restricted environments. Relies exclusively on the standard Overleaf HTTP API. Slower, but requires no infrastructure permissions.

## Command Reference

| Command | Description |
| :--- | :--- |
| `install` | Add the current binary's directory to the system PATH. |
| `init` | Interactive setup for `overleaf_config.json`. |
| `push` | Upload local files to the Overleaf project. Use `--delete` to prune remote orphans. |
| `watch` | Watch the local directory and push changes immediately on save. |
| `compile` | Trigger a LaTeX compilation on the server. |
| `logs` | Retrieve and display LaTeX errors/warnings from the container. |
| `pdf` | Download the latest compiled PDF from the container. |
| `project create` | Create a new project in Overleaf. |
| `project rm` | Permanently delete a project. |

## Prerequisites

- **Docker**: (Optional) Required for optimized log/PDF access on local self-hosted instances.
- **Node/npx**: (Optional) Required for project creation/deletion.
- **Python**: (Optional) Required if using the provided custom auth scripts.

## License

MIT
