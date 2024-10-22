package postconfirmationtrigger_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	postconfirmationtrigger "github.com/Slimo300/Reminder-Serverless-Go/pkg/handlers/post-confirmation-trigger"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type mockSns struct{}

func (m *mockSns) Subscribe(context.Context, *sns.SubscribeInput, ...func(*sns.Options)) (*sns.SubscribeOutput, error) {
	return &sns.SubscribeOutput{
		SubscriptionArn: aws.String("some_arn"),
	}, nil
}

type mockCognito struct{}

func (m *mockCognito) AdminUpdateUserAttributes(context.Context, *cognito.AdminUpdateUserAttributesInput, ...func(*cognito.Options)) (*cognito.AdminUpdateUserAttributesOutput, error) {
	return nil, nil
}

func TestHandler(t *testing.T) {
	handler := postconfirmationtrigger.Handler{
		SnsClient:     &mockSns{},
		CognitoClient: &mockCognito{},
	}

	testCases := []struct {
		name           string
		event          events.CognitoEventUserPoolsPostConfirmation
		expectedResult events.CognitoEventUserPoolsPostConfirmation
		expectedError  error
		returnErr      bool
	}{
		{
			name: "no sub",
			event: events.CognitoEventUserPoolsPostConfirmation{
				Request: events.CognitoEventUserPoolsPostConfirmationRequest{
					UserAttributes: map[string]string{
						"phone_number": "+11123456789",
					},
				},
			},
			expectedResult: events.CognitoEventUserPoolsPostConfirmation{
				Request: events.CognitoEventUserPoolsPostConfirmationRequest{
					UserAttributes: map[string]string{
						"phone_number": "+11123456789",
					},
				},
			},
			expectedError: errors.New("invalid user attributes"),
			returnErr:     true,
		},
		{
			name: "no phone_number",
			event: events.CognitoEventUserPoolsPostConfirmation{
				Request: events.CognitoEventUserPoolsPostConfirmationRequest{
					UserAttributes: map[string]string{
						"sub": "1",
					},
				},
			},
			expectedResult: events.CognitoEventUserPoolsPostConfirmation{
				Request: events.CognitoEventUserPoolsPostConfirmationRequest{
					UserAttributes: map[string]string{
						"sub": "1",
					},
				},
			},
			expectedError: errors.New("invalid user attributes"),
			returnErr:     true,
		},
		{
			name: "no sub",
			event: events.CognitoEventUserPoolsPostConfirmation{
				Request: events.CognitoEventUserPoolsPostConfirmationRequest{
					UserAttributes: map[string]string{
						"phone_number": "+11123456789",
						"sub":          "1",
					},
				},
			},
			expectedResult: events.CognitoEventUserPoolsPostConfirmation{
				Request: events.CognitoEventUserPoolsPostConfirmationRequest{
					UserAttributes: map[string]string{
						"phone_number": "+11123456789",
						"sub":          "1",
					},
				},
			},
			expectedError: nil,
			returnErr:     false,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			res, err := handler.Handle(tC.event)

			if !reflect.DeepEqual(res, tC.expectedResult) {
				t.Errorf("Response received: %v is different than expected: %v", res, tC.expectedResult)
			}
			if tC.returnErr && tC.expectedError.Error() != err.Error() {
				t.Errorf("Error received: %v is different than expected: %v", err, tC.expectedError)
			}
			if !tC.returnErr && tC.expectedError != err {
				t.Errorf("Error received: %v is different than expected: %v", err, tC.expectedError)
			}
		})
	}
}
