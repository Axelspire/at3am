# Contributing to at3am

Thank you for your interest in contributing to at3am! This document outlines the rules and expectations for contributions.

## Contributor License Agreement (CLA)

**All contributors must sign a CLA before any contribution can be merged.**

By signing the CLA, you:

1. **Grant a patent license** — You grant Daniel Cvrcek and the at3am project a perpetual, worldwide, non-exclusive, royalty-free, irrevocable patent license covering any patent claims licensable by you that are necessarily infringed by your contribution.
2. **Confirm original work** — You represent that your contribution is your original work (or that you have the right to submit it) and that you have the legal authority to grant the above license.
3. **Allow relicensing** — You acknowledge that your contribution may be incorporated into at3am Enterprise or other derivative works under different license terms, and you grant the project maintainer the right to sublicense your contribution accordingly.

To sign the CLA, include the following statement in your first pull request:

```
I have read and agree to the at3am Contributor License Agreement.
I confirm that my contribution is my original work and I grant the
patent and copyright licenses described in CONTRIBUTING.md.

Signed: [Your Full Name] <[your@email.com]>
Date: [YYYY-MM-DD]
```

### Why a CLA?

The CLA protects both contributors and the project. It ensures:
- Clear patent grants that protect all users of at3am
- The ability to maintain both the free (Apache 2.0) and enterprise editions
- Legal clarity for organisations adopting at3am in production

## Code Style

### Go Conventions

- Follow standard Go conventions (`gofmt`, `go vet`, `golint`)
- Use `go fmt` before committing — CI will reject unformatted code
- Keep functions focused and under 50 lines where practical
- Use meaningful variable and function names
- Add doc comments to all exported types, functions, and methods
- Avoid package-level mutable state

### Project Structure

```
cmd/at3am/          — CLI entry point
cmd/at3am-hook/     — Certbot hook wrapper
internal/           — Internal packages (not importable by external code)
  config/           — Configuration types and profiles
  resolver/         — DNS resolver pool and querying
  confidence/       — MPIC-aware confidence scoring engine
  ttl/              — TTL analysis
  diagnostics/      — Diagnostic explanations
  output/           — Output formatting and Prometheus metrics
  mock/             — Mock DNS scenarios for testing
  wait/             — Core polling loop orchestrator
```

### Commit Messages

- Use the imperative mood: "Add feature" not "Added feature"
- Keep the first line under 72 characters
- Reference issue numbers where applicable: `Fix #42: handle nil response`

## Testing Requirements

**Every pull request must include tests.** We maintain a minimum of 95% code coverage.

### Rules

1. **All new code must have tests** — No exceptions. If you add a function, add tests for it.
2. **All modified code must have updated tests** — If you change behaviour, update the corresponding tests.
3. **Tests must pass** — `go test ./...` must exit 0 before submitting.
4. **Coverage must not decrease** — Run `go test -coverprofile=coverage.out ./...` and verify with `go tool cover -func=coverage.out`.
5. **Use table-driven tests** where multiple input/output combinations are tested.
6. **Use the mock package** (`internal/mock`) for DNS-dependent tests — do not make real DNS queries in unit tests.
7. **Test edge cases** — nil inputs, empty slices, error paths, boundary values.

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run with race detector
go test -race ./...

# Run a specific package
go test ./internal/confidence/
```

## Pull Request Process

1. **Fork the repository** and create a feature branch from `main`
2. **Sign the CLA** (see above) in your first PR
3. **Write your code** following the style guidelines
4. **Write tests** meeting the coverage requirements
5. **Run the full test suite** locally: `go test -race ./...`
6. **Run vet and fmt**: `go vet ./... && gofmt -l .`
7. **Submit a pull request** with a clear description of the change
8. **Respond to review feedback** promptly

## Scope

at3am is intentionally focused:

- **In scope**: DNS propagation monitoring, confidence scoring, output formatting, CLI interface, mock testing support
- **Out of scope**: ACME client functionality, DNS record creation/deletion, certificate management, persistent storage

Features outside this scope belong in [at3am Enterprise](../ENTERPRISE.md).

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include reproduction steps, expected vs actual behaviour, and your environment (OS, Go version, at3am version)
- For security vulnerabilities, email security@axelspire.com instead of opening a public issue

## License

By contributing to at3am, you agree that your contributions will be licensed under the [Apache License 2.0](../LICENSE).

