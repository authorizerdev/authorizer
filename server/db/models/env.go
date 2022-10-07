package models

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// Env model for db
type Env struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id"`
	EnvData   string `json:"env" bson:"env" cql:"env"`
	Hash      string `json:"hash" bson:"hash" cql:"hash"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at" cql:"updated_at"`
	CreatedAt int64  `json:"created_at" bson:"created_at" cql:"created_at"`
}
