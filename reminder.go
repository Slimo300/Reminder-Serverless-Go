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

	// Lambda bundling options
	bundlingOptions := &golambda.BundlingOptions{
		GoBuildFlags: jsii.Strings(`-ldflags "-s -w"`),
		Environment: &map[string]*string{
			"CGO_ENABLED": jsii.String("0"),
		},
	}

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
		Bundling: bundlingOptions,
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

	// Creating DynamoDB Alarms Table

	alarmsTable := awsdynamodb.NewTable(stack, jsii.String("GO_AlarmTable"), &awsdynamodb.TableProps{
		TableName: jsii.String("GO_AlarmTable"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("UserID"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("EventID"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	// Creating DynamoDB Verification Codes Table

	codesTable := awsdynamodb.NewTable(stack, jsii.String("GO_CodesTable"), &awsdynamodb.TableProps{
		TableName: jsii.String("GO_CodesTable"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("UserID"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TimeToLiveAttribute: jsii.String("ExpireOn"),
		RemovalPolicy:       awscdk.RemovalPolicy_DESTROY,
	})

	// Creating Lambda functions and adding permissions to them

	// Creating Alarm Executor Function
	alarmExecutorLambda := golambda.NewGoFunction(stack, jsii.String("GO_AlarmExecutor"), &golambda.GoFunctionProps{
		FunctionName: jsii.String("GO_AlarmExecutor"),
		Entry:        jsii.String("lambdas/alarm-executor"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Architecture: awslambda.Architecture_ARM_64(),
		Environment: &map[string]*string{
			"SNS_TOPIC_ARN": snsTopic.TopicArn(),
		},
		Bundling: bundlingOptions,
	})
	alarmExecutorLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("sns:Publish"),
		Resources: jsii.Strings(*snsTopic.TopicArn()),
	}))

	lambdaExecutorInvokeRole := awsiam.NewRole(stack, jsii.String("GO_AlarmExecutorInvokeRole"), &awsiam.RoleProps{
		RoleName:  jsii.String("GO_AlarmExecutorInvokeRole"),
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("scheduler.amazonaws.com"), nil),
	})
	lambdaExecutorInvokeRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("lambda:InvokeFunction"),
		Resources: jsii.Strings(*alarmExecutorLambda.FunctionArn()),
	}))

	// Alarm Creator Function
	alarmCreatorLambda := golambda.NewGoFunction(stack, jsii.String("GO_AlarmCreator"), &golambda.GoFunctionProps{
		FunctionName: jsii.String("GO_AlarmCreator"),
		Entry:        jsii.String("lambdas/alarm-creator"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Architecture: awslambda.Architecture_ARM_64(),
		Environment: &map[string]*string{
			"DYNAMO_TABLE_NAME":   alarmsTable.TableArn(),
			"LAMBDA_FUNCTION_ARN": alarmExecutorLambda.FunctionArn(),
			"ROLE_ARN":            lambdaExecutorInvokeRole.RoleArn(),
		},
		Bundling: bundlingOptions,
	})
	alarmCreatorLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("dynamodb:PutItem"),
		Resources: jsii.Strings(*alarmsTable.TableArn()),
	}))
	alarmCreatorLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("scheduler:CreateSchedule"),
		Resources: jsii.Strings("*"),
	}))
	alarmCreatorLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("iam:PassRole"),
		Resources: jsii.Strings(*lambdaExecutorInvokeRole.RoleArn()),
	}))

	// Alarm Getter Function
	alarmGetterLambda := golambda.NewGoFunction(stack, jsii.String("GO_AlarmGetter"), &golambda.GoFunctionProps{
		FunctionName: jsii.String("GO_AlarmGetter"),
		Entry:        jsii.String("lambdas/alarm-getter"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Architecture: awslambda.Architecture_ARM_64(),
		Environment: &map[string]*string{
			"DYNAMO_TABLE_NAME": alarmsTable.TableName(),
		},
		Bundling: bundlingOptions,
	})
	alarmGetterLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("dynamodb:Query"),
		Resources: jsii.Strings(*alarmsTable.TableArn()),
	}))

	// Alarm Deleter Function
	alarmDeleterLambda := golambda.NewGoFunction(stack, jsii.String("GO_AlarmDeleter"), &golambda.GoFunctionProps{
		FunctionName: jsii.String("GO_AlarmDeleter"),
		Entry:        jsii.String("lambdas/alarm-deleter"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Architecture: awslambda.Architecture_ARM_64(),
		Environment: &map[string]*string{
			"DYNAMO_TABLE_NAME": alarmsTable.TableName(),
		},
		Bundling: bundlingOptions,
	})
	alarmDeleterLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("dynamodb:GetItem", "dynamodb:DeleteItem"),
		Resources: jsii.Strings(*alarmsTable.TableArn()),
	}))
	alarmDeleterLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("scheduler:DeleteSchedule"),
		Resources: jsii.Strings("*"),
	}))

	// Phone Number Modifier Function
	phoneModifierLambda := golambda.NewGoFunction(stack, jsii.String("GO_PhoneModifier"), &golambda.GoFunctionProps{
		FunctionName: jsii.String("GO_PhoneModifier"),
		Entry:        jsii.String("lambdas/phone-modifier"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Architecture: awslambda.Architecture_ARM_64(),
		Environment: &map[string]*string{
			"DYNAMO_TABLE_NAME": codesTable.TableArn(),
		},
		Bundling: bundlingOptions,
	})
	phoneModifierLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("dynamodb:PutItem"),
		Resources: jsii.Strings(*codesTable.TableArn()),
	}))
	phoneModifierLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("sns:Publish"),
		Resources: jsii.Strings("*"),
	}))

	// Phone Number Verifier Function
	phoneVerifierLambda := golambda.NewGoFunction(stack, jsii.String("GO_PhoneVerifier"), &golambda.GoFunctionProps{
		FunctionName: jsii.String("GO_PhoneVerifier"),
		Entry:        jsii.String("lambdas/phone-verifier"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		Architecture: awslambda.Architecture_ARM_64(),
		Environment: &map[string]*string{
			"DYNAMO_TABLE_NAME": codesTable.TableArn(),
			"SNS_TOPIC_ARN":     snsTopic.TopicArn(),
			"USER_POOL_ID":      userPool.UserPoolId(),
		},
		Bundling: bundlingOptions,
	})
	phoneVerifierLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("dynamodb:GetItem", "dynamodb:DeleteItem"),
		Resources: jsii.Strings(*codesTable.TableArn()),
	}))
	phoneVerifierLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("sns:Subscribe", "sns:Unsubscribe"),
		Resources: jsii.Strings(*snsTopic.TopicArn()),
	}))
	phoneVerifierLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("cognito-idp:AdminUpdateUserAttributes"),
		Resources: jsii.Strings(*userPool.UserPoolArn()),
	}))

	// Defining Rest API in API Gateway
	myGateway := awsapigateway.NewRestApi(stack, jsii.String("GO_RestApi"), &awsapigateway.RestApiProps{
		DefaultCorsPreflightOptions: &awsapigateway.CorsOptions{
			AllowOrigins: &[]*string{jsii.String("*")},
			AllowMethods: &[]*string{jsii.String("OPTIONS"), jsii.String("GET"), jsii.String("POST"), jsii.String("DELETE")},
		},
		RestApiName: jsii.String("GO_RestApi"),
	})

	cognitoAuthorizer := awsapigateway.NewCognitoUserPoolsAuthorizer(stack, jsii.String("GO_Authorizer"), &awsapigateway.CognitoUserPoolsAuthorizerProps{
		CognitoUserPools: &[]awscognito.IUserPool{userPool},
	})

	alarmCreatorIntegration := awsapigateway.NewLambdaIntegration(alarmCreatorLambda, nil)
	alarmGetterIntegration := awsapigateway.NewLambdaIntegration(alarmGetterLambda, nil)
	alarmDeleterIntegration := awsapigateway.NewLambdaIntegration(alarmDeleterLambda, nil)
	phoneModifierIntegration := awsapigateway.NewLambdaIntegration(phoneModifierLambda, nil)
	phoneVerifierIntegration := awsapigateway.NewLambdaIntegration(phoneVerifierLambda, nil)

	alarmsResource := myGateway.Root().AddResource(jsii.String("alarms"), nil)
	alarmsResource.AddMethod(jsii.String("POST"), alarmCreatorIntegration, &awsapigateway.MethodOptions{
		AuthorizationType: awsapigateway.AuthorizationType_COGNITO,
		Authorizer:        cognitoAuthorizer,
	})
	alarmsResource.AddMethod(jsii.String("GET"), alarmGetterIntegration, &awsapigateway.MethodOptions{
		AuthorizationType: awsapigateway.AuthorizationType_COGNITO,
		Authorizer:        cognitoAuthorizer,
	})
	alarmIDResource := alarmsResource.AddResource(jsii.String("{id}"), nil)
	alarmIDResource.AddMethod(jsii.String("DELETE"), alarmDeleterIntegration, &awsapigateway.MethodOptions{
		AuthorizationType: awsapigateway.AuthorizationType_COGNITO,
		Authorizer:        cognitoAuthorizer,
	})

	phoneModiferResource := myGateway.Root().AddResource(jsii.String("update-phone-number"), nil)
	phoneModiferResource.AddMethod(jsii.String("POST"), phoneModifierIntegration, &awsapigateway.MethodOptions{
		AuthorizationType: awsapigateway.AuthorizationType_COGNITO,
		Authorizer:        cognitoAuthorizer,
	})
	phoneVeriferResource := myGateway.Root().AddResource(jsii.String("verify-phone-number"), nil)
	phoneVeriferResource.AddMethod(jsii.String("POST"), phoneVerifierIntegration, &awsapigateway.MethodOptions{
		AuthorizationType: awsapigateway.AuthorizationType_COGNITO,
		Authorizer:        cognitoAuthorizer,
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

func env() *awscdk.Environment {
	return nil
}
