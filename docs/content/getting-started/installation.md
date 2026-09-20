---
title: Install sombra
weight: 1
aliases:
  - /user-guide/installation/
---

## Installation Options

You can install `sombra` using one of the following methods:

---

### Option 1: Install via Go

If you have Go 1.26 or later, you can use:

```bash
go install github.com/yunier-rojas/sombra-cli/cmd/sombra@latest
```

> This places the `sombra` binary in `$(go env GOPATH)/bin`.
> Make sure that directory is in your system `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

---

### Option 2: Download Prebuilt Binaries

1. Go to the [GitHub Releases](https://github.com/yunier-rojas/sombra-cli/releases).

2. Download the binary for your platform:

	* `sombra-linux-amd64`
	* `sombra-darwin-arm64`
	* `sombra-windows-amd64.exe`

3. Make it executable (Linux/macOS):

```bash
chmod +x sombra-*
mv sombra-* /usr/local/bin/sombra
```

4. Confirm installation:

```bash
sombra --help
```

---

### Option 3: Build from Source

If you prefer to build manually:

```bash
git clone https://github.com/yunier-rojas/sombra-cli.git
cd sombra-cli
go mod tidy
make build WHAT=sombra
```

Output is saved in the `build/` directory.

---

## Requirements

* Go **1.26+**
* A shell or terminal (Linux, macOS, or Windows PowerShell)

---

Need help? Open an [issue](https://github.com/yunier-rojas/sombra-cli/issues) or [contact me](../contact.md).
