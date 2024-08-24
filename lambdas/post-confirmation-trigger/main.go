package main

import (
	"context"
	"log"

	postconfirmationtrigger "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/post-confirmation-trigger"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func main() {

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Println(err)
		return
	}

	handler := postconfirmationtrigger.Handler{
		SnsClient:     sns.NewFromConfig(cfg),
		CognitoClient: cognitoidentityprovider.NewFromConfig(cfg),
	}

	lambda.Start(handler.Handle)
}
