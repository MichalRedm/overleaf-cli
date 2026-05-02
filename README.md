# Overleaf CLI

[![Release](https://github.com/MichalRedm/overleaf-cli/actions/workflows/release.yml/badge.svg)](https://github.com/MichalRedm/overleaf-cli/actions/workflows/release.yml)
[![CI](https://github.com/MichalRedm/overleaf-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/MichalRedm/overleaf-cli/actions/workflows/ci.yml)

**Overleaf CLI** is a robust, enterprise-ready utility designed to synchronize local directory structures with **any** Overleaf instance—including self-hosted Community Edition, Professional/Server Pro, and institutional Enterprise deployments with SSO (SAML/OAuth). It provides a seamless bridge between your favorite local LaTeX editors and the Overleaf web interface.

## Key Features

- **Universal Compatibility**: Works with standard Overleaf.com, self-hosted ShareLaTeX, and institutional University instances.
- **Advanced Authentication**: Native support for standard logins and delegated support for complex SSO/SAML/OAuth flows via custom authentication scripts.
- **Native Entity Discovery**: Bypasses restricted REST APIs using a native Go `socket.io` implementation to retrieve project structures directly from the server.
- **Bidirectional-ish Sync**: Authoritative local-to-remote mirroring (`push`) with intelligent orphan pruning.
- **Background Watch Mode**: Automatically sync changes to the cloud the moment you save them locally.
- **Project Management**: Create, delete, and initialize projects directly from the terminal.
- **Cloud Compilation**: Trigger remote builds and stream LaTeX errors/warnings or download the resulting PDF.
- **Docker Integration**: Optimized performance for local self-hosted instances via direct container interaction.

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
   This will create a `.overleaf/` directory containing `config.json`. Provide your Overleaf URL, credentials (email/password), and Project ID.
   *(Legacy `overleaf_config.json` will be automatically migrated to `.overleaf/config.json` on the first run).*

2. **Sync Local Files (Incremental)**:
   ```bash
   overleaf-cli push --src ./my-project --delete
   ```
   By default, the CLI uses a local state tracker (`.overleaf/state.json`) to only upload files that have actually changed. Use `--force` to re-upload everything.

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

For instances using non-standard login (e.g., SAML, OAuth, SSO), you can use the `custom` authentication type. This allows you to point to an external script (e.g. using Playwright or Puppeteer) that handles the complex login flow and returns the session cookie.

1. Set `"auth_type": "custom"` and `"auth_command": "python auth_script.py"` in your config.
2. The command will receive `OVERLEAF_EMAIL`, `OVERLEAF_PASSWORD`, and `OVERLEAF_URL` as environment variables.
3. The command must print **ONLY the raw session cookie value** (e.g., the value of `overleaf.sid` or `sharelatex.sid`) to `stdout`. 
   - **Important**: Any logging or debug messages must be redirected to `stderr`.
   - **Tip**: Ensure your script waits for the redirect to `/project` before capturing the cookie to guarantee the session is fully established.

### Native Entity Discovery

On instances where the standard REST API is restricted (e.g., returns paths without IDs), the CLI automatically uses **Native Discovery**:
- It establishes a temporary Socket.io connection to the server.
- It joins the project and retrieves the full directory structure (tree) with IDs.
- This process is fully automatic and requires no manual configuration or external scripts.

## Hybrid Mode (Web API vs Docker)

The CLI supports two distinct operational modes based on your `overleaf_config.json`:

- **Docker Mode**: Best for local self-hosted instances. Requires access to the Docker socket. Allows direct database access for faster status checks and direct file system access for PDF retrieval.
- **Web API Mode**: Best for remote hosted instances or restricted environments. Relies exclusively on the standard Overleaf HTTP API. Slower, but requires no infrastructure permissions.

## Command Reference

| Command | Description |
| :--- | :--- |
| `install` | Add the current binary's directory to the system PATH. |
| `init` | Interactive setup for `.overleaf/config.json`. |
| `push` | Incremental upload of local files to the Overleaf project. Use `--delete` to prune remote orphans, `--force` to bypass state tracking. |
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
