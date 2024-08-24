package postconfirmationtrigger

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type SnsApiClient interface {
	Subscribe(context.Context, *sns.SubscribeInput, ...func(*sns.Options)) (*sns.SubscribeOutput, error)
}
type CognitoApiClient interface {
	AdminUpdateUserAttributes(context.Context, *cognito.AdminUpdateUserAttributesInput, ...func(*cognito.Options)) (*cognito.AdminUpdateUserAttributesOutput, error)
}

type Handler struct {
	CognitoClient CognitoApiClient
	SnsClient     SnsApiClient
}

func (h *Handler) Handle(event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {

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

	subResponse, err := h.SnsClient.Subscribe(context.Background(), &sns.SubscribeInput{
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

	if _, err := h.CognitoClient.AdminUpdateUserAttributes(context.Background(), &cognito.AdminUpdateUserAttributesInput{
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
