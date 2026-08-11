#!/bin/sh
# Generates a CA and a leaf certificate for the CIMD mock host.
#
# Idempotent: compose may start this more than once across a run, and
# regenerating would invalidate the bundle a running server already loaded.
set -eu
OUT=/certs
if [ -f "$OUT/bundle.crt" ] && [ -f "$OUT/server.key" ]; then
  echo "certs already present, leaving them alone"
  exit 0
fi
mkdir -p "$OUT"

openssl req -x509 -newkey rsa:2048 -sha256 -days 3650 -nodes \
  -keyout "$OUT/ca.key" -out "$OUT/ca.crt" \
  -subj "/CN=authorizer-e2e-local-ca" >/dev/null 2>&1

# SAN, not CN: Go's TLS stack ignores CN entirely, so a CN-only certificate
# would fail verification with a confusing "certificate is not valid for any
# names" rather than an obvious misconfiguration.
openssl req -newkey rsa:2048 -sha256 -nodes \
  -keyout "$OUT/server.key" -out "$OUT/server.csr" \
  -subj "/CN=cimd-client" >/dev/null 2>&1
printf "subjectAltName=DNS:cimd-client,DNS:localhost\nextendedKeyUsage=serverAuth\n" > "$OUT/san.cnf"
openssl x509 -req -in "$OUT/server.csr" -CA "$OUT/ca.crt" -CAkey "$OUT/ca.key" \
  -CAcreateserial -out "$OUT/server.crt" -days 3650 -sha256 \
  -extfile "$OUT/san.cnf" >/dev/null 2>&1

# The bundle APPENDS to the public roots rather than replacing them: Go's
# SSL_CERT_FILE overrides the system pool wholesale, so shipping only our CA
# would silently break every other TLS dial the server makes.
cat /etc/ssl/certs/ca-certificates.crt "$OUT/ca.crt" > "$OUT/bundle.crt"

chmod 644 "$OUT/bundle.crt" "$OUT/server.crt" "$OUT/server.key"
echo "generated CA + leaf for cimd-client"
