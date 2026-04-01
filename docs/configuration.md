# Configuration

Driftr uses configuration at two levels: global (`config.toml`) and per-project (`.driftr.toml` or `package.json`).

## Global Configuration

**Location:** `~/.driftr/config/config.toml`

This file stores your global default tool versions. It is created and managed by `driftr default`.

### Format

```toml
[default]
node = "22.14.0"

[default.tools]
pnpm = "9.15.0"
yarn = "1.22.22"
```

### Fields

| Section           | Key    | Type   | Description                     |
|-------------------|--------|--------|---------------------------------|
| `[default]`       | `node` | string | Global default Node.js version  |
| `[default.tools]` | `pnpm` | string | Global default pnpm version     |
| `[default.tools]` | `yarn` | string | Global default yarn version     |

### Example

After running:

```bash
driftr default node@22.14.0
driftr default pnpm@9.15.0
```

The config file will contain both tool defaults.

## Project Configuration

Driftr supports two project config formats. On first `driftr pin`, you choose which to use. The choice is auto-detected on subsequent runs.

### Option 1: `.driftr.toml` (recommended)

**Location:** `.driftr.toml` in the project root

```toml
[tools]
node = "22.14.0"
pnpm = "9.15.0"
```

| Section   | Key    | Type   | Description                                |
|-----------|--------|--------|--------------------------------------------|
| `[tools]` | `node` | string | Pinned Node.js version for this project    |
| `[tools]` | `pnpm` | string | Pinned pnpm version for this project       |
| `[tools]` | `yarn` | string | Pinned yarn version for this project       |

### Option 2: `package.json`

**Location:** `driftr` key in an existing `package.json`

```json
{
  "name": "my-project",
  "driftr": {
    "node": "22.14.0"
  }
}
```

| Key            | Type   | Description                              |
|----------------|--------|------------------------------------------|
| `driftr.node`  | string | Pinned Node.js version for this project  |

This format is useful when you want to keep all project tooling config in `package.json` without an extra dotfile.

**Note:** The `package.json` format currently only supports `node`. For pnpm and yarn pinning, use `.driftr.toml`.

**Note:** `package.json` must already exist — Driftr will not create it. Run `npm init` first if needed.

### Migrating Between Formats

```bash
# Switch from current format to the other
driftr pin node@22.14.0 --migrate
```

This writes the version in the new format and removes the old config (deletes `.driftr.toml` or removes the `driftr` key from `package.json`).

### Directory Walk Behavior

When resolving a version, Driftr walks up from the current directory to the filesystem root. In each directory, it checks `.driftr.toml` first, then `package.json`:

```
/home/user/my-project/packages/core/   <- cwd, no config
/home/user/my-project/packages/        <- no config
/home/user/my-project/                 <- .driftr.toml found! uses this
```

If `.driftr.toml` and `package.json` both exist in the same directory, `.driftr.toml` takes priority.

This means:

- You only need one config at the project root
- All subdirectories inherit the pinned version
- A nested config overrides the parent

### Version Control

Your project config (`.driftr.toml` or `package.json`) **should be committed** to version control. This ensures all team members use the same tool versions.

```bash
git add .driftr.toml   # or package.json
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
    pnpm                      shell script -> driftr shim pnpm
    pnpx                      shell script -> driftr shim pnpx
    yarn                      shell script -> driftr shim yarn
  tools/
    node/
      22.14.0/
        bin/node, npm, npx
    pnpm/
      9.15.0/
        bin/pnpm, pnpx (symlink)
    yarn/
      1.22.22/
        bin/yarn.js
        lib/
        package.json
  config/
    config.toml               global configuration
  cache/
    node-v22.14.0-*.tar.gz    cached Node.js archives
    pnpm-9.15.0-*             cached pnpm binaries
    yarn-1.22.22.tgz          cached yarn tarballs
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
- pnpm and yarn pinning in `package.json` format (currently `.driftr.toml` only)
- Mirror configuration for custom download sources
