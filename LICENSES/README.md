# Third-Party Licenses

This directory contains the licenses for all DNS provider packages integrated into **at3am-hook** via the [libdns](https://github.com/libdns/libdns) ecosystem.

## Core Dependencies

- **libdns** — MIT License (https://github.com/libdns/libdns)
- **miekg/dns** — BSD 3-Clause License (https://github.com/miekg/dns)
- **spf13/cobra** — Apache License 2.0 (https://github.com/spf13/cobra)

## DNS Provider Packages (54 total)

**53 providers are MIT-licensed; 1 is Apache 2.0-licensed:**
- **Apache 2.0:** dnsimple
- **MIT:** all others (53 providers)

Each provider's license is included as `LICENSE-<provider>` in this directory.

### Batch 1 (Original 13)
cloudflare, route53, googleclouddns, azure, digitalocean, hetzner, godaddy, namecheap, porkbun, ovh, gandi, linode, powerdns

### Batch 2 (Additional 30)
dnsimple, scaleway, inwx, rfc2136, netlify, tencentcloud, alidns, desec, transip, bunny, ionos, namesilo, acmedns, duckdns, he, infomaniak, luadns, cloudns, directadmin, loopia, glesys, huaweicloud, acmeproxy, dynv6, mythicbeasts, simplydotcom, netcup, dynu, gcore

### Batch 3 (Additional 14)
all-inkl, autodns, bluecat, dnsupdate, domainnameshop, mijnhost, njalla, regfish, tecnocratica, unifi, websupport, wedos, westcn, volcengine

## License Compliance

**at3am** is licensed under the Apache License 2.0 with patent grant. All integrated provider packages are compatible with this license:
- **53 providers:** MIT License (compatible with Apache 2.0)
- **1 provider:** Apache License 2.0 (dnsimple — same license as at3am)

When distributing **at3am-hook**, ensure that:
1. A copy of the Apache License 2.0 is included (see `../LICENSE`)
2. This directory (`LICENSES/`) is included with all provider licenses
3. Any modifications to provider code are documented

## Obtaining Full Source

All provider packages are open-source and available at:
```
https://github.com/libdns/<provider>
```

For example:
- https://github.com/libdns/cloudflare
- https://github.com/libdns/route53
- etc.

