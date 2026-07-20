package storage

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// testSAMLIDPStorageOperations exercises the SAMLServiceProvider and SAMLIDPKey
// CRUD across every configured DB provider (the multi-DB parity guard).
func testSAMLIDPStorageOperations(t *testing.T, ctx context.Context, p Provider) {
	orgID := "org-" + uuid.New().String()
	entityID := "https://sp-" + uuid.New().String() + ".example.com/metadata"

	// --- SAMLServiceProvider ---
	sp, err := p.AddSAMLServiceProvider(ctx, &schemas.SAMLServiceProvider{
		OrgID:             orgID,
		Name:              "Zendesk",
		EntityID:          entityID,
		ACSURL:            "https://sp.example.com/acs",
		NameIDFormat:      "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		MappedAttributes:  refs.NewStringRef(`{"email":"email"}`),
		AllowIDPInitiated: true,
		IsActive:          true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, sp.ID)

	byID, err := p.GetSAMLServiceProviderByID(ctx, sp.ID)
	require.NoError(t, err)
	assert.Equal(t, "Zendesk", byID.Name)
	assert.Equal(t, entityID, byID.EntityID)
	assert.True(t, byID.AllowIDPInitiated)

	byEntity, err := p.GetSAMLServiceProviderByOrgAndEntityID(ctx, orgID, entityID)
	require.NoError(t, err)
	assert.Equal(t, sp.ID, byEntity.ID)

	// Wrong org must not resolve the record.
	_, err = p.GetSAMLServiceProviderByOrgAndEntityID(ctx, "org-other", entityID)
	assert.Error(t, err, "SP lookup must be org-scoped")

	byEntity.Name = "Zendesk Prod"
	byEntity.IsActive = false
	updated, err := p.UpdateSAMLServiceProvider(ctx, byEntity)
	require.NoError(t, err)
	assert.Equal(t, "Zendesk Prod", updated.Name)
	assert.False(t, updated.IsActive)

	list, page, err := p.ListSAMLServiceProviders(ctx, orgID, &model.Pagination{Limit: 10, Offset: 0, Page: 1})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
	assert.GreaterOrEqual(t, page.Total, int64(1))

	require.NoError(t, p.DeleteSAMLServiceProvider(ctx, updated))
	_, err = p.GetSAMLServiceProviderByID(ctx, sp.ID)
	assert.Error(t, err, "deleted SP must not be retrievable")

	// --- SAMLIDPKey ---
	keyOrg := "keyorg-" + uuid.New().String()
	key, err := p.AddSAMLIDPKey(ctx, &schemas.SAMLIDPKey{
		OrgID:         keyOrg,
		CertPEM:       "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		PrivateKeyEnc: "encrypted-blob",
		Algorithm:     "RS256",
		Status:        schemas.SAMLIDPKeyStatusCurrent,
	})
	require.NoError(t, err)
	require.NotEmpty(t, key.ID)

	gotKey, err := p.GetSAMLIDPKeyByID(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, schemas.SAMLIDPKeyStatusCurrent, gotKey.Status)
	assert.Equal(t, "encrypted-blob", gotKey.PrivateKeyEnc, "encrypted private key must round-trip")

	gotKey.Status = schemas.SAMLIDPKeyStatusActive
	_, err = p.UpdateSAMLIDPKey(ctx, gotKey)
	require.NoError(t, err)

	keys, err := p.ListSAMLIDPKeys(ctx, keyOrg)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, schemas.SAMLIDPKeyStatusActive, keys[0].Status)

	require.NoError(t, p.DeleteSAMLIDPKey(ctx, keys[0]))
	remaining, err := p.ListSAMLIDPKeys(ctx, keyOrg)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}
