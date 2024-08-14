package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/scheduler"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

var dynamoClient *dynamodb.DynamoDB
var schedulerClient *scheduler.Scheduler

type RuleEntry struct {
	RuleID string
	Value  string
}

type ScheduleType int

const (
	AT ScheduleType = iota
	CRON
)

func (t ScheduleType) String() string {
	switch t {
	case AT:
		return "at"
	case CRON:
		return "cron"
	default:
		panic("wrong schedule type")
	}
}

type createScheduleInput struct {
	RuleID             string
	ScheduleExpression string
	Timezone           string
	Message            string
	UserID             string
	ScheduleType       ScheduleType
}

func CreateSchedule(ctx context.Context, input createScheduleInput) error {
	ruleID := uuid.NewString()

	if err := ctx.Err(); err != nil {
		return err
	}

	lambdaInput, err := json.Marshal(map[string]string{
		"userID":  input.UserID,
		"message": input.Message,
	})
	if err != nil {
		return err
	}

	if _, err := schedulerClient.CreateScheduleWithContext(ctx, &scheduler.CreateScheduleInput{
		ActionAfterCompletion:      aws.String(scheduler.ActionAfterCompletionDelete),
		Description:                &input.Message,
		Name:                       &ruleID,
		ScheduleExpression:         aws.String(fmt.Sprintf("%s(%s)", input.ScheduleType.String(), input.ScheduleExpression)),
		ScheduleExpressionTimezone: &input.Timezone,
		Target: &scheduler.Target{
			Arn:     aws.String(os.Getenv("LAMBDA_FUNCTION_ARN")),
			RoleArn: aws.String(os.Getenv("ROLE_ARN")),
			Input:   aws.String(string(lambdaInput)),
		},
		FlexibleTimeWindow: &scheduler.FlexibleTimeWindow{
			Mode: aws.String("OFF"),
		},
	}); err != nil {
		return err
	}

	return nil
}

// It returns internal server error
func Internal() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"error": "internal server error"}`,
	}, nil
}

// It returns bad request response with given message
func BadRequest(message string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusBadRequest,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       fmt.Sprintf(`{"error": "%s"}`, message),
	}, nil
}

// It returns bad request response with given message
func Forbidden(message string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusForbidden,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       fmt.Sprintf(`{"error": "%s"}`, message),
	}, nil
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	token, ok := request.Headers["Authorization"]
	if !ok {
		return Forbidden("no authorization header")
	}

	decodedToken, _, err := jwt.NewParser(nil).ParseUnverified(token, &jwt.RegisteredClaims{})
	if err != nil {
		return Internal()
	}

	userID := decodedToken.Claims.(*jwt.RegisteredClaims).Subject
	if userID == "" {
		return BadRequest("invalid token payload")
	}

	var reqBody struct {
		Message  string   `json:"message"`
		Dates    []string `json:"dates"`
		Crons    []string `json:"crons"`
		Timezone string   `json:"timezone"`
	}

	if err := json.Unmarshal([]byte(request.Body), &reqBody); err != nil {
		return BadRequest("invalid request body")
	}

	dateMap := make(map[string]*dynamodb.AttributeValue)
	cronMap := make(map[string]*dynamodb.AttributeValue)

	dateChan := make(chan *RuleEntry)
	cronChan := make(chan *RuleEntry)
	errors := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case entry := <-dateChan:
				dateMap[entry.RuleID] = &dynamodb.AttributeValue{
					S: &entry.Value,
				}
			case entry := <-cronChan:
				cronMap[entry.RuleID] = &dynamodb.AttributeValue{
					S: &entry.Value,
				}
			case err := <-errors:
				log.Println(err.Error())
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup

	for _, date := range reqBody.Dates {

		wg.Add(1)
		go func(expr string) {
			ruleID := uuid.NewString()

			if err := CreateSchedule(ctx, createScheduleInput{
				RuleID:             ruleID,
				UserID:             userID,
				ScheduleExpression: expr,
				ScheduleType:       AT,
				Message:            reqBody.Message,
				Timezone:           reqBody.Timezone,
			}); err != nil {
				errors <- err
			}

			// sending entry to dateChan for further handling
			dateChan <- &RuleEntry{
				RuleID: ruleID,
				Value:  expr,
			}

			wg.Done()
		}(date)
	}

	for _, cron := range reqBody.Crons {

		wg.Add(1)
		go func(expr string) {
			ruleID := uuid.NewString()

			if err := CreateSchedule(ctx, createScheduleInput{
				RuleID:             ruleID,
				UserID:             userID,
				ScheduleExpression: expr,
				ScheduleType:       CRON,
				Message:            reqBody.Message,
				Timezone:           reqBody.Timezone,
			}); err != nil {
				errors <- err
			}

			// sending entry to dateChan for further handling
			cronChan <- &RuleEntry{
				RuleID: ruleID,
				Value:  expr,
			}

			wg.Done()
		}(cron)
	}

	wg.Wait()
	// If no error occured and context is still valid cancel it
	if ctx.Err() == nil {
		cancel()
	}

	if _, err := dynamoClient.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_EVENTS_TABLE")),
		Item: map[string]*dynamodb.AttributeValue{
			"EventID":  {S: aws.String(uuid.NewString())},
			"UserID":   {S: &userID},
			"Title":    {S: &reqBody.Message},
			"Crons":    {M: cronMap},
			"Dates":    {M: dateMap},
			"Timezone": {S: &reqBody.Timezone},
		},
	}); err != nil {
		log.Println(err.Error())
		return Internal()
	}

	responseBody := map[string]string{
		"response": "You hit this route",
	}

	responseJSON, err := json.Marshal(responseBody)
	if err != nil {
		return Internal()
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
	awsSession := session.Must(session.NewSession(aws.NewConfig()))

	dynamoClient = dynamodb.New(awsSession)
	schedulerClient = scheduler.New(awsSession)

	lambda.Start(Handler)
}
