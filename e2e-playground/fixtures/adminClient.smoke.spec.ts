// e2e-playground/fixtures/adminClient.smoke.spec.ts
import { test, expect } from '@playwright/test';
import { createOrg, addVerifiedDomain } from './adminClient';

test('admin seed client can create an org and verify a domain', async () => {
  const org = await createOrg(`e2e-smoke-${Date.now()}`);
  expect(org.id).toBeTruthy();
  await expect(addVerifiedDomain(org.id, `${org.id}.example.com`)).resolves.not.toThrow();
});
