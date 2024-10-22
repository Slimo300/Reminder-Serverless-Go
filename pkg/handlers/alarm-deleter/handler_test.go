package alarmdeleter_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	alarmdeleter "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/alarm-deleter"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

type mockDynamoDB struct {
}

func (m *mockDynamoDB) GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return &dynamodb.GetItemOutput{
		Item: map[string]dynamotypes.AttributeValue{
			"Crons": &dynamotypes.AttributeValueMemberM{
				Value: map[string]dynamotypes.AttributeValue{
					"1": &dynamotypes.AttributeValueMemberS{Value: "1"},
					"2": &dynamotypes.AttributeValueMemberS{Value: "2"},
				},
			},
			"Dates": &dynamotypes.AttributeValueMemberM{
				Value: map[string]dynamotypes.AttributeValue{
					"1": &dynamotypes.AttributeValueMemberS{Value: "1"},
					"2": &dynamotypes.AttributeValueMemberS{Value: "2"},
				},
			},
		},
	}, nil
}
func (m *mockDynamoDB) DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return nil, nil
}

type mockScheduler struct {
	*sync.Mutex
	counter   int
	failureAt int
}

func (m *mockScheduler) DeleteSchedule(ctx context.Context, input *scheduler.DeleteScheduleInput, opts ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
	m.Lock()
	m.counter++
	m.Unlock()

	if ctx.Err() != nil {
		return nil, context.Canceled
	}
	if m.counter == m.failureAt {
		return nil, errors.New("some error")
	}
	return nil, nil
}

func TestHandler(t *testing.T) {
	testCases := []struct {
		name               string
		request            events.APIGatewayProxyRequest
		expectedBody       string
		expectedStatusCode int
		returnResult       bool
		failureAt          int
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
			name: "no path parameter",
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"no eventID specified"}`,
			expectedStatusCode: 400,
		},
		{
			name: "context cancelation first off",
			request: events.APIGatewayProxyRequest{
				PathParameters: map[string]string{
					"id": "1",
				},
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"internal server error"}`,
			failureAt:          1,
			returnResult:       true,
			expectedStatusCode: 500,
		}, {
			name: "context cancelation middle",
			request: events.APIGatewayProxyRequest{
				PathParameters: map[string]string{
					"id": "1",
				},
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"internal server error"}`,
			failureAt:          2,
			returnResult:       true,
			expectedStatusCode: 500,
		},
		{
			name: "context cancelation last",
			request: events.APIGatewayProxyRequest{
				PathParameters: map[string]string{
					"id": "1",
				},
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"internal server error"}`,
			failureAt:          4,
			returnResult:       true,
			expectedStatusCode: 500,
		},
		{
			name: "success",
			request: events.APIGatewayProxyRequest{
				PathParameters: map[string]string{
					"id": "1",
				},
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"ok"}`,
			failureAt:          0,
			returnResult:       true,
			expectedStatusCode: 200,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := alarmdeleter.Handler{
				DynamoClient:    &mockDynamoDB{},
				SchedulerClient: &mockScheduler{failureAt: testCase.failureAt, Mutex: &sync.Mutex{}},
			}
			response, _ := handler.Handle(testCase.request)
			if response.Body != testCase.expectedBody {
				t.Errorf("Expected response %v, but got %v", testCase.expectedBody, response.Body)
			}
			if response.StatusCode != testCase.expectedStatusCode {
				t.Errorf("Expected status code %v, but got %v", testCase.expectedStatusCode, response.StatusCode)
			}
		})
	}
}
