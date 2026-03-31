# Installation

## Building from Source

Driftr is written in Go. You need Go 1.23 or later to build it.

### Prerequisites

- [Go](https://go.dev/dl/) 1.23+
- Git
- macOS or Linux (Windows support is planned)

### Clone and Build

```bash
git clone https://github.com/DriftrLabs/driftr.git
cd driftr
go build -o driftr ./cmd/driftr/
```

### Install the Binary

Move the binary to a directory in your `PATH`:

```bash
sudo mv driftr /usr/local/bin/
```

Or install it directly with Go:

```bash
go install github.com/DriftrLabs/driftr/cmd/driftr@latest
```

## Initial Setup

After installing the binary, run setup to create directories and shim scripts:

```bash
driftr setup
```

This creates the following structure:

```
~/.driftr/
  bin/           shim scripts (node, npm, npx)
  tools/node/    installed Node.js versions
  config/        global configuration
  cache/         downloaded archives
```

## PATH Configuration

Add the shim directory to the **beginning** of your `PATH` so Driftr's shims take priority over any system-installed Node.js.

### Zsh (~/.zshrc)

```bash
export PATH="$HOME/.driftr/bin:$PATH"
```

### Bash (~/.bashrc or ~/.bash_profile)

```bash
export PATH="$HOME/.driftr/bin:$PATH"
```

Then reload your shell:

```bash
source ~/.zshrc   # or source ~/.bashrc
```

### Verify

```bash
which node
# Should output: /Users/<you>/.driftr/bin/node

driftr --help
# Should show the Driftr help menu
```

## Docker

Driftr can also be tested in a Docker container without affecting your local environment.

### Build the image

```bash
docker build -t driftr .
```

### Run commands

```bash
docker run --rm driftr install node@22
docker run --rm driftr list
```

### Run the integration test suite

```bash
docker build -f Dockerfile.test -t driftr-test .
docker run --rm driftr-test
```

## Uninstalling

To remove Driftr:

1. Remove the binary:
   ```bash
   rm /usr/local/bin/driftr
   ```

2. Remove Driftr data:
   ```bash
   rm -rf ~/.driftr
   ```

3. Remove the `PATH` entry from your shell profile.
