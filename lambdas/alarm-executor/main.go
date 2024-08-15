package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

var snsClient *sns.Client

type alarmEvent struct {
	UserID  string `json:"userID"`
	Message string `json:"message"`
}

func Handler(event alarmEvent) error {

	if _, err := snsClient.Publish(context.Background(), &sns.PublishInput{
		TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
		Message:  &event.Message,
		MessageAttributes: map[string]types.MessageAttributeValue{
			"userID": {
				DataType:    aws.String("String"),
				StringValue: &event.UserID,
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func main() {

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Println(err)
		return
	}

	snsClient = sns.NewFromConfig(cfg)

	lambda.Start(Handler)
}
