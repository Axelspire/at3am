# at3am — Intelligent DNS-01 Validation for ACME Clients

**at3am** is a stateless CLI tool that watches global DNS resolvers in parallel and signals exactly when a DNS-01 TXT record has propagated with sufficient confidence for Let's Encrypt (or any ACME server) to validate it.

It is aligned with [Let's Encrypt's Multi-Perspective Issuance Corroboration (MPIC)](https://letsencrypt.org/2020/02/19/multi-perspective-validation/) and works as a drop-in Certbot manual auth hook or as a standalone propagation oracle in any CI/CD pipeline.

---

## 🚀 Quick Install

**Linux, macOS, or Git Bash:**
```bash
curl -sfL https://raw.githubusercontent.com/Axelspire/at3am/main/install.sh | sh
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/Axelspire/at3am/main/install.ps1 | iex
```

After installation run `at3am --help` to verify.

---

## Table of Contents

- [Quick Install](#-quick-install)
- [Quick Start](#quick-start)
- [Building from Source](#building-from-source)
- [Testing](#testing)
  - [Unit tests](#unit-tests)
  - [Integration tests](#integration-tests)
- [Installation](#installation)
- [at3am wait — Full Parameter Reference](#at3am-wait--full-parameter-reference)
  - [Required flags](#required-flags)
  - [Profiles](#profiles)
  - [Timing](#timing)
  - [Output](#output)
  - [Logging](#logging)
  - [Resolvers](#resolvers)
  - [DNSSEC Validation](#dnssec-validation)
  - [Challenge Type](#challenge-type)
  - [Automation](#automation)
  - [Advanced / Tuning](#advanced--tuning)
  - [Mock mode](#mock-mode)
- [at3am-hook — Certbot Integration](#at3am-hook--certbot-integration)
  - [Installation](#installation-1)
  - [All 54 DNS Providers](#all-54-dns-providers)
  - [Provider Credentials](#provider-credentials)
  - [Usage](#usage)
  - [Environment variables](#environment-variables)
- [Exit Codes](#exit-codes)
- [Usage Examples](#usage-examples)
- [Mock Scenarios](#mock-scenarios)
- [Prometheus Metrics](#prometheus-metrics)
- [License & Attribution](#license--attribution)

---

## Quick Start

### Build & Install

```bash
git clone https://github.com/axelspire/at3am && cd at3am
make build
sudo install -m 755 bin/at3am bin/at3am-hook /usr/local/bin/
```

### Standalone: Wait for DNS propagation

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "your-validation-token" \
  --profile default
```

### Certbot (with automatic DNS provider detection)

```bash
sudo certbot certonly --manual \
  --manual-auth-hook    "at3am-hook manual-auth" \
  --manual-cleanup-hook "at3am-hook manual-cleanup" \
  --preferred-challenges dns \
  -d example.com
```

### acme.sh (with DNS provider)

```bash
# Set up at3am as the DNS hook
export HOOK_DOMAIN_CONF="/etc/acme.sh/at3am.conf"
export AT3AM_DNS_PROVIDER=cloudflare
export AT3AM_DNS_CREDS=/etc/acme.sh/cloudflare.yaml

# Issue certificate
acme.sh --issue \
  --dns "dns_at3am" \
  -d example.com \
  -d "*.example.com"

# Renewal (add to cron)
acme.sh --cron --home /etc/acme.sh
```

### lego (with DNS provider)

```bash
# Set up environment
export LEGO_PROPAGATION_TIMEOUT=600
export AT3AM_DNS_PROVIDER=route53
export AT3AM_DNS_CREDS=/etc/lego/route53.yaml

# Issue certificate
lego --email admin@example.com \
  --dns "dns_at3am" \
  --domains example.com \
  --domains "*.example.com" \
  run
```

### step-ca / Smallstep (with DNS provider)

```bash
# Configure step-ca provisioner with DNS hook
step ca provisioner add dns-provisioner \
  --type ACME \
  --challenge dns-01 \
  --dns-hook "at3am-hook manual-auth" \
  --dns-cleanup-hook "at3am-hook manual-cleanup"

# Issue certificate
step ca certificate example.com cert.pem key.pem \
  --provisioner dns-provisioner
```

### Manual DNS workflow (any ACME client)

```bash
#!/bin/bash
DOMAIN="example.com"
TOKEN="$1"  # From ACME server

# 1. Create DNS record via your provider
# (your DNS API call here)

# 2. Wait for propagation
at3am wait \
  --domain "_acme-challenge.${DOMAIN}" \
  --expected "${TOKEN}" \
  --profile default

# 3. Signal ACME server to validate
# (your ACME client continues here)

# 4. Clean up DNS record
# (your DNS API call here)
```

---

## Building from Source

**Requirements:** Go 1.21+ (tested on 1.26.1), GNU Make.

```bash
git clone https://github.com/axelspire/at3am
cd at3am
```

| Command | Description |
|---------|-------------|
| `make build` | Build both binaries for the current platform → `bin/` |
| `make build-linux` | Linux amd64 → `dist/linux-amd64/` |
| `make build-darwin` | macOS amd64 → `dist/darwin-amd64/` |
| `make build-darwin-arm64` | macOS arm64 (Apple Silicon) → `dist/darwin-arm64/` |
| `make build-windows` | Windows amd64 → `dist/windows-amd64/` |
| `make build-all` | All four platforms |
| `make test` | Run all tests |
| `make coverage` | Tests with coverage report |
| `make check` | `go fix` + `go vet` + `gofmt` |
| `make clean` | Remove `bin/` and `dist/` |

Version information (git tag, commit hash, build timestamp) is baked into both binaries automatically via ldflags. Check it with:

```bash
at3am version
at3am-hook version
```

---

## Testing

See [docs/TESTING.md](docs/TESTING.md) for comprehensive testing guide covering:
- Unit tests with 87.2% coverage
- Mock integration tests (no credentials required)
- Live Cloudflare integration tests
- Propagation metrics measurement
- CI/CD integration examples

Quick start:
```bash
make test              # Run all unit tests
make coverage          # Generate coverage report
go test -timeout 5m ./test/integration/ -v -run TestCertbotMock  # Mock integration tests
```

---

## Installation

After building, copy the binaries to a directory in your `PATH`. The hook must be reachable by `root` (the user that runs `certbot`):

```bash
sudo install -m 755 bin/at3am     /usr/local/bin/at3am
sudo install -m 755 bin/at3am-hook /usr/local/bin/at3am-hook
```

Verify:

```bash
at3am version
sudo at3am-hook version   # confirm root can find it
```

> **Important for Certbot:** Certbot runs hooks as `root`. The binary must be in a directory that is in root's `PATH` (e.g. `/usr/local/bin`) and must have the executable bit set. Simply having `./bin/at3am-hook` in your working directory is not enough.

---

## at3am wait — Full Parameter Reference

```
at3am wait --domain <fqdn> --expected <token> [flags]
```

### Required flags

| Flag | Short | Description |
|------|-------|-------------|
| `--domain <fqdn>` | `-d` | The full DNS name to check, e.g. `_acme-challenge.example.com` |
| `--expected <value>` | `-e` | The exact TXT record value to wait for |

### Profiles

A profile sets a pre-tuned combination of auth threshold, public threshold, consecutive-pass count, poll interval, and timeout. All profile settings can be overridden by explicit flags — the profile is applied first, then any flag you set takes precedence.

```bash
at3am wait -d _acme-challenge.example.com -e "token" --profile default
```

| Profile | Auth threshold | Public threshold | Consecutive passes | Poll interval | Timeout |
|---------|---------------|-----------------|-------------------|--------------|---------|
| `strict` | ALL auth NS | ALL public − 2 | 3 | 10 s | 600 s |
| `default` *(default)* | ALL auth NS | ≥ 1 public | 2 | 5 s | 300 s |
| `fast` | ≥ ceil(N/2) auth NS | ≥ 1 public | 1 | 2 s | 120 s |
| `yolo` | ≥ 1 auth NS | ≥ 0 public | 1 | 1 s | 60 s |

**READY** is declared when `auth_correct ≥ auth_threshold AND public_correct ≥ public_threshold` for `consecutive` consecutive polls in a row.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--profile <name>` | | `default` | Pre-built profile: `strict`, `default`, `fast`, `yolo` |

### Timing

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--timeout <duration>` | `-t` | `5m0s` | Maximum time to wait before giving up |
| `--interval <duration>` | `-i` | `5s` | Time between polling rounds |
| `--consecutive <n>` | | `2` | How many consecutive passing polls are required |

Durations accept Go format: `30s`, `5m`, `10m30s`, `1h`.

### Output

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output <format>` | `-o` | `human` | Output format: `human`, `json`, `quiet` |
| `--prometheus-port <port>` | | `0` (off) | Expose a Prometheus `/metrics` endpoint on this port |

**`human`** — coloured, human-readable status lines printed on each poll plus a final result.  
**`json`** — one JSON object per poll, suitable for log aggregators and SIEM pipelines.  
**`quiet`** — no output during polling; only the final exit code matters (ideal for scripts).

### Logging

Logging is independent of `--output`. It is disabled by default and must be explicitly enabled.

| Flag | Default | Description |
|------|---------|-------------|
| `--debug` | off | Enable DEBUG-level logging to stdout. Shows every individual resolver query result, pool phase transitions, and DNSSEC per-resolver status. Shorthand for `--log-level debug`. |
| `--log-level <level>` | *(off)* | Stdout log level: `debug`, `info`, `warn`, `error`. Only that level and above are printed. |
| `--log-file <path>` | *(off)* | Write production logs (INFO+) to a file. Appends. Captures startup config, per-poll summary, ready events, and total latency. DEBUG lines are never written to the file. |

`--debug` and `--log-file` can be used simultaneously: full debug detail on screen, clean operational log on disk.

Log line format:
```
2026-03-14 17:48:21.368:INFO:wait/wait.go:160:poll #1 | elapsed=5.002s auth=2/2(need 2) pub=25/25(need 1) ready=true scenario=full_propagation
```

### Resolvers

| Flag | Default | Description |
|------|---------|-------------|
| `--resolver <ip:port>` | *(none)* | Add a custom resolver. Can be specified multiple times. Added to the built-in pool of 25+ public resolvers. |

Authoritative nameservers for the domain are **auto-discovered** at startup via NS lookup. Auth NS receive higher scoring weight than public resolvers.

### DNSSEC Validation

| Flag | Default | Description |
|------|---------|-------------|
| `--dnssec-validate` | off | Set the DO (DNSSEC OK) bit on every query. Record the AD (Authenticated Data) bit from responses. Results are **informational only** — DNSSEC status does not block readiness. |

When enabled, human output shows:
```
🔒 DNSSEC: 14/18 resolvers authenticated (AD bit)
```

### Challenge Type

| Flag | Default | Description |
|------|---------|-------------|
| `--challenge-type <type>` | `dns-01` | ACME challenge variant. `dns-01` uses auth-first gating (prevents negative-caching before auth NS confirm). `persist` (DNS-PERSIST-01 draft) queries all resolvers simultaneously — suitable for permanently pre-provisioned TXT records. |

### Automation

These hooks execute **after** readiness is confirmed (exit code 0 only).

| Flag | Default | Description |
|------|---------|-------------|
| `--on-ready <command>` | *(none)* | Shell command to run when ready. Supports template variables: `$DOMAIN`, `$CONFIDENCE`, `$ELAPSED`. |
| `--webhook <url>` | *(none)* | HTTP POST to this URL when ready. Body is a JSON object with `domain`, `confidence`, and `elapsed_seconds`. |

Example:
```bash
at3am wait -d _acme-challenge.example.com -e "token" \
  --on-ready 'echo "Done: $DOMAIN in $ELAPSED at $CONFIDENCE% confidence"'
```

### Advanced / Tuning

These override the internal scoring weights used by the legacy (non-profile) engine. Ignored when a `--profile` is set.

| Flag | Default | Description |
|------|---------|-------------|
| `--threshold <0-100>` | `95.0` | Legacy confidence threshold percentage |
| `--auth-weight <0-1>` | `0.6` | Legacy weight given to authoritative NS results |
| `--public-weight <0-1>` | `0.4` | Legacy weight given to public resolver results |

### Mock mode

For testing pipelines without real DNS records.

| Flag | Default | Description |
|------|---------|-------------|
| `--mock` | off | Enable mock DNS mode. No real DNS queries are made. |
| `--mock-scenario <name>` | `instant` | Which mock scenario to run. See [Mock Scenarios](#mock-scenarios). |

---

## at3am-hook — Certbot Integration

`at3am-hook` is a **fully self-contained** Certbot manual auth hook. It:

1. **Creates** the `_acme-challenge` TXT record via your DNS provider's API
2. **Waits** until that record has propagated globally with high confidence
3. **Deletes** the record in the cleanup hook after Certbot validates

No helper scripts or separate binaries are needed. DNS provider support is compiled directly into the binary via [**libdns**](https://github.com/libdns/libdns) (MIT License).

### All 54 DNS Providers

All providers are compiled into the binary — no plugins or external dependencies. NS-based autodetection is available for most; for others set `AT3AM_DNS_PROVIDER` explicitly.

| Provider | Key | Required credentials |
|----------|-----|----------------------|
| **acmedns** | `acmedns` | `server_url` + `username` + `password` + `subdomain` |
| **ACME-Proxy** | `acmeproxy` | `address` (+ optional `username` / `password`) |
| **AliDNS** | `alidns` | `access_key_id` + `access_key_secret` |
| **ALL-INKL** | `all-inkl` | `kas_username` + `kas_password` |
| **AutoDNS** | `autodns` | `username` + `password` |
| **Azure DNS** | `azure` | `subscription_id` + `resource_group_name` |
| **BlueCat** | `bluecat` | `server_url` + `username` + `password` + `configuration_name` + `view_name` |
| **Bunny** | `bunny` | `access_key` |
| **Cloudflare** | `cloudflare` | `api_token` |
| **ClouDNS** | `cloudns` | `auth_id` + `auth_password` |
| **deSEC** | `desec` | `token` |
| **DigitalOcean** | `digitalocean` | `auth_token` |
| **DirectAdmin** | `directadmin` | `server_url` + `user` + `login_key` |
| **DNSimple** | `dnsimple` | `api_access_token` |
| **DNS Update** | `dnsupdate` | `addr` (+ optional `tsig`) |
| **Domainnameshop** | `domainnameshop` | `api_token` + `api_secret` |
| **DuckDNS** | `duckdns` | `api_token` |
| **Dynu** | `dynu` | `api_token` |
| **Dynv6** | `dynv6` | `token` |
| **Gandi** | `gandi` | `bearer_token` |
| **G-Core** | `gcore` | `api_key` |
| **GleSYS** | `glesys` | `project` + `api_key` |
| **GoDaddy** | `godaddy` | `api_token` (`key:secret`) |
| **Google Cloud DNS** | `googleclouddns` | `gcp_project` |
| **Hurricane Electric** | `he` | `api_key` |
| **Hetzner** | `hetzner` | `auth_api_token` |
| **Huawei Cloud** | `huaweicloud` | `access_key_id` + `secret_access_key` + `region_id` |
| **Infomaniak** | `infomaniak` | `api_token` |
| **INWX** | `inwx` | `username` + `password` |
| **IONOS** | `ionos` | `auth_api_token` |
| **Linode** | `linode` | `api_token` |
| **Loopia** | `loopia` | `username` + `password` |
| **LuaDNS** | `luadns` | `email` + `api_key` |
| **mijn.host** | `mijnhost` | `api_key` |
| **Mythic Beasts** | `mythicbeasts` | `key_id` + `secret` |
| **Namecheap** | `namecheap` | `api_key` + `user` |
| **NameSilo** | `namesilo` | `api_token` |
| **Netcup** | `netcup` | `customer_number` + `api_key` + `api_password` |
| **Netlify** | `netlify` | `personal_access_token` |
| **Njalla** | `njalla` | `api_token` |
| **OVH** | `ovh` | `endpoint` + `application_key` + `application_secret` + `consumer_key` |
| **Porkbun** | `porkbun` | `api_key` + `api_secret_key` |
| **PowerDNS** | `powerdns` | `server_url` + `api_token` + `server_id` |
| **Regfish** | `regfish` | `api_token` |
| **RFC 2136** | `rfc2136` | `server` (+ optional TSIG) |
| **AWS Route53** | `route53` | `access_key_id` + `secret_access_key` (or IAM role) |
| **Scaleway** | `scaleway` | `secret_key` |
| **Simply.com** | `simplydotcom` | `account_name` + `api_key` |
| **Tecnocratica** | `tecnocratica` | `api_token` |
| **Tencent Cloud** | `tencentcloud` | `secret_id` + `secret_key` |
| **TransIP** | `transip` | `login` |
| **UniFi** | `unifi` | `api_key` + `base_url` |
| **Volcengine** | `volcengine` | `access_key_id` + `access_key_secret` |
| **WebSupport** | `websupport` | `api_key` + `api_secret` |
| **WEDOS** | `wedos` | `username` + `password` |
| **West.cn** | `westcn` | `username` + `api_password` |

### Installation

```bash
# Build
make build

# Install for root (required for sudo certbot)
sudo install -m 755 bin/at3am-hook /usr/local/bin/at3am-hook

# Verify root can find and run it
sudo at3am-hook version
```

### Provider credentials

The hook reads credentials from a YAML file. On first run with an unknown provider,
a commented template is created automatically.

**Default path:** `/etc/at3am/<provider>.yaml`
**Override:** `export AT3AM_DNS_CREDS=/path/to/creds.yaml`

Example for Cloudflare (`/etc/at3am/cloudflare.yaml`):
```yaml
provider: cloudflare

cloudflare:
  api_token: "your-scoped-api-token"
```

Example for Route53 — using explicit keys:
```yaml
provider: route53

route53:
  access_key_id: "AKIAIOSFODNN7EXAMPLE"
  secret_access_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  region: "us-east-1"
```

Route53 with an IAM role (EC2 / ECS / Lambda — leave keys empty and the AWS SDK credential chain is used):
```yaml
provider: route53

route53:
  region: "us-east-1"
```

Azure with Managed Identity (leave tenant/client/secret empty):
```yaml
provider: azure

azure:
  subscription_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  resource_group_name: "my-dns-rg"
```

Protect the file:
```bash
sudo chmod 600 /etc/at3am/cloudflare.yaml
sudo chown root:root /etc/at3am/cloudflare.yaml
```

Individual credential values can also be overridden with env vars of the form
`AT3AM_DNS_<UPPER_KEY>`, e.g.:
```bash
export AT3AM_DNS_API_TOKEN="my-token"
```

### Usage

```bash
sudo certbot certonly --manual \
  --manual-auth-hook    "at3am-hook manual-auth" \
  --manual-cleanup-hook "at3am-hook manual-cleanup" \
  --preferred-challenges dns \
  -d example.com
```

> The quotes around `"at3am-hook manual-auth"` are required — Certbot passes the entire string as a shell command and the subcommand must be included.

The hook **autodetects** your provider by resolving the NS records of your domain and matching them against the table above. To override:

```bash
export AT3AM_DNS_PROVIDER=cloudflare
export AT3AM_DNS_CREDS=/etc/at3am/cloudflare.yaml
```

For multiple domains:
```bash
sudo certbot certonly --manual \
  --manual-auth-hook    "at3am-hook manual-auth" \
  --manual-cleanup-hook "at3am-hook manual-cleanup" \
  --preferred-challenges dns \
  -d example.com -d www.example.com -d api.example.com
```

For renewals, add to `/etc/letsencrypt/renewal/example.com.conf`:
```ini
[renewalparams]
manual_auth_hook = at3am-hook manual-auth
manual_cleanup_hook = at3am-hook manual-cleanup
```

### What happens during manual-auth

```
certbot calls "at3am-hook manual-auth"
  │
  ├── 1. Autodetect provider via NS records (or read AT3AM_DNS_PROVIDER)
  ├── 2. Load credentials from YAML (create template if file missing)
  ├── 3. Early access test: create + delete _at3am_test_<epoch> TXT record
  │         → fails fast if credentials are wrong before touching the real zone
  ├── 4. Create _acme-challenge.<domain> TXT = <CERTBOT_VALIDATION>
  └── 5. Poll global resolvers until propagation is confirmed (at3am engine)

certbot validates with Let's Encrypt

certbot calls "at3am-hook manual-cleanup"
  └── 6. Delete _acme-challenge.<domain> TXT record
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CERTBOT_DOMAIN` | *(set by Certbot)* | Bare domain, e.g. `example.com` |
| `CERTBOT_VALIDATION` | *(set by Certbot)* | TXT value to provision |
| `AT3AM_DNS_PROVIDER` | *(autodetect)* | Provider name (see table above) |
| `AT3AM_DNS_CREDS` | `/etc/at3am/<provider>.yaml` | Path to YAML credentials file |
| `AT3AM_SKIP_DNS` | `0` | Set to `1` to skip record create/delete (propagation-wait only) |
| `AT3AM_PROFILE` | `default` | Readiness profile: `strict`, `default`, `fast`, `yolo` |
| `AT3AM_OUTPUT` | `quiet` | Output format: `human`, `json`, `quiet` |
| `AT3AM_LOG_LEVEL` | *(off)* | Stdout log level: `debug`, `info`, `warn`, `error` |
| `AT3AM_LOG_FILE` | *(off)* | Path to write a production log file (INFO+ events) |
| `AT3AM_CHALLENGE_TYPE` | `dns-01` | `dns-01` or `persist` |

Example with explicit provider and logging:
```bash
export AT3AM_DNS_PROVIDER=cloudflare
export AT3AM_DNS_CREDS=/etc/at3am/cloudflare.yaml
export AT3AM_PROFILE=strict
export AT3AM_LOG_FILE=/var/log/at3am.log
export AT3AM_LOG_LEVEL=info

sudo certbot certonly --manual \
  --manual-auth-hook    "at3am-hook manual-auth" \
  --manual-cleanup-hook "at3am-hook manual-cleanup" \
  --preferred-challenges dns \
  -d example.com
```

### License & Attribution

`at3am-hook` statically links **54 DNS provider packages** via [libdns](https://github.com/libdns/libdns):
- **53 providers** — MIT License
- **1 provider** (dnsimple) — Apache 2.0

All licenses are compatible with at3am's Apache 2.0 license. See the `LICENSES/` directory for every provider's license text. When distributing, include both `LICENSE` and the `LICENSES/` directory.

---

## Exit Codes

Both `at3am` and `at3am-hook` use the same exit codes:

| Code | Meaning |
|------|---------|
| `0` | Success — record confirmed propagated |
| `1` | Timeout — propagation not confirmed within the time limit |
| `2` | Configuration error — bad flags or env vars |
| `3` | DNS error — fatal resolver failure |

---

## Usage Examples

### Minimal — wait with defaults

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue"
```

Uses the `default` profile: ALL auth NS + ≥1 public, 2 consecutive passes, 5 min timeout.

### Strict — production / high-security use

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --profile strict
```

Requires ALL auth NS + (ALL public − 2) for 3 consecutive polls. 10 min timeout.

### Fast — internal / staging environments

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --profile fast
```

Passes when ≥ half of auth NS confirm, plus 1 public resolver. Single pass. 2 min timeout.

### JSON output for log pipelines

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --output json \
  --log-file /var/log/at3am/example.com.log
```

### Debug — troubleshoot a specific domain

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --debug \
  --profile yolo \
  --output human
```

Prints every individual resolver query with latency, DNSSEC AD bit, and pool phase transitions.

### DNSSEC-signed zone

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --dnssec-validate \
  --profile default
```

### Custom resolver added to the pool

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --resolver 192.0.2.1:53 \
  --resolver 198.51.100.1:53
```

### Override timeout and interval from a profile

Profiles set sensible defaults but every timing flag wins over the profile:

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --profile strict \
  --timeout 20m \
  --interval 15s
```

### Run a command when ready

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --on-ready 'systemctl reload nginx && echo "Cert ready at $ELAPSED for $DOMAIN"'
```

### Post a webhook when ready

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --webhook https://hooks.example.com/cert-ready
```

### DNS-PERSIST-01 (long-lived TXT record)

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "permanent-token" \
  --challenge-type persist \
  --profile default
```

Queries all resolvers simultaneously without auth-first gating.

### In a shell script (quiet mode + exit code)

```bash
#!/bin/bash
set -e

TOKEN="$1"
DOMAIN="$2"

echo "Waiting for DNS propagation..."
at3am wait \
  --domain "_acme-challenge.${DOMAIN}" \
  --expected "${TOKEN}" \
  --output quiet \
  --profile default

echo "DNS propagated. Proceeding with certificate issuance."
```

### Certbot with full logging

```bash
export AT3AM_PROFILE=strict
export AT3AM_LOG_FILE=/var/log/letsencrypt/at3am.log

sudo certbot certonly --manual \
  --manual-auth-hook    "at3am-hook manual-auth" \
  --manual-cleanup-hook "at3am-hook manual-cleanup" \
  --preferred-challenges dns \
  --non-interactive \
  --agree-tos \
  --email admin@example.com \
  -d example.com \
  -d "*.example.com"
```

---

## Mock Scenarios

Mock mode (`--mock`) replaces real DNS queries with deterministic in-process behaviour. Useful for testing pipelines and automation.

| Scenario | Description |
|----------|-------------|
| `instant` *(default)* | Record is immediately visible to all resolvers |
| `slow_propagation` | Auth NS confirm immediately; public resolvers pick it up after 3 polls |
| `flaky` | Some resolvers intermittently time out; should still eventually pass |
| `partial` | A fixed subset of resolvers always see it; others never do — useful for testing `fast`/`yolo` profiles |
| `timeout` | Record never appears; always exits with code 1 |

```bash
# Test your script against a slow propagation
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "test-token" \
  --mock \
  --mock-scenario slow_propagation \
  --profile default \
  --output human
```

---

## Prometheus Metrics

Expose a Prometheus metrics endpoint during a long wait:

```bash
at3am wait \
  --domain _acme-challenge.example.com \
  --expected "abc123tokenvalue" \
  --prometheus-port 9090 \
  --profile strict
```

Scrape at `http://localhost:9090/metrics`. Available metrics include per-resolver found/error counts, auth/public scores, and overall confidence.

