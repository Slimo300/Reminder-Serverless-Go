package phonemodifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Slimo300/Reminder-Serverless-Go/pkg/features/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/thanhpk/randstr"
)

type SnsApiClient interface {
	Publish(context.Context, *sns.PublishInput) (*sns.PublishOutput, error)
}
type DynamoApiClient interface {
	PutItem(context.Context, *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

type Handler struct {
	SnsClient    SnsApiClient
	DynamoClient DynamoApiClient
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
	phone_number, ok := claims["phone_number"].(string)
	if !ok {
		return errors.Unauthorized("authorization data not found")
	}
	subscriptionArn, ok := claims["custom:subscription_arn"].(string)
	if !ok {
		return errors.Unauthorized("authorization data not found")
	}

	var reqBody struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.Unmarshal([]byte(request.Body), &reqBody); err != nil {
		return errors.BadRequest("invalid request body")
	}

	verificationCode := randstr.Dec(6)
	expirationTimestamp := time.Now().Add(24 * time.Hour).Unix()

	if _, err := h.DynamoClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Item: map[string]dynamotypes.AttributeValue{
			"UserID":           &dynamotypes.AttributeValueMemberS{Value: userID},
			"PhoneNumber":      &dynamotypes.AttributeValueMemberS{Value: reqBody.PhoneNumber},
			"VerificationCode": &dynamotypes.AttributeValueMemberS{Value: verificationCode},
			"SubscriptionArn":  &dynamotypes.AttributeValueMemberS{Value: subscriptionArn},
			"ExpireOn":         &dynamotypes.AttributeValueMemberN{Value: fmt.Sprint(expirationTimestamp)},
		},
	}); err != nil {
		return errors.Internal(err)
	}

	if _, err := h.SnsClient.Publish(context.Background(), &sns.PublishInput{
		PhoneNumber: &phone_number,
		Message:     aws.String(fmt.Sprintf("Your verification code: %s", verificationCode)),
	}); err != nil {
		return errors.Internal(err)
	}

	responseJSON, err := json.Marshal(map[string]string{
		"message": "verification code sent",
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
