package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscognito"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssns"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"

	golambda "github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"
)

type AlerterStackProps struct {
	awscdk.StackProps
}

func NewAlerterStack(scope constructs.Construct, id string, props *AlerterStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Creating an SNS Topic

	snsTopic := awssns.NewTopic(stack, jsii.String("GO_ReminderSnsTopic"), &awssns.TopicProps{
		EnforceSSL: jsii.Bool(true),
		TopicName:  jsii.String("GO_ReminderSnsTopic"),
	})

	// Creating Cognito User Pool

	userPool := awscognito.NewUserPool(stack, jsii.String("GO_ReminderUserPool"), &awscognito.UserPoolProps{
		UserPoolName: jsii.String("GO_ReminderUserPool"),
		SignInAliases: &awscognito.SignInAliases{
			Username: jsii.Bool(true),
			Phone:    jsii.Bool(true),
		},
		SelfSignUpEnabled: jsii.Bool(true),
		CustomAttributes: &map[string]awscognito.ICustomAttribute{
			"subscription_arn": awscognito.NewStringAttribute(&awscognito.StringAttributeProps{Mutable: jsii.Bool(true)}),
		},
		AccountRecovery: awscognito.AccountRecovery_PHONE_ONLY_WITHOUT_MFA,
		AutoVerify: &awscognito.AutoVerifiedAttrs{
			Phone: jsii.Bool(true),
		},
	})

	_ = awscognito.NewUserPoolClient(stack, jsii.String("GO_ReminderUserPoolClient"), &awscognito.UserPoolClientProps{
		UserPool:           userPool,
		UserPoolClientName: jsii.String("GO_ReminderUserPoolClient"),
	})

	// Creating Post Confirmation Trigger for Cognito User Pool

	postConfirmationLambda := golambda.NewGoFunction(stack, jsii.String("GO_PostConfirmationTrigger"), &golambda.GoFunctionProps{
		FunctionName: jsii.String("GO_PostConfirmationTrigger"),
		Entry:        jsii.String("lambdas/post-confirmation-trigger"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Architecture: awslambda.Architecture_ARM_64(),
		Environment: &map[string]*string{
			"SNS_TOPIC_ARN": snsTopic.TopicArn(),
		},
	})
	postConfirmationLambda.Role().AttachInlinePolicy(awsiam.NewPolicy(stack, jsii.String("GO_PostConfirmationTriggerRole"), &awsiam.PolicyProps{
		Statements: &[]awsiam.PolicyStatement{
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Actions:   jsii.Strings("cognito-idp:AdminUpdateUserAttributes"),
				Resources: jsii.Strings(*userPool.UserPoolArn()),
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Actions:   jsii.Strings("sns:Subscribe"),
				Resources: jsii.Strings(*snsTopic.TopicArn()),
			}),
		},
	}))

	userPool.AddTrigger(awscognito.UserPoolOperation_POST_CONFIRMATION(), postConfirmationLambda, awscognito.LambdaVersion_V1_0)

	// Creating DynamoDB Table

	alertsTable := awsdynamodb.NewTable(stack, jsii.String("GO_AlarmTable"), &awsdynamodb.TableProps{
		TableName: jsii.String("GO_AlarmTable"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("eventID"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	// Creating Lambda functions and adding permissions to them

	alarmCreatorLambda := golambda.NewGoFunction(stack, jsii.String("GO_AlarmCreator"), &golambda.GoFunctionProps{
		FunctionName: jsii.String("GO_AlarmCreator"),
		Entry:        jsii.String("lambdas/alarm-creator"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Architecture: awslambda.Architecture_ARM_64(),
	})
	alarmCreatorLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("dynamodb:PutItem"),
		Resources: jsii.Strings(*alertsTable.TableArn()),
	}))
	alarmCreatorLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("scheduler:CreateSchedule"),
		Resources: jsii.Strings("*"),
	}))

	myGateway := awsapigateway.NewRestApi(stack, jsii.String("GO_RestApi"), &awsapigateway.RestApiProps{
		DefaultCorsPreflightOptions: &awsapigateway.CorsOptions{
			AllowOrigins: &[]*string{jsii.String("*")},
			AllowMethods: &[]*string{jsii.String("OPTIONS"), jsii.String("GET"), jsii.String("POST"), jsii.String("DELETE")},
		},
		RestApiName: jsii.String("GO_RestApi"),
	})

	integration := awsapigateway.NewLambdaIntegration(alarmCreatorLambda, nil)

	alarmsResource := myGateway.Root().AddResource(jsii.String("alarms"), nil)
	alarmsResource.AddMethod(jsii.String("POST"), integration, &awsapigateway.MethodOptions{
		AuthorizationType: awsapigateway.AuthorizationType_COGNITO,
		Authorizer: awsapigateway.NewCognitoUserPoolsAuthorizer(stack, jsii.String("GO_Authorizer"), &awsapigateway.CognitoUserPoolsAuthorizerProps{
			CognitoUserPools: &[]awscognito.IUserPool{userPool},
		}),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewAlerterStack(app, "ReminderStack", &AlerterStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
