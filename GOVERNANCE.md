# Authorizer Governance

This document describes how the Authorizer project is governed. It is intended to be a living document that evolves as the community grows.

## Project Mission

Provide an open-source, self-hosted authentication and authorization platform that:

- Respects user data sovereignty.
- Works with any reasonable database backend.
- Implements modern, standards-compliant OAuth2/OIDC protocols.
- Is operable as a single Go binary.
- Stays approachable for individual developers and operable for enterprise teams.

## Roles

### Users

Anyone who uses Authorizer. Users contribute by filing bug reports, sharing experiences, requesting features, and helping each other in community channels.

### Contributors

Anyone who has contributed code, documentation, design, examples, translations, or any other improvement merged into an Authorizer repository. Contributors are listed automatically via Git history. There is no formal application process.

### Maintainers

Maintainers have write access to one or more Authorizer repositories and are responsible for the long-term health of the project. They:

- Review and merge pull requests.
- Triage issues.
- Cut releases.
- Set technical direction.
- Enforce the Code of Conduct.

The current maintainer list is in [MAINTAINERS.md](MAINTAINERS.md).

### Project Lead

While the project has a single lead maintainer, that person acts as the final arbiter on technical disputes that cannot be resolved by consensus. This role exists explicitly until the maintainer group expands; at that point this section will be replaced by collective decision-making.

## Becoming a Maintainer

A contributor may be invited to become a maintainer when they have demonstrated:

- **Sustained contribution** over a period of at least **3 months**.
- **Technical judgement** through high-quality reviews and code.
- **Community alignment** through respectful, constructive engagement.
- **Reliability** in following through on commitments.

The process:

1. An existing maintainer nominates the contributor by opening a PR adding them to `MAINTAINERS.md`.
2. The nomination is open for **2 weeks** of public comment.
3. The nomination is accepted if there is **consensus among existing maintainers** and no unaddressed objections from the community.
4. The new maintainer is granted write access and announced in the next release notes.

There is no fixed cap on maintainer count.

## Stepping Down

A maintainer may step down at any time by opening a PR moving themselves to the **Emeritus Maintainers** section of `MAINTAINERS.md`. Emeritus maintainers retain credit for their contributions but no longer have merge authority.

A maintainer may be moved to emeritus status by consensus of the other maintainers if they have been inactive for **6 months** and have not responded to outreach.

## Decision Making

Authorizer aims to make decisions by **lazy consensus**:

- A maintainer proposes a change.
- If no maintainer raises an objection within a reasonable time (**7 days** for routine work, **14 days** for architectural changes), the change moves forward.
- If objections are raised, the proposers and objectors work toward resolution in the issue or PR thread.
- If consensus cannot be reached, the project lead has the final say. This is a deliberate choice for the current single-maintainer phase and will be replaced with maintainer voting once there are three or more maintainers.

### Scope of Decision

| Decision Type | Who decides |
|---|---|
| Bug fixes, minor features, docs | Any maintainer |
| Breaking changes, new public APIs | Maintainer consensus |
| Major architectural shifts, license, governance changes | Maintainer consensus + 2-week community notice |
| Security fixes under embargo | Security maintainers (subset of all maintainers) |

## Conflict Resolution

1. Discuss in the relevant issue or PR.
2. Escalate to a private maintainer channel if discussion becomes unproductive.
3. The project lead has final authority. This authority transitions to maintainer voting once the maintainer group expands.

All participants are expected to follow the [Code of Conduct](CODE_OF_CONDUCT.md). Violations are handled per the CoC enforcement process.

## Release Management

- **Versioning**: [Semantic Versioning 2.0.0](https://semver.org/).
- **Cadence**: Patch releases as needed, minor releases approximately monthly, major releases approximately yearly.
- **Release notes**: Generated from PR titles + manual curation in `CHANGELOG.md`.
- **Release authority**: Any maintainer can cut a release after CI passes and maintainer consensus on inclusion.

## Code of Conduct

Authorizer adopts the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md). All participants in any Authorizer space — repositories, Discord, Slack, conferences, social media — are expected to follow it.

Report violations to **conduct@authorizer.dev**. Reports are reviewed by maintainers who are not involved in the incident. If a maintainer is the subject of a report, that maintainer recuses themselves.

## Intellectual Property

- All contributions are licensed under the project's license at the time of contribution. The project is currently distributed under the MIT License (see [LICENSE](LICENSE)). The project is evaluating a transition to the Apache License 2.0 as part of preparing a CNCF Sandbox application; any license change will follow the **architectural change** decision path described above and will require contributor consent before adoption.
- The **Authorizer** name and logo are owned by the project. The project is committed to neutral, community-oriented stewardship of these assets.
- Future ownership of trademark and assets may be transferred to a neutral foundation. Any such transfer will be announced publicly and discussed openly before completion.

## Modifying This Document

Changes to this governance document follow the **architectural change** decision path: maintainer consensus plus a **2-week community notice period** on a public PR.

---

*Last updated: 2026-05-20*
