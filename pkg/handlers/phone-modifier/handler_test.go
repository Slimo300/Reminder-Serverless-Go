package phonemodifier_test

import (
	"context"
	"testing"

	phonemodifier "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/phone-modifier"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type mockSns struct{}

func (m *mockSns) Publish(context.Context, *sns.PublishInput) (*sns.PublishOutput, error) {
	return nil, nil
}

type mockDynamo struct{}

func (m *mockDynamo) PutItem(context.Context, *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return nil, nil
}

func TestHandler(t *testing.T) {

	handler := phonemodifier.Handler{
		DynamoClient: &mockDynamo{},
		SnsClient:    &mockSns{},
	}

	testCases := []struct {
		name               string
		request            events.APIGatewayProxyRequest
		expectedBody       string
		expectedStatusCode int
	}{
		{
			name: "no authorizer",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{},
			},
			expectedBody:       `{"message":"authorization data not found"}`,
			expectedStatusCode: 401,
		},
		{
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
			name: "no phone_number",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"authorization data not found"}`,
			expectedStatusCode: 401,
		},
		{
			name: "no subscription arn",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":          "1",
							"phone_number": "+11 123456789",
						},
					},
				},
			},
			expectedBody:       `{"message":"authorization data not found"}`,
			expectedStatusCode: 401,
		},
		{
			name: "no request body",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":                     "1",
							"phone_number":            "+11123456789",
							"custom:subscription_arn": "some_arn",
						},
					},
				},
			},
			expectedBody:       `{"message":"invalid request body"}`,
			expectedStatusCode: 400,
		},
		{
			name: "success",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":                     "1",
							"phone_number":            "+11123456789",
							"custom:subscription_arn": "some_arn",
						},
					},
				},
				Body: `{"phone_number":"+11987654321"}`,
			},
			expectedBody:       `{"message":"verification code sent"}`,
			expectedStatusCode: 200,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			res, err := handler.Handle(tC.request)
			if err != nil {
				t.Errorf("Error occured when handling request: %v", err)
			}

			if res.Body != tC.expectedBody {
				t.Errorf("Received result: %v is different than expected one: %v", res.Body, tC.expectedBody)
			}
			if res.StatusCode != tC.expectedStatusCode {
				t.Errorf("Received status code: %v is different than expected one: %v", res.StatusCode, tC.expectedStatusCode)
			}
		})
	}
}
