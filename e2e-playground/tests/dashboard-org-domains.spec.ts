// e2e-playground/tests/dashboard-org-domains.spec.ts
import { test, expect, Page } from '@playwright/test';
import crypto from 'node:crypto';
import { createOrg } from '../fixtures/adminClient';

// PR #707 added web/dashboard/src/components/OrgDomains.tsx - the admin UI
// for managing an org's verified email domains (home-realm-discovery
// routing). Prior specs (magic-link, mfa-routing-matrix, etc.) only ever
// seed verified domains via the `_add_verified_org_domain` GraphQL bypass
// (fixtures/adminClient.ts addVerifiedDomain) for test setup - this is the
// first spec exercising the actual dashboard UI.

const ADMIN_SECRET = process.env.AUTHORIZER_ADMIN_SECRET || 'e2e-admin-secret';

// randomDomain returns a syntactically-valid, never-colliding, never-resolvable
// test domain. Hyphens are stripped from the UUID and a letter prefix added so
// the label can never violate IDNA's no-leading/trailing-hyphen or
// no-double-hyphen rules (internal/service/org_domain_util.go normalizeDomain
// calls idna.Lookup.ToASCII). "example.com" is a safe, non-registrable-looking
// suffix that is neither a public suffix on its own (so guardVerifiableDomain's
// public-suffix guard doesn't reject it) nor in the consumer-domain blocklist.
function randomDomain(): string {
  return `d${crypto.randomUUID().replace(/-/g, '')}.example.com`;
}

// loginAsAdmin drives the real dashboard login form (web/dashboard/src/pages/Auth.tsx).
// isOnboardingCompleted is hardcoded true server-side (internal/http_handlers/dashboard.go),
// so hasAdminSecret() always resolves to the "Login" (not "Sign up") branch -
// confirmed live below, not just by reading the source.
async function loginAsAdmin(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/dashboard/`);
  await page.locator('#admin-secret').fill(ADMIN_SECRET);
  await page.getByRole('button', { name: 'Login' }).click();
  // Auth.tsx navigates to '/' (-> /dashboard/ with the router's basename) on
  // successful login; wait for that instead of a fixed sleep.
  await page.waitForURL((url) => /^\/dashboard\/?$/.test(url.pathname), { timeout: 10_000 });

  // Every other spec in this suite only ever does client-side routing after
  // login (no full navigation), so this race has never surfaced before: in
  // this headless/Docker Chromium, the login response's Set-Cookie can land
  // in the browser's cookie jar a beat after the fetch's JS promise (and
  // waitForURL) resolves. Every test in this file immediately follows login
  // with a real page.goto() to a deep dashboard route, which starts a fresh
  // network request - if that request races the cookie-jar write, it goes
  // out unauthenticated and the fresh page mounts logged out. Wait for the
  // cookie to actually be in the jar before navigating away.
  await expect
    .poll(async () => (await page.context().cookies()).some((c) => c.name === 'authorizer-admin'), {
      timeout: 5_000,
    })
    .toBe(true);
}

test.describe('Dashboard — Org verified domains', () => {
  test('add a domain without DNS verification (super-admin), then delete it', async ({ page, baseURL }) => {
    const org = await createOrg(`org-domains-${crypto.randomUUID()}`);
    const domain = randomDomain();

    await loginAsAdmin(page, baseURL!);
    await page.goto(`${baseURL}/dashboard/identity/organizations/${org.id}`);

    await page.getByRole('button', { name: 'Add Domain' }).click();
    // Domain-entry step is the dialog's immediate state (challenge === null
    // initially) - #org-domain-input should already be visible, no
    // intermediate step.
    const domainInput = page.locator('#org-domain-input');
    await expect(domainInput).toBeVisible();
    await domainInput.fill(domain);
    await page.getByRole('button', { name: 'Add without DNS verification (super-admin)' }).click();

    // Dialog closes and the table re-fetches on success.
    await expect(page.getByRole('dialog')).toBeHidden();
    const row = page.getByRole('row', { name: domain });
    await expect(row).toBeVisible();
    // AddVerifiedOrgDomain marks the row verified immediately - the badge
    // renders a formatted verified_at date (dayjs "MMM D, YYYY"), not a dash.
    await expect(row.getByText(/^[A-Z][a-z]{2} \d{1,2}, \d{4}$/)).toBeVisible();

    await row.getByRole('button', { name: 'Delete' }).click();
    const confirmDialog = page.getByRole('dialog');
    await expect(confirmDialog.getByText('Delete Verified Domain')).toBeVisible();
    await expect(confirmDialog.getByText(domain)).toBeVisible();
    await confirmDialog.getByRole('button', { name: 'Delete' }).click();

    await expect(page.getByRole('row', { name: domain })).toHaveCount(0);
    await expect(page.getByText('No verified domains yet.')).toBeVisible();
  });

  test('DNS challenge flow shows the TXT record and a not-verified hint on failed verification', async ({
    page,
    baseURL,
  }) => {
    const org = await createOrg(`org-domains-dns-${crypto.randomUUID()}`);
    const domain = randomDomain();

    await loginAsAdmin(page, baseURL!);
    await page.goto(`${baseURL}/dashboard/identity/organizations/${org.id}`);

    await page.getByRole('button', { name: 'Add Domain' }).click();
    await page.locator('#org-domain-input').fill(domain);
    await page.getByRole('button', { name: 'Request DNS Challenge' }).click();

    // Challenge step: the TXT record to publish is shown via two CopyField
    // rows (internal/service/org_domain_dns.go challengeRecordName/Value).
    await expect(page.getByText('Record name')).toBeVisible();
    await expect(page.getByText('Record value')).toBeVisible();
    await expect(page.getByText(`_authorizer-challenge.${domain}`)).toBeVisible();
    await expect(page.getByText(/^authorizer-domain-verification=/)).toBeVisible();

    // The test domain has no real DNS records, so verification must fail with
    // the retryable "DNS not propagated yet" hint (VerifyOrgDomain returns
    // "dns verification failed: ..." for both NXDOMAIN and a missing/mismatched
    // TXT record - either way OrgDomains.tsx surfaces this exact hint text).
    await page.getByRole('button', { name: "I've published this record — Verify" }).click();
    await expect(
      page.getByText(
        'Not verified yet. DNS changes can take a few minutes to propagate — leave the record in place and try again shortly.'
      )
    ).toBeVisible({ timeout: 10_000 });

    // The challenge stays on screen (not discarded) so the tenant can retry.
    await expect(page.getByText(`_authorizer-challenge.${domain}`)).toBeVisible();
  });
});
