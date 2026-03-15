# Testing at3am

This guide covers all testing approaches for at3am: unit tests, integration tests, and propagation metrics.

## Quick Start

```bash
# Run all unit tests (fast, no network)
make test

# Run with coverage report
make coverage

# Run mock integration tests (no credentials needed)
go test -timeout 5m ./test/integration/ -v -run TestCertbotMock
```

## Unit Tests

Fast, isolated tests with no external dependencies.

```bash
# Run all unit tests
make test

# Run specific package
go test ./internal/confidence/ -v

# Run with coverage
make coverage

# Run with race detector
go test -race ./...

# Run specific test
go test ./cmd/at3am/ -v -run TestVersionCommand
```

### Coverage

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

**Test files:** 15 test files with 120+ test cases
- Table-driven tests for comprehensive input coverage
- Mock DNS system for resolver testing without external deps
- All edge cases covered (nil inputs, errors, boundaries, level filtering)

## Integration Tests

### Mock Tests (No Credentials Required)

Test the full Certbot workflow with simulated DNS. These are fast and deterministic.

```bash
# All mock integration tests
go test -timeout 5m ./test/integration/ -v

# Instant propagation scenario
go test -timeout 5m ./test/integration/ -v -run TestCertbotMock

# Slow propagation scenario
go test -timeout 5m ./test/integration/ -v -run TestCertbotMockSlowPropagation

# Multiple domains (renewal workflow)
go test -timeout 5m ./test/integration/ -v -run TestCertbotMockMultipleDomains
```

**What they test:**
- ✅ manual-auth hook (record creation)
- ✅ Propagation wait with mock resolver
- ✅ manual-cleanup hook (record deletion)
- ✅ Slow propagation handling
- ✅ Multiple domain renewal workflow

### Real Cloudflare Integration Test

Tests against real Cloudflare DNS with actual propagation.

**Requirements:**
- Cloudflare API token (with `Zone.DNS.Edit` permission)
- Cloudflare zone ID
- Test domain

**Setup credentials:**

Option 1: `.env/cloudflare.env` file (recommended for local testing):
```bash
mkdir -p .env
cat > .env/cloudflare.env <<'EOF'
CF_API_TOKEN=your-cloudflare-api-token
CF_ZONE_ID=your-zone-id
TEST_DOMAIN=yourdomain.com
EOF
chmod 600 .env/cloudflare.env
```

Option 2: Environment variables (recommended for CI/CD):
```bash
export CF_API_TOKEN="your-token"
export CF_ZONE_ID="your-zone-id"
export TEST_DOMAIN="example.com"
```

**Get credentials:**
```bash
# 1. Create API token at https://dash.cloudflare.com/profile/api-tokens
#    Minimum permissions: Zone.DNS.Edit
# 2. Get zone ID
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://api.cloudflare.com/client/v4/zones?name=example.com | jq '.result[0].id'
```

**Run the test:**
```bash
go test -timeout 10m ./test/integration/ -v -run TestCertbotCloudflare
```

> **Security:** `.env/` is listed in `.gitignore` and is never committed.
> Environment variables take precedence over file values and are suitable for CI/CD secrets.

**What it tests:**
- ✅ Cloudflare API record creation
- ✅ Cloudflare API record deletion
- ✅ DNS propagation detection with real resolvers
- ✅ at3am scoring engine with real global DNS
- ✅ Full Certbot workflow simulation

### Real Propagation Metrics Test

Measures actual DNS propagation metrics across global resolvers. Generates detailed reports.

**Requirements:** Same as Cloudflare test (CF_API_TOKEN, CF_ZONE_ID)

**Run the test:**
```bash
export CF_API_TOKEN="your-token"
export CF_ZONE_ID="your-zone-id"
export AT3AM_INTEGRATION=1

go test -timeout 30m -v ./test/integration/ -run TestPropagation
```

**What it measures:**
- Time to first resolver seeing the record
- Per-resolver propagation timeline
- TTL handling (60s, auto, etc.)
- Subdomain propagation patterns
- Propagation metrics across different scenarios

## Test Reference

| Test Type | Command | Time | Credentials |
|-----------|---------|------|-------------|
| Unit tests | `make test` | ~50s | None |
| Mock (instant) | `go test -timeout 5m ./test/integration/ -v -run TestCertbotMock` | ~2s | None |
| Mock (slow) | `go test -timeout 5m ./test/integration/ -v -run TestCertbotMockSlowPropagation` | ~30s | None |
| Mock (renewal) | `go test -timeout 5m ./test/integration/ -v -run TestCertbotMockMultipleDomains` | ~10s | None |
| Cloudflare | `go test -tags=integration -timeout 10m ./test/integration/ -v -run TestCertbotCloudflare` | ~45s | CF_API_TOKEN, CF_ZONE_ID |
| Propagation metrics | `AT3AM_INTEGRATION=1 go test -timeout 30m ./test/integration/ -run TestPropagation` | ~30m | CF_API_TOKEN, CF_ZONE_ID |

## Troubleshooting

### Test times out
- Increase `-timeout` flag (e.g., `-timeout 15m`)
- Check network connectivity
- Verify Cloudflare API token is valid

### Cloudflare API errors
- Verify `CF_API_TOKEN` has `Zone.DNS.Edit` permission
- Verify `CF_ZONE_ID` is correct for your domain
- Check Cloudflare API status: https://www.cloudflarestatus.com/

### Mock test fails
- Ensure mock scenarios are defined in `internal/mock/`
- Check that resolver pool is initialized correctly

## CI/CD Integration

### GitHub Actions (mock tests only)
```yaml
- name: Run integration tests
  run: go test -timeout 5m ./test/integration/ -v
```

### GitHub Actions (with Cloudflare)
```yaml
- name: Run Cloudflare integration tests
  env:
    CF_API_TOKEN: ${{ secrets.CF_API_TOKEN }}
    CF_ZONE_ID: ${{ secrets.CF_ZONE_ID }}
  run: go test -tags=integration -timeout 10m ./test/integration/ -v
```

## See Also

- [BUILD.md](BUILD.md) — Build instructions
- [CONTRIBUTING.md](CONTRIBUTING.md) — Development guidelines
- [test/integration/README.md](../test/integration/README.md) — Integration test details

