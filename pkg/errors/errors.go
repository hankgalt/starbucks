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

type AppError struct {
	Inner      error
	Message    string
	StackTrace string
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
