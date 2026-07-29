#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
openssl req -x509 -newkey rsa:2048 -keyout idp-key.pem -out idp-cert.pem \
  -days 3650 -nodes -subj "/CN=mock-saml-idp.e2e-playground.local"
cp idp-cert.pem idp-key.pem ../../mocks/mock-saml-idp/certs/
echo "Generated test-only SAML IdP cert/key (not used for anything real)."
