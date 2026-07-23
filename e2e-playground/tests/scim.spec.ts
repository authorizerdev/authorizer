// e2e-playground/tests/scim.spec.ts
import { test, expect } from '@playwright/test';
import { createOrg, createSCIMEndpoint } from '../fixtures/adminClient';

test.describe('SCIM', () => {
  test('provision, update, and deprovision a user via SCIM', async ({ request }) => {
    const org = await createOrg(`scim-${Date.now()}`);
    const { token } = await createSCIMEndpoint(org.id);
    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/scim+json' };
    const email = `scim-user-${org.id}@example.com`;

    const createRes = await request.post('/scim/v2/Users', {
      headers,
      data: {
        schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'],
        userName: email,
        name: { givenName: 'Katherine', familyName: 'Johnson' },
        emails: [{ value: email, primary: true }],
        active: true,
      },
    });
    expect(createRes.status()).toBe(201);
    const created = await createRes.json();
    expect(created.userName).toBe(email);

    const patchRes = await request.patch(`/scim/v2/Users/${created.id}`, {
      headers,
      data: { schemas: ['urn:ietf:params:scim:api:messages:2.0:PatchOp'], Operations: [{ op: 'replace', value: { active: false } }] },
    });
    expect(patchRes.status()).toBe(200);
    const patched = await patchRes.json();
    expect(patched.active).toBe(false);

    const deleteRes = await request.delete(`/scim/v2/Users/${created.id}`, { headers });
    expect(deleteRes.status()).toBe(204);
  });

  test('cross-org isolation: org A token cannot read org B users', async ({ request }) => {
    const orgA = await createOrg(`scim-a-${Date.now()}`);
    const orgB = await createOrg(`scim-b-${Date.now()}`);
    const { token: tokenA } = await createSCIMEndpoint(orgA.id);
    const { token: tokenB } = await createSCIMEndpoint(orgB.id);
    const headersB = { Authorization: `Bearer ${tokenB}`, 'Content-Type': 'application/scim+json' };

    const createRes = await request.post('/scim/v2/Users', {
      headers: headersB,
      data: {
        schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'],
        userName: `org-b-only-${orgB.id}@example.com`,
        active: true,
      },
    });
    const orgBUser = await createRes.json();

    const crossOrgRes = await request.get(`/scim/v2/Users/${orgBUser.id}`, {
      headers: { Authorization: `Bearer ${tokenA}` },
    });
    expect(crossOrgRes.status()).toBe(404);
  });
});
