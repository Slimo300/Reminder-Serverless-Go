package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/google/uuid"
)

var dynamoClient *dynamodb.Client
var schedulerClient *scheduler.Client

type ruleEntry struct {
	RuleID string
	Value  string
}

type scheduleType int

const (
	AT scheduleType = iota
	CRON
)

func (t scheduleType) string() string {
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
	ScheduleType       scheduleType
}

func createSchedule(ctx context.Context, input createScheduleInput) error {
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

	if _, err := schedulerClient.CreateSchedule(ctx, &scheduler.CreateScheduleInput{
		ActionAfterCompletion:      schedulertypes.ActionAfterCompletionDelete,
		Description:                &input.Message,
		Name:                       &ruleID,
		ScheduleExpression:         aws.String(fmt.Sprintf("%s(%s)", input.ScheduleType.string(), input.ScheduleExpression)),
		ScheduleExpressionTimezone: &input.Timezone,
		Target: &schedulertypes.Target{
			Arn:     aws.String(os.Getenv("LAMBDA_FUNCTION_ARN")),
			RoleArn: aws.String(os.Getenv("ROLE_ARN")),
			Input:   aws.String(string(lambdaInput)),
		},
		FlexibleTimeWindow: &schedulertypes.FlexibleTimeWindow{
			Mode: schedulertypes.FlexibleTimeWindowModeOff,
		},
	}); err != nil {
		return err
	}

	return nil
}

func simplifyDynamoDBItem(item map[string]dynamotypes.AttributeValue) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range item {
		switch v := value.(type) {
		case *dynamotypes.AttributeValueMemberS:
			result[key] = v.Value
		case *dynamotypes.AttributeValueMemberN:
			result[key] = v.Value
		case *dynamotypes.AttributeValueMemberBOOL:
			result[key] = v.Value
		case *dynamotypes.AttributeValueMemberM:
			subMap := make(map[string]interface{})
			for subKey, subValue := range v.Value {
				subMap[subKey] = simplifyDynamoDBItem(map[string]dynamotypes.AttributeValue{subKey: subValue})[subKey]
			}
			result[key] = subMap
		case *dynamotypes.AttributeValueMemberL:
			var list []interface{}
			for _, subValue := range v.Value {
				list = append(list, simplifyDynamoDBItem(map[string]dynamotypes.AttributeValue{"": subValue})[""])
			}
			result[key] = list
		}
	}
	return result
}

func errorResponse(message string, code int) (events.APIGatewayProxyResponse, error) {
	responseJSON, _ := json.Marshal(map[string]string{
		"message": message,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
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
func badRequest(message string) (events.APIGatewayProxyResponse, error) {
	return errorResponse(message, http.StatusBadRequest)
}

// It returns bad request response with given message
func forbidden(message string) (events.APIGatewayProxyResponse, error) {
	return errorResponse(message, http.StatusForbidden)
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

	var reqBody struct {
		Message  string   `json:"message"`
		Dates    []string `json:"dates"`
		Crons    []string `json:"crons"`
		Timezone string   `json:"timezone"`
	}

	if err := json.Unmarshal([]byte(request.Body), &reqBody); err != nil {
		return badRequest("invalid request body")
	}

	dateMap := make(map[string]dynamotypes.AttributeValue)
	cronMap := make(map[string]dynamotypes.AttributeValue)

	dateChan := make(chan *ruleEntry)
	cronChan := make(chan *ruleEntry)
	errChan := make(chan error)
	defer close(dateChan)
	defer close(cronChan)
	defer close(errChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case entry := <-dateChan:
				dateMap[entry.RuleID] = &dynamotypes.AttributeValueMemberS{Value: entry.Value}
			case entry := <-cronChan:
				cronMap[entry.RuleID] = &dynamotypes.AttributeValueMemberS{Value: entry.Value}
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup

	for _, date := range reqBody.Dates {

		wg.Add(1)
		go func(expr string) {
			defer wg.Done()

			ruleID := uuid.NewString()

			if err := createSchedule(ctx, createScheduleInput{
				RuleID:             ruleID,
				UserID:             userID,
				ScheduleExpression: expr,
				ScheduleType:       AT,
				Message:            reqBody.Message,
				Timezone:           reqBody.Timezone,
			}); err != nil {
				errChan <- err
				cancel()
			}

			// sending entry to dateChan for further handling
			dateChan <- &ruleEntry{
				RuleID: ruleID,
				Value:  expr,
			}
		}(date)
	}

	for _, cron := range reqBody.Crons {
		wg.Add(1)
		go func(expr string) {
			defer wg.Done()

			ruleID := uuid.NewString()

			if err := createSchedule(ctx, createScheduleInput{
				RuleID:             ruleID,
				UserID:             userID,
				ScheduleExpression: expr,
				ScheduleType:       CRON,
				Message:            reqBody.Message,
				Timezone:           reqBody.Timezone,
			}); err != nil {
				errChan <- err
				cancel()
			}

			// sending entry to dateChan for further handling
			cronChan <- &ruleEntry{
				RuleID: ruleID,
				Value:  expr,
			}
		}(cron)
	}

	// We wait for all goroutines to finish and cancel a context to
	// end last goroutine if it wasn't cancelled before
	wg.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		return internal(<-errChan)
	} else {
		cancel()
	}

	item := map[string]dynamotypes.AttributeValue{
		"EventID":  &dynamotypes.AttributeValueMemberS{Value: uuid.NewString()},
		"UserID":   &dynamotypes.AttributeValueMemberS{Value: userID},
		"Title":    &dynamotypes.AttributeValueMemberS{Value: reqBody.Message},
		"Crons":    &dynamotypes.AttributeValueMemberM{Value: cronMap},
		"Dates":    &dynamotypes.AttributeValueMemberM{Value: dateMap},
		"Timezone": &dynamotypes.AttributeValueMemberS{Value: reqBody.Timezone},
	}

	if _, err := dynamoClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Item:      item,
	}); err != nil {
		return internal(err)
	}

	responseJSON, err := json.Marshal(simplifyDynamoDBItem(item))
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
		StatusCode: http.StatusCreated,
	}, nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return
	}

	dynamoClient = dynamodb.NewFromConfig(cfg)
	schedulerClient = scheduler.NewFromConfig(cfg)

	lambda.Start(Handler)
}
