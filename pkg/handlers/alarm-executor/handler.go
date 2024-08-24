package alarmexecutor

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type AlarmEvent struct {
	UserID  string `json:"userID"`
	Message string `json:"message"`
}

type SnsApiClient interface {
	Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
}

type Handler struct {
	SNSClient SnsApiClient
}

func (h *Handler) Handle(event AlarmEvent) error {

	if _, err := h.SNSClient.Publish(context.Background(), &sns.PublishInput{
		TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
		Message:  &event.Message,
		MessageAttributes: map[string]types.MessageAttributeValue{
			"userID": {
				DataType:    aws.String("String"),
				StringValue: &event.UserID,
			},
		},
	}); err != nil {
		return err
	}

	return nil
}
