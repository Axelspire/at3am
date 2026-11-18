# at3am — install (DNS-01 propagation oracle)

**v1 product:** `at3am` is a **DNS-01 propagation oracle / Certbot hook helper**. It watches global resolvers and signals when a challenge TXT has propagated. It is **not** a full ACME certificate issuer yet.

**Enterprise channel (later):** a full RFC 8555 ACME client may ship under the **same** `at3am` brand and customer path `agent/at3am/` — not a rename and not `agent/at3am-enterprise/`. See `ENTERPRISE.md`.

**Not this product:** datasink HTTP routes under `/v1/at3am/*` are historical API path names on the datasink ALB. They are unrelated to this client.

## 3AM customers (vanity — preferred)

Fleet pin is **`agents.at3am.module`** only (release tag, e.g. `@MODULE@`). There is no separate SPA `pin` field.

```bash
# Linux / macOS
curl -fsSL @THREEAM_AGENT_BASE@/install.sh | sh

# Windows (PowerShell)
irm @THREEAM_AGENT_BASE@/install.ps1 | iex
```

- Digests: installers verify `tags/@MODULE@/<platform>` against the zip bytes.
- Large archives: vanity may publish `archive/@MODULE@/<platform>.url` (APIGW ~10MB cap); installers fall back to that URL (tagged GitHub asset) and still require the vanity digest.
- **Self-upgrade:** every `at3am` invocation checks `@THREEAM_AGENT_BASE@/MODULE` and refreshes when the fleet module moves. Set `AT3AM_NO_UPGRADE=1` to skip.
- Companion (server-side): `companion/VERSION` and env template are fetched from the vanity tree (same pattern as acme.sh).

Module on this customer: **`@MODULE@`** · companion **`@COMPANION@`**.

## Community / OSS (no 3AM vanity)

Do **not** use `releases/latest` for deliberate installs. Pin an explicit tag:

```bash
AT3AM_MODULE=v0.2.2 curl -fsSL https://raw.githubusercontent.com/Axelspire/at3am/v0.2.2/install.sh | sh
```

```powershell
$env:AT3AM_MODULE = "v0.2.2"
irm https://raw.githubusercontent.com/Axelspire/at3am/v0.2.2/install.ps1 | iex
```

## After install

```bash
at3am --help
at3am version
# Certbot hook (example):
#   --manual-auth-hook at3am-hook
```

## Upgrade

- **3AM vanity:** automatic on every run (wrapper + optional binary self-upgrade). Re-running the install one-liner also refreshes companion + digests.
- **Community:** set a new `AT3AM_MODULE` and re-run install.

## Related

- Repo README / `ENTERPRISE.md`
- Packaging design: 3AM agent packaging (acme.sh / win-acme contract)
- Jira: [CORE-62](https://axelspire.atlassian.net/browse/CORE-62)
