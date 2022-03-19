package models

import "github.com/authorizerdev/authorizer/server/graph/model"

// VerificationRequest model for db
type VerificationRequest struct {
	Key         string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty"` // for arangodb
	ID          string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id"`
	Token       string `gorm:"type:text" json:"token" bson:"token" cql:"token"`
	Identifier  string `gorm:"uniqueIndex:idx_email_identifier" json:"identifier" bson:"identifier" cql:"identifier"`
	ExpiresAt   int64  `json:"expires_at" bson:"expires_at" cql:"expires_at"`
	CreatedAt   int64  `json:"created_at" bson:"created_at" cql:"created_at"`
	UpdatedAt   int64  `json:"updated_at" bson:"updated_at" cql:"updated_at"`
	Email       string `gorm:"uniqueIndex:idx_email_identifier" json:"email" bson:"email" cql:"email"`
	Nonce       string `gorm:"type:text" json:"nonce" bson:"nonce" cql:"nonce"`
	RedirectURI string `gorm:"type:text" json:"redirect_uri" bson:"redirect_uri" cql:"redirect_uri"`
}

func (v *VerificationRequest) AsAPIVerificationRequest() *model.VerificationRequest {
	token := v.Token
	createdAt := v.CreatedAt
	updatedAt := v.UpdatedAt
	email := v.Email
	nonce := v.Nonce
	redirectURI := v.RedirectURI
	expires := v.ExpiresAt
	identifier := v.Identifier
	return &model.VerificationRequest{
		ID:          v.ID,
		Token:       &token,
		Identifier:  &identifier,
		Expires:     &expires,
		CreatedAt:   &createdAt,
		UpdatedAt:   &updatedAt,
		Email:       &email,
		Nonce:       &nonce,
		RedirectURI: &redirectURI,
	}
}
