package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

var (
	dynamoClient  *dynamodb.Client
	cognitoClient *cognitoidentityprovider.Client
	snsClient     *sns.Client
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
func internal(err error) (events.APIGatewayProxyResponse, error) {
	log.Println(err)
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
	userName, ok := claims["cognito:username"].(string)
	if !ok {
		return forbidden("authorization data not found")
	}

	var reqBody struct {
		VerificationCode string `json:"verification_code"`
	}
	if err := json.Unmarshal([]byte(request.Body), &reqBody); err != nil {
		return badRequest("invalid request body")
	}

	item, err := dynamoClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Key: map[string]dynamotypes.AttributeValue{
			"UserID": &dynamotypes.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return internal(err)
	}

	newPhoneNumber := item.Item["PhoneNumber"].(*dynamotypes.AttributeValueMemberS).Value
	verificationCode := item.Item["VerificationCode"].(*dynamotypes.AttributeValueMemberS).Value
	subscriptionArn := item.Item["SubscriptionArn"].(*dynamotypes.AttributeValueMemberS).Value

	if verificationCode != reqBody.VerificationCode {
		return forbidden("verification code is incorrect")
	}

	if _, err := dynamoClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(os.Getenv("DYNAMO_TABLE_NAME")),
		Key: map[string]dynamotypes.AttributeValue{
			"UserID": &dynamotypes.AttributeValueMemberS{Value: userID},
		},
	}); err != nil {
		return internal(err)
	}

	if _, err := snsClient.Unsubscribe(context.Background(), &sns.UnsubscribeInput{
		SubscriptionArn: &subscriptionArn,
	}); err != nil {
		return internal(err)
	}

	filterPolicy, err := json.Marshal(map[string]interface{}{
		"userID": []string{userID},
	})
	if err != nil {
		return internal(err)
	}

	subResponse, err := snsClient.Subscribe(context.Background(), &sns.SubscribeInput{
		TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
		Protocol: aws.String("sms"),
		Endpoint: aws.String(newPhoneNumber),
		Attributes: map[string]string{
			"FilterPolicy": string(filterPolicy),
		},
		ReturnSubscriptionArn: true,
	})
	if err != nil {
		return internal(err)
	}

	if _, err := cognitoClient.AdminUpdateUserAttributes(context.Background(), &cognitoidentityprovider.AdminUpdateUserAttributesInput{
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
		return internal(err)
	}

	responseJSON, err := json.Marshal(map[string]string{
		"phone_number": newPhoneNumber,
	})
	if err != nil {
		return internal(err)
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

	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)
	dynamoClient = dynamodb.NewFromConfig(cfg)
	snsClient = sns.NewFromConfig(cfg)

	lambda.Start(Handler)
}
