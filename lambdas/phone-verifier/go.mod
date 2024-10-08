module github.com/Slimo300/Reminder-Serverless-Go/lambdas/phone-verifier

go 1.22.0

require (
	github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/phone-verifier v0.0.0-20240824160752-4e7921ee5bb6
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go-v2/config v1.27.28
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.43.2
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.34.5
	github.com/aws/aws-sdk-go-v2/service/sns v1.31.4
)

require (
	github.com/Slimo300/Reminder-Serverless-Go/pkg/features/errors v0.0.0-20240821145950-d2da7dbd1a33 // indirect
	github.com/aws/aws-sdk-go-v2 v1.30.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.28 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.9.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.4 // indirect
	github.com/aws/smithy-go v1.20.4 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)
