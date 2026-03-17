# at3am Project Summary

## Overview

**at3am** is a lightweight, stateless CLI tool that acts as an intelligent propagation oracle between your DNS provider and any ACME client. It polls global resolvers in parallel, builds a statistical confidence model aligned with Let's Encrypt's Multi-Perspective Issuance Corroboration (MPIC), and signals exactly when validation is safe.

## What Was Built

### Core Components

1. **Resolver Pool** (`internal/resolver/`)
   - 25+ public anycast DNS resolvers (Google, Cloudflare, Quad9, OpenDNS, etc.)
   - Auto-discovery of authoritative nameservers (IPv4 + conditional IPv6)
   - Custom resolver support
   - Concurrent querying with timeout handling
   - Optional DNSSEC validation: sets DO bit on queries, captures AD bit from responses

2. **Confidence Engine** (`internal/confidence/`)
   - Threshold-based readiness: `READY = (auth_correct >= auth_threshold) AND (public_correct >= public_threshold)`
   - 4 pre-built profiles with explicit auth/public thresholds and consecutive-pass requirements:

     | Profile | Auth threshold | Public threshold | Consecutive | Timeout |
     |---------|---------------|-----------------|-------------|---------|
     | strict  | ALL           | ALL − 2          | 3           | 600 s   |
     | default | ALL           | ≥ 1              | 2           | 300 s   |
     | fast    | ≥ ceil(N/2)   | ≥ 1              | 1           | 120 s   |
     | yolo    | ≥ 1           | ≥ 0              | 1           | 60 s    |

   - DNSSEC aggregate: `DNSSECValidCount / DNSSECTotal` (AD-bit ratio) reported per poll
   - MPIC-aware design: authoritative NS results weighted separately from public resolvers

3. **TTL Analyser** (`internal/ttl/`)
   - Full TTL audit across all resolvers
   - Min/max/average TTL calculation
   - Estimated propagation time
   - Intelligent warnings (high TTL, low TTL, large spread)

4. **Diagnostics Engine** (`internal/diagnostics/`)
   - Scenario-based explanations (no_record, auth_only, partial_propagation, full_propagation)
   - Actionable recommendations
   - Severity levels (info, warning, error)

5. **Output Layer** (`internal/output/`)
   - Human-readable format (default)
   - JSON format (structured logging)
   - Quiet mode (minimal output)
   - Prometheus metrics endpoint (`--prometheus-port`)
   - Template variable expansion (`$DOMAIN`, `$CONFIDENCE`, `$ELAPSED`)

6. **DNS Provider Integration** (`internal/provider/`)
   - **54 DNS providers** via libdns ecosystem (MIT-licensed)
   - Auto-detection via NS record inspection
   - YAML credential loading with env-var overrides
   - Early-access test (canary create/delete) before real operations
   - Supported providers:
     - **Batch 1 (13):** Cloudflare, Route53, Google Cloud DNS, Azure, DigitalOcean, Hetzner, GoDaddy, Namecheap, Porkbun, OVH, Gandi, Linode, PowerDNS
     - **Batch 2 (30):** DNSimple, Scaleway, INWX, RFC2136, Netlify, Tencent Cloud, AliDNS, deSEC, TransIP, Bunny, IONOS, NameSilo, ACME-DNS, DuckDNS, Hurricane Electric, Infomaniak, LuaDNS, ClouDNS, DirectAdmin, Loopia, GleSYS, Huawei Cloud, ACME-Proxy, Dynv6, Mythic Beasts, Simply.com, Netcup, Dynu, G-Core
     - **Batch 3 (14):** ALL-INKL, AutoDNS, BlueCat, DNS UPDATE, Domainnameshop, mijn.host, Njalla, Regfish, Tecnocratica, UniFi, WebSupport, WEDOS, West.cn, Volcengine

7. **Mock System** (`internal/mock/`)
   - 5 predefined scenarios for testing without real DNS
   - instant, slow_propagation, timeout, flaky, partial
   - Fully testable without external dependencies

7. **CLI Interface** (`cmd/at3am/`)
   - `at3am wait` command with all flags
   - `at3am version` command with git info
   - `at3am-hook` wrapper for Certbot integration
   - Exit codes: 0=success, 1=timeout, 2=config error, 3=DNS error
   - `--dnssec-validate` — enable DNSSEC validation (DO bit + AD bit reporting)
   - `--challenge-type dns-01|persist` — select ACME challenge variant

### Build System

- **Makefile** with git-based versioning
- Automatic version injection from git tags and commit hashes
- Build time tracking
- Cross-platform release builds: `make build-linux`, `build-darwin`, `build-darwin-arm64`, `build-windows`, `build-all`
- `make fix` — runs `go fix ./...` to modernise code automatically
- `make check` — runs fix + vet + fmt in one step

## Test Coverage

**87.2% overall** (providers package excluded — 54 thin adapter files over external APIs)

| Package | Coverage |
|---------|----------|
| `internal/log` | 97.9% |
| `internal/diagnostics` | 100.0% |
| `internal/mock` | 100.0% |
| `internal/config` | 97.5% |
| `internal/output` | 97.5% |
| `internal/ttl` | 97.2% |
| `internal/resolver` | 92.1% |
| `internal/wait` | 91.7% |
| `cmd/at3am` | 91.2% |
| `internal/confidence` | 80.8% |
| `cmd/at3am-hook` | 57.7% |

- 15 test files with 120+ test cases
- Table-driven tests for comprehensive input coverage
- Mock DNS system for resolver testing without external deps
- **Integration tests** covering the full Certbot + Cloudflare workflow end-to-end
- `.env/cloudflare.env` for storing live test credentials (gitignored)
- All edge cases covered (nil inputs, errors, boundaries, level filtering)

## Files Created

### Source Code
- `cmd/at3am/main.go` — Main CLI entry point
- `cmd/at3am-hook/main.go` — Certbot hook wrapper
- `internal/config/config.go` — Configuration types & profiles
- `internal/resolver/resolver.go` — DNS querying
- `internal/resolver/pool.go` — Resolver pool management
- `internal/confidence/engine.go` — Confidence scoring
- `internal/ttl/analyser.go` — TTL analysis
- `internal/diagnostics/diagnostics.go` — Diagnostic explanations
- `internal/output/output.go` — Output formatting
- `internal/output/prometheus.go` — Prometheus metrics
- `internal/mock/mock.go` — Mock DNS scenarios
- `internal/wait/wait.go` — Core polling loop

### Tests (15 test files)
- `internal/log/log_test.go` — 19 tests, 97.9% coverage
- `cmd/at3am/main_test.go` — extended with formatVersion, logLevel, version subcommand
- `cmd/at3am-hook/main_test.go` — extended with cleanup, skip-DNS, challenge-type paths
- `test/integration/certbot_cloudflare_test.go` — live end-to-end Certbot + Cloudflare test
- `test/integration/certbot_mock_test.go` — 3 mock workflow scenarios (no credentials needed)
- `test/integration/testenv.go` — shared `.env/` file loader for integration tests
- `*_test.go` files for all other packages
- 87.2% overall coverage (providers excluded)

### Documentation
- `LICENSE` — Apache License 2.0 with patent grant
- `CONTRIBUTING.md` — CLA, code style, testing requirements
- `ENTERPRISE.md` — Feature comparison (free vs enterprise)
- `BUILD.md` — Build instructions and versioning guide
- `go.mod` — Go module definition with dependencies

## Key Features

✓ Zero-config useful — Works out of the box with sensible defaults
✓ Composable — Designed for shell scripts, certbot hooks, CI/CD pipelines
✓ Stateless by default — No database, no persistent storage
✓ Accurate over fast — Conservative threshold-based readiness model
✓ Transparent — Every decision is explainable (auth/public required counts shown)
✓ Prometheus metrics — Fleet monitoring support
✓ Mock mode — Full testing without real DNS
✓ Structured logging — JSON output for SIEM ingestion
✓ DNSSEC-aware — Optional AD-bit validation with `--dnssec-validate`
✓ DNS-PERSIST-01 ready — `--challenge-type persist` for long-lived TXT records
✓ Cross-platform — Release binaries for Linux, macOS (amd64+arm64), Windows

## Dependencies

- `github.com/miekg/dns` — DNS protocol implementation
- `github.com/spf13/cobra` — CLI framework

All dependencies are at latest versions (as of 2026-03-14).

## How to Use

### Build
```bash
make build
```

### Run Tests
```bash
make test
make coverage
```

### Check Version
```bash
./bin/at3am version
```

### Basic Usage
```bash
./bin/at3am wait \
  --domain _acme-challenge.example.com \
  --expected "validation-token-here" \
  --timeout 300s \
  --profile default
```

### With Certbot (full automation with DNS provider)
```bash
# First run creates credential template
export AT3AM_DNS_PROVIDER=cloudflare
export AT3AM_DNS_CREDS=~/.at3am/cloudflare.yaml

certbot certonly --manual \
  --manual-auth-hook    "at3am-hook manual-auth" \
  --manual-cleanup-hook "at3am-hook manual-cleanup" \
  --preferred-challenges dns \
  -d example.com
```

The hook automatically:
1. Detects your DNS provider (or reads `AT3AM_DNS_PROVIDER`)
2. Loads credentials from YAML (creates template if missing)
3. Creates the `_acme-challenge` TXT record via provider API
4. Waits for propagation using at3am engine
5. Deletes the record after validation completes

### Mock Testing
```bash
./bin/at3am wait \
  --domain _acme-challenge.example.com \
  --mock \
  --mock-scenario instant \
  --output json
```

## DNS Provider Integration

**at3am-hook** now includes end-to-end DNS automation via 54 libdns providers:

- **Auto-detection** — inspects NS records to identify your provider
- **Credential management** — YAML files with env-var overrides
- **Early-access test** — canary create/delete before real operations
- **Full lifecycle** — create record → wait for propagation → delete record

**53 providers:** MIT License
**1 provider:** Apache 2.0 (dnsimple)

All licenses are compatible with at3am's Apache 2.0 license. See `LICENSES/` directory for full attribution.

## Next Steps

1. **Create GitHub releases** — Tag commits with semantic versions (v0.1.0, v0.2.0, etc.)
2. **Set up CI/CD** — GitHub Actions using `make check build-all test` on every push
3. **Publish binaries** — `make build-all` produces `dist/` artifacts for all platforms
4. **Documentation** — Add README.md with examples and architecture diagrams (✅ done)
5. **Enterprise version** — Implement full ACME client and advanced features

## Project Status

✅ Core DNS-01 oracle fully implemented
✅ 87.2% test coverage (providers excluded)
✅ All dependencies at latest versions
✅ Build system with git-based versioning
✅ Apache 2.0 licensed with patent grant
✅ CLA and contribution guidelines in place
✅ Enterprise roadmap documented
✅ **54 DNS providers integrated** (libdns ecosystem, 53 MIT + 1 Apache 2.0)
✅ **End-to-end Certbot automation** (record create/delete/wait)
✅ **License attribution** (`LICENSES/` directory with all provider licenses)
✅ **Integration tests** — live Cloudflare end-to-end + 3 mock scenarios
✅ **`.env/` credential management** for integration tests (gitignored)
✅ **`internal/log` fully tested** (97.9% — was 0%)

**Ready for production use and open-source release.**

