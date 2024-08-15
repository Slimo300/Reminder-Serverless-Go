package main

import (
	"context"
	"encoding/json"
	"errors"
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

var (
	dynamoClient    *dynamodb.Client
	schedulerClient *scheduler.Client
)

func errorResponse(message string, code int) (events.APIGatewayProxyResponse, error) {
	responseJSON, _ := json.Marshal(map[string]string{
		"message": message,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: code,
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

	eventID := request.PathParameters["id"]
	if eventID == "" {
		return badRequest("no eventID specified")
	}

	res, err := dynamoClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]dynamotypes.AttributeValue{
			"UserID":  &dynamotypes.AttributeValueMemberS{Value: userID},
			"EventID": &dynamotypes.AttributeValueMemberS{Value: eventID},
		},
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
	})
	if err != nil {
		return internal(err)
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
			if _, err := schedulerClient.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
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
			if _, err := schedulerClient.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
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
		return internal(<-errChan)
	}

	if _, err := dynamoClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Key: map[string]dynamotypes.AttributeValue{
			"UserID":  &dynamotypes.AttributeValueMemberS{Value: userID},
			"EventID": &dynamotypes.AttributeValueMemberS{Value: eventID},
		},
	}); err != nil {
		return internal(err)
	}

	responseJSON, err := json.Marshal(map[string]string{
		"message": "ok",
	})
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
		StatusCode: 200,
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
