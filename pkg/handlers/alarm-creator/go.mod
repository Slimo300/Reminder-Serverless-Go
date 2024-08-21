module github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/alarm-creator

go 1.22.0

require (
	github.com/Slimo300/Reminder-Serverless-Go/pkg/features/dynamomapper v0.0.0-20240821145950-d2da7dbd1a33
	github.com/Slimo300/Reminder-Serverless-Go/pkg/features/errors v0.0.0-20240821145950-d2da7dbd1a33
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go-v2 v1.30.4
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.34.5
	github.com/aws/aws-sdk-go-v2/service/scheduler v1.10.4
	github.com/google/uuid v1.6.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.9.17 // indirect
	github.com/aws/smithy-go v1.20.4 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)
