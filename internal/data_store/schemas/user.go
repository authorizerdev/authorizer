package schemas

import (
	"encoding/json"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// User model for db
type User struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID  string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	Email                    *string `gorm:"index" json:"email" bson:"email" cql:"email" dynamo:"email" index:"email,hash"`
	EmailVerifiedAt          *int64  `json:"email_verified_at" bson:"email_verified_at" cql:"email_verified_at" dynamo:"email_verified_at"`
	Password                 *string `json:"password" bson:"password" cql:"password" dynamo:"password"`
	SignupMethods            string  `json:"signup_methods" bson:"signup_methods" cql:"signup_methods" dynamo:"signup_methods"`
	GivenName                *string `json:"given_name" bson:"given_name" cql:"given_name" dynamo:"given_name"`
	FamilyName               *string `json:"family_name" bson:"family_name" cql:"family_name" dynamo:"family_name"`
	MiddleName               *string `json:"middle_name" bson:"middle_name" cql:"middle_name" dynamo:"middle_name"`
	Nickname                 *string `json:"nickname" bson:"nickname" cql:"nickname" dynamo:"nickname"`
	Gender                   *string `json:"gender" bson:"gender" cql:"gender" dynamo:"gender"`
	Birthdate                *string `json:"birthdate" bson:"birthdate" cql:"birthdate" dynamo:"birthdate"`
	PhoneNumber              *string `gorm:"index" json:"phone_number" bson:"phone_number" cql:"phone_number" dynamo:"phone_number"`
	PhoneNumberVerifiedAt    *int64  `json:"phone_number_verified_at" bson:"phone_number_verified_at" cql:"phone_number_verified_at" dynamo:"phone_number_verified_at"`
	Picture                  *string `json:"picture" bson:"picture" cql:"picture" dynamo:"picture"`
	Roles                    string  `json:"roles" bson:"roles" cql:"roles" dynamo:"roles"`
	RevokedTimestamp         *int64  `json:"revoked_timestamp" bson:"revoked_timestamp" cql:"revoked_timestamp" dynamo:"revoked_timestamp"`
	IsMultiFactorAuthEnabled *bool   `json:"is_multi_factor_auth_enabled" bson:"is_multi_factor_auth_enabled" cql:"is_multi_factor_auth_enabled" dynamo:"is_multi_factor_auth_enabled"`
	UpdatedAt                int64   `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
	CreatedAt                int64   `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	AppData                  *string `json:"app_data" bson:"app_data" cql:"app_data" dynamo:"app_data"`
}

func (user *User) AsAPIUser() *model.User {
	isEmailVerified := user.EmailVerifiedAt != nil
	isPhoneVerified := user.PhoneNumberVerifiedAt != nil
	appDataMap := make(map[string]interface{})
	json.Unmarshal([]byte(refs.StringValue(user.AppData)), &appDataMap)
	// id := user.ID
	// if strings.Contains(id, Collections.User+"/") {
	// 	id = strings.TrimPrefix(id, Collections.User+"/")
	// }
	return &model.User{
		ID:                       user.ID,
		Email:                    user.Email,
		EmailVerified:            isEmailVerified,
		SignupMethods:            user.SignupMethods,
		GivenName:                user.GivenName,
		FamilyName:               user.FamilyName,
		MiddleName:               user.MiddleName,
		Nickname:                 user.Nickname,
		PreferredUsername:        user.Email,
		Gender:                   user.Gender,
		Birthdate:                user.Birthdate,
		PhoneNumber:              user.PhoneNumber,
		PhoneNumberVerified:      &isPhoneVerified,
		Picture:                  user.Picture,
		Roles:                    strings.Split(user.Roles, ","),
		RevokedTimestamp:         user.RevokedTimestamp,
		IsMultiFactorAuthEnabled: user.IsMultiFactorAuthEnabled,
		CreatedAt:                refs.NewInt64Ref(user.CreatedAt),
		UpdatedAt:                refs.NewInt64Ref(user.UpdatedAt),
		AppData:                  appDataMap,
	}
}

func (user *User) ToMap() map[string]interface{} {
	res := map[string]interface{}{}
	data, _ := json.Marshal(user) // Convert to a json string
	json.Unmarshal(data, &res)    // Convert to a map
	return res
}
