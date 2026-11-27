// e2e-playground/fixtures/mailpit.ts
import { APIRequestContext } from '@playwright/test';

const MAILPIT_BASE = process.env.MAILPIT_BASE_URL || 'http://localhost:8025';

// waitForEmailLink polls Mailpit for a message to `email` whose plain-text body
// contains a link matching `pattern` (first capture group = the URL), and
// returns that URL.
//
// Uses /api/v1/search?query=to:<address> rather than the /api/v1/messages
// summary list tests/magic-link.spec.ts walks: every instance in the stack
// shares one Mailpit, and the summary list is paginated (50 newest), so in a
// full parallel run a message can scroll off it. The search is exact on the
// recipient, and every spec mints a unique address.
//
// Matching on the link pattern, not just the recipient, matters on instances
// that mail more than one thing to the same address (authorizer-email-verify
// sends a signup verification mail before the reset mail). The body is read
// from Text, not HTML: HTML entity-encodes "&" in the URL.
export async function waitForEmailLink(
  request: APIRequestContext,
  email: string,
  pattern: RegExp
): Promise<string> {
  const deadline = Date.now() + 15_000;
  while (Date.now() < deadline) {
    const res = await request.get(`${MAILPIT_BASE}/api/v1/search`, {
      params: { query: `to:"${email}"`, limit: 50 },
    });
    if (res.ok()) {
      const { messages } = (await res.json()) as { messages: { ID: string }[] };
      for (const m of messages ?? []) {
        const detail = await (await request.get(`${MAILPIT_BASE}/api/v1/message/${m.ID}`)).json();
        const match = pattern.exec(detail.Text ?? '');
        if (match) return match[1].replace(/[)\].,>]+$/, '');
      }
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`no email to ${email} with a link matching ${pattern} within 15s`);
}

export const VERIFY_EMAIL_LINK = /(https?:\/\/\S+\/verify_email\?\S+)/;
export const RESET_PASSWORD_LINK = /(https?:\/\/\S+\/app\/reset-password\?\S+)/;
