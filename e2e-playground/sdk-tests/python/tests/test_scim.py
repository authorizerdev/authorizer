"""SCIM — SDK-drivable surface vs. the inbound provisioning protocol.

Architecture (honest split):

  * SDK surface — the *administration* of SCIM: creating an org, provisioning
    a SCIM endpoint + bearer token, reading it back, rotating the token,
    deleting it; plus registering the SCIM lifecycle webhooks and reading
    webhook delivery logs. All of that goes through authorizer-py here.

  * NOT SDK surface — the SCIM 2.0 protocol itself (POST/PATCH/DELETE
    /scim/v2/Users). This is an INBOUND, third-party-facing REST spec (RFC
    7644) that an external IdP's provisioning connector speaks to Authorizer.
    It is deliberately not wrapped by the public/admin SDK, so those calls are
    made with a raw httpx client — transparently labeled, not dressed up as
    "SDK". Their server-side effects are then verified back THROUGH the SDK
    (admin users query, webhook logs).

Mirrors tests/scim.spec.ts.
"""

from __future__ import annotations

import hashlib
import hmac

import httpx
import pytest
from authorizer import (
    AddWebhookRequest,
    AuthorizerAdminClient,
    CreateOrganizationRequest,
    CreateScimEndpointRequest,
    ListUsersRequest,
    ListWebhookLogRequest,
    ScimEndpointRequest,
)

import helpers

pytestmark = pytest.mark.live

SCIM_CORE = "urn:ietf:params:scim:schemas:core:2.0:User"
SCIM_PATCH = "urn:ietf:params:scim:api:messages:2.0:PatchOp"


def _scim_client(token: str) -> httpx.Client:
    return httpx.Client(
        base_url=helpers.AUTHORIZER_BASE_URL,
        headers={"Authorization": f"Bearer {token}", "Content-Type": "application/scim+json"},
        timeout=10.0,
    )


def test_scim_endpoint_lifecycle_via_sdk(admin: AuthorizerAdminClient) -> None:
    """create / read / rotate-token / delete a SCIM endpoint — all SDK."""
    org = admin.create_organization(
        CreateOrganizationRequest(name=helpers.random_email("scim-org").split("@")[0])
    )

    created = admin.create_scim_endpoint(CreateScimEndpointRequest(org_id=org.id))
    assert created.token, "endpoint creation must return a bearer token"
    assert created.scim_endpoint.org_id == org.id
    assert created.scim_endpoint.enabled is True

    fetched = admin.get_scim_endpoint(ScimEndpointRequest(org_id=org.id))
    assert fetched.id == created.scim_endpoint.id

    rotated = admin.rotate_scim_token(ScimEndpointRequest(org_id=org.id))
    assert rotated.token and rotated.token != created.token, "rotation must mint a new token"

    deleted = admin.delete_scim_endpoint(ScimEndpointRequest(org_id=org.id))
    assert deleted.message


def test_scim_provisioning_webhooks_end_to_end(admin: AuthorizerAdminClient) -> None:
    """SDK registers webhooks + reads logs; raw HTTP drives the SCIM protocol."""
    endpoint = f"{helpers.WEBHOOK_SINK_BASE_URL}/webhook"
    for event in ("user.provisioned", "user.scim_updated", "user.deprovisioned"):
        admin.add_webhook(AddWebhookRequest(event_name=event, endpoint=endpoint, enabled=True))

    org = admin.create_organization(
        CreateOrganizationRequest(name=helpers.random_email("scim-wh").split("@")[0])
    )
    token = admin.create_scim_endpoint(CreateScimEndpointRequest(org_id=org.id)).token
    email = f"scim-webhook-user-{org.id}@example.com"

    # --- raw SCIM protocol (inbound REST, not SDK surface) --- #
    with _scim_client(token) as scim:
        create = scim.post(
            "/scim/v2/Users",
            json={
                "schemas": [SCIM_CORE],
                "userName": email,
                "name": {"givenName": "Katherine", "familyName": "Johnson"},
                "emails": [{"value": email, "primary": True}],
                "active": True,
            },
        )
        assert create.status_code == 201
        user_id = create.json()["id"]
        assert create.json()["userName"] == email

        patch = scim.patch(
            f"/scim/v2/Users/{user_id}",
            json={
                "schemas": [SCIM_PATCH],
                "Operations": [{"op": "replace", "path": "name.givenName", "value": "Kate"}],
            },
        )
        assert patch.status_code == 200

        delete = scim.delete(f"/scim/v2/Users/{user_id}")
        assert delete.status_code == 204

    # --- verify server-side effects back through the SDK / sinks --- #
    # Webhook delivery (detached goroutine) — poll the sink, verify HMAC.
    events = helpers.wait_for_webhook_events(email)
    assert sorted(events) == ["user.deprovisioned", "user.provisioned", "user.scim_updated"]
    for name, delivered in events.items():
        assert delivered["body"]["event_name"] == name
        assert delivered["body"]["user"]["email"] == email
        expected = hmac.new(
            helpers.CLIENT_SECRET.encode(), delivered["rawBody"].encode(), hashlib.sha256
        ).hexdigest()
        assert delivered["signature"] == expected, f"HMAC mismatch for {name}"

    # SDK reads the webhook delivery logs — must show successful (200) deliveries.
    logs = admin.webhook_logs(ListWebhookLogRequest())
    assert any(log.http_status == 200 for log in logs.webhook_logs)


def test_scim_provisioned_user_visible_via_admin_sdk(admin: AuthorizerAdminClient) -> None:
    """A user provisioned over raw SCIM is readable through the admin SDK."""
    org = admin.create_organization(
        CreateOrganizationRequest(name=helpers.random_email("scim-admin").split("@")[0])
    )
    token = admin.create_scim_endpoint(CreateScimEndpointRequest(org_id=org.id)).token
    email = f"scim-admin-{org.id}@example.com"

    with _scim_client(token) as scim:
        create = scim.post(
            "/scim/v2/Users",
            json={
                "schemas": [SCIM_CORE],
                "userName": email,
                "name": {"givenName": "Ada", "familyName": "Lovelace"},
                "emails": [{"value": email, "primary": True}],
                "active": True,
            },
        )
        assert create.status_code == 201

    users = admin.users(ListUsersRequest(query=email)).users
    match = next((u for u in users if u.email == email), None)
    assert match is not None, "SCIM-provisioned user must be visible to the admin SDK"
    assert match.given_name == "Ada"
    assert match.family_name == "Lovelace"
