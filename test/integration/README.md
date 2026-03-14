# Integration Tests

This directory contains integration tests for **at3am-hook** with real and mock DNS providers.

## Test Files

### `certbot_cloudflare_test.go`
**Real Cloudflare integration test** — requires valid Cloudflare credentials.

Tests the full Certbot DNS-01 workflow:
1. Create a TXT record via Cloudflare API
2. Wait for propagation using at3am's scoring engine
3. Delete the record

**Requirements:**
- `CF_API_TOKEN` — Cloudflare API token (scoped to DNS write)
- `CF_ZONE_ID` — Cloudflare zone ID for your domain
- `TEST_DOMAIN` — Domain to test (default: `axelspire.com`)

**Run:**
```bash
export CF_API_TOKEN="your-token"
export CF_ZONE_ID="your-zone-id"
export TEST_DOMAIN="example.com"
go test -tags=integration -timeout 10m ./test/integration/ -v -run TestCertbotCloudflare
```

**What it tests:**
- ✅ Cloudflare API record creation
- ✅ Cloudflare API record deletion
- ✅ DNS propagation detection
- ✅ at3am scoring engine with real resolvers
- ✅ Full Certbot workflow simulation

### `certbot_mock_test.go`
**Mock integration tests** — no credentials required.

Tests the Certbot workflow with mock DNS:
- `TestCertbotMock` — instant propagation scenario
- `TestCertbotMockSlowPropagation` — delayed propagation scenario
- `TestCertbotMockMultipleDomains` — renewal with multiple domains

**Run:**
```bash
go test -timeout 5m ./test/integration/ -v -run TestCertbotMock
```

**What it tests:**
- ✅ manual-auth hook simulation (record creation)
- ✅ Propagation wait with mock resolver
- ✅ manual-cleanup hook simulation (record deletion)
- ✅ Slow propagation handling
- ✅ Multiple domain renewal workflow

### `propagation_test.go`
**Existing propagation test** — measures real DNS propagation metrics.

## Running Tests

### All integration tests (mock only)
```bash
go test -timeout 5m ./test/integration/ -v
```

### With real Cloudflare provider
```bash
export CF_API_TOKEN="your-token"
export CF_ZONE_ID="your-zone-id"
go test -tags=integration -timeout 10m ./test/integration/ -v
```

### Specific test
```bash
go test -timeout 5m ./test/integration/ -v -run TestCertbotMock
```

### With verbose output
```bash
go test -timeout 5m ./test/integration/ -v -run TestCertbotMock -v
```

## Test Scenarios

### Mock Scenarios (in `certbot_mock_test.go`)

1. **Instant Propagation** (`instant`)
   - Record visible immediately
   - Tests fast-path scoring
   - Typical wait: 5-10 seconds

2. **Slow Propagation** (`slow`)
   - Record takes time to propagate
   - Tests strict profile scoring
   - Typical wait: 30-60 seconds

3. **Multiple Domains** (renewal)
   - Sequential domain processing
   - Tests cleanup between domains
   - Simulates `certbot renew` workflow

## Cloudflare Setup

To run the real Cloudflare test:

1. **Create a Cloudflare API token:**
   - Go to https://dash.cloudflare.com/profile/api-tokens
   - Create token with permissions: `Zone.DNS.Edit`
   - Scope to your test domain

2. **Get your zone ID:**
   ```bash
   curl -H "Authorization: Bearer YOUR_TOKEN" \
     https://api.cloudflare.com/client/v4/zones?name=example.com | jq '.result[0].id'
   ```

3. **Run the test:**
   ```bash
   export CF_API_TOKEN="your-token"
   export CF_ZONE_ID="your-zone-id"
   export TEST_DOMAIN="example.com"
   go test -tags=integration -timeout 10m ./test/integration/ -v -run TestCertbotCloudflare
   ```

## Expected Output

### Mock test (successful)
```
=== RUN   TestCertbotMock
    certbot_mock_test.go:XX: Starting mock Certbot + at3am-hook integration test
    certbot_mock_test.go:XX:   Domain: example.com
    certbot_mock_test.go:XX:   Challenge: _acme-challenge.example.com
    certbot_mock_test.go:XX:   Token: mock-validation-token-12345
    certbot_mock_test.go:XX: Step 1: Simulating manual-auth hook (create record)...
    certbot_mock_test.go:XX:   ✓ Record created (mocked)
    certbot_mock_test.go:XX: Step 2: Waiting for propagation (mock scenario)...
    certbot_mock_test.go:XX:   Resolver pool: 8 resolvers
    certbot_mock_test.go:XX:   ✓ Propagation confirmed (mock)
    certbot_mock_test.go:XX: Step 3: Simulating manual-cleanup hook (delete record)...
    certbot_mock_test.go:XX:   ✓ Record deleted (mocked)
    certbot_mock_test.go:XX: ✓ Mock Certbot + at3am-hook integration test passed
--- PASS: TestCertbotMock (2.34s)
```

### Cloudflare test (successful)
```
=== RUN   TestCertbotCloudflare
    certbot_cloudflare_test.go:XX: Starting Certbot + Cloudflare integration test
    certbot_cloudflare_test.go:XX:   Domain: example.com
    certbot_cloudflare_test.go:XX:   Challenge: _acme-challenge.example.com
    certbot_cloudflare_test.go:XX:   Token: a1b2c3d4e5f6...
    certbot_cloudflare_test.go:XX: Step 1: Creating TXT record via Cloudflare API...
    certbot_cloudflare_test.go:XX:   Record created: abc123def456
    certbot_cloudflare_test.go:XX: Step 2: Waiting for propagation...
    certbot_cloudflare_test.go:XX:   Resolver pool: 8 public, 2 auth, 10 total
    certbot_cloudflare_test.go:XX:   ✓ Propagation confirmed
    certbot_cloudflare_test.go:XX: Step 3: Verifying record visibility...
    certbot_cloudflare_test.go:XX:   ✓ Record verified visible
    certbot_cloudflare_test.go:XX: ✓ Certbot + Cloudflare integration test passed
--- PASS: TestCertbotCloudflare (45.23s)
```

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

Add to GitHub Actions:
```yaml
- name: Run integration tests
  run: go test -timeout 5m ./test/integration/ -v
```

For Cloudflare tests (with secrets):
```yaml
- name: Run Cloudflare integration tests
  env:
    CF_API_TOKEN: ${{ secrets.CF_API_TOKEN }}
    CF_ZONE_ID: ${{ secrets.CF_ZONE_ID }}
  run: go test -tags=integration -timeout 10m ./test/integration/ -v
```

