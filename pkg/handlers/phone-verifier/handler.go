package phoneverifier

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/Slimo300/Reminder-Serverless-Go/pkg/features/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type SnsApiClient interface {
	Subscribe(context.Context, *sns.SubscribeInput) (*sns.SubscribeOutput, error)
	Unsubscribe(context.Context, *sns.UnsubscribeInput) (*sns.UnsubscribeOutput, error)
}
type DynamoApiClient interface {
	GetItem(context.Context, *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
}
type CognitoApiClient interface {
	AdminUpdateUserAttributes(context.Context, *cognito.AdminUpdateUserAttributesInput) (*cognito.AdminUpdateUserAttributesOutput, error)
}

type Handler struct {
	SnsClient     SnsApiClient
	DynamoClient  DynamoApiClient
	CognitoClient CognitoApiClient
}

func (h *Handler) Handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{})
	if !ok {
		return errors.Unauthorized("authorization data not found")
	}
	userID, ok := claims["sub"].(string)
	if !ok {
		return errors.Unauthorized("authorization data not found")
	}
	userName, ok := claims["cognito:username"].(string)
	if !ok {
		return errors.Unauthorized("authorization data not found")
	}

	var reqBody struct {
		VerificationCode string `json:"verification_code"`
	}
	if err := json.Unmarshal([]byte(request.Body), &reqBody); err != nil {
		return errors.BadRequest("invalid request body")
	}

	item, err := h.DynamoClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Key: map[string]dynamotypes.AttributeValue{
			"UserID": &dynamotypes.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return errors.Internal(err)
	}

	newPhoneNumber := item.Item["PhoneNumber"].(*dynamotypes.AttributeValueMemberS).Value
	verificationCode := item.Item["VerificationCode"].(*dynamotypes.AttributeValueMemberS).Value
	subscriptionArn := item.Item["SubscriptionArn"].(*dynamotypes.AttributeValueMemberS).Value

	if verificationCode != reqBody.VerificationCode {
		return errors.Unauthorized("verification code is incorrect")
	}

	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())

	defer close(errChan)
	defer cancel()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		if _, err := h.DynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
			Key: map[string]dynamotypes.AttributeValue{
				"UserID": &dynamotypes.AttributeValueMemberS{Value: userID},
			},
		}); err != nil {
			select {
			case errChan <- err:
				cancel()
			default:
			}
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if _, err := h.SnsClient.Unsubscribe(ctx, &sns.UnsubscribeInput{
			SubscriptionArn: &subscriptionArn,
		}); err != nil {
			select {
			case errChan <- err:
				cancel()
			default:
			}
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		filterPolicy, err := json.Marshal(map[string]interface{}{
			"userID": []string{userID},
		})
		if err != nil {
			select {
			case errChan <- err:
				cancel()
			default:
				return
			}
		}

		subResponse, err := h.SnsClient.Subscribe(ctx, &sns.SubscribeInput{
			TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
			Protocol: aws.String("sms"),
			Endpoint: aws.String(newPhoneNumber),
			Attributes: map[string]string{
				"FilterPolicy": string(filterPolicy),
			},
			ReturnSubscriptionArn: true,
		})
		if err != nil {
			select {
			case errChan <- err:
				cancel()
			default:
			}
			return
		}

		if _, err := h.CognitoClient.AdminUpdateUserAttributes(ctx, &cognito.AdminUpdateUserAttributesInput{
			UserPoolId: aws.String(os.Getenv("USER_POOL_ID")),
			Username:   &userName,
			UserAttributes: []cognitotypes.AttributeType{
				{
					Name:  aws.String("phone_number"),
					Value: &newPhoneNumber,
				},
				{
					Name:  aws.String("phone_number_verified"),
					Value: aws.String("true"),
				},
				{
					Name:  aws.String("custom:subscription_arn"),
					Value: subResponse.SubscriptionArn,
				},
			},
		}); err != nil {
			select {
			case errChan <- err:
				cancel()
			default:
			}
			return
		}
	}()

	wg.Wait()
	if ctx.Err() != nil {
		return errors.Internal(<-errChan)
	}

	responseJSON, err := json.Marshal(map[string]string{
		"phone_number": newPhoneNumber,
	})
	if err != nil {
		return errors.Internal(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
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
