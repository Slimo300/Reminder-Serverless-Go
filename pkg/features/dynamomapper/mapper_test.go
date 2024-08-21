package dynamomapper_test

import (
	"reflect"
	"testing"

	"github.com/Slimo300/Reminder-Serverless-Go/pkg/features/dynamomapper"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestSimplifyDynamoDBItem(t *testing.T) {
	testCases := []struct {
		name           string
		item           map[string]types.AttributeValue
		expectedResult map[string]interface{}
	}{
		{
			name: "string type",
			item: map[string]types.AttributeValue{
				"some": &types.AttributeValueMemberS{Value: "thing"},
			},
			expectedResult: map[string]interface{}{
				"some": "thing",
			},
		},
		{
			name: "boolean type",
			item: map[string]types.AttributeValue{
				"isOK": &types.AttributeValueMemberBOOL{Value: true},
			},
			expectedResult: map[string]interface{}{
				"isOK": true,
			},
		},
		{
			name: "number type",
			item: map[string]types.AttributeValue{
				"some": &types.AttributeValueMemberN{Value: "125.10"},
			},
			expectedResult: map[string]interface{}{
				"some": "125.10",
			},
		},
		{
			name: "map type",
			item: map[string]types.AttributeValue{
				"some": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
					"thing": &types.AttributeValueMemberS{Value: "else"},
				}},
			},
			expectedResult: map[string]interface{}{
				"some": map[string]interface{}{
					"thing": "else",
				},
			},
		},
		{
			name: "string type",
			item: map[string]types.AttributeValue{
				"some": &types.AttributeValueMemberL{Value: []types.AttributeValue{
					&types.AttributeValueMemberS{Value: "thing"}, &types.AttributeValueMemberS{Value: "else"},
				}},
			},
			expectedResult: map[string]interface{}{
				"some": []interface{}{"thing", "else"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := dynamomapper.SimplifyDynamoDBItem(testCase.item)

			if !reflect.DeepEqual(result, testCase.expectedResult) {
				t.Errorf("Expected response: %v is different than actual one: %v", testCase.expectedResult, result)
			}
		})
	}
}
