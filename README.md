# Reminder-Serverless-Go

## This is a serverless app build entirely on AWS with Go language on backend and React on frontend.

The main goal of this application is to send SMS reminders about events, duties and all the stuff we tend to forget. Application supports both one time events and cyclic events of which you can have as many as you need for one topic/event/reminder.

## Architecture

This application uses several AWS services that work together to deliver us the functionality we need. It uses EventBridge Scheduler in order to create alarms on given timestamp or cron expression, SNS Topic for sending SMS notifications (read about SNS Sandbox first if you intend to use it), Cognito User Pool for handling authentication and authorization and two DynamoDB tables - one for storing events data, and one for logic behind changing phone numbers assigned to an account.

For handling our application buisness logic, there are 7 AWS Lambda functions written in Go language that do following actions:
- alarm-creator - integrated with API Gateway, it creates one event with any number of timestamp or cron based alarms
- alarm-getter - integrated with API Gateway, it returns all events that belong to a user making request
- alarm-deleter - integrated with API Gateway, it deletes one event with all its alarms (events are not deleted automatically even if there won't be any alarms anymore)
- phone-number-modifier - integrated with API Gateway, it saves new phone number along with generated verification code in DynamoDB
- phone-number-verifier - integrated with API Gateway, it checks provided verification code and changes user phone number both in Cognito User Pool and SNS subscription
- alarm-executor - executed by EventBridge Scheduler when alarm is set on, it sends SMS to user who created the alarm
- post-confirmation-trigger - executed as Cognito User Pool trigger when new user is signed up. It creates a new SNS subscription for him

## How to run

Application is build with AWS CDK so to run it you need to:
- have access to AWS account,
- have AWS CDK installed
```console
foo@bar:~$ npm install -g aws-cdk
...

When you have your AWS keys in place (either in environment variables or in ~/.aws/credentials file), and CDK installed, just run:
```console
foo@bar:~$ cdk deploy
...

For now frontend code is not deployed with application to AWS, although there is possibility to deploy it to S3 as static site or to deploy it with AWS Amplify.
