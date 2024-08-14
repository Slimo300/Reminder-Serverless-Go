package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/sns"
)

var (
	snsClient     *sns.SNS
	cognitoClient *cognitoidentityprovider.CognitoIdentityProvider
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

	subResponse, err := snsClient.Subscribe(&sns.SubscribeInput{
		TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
		Protocol: aws.String("sms"),
		Endpoint: aws.String(phoneNumber),
		Attributes: map[string]*string{
			"FilterPolicy": aws.String(string(filterPolicy)),
		},
		ReturnSubscriptionArn: aws.Bool(true),
	})
	if err != nil {
		log.Println(err.Error())
		return event, err
	}

	if _, err := cognitoClient.AdminUpdateUserAttributes(&cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(event.UserPoolID),
		Username:   &event.UserName,
		UserAttributes: []*cognitoidentityprovider.AttributeType{{
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

	awsSession := session.Must(session.NewSession(aws.NewConfig()))

	snsClient = sns.New(awsSession)
	cognitoClient = cognitoidentityprovider.New(awsSession)

	lambda.Start(Handler)
}
