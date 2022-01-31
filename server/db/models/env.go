package models

// Env model for db
type Env struct {
	Key       string `json:"_key,omitempty" bson:"_key"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`
	EnvData   string `gorm:"type:text" json:"env" bson:"env"`
	Hash      string `gorm:"type:text" json:"hash" bson:"hash"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at"`
	CreatedAt int64  `json:"created_at" bson:"created_at"`
}
