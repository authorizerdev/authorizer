package schemas

// Note: any change here should be reflected in providers/cassandra/provider.go as it does not have model support in collection creation

// Application represents a machine-to-machine (M2M) application / service account
type Application struct {
	Key          string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID           string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	Name         string `gorm:"type:varchar(256);uniqueIndex" json:"name" bson:"name" cql:"name" dynamo:"name"`
	Description  string `gorm:"type:text" json:"description" bson:"description" cql:"description" dynamo:"description"`
	ClientID     string `gorm:"type:char(36);uniqueIndex" json:"client_id" bson:"client_id" cql:"client_id" dynamo:"client_id"`
	ClientSecret string `gorm:"type:text" json:"client_secret" bson:"client_secret" cql:"client_secret" dynamo:"client_secret"`
	Scopes       string `gorm:"type:text" json:"scopes" bson:"scopes" cql:"scopes" dynamo:"scopes"`
	Roles        string `gorm:"type:text" json:"roles" bson:"roles" cql:"roles" dynamo:"roles"`
	IsActive     bool   `json:"is_active" bson:"is_active" cql:"is_active" dynamo:"is_active"`
	CreatedBy    string `gorm:"type:char(36)" json:"created_by" bson:"created_by" cql:"created_by" dynamo:"created_by"`
	CreatedAt    int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt    int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
