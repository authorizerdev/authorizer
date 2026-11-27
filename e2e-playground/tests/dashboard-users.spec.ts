// e2e-playground/tests/dashboard-users.spec.ts
//
// Guards #927 (Users page: created_at sort + email-verified filter) and #931
// (--organization-name / --organization-logo on the admin login page and
// sidebar), through the real dashboard UI.
//
// Runs in the `email-verify` project against authorizer-email-verify: its
// signups start unverified (--enable-email-verification), so verified and
// unverified users can both be seeded, and it carries the custom branding
// flags (docker-compose.yml).
import { test, expect, Page, APIRequestContext } from '@playwright/test';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';
import { getUserIdByEmail, verifyUserEmail } from '../fixtures/adminClient';

const ADMIN_SECRET = process.env.AUTHORIZER_ADMIN_SECRET || 'e2e-admin-secret';
const PASSWORD = 'Str0ngPassw0rd!';
// Must match authorizer-email-verify's flags in docker-compose.yml.
const ORG_NAME = 'Acme E2E';
const ORG_LOGO = 'https://example.invalid/acme-e2e-logo.png';

async function loginAsAdmin(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/dashboard/`);
  await page.locator('#admin-secret').fill(ADMIN_SECRET);
  await page.getByRole('button', { name: 'Login' }).click();
  await page.waitForURL((url) => /^\/dashboard\/?$/.test(url.pathname));
  // See tests/dashboard-org-domains.spec.ts: the admin cookie can land in the
  // jar a beat after the login promise resolves; wait for it before a full
  // navigation.
  await expect
    .poll(async () => (await page.context().cookies()).some((c) => c.name === 'authorizer-admin'))
    .toBe(true);
}

async function signup(request: APIRequestContext, baseURL: string, email: string) {
  await graphql(request, baseURL, `mutation ($params: SignUpRequest!) { signup(params: $params) { message } }`, {
    params: { email, password: PASSWORD, confirm_password: PASSWORD },
  });
}

// Emails in table order, as rendered (first line of the first cell).
async function renderedEmails(page: Page, marker: string): Promise<string[]> {
  const cells = page.locator('tbody tr td:first-child > div:first-child');
  return (await cells.allTextContents()).map((t) => t.trim()).filter((t) => t.includes(marker));
}

test.describe('Dashboard — branding (#931)', () => {
  test('admin login page and sidebar show --organization-name / --organization-logo', async ({ page, baseURL }) => {
    await page.goto(`${baseURL}/dashboard/`);
    const loginLogo = page.getByRole('img', { name: `${ORG_NAME} logo` });
    await expect(loginLogo).toHaveAttribute('src', ORG_LOGO);
    await expect(page.getByText(ORG_NAME, { exact: true })).toBeVisible();
    await expect(page.getByText('AUTHORIZER', { exact: true })).toHaveCount(0);

    await loginAsAdmin(page, baseURL!);
    const sidebarLogo = page.getByRole('img', { name: `${ORG_NAME} logo` }).first();
    await expect(sidebarLogo).toHaveAttribute('src', ORG_LOGO);
    await expect(page.getByText(ORG_NAME, { exact: true }).first()).toBeVisible();
  });
});

test.describe('Dashboard — Users sort and filter (#927)', () => {
  test('newest/oldest order, email-verified filter, filter + search, header toggle', async ({ page, request, baseURL }) => {
    // One unique marker scopes every assertion to this test's users, even
    // though other specs share the instance's user table.
    const marker = `dusr${crypto.randomUUID().replace(/-/g, '').slice(0, 12)}`;
    const emails = ['a', 'b', 'c'].map((s) => `${marker}-${s}@example.com`);
    for (let i = 0; i < emails.length; i++) {
      // created_at has one-second resolution; space the signups past it so
      // the order is defined rather than tied.
      if (i > 0) await new Promise((r) => setTimeout(r, 1100));
      await signup(request, baseURL!, emails[i]);
    }
    // b verified; a and c not.
    await verifyUserEmail(await getUserIdByEmail(emails[1], baseURL!), { givenName: 'Dash', familyName: 'Verified' }, baseURL!);

    const newestFirst = [emails[2], emails[1], emails[0]];
    const oldestFirst = [emails[0], emails[1], emails[2]];

    await loginAsAdmin(page, baseURL!);
    await page.goto(`${baseURL}/dashboard/users`);
    await page.getByPlaceholder('Search by email, name, or ID...').fill(marker);

    // Default: newest first.
    await expect.poll(() => renderedEmails(page, marker)).toEqual(newestFirst);

    const sort = page.getByLabel('Sort by created date');
    await sort.selectOption('ASC');
    await expect.poll(() => renderedEmails(page, marker)).toEqual(oldestFirst);
    await sort.selectOption('DESC');
    await expect.poll(() => renderedEmails(page, marker)).toEqual(newestFirst);

    // Filter combined with the search.
    const filter = page.getByLabel('Filter by email verification');
    await filter.selectOption('verified');
    await expect.poll(() => renderedEmails(page, marker)).toEqual([emails[1]]);
    await filter.selectOption('unverified');
    await expect.poll(() => renderedEmails(page, marker)).toEqual([emails[2], emails[0]]);
    await filter.selectOption('all');
    await expect.poll(() => renderedEmails(page, marker)).toEqual(newestFirst);

    // The Created At header toggles the order and keeps the selector in sync.
    const header = page.getByRole('button', { name: 'Created At' });
    await header.click();
    await expect.poll(() => renderedEmails(page, marker)).toEqual(oldestFirst);
    await expect(sort).toHaveValue('ASC');
    await header.click();
    await expect.poll(() => renderedEmails(page, marker)).toEqual(newestFirst);
    await expect(sort).toHaveValue('DESC');
  });
});
