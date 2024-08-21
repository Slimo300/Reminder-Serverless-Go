package errors

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func ErrorResponse(message string, code int) (events.APIGatewayProxyResponse, error) {

	responseJSON, _ := json.Marshal(map[string]string{
		"message": message,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Headers":     "Content-Type",
			"Access-Control-Allow-Methods":     "OPTIONS, GET, POST, DELETE",
			"Access-Control-Allow-Credentials": "true",
		},
		Body: string(responseJSON),
	}, nil
}

// It returns internal server error
func Internal(err error) (events.APIGatewayProxyResponse, error) {
	log.Println(err.Error())
	return ErrorResponse("internal server error", http.StatusInternalServerError)
}

// It returns bad request response with given message
func Unauthorized(message string) (events.APIGatewayProxyResponse, error) {
	return ErrorResponse(message, http.StatusUnauthorized)
}

// It returns bad request response with given message
func BadRequest(message string) (events.APIGatewayProxyResponse, error) {
	return ErrorResponse(message, http.StatusBadRequest)
}
