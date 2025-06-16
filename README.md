## Introduction

**WDMT** /ˌwɪdiˈɛmˈtiː/ (Web Developer Maintenance Tool) is a CLI that securely removes bulky development artefacts such as `node_modules`, `.next`, or `dist`. It focuses on correctness and safety, ensuring that no file outside the intended directory tree is ever touched.

### Motivation

Have you ever seen 200GB+ taken by the System Data on your Mac?

Large dependency folders and build outputs quickly bloat local check-outs.  Manually pruning them with `rm -rf` or similar commands is risky-one wrong path or an unexpected symlink can wipe unrelated data.

**WDMT** gives developers a fast and interactive way to reclaim disk space without compromising security.

### Key Features

- **🔒 Symlink Attack Prevention** — Never follows malicious symlinks  
- **🛡️ Path Traversal Protection** — Blocks `../../../etc/passwd` style attacks  
- **⚡ Interactive Selection** — Smart path display with multiple view modes  
- **📊 Progress Visualisation** — Beautiful progress bars powered by Charmbracelet  
- **🎯 Secure by Design** — Robust security validation  
- **📱 Cross-Platform** — Works on macOS, Linux, and Windows  
- **🔍 Enhanced Path Display** — Smart, condensed, and full path viewing modes with keyboard shortcuts  

### Quick Start

#### Installation

```bash
go install github.com/neg4n/wdmt@latest
```

or

```bash
curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/neg4n/wdmt/main/install.sh | sh
```

#### Usage

1. Launch your preferred terminal emulator
2. Navigate to the directory you want to clean up _(e.g. `~/projects` if you keep all your programming projects there)_
3. Run `wdmt` in the terminal
4. Follow the interactive prompts to select and delete directories

### Security Architecture

| Feature | Traditional tools | WDMT |
|---------|------------------|------|
| Symlink safety | Often follows symlinks | Never follows symlinks |
| Path validation | Basic checks | Multi-layer validation |
| Injection protection | None | UTF-8 & null-byte filtering |
| Race-condition defence | Vulnerable | Just-in-time validation |
| Filesystem boundaries | Can cross devices | Device-ID tracking |

> **Note**  
> Utilities such as `rimraf` rely on Node.js filesystem calls that may follow symlinks if the caller is not careful. WDMT performs explicit checks on every path component and refuses to delete if any part resolves outside the intended directory.

### Supported Targets

WDMT detects and cleans the following artefacts:

| Type | Directories |
|------|-------------|
| Node.js | `node_modules` |
| Build output | `dist`, `build`, `.output` |
| Framework cache | `.next`, `.nuxt`, `.vite`, `.turbo` |
| Test coverage | `coverage`, `.nyc_output`, `lib-cov` |
| Bundler cache | `.parcel-cache`, `.webpack`, `.rollup.cache` |
| Temporary | `tmp`, `temp`, `.cache` |
| System files | `.DS_Store`, `Thumbs.db` |

### Development

#### Running Tests

```bash
# All tests
go test ./...

# Security tests with coverage
go test ./internal/cleaner -v -cover
```

> [!TIP]  
> All security tests run in isolated temporary directories to avoid touching real data.

### Comparison

| Tool | Security | Interactive | Cross-Platform |
|------|----------|-------------|----------------|
| **WDMT** | ✅ Robust | ✅ Modern UI | ✅ Go-based |
| `rimraf` | ❌ Basic | ❌ No | ✅ Node.js |
| `rm -rf` | ❌ Dangerous | ❌ No | ⚠️ Unix only |
| `npkill` | ⚠️ Limited | ✅ Yes | ✅ Node.js |

## License

The MIT License