import express from 'express';
import * as samlify from 'samlify';
import crypto from 'node:crypto';
import fs from 'node:fs';
import https from 'node:https';
import path from 'node:path';

export const app = express();
app.use(express.urlencoded({ extended: true }));

// ponytail: mock IdP only ever builds its own responses (never validates
// inbound XML against an XSD), so a real schema validator dependency isn't
// worth adding — skip validation. samlify throws on any parse call without
// one configured, so this is required, not optional.
samlify.setSchemaValidator({ validate: () => Promise.resolve('skipped') });

const CERT_DIR = path.join(__dirname, 'certs');
const idpCert = fs.readFileSync(path.join(CERT_DIR, 'idp-cert.pem'), 'utf8');
const idpKey = fs.readFileSync(path.join(CERT_DIR, 'idp-key.pem'), 'utf8');

const IDP_ENTITY_ID = 'http://mock-saml-idp:4001/metadata';
const SSO_URL = 'http://mock-saml-idp:4001/sso';

// SAML attribute names must match Authorizer's default SP attribute mapping
// (internal/http_handlers/saml_sp.go: samlDefaultAttributeMapping) so the
// JIT-provisioned profile picks up email/given_name/family_name.
const idp = samlify.IdentityProvider({
  entityID: IDP_ENTITY_ID,
  privateKey: idpKey,
  isAssertionEncrypted: false,
  signingCert: idpCert,
  wantAuthnRequestsSigned: false,
  singleSignOnService: [{ Binding: samlify.Constants.namespace.binding.redirect, Location: SSO_URL }],
  loginResponseTemplate: {
    context: samlify.SamlLib.defaultLoginResponseTemplate.context,
    attributes: [
      { name: 'email', valueTag: 'email', nameFormat: 'urn:oasis:names:tc:SAML:2.0:attrname-format:basic', valueXsiType: 'xs:string' },
      { name: 'firstName', valueTag: 'firstName', nameFormat: 'urn:oasis:names:tc:SAML:2.0:attrname-format:basic', valueXsiType: 'xs:string' },
      { name: 'lastName', valueTag: 'lastName', nameFormat: 'urn:oasis:names:tc:SAML:2.0:attrname-format:basic', valueXsiType: 'xs:string' },
    ],
  },
});

// Test-configurable subject for the next issued assertion.
let nextUser = { email: 'mock-saml-user@example.com', givenName: 'Mock', familyName: 'User' };

app.post('/__configure', express.json(), (req, res) => {
  nextUser = req.body;
  res.sendStatus(204);
});

app.get('/metadata', (_req, res) => {
  res.type('application/samlmetadata+xml').send(idp.getMetadata());
});

// SP-initiated: Authorizer's crewjam/saml SP redirects here with a
// base64+deflated AuthnRequest (HTTP-Redirect binding) plus RelayState.
// The real ACS URL and SP entity ID are parsed out of the AuthnRequest
// itself (samlify's parseLoginRequest) rather than trusted from a query
// param — Authorizer never sends one, and the assertion's Audience must
// match the SP's real entityID or the SP-side validation rejects it.
app.get('/sso', async (req, res) => {
  // ponytail: Express 4's async-handler rejections don't reach the default
  // error handler (that's Express 5 only) and become unhandled promise
  // rejections, which crash this long-lived shared process. Guard the whole
  // body so a malformed SAMLRequest 400s instead of taking down every other
  // in-flight SAML spec.
  try {
    const relayState = String(req.query.RelayState || '');
    const placeholderSp = samlify.ServiceProvider({ entityID: 'unused-during-parse' });
    // samlify's own `RequestInfo` and `FlowResult` types aren't exported from
    // the package, so the boundary between parseLoginRequest's return value
    // and createLoginResponse's expected input is untyped here.
    const requestInfo: any = await idp.parseLoginRequest(placeholderSp, 'redirect', { query: req.query });
    const acsUrl = String((requestInfo.extract.request as { assertionConsumerServiceUrl?: string }).assertionConsumerServiceUrl);
    const spEntityId = String(requestInfo.extract.issuer);

    const sp = samlify.ServiceProvider({
      entityID: spEntityId,
      assertionConsumerService: [{ Binding: samlify.Constants.namespace.binding.post, Location: acsUrl }],
    });

    // samlify's default (non-custom) response builder always renders an empty
    // AttributeStatement — the `loginResponseTemplate.attributes` config above
    // only takes effect when a customTagReplacement callback fills the baked
    // `{attrX}` placeholders itself. We replicate the same tag values samlify
    // computes internally (see samlify/build/src/binding-post.js) plus ours.
    const customTagReplacement = (template: string) => {
      const now = new Date();
      const notOnOrAfter = new Date(now.getTime() + 5 * 60 * 1000).toISOString();
      const tvalue: Record<string, string | number | boolean | null | undefined> = {
        ID: `_${crypto.randomUUID()}`,
        AssertionID: `_${crypto.randomUUID()}`,
        Destination: acsUrl,
        Audience: spEntityId,
        EntityID: spEntityId,
        SubjectRecipient: acsUrl,
        Issuer: IDP_ENTITY_ID,
        IssueInstant: now.toISOString(),
        AssertionConsumerServiceURL: acsUrl,
        StatusCode: samlify.Constants.StatusCode.Success,
        ConditionsNotBefore: now.toISOString(),
        ConditionsNotOnOrAfter: notOnOrAfter,
        SubjectConfirmationDataNotOnOrAfter: notOnOrAfter,
        NameIDFormat: 'urn:oasis:names:tc:SAML:2.0:nameid-format:emailAddress',
        NameID: nextUser.email,
        InResponseTo: (requestInfo.extract.request as { id?: string }).id ?? '',
        AuthnStatement: '',
        attrEmail: nextUser.email,
        attrFirstName: nextUser.givenName,
        attrLastName: nextUser.familyName,
      };
      return { id: tvalue.ID as string, context: samlify.SamlLib.replaceTagsByValue(template, tvalue) };
    };

    const { context } = await idp.createLoginResponse(sp, requestInfo, 'post', nextUser, customTagReplacement);

    res.send(`
      <html><body onload="document.forms[0].submit()">
        <form method="post" action="${acsUrl}">
          <input type="hidden" name="SAMLResponse" value="${context}" />
          <input type="hidden" name="RelayState" value="${relayState}" />
        </form>
      </body></html>
    `);
  } catch (err) {
    console.error('mock-saml-idp /sso failed:', err);
    res.sendStatus(400);
  }
});

// --- Fake SP role (Task 10: SAML IdP-side conformance) ---
// samlify supports both IdP and SP roles from the same package — reuse it
// here as a stand-in external SP so these tests drive a REAL SP-initiated
// flow against Authorizer's actual IdP-side routes
// (internal/http_handlers/saml_idp.go: SAMLIDPSSOHandler expects a genuine
// AuthnRequest built via crewjam's saml.NewIdpAuthnRequest, not a hand-rolled
// query param), mirroring how this file already stands in for a real IdP on
// the SP-side spec (Task 9).
//
// Per-flow state is keyed by the caller-supplied `relay_state` (never a
// single shared slot) so concurrent test runs against this same long-lived
// container don't clobber each other.
const pendingFakeSPFlows = new Map<
  string,
  { authorizerBase: string; org: string; entityId: string; acsUrl: string }
>();
const completedFakeSPFlows = new Map<
  string,
  { nameID: string; issuer: string; audience: string; attributes: Record<string, unknown> }
>();

// buildAuthorizerIdp fetches Authorizer's REAL, per-org IdP metadata
// (entity id + signing cert are unique per org) and hydrates a samlify IdP
// entity from it, rather than hardcoding anything — the assertion's Issuer
// and signature must match this exactly (samlify's postFlow rejects on
// ERR_UNMATCH_ISSUER / FAILED_TO_VERIFY_SIGNATURE otherwise).
async function buildAuthorizerIdp(authorizerBase: string, org: string) {
  const res = await fetch(`${authorizerBase}/saml/idp/${encodeURIComponent(org)}/metadata`);
  if (!res.ok) throw new Error(`failed to fetch Authorizer IdP metadata: ${res.status}`);
  const metadata = await res.text();
  return samlify.IdentityProvider({ metadata });
}

function fakeServiceProvider(entityId: string, acsUrl: string) {
  return samlify.ServiceProvider({
    entityID: entityId,
    assertionConsumerService: [{ Binding: samlify.Constants.namespace.binding.post, Location: acsUrl }],
  });
}

// GET /fake-sp/start: builds a real AuthnRequest (HTTP-Redirect binding,
// unsigned) and redirects the browser to Authorizer's SP-initiated SSO
// endpoint. Unsigned is deliberate, not a shortcut: crewjam/saml has no
// support for signed AuthnRequests at all ("TODO(ross): support signed authn
// requests" in identity_provider.go Validate()), and Authorizer's IdP
// metadata never sets WantAuthnRequestsSigned, so samlify's SP/IdP
// signed-flag agreement check (both default false) passes cleanly.
app.get('/fake-sp/start', async (req, res) => {
  try {
    const authorizerBase = String(req.query.authorizer_base || '');
    const org = String(req.query.org || '');
    const entityId = String(req.query.entity_id || '');
    const acsUrl = String(req.query.acs_url || '');
    const relayState = String(req.query.relay_state || '');
    if (!authorizerBase || !org || !entityId || !acsUrl || !relayState) {
      res.status(400).send('missing required query param(s): authorizer_base, org, entity_id, acs_url, relay_state');
      return;
    }
    pendingFakeSPFlows.set(relayState, { authorizerBase, org, entityId, acsUrl });

    const idp = await buildAuthorizerIdp(authorizerBase, org);
    const sp = fakeServiceProvider(entityId, acsUrl);
    // samlify's own binding-context types aren't discriminated on the
    // `binding` string argument, so `.context` (the redirect URL) needs a
    // narrowing cast here — same untyped-boundary situation as /sso above.
    const { context } = sp.createLoginRequest(idp, 'redirect', { relayState }) as { context: string };
    res.redirect(context);
  } catch (err) {
    console.error('fake-sp /start failed:', err);
    res.sendStatus(400);
  }
});

// POST /fake-sp/acs: the fake SP's Assertion Consumer Service. Validates the
// signed assertion Authorizer posts here — samlify verifies the XML-DSIG
// signature against the cert published in Authorizer's own IdP metadata —
// and stores the parsed result for the test to read back via GET
// /fake-sp/last. A response that fails validation (bad signature, wrong
// audience/issuer, expired) throws and 400s here rather than being recorded.
app.post('/fake-sp/acs', async (req, res) => {
  try {
    const relayState = String(req.body.RelayState || '');
    const pending = pendingFakeSPFlows.get(relayState);
    if (!pending) {
      res.status(400).send('unknown or expired relay state');
      return;
    }
    const idp = await buildAuthorizerIdp(pending.authorizerBase, pending.org);
    const sp = fakeServiceProvider(pending.entityId, pending.acsUrl);
    const result = await sp.parseLoginResponse(idp, 'post', { body: req.body });
    pendingFakeSPFlows.delete(relayState);
    completedFakeSPFlows.set(relayState, {
      nameID: String(result.extract.nameID || ''),
      issuer: String(result.extract.issuer || ''),
      audience: String(result.extract.audience || ''),
      attributes: (result.extract.attributes as Record<string, unknown>) || {},
    });
    res.status(200).send('ok');
  } catch (err) {
    console.error('fake-sp /acs failed:', err);
    res.sendStatus(400);
  }
});

// GET /fake-sp/last?relay_state=...: test-facing readback of a completed flow.
app.get('/fake-sp/last', (req, res) => {
  const relayState = String(req.query.relay_state || '');
  const result = completedFakeSPFlows.get(relayState);
  if (!result) {
    res.sendStatus(404);
    return;
  }
  res.json(result);
});

if (require.main === module) {
  // Authorizer's org-SAML-connection admin API rejects a non-https
  // idp_sso_url outright (internal/service/admin_org_saml.go
  // validateSAMLHTTPSURL) with no dev/test bypass — unlike the OIDC broker,
  // which relaxes this under --env=e2e. Real IdPs are https-only in
  // practice too, so serve TLS here (reusing this same signing cert/key as
  // the server cert) rather than weakening that production check.
  // Callers must trust/ignore this self-signed cert (see saml-sp.spec.ts's
  // `ignoreHTTPSErrors`).
  https.createServer({ cert: idpCert, key: idpKey }, app).listen(4001, () => console.log('mock-saml-idp listening on :4001 (https)'));
}
