package alarmdeleter_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

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
	counter          int
	executionPlanner []schedulerExecution
}

type schedulerExecution struct {
	executionTime time.Duration
	returnedError error
}

func (m *mockScheduler) DeleteSchedule(context.Context, *scheduler.DeleteScheduleInput, ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
	m.Lock()
	executionDetails := m.executionPlanner[m.counter]
	m.counter++
	m.Unlock()

	time.Sleep(executionDetails.executionTime)
	return nil, executionDetails.returnedError
}

func TestHandler(t *testing.T) {

	testCases := []struct {
		name               string
		request            events.APIGatewayProxyRequest
		expectedBody       string
		expectedStatusCode int
		returnResult       bool
		executionDetails   []schedulerExecution
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
			expectedBody: `{"message":"internal server error"}`,
			executionDetails: []schedulerExecution{
				{
					executionTime: 1 * time.Second,
					returnedError: errors.New("some error"),
				},
				{
					executionTime: 2 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 2 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 2 * time.Second,
					returnedError: nil,
				},
			},
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
			expectedBody: `{"message":"internal server error"}`,
			executionDetails: []schedulerExecution{
				{
					executionTime: 2 * time.Second,
					returnedError: errors.New("some error"),
				},
				{
					executionTime: 1 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 3 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 3 * time.Second,
					returnedError: nil,
				},
			},
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
			expectedBody: `{"message":"internal server error"}`,
			executionDetails: []schedulerExecution{
				{
					executionTime: 2 * time.Second,
					returnedError: errors.New("some error"),
				},
				{
					executionTime: 1 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 1 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 1 * time.Second,
					returnedError: nil,
				},
			},
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
			expectedBody: `{"message":"ok"}`,
			executionDetails: []schedulerExecution{
				{
					executionTime: 1 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 1 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 1 * time.Second,
					returnedError: nil,
				},
				{
					executionTime: 1 * time.Second,
					returnedError: nil,
				},
			},
			returnResult:       true,
			expectedStatusCode: 200,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			handler := alarmdeleter.Handler{
				DynamoClient:    &mockDynamoDB{},
				SchedulerClient: &mockScheduler{executionPlanner: testCase.executionDetails, Mutex: &sync.Mutex{}},
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
