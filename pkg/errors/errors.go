package errors

import (
	"errors"
	"fmt"
	"runtime/debug"
)

var (
	ErrUserAccessDenied = errors.New("you do not have access to the requested resource")
	ErrNotFound         = errors.New("the requested resource not found")
	ErrTooManyRequests  = errors.New("you have exceeded throttle")
	ErrNilContext       = errors.New("context is nil")
)

const (
	ERROR_GEOCODING_POSTALCODE     string = "error geocoding"
	ERROR_ENCODING_LAT_LONG        string = "error encoding lat/long"
	ERROR_DECODING_BOUNDS          string = "error decoding bounds"
	ERROR_NO_STORE_FOUND           string = "no store found"
	ERROR_NO_STORE_FOUND_FOR_ID    string = "no store found for id"
	ERROR_STORE_ID_ALREADY_EXISTS  string = "store already exists"
	ERROR_UNMARSHALLING_STORE_JSON string = "error unmarshalling store json to store"
	ERROR_MARSHALLING_RESULT       string = "error marshalling result to store json"
)

type AppError struct {
	Inner      error
	Message    string
	StackTrace string
}

func NewAppError(errMsg string) AppError {
	return AppError{
		Inner:      errors.New(errMsg),
		Message:    errMsg,
		StackTrace: string(debug.Stack()),
	}
}

func WrapError(err error, msgf string, msgArgs ...interface{}) AppError {
	errMsg := err.Error()
	if errMsg != "" {
		errMsg = fmt.Sprintf(msgf, msgArgs...)
	}
	return AppError{
		Inner:      err,
		Message:    errMsg,
		StackTrace: string(debug.Stack()),
	}
}

func (err AppError) Error() string {
	return err.Message
}
