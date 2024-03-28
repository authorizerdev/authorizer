package cassandradb

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// AddUser to save user information in database
func (p *provider) AddUser(ctx context.Context, user *models.User) (*models.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if user.Roles == "" {
		defaultRoles, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
		if err != nil {
			return user, err
		}
		user.Roles = defaultRoles
	}

	if user.PhoneNumber != nil && strings.TrimSpace(refs.StringValue(user.PhoneNumber)) != "" {
		if u, _ := p.GetUserByPhoneNumber(ctx, refs.StringValue(user.PhoneNumber)); u != nil && u.ID != user.ID {
			return user, fmt.Errorf("user with given phone number already exists")
		}
	} else if user.Email != nil && strings.TrimSpace(refs.StringValue(user.Email)) != "" {
		if u, _ := p.GetUserByEmail(ctx, refs.StringValue(user.Email)); u != nil && u.ID != user.ID {
			return user, fmt.Errorf("user with given email already exists")
		}
	}

	user.CreatedAt = time.Now().Unix()
	user.UpdatedAt = time.Now().Unix()

	bytes, err := json.Marshal(user)
	if err != nil {
		return user, err
	}

	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	userMap := map[string]interface{}{}
	err = decoder.Decode(&userMap)
	if err != nil {
		return user, err
	}

	fields := "("
	values := "("
	for key, value := range userMap {
		if value != nil {
			if key == "_id" {
				fields += "id,"
			} else {
				fields += key + ","
			}

			valueType := reflect.TypeOf(value)
			if valueType.Name() == "string" {
				values += fmt.Sprintf("'%s',", value.(string))
			} else {
				values += fmt.Sprintf("%v,", value)
			}
		}
	}

	fields = fields[:len(fields)-1] + ")"
	values = values[:len(values)-1] + ")"

	query := fmt.Sprintf("INSERT INTO %s %s VALUES %s IF NOT EXISTS", KeySpace+"."+models.Collections.User, fields, values)
	err = p.db.Query(query).Exec()

	if err != nil {
		return user, err
	}

	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	user.UpdatedAt = time.Now().Unix()

	bytes, err := json.Marshal(user)
	if err != nil {
		return user, err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	userMap := map[string]interface{}{}
	err = decoder.Decode(&userMap)
	if err != nil {
		return user, err
	}

	updateFields := ""
	for key, value := range userMap {
		if key == "_id" {
			continue
		}

		if key == "_key" {
			continue
		}

		if value == nil {
			updateFields += fmt.Sprintf("%s = null, ", key)
			continue
		}

		valueType := reflect.TypeOf(value)
		if valueType.Name() == "string" {
			updateFields += fmt.Sprintf("%s = '%s', ", key, value.(string))
		} else {
			updateFields += fmt.Sprintf("%s = %v, ", key, value)
		}
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = '%s'", KeySpace+"."+models.Collections.User, updateFields, user.ID)
	err = p.db.Query(query).Exec()
	if err != nil {
		return user, err
	}

	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *models.User) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+models.Collections.User, user.ID)
	err := p.db.Query(query).Exec()
	if err != nil {
		return err
	}
	getSessionsQuery := fmt.Sprintf("SELECT id FROM %s WHERE user_id = '%s' ALLOW FILTERING", KeySpace+"."+models.Collections.Session, user.ID)
	scanner := p.db.Query(getSessionsQuery).Iter().Scanner()
	sessionIDs := ""
	for scanner.Next() {
		var wlID string
		err = scanner.Scan(&wlID)
		if err != nil {
			return err
		}
		sessionIDs += fmt.Sprintf("'%s',", wlID)
	}
	sessionIDs = strings.TrimSuffix(sessionIDs, ",")
	deleteSessionQuery := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+models.Collections.Session, sessionIDs)
	err = p.db.Query(deleteSessionQuery).Exec()
	if err != nil {
		return err
	}

	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination) (*model.Users, error) {
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
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, app_data, created_at, updated_at FROM %s LIMIT %d", KeySpace+"."+models.Collections.User,
		pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var user models.User
			err := scanner.Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods,
				&user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber,
				&user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled,
				&user.AppData, &user.CreatedAt, &user.UpdatedAt)
			if err != nil {
				return nil, err
			}
			responseUsers = append(responseUsers, user.AsAPIUser())
		}
		counter++
	}
	return &model.Users{
		Pagination: paginationClone,
		Users:      responseUsers,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, app_data, created_at, updated_at FROM %s WHERE email = '%s' LIMIT 1 ALLOW FILTERING", KeySpace+"."+models.Collections.User, email)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled, &user.AppData, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, app_data, created_at, updated_at FROM %s WHERE id = '%s' LIMIT 1", KeySpace+"."+models.Collections.User, id)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled, &user.AppData, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUsers to update multiple users, with parameters of user IDs slice
// If ids set to nil / empty all the users will be updated
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	// set updated_at time for all users
	data["updated_at"] = time.Now().Unix()

	updateFields := ""
	for key, value := range data {
		if key == "_id" {
			continue
		}

		if key == "_key" {
			continue
		}

		if value == nil {
			updateFields += fmt.Sprintf("%s = null,", key)
			continue
		}

		valueType := reflect.TypeOf(value)
		if valueType.Name() == "string" {
			updateFields += fmt.Sprintf("%s = '%s', ", key, value.(string))
		} else {
			updateFields += fmt.Sprintf("%s = %v, ", key, value)
		}
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")
	query := ""
	if len(ids) > 0 {
		idsString := ""
		for _, id := range ids {
			idsString += fmt.Sprintf("'%s', ", id)
		}
		idsString = strings.Trim(idsString, " ")
		idsString = strings.TrimSuffix(idsString, ",")
		query = fmt.Sprintf("UPDATE %s SET %s WHERE id IN (%s)", KeySpace+"."+models.Collections.User, updateFields, idsString)
		err := p.db.Query(query).Exec()
		if err != nil {
			return err
		}
	} else {
		// get all ids
		getUserIDsQuery := fmt.Sprintf(`SELECT id FROM %s`, KeySpace+"."+models.Collections.User)
		scanner := p.db.Query(getUserIDsQuery).Iter().Scanner()
		// only 100 ids are allowed in 1 query
		// hence we need create multiple update queries
		idsString := ""
		idsStringArray := []string{idsString}
		counter := 1
		for scanner.Next() {
			var id string
			err := scanner.Scan(&id)
			if err == nil {
				idsString += fmt.Sprintf("'%s', ", id)
			}
			counter++
			if counter > 100 {
				idsStringArray = append(idsStringArray, idsString)
				counter = 1
				idsString = ""
			} else {
				// update the last index of array when count is less than 100
				idsStringArray[len(idsStringArray)-1] = idsString
			}
		}

		for _, idStr := range idsStringArray {
			idStr = strings.Trim(idStr, " ")
			idStr = strings.TrimSuffix(idStr, ",")
			query = fmt.Sprintf("UPDATE %s SET %s WHERE id IN (%s)", KeySpace+"."+models.Collections.User, updateFields, idStr)
			err := p.db.Query(query).Exec()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error) {
	var user models.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, app_data, created_at, updated_at FROM %s WHERE phone_number = '%s' LIMIT 1 ALLOW FILTERING", KeySpace+"."+models.Collections.User, phoneNumber)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled, &user.AppData, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
