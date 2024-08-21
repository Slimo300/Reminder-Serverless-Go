package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/thanhpk/randstr"
)

var (
	dynamoClient *dynamodb.Client
	snsClient    *sns.Client
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

func forbidden(message string) (events.APIGatewayProxyResponse, error) {
	return errorResponse(message, http.StatusForbidden)
}
func internal() (events.APIGatewayProxyResponse, error) {
	return errorResponse("internal server error", http.StatusInternalServerError)
}
func badRequest(message string) (events.APIGatewayProxyResponse, error) {
	return errorResponse(message, http.StatusBadRequest)
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
	phone_number, ok := claims["phone_number"].(string)
	if !ok {
		return forbidden("authorization data not found")
	}
	subscriptionArn, ok := claims["custom:subscription_arn"].(string)
	if !ok {
		return forbidden("authorization data not found")
	}

	var reqBody struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.Unmarshal([]byte(request.Body), &reqBody); err != nil {
		return badRequest("invalid request body")
	}

	verificationCode := randstr.Dec(6)
	expirationTimestamp := time.Now().Add(24 * time.Hour).Unix()

	if _, err := dynamoClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Item: map[string]dynamotypes.AttributeValue{
			"UserID":           &dynamotypes.AttributeValueMemberS{Value: userID},
			"PhoneNumber":      &dynamotypes.AttributeValueMemberS{Value: reqBody.PhoneNumber},
			"VerificationCode": &dynamotypes.AttributeValueMemberS{Value: verificationCode},
			"SubscriptionArn":  &dynamotypes.AttributeValueMemberS{Value: subscriptionArn},
			"ExpireOn":         &dynamotypes.AttributeValueMemberN{Value: fmt.Sprint(expirationTimestamp)},
		},
	}); err != nil {
		return internal()
	}

	if _, err := snsClient.Publish(context.Background(), &sns.PublishInput{
		PhoneNumber: &phone_number,
		Message:     aws.String(fmt.Sprintf("Your verification code: %s", verificationCode)),
	}); err != nil {
		return internal()
	}

	responseJSON, err := json.Marshal(map[string]string{
		"message": "verification code sent",
	})
	if err != nil {
		return internal()
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

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Println(err)
		return
	}

	snsClient = sns.NewFromConfig(cfg)
	dynamoClient = dynamodb.NewFromConfig(cfg)

	lambda.Start(Handler)
}
