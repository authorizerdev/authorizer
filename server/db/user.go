package db

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID              uuid.UUID `gorm:"primaryKey;type:char(36)"`
	FirstName       string
	LastName        string
	Email           string `gorm:"unique"`
	Password        string `gorm:"type:text"`
	SignupMethod    string
	EmailVerifiedAt int64
	CreatedAt       int64  `gorm:"autoCreateTime"`
	UpdatedAt       int64  `gorm:"autoUpdateTime"`
	Image           string `gorm:"type:text"`
	Roles           string
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = uuid.New()

	return
}

// SaveUser function to add user even with email conflict
func (mgr *manager) SaveUser(user User) (User, error) {
	result := mgr.db.Clauses(
		clause.OnConflict{
			UpdateAll: true,
			Columns:   []clause.Column{{Name: "email"}},
		}).Create(&user)

	if result.Error != nil {
		log.Println(result.Error)
		return user, result.Error
	}
	return user, nil
}

// UpdateUser function to update user with ID conflict
func (mgr *manager) UpdateUser(user User) (User, error) {
	user.UpdatedAt = time.Now().Unix()
	result := mgr.db.Clauses(
		clause.OnConflict{
			UpdateAll: true,
			Columns:   []clause.Column{{Name: "email"}},
		}).Create(&user)

	if result.Error != nil {
		log.Println(result.Error)
		return user, result.Error
	}
	return user, nil
}

// GetUsers function to get all users
func (mgr *manager) GetUsers() ([]User, error) {
	var users []User
	result := mgr.db.Find(&users)
	if result.Error != nil {
		log.Println(result.Error)
		return users, result.Error
	}
	return users, nil
}

func (mgr *manager) GetUserByEmail(email string) (User, error) {
	var user User
	result := mgr.db.Where("email = ?", email).First(&user)

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

func (mgr *manager) GetUserByID(id string) (User, error) {
	var user User
	result := mgr.db.Where("id = ?", id).First(&user)

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

func (mgr *manager) UpdateVerificationTime(verifiedAt int64, id uuid.UUID) error {
	user := &User{
		ID: id,
	}
	result := mgr.db.Model(&user).Where("id = ?", id).Update("email_verified_at", verifiedAt)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (mgr *manager) DeleteUser(email string) error {
	var user User
	result := mgr.db.Where("email = ?", email).Delete(&user)

	if result.Error != nil {
		log.Println(`Error deleting user:`, result.Error)
		return result.Error
	}

	return nil
}
