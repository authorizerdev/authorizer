package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// VerificationRequest model for db
type VerificationRequest struct {
	Key         string `json:"_key,omitempty" bson:"_key" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID          string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	Token       string `json:"token" bson:"token" cql:"jwt_token" dynamo:"token" index:"token,hash"`
	Identifier  string `gorm:"uniqueIndex:idx_email_identifier;type:varchar(64)" json:"identifier" bson:"identifier" cql:"identifier" dynamo:"identifier"`
	ExpiresAt   int64  `json:"expires_at" bson:"expires_at" cql:"expires_at" dynamo:"expires_at"`
	Email       string `gorm:"uniqueIndex:idx_email_identifier;type:varchar(256)" json:"email" bson:"email" cql:"email" dynamo:"email"`
	Nonce       string `json:"nonce" bson:"nonce" cql:"nonce" dynamo:"nonce"`
	RedirectURI string `json:"redirect_uri" bson:"redirect_uri" cql:"redirect_uri" dynamo:"redirect_uri"`
	CreatedAt   int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt   int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

func (v *VerificationRequest) AsAPIVerificationRequest() *model.VerificationRequest {
	id := v.ID
	if strings.Contains(id, Collections.VerificationRequest+"/") {
		id = strings.TrimPrefix(id, Collections.VerificationRequest+"/")
	}

	return &model.VerificationRequest{
		ID:          id,
		Token:       refs.NewStringRef(v.Token),
		Identifier:  refs.NewStringRef(v.Identifier),
		Expires:     refs.NewInt64Ref(v.ExpiresAt),
		Email:       refs.NewStringRef(v.Email),
		Nonce:       refs.NewStringRef(v.Nonce),
		RedirectURI: refs.NewStringRef(v.RedirectURI),
		CreatedAt:   refs.NewInt64Ref(v.CreatedAt),
		UpdatedAt:   refs.NewInt64Ref(v.UpdatedAt),
	}
}
