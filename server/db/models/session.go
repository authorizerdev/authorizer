package models

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// Session model for db
type Session struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id"`
	UserID    string `gorm:"type:char(36),index:" json:"user_id" bson:"user_id" cql:"user_id"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-" bson:"-" cql:"-"`
	UserAgent string `json:"user_agent" bson:"user_agent" cql:"user_agent"`
	IP        string `json:"ip" bson:"ip" cql:"ip"`
	CreatedAt int64  `json:"created_at" bson:"created_at" cql:"created_at"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at" cql:"updated_at"`
}
