package dynamodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (p *provider) putItem(ctx context.Context, table string, v interface{}) error {
	item, err := marshalStruct(v)
	if err != nil {
		return err
	}
	_, err = p.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      item,
	})
	return err
}

// putItemIfAbsent writes an item only when no item with the same partition key
// (hashKey) already exists. The condition is evaluated atomically by DynamoDB,
// so it is a race-free first-writer-wins insert. On a lost race it returns a
// *types.ConditionalCheckFailedException (detect with errors.As).
func (p *provider) putItemIfAbsent(ctx context.Context, table, hashKey string, v interface{}) error {
	item, err := marshalStruct(v)
	if err != nil {
		return err
	}
	_, err = p.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(table),
		Item:                item,
		ConditionExpression: aws.String(fmt.Sprintf("attribute_not_exists(%s)", hashKey)),
	})
	return err
}

func (p *provider) getItemByHash(ctx context.Context, table, hashKey, hashValue string, out interface{}) error {
	res, err := p.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key: map[string]types.AttributeValue{
			hashKey: &types.AttributeValueMemberS{Value: hashValue},
		},
	})
	if err != nil {
		return err
	}
	if len(res.Item) == 0 {
		return fmt.Errorf("no record found")
	}
	return unmarshalItem(res.Item, out)
}

func (p *provider) deleteItemByHash(ctx context.Context, table, hashKey, hashValue string) error {
	_, err := p.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(table),
		Key: map[string]types.AttributeValue{
			hashKey: &types.AttributeValueMemberS{Value: hashValue},
		},
	})
	return err
}

func (p *provider) scanAllRaw(ctx context.Context, table string, index *string, filter *expression.ConditionBuilder) ([]map[string]types.AttributeValue, error) {
	var built expression.Expression
	var hasFilter bool
	if filter != nil {
		e, err := expression.NewBuilder().WithFilter(*filter).Build()
		if err != nil {
			return nil, err
		}
		built = e
		hasFilter = true
	}
	var out []map[string]types.AttributeValue
	var start map[string]types.AttributeValue
	for {
		in := &dynamodb.ScanInput{
			TableName:         aws.String(table),
			ExclusiveStartKey: start,
		}
		if index != nil {
			in.IndexName = index
		}
		if hasFilter {
			in.FilterExpression = built.Filter()
			in.ExpressionAttributeNames = built.Names()
			in.ExpressionAttributeValues = built.Values()
		}
		res, err := p.client.Scan(ctx, in)
		if err != nil {
			return nil, err
		}
		out = append(out, res.Items...)
		if res.LastEvaluatedKey == nil {
			break
		}
		start = res.LastEvaluatedKey
	}
	return out, nil
}

func (p *provider) scanCount(ctx context.Context, table string, filter *expression.ConditionBuilder) (int64, error) {
	var built expression.Expression
	var hasFilter bool
	if filter != nil {
		e, err := expression.NewBuilder().WithFilter(*filter).Build()
		if err != nil {
			return 0, err
		}
		built = e
		hasFilter = true
	}
	var total int64
	var start map[string]types.AttributeValue
	for {
		in := &dynamodb.ScanInput{
			TableName:         aws.String(table),
			Select:            types.SelectCount,
			ExclusiveStartKey: start,
		}
		if hasFilter {
			in.FilterExpression = built.Filter()
			in.ExpressionAttributeNames = built.Names()
			in.ExpressionAttributeValues = built.Values()
		}
		res, err := p.client.Scan(ctx, in)
		if err != nil {
			return 0, err
		}
		total += int64(res.Count)
		if res.LastEvaluatedKey == nil {
			break
		}
		start = res.LastEvaluatedKey
	}
	return total, nil
}

func (p *provider) queryEq(ctx context.Context, table, indexName, pkAttr, pkVal string, filter *expression.ConditionBuilder) ([]map[string]types.AttributeValue, error) {
	kc := expression.Key(pkAttr).Equal(expression.Value(pkVal))
	var eb expression.Builder
	if filter != nil {
		eb = expression.NewBuilder().WithKeyCondition(kc).WithFilter(*filter)
	} else {
		eb = expression.NewBuilder().WithKeyCondition(kc)
	}
	expr, err := eb.Build()
	if err != nil {
		return nil, err
	}
	var out []map[string]types.AttributeValue
	var start map[string]types.AttributeValue
	for {
		in := &dynamodb.QueryInput{
			TableName:                 aws.String(table),
			IndexName:                 aws.String(indexName),
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			ExclusiveStartKey:         start,
		}
		if filter != nil {
			in.FilterExpression = expr.Filter()
		}
		res, err := p.client.Query(ctx, in)
		if err != nil {
			return nil, err
		}
		out = append(out, res.Items...)
		if res.LastEvaluatedKey == nil {
			break
		}
		start = res.LastEvaluatedKey
	}
	return out, nil
}

func (p *provider) queryEqLimit(ctx context.Context, table, indexName, pkAttr, pkVal string, filter *expression.ConditionBuilder, limit int32) ([]map[string]types.AttributeValue, error) {
	kc := expression.Key(pkAttr).Equal(expression.Value(pkVal))
	var eb expression.Builder
	if filter != nil {
		eb = expression.NewBuilder().WithKeyCondition(kc).WithFilter(*filter)
	} else {
		eb = expression.NewBuilder().WithKeyCondition(kc)
	}
	expr, err := eb.Build()
	if err != nil {
		return nil, err
	}
	in := &dynamodb.QueryInput{
		TableName:                 aws.String(table),
		IndexName:                 aws.String(indexName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Limit:                     aws.Int32(limit),
	}
	if filter != nil {
		in.FilterExpression = expr.Filter()
	}
	res, err := p.client.Query(ctx, in)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// queryEqUntil pages through the pkAttr=pkVal partition (like queryEq) but
// stops as soon as match reports a hit, instead of materializing every page
// before filtering. maxItems is a defensive circuit-breaker against unbounded
// scan cost on a pathologically large partition — NOT a realistic ceiling;
// normal callers exhaust the partition long before reaching it. match may
// return an error (e.g. a decode failure), which aborts the scan immediately
// rather than being swallowed as a non-match; on a hit it is responsible for
// capturing whatever it needs from the item (queryEqUntil only reports found/
// not-found, it does not return the raw item — callers already decode it
// inside match to test it, so returning it again would be redundant).
func (p *provider) queryEqUntil(ctx context.Context, table, indexName, pkAttr, pkVal string, maxItems int, match func(map[string]types.AttributeValue) (bool, error)) (bool, error) {
	kc := expression.Key(pkAttr).Equal(expression.Value(pkVal))
	expr, err := expression.NewBuilder().WithKeyCondition(kc).Build()
	if err != nil {
		return false, err
	}
	var start map[string]types.AttributeValue
	examined := 0
	for {
		in := &dynamodb.QueryInput{
			TableName:                 aws.String(table),
			IndexName:                 aws.String(indexName),
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			ExclusiveStartKey:         start,
		}
		res, err := p.client.Query(ctx, in)
		if err != nil {
			return false, err
		}
		for _, it := range res.Items {
			ok, err := match(it)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
			examined++
			if examined >= maxItems {
				p.dependencies.Log.Warn().Str("table", table).Str("partition_key", pkVal).Int("examined", examined).
					Msg("queryEqUntil: hit the scan safety cap without a match")
				return false, nil
			}
		}
		if res.LastEvaluatedKey == nil {
			return false, nil
		}
		start = res.LastEvaluatedKey
	}
}

func (p *provider) scanFilteredLimit(ctx context.Context, table string, index *string, filter *expression.ConditionBuilder, limit int32) ([]map[string]types.AttributeValue, error) {
	in := &dynamodb.ScanInput{
		TableName: aws.String(table),
		Limit:     aws.Int32(limit),
	}
	if index != nil {
		in.IndexName = index
	}
	if filter != nil {
		expr, err := expression.NewBuilder().WithFilter(*filter).Build()
		if err != nil {
			return nil, err
		}
		in.FilterExpression = expr.Filter()
		in.ExpressionAttributeNames = expr.Names()
		in.ExpressionAttributeValues = expr.Values()
	}
	res, err := p.client.Scan(ctx, in)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func (p *provider) scanFilteredAll(ctx context.Context, table string, index *string, filter *expression.ConditionBuilder) ([]map[string]types.AttributeValue, error) {
	return p.scanAllRaw(ctx, table, index, filter)
}

func (p *provider) updateByHashKey(ctx context.Context, tableName, hashKeyName, hashValue string, item interface{}) error {
	return p.updateByHashKeyWithRemoves(ctx, tableName, hashKeyName, hashValue, item, nil)
}

// updateByHashKeyWithRemoves runs UpdateItem with SET from marshalled fields and optional REMOVE
// of attribute names (e.g. when mapping SQL NULL for optional pointer fields — nil is omitted from
// SET but the old DynamoDB attribute must be explicitly removed).
func (p *provider) updateByHashKeyWithRemoves(ctx context.Context, tableName, hashKeyName, hashValue string, item interface{}, removeAttrs []string) error {
	var attrs map[string]types.AttributeValue
	var err error
	switch m := item.(type) {
	case map[string]interface{}:
		attrs, err = marshalMapStringInterface(m)
	default:
		attrs, err = marshalStruct(item)
	}
	if err != nil {
		return err
	}
	delete(attrs, hashKeyName)

	names := map[string]string{}
	vals := map[string]types.AttributeValue{}
	var sets []string
	i := 0
	for k, v := range attrs {
		nk := "#n" + fmt.Sprint(i)
		vk := ":v" + fmt.Sprint(i)
		names[nk] = k
		vals[vk] = v
		sets = append(sets, nk+" = "+vk)
		i++
	}

	var removeParts []string
	for j, attr := range removeAttrs {
		rk := "#r" + fmt.Sprint(j)
		names[rk] = attr
		removeParts = append(removeParts, rk)
	}

	if len(sets) == 0 && len(removeParts) == 0 {
		return nil
	}

	var exprParts []string
	if len(sets) > 0 {
		exprParts = append(exprParts, "SET "+strings.Join(sets, ", "))
	}
	if len(removeParts) > 0 {
		exprParts = append(exprParts, "REMOVE "+strings.Join(removeParts, ", "))
	}
	updateExpr := strings.Join(exprParts, " ")

	in := &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			hashKeyName: &types.AttributeValueMemberS{Value: hashValue},
		},
		UpdateExpression:         aws.String(updateExpr),
		ExpressionAttributeNames: names,
	}
	if len(vals) > 0 {
		in.ExpressionAttributeValues = vals
	}
	_, err = p.client.UpdateItem(ctx, in)
	return err
}

func (p *provider) scanPageIter(ctx context.Context, table string, filter *expression.ConditionBuilder, pageLimit int32, startKey map[string]types.AttributeValue) ([]map[string]types.AttributeValue, map[string]types.AttributeValue, error) {
	in := &dynamodb.ScanInput{
		TableName:         aws.String(table),
		Limit:             aws.Int32(pageLimit),
		ExclusiveStartKey: startKey,
	}
	if filter != nil {
		expr, err := expression.NewBuilder().WithFilter(*filter).Build()
		if err != nil {
			return nil, nil, err
		}
		in.FilterExpression = expr.Filter()
		in.ExpressionAttributeNames = expr.Names()
		in.ExpressionAttributeValues = expr.Values()
	}
	res, err := p.client.Scan(ctx, in)
	if err != nil {
		return nil, nil, err
	}
	return res.Items, res.LastEvaluatedKey, nil
}

func strPtr(s string) *string { return &s }
