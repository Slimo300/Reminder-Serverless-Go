package dynamomapper

import "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

func SimplifyDynamoDBItem(item map[string]types.AttributeValue) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range item {
		switch v := value.(type) {
		case *types.AttributeValueMemberS:
			result[key] = v.Value
		case *types.AttributeValueMemberN:
			result[key] = v.Value
		case *types.AttributeValueMemberBOOL:
			result[key] = v.Value
		case *types.AttributeValueMemberM:
			subMap := make(map[string]interface{})
			for subKey, subValue := range v.Value {
				subMap[subKey] = SimplifyDynamoDBItem(map[string]types.AttributeValue{subKey: subValue})[subKey]
			}
			result[key] = subMap
		case *types.AttributeValueMemberL:
			var list []interface{}
			for _, subValue := range v.Value {
				list = append(list, SimplifyDynamoDBItem(map[string]types.AttributeValue{"": subValue})[""])
			}
			result[key] = list
		}
	}
	return result
}
