# at3am companion (server-side)

Proprietary / customer-vanity companion for 3AM installs. Published under
`agent/at3am/companion/` on each customer root GET — **outside** the multi-arch
release zip (same layout contract as acme.sh / win-acme).

## v1 contents

| File | Role |
|---|---|
| `VERSION` | Companion version (fleet may assert; catalog `companion`) |
| `agent.env.tmpl` | Template for local `~/.config/at3am/agent.env` |
| `README.md` | This file |

v1 does **not** fold the datasink scan companion into at3am (that stays with
acme.sh). Future scan/hooks for at3am land here without changing the public path.

## Install behavior

Vanity `install.sh` / `install.ps1` fetch companion objects after the binary zip
and persist `THREEAM_AGENT_BASE` for self-upgrade on every agent run.
