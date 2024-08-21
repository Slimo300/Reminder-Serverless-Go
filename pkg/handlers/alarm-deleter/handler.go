package alarmdeleter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/google/uuid"

	pkgerrors "github.com/Slimo300/Reminder-Serverless-Go/pkg/features/errors"
)

type DynamoApiClient interface {
	GetItem(context.Context, *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput) (*dynamodb.DeleteBackupOutput, error)
}

type SchedulerApiClient interface {
	DeleteSchedule(context.Context, *scheduler.DeleteScheduleInput) (*scheduler.DeleteScheduleOutput, error)
}

type Handler struct {
	DynamoClient    DynamoApiClient
	SchedulerClient SchedulerApiClient
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

	eventID := request.PathParameters["id"]
	if eventID == "" {
		return pkgerrors.BadRequest("no eventID specified")
	}

	res, err := h.DynamoClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]dynamotypes.AttributeValue{
			"UserID":  &dynamotypes.AttributeValueMemberS{Value: userID},
			"EventID": &dynamotypes.AttributeValueMemberS{Value: eventID},
		},
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
	})
	if err != nil {
		return pkgerrors.Internal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	defer close(errChan)
	defer cancel()
	var wg sync.WaitGroup

	for key := range res.Item["Dates"].(*dynamotypes.AttributeValueMemberM).Value {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}
			var errNotFound *schedulertypes.ResourceNotFoundException
			if _, err := h.SchedulerClient.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
				Name:        &key,
				ClientToken: aws.String(uuid.NewString()),
			}); err != nil && !errors.As(err, &errNotFound) {
				// Here we try to send error to errChan but if it is not answering we return as it means
				// that other goroutine already published an error
				select {
				case errChan <- err:
					cancel()
				default:
					return
				}
			}
		}(key)
	}
	for key := range res.Item["Crons"].(*dynamotypes.AttributeValueMemberM).Value {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}
			var errNotFound *schedulertypes.ResourceNotFoundException
			if _, err := h.SchedulerClient.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
				Name:        &key,
				ClientToken: aws.String(uuid.NewString()),
			}); err != nil && !errors.As(err, &errNotFound) {
				// Here we try to send error to errChan but if it is not answering we return as it means
				// that other goroutine already published an error
				select {
				case errChan <- err:
					cancel()
				default:
					return
				}
			}
		}(key)
	}

	wg.Wait()
	if errors.Is(ctx.Err(), context.Canceled) {
		return pkgerrors.Internal(<-errChan)
	}

	if _, err := h.DynamoClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Key: map[string]dynamotypes.AttributeValue{
			"UserID":  &dynamotypes.AttributeValueMemberS{Value: userID},
			"EventID": &dynamotypes.AttributeValueMemberS{Value: eventID},
		},
	}); err != nil {
		return pkgerrors.Internal(err)
	}

	responseJSON, err := json.Marshal(map[string]string{
		"message": "ok",
	})
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
		StatusCode: 200,
	}, nil
}
