package phoneverifier_test

import (
	"context"
	"errors"
	"testing"

	phoneverifier "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/phone-verifier"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type mockSns struct {
	SubscribeError   error
	UnsubscribeError error
}

func (m *mockSns) Subscribe(context.Context, *sns.SubscribeInput, ...func(*sns.Options)) (*sns.SubscribeOutput, error) {
	return &sns.SubscribeOutput{
		SubscriptionArn: aws.String("some_arn"),
	}, m.SubscribeError
}
func (m *mockSns) Unsubscribe(context.Context, *sns.UnsubscribeInput, ...func(*sns.Options)) (*sns.UnsubscribeOutput, error) {
	return nil, m.UnsubscribeError
}

type mockDynamo struct {
	DeleteItemError error
}

func (m *mockDynamo) GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return &dynamodb.GetItemOutput{
		Item: map[string]dynamotypes.AttributeValue{
			"UserID":           &dynamotypes.AttributeValueMemberS{Value: "1"},
			"VerificationCode": &dynamotypes.AttributeValueMemberS{Value: "123456"},
			"SubscriptionArn":  &dynamotypes.AttributeValueMemberS{Value: "some arn"},
			"PhoneNumber":      &dynamotypes.AttributeValueMemberS{Value: "+11123456789"},
		},
	}, nil
}
func (m *mockDynamo) DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return nil, m.DeleteItemError
}

type mockCognito struct {
	AdminUpdateError error
}

func (m *mockCognito) AdminUpdateUserAttributes(context.Context, *cognito.AdminUpdateUserAttributesInput, ...func(*cognito.Options)) (*cognito.AdminUpdateUserAttributesOutput, error) {
	return nil, m.AdminUpdateError
}

func TestHandler(t *testing.T) {
	testCases := []struct {
		name               string
		request            events.APIGatewayProxyRequest
		expectedBody       string
		expectedStatusCode int
		deleteItemError    error
		subscribeError     error
		unsubscribeError   error
		adminUpdateError   error
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
			name: "no username",
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
			name: "no request body",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":              "1",
							"cognito:username": "user",
						},
					},
				},
			},
			expectedBody:       `{"message":"invalid request body"}`,
			expectedStatusCode: 400,
		},
		{
			name: "verification code incorrect",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":              "1",
							"cognito:username": "user",
						},
					},
				},
				Body: `{"verification_code":"123"}`,
			},
			expectedBody:       `{"message":"verification code is incorrect"}`,
			expectedStatusCode: 401,
		},
		{
			name: "delete item error",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":              "1",
							"cognito:username": "user",
						},
					},
				},
				Body: `{"verification_code":"123456"}`,
			},
			deleteItemError:    errors.New("some error"),
			expectedBody:       `{"message":"internal server error"}`,
			expectedStatusCode: 500,
		},
		{
			name: "subscribe error",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":              "1",
							"cognito:username": "user",
						},
					},
				},
				Body: `{"verification_code":"123456"}`,
			},
			subscribeError:     errors.New("some error"),
			expectedBody:       `{"message":"internal server error"}`,
			expectedStatusCode: 500,
		},
		{
			name: "unsubscribe error",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":              "1",
							"cognito:username": "user",
						},
					},
				},
				Body: `{"verification_code":"123456"}`,
			},
			unsubscribeError:   errors.New("some error"),
			expectedBody:       `{"message":"internal server error"}`,
			expectedStatusCode: 500,
		},
		{
			name: "admin update error",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":              "1",
							"cognito:username": "user",
						},
					},
				},
				Body: `{"verification_code":"123456"}`,
			},
			adminUpdateError:   errors.New("some error"),
			expectedBody:       `{"message":"internal server error"}`,
			expectedStatusCode: 500,
		},
		{
			name: "success",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub":              "1",
							"cognito:username": "user",
						},
					},
				},
				Body: `{"verification_code":"123456"}`,
			},
			expectedBody:       `{"phone_number":"+11123456789"}`,
			expectedStatusCode: 200,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			handler := phoneverifier.Handler{
				DynamoClient:  &mockDynamo{DeleteItemError: tC.deleteItemError},
				SnsClient:     &mockSns{SubscribeError: tC.subscribeError, UnsubscribeError: tC.unsubscribeError},
				CognitoClient: &mockCognito{AdminUpdateError: tC.adminUpdateError},
			}

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
