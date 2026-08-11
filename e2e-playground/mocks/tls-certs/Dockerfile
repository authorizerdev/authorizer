# One-shot certificate authority for the e2e stack.
#
# CIMD requires an https client_id and the AUTHORIZER SERVER is what fetches it,
# so the mock client host must serve TLS with a certificate the server trusts.
# This generates both halves into a shared volume: the leaf for cimd-client, and
# a CA bundle the server is pointed at.
#
# Generated at run time, never committed — a private key in the repository would
# be a real key regardless of what it is labelled.
FROM alpine:3.20
RUN apk add --no-cache openssl ca-certificates
COPY generate.sh /generate.sh
RUN chmod +x /generate.sh
ENTRYPOINT ["/generate.sh"]
