import express from 'express';
import * as samlify from 'samlify';
import crypto from 'node:crypto';
import fs from 'node:fs';
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

if (require.main === module) {
  app.listen(4001, () => console.log('mock-saml-idp listening on :4001'));
}
