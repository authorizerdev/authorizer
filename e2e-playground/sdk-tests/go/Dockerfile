# e2e-playground/sdk-tests/go/Dockerfile
#
# Runs the SDK-driven Go suite INSIDE the compose network (same rationale as the
# playwright service): so the docker-internal hostnames the tests target —
# especially the WebAuthn instance's pinned origin webauthn.e2e-playground.test —
# resolve and match the servers' --url/--allowed-origins exactly.
#
# go 1.25.5 is required by the authorizer-go SDK's own go.mod; golang:1.25 tracks
# the latest 1.25.x patch, which satisfies it.
FROM golang:1.25

WORKDIR /sdk-tests

# Cache module downloads separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# The suite talks to the live stack, so it's the container's runtime command
# (not a build step). -count=1 bypasses the test cache; -v surfaces each case.
CMD ["go", "test", "-count=1", "-v", "./..."]
