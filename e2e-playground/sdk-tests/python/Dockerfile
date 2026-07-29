# Runs the Python SDK e2e suite from inside the compose network, for the exact
# same reason the sibling `playwright` service does: the social-OAuth redirect
# chain (mock-oauth:4000) and the WebAuthn RP origin
# (webauthn.e2e-playground.test:8080) only resolve/validate from within the
# docker-internal network. Never started by `up`; invoked via
# `docker compose run --rm python-sdk ...`.
FROM python:3.12-slim

WORKDIR /sdk-tests

# Install deps first for layer caching. requirements.txt pins the PUBLISHED
# authorizer-py — no source of this repo's Go server or the SDK is copied in.
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY . ./

# -p no:cacheprovider keeps the read-only image clean; -s streams poll output.
ENTRYPOINT ["python", "-m", "pytest", "-p", "no:cacheprovider"]
