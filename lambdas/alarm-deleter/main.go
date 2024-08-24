package main

import (
	"context"

	alarmdeleter "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/alarm-deleter"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return
	}

	handler := alarmdeleter.Handler{
		DynamoClient:    dynamodb.NewFromConfig(cfg),
		SchedulerClient: scheduler.NewFromConfig(cfg),
	}

	lambda.Start(handler.Handle)
}
