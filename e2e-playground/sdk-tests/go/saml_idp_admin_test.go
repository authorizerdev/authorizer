package sdktests

// SAML IdP admin surface — fully SDK-drivable, no browser needed.
//
// Authorizer-as-IdP exposes a real admin API for managing downstream SAML
// Service Providers and the IdP signing keys, and authorizer-go wraps all of it
// with typed proto methods. That admin/config plane is exactly what belongs in
// an SDK and is exercised here end to end: SP create → get → list → update →
// key rotation → key listing → delete.
//
// The SAML *login ceremony itself* (AuthnRequest redirect, IdP form POST, signed
// assertion back to the SP ACS) is inherently a browser/form-post flow with no
// SDK login-initiation surface — that stays in the Playwright suite
// (tests/saml-idp.spec.ts, tests/saml-sp.spec.ts). See README.

import (
	"strings"
	"testing"

	authorizer "github.com/authorizerdev/authorizer-go/v2"
	authorizerv1 "github.com/authorizerdev/authorizer-proto-go/authorizer/v1"
)

func newOrg(t *testing.T, admin *authorizer.AuthorizerAdminClient, prefix string) *authorizer.Organization {
	t.Helper()
	org, err := admin.CreateOrganization(&authorizer.CreateOrganizationRequest{
		Name: randomSlug(prefix),
	})
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}
	return org
}

func TestSAMLIdP_ServiceProviderCRUDAndKeyRotation(t *testing.T) {
	admin := adminClient(t, baseURL)
	org := newOrg(t, admin, "saml-sp")

	entityID := "https://sp.example.com/" + org.ID
	acsURL := "https://sp.example.com/acs"

	// create
	createRes, err := admin.CreateSamlServiceProvider(&authorizerv1.CreateSamlServiceProviderRequest{
		OrgId:    org.ID,
		Name:     "e2e-sp",
		EntityId: entityID,
		AcsUrl:   acsURL,
	})
	if err != nil {
		t.Fatalf("CreateSamlServiceProvider: %v", err)
	}
	sp := createRes.GetSamlServiceProvider()
	if sp.GetId() == "" || sp.GetEntityId() != entityID || sp.GetAcsUrl() != acsURL {
		t.Fatalf("unexpected SP after create: %+v", sp)
	}

	// get by id
	got, err := admin.GetSamlServiceProvider(&authorizerv1.GetSamlServiceProviderRequest{Id: sp.GetId()})
	if err != nil {
		t.Fatalf("GetSamlServiceProvider: %v", err)
	}
	if got.GetSamlServiceProvider().GetEntityId() != entityID {
		t.Fatalf("get returned wrong SP: %+v", got.GetSamlServiceProvider())
	}

	// list for the org includes it
	list, err := admin.ListSamlServiceProviders(&authorizerv1.ListSamlServiceProvidersRequest{OrgId: org.ID})
	if err != nil {
		t.Fatalf("ListSamlServiceProviders: %v", err)
	}
	if !containsSP(list.GetSamlServiceProviders(), sp.GetId()) {
		t.Fatalf("listed SPs did not contain %s", sp.GetId())
	}

	// update name + active flag
	newName := "e2e-sp-renamed"
	inactive := false
	upd, err := admin.UpdateSamlServiceProvider(&authorizerv1.UpdateSamlServiceProviderRequest{
		Id:       sp.GetId(),
		Name:     &newName,
		IsActive: &inactive,
	})
	if err != nil {
		t.Fatalf("UpdateSamlServiceProvider: %v", err)
	}
	if upd.GetSamlServiceProvider().GetName() != newName {
		t.Fatalf("update did not apply name: %+v", upd.GetSamlServiceProvider())
	}
	if upd.GetSamlServiceProvider().GetIsActive() {
		t.Fatalf("update did not deactivate SP")
	}

	// rotate the org's IdP signing cert, then confirm the new key is listed
	rot, err := admin.RotateSamlIdpCert(&authorizerv1.RotateSamlIdpCertRequest{OrgId: org.ID})
	if err != nil {
		t.Fatalf("RotateSamlIdpCert: %v", err)
	}
	rotated := rot.GetSamlIdpKey()
	if rotated.GetId() == "" || !strings.Contains(rotated.GetCertPem(), "CERTIFICATE") {
		t.Fatalf("rotated key missing id/cert PEM: %+v", rotated)
	}

	keys, err := admin.ListSamlIdpKeys(&authorizerv1.ListSamlIdpKeysRequest{OrgId: org.ID})
	if err != nil {
		t.Fatalf("ListSamlIdpKeys: %v", err)
	}
	if !containsKey(keys.GetSamlIdpKeys(), rotated.GetId()) {
		t.Fatalf("rotated key %s not present in ListSamlIdpKeys", rotated.GetId())
	}

	// delete the SP
	if _, err := admin.DeleteSamlServiceProvider(&authorizerv1.DeleteSamlServiceProviderRequest{Id: sp.GetId()}); err != nil {
		t.Fatalf("DeleteSamlServiceProvider: %v", err)
	}
	if _, err := admin.GetSamlServiceProvider(&authorizerv1.GetSamlServiceProviderRequest{Id: sp.GetId()}); err == nil {
		t.Fatalf("expected GetSamlServiceProvider to fail after delete")
	}
}

func containsSP(sps []*authorizerv1.SamlServiceProvider, id string) bool {
	for _, s := range sps {
		if s.GetId() == id {
			return true
		}
	}
	return false
}

func containsKey(keys []*authorizerv1.SamlIdpKey, id string) bool {
	for _, k := range keys {
		if k.GetId() == id {
			return true
		}
	}
	return false
}
