package alarmcreator_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	alarmcreator "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/alarm-creator"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

type mockDynamoDB struct {
}

func (m *mockDynamoDB) PutItem(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return nil, nil
}

type mockScheduler struct {
	*sync.Mutex
	counter   int
	failureAt int
}

func (m *mockScheduler) CreateSchedule(ctx context.Context, input *scheduler.CreateScheduleInput, opts ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error) {
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
		requestBody        alarmcreator.RequestBody
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
			name: "no message",
			requestBody: alarmcreator.RequestBody{
				Message:  "",
				Timezone: "Europe/Warsaw",
				Dates:    []string{"2012-12-04T12:12"},
				Crons:    []string{},
			},
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"\"message\" cannot be an empty string"}`,
			expectedStatusCode: 400,
		},
		{
			name: "no timezone",
			requestBody: alarmcreator.RequestBody{
				Message:  "some message",
				Timezone: "",
				Dates:    []string{"2012-12-04T12:12"},
				Crons:    []string{},
			},
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"\"timezone\" cannot be an empty string"}`,
			expectedStatusCode: 400,
		},
		{
			name: "no cron or date",
			requestBody: alarmcreator.RequestBody{
				Message:  "some message",
				Timezone: "Europe/Warsaw",
				Dates:    []string{},
				Crons:    []string{},
			},
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"there are no crons or dates specified"}`,
			expectedStatusCode: 400,
		},
		{
			name: "no cron or date",
			requestBody: alarmcreator.RequestBody{
				Message:  "some message",
				Timezone: "Europe/Warsaw",
				Dates:    []string{},
				Crons:    []string{},
			},
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"there are no crons or dates specified"}`,
			expectedStatusCode: 400,
		},
		{
			name: "context cancelation first off",
			requestBody: alarmcreator.RequestBody{
				Message:  "some message",
				Timezone: "Europe/Warsaw",
				Dates:    []string{"2012-12-04T12:12", "2013-12-04T12:12", "2014-12-04T12:12"},
				Crons:    []string{"0 10 4 10 * ? 2024", "0 10 4 11 * ? 2024", "0 10 4 12 * ? 2024"},
			},
			request: events.APIGatewayProxyRequest{
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
			requestBody: alarmcreator.RequestBody{
				Message:  "some message",
				Timezone: "Europe/Warsaw",
				Dates:    []string{"2012-12-04T12:12", "2013-12-04T12:12", "2014-12-04T12:12"},
				Crons:    []string{"0 10 4 10 * ? 2024", "0 10 4 11 * ? 2024", "0 10 4 12 * ? 2024"},
			},
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"internal server error"}`,
			failureAt:          3,
			returnResult:       true,
			expectedStatusCode: 500,
		},
		{
			name: "context cancelation last",
			requestBody: alarmcreator.RequestBody{
				Message:  "some message",
				Timezone: "Europe/Warsaw",
				Dates:    []string{"2012-12-04T12:12", "2013-12-04T12:12", "2014-12-04T12:12"},
				Crons:    []string{"0 10 4 10 * ? 2024", "0 10 4 11 * ? 2024", "0 10 4 12 * ? 2024"},
			},
			request: events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					Authorizer: map[string]interface{}{
						"claims": map[string]interface{}{
							"sub": "1",
						},
					},
				},
			},
			expectedBody:       `{"message":"internal server error"}`,
			failureAt:          6,
			returnResult:       true,
			expectedStatusCode: 500,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := alarmcreator.Handler{
				DynamoClient:    &mockDynamoDB{},
				SchedulerClient: &mockScheduler{failureAt: testCase.failureAt, Mutex: &sync.Mutex{}},
			}

			jsonBody, _ := json.Marshal(testCase.requestBody)
			testCase.request.Body = string(jsonBody)

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

func TestHandleSuccess(t *testing.T) {
	handler := alarmcreator.Handler{
		DynamoClient: &mockDynamoDB{},
		SchedulerClient: &mockScheduler{
			Mutex: &sync.Mutex{}},
	}

	requestBody := alarmcreator.RequestBody{
		Message:  "some message",
		Timezone: "Europe/Warsaw",
		Dates:    []string{"2012-12-04T12:12", "2013-12-04T12:12"},
		Crons:    []string{"0 10 4 10 * ? 2024", "0 10 4 11 * ? 2024"},
	}

	jsonRequestBody, _ := json.Marshal(requestBody)

	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]interface{}{
				"claims": map[string]interface{}{
					"sub": "1",
				},
			},
		},
		Body: string(jsonRequestBody),
	}

	res, err := handler.Handle(request)
	if err != nil {
		t.Errorf("Error occured during HandleSuccess test: %v", err)
	}

	var decodedResult map[string]interface{}
	if err := json.NewDecoder(strings.NewReader(res.Body)).Decode(&decodedResult); err != nil {
		t.Errorf("Error decoding response in HandleSuccess test: %v", err)
	}

	if decodedResult["Timezone"] != requestBody.Timezone {
		t.Errorf("Returned timezone: %v different than expected: %v", decodedResult["Timezone"], requestBody.Timezone)
	}
	if decodedResult["Title"] != requestBody.Message {
		t.Errorf("Returned message: %v different than expected: %v", decodedResult["Message"], requestBody.Message)
	}

	crons := decodedResult["Crons"].(map[string]interface{})
	dates := decodedResult["Dates"].(map[string]interface{})

	for key, resCron := range crons {
		for _, reqCron := range requestBody.Crons {
			if resCron == reqCron {
				delete(crons, key)
			}
		}
	}
	if len(crons) != 0 {
		t.Errorf("Unexpected cron expressions: %v", crons)
	}

	for key, resDate := range dates {
		for _, reqDate := range requestBody.Dates {
			if resDate == reqDate {
				delete(dates, key)
			}
		}
	}
	if len(crons) != 0 {
		t.Errorf("Unexpected dates expressions: %v", crons)
	}
}
