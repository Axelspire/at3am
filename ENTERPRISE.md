# at3am Enterprise: Full ACME Client for Large-Scale Deployments

at3am (the free DNS-01 oracle) solves propagation pain for any ACME client. For enterprises managing certificates with advanced security, compliance, and automation needs, at3am Enterprise builds on this core with a production-grade ACME client.

## Feature Comparison

### DNS-01 Propagation Oracle (Core)

| Feature                          | at3am (Free) | at3am Enterprise (Paid) |
|----------------------------------|--------------|--------------------------|
| Intelligent DNS-01 propagation oracle | ✓           | ✓                       |
| Multi-resolver polling & MPIC confidence scoring | ✓           | ✓                       |
| TTL analyser & diagnostics       | ✓           | ✓                       |
| Mock mode for CI/CD testing      | ✓           | ✓                       |
| Prometheus metrics & structured logging | ✓           | ✓                       |
| DNSSEC validation (DO/AD bit)    | ✓           | ✓                       |
| DNS-PERSIST-01 (`--challenge-type persist`) | ✓           | ✓                       |
| **54 DNS providers** via libdns (Certbot hook) | ✓           | ✓                       |
| Full record lifecycle (create → wait → delete) | ✓           | ✓                       |
| NS-based provider autodetection  | ✓           | ✓                       |
| Distributed multi-vantage polling | ✗           | ✓                       |
| Provider-specific propagation optimization | ✗           | ✓                       |
| High-concurrency batch polling   | ✗           | ✓                       |
| Adaptive geo-weighted resolvers  | ✗           | ✓                       |
| Real-time auditing and compliance logs | ✗           | ✓                       |
| Proactive alerting and monitoring | ✗           | ✓                       |

### Full ACME Client & Enterprise Features

| Feature                          | at3am (Free) | at3am Enterprise (Paid) |
|----------------------------------|--------------|--------------------------|
| Full RFC 8555 ACME client (issue/renew) | ✗           | ✓                       |
| HSM / PKCS#11 support (AWS, Azure, Thales, etc.) | ✗           | ✓                       |
| PQC hybrid issuance (ML-KEM + classical) | ✗           | ✓                       |
| Multi-tenant RBAC & YAML policies | ✗           | ✓                       |
| Zero-downtime rotation agent (blue/green, health gates) | ✗           | ✓                       |
| Central dashboard + REST/gRPC API | ✗           | ✓                       |
| Audit & compliance reporting (ISO 27001, SOC 2, CNSA 2.0) | ✗           | ✓                       |
| Private CA integration (Vault PKI, step-ca, EJBCA) | ✗           | ✓                       |
| GitOps / Terraform / Ansible support | ✗           | ✓                       |
| High-availability clustered daemon | ✗           | ✓                       |
| Premium support SLA & managed SaaS option | ✗           | ✓                       |

## Why Enterprise?

If you're dealing with large fleets, hybrid environments, or regulatory requirements, at3am Enterprise eliminates random failures while adding the controls you need. It's the reliable upgrade for teams outgrowing certbot or acme.sh.

## Pricing & Access

- **Annual subscription**: Simple pricing structure
- **Get started**: Email sales@yourdomain.com or visit [yourwebsite.com/at3am-enterprise] for a demo.

For questions or feature requests on the free version, open an issue here.

