package integration_tests

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestOrgVerifiedDomains covers Phase 2: DNS-TXT self-serve verification,
// super-admin trusted-assert, first-writer-wins uniqueness (the ATO invariant),
// public-suffix/consumer guards, normalization, org-scoped isolation, and the
// org-delete cascade that frees a domain for re-claim. DNS is fully mocked.
func TestOrgVerifiedDomains(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	asSuperAdmin := func() {
		clearCookies(ts)
		ts.GinContext.Request.Header.Del("Authorization")
		setAdminCookie(t, ts)
	}
	asUser := func(token string) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+token)
	}
	asAnon := func() {
		clearCookies(ts)
		ts.GinContext.Request.Header.Del("Authorization")
	}

	signupUser := func() (string, string) {
		asAnon()
		email := "org_domain_" + uuid.NewString() + "@authorizer.test"
		password := "Password@123"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, res.AccessToken)
		return res.User.ID, *res.AccessToken
	}
	createOrg := func(prefix string) string {
		asSuperAdmin()
		org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: prefix + "-" + uuid.NewString(),
		})
		require.NoError(t, err)
		return org.ID
	}
	addAdmin := func(orgID, userID string) {
		asSuperAdmin()
		_, err := ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID: orgID, UserID: userID, Roles: []string{constants.OrgRoleAdmin},
		})
		require.NoError(t, err)
	}
	freshDomain := func(prefix string) string {
		return prefix + "-" + strings.ToLower(uuid.NewString()[:8]) + ".com"
	}
	// publishTXT programs the mock resolver so the domain's challenge resolves.
	publishTXT := func(ch *model.OrgDomainChallenge) {
		ts.DNSResolver.Set(ch.RecordName, []string{ch.RecordValue})
	}

	t.Run("request mints a challenge but NO durable row", func(t *testing.T) {
		orgID := createOrg("acme")
		adminID, tok := signupUser()
		addAdmin(orgID, adminID)
		domain := freshDomain("acme")

		asUser(tok)
		ch, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.NoError(t, err)
		require.Equal(t, domain, ch.Domain)
		require.Equal(t, "TXT", ch.RecordType)
		require.Equal(t, "_authorizer-challenge."+domain, ch.RecordName)
		require.True(t, strings.HasPrefix(ch.RecordValue, "authorizer-domain-verification="))

		// No verified row exists yet, and the pending challenge is not routable.
		_, err = ts.StorageProvider.GetOrgDomainByDomain(ctx, domain)
		require.Error(t, err, "request must not create a durable row")
	})

	t.Run("verify with correct TXT inserts the row; wrong TXT does not", func(t *testing.T) {
		orgID := createOrg("verify")
		adminID, tok := signupUser()
		addAdmin(orgID, adminID)
		domain := freshDomain("verify")

		asUser(tok)
		ch, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.NoError(t, err)

		// Wrong TXT published → verify fails, no row, challenge survives.
		ts.DNSResolver.Set(ch.RecordName, []string{"authorizer-domain-verification=not-the-token"})
		_, err = ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.Error(t, err)
		_, err = ts.StorageProvider.GetOrgDomainByDomain(ctx, domain)
		require.Error(t, err, "failed verify must not create a row")

		// Correct TXT published → verify succeeds (challenge survived the retry).
		publishTXT(ch)
		row, err := ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.NoError(t, err)
		require.Equal(t, domain, row.Domain)
		require.Equal(t, orgID, row.OrgID)

		// Idempotent: re-verify returns success without a duplicate.
		row2, err := ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.NoError(t, err)
		require.Equal(t, orgID, row2.OrgID)
	})

	t.Run("first-writer-wins: second org cannot verify an owned domain", func(t *testing.T) {
		orgA := createOrg("fwa")
		orgB := createOrg("fwb")
		adminAID, tokA := signupUser()
		adminBID, tokB := signupUser()
		addAdmin(orgA, adminAID)
		addAdmin(orgB, adminBID)
		domain := freshDomain("shared")

		// Org A verifies first.
		asUser(tokA)
		chA, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgA, Domain: domain})
		require.NoError(t, err)
		publishTXT(chA)
		_, err = ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgA, Domain: domain})
		require.NoError(t, err)

		// Org B requests + publishes ITS token, but verify must be rejected — the
		// domain already routes to org A (invariant 2).
		asUser(tokB)
		chB, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgB, Domain: domain})
		require.NoError(t, err)
		publishTXT(chB)
		_, err = ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgB, Domain: domain})
		require.ErrorContains(t, err, "domain_already_verified_by_another_org")

		// Super-admin trusted-assert of the already-owned domain also fails.
		asSuperAdmin()
		_, err = ts.GraphQLProvider.AddVerifiedOrgDomain(ctx, &model.AddVerifiedOrgDomainRequest{OrgID: orgB, Domain: domain})
		require.ErrorContains(t, err, "domain_already_verified_by_another_org")
	})

	t.Run("public-suffix and consumer domains are rejected", func(t *testing.T) {
		orgID := createOrg("psl")
		adminID, tok := signupUser()
		addAdmin(orgID, adminID)

		asUser(tok)
		for _, bad := range []string{"gmail.com", "outlook.com", "com", "co.uk"} {
			_, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgID, Domain: bad})
			require.Error(t, err, "must reject %s", bad)
		}
		// Super-admin trusted-assert is subject to the same guard.
		asSuperAdmin()
		_, err := ts.GraphQLProvider.AddVerifiedOrgDomain(ctx, &model.AddVerifiedOrgDomainRequest{OrgID: orgID, Domain: "gmail.com"})
		require.Error(t, err)
	})

	t.Run("normalization: mixed case / whitespace / wildcard", func(t *testing.T) {
		orgID := createOrg("norm")
		adminID, tok := signupUser()
		addAdmin(orgID, adminID)
		base := freshDomain("norm")

		asUser(tok)
		ch, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{
			OrgID:  orgID,
			Domain: "  *." + strings.ToUpper(base) + " ",
		})
		require.NoError(t, err)
		require.Equal(t, base, ch.Domain, "domain must be normalized to lowercase, wildcard stripped")

		// Malformed inputs are rejected.
		for _, bad := range []string{"acme.com/login", "https://acme.com", "acme.com:8080", "notadomain", ""} {
			_, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgID, Domain: bad})
			require.Error(t, err, "must reject %q", bad)
		}
	})

	t.Run("trusted-assert is super-admin only", func(t *testing.T) {
		orgID := createOrg("trust")
		adminID, tok := signupUser()
		addAdmin(orgID, adminID)
		domain := freshDomain("trust")

		// Super-admin: allowed, no proof.
		asSuperAdmin()
		row, err := ts.GraphQLProvider.AddVerifiedOrgDomain(ctx, &model.AddVerifiedOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.NoError(t, err)
		require.Equal(t, domain, row.Domain)

		// Org admin of the SAME org: denied (trusted-assert is never org-admin).
		asUser(tok)
		_, err = ts.GraphQLProvider.AddVerifiedOrgDomain(ctx, &model.AddVerifiedOrgDomainRequest{OrgID: orgID, Domain: freshDomain("trust2")})
		require.Error(t, err)
	})

	t.Run("org-scoped isolation: A cannot list/verify/delete B's domain", func(t *testing.T) {
		orgA := createOrg("isoa")
		orgB := createOrg("isob")
		adminAID, tokA := signupUser()
		adminBID, tokB := signupUser()
		addAdmin(orgA, adminAID)
		addAdmin(orgB, adminBID)
		domainB := freshDomain("isob")

		// Org B verifies its domain.
		asUser(tokB)
		chB, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgB, Domain: domainB})
		require.NoError(t, err)
		publishTXT(chB)
		_, err = ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgB, Domain: domainB})
		require.NoError(t, err)

		// Org A admin cannot list org B's domains.
		asUser(tokA)
		_, err = ts.GraphQLProvider.OrgDomains(ctx, &model.ListOrgDomainsRequest{OrgID: orgB})
		require.Error(t, err)

		// Org A admin cannot delete org B's domain (auth keys on the loaded row's org).
		_, err = ts.GraphQLProvider.DeleteOrgDomain(ctx, &model.DeleteOrgDomainRequest{Domain: domainB})
		require.Error(t, err)

		// The domain still belongs to org B.
		asUser(tokB)
		list, err := ts.GraphQLProvider.OrgDomains(ctx, &model.ListOrgDomainsRequest{OrgID: orgB})
		require.NoError(t, err)
		require.Len(t, list.OrgDomains, 1)
		require.Equal(t, domainB, list.OrgDomains[0].Domain)
	})

	t.Run("org-admin deletes own domain; org-delete cascades and frees it", func(t *testing.T) {
		orgA := createOrg("cascadea")
		orgB := createOrg("cascadeb")
		adminAID, tokA := signupUser()
		adminBID, tokB := signupUser()
		addAdmin(orgA, adminAID)
		addAdmin(orgB, adminBID)
		domain := freshDomain("cascade")

		// Org A verifies two domains; deletes one directly.
		asUser(tokA)
		ch, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgA, Domain: domain})
		require.NoError(t, err)
		publishTXT(ch)
		_, err = ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgA, Domain: domain})
		require.NoError(t, err)

		extra := freshDomain("cascade-extra")
		ch2, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgA, Domain: extra})
		require.NoError(t, err)
		publishTXT(ch2)
		_, err = ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgA, Domain: extra})
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.DeleteOrgDomain(ctx, &model.DeleteOrgDomainRequest{Domain: extra})
		require.NoError(t, err)
		asUser(tokA)
		list, err := ts.GraphQLProvider.OrgDomains(ctx, &model.ListOrgDomainsRequest{OrgID: orgA})
		require.NoError(t, err)
		require.Len(t, list.OrgDomains, 1)

		// Delete org A (super-admin) → its remaining domain must be freed (M1).
		asSuperAdmin()
		_, err = ts.GraphQLProvider.DeleteOrganization(ctx, &model.OrganizationRequest{ID: orgA})
		require.NoError(t, err)
		_, err = ts.StorageProvider.GetOrgDomainByDomain(ctx, domain)
		require.Error(t, err, "cascade must remove the org's domains")

		// Org B can now claim the freed domain.
		asUser(tokB)
		chB, err := ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgB, Domain: domain})
		require.NoError(t, err)
		publishTXT(chB)
		reclaimed, err := ts.GraphQLProvider.VerifyOrgDomain(ctx, &model.VerifyOrgDomainRequest{OrgID: orgB, Domain: domain})
		require.NoError(t, err)
		require.Equal(t, orgB, reclaimed.OrgID)
	})

	t.Run("plain member and non-member cannot request", func(t *testing.T) {
		orgID := createOrg("deny")
		memberID, memberTok := signupUser()
		asSuperAdmin()
		_, err := ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{OrgID: orgID, UserID: memberID, Roles: []string{"billing"}})
		require.NoError(t, err)
		_, outsiderTok := signupUser()
		domain := freshDomain("deny")

		asUser(memberTok)
		_, err = ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.Error(t, err)

		asUser(outsiderTok)
		_, err = ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.Error(t, err)

		asAnon()
		_, err = ts.GraphQLProvider.RequestOrgDomain(ctx, &model.RequestOrgDomainRequest{OrgID: orgID, Domain: domain})
		require.Error(t, err)
	})
}
