package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

var (
	snsClient     *sns.Client
	cognitoClient *cognitoidentityprovider.Client
)

func Handler(event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {

	phoneNumber := event.Request.UserAttributes["phone_number"]
	sub := event.Request.UserAttributes["sub"]
	if phoneNumber == "" || sub == "" {
		return event, errors.New("invalid user attributes")
	}

	filterPolicy, err := json.Marshal(map[string]interface{}{
		"userID": []string{sub},
	})
	if err != nil {
		log.Println(err.Error())
		return event, err
	}

	subResponse, err := snsClient.Subscribe(context.Background(), &sns.SubscribeInput{
		TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
		Protocol: aws.String("sms"),
		Endpoint: aws.String(phoneNumber),
		Attributes: map[string]string{
			"FilterPolicy": string(filterPolicy),
		},
		ReturnSubscriptionArn: true,
	})
	if err != nil {
		log.Println(err.Error())
		return event, err
	}

	if _, err := cognitoClient.AdminUpdateUserAttributes(context.Background(), &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(event.UserPoolID),
		Username:   &event.UserName,
		UserAttributes: []cognitotypes.AttributeType{{
			Name:  aws.String("custom:subscription_arn"),
			Value: subResponse.SubscriptionArn,
		}},
	}); err != nil {
		log.Println(err.Error())
		return event, err
	}

	return event, nil
}

func main() {

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Println(err)
		return
	}

	snsClient = sns.NewFromConfig(cfg)
	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)

	lambda.Start(Handler)
}
