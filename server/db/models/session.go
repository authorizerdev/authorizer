package models

// Session model for db
type Session struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`
	UserID    string `gorm:"type:char(36),index:" json:"user_id" bson:"user_id"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-" bson:"-"`
	UserAgent string `json:"user_agent" bson:"user_agent"`
	IP        string `json:"ip" bson:"ip"`
	CreatedAt int64  `json:"created_at" bson:"created_at"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at"`
}
