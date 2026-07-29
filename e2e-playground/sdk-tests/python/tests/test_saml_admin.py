"""SAML — the SDK-drivable administration surface.

What IS SDK surface (tested here through authorizer-py's admin client):
  * Service Provider registry CRUD (create / read / list / update / delete) —
    the IdP-side record of a downstream SP.
  * IdP signing-key lifecycle (rotate / list / retire).
  * SP metadata XML import (parse-only) into entity_id / acs_url / certificate.

What is NOT SDK surface (and why): the actual SAML SSO ceremony —
SP-initiated AuthnRequest (HTTP-Redirect binding) -> Authorizer login UI ->
auto-submitted, XML-signed <Response> POSTed to the SP's ACS URL — is a
browser + HTML-form-POST protocol (crewjam WriteResponse writes a self-posting
form). There is no request/response method for it and there cannot be: the SDK
never sees the signed assertion, the browser carries it. That flow is covered
by tests/saml-idp.spec.ts / tests/saml-sp.spec.ts in the Playwright suite.

Mirrors the admin-side setup those specs perform via _create_saml_service_provider.
"""

from __future__ import annotations

import pytest
from authorizer import (
    AuthorizerAdminClient,
    CreateOrganizationRequest,
    CreateSAMLServiceProviderRequest,
    ImportSAMLSPMetadataRequest,
    ListSAMLIDPKeysRequest,
    ListSAMLServiceProvidersRequest,
    RetireSAMLIDPKeyRequest,
    RotateSAMLIDPCertRequest,
    SAMLServiceProviderRequest,
    UpdateSAMLServiceProviderRequest,
)
from authorizer.exceptions import AuthorizerError

import helpers

pytestmark = pytest.mark.live

SP_METADATA_XML = """<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
                  entityID="https://sp.example.com/saml/metadata">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://sp.example.com/saml/acs" index="0"/>
  </SPSSODescriptor>
</EntityDescriptor>"""


def _new_org(admin: AuthorizerAdminClient, prefix: str) -> str:
    org = admin.create_organization(
        CreateOrganizationRequest(name=helpers.random_email(prefix).split("@")[0])
    )
    return org.id


def test_saml_service_provider_crud(admin: AuthorizerAdminClient) -> None:
    org_id = _new_org(admin, "saml-sp")
    entity_id = f"sp-{org_id}"
    acs_url = "https://sp.example.test/acs"

    sp = admin.create_saml_service_provider(
        CreateSAMLServiceProviderRequest(
            org_id=org_id, name="fake-sp", entity_id=entity_id, acs_url=acs_url
        )
    )
    assert sp.entity_id == entity_id
    assert sp.acs_url == acs_url
    assert sp.is_active is True

    fetched = admin.get_saml_service_provider(SAMLServiceProviderRequest(id=sp.id))
    assert fetched.id == sp.id

    listed = admin.list_saml_service_providers(ListSAMLServiceProvidersRequest(org_id=org_id))
    assert any(p.id == sp.id for p in listed.saml_service_providers)

    updated = admin.update_saml_service_provider(
        UpdateSAMLServiceProviderRequest(id=sp.id, name="renamed-sp", is_active=False)
    )
    assert updated.name == "renamed-sp"
    assert updated.is_active is False

    deleted = admin.delete_saml_service_provider(SAMLServiceProviderRequest(id=sp.id))
    assert deleted.message


def test_saml_idp_signing_key_rotation_lifecycle(admin: AuthorizerAdminClient) -> None:
    org_id = _new_org(admin, "saml-idp")

    key1 = admin.rotate_saml_idp_cert(RotateSAMLIDPCertRequest(org_id=org_id))
    assert key1.status == "current"
    assert key1.cert_pem.startswith("-----BEGIN CERTIFICATE-----")

    # Second rotation demotes key1 to "active" and mints a new "current".
    key2 = admin.rotate_saml_idp_cert(RotateSAMLIDPCertRequest(org_id=org_id))
    assert key2.status == "current"
    assert key2.id != key1.id

    keys = {k.id: k.status for k in admin.list_saml_idp_keys(ListSAMLIDPKeysRequest(org_id=org_id))}
    assert keys[key1.id] == "active"
    assert keys[key2.id] == "current"

    # The current key cannot be retired; a superseded ("active") one can.
    with pytest.raises(AuthorizerError) as exc:
        admin.retire_saml_idp_key(RetireSAMLIDPKeyRequest(id=key2.id))
    assert "cannot retire the current signing key" in exc.value.message.lower()

    retired = admin.retire_saml_idp_key(RetireSAMLIDPKeyRequest(id=key1.id))
    assert retired.message
    after_keys = admin.list_saml_idp_keys(ListSAMLIDPKeysRequest(org_id=org_id))
    after = {k.id: k.status for k in after_keys}
    assert after[key1.id] == "retired"


def test_saml_sp_metadata_import_parses(admin: AuthorizerAdminClient) -> None:
    result = admin.import_saml_sp_metadata(
        ImportSAMLSPMetadataRequest(metadata_xml=SP_METADATA_XML)
    )
    assert result.entity_id == "https://sp.example.com/saml/metadata"
    assert result.acs_url == "https://sp.example.com/saml/acs"
