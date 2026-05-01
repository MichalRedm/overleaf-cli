# Overleaf CLI

Overleaf CLI is a robust, single-binary utility designed to synchronize local directory structures with self-hosted Overleaf (ShareLaTeX) Community Edition instances. It provides a seamless bridge between your favorite local LaTeX editors and the Overleaf web interface.

## Key Features

- **Bidirectional-ish Sync**: Authoritative local-to-remote mirroring (`push`) with optional remote orphan deletion.
- **Fast & Reliable**: Written in Go for high performance and minimal footprint.
- **Docker Integration**: Directly queries MongoDB and accesses logs/PDFs via Docker for maximum reliability in self-hosted environments.
- **Background Watch Mode**: Automatically sync changes as you save them locally.
- **Project Lifecycle Management**: Create and delete projects via CLI (powered by Playwright).
- **Log Streaming**: Tail LaTeX compilation logs directly in your terminal.

## Installation

### Prerequisites
- **Docker**: The tool interacts with the `sharelatex` and `mongo` containers.
- **Node/npx**: (Optional) Required for project creation/deletion.

### Download
Download the latest binary for your OS from the [Releases](https://github.com/mizhal/overleaf-cli/releases) page.

### Build from Source
```bash
go mod download
go build -o overleaf-cli
```

## Quick Start

1. **Initialize Configuration**:
   ```bash
   overleaf-cli init
   ```
   This will prompt for your Overleaf URL, session cookie (`overleaf.sid`), and Project ID.

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

## Command Reference

| Command | Description |
| :--- | :--- |
| `init` | Interactive setup for `overleaf_config.json`. |
| `push` | Upload local files to the Overleaf project. Use `--delete` to prune remote orphans. |
| `watch` | Watch the local directory and push changes immediately on save. |
| `compile` | Trigger a LaTeX compilation on the server. |
| `logs` | Retrieve and display LaTeX errors/warnings from the container. |
| `pdf` | Download the latest compiled PDF from the container. |
| `project create` | Create a new project in Overleaf. |
| `project rm` | Permanently delete a project. |

## Configuration

The tool uses `overleaf_config.json` (default) which looks like this:

```json
{
    "base_url": "http://localhost:80",
    "project_id": "644f...",
    "cookie": "s%3A...",
    "root_folder_id": "644f..."
}
```

## License
MIT
