package db

import (
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID              uint `gorm:"primaryKey"`
	FirstName       string
	LastName        string
	Email           string `gorm:"unique"`
	Password        string
	SignupMethod    string
	EmailVerifiedAt int64
	CreatedAt       int64 `gorm:"autoCreateTime"`
	UpdatedAt       int64 `gorm:"autoUpdateTime"`
	Image           string
}

func (user *User) BeforeSave(tx *gorm.DB) error {
	// Modify current operation through tx.Statement, e.g:
	if user.Password != "" {
		if pw, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost); err == nil {
			tx.Statement.SetColumn("Password", string(pw))
		}
	}

	return nil
}

// SaveUser function to add user
func (mgr *manager) SaveUser(user User) (User, error) {
	result := mgr.db.Clauses(clause.OnConflict{UpdateAll: true, Columns: []clause.Column{{Name: "email"}}}).Create(&user)

	if result.Error != nil {
		log.Println(result.Error)
		return user, result.Error
	}
	log.Println("===== USER ID =====")
	log.Println(user.ID)
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

func (mgr *manager) UpdateVerificationTime(verifiedAt int64, id uint) error {
	user := &User{
		ID: id,
	}
	result := mgr.db.Model(&user).Where("id = ?", id).Update("email_verified_at", verifiedAt)

	if result.Error != nil {
		return result.Error
	}

	return nil
}
