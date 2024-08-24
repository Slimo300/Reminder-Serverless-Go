package main

import (
	"context"

	alarmgetter "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/alarm-getter"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return
	}

	handler := alarmgetter.AlarmGetterHandler{
		DynamoClient: dynamodb.NewFromConfig(cfg),
	}

	lambda.Start(handler.Handle)
}
