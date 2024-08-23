package alarmgetter_test

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	alarmgetter "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/alarm-getter"
)

type mockDynamoDB struct {
	dynamodb.QueryAPIClient
	users map[string]bool
}

func (d *mockDynamoDB) Query(ctx context.Context, in *dynamodb.QueryInput, opts ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	userID := in.ExpressionAttributeValues[":userID"].(*types.AttributeValueMemberS).Value

	if !d.users[userID] {
		return &dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{},
		}, nil
	}

	return &dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{
			{
				"userID": &types.AttributeValueMemberS{Value: userID},
			},
		},
	}, nil
}

func TestHandler(t *testing.T) {

	handler := &alarmgetter.AlarmGetterHandler{
		DynamoClient: &mockDynamoDB{
			users: map[string]bool{
				"1": true,
			},
		},
	}

	testCases := []struct {
		name               string
		request            events.APIGatewayProxyRequest
		expectedBody       string
		expectedStatusCode int
	}{
		{
			// mock a request with an empty SourceIP
			name: "no authorizer",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{},
			},
			expectedBody:       `{"message":"authorization data not found"}`,
			expectedStatusCode: 401,
		},
		{
			// mock a request with an empty SourceIP
			name: "no sub",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{},
					},
				},
			},
			expectedBody:       `{"message":"authorization data not found"}`,
			expectedStatusCode: 401,
		},
		{
			name: "no data",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "2",
						},
					},
				},
			},
			expectedBody:       `[]`,
			expectedStatusCode: 200,
		},
		{
			name: "ok",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `[{"userID":"1"}]`,
			expectedStatusCode: 200,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			response, _ := handler.Handle(testCase.request)
			if response.Body != testCase.expectedBody {
				t.Errorf("Expected response %v, but got %v", testCase.expectedBody, response.Body)
			}

			if response.StatusCode != testCase.expectedStatusCode {
				t.Errorf("Expected status code 200, but got %v", response.StatusCode)
			}
		})
	}
}
