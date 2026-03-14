.PHONY: build build-at3am build-hook \
        build-linux build-darwin build-darwin-arm64 build-windows build-all \
        clean test coverage fix vet fmt check help version

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
	@echo "  make build              - Build for the current platform"
	@echo "  make build-linux        - Linux amd64 → dist/linux-amd64/"
	@echo "  make build-darwin       - macOS amd64 → dist/darwin-amd64/"
	@echo "  make build-darwin-arm64 - macOS arm64 (Apple Silicon) → dist/darwin-arm64/"
	@echo "  make build-windows      - Windows amd64 → dist/windows-amd64/"
	@echo "  make build-all          - All platforms"
	@echo "  make test               - Run all tests"
	@echo "  make coverage           - Run tests with coverage report"
	@echo "  make fix                - Run go fix to modernise code"
	@echo "  make vet                - Run go vet"
	@echo "  make fmt                - Check formatting with gofmt"
	@echo "  make check              - fix + vet + fmt"
	@echo "  make clean              - Remove built binaries and dist/"
	@echo "  make version            - Show version info"

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

# ── Cross-platform builds ─────────────────────────────────────────────────────

build-linux:
	@echo "Building Linux amd64 ($(VERSION))..."
	@mkdir -p dist/linux-amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/linux-amd64/at3am      ./cmd/at3am
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/linux-amd64/at3am-hook ./cmd/at3am-hook
	@echo "✓ dist/linux-amd64/"

build-darwin:
	@echo "Building macOS amd64 ($(VERSION))..."
	@mkdir -p dist/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/darwin-amd64/at3am      ./cmd/at3am
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/darwin-amd64/at3am-hook ./cmd/at3am-hook
	@echo "✓ dist/darwin-amd64/"

build-darwin-arm64:
	@echo "Building macOS arm64 ($(VERSION))..."
	@mkdir -p dist/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/darwin-arm64/at3am      ./cmd/at3am
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/darwin-arm64/at3am-hook ./cmd/at3am-hook
	@echo "✓ dist/darwin-arm64/"

build-windows:
	@echo "Building Windows amd64 ($(VERSION))..."
	@mkdir -p dist/windows-amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/windows-amd64/at3am.exe      ./cmd/at3am
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/windows-amd64/at3am-hook.exe ./cmd/at3am-hook
	@echo "✓ dist/windows-amd64/"

build-all: build-linux build-darwin build-darwin-arm64 build-windows
	@echo "✓ All platform builds complete (version: $(VERSION))"

# ── Quality targets ───────────────────────────────────────────────────────────

fix:
	@echo "Running go fix..."
	go fix ./...

vet:
	@echo "Running go vet..."
	go vet ./...

fmt:
	@echo "Checking code format..."
	gofmt -l .
	@echo "Run 'gofmt -w .' to fix formatting"

check: fix vet fmt
	@echo "✓ check complete"

# ── Test targets ──────────────────────────────────────────────────────────────

test:
	@echo "Running tests..."
	go test -count=1 -timeout 120s ./...

coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out -count=1 -timeout 120s ./...
	go tool cover -func=coverage.out | tail -1
	@echo "Full report: go tool cover -html=coverage.out"

# ── Cleanup ───────────────────────────────────────────────────────────────────

clean:
	@echo "Cleaning..."
	rm -rf bin/ dist/ coverage.out
	go clean

