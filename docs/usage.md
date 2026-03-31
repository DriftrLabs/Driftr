# CLI Usage

All commands support the `-v` / `--verbose` flag for detailed output.

## driftr install

Download and install a Node.js version.

```bash
# Install latest Node.js 22.x
driftr install node@22

# Install a specific version
driftr install node@22.14.0

# Verbose output (shows download URL, checksum verification)
driftr install node@22 -v
```

**What happens:**
1. If a partial version is given (e.g. `22`), Driftr resolves it to the latest matching release by querying the Node.js release index
2. Downloads the archive from `nodejs.org`
3. Verifies the SHA256 checksum against the official `SHASUMS256.txt`
4. Extracts the archive to `~/.driftr/tools/node/<version>/`

**Notes:**
- Reinstalling an already-installed version is a no-op
- Downloaded archives are cached in `~/.driftr/cache/`
- If checksum verification fails, the cached archive is deleted automatically
- If extraction fails, the partial installation is cleaned up

## driftr default

Set the global default Node.js version.

```bash
driftr default node@22.14.0
```

The global default is used whenever you run `node` outside a project with a pinned version.

**Requirements:**
- The version must already be installed

## driftr pin

Pin a Node.js version to the current project.

```bash
cd my-project
driftr pin node@22.14.0
```

On first use, Driftr prompts you to choose a storage format:

```
No existing project config found. How should the Node.js version be stored?
  1) .driftr.toml (recommended)
  2) package.json (driftr key)
Choose [1/2]:
```

Choosing `.driftr.toml` creates:

```toml
[tools]
node = "22.14.0"
```

Choosing `package.json` adds a `driftr` key to your existing `package.json`:

```json
{
  "name": "my-project",
  "driftr": {
    "node": "22.14.0"
  }
}
```

Subsequent `driftr pin` commands detect the existing format and reuse it automatically.

**Migrating between formats:**

```bash
# Switch from .driftr.toml to package.json (or vice versa)
driftr pin node@22.14.0 --migrate
```

This writes the version in the other format and removes the old config.

**Requirements:**
- The version must already be installed
- `package.json` format requires an existing `package.json` file (run `npm init` first)

**Behavior:**
- Anyone who clones the project and has Driftr set up will automatically use the pinned version
- The pinned version takes priority over the global default
- Nested directories inherit the pin until another config overrides it
- In non-interactive environments (CI), defaults to `.driftr.toml`

## driftr list

List all installed Node.js versions.

```bash
driftr list
```

Output example:

```
Installed Node.js versions:
    20.11.0
  * 22.14.0
    24.0.0

  * = global default
```

**Alias:** `driftr ls`

## driftr which

Show which binary Driftr would execute, and why.

```bash
driftr which node
```

Output example:

```
Tool:    node
Version: 22.14.0
Binary:  /home/user/.driftr/tools/node/22.14.0/bin/node
Source:  project config
Project: /home/user/my-project
```

**With verbose tracing:**

```bash
driftr which node -v
```

This shows each step of the resolution chain:

```
  [resolve] Starting Node.js version resolution
  [resolve] Step 1: No explicit override
  [resolve] Step 2: Searching for project config from /home/user/my-project
  [resolve]   Checking: /home/user/my-project/.driftr.toml
  [resolve] Resolved: 22.14.0 from project config (/home/user/my-project)
Tool:    node
Version: 22.14.0
Binary:  /home/user/.driftr/tools/node/22.14.0/bin/node
Source:  project config
Project: /home/user/my-project
```

## driftr run

Run a command under a specific Node.js version without changing any persistent settings.

```bash
# Run npm test using Node 24
driftr run --node 24.0.0 -- npm test

# Run a script with a different version
driftr run --node 20.11.0 -- node script.js
```

**Behavior:**
- The global default and project pin are not changed
- The `--` separator is required between flags and the command
- Exit codes are preserved

## driftr setup

Initialize Driftr directories and generate shim scripts.

```bash
driftr setup
```

**What it creates:**
- `~/.driftr/bin/` with shims for `node`, `npm`, `npx`
- `~/.driftr/tools/node/` for installed versions
- `~/.driftr/config/` for global settings
- `~/.driftr/cache/` for downloads

Run this once after installing Driftr, and again after upgrading to regenerate shims.

## Resolution Order

When you run `node` (or `npm`/`npx`), Driftr resolves the version in this order:

| Priority | Source | When |
|----------|--------|------|
| 1 | Explicit `--node` flag | `driftr run --node 24 -- ...` |
| 2 | Project `.driftr.toml` | Found in current or parent directory |
| 3 | `package.json` driftr key | Found in current or parent directory |
| 4 | Global default | Set via `driftr default` |

If no version is configured at any level, Driftr prints an actionable error:

```
no Node.js version configured. Run `driftr install node@<version>` and `driftr default node@<version>`
```

## Typical Workflow

```bash
# One-time setup
driftr setup
echo 'export PATH="$HOME/.driftr/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Install versions you need
driftr install node@22
driftr install node@24

# Set a global fallback
driftr default node@24.0.0

# Pin projects
cd project-a && driftr pin node@22.14.0
cd project-b && driftr pin node@24.0.0

# Just use node normally -- Driftr handles the rest
cd project-a && node -v   # v22.14.0
cd project-b && node -v   # v24.0.0
cd ~         && node -v   # v24.0.0 (global default)
```
