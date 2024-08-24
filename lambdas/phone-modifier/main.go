package main

import (
	"context"

	phonemodifier "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/phone-modifier"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return
	}

	handler := phonemodifier.Handler{
		SnsClient:    sns.NewFromConfig(cfg),
		DynamoClient: dynamodb.NewFromConfig(cfg),
	}

	lambda.Start(handler.Handle)
}
