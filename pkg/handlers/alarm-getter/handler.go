package alarmgetter

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/Slimo300/Reminder-Serverless-Go/pkg/features/dynamomapper"
	"github.com/Slimo300/Reminder-Serverless-Go/pkg/features/errors"
)

type AlarmGetterHandler struct {
	DynamoClient dynamodb.QueryAPIClient
}

func (h *AlarmGetterHandler) Handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	claims, ok := request.RequestContext.Authorizer["claims"]
	if !ok {
		return errors.Unauthorized("authorization data not found")
	}
	userID, ok := claims.(map[string]interface{})["sub"].(string)
	if !ok {
		return errors.Unauthorized("authorization data not found")
	}

	response, err := h.DynamoClient.Query(context.Background(), &dynamodb.QueryInput{
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
		return errors.Internal(err)
	}

	result := []map[string]interface{}{}
	for _, item := range response.Items {
		result = append(result, dynamomapper.SimplifyDynamoDBItem(item))
	}

	responseJSON, err := json.Marshal(result)
	if err != nil {
		return errors.Internal(err)
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
