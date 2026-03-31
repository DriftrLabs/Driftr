# Configuration

Driftr uses TOML configuration files at two levels: global and per-project.

## Global Configuration

**Location:** `~/.driftr/config/config.toml`

This file stores your global default Node.js version. It is created and managed by `driftr default`.

### Format

```toml
[default]
node = "22.14.0"
```

### Fields

| Section | Key | Type | Description |
|---------|-----|------|-------------|
| `[default]` | `node` | string | Global default Node.js version |

### Example

After running:

```bash
driftr default node@22.14.0
```

The config file will contain:

```toml
[default]
node = "22.14.0"
```

## Project Configuration

**Location:** `.driftr.toml` in the project root

This file pins tool versions for a specific project. It is created and managed by `driftr pin`.

### Format

```toml
[tools]
node = "22.14.0"
```

### Fields

| Section | Key | Type | Description |
|---------|-----|------|-------------|
| `[tools]` | `node` | string | Pinned Node.js version for this project |

### Example

After running:

```bash
cd my-project
driftr pin node@22.14.0
```

A `.driftr.toml` file is created in the current directory:

```toml
[tools]
node = "22.14.0"
```

### Directory Walk Behavior

When resolving a version, Driftr searches for `.driftr.toml` starting from the current working directory and walking up to the filesystem root.

```
/home/user/my-project/packages/core/   <- cwd, no .driftr.toml
/home/user/my-project/packages/        <- no .driftr.toml
/home/user/my-project/                 <- .driftr.toml found! uses this
```

This means:
- You only need one `.driftr.toml` at the project root
- All subdirectories inherit the pinned version
- A nested `.driftr.toml` overrides the parent

### Version Control

The `.driftr.toml` file **should be committed** to version control. This ensures all team members use the same Node.js version for the project.

Add it to your project:

```bash
git add .driftr.toml
git commit -m "Pin Node.js version with Driftr"
```

## Storage Layout

Driftr stores all data under `~/.driftr/`:

```
~/.driftr/
  bin/                        shim scripts
    node                      shell script -> driftr shim node
    npm                       shell script -> driftr shim npm
    npx                       shell script -> driftr shim npx
  tools/
    node/
      22.14.0/                extracted Node.js installation
        bin/
          node
          npm
          npx
        lib/
        include/
      24.0.0/
  config/
    config.toml               global configuration
  cache/
    node-v22.14.0-*.tar.gz    cached downloads
```

### Cache

Downloaded archives are cached in `~/.driftr/cache/`. Subsequent installs of the same version skip the download. To force a re-download, delete the cached archive:

```bash
rm ~/.driftr/cache/node-v22.14.0-*.tar.gz
driftr install node@22.14.0
```

## Future Compatibility

The configuration format is designed for extension. Future versions may add:

- `.nvmrc` and `.node-version` file support as alternative resolution sources
- Package manager pinning (`pnpm`, `yarn`) in `.driftr.toml`
- Mirror configuration for custom download sources
- `package.json` `engines` field support

```toml
# Future .driftr.toml (not yet supported)
[tools]
node = "22.14.0"
pnpm = "10.0.0"
```
