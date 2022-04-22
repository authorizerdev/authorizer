package cassandradb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// AddUser to save user information in database
func (p *provider) AddUser(user models.User) (models.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if user.Roles == "" {
		user.Roles = strings.Join(envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyDefaultRoles), ",")
	}

	user.CreatedAt = time.Now().Unix()
	user.UpdatedAt = time.Now().Unix()

	bytes, err := json.Marshal(user)
	if err != nil {
		return user, err
	}
	userMap := map[string]interface{}{}
	json.Unmarshal(bytes, &userMap)

	fields := "("
	values := "("
	for key, value := range userMap {
		if value != nil {
			fields += key + ","

			valueType := reflect.TypeOf(value)
			if valueType.Kind() == reflect.String {
				values += "'" + value.(string) + "',"
			} else {
				values += fmt.Sprintf("%v", value) + ","
			}
		}
	}

	fields = fields[:len(fields)-1] + ")"
	values = values[:len(values)-1] + ")"

	query := fmt.Sprintf("INSERT INTO %s %s VALUES %s", KeySpace+"."+models.Collections.User, fields, values)

	err = p.db.Query(query).Exec()
	if err != nil {
		return user, err
	}

	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(user models.User) (models.User, error) {
	user.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(user)
	if err != nil {
		return user, err
	}
	userMap := map[string]interface{}{}
	json.Unmarshal(bytes, &userMap)

	updateFields := ""
	for key, value := range userMap {
		if value != nil {
			valueType := reflect.TypeOf(value)
			if valueType.Kind() == reflect.String {
				updateFields += key + " = '" + value.(string) + "',"
			} else {
				updateFields += key + " = " + fmt.Sprintf("%v", value) + ","
			}
		}
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = '%s'", KeySpace+"."+models.Collections.User, updateFields, user.ID)
	err = p.db.Query(query).Exec()
	if err != nil {
		return user, err
	}

	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(user models.User) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+models.Collections.User, user.ID)

	return p.db.Query(query).Exec()
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(pagination model.Pagination) (*model.Users, error) {
	responseUsers := []*model.User{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+models.Collections.User)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, err
	}

	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, created_at, updated_at FROM %s ORDER BY created_at DESC LIMIT %d", KeySpace+"."+models.Collections.User, pagination.Limit+pagination.Offset)

	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var user models.User
			err := scanner.Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.CreatedAt, &user.UpdatedAt)
			if err != nil {
				return nil, err
			}
			responseUsers = append(responseUsers, user.AsAPIUser())
		}
		counter++
	}
	return &model.Users{
		Users:      responseUsers,
		Pagination: &paginationClone,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(email string) (models.User, error) {
	var user models.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, created_at, updated_at FROM %s WHERE email = '%s' LIMIT 1", KeySpace+"."+models.Collections.User, email)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return user, err
	}
	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(id string) (models.User, error) {
	var user models.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, created_at, updated_at FROM %s WHERE id = '%s' LIMIT 1", KeySpace+"."+models.Collections.User, id)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return user, err
	}
	return user, nil
}
