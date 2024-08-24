package main

import (
	"context"

	alarmcreator "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/alarm-creator"
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

	handler := alarmcreator.Handler{
		DynamoClient:    dynamodb.NewFromConfig(cfg),
		SchedulerClient: scheduler.NewFromConfig(cfg),
	}

	lambda.Start(handler.Handle)
}
