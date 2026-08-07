# Email verification contract

Authorizer resolves a local account from an email address in several flows —
password signup, magic link, and every social/enterprise federated login. This
document states when an address counts as **verified**, why federated logins are
held to that bar, and what operators need to configure.

> **Behaviour change in 2.4.0.** A social login whose provider does not attest
> the email address is now refused. See [Upgrading](#upgrading) for what to
> configure if this affects your deployment.

## Why an address must be attested

OAuth 2.0 carries no identity claims at all. OpenID Connect added `email` — and
alongside it a **separate `email_verified` boolean**, precisely because `email`
on its own proves nothing about who controls the mailbox.

That distinction matters here because the email address is what selects the
local account. `GetUserByEmail` decides signup-vs-login, and the login branch
merges the incoming federated identity into whatever account already holds the
address. If an attacker can make a provider assert an address they do not own,
they land in that account's session.

This is the **nOAuth** attack class. Microsoft Entra is the sharp case:

- Entra **v2 ID tokens carry no `email_verified` claim at all**.
- Entra's `email` is a *mutable, unverified* directory attribute — any tenant
  admin can set it to any string, including someone else's address.
- The multi-tenant endpoints (`common`, `organizations`, `consumers`) sign with
  Microsoft's **global** keys, so a token minted in a free attacker-owned tenant
  has a valid signature and a valid `aud`. Only the tenant distinguishes it.

So: register a free Entra tenant, set a user's `email` to `victim@example.com`,
click "Login with Microsoft", and without this contract you are in the victim's
account.

## The three connection classes

Authorizer treats connections the same way Auth0 does:

| Connection class | Is the address attested? | Behaviour |
|---|---|---|
| **Database** (password signup) | No — nobody has vouched for it | `email_verified` is `false` until the user clicks the verification link Authorizer mails them (requires `--enable-email-verification=true`; with verification disabled the address is marked verified at signup) |
| **Social** (Google, Apple, GitHub, …) | Usually yes — the provider vouches | The provider's own signal is imported directly; no separate verification round-trip |
| **Enterprise / Azure AD / OIDC** | **Not guaranteed** — enterprise directories do not promise it | Requires an explicit trust decision from the operator (see below) |

## Per-provider signal

Each provider is read from the signal that provider actually emits. There is no
single claim that works everywhere, and treating one provider's silence as
another's "true" is exactly what created the vulnerability.

| Provider | Signal | Notes |
|---|---|---|
| Google | `email_verified` (ID token) | Standard OIDC |
| Apple | `email_verified` (ID token) | Documented as "a string or Boolean value" — both forms accepted |
| Twitch | `email_verified` (ID token) | Standard OIDC |
| LinkedIn | `email_verified` (userinfo) | Both bool and quoted-string forms accepted |
| Microsoft | `xms_edov`, **or** a trusted tenant | No `email_verified` exists on Entra v2 — see below |
| GitHub | the address is verified by construction | The public `/user` email and the `/user/emails` fallback are both filtered to verified addresses |
| Discord | `verified` (`/users/@me`) | Synthetic fallback address is trusted by construction |
| Facebook | trusted | Graph API only returns a `email` for an account with a confirmed primary address |
| Twitter/X | trusted | `confirmed_email` is confirmed by definition; synthetic fallback is trusted by construction |
| Roblox | `email_verified` (userinfo) | Synthetic fallback is trusted by construction |

**"Trusted by construction"** means the synthetic fallback addresses
(`discord-<id>@discord.oauth.internal`, and the Twitter/Roblox equivalents) live
on reserved, non-routable domains keyed by the provider's permanent user id. No
real mailbox can occupy them, so they can never collide with an address someone
could otherwise prove they own.

## Microsoft / Entra specifics

Two independent things make a Microsoft address trustworthy. Either is enough:

1. **`xms_edov`** — "email domain owner verified", Microsoft's own attestation
   that the token's tenant owns the address's domain. It is an *optional claim*;
   enable it in the app registration's token configuration.
2. **A tenant the operator trusts** — the address can then only have come from a
   directory you already control or vouch for.

A tenant is trusted when either:

- `--microsoft-tenant-id` is a **specific** tenant (a GUID or verified domain
  name) rather than one of the multi-tenant aliases `common`, `organizations`,
  `consumers`; or
- `--microsoft-allowed-tenants` is set and the token's `tid` is in it.

Independently of the trust decision, every Microsoft ID token must satisfy:

- `tid` is present;
- `iss` equals `https://login.microsoftonline.com/<tid>/v2.0` — a token may not
  claim one tenant in `iss` and another in `tid`;
- if a specific tenant is pinned, `tid` matches it;
- if an allowlist is configured, `tid` is in it.

`--microsoft-tenant-id` defaults to `common`. A deployment left on that default
with no allowlist and no `xms_edov` will now **refuse** Microsoft logins rather
than accept an unattested address. That is deliberate: it is exactly the
exploitable configuration.

## Configuration

```
--microsoft-tenant-id=<guid|domain|common|organizations|consumers>
    Default: common. A specific value pins the tenant and makes its addresses
    trustworthy.

--microsoft-allowed-tenants=<tid>,<tid>
    Entra tenant IDs permitted to sign in when --microsoft-tenant-id is a
    multi-tenant alias. Empty allows any tenant, but an untrusted tenant's
    email will not link to an existing account.

--enable-email-verification=true
    Database connections only: require the user to click a mailed link before
    the address counts as verified.

--oauth-allow-unverified-provider-email=false
    Compatibility escape hatch for 2.3.x upgrades. See below.
```

## Compatibility mode

`--oauth-allow-unverified-provider-email` exists so a deployment upgrading from
2.3.x is not locked out the moment it restarts. It is deliberately **not** a
plain "turn the check off" switch — that would restore the vulnerability
verbatim.

With the flag set, an unattested address may:

- create a **brand-new** account — it selects nobody, so it harms nobody; or
- return to an account **this same provider already owns** — a returning user.

It may **never** merge into an account another credential owns. That one
restriction removes the entire cross-credential takeover: an Entra tenant cannot
reach a password account, a Google account, or any other provider's account,
which is every practical form of the attack.

**Residual risk it does not cover**, and the reason this mode is temporary: two
principals of the *same* unattested provider — two Entra tenants both asserting
one address — can still collide. Pinning `--microsoft-tenant-id` or setting
`--microsoft-allowed-tenants` closes that, and is the actual fix. The server logs
a warning on every boot while the flag is set.

### Knock-on effects of compatibility mode

Two behaviours change to keep compatibility mode honest:

- **`email_verified` in our own database reflects reality.** A social signup used
  to write `email_verified=true` unconditionally. It now does so only when the
  provider attested the address, because downstream consumers trust that column
  (SAML IdP issuance, for one, refuses to assert an unverified email as the
  Subject NameID). An account created in compatibility mode from an unattested
  address is stored as unverified — which is the truth.
- **The pre-hijack guard is scoped to other credentials.** That guard deletes an
  *unverified* pre-existing account rather than linking to it. It now fires only
  when the account was created by a *different* method. An unverified account
  this same provider already owns is not a squatter, it is the same principal's
  own account — deleting it would recreate the account on every login, silently
  dropping its id, roles and org memberships each time. The squatter case
  (attacker pre-registers `victim@example.com` by password, victim later signs in
  via Google) is unaffected.

## The pre-hijack delete is now bounded

Independent of the verification contract, that guard's deletion is now limited
to accounts that hold no state.

The replacement account is created with a fresh id, so deleting a real account
still **destroys** everything that account owned — its FGA grants, org
memberships, enrolled authenticators, passkeys and federated identities. The
rows are no longer *orphaned* (see below), but they are gone, and the new
account inherits none of them.

A squatter's account is empty by definition — created to intercept an address and
never used. So before deleting, the callback checks for org memberships,
passkeys, enrolled authenticators and FGA grants. If it finds any, it refuses the
login instead:

```json
{
  "error": "email_already_registered",
  "error_description": "An unverified account already exists for this email address. Verify it or sign in with the method that created it."
}
```

Refusing is recoverable; deleting is not. A lookup fault counts as "has state"
for the same reason. The delete, when it does happen, now emits an
`oauth.unverified_account_replaced` audit event carrying the destroyed user id.

> **Resolved ([#747](https://github.com/authorizerdev/authorizer/issues/747)):**
> `StorageProvider.DeleteUser` used to cascade to sessions and nothing else, so
> every hard delete — including the admin `_delete_user` path — left orphaned
> rows pointing at a dead user id. The worst of them, an orphaned
> `authorizer_federated_identities` row, was a permanent SSO lockout:
> `jitProvisionFederatedUser` resolves a returning principal through it, fails
> closed when the user is gone, and the unique `(org_id, issuer, subject)` triple
> blocks re-provisioning.
>
> The cascade now covers every collection in `schemas.UserOwnedCollections`
> (sessions, federated identities, org memberships, authenticators, passkeys,
> session tokens, MFA sessions) on all six backends, and the admin delete
> additionally purges the user's FGA tuples. Soft deletes (`deactivate_account`,
> revoke access) do **not** cascade — the account is meant to come back.
>
> The `accountHasState` bound above is kept regardless: refusing is still
> recoverable and deleting still is not.

## How a user verifies their address

**Signup already mails a verification link.** With `--enable-email-verification`,
`signup` creates the verification request and sends the mail before returning
"Verification email has been sent. Please check your inbox". Clicking that link
is the normal path and needs nothing else.

The link is valid for 30 minutes. If it expires or never arrives:

| Route | Who drives it | Notes |
|---|---|---|
| **`resend_verify_email`** | the user | The primary recovery. Mints a fresh link for the same address. |
| **Password login** | the user | An unverified account's password login emails an OTP instead; verifying that OTP marks the address verified (`verify_otp.go`). |
| **`_update_user { email_verified: true }`** | an admin | The operator escape hatch when the user genuinely cannot receive mail. |
| Forgot password | the user | Completing a token reset also verifies the address — a side effect of proving mailbox control, not the route to reach for. |

Two fixes make the table above actually hold:

- **`resend_verify_email` no longer dead-ends.** It required a *pending*
  verification request. Expired rows are still returned by
  `GetVerificationRequestByEmail` (there is no expiry filter), so the usual
  expired-link case already worked — but a password-login attempt **purges** the
  expired row (`login.go`), and after that the endpoint silently did nothing.
  It now mints a fresh request when none exists, gated on the address actually
  being unverified so it cannot be used as an open mailer.
- **Admin force-verify works standalone.** `_update_user`'s "at least one param"
  gate omitted `email_verified` and `phone_number_verified` even though both are
  applied further down, so a call setting only `email_verified` was rejected
  unless padded with an unrelated field.

### How long a verification link is valid

**30 minutes.** `CreateVerificationToken` mints the JWT with `exp` 30 minutes out
(`internal/token/verification_token.go`), and the stored row carries a matching
`expires_at`.

Expiry is enforced by **JWT validation**, not by the row: redemption parses and
validates the token before anything else, so an expired link is refused even if
the row is still present. The `expires_at` column is used elsewhere — the login
path reads it to decide whether a *pending* verification should block a sign-in
or be cleared and replaced.

A link also stops working **before** its 30 minutes if a newer one is issued:
each request rotates the nonce, and redemption checks the token's nonce against
the stored row. So the most recent link is always the only valid one — requesting
a fresh link invalidates the previous one immediately.

### The link expired — what now

Nothing here needs an administrator.

1. **Request a new link.** `resend_verify_email` with the same `identifier`
   (`basic_auth_signup` for a normal signup) mints a fresh request and mails it.
   This works whether or not a pending request still exists — it will create one
   if the old row is gone.
2. **Or just log in with your password.** An unverified account's password login
   emails a one-time code instead of a session; entering that code verifies the
   address (`verify_otp.go`). This is why an expired link is rarely noticed by
   password users.
3. **Operator fallback.** An admin can force-verify from the dashboard
   (**Mark Email Verified**) or via `_update_user { email_verified: true }`.
   Prefer **Resend Verification Email** where possible — it has the user prove
   control rather than asserting it on their behalf.

The response to a resend is deliberately generic ("if a verification is pending
…") and identical whether or not the address exists, so the endpoint cannot be
used to test which addresses are registered.

### Hard requirement: email verification needs a working email service

`--enable-email-verification=true` with no SMTP configured is now a **fatal
startup error**. Every route in the table above terminates at the same mailbox,
so without a mail path a user is created unverified and can never recover — and
an unverified account also blocks a federated login for that address. Configure
`--smtp-host`, `--smtp-port` and `--smtp-sender-email`, or turn email
verification off.

### This is not the same as Auth0's post-login email check

A post-login Action like:

```js
exports.onExecutePostLogin = async (event, api) => {
  if (!event.user.email_verified) {
    api.access.deny('Please verify your email address before logging in.');
  }
};
```

is a **login policy**: "should this user, whoever they are, be let in before
confirming their own address?" It is reasonably opt-in, and Authorizer's
equivalent is `--enable-email-verification`.

What this document describes is **identity resolution**: "which local account
does this federated assertion refer to?" Getting that wrong does not inconvenience
the legitimate user — it hands their account to somebody else. Which is why the
default is secure and the escape hatch is narrowed rather than total.

## What a refusal looks like

The callback returns `400` before any local account lookup, so no account is
created and no existing account is touched:

```json
{
  "error": "email_not_verified",
  "error_description": "The identity provider did not confirm that you own this email address."
}
```

A `oauth_email_unverified` security metric and an audit event are recorded.

## Upgrading

Most deployments need no change — Google, Apple, GitHub, Discord, Facebook,
LinkedIn, Twitch, Twitter and Roblox all supply a signal already.

**If you use Microsoft login**, pick one:

- pin `--microsoft-tenant-id` to your tenant (single-tenant deployments — the
  common case, and the best option);
- set `--microsoft-allowed-tenants` to the tenants you serve (multi-tenant SaaS);
- enable the `xms_edov` optional claim in your Entra app registration.

If you need more time, set `--oauth-allow-unverified-provider-email=true` as a
stopgap — existing users keep working and cross-credential takeover stays
blocked. It is not a substitute for one of the three fixes above; see
[Compatibility mode](#compatibility-mode).

## Tests

| What | Where |
|---|---|
| Claim decoding, tenant validation, `xms_edov` | `internal/http_handlers/oauth_noauth_test.go` |
| Social vouches / enterprise refused / nOAuth takeover attempt | `e2e-playground/tests/social/email-verification-contract.spec.ts` |
| Database connection stays unverified until the link is clicked | `e2e-playground/tests/email-verification-database.spec.ts` |
| `resend_verify_email` recovery, and that it is not an open mailer | `e2e-playground/tests/email-verification-database.spec.ts` |
| The rendered web/app journey: signup form → "check your inbox" → click link | `e2e-playground/tests/email-verification-ui.spec.ts` |
| Purpose binding, resend, and reset-verifies-email | `internal/integration_tests/verification_token_purpose_test.go` |

## Operator actions in the dashboard

The Users table (`web/dashboard`) exposes both operator routes per user, shown
only when the relevant identifier is actually unverified:

- **Mark Email Verified** — asserts the address is good without mailing anything.
  Sends only `email_verified`; deliberately not `email`, since that param drives
  the change-address flow in `_update_user` (which clears verification and mails
  a new link).
- **Resend Verification Email** — mails a fresh link so the user proves it
  themselves. Preferred when the admin has no independent reason to trust the
  address.
- **Mark Phone Verified** — the phone equivalent.

These were previously a single "Verify User" item that appeared only when *both*
the email and the phone were unverified, so a user with a verified phone and an
unverified email had no path at all.
