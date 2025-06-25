## Introduction

**WDMT** /ˌwɪdiˈɛmˈtiː/ (Web Developer Maintenance Tool) is a CLI that securely removes bulky development artefacts such as `node_modules`, `.next`, or `dist`. It focuses on correctness and safety, ensuring that no file outside the intended directory tree is ever touched.

## Preview



https://github.com/user-attachments/assets/66d4fe38-4503-48c8-b540-a494b653bcfd




### Motivation

Have you ever seen 200GB+ taken by the System Data on your Mac?

Large dependency folders and build outputs quickly bloat local check-outs.  Manually pruning them with `rm -rf` or similar commands is risky-one wrong path or an unexpected symlink can wipe unrelated data.

**WDMT** gives developers a fast and interactive way to reclaim disk space without compromising security.

### Key Features

- **🔒 Symlink Attack Prevention** — Never follows malicious symlinks  
- **🛡️ Path Traversal Protection** — Blocks `../../../etc/passwd` style attacks  
- **⚡ Interactive Selection** — Smart path display with multiple view modes  
- **📊 Progress Visualisation** — Beautiful progress bars created using [charmbracelet](charm.sh) libraries  
- **🎯 Secure by Design** — Robust security validation  
- **📱 Cross-Platform** — Works on macOS, Linux, and Windows  
- **🔍 Enhanced Path Display** — Smart, condensed, and full path viewing modes with keyboard shortcuts  
- **📏 Accurate Size Calculation** — Block-based size estimation (4KB blocks) matches actual disk usage

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

WDMT uses a **two-phase security model** optimized for both performance and safety:

#### **Discovery Phase (Scanner)**
- **🔍 Fast & Minimal Security** — Essential symlink detection for safe directory traversal
- **⚡ Performance Optimized** — Lightweight validation enables sub-second scanning
- **👀 User Review Required** — Always displays confirmation screen before deletion

#### **Deletion Phase (Cleaner)**
- **🛡️ Full Security Suite** — Complete protection when it matters most

| Feature | Traditional tools | WDMT Scanner | WDMT Cleaner |
|---------|------------------|--------------|--------------|
| Symlink safety | Often follows symlinks | Essential detection | Never follows symlinks |
| Path validation | Basic checks | Basic boundary checks | Multi-layer validation |
| Injection protection | None | N/A (user review) | UTF-8 & null-byte filtering |
| Race-condition defence | Vulnerable | N/A (user review) | Just-in-time validation |
| Filesystem boundaries | Can cross devices | N/A (user review) | Device-ID tracking |

> [!IMPORTANT]  
> The scanner prioritizes speed for discovery, while the cleaner enforces complete security during deletion. **Always review the confirmation screen** to verify what you're deleting, as this is your primary defense against accidental deletions.

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
