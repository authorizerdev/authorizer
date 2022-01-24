package models

// Collections / Tables available for authorizer in the database
type CollectionList struct {
	User                string
	VerificationRequest string
	Session             string
	Env                 string
}

var (
	// Prefix for table name / collection names
	Prefix = "authorizer_"
	// Collections / Tables available for authorizer in the database (used for dbs other than gorm)
	Collections = CollectionList{
		User:                Prefix + "users",
		VerificationRequest: Prefix + "verification_requests",
		Session:             Prefix + "sessions",
		Env:                 Prefix + "env",
	}
)
