package dynamodb

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/guregu/dynamo"
)

// As updpate all item not supported so set manually via Set and SetNullable for empty field
func UpdateByHashKey(table dynamo.Table, hashKey string, hashValue string, item interface{}) error {
	existingValue, err := dynamo.MarshalItem(item)
	var i interface{}

	if err != nil {
		return err
	}

	nullableValue, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}

	u := table.Update(hashKey, hashValue)
	for k, v := range existingValue {
		if k == hashKey {
			continue
		}
		u = u.Set(k, v)
	}

	for k, v := range nullableValue {
		if k == hashKey {
			continue
		}
		dynamodbattribute.Unmarshal(v, &i)
		if i == nil {
			u = u.SetNullable(k, v)
		}
	}

	err = u.Run()
	if err != nil {
		return err
	}

	return nil
}
