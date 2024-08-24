package main

import (
	"context"

	alarmexecutor "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/alarm-executor"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func main() {

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return
	}

	handler := alarmexecutor.Handler{
		SNSClient: sns.NewFromConfig(cfg),
	}

	lambda.Start(handler.Handle)
}
