package alarmcreator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/google/uuid"

	"github.com/Slimo300/Reminder-Serverless-Go/pkg/features/dynamomapper"
	pkgerrors "github.com/Slimo300/Reminder-Serverless-Go/pkg/features/errors"
)

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

type DynamoApiClient interface {
	PutItem(context.Context, *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}
type SchedulerApiClient interface {
	CreateSchedule(context.Context, *scheduler.CreateScheduleInput) (*scheduler.CreateScheduleOutput, error)
}

type Handler struct {
	DynamoClient    DynamoApiClient
	SchedulerClient SchedulerApiClient
}

type createScheduleInput struct {
	RuleID             string
	ScheduleExpression string
	Timezone           string
	Message            string
	UserID             string
	ScheduleType       scheduleType
}

func (h *Handler) createSchedule(ctx context.Context, input createScheduleInput) error {
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

	if _, err := h.SchedulerClient.CreateSchedule(ctx, &scheduler.CreateScheduleInput{
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

func (h *Handler) Handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{})
	if !ok {
		return pkgerrors.Unauthorized("authorization data not found")
	}
	userID, ok := claims["sub"].(string)
	if !ok {
		return pkgerrors.Unauthorized("authorization data not found")
	}

	var reqBody struct {
		Message  string   `json:"message"`
		Dates    []string `json:"dates"`
		Crons    []string `json:"crons"`
		Timezone string   `json:"timezone"`
	}

	if err := json.Unmarshal([]byte(request.Body), &reqBody); err != nil {
		return pkgerrors.BadRequest("invalid request body")
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

			if err := h.createSchedule(ctx, createScheduleInput{
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

			if err := h.createSchedule(ctx, createScheduleInput{
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
		return pkgerrors.Internal(<-errChan)
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

	if _, err := h.DynamoClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Item:      item,
	}); err != nil {
		return pkgerrors.Internal(err)
	}

	responseJSON, err := json.Marshal(dynamomapper.SimplifyDynamoDBItem(item))
	if err != nil {
		return pkgerrors.Internal(err)
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
