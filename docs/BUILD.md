# Building at3am

## Quick Start

```bash
# Build both binaries
make build

# Run tests
make test

# Check coverage
make coverage

# Clean build artifacts
make clean
```

## Version Information

The build system automatically injects version information from git tags and commit hashes at build time.

### How It Works

The Makefile uses `git describe` to extract version information:

```bash
VERSION = $(shell git describe --tags --always --dirty)
COMMIT  = $(shell git rev-parse --short HEAD)
BUILD_TIME = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
```

These are passed to the Go compiler via `-ldflags`:

```bash
go build -ldflags "-X main.version=v0.1.0 -X main.commit=abc1234 -X main.buildTime=2025-03-14T12:00:00Z"
```

### Version Output

Check the version of a built binary:

```bash
$ ./bin/at3am version
at3am v0.1.0
Commit:  abc1234
Built:   2025-03-14T12:00:00Z
```

### Git Tagging

To create a release, tag the commit:

```bash
git tag v0.1.0
git push origin v0.1.0
make build
```

The binary will automatically include:
- **Version**: `v0.1.0` (from the tag)
- **Commit**: Short commit hash
- **Build Time**: ISO 8601 timestamp

### Development Builds

When building without a git tag, the version defaults to:
- **Version**: `dev` (or the latest tag + commits since)
- **Commit**: `unknown` (if not in a git repo)
- **Build Time**: Current UTC time

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build both at3am and at3am-hook |
| `make build-at3am` | Build at3am only |
| `make build-hook` | Build at3am-hook only |
| `make test` | Run all tests |
| `make coverage` | Run tests with coverage report |
| `make clean` | Remove bin/ and coverage.out |
| `make vet` | Run go vet |
| `make fmt` | Check code formatting |
| `make version` | Show version info from git |
| `make help` | Show this help |

## Cross-Compilation

To build for a different OS/architecture:

```bash
GOOS=linux GOARCH=amd64 make build-at3am
GOOS=darwin GOARCH=arm64 make build-at3am
GOOS=windows GOARCH=amd64 make build-at3am
```

## CI/CD Integration

The Makefile is designed to work with CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Build
  run: make build

- name: Test
  run: make test

- name: Coverage
  run: make coverage
```

The version is automatically set from git tags, so releases are straightforward:

```yaml
- name: Build Release
  if: startsWith(github.ref, 'refs/tags/')
  run: make build
```

