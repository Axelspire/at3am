# DNS Providers — Complete Reference

**at3am-hook** supports **54 DNS providers** via the [libdns](https://github.com/libdns/libdns) ecosystem.

## All 54 Providers and Authentictation Methods

1. **cloudflare** — `api_token`
2. **route53** — `access_key_id` + `secret_access_key` (or IAM role)
3. **googleclouddns** — `gcp_project` (+ optional service account JSON)
4. **azure** — `subscription_id` + `resource_group_name` (+ optional SP creds)
5. **digitalocean** — `auth_token`
6. **hetzner** — `auth_api_token`
7. **godaddy** — `api_token` (format: `key:secret`)
8. **namecheap** — `api_key` + `user`
9. **porkbun** — `api_key` + `api_secret_key`
10. **ovh** — `endpoint` + `application_key` + `application_secret` + `consumer_key`
11. **gandi** — `bearer_token`
12. **linode** — `api_token`
13. **powerdns** — `server_url` + `api_token` + `server_id`
14. **dnsimple** — `api_access_token`
15. **scaleway** — `secret_key`
16. **inwx** — `username` + `password` (+ optional TOTP)
17. **rfc2136** — `server` + optional TSIG
18. **netlify** — `personal_access_token`
19. **tencentcloud** — `secret_id` + `secret_key`
20. **alidns** — `access_key_id` + `access_key_secret`
21. **desec** — `token`
22. **transip** — `login`
23. **bunny** — `access_key`
24. **ionos** — `auth_api_token`
25. **namesilo** — `api_token`
26. **acmedns** — `server_url` + `username` + `password` + `subdomain`
27. **duckdns** — `api_token`
28. **he** (Hurricane Electric) — `api_key`
29. **infomaniak** — `api_token`
30. **luadns** — `email` + `api_key`
31. **cloudns** — `auth_id` (or `sub_auth_id`) + `auth_password`
32. **directadmin** — `server_url` + `user` + `login_key`
33. **loopia** — `username` + `password`
34. **glesys** — `project` + `api_key`
35. **huaweicloud** — `access_key_id` + `secret_access_key` + `region_id`
36. **acmeproxy** — `address` (+ optional `username`/`password`)
37. **dynv6** — `token`
38. **mythicbeasts** — `key_id` + `secret`
39. **simplydotcom** — `account_name` + `api_key`
40. **netcup** — `customer_number` + `api_key` + `api_password`
41. **dynu** — `api_token`
42. **gcore** — `api_key`
43. **all-inkl** — `kas_username` + `kas_password`
44. **autodns** — `username` + `password`
45. **bluecat** — `server_url` + `username` + `password` + `configuration_name` + `view_name`
46. **dnsupdate** — `addr` + optional `tsig`
47. **domainnameshop** — `api_token` + `api_secret`
48. **mijnhost** — `api_key`
49. **njalla** — `api_token`
50. **regfish** — `api_token`
51. **tecnocratica** — `api_token`
52. **unifi** — `api_key` + `base_url`
53. **websupport** — `api_key` + `api_secret`
54. **wedos** — `username` + `password`
55. **westcn** — `username` + `api_password`
56. **volcengine** — `access_key_id` + `access_key_secret`

## Usage

### Auto-detection
```bash
export AT3AM_DNS_CREDS=/etc/at3am/cloudflare.yaml
certbot certonly --manual \
  --manual-auth-hook    "at3am-hook manual-auth" \
  --manual-cleanup-hook "at3am-hook manual-cleanup" \
  -d example.com
```

### Explicit provider
```bash
export AT3AM_DNS_PROVIDER=route53
export AT3AM_DNS_CREDS=/etc/at3am/route53.yaml
certbot certonly --manual \
  --manual-auth-hook    "at3am-hook manual-auth" \
  --manual-cleanup-hook "at3am-hook manual-cleanup" \
  -d example.com
```

## Credential Files

Each provider has a YAML template. On first run, if the file doesn't exist, a commented template is created:

```yaml
provider: cloudflare

cloudflare:
  api_token: "your-scoped-api-token"
```

Protect the file:
```bash
sudo chmod 600 /etc/at3am/cloudflare.yaml
```

## License

**53 providers:** MIT License
**1 provider:** Apache 2.0 (dnsimple)

All licenses are compatible with at3am's Apache 2.0 license. See `LICENSES/` for full attribution.

