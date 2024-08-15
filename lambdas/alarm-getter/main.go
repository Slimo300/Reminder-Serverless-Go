package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	dynamoClient *dynamodb.Client
)

func errorResponse(message string, code int) (events.APIGatewayProxyResponse, error) {
	responseJSON, _ := json.Marshal(map[string]string{
		"message": message,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Headers":     "Content-Type",
			"Access-Control-Allow-Methods":     "OPTIONS, GET, POST, DELETE",
			"Access-Control-Allow-Credentials": "true",
		},
		Body: string(responseJSON),
	}, nil
}

// It returns internal server error
func internal(err error) (events.APIGatewayProxyResponse, error) {
	log.Println(err.Error())
	return errorResponse("internal server error", http.StatusInternalServerError)
}

// It returns bad request response with given message
func forbidden(message string) (events.APIGatewayProxyResponse, error) {
	return errorResponse(message, http.StatusForbidden)
}

func simplifyDynamoDBItems(items []map[string]types.AttributeValue) []map[string]interface{} {
	result := []map[string]interface{}{}
	for _, item := range items {
		result = append(result, simplifyDynamoDBItem(item))
	}

	return result
}

func simplifyDynamoDBItem(item map[string]types.AttributeValue) map[string]interface{} {
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
				subMap[subKey] = simplifyDynamoDBItem(map[string]types.AttributeValue{subKey: subValue})[subKey]
			}
			result[key] = subMap
		case *types.AttributeValueMemberL:
			var list []interface{}
			for _, subValue := range v.Value {
				list = append(list, simplifyDynamoDBItem(map[string]types.AttributeValue{"": subValue})[""])
			}
			result[key] = list
		}
	}
	return result
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{})
	if !ok {
		return forbidden("authorization data not found")
	}
	userID, ok := claims["sub"].(string)
	if !ok {
		return forbidden("authorization data not found")
	}

	response, err := dynamoClient.Query(context.Background(), &dynamodb.QueryInput{
		ExpressionAttributeNames: map[string]string{
			"#userID": "UserID",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userID": &types.AttributeValueMemberS{Value: userID},
		},
		KeyConditionExpression: aws.String("#userID = :userID"),
		TableName:              aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
	})
	if err != nil {
		return internal(err)
	}

	responseJSON, err := json.Marshal(simplifyDynamoDBItems(response.Items))
	if err != nil {
		return internal(err)
	}

	return events.APIGatewayProxyResponse{
		Body: string(responseJSON),
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Headers":     "Content-Type",
			"Access-Control-Allow-Methods":     "OPTIONS, GET, POST, DELETE",
			"Access-Control-Allow-Credentials": "true",
		},
		StatusCode: 200,
	}, nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return
	}

	dynamoClient = dynamodb.NewFromConfig(cfg)

	lambda.Start(Handler)
}
