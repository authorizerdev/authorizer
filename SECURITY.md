# Security Policy

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.**

We use [GitHub Security Advisories](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability) for coordinated disclosure. To report a vulnerability:

1. Go to the **Security** tab of this repository.
2. Click **Report a vulnerability**.
3. Fill in the form with as much detail as you can:
   - Type of issue (auth bypass, token leak, SQL injection, SSRF, etc.).
   - Affected component (HTTP endpoint, GraphQL operation, OAuth flow, storage backend, etc.).
   - Affected versions.
   - Step-by-step reproduction.
   - Proof-of-concept or exploit code, if available.
   - Impact assessment (what an attacker can achieve).

<!-- TODO: set up security@authorizer.dev mailbox and update this address. -->
If GitHub Security Advisories is unavailable, email **lakhan.m.samani@gmail.com** with the same information.

## Response Process

| Stage | Target |
|---|---|
| Initial acknowledgement | within **72 hours** |
| Triage decision (accepted / not accepted) | within **7 days** |
| Fix released for accepted critical/high-severity issues | within **30 days** of triage |
| Public disclosure | coordinated with reporter, typically **90 days** after report or sooner if fix is released |

We follow [Coordinated Vulnerability Disclosure](https://en.wikipedia.org/wiki/Coordinated_vulnerability_disclosure). We will:

- Confirm receipt of your report.
- Keep you informed of progress toward a fix.
- Credit you in the security advisory unless you request anonymity.
- Coordinate the public disclosure date with you.

## Supported Versions

Security fixes are released against:

- The **current major version** (currently `2.x`).
- The **previous major version** for **6 months** after a new major release.

| Version | Supported          |
|---------|--------------------|
| 2.x     | :white_check_mark: |
| 1.x     | :x: (EOL)          |

## Scope

In scope for security reports:

- The main `authorizer` server.
- Official SDKs: `authorizer-js`, `authorizer-react`, `authorizer-vue`, `authorizer-svelte`, `authorizer-go`, `authorizer-flutter-sdk`.
- The official Helm chart in `authorizer-helm-chart`.
- Documentation that, if followed, would lead to an insecure deployment.

Out of scope:

- Third-party plugins or forks.
- Misconfiguration in user-controlled deployment infrastructure (unless caused by misleading defaults or documentation).
- Vulnerabilities in dependencies that are not exploitable in Authorizer's usage of them.
- Self-XSS or social-engineering attacks.

## Safe Harbour

We will not pursue legal action against researchers who:

- Make a good-faith effort to comply with this policy.
- Report vulnerabilities only to us through the channels above.
- Do not access, modify, or destroy data that does not belong to them.
- Do not perform attacks against production deployments other than their own test environment.

## Bug Bounty

Authorizer does not currently operate a paid bug bounty program. We are grateful for responsible disclosure and credit researchers publicly with their consent.

## Encryption

<!-- TODO: set up security@authorizer.dev mailbox and update this address. -->
If you wish to send encrypted reports, request a PGP key by emailing lakhan.m.samani@gmail.com.

---

*Last updated: 2026-05-20*
