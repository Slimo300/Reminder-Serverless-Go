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
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}
type SchedulerApiClient interface {
	CreateSchedule(context.Context, *scheduler.CreateScheduleInput, ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error)
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

type RequestBody struct {
	Message  string   `json:"message"`
	Timezone string   `json:"timezone"`
	Dates    []string `json:"dates"`
	Crons    []string `json:"crons"`
}

func (b *RequestBody) Validate() error {
	if len(b.Crons) == 0 && len(b.Dates) == 0 {
		return errors.New("there are no crons or dates specified")
	}
	if b.Message == "" {
		return errors.New(`"message" cannot be an empty string`)
	}
	if b.Timezone == "" {
		return errors.New(`"timezone" cannot be an empty string`)
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

	var reqBody RequestBody
	if err := json.Unmarshal([]byte(request.Body), &reqBody); err != nil {
		return pkgerrors.BadRequest("invalid request body")
	}
	if err := reqBody.Validate(); err != nil {
		return pkgerrors.BadRequest(err.Error())
	}

	cronMap := make(map[string]dynamotypes.AttributeValue)
	dateMap := make(map[string]dynamotypes.AttributeValue)

	cronMutex := &sync.Mutex{}
	dateMutex := &sync.Mutex{}

	errChan := make(chan error, 1)
	defer close(errChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
				select {
				case errChan <- err:
					cancel()
				default:
				}
				return
			}
			dateMutex.Lock()
			dateMap[ruleID] = &dynamotypes.AttributeValueMemberS{Value: expr}
			dateMutex.Unlock()

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
				select {
				case errChan <- err:
					cancel()
				default:
				}
				return
			}

			// sending entry to dateChan for further handling
			cronMutex.Lock()
			cronMap[ruleID] = &dynamotypes.AttributeValueMemberS{Value: expr}
			cronMutex.Unlock()
		}(cron)
	}

	// We wait for all goroutines to finish and cancel a context to
	// end last goroutine if it wasn't cancelled before
	wg.Wait()

	select {
	case err := <-errChan:
		return pkgerrors.Internal(err)
	default:
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
