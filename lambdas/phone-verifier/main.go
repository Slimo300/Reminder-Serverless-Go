package main

import (
	"context"

	phoneverifier "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/phone-verifier"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return
	}

	handler := phoneverifier.Handler{
		CognitoClient: cognitoidentityprovider.NewFromConfig(cfg),
		DynamoClient:  dynamodb.NewFromConfig(cfg),
		SnsClient:     sns.NewFromConfig(cfg),
	}

	lambda.Start(handler.Handle)
}
