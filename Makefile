.PHONY: build build-at3am build-hook \
        build-linux build-linux-arm64 build-linux-386 build-linux-arm \
        build-darwin build-darwin-arm64 \
        build-windows build-windows-arm64 \
        build-all clean test coverage fix vet fmt check help version

# Version information from git
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Build flags
LDFLAGS := -ldflags "\
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildTime=$(BUILD_TIME)"

help:
	@echo "at3am build targets:"
	@echo "  make build                - Build for the current platform"
	@echo "  make build-linux          - Linux amd64 → dist/linux-amd64/"
	@echo "  make build-linux-arm64    - Linux arm64 → dist/linux-arm64/"
	@echo "  make build-linux-386      - Linux 386 → dist/linux-386/"
	@echo "  make build-linux-arm      - Linux arm → dist/linux-arm/"
	@echo "  make build-darwin         - macOS amd64 → dist/darwin-amd64/"
	@echo "  make build-darwin-arm64   - macOS arm64 → dist/darwin-arm64/"
	@echo "  make build-windows        - Windows amd64 → dist/windows-amd64/"
	@echo "  make build-windows-arm64  - Windows arm64 → dist/windows-arm64/"
	@echo "  make build-all            - Build all targets"
	@echo "  make clean                - Remove built binaries and dist/"

version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Build:   $(BUILD_TIME)"

# ── Current-platform build ────────────────────────────────────────────────────

build: build-at3am build-hook
	@echo "✓ Build complete (version: $(VERSION))"

build-at3am:
	@echo "Building at3am ($(VERSION))..."
	go build $(LDFLAGS) -o bin/at3am ./cmd/at3am

build-hook:
	@echo "Building at3am-hook ($(VERSION))..."
	go build $(LDFLAGS) -o bin/at3am-hook ./cmd/at3am-hook

# ── Linux builds ──────────────────────────────────────────────────────────────

build-linux:
	@echo "Building Linux amd64..."
	@mkdir -p dist/linux-amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/linux-amd64/at3am      ./cmd/at3am
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/linux-amd64/at3am-hook ./cmd/at3am-hook

build-linux-arm64:
	@echo "Building Linux arm64..."
	@mkdir -p dist/linux-arm64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/linux-arm64/at3am      ./cmd/at3am
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/linux-arm64/at3am-hook ./cmd/at3am-hook

build-linux-386:
	@echo "Building Linux 386..."
	@mkdir -p dist/linux-386
	GOOS=linux GOARCH=386 go build $(LDFLAGS) -o dist/linux-386/at3am      ./cmd/at3am
	GOOS=linux GOARCH=386 go build $(LDFLAGS) -o dist/linux-386/at3am-hook ./cmd/at3am-hook

build-linux-arm:
	@echo "Building Linux arm..."
	@mkdir -p dist/linux-arm
	GOOS=linux GOARCH=arm go build $(LDFLAGS) -o dist/linux-arm/at3am      ./cmd/at3am
	GOOS=linux GOARCH=arm go build $(LDFLAGS) -o dist/linux-arm/at3am-hook ./cmd/at3am-hook

# ── macOS builds ──────────────────────────────────────────────────────────────

build-darwin:
	@echo "Building macOS amd64..."
	@mkdir -p dist/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/darwin-amd64/at3am      ./cmd/at3am
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/darwin-amd64/at3am-hook ./cmd/at3am-hook

build-darwin-arm64:
	@echo "Building macOS arm64..."
	@mkdir -p dist/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/darwin-arm64/at3am      ./cmd/at3am
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/darwin-arm64/at3am-hook ./cmd/at3am-hook

# ── Windows builds ─────────────────────────────────────────────────────────────

build-windows:
	@echo "Building Windows amd64..."
	@mkdir -p dist/windows-amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/windows-amd64/at3am.exe      ./cmd/at3am
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/windows-amd64/at3am-hook.exe ./cmd/at3am-hook

build-windows-arm64:
	@echo "Building Windows arm64..."
	@mkdir -p dist/windows-arm64
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/windows-arm64/at3am.exe      ./cmd/at3am
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/windows-arm64/at3am-hook.exe ./cmd/at3am-hook

# ── Build everything ───────────────────────────────────────────────────────────

build-all: \
	build-linux \
	build-linux-arm64 \
	build-linux-386 \
	build-linux-arm \
	build-darwin \
	build-darwin-arm64 \
	build-windows \
	build-windows-arm64

	@echo "✓ All platform builds complete (version: $(VERSION))"