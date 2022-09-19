package errors

import (
	"fmt"
	"runtime/debug"
)

type AppError struct {
	Inner      error
	Message    string
	StackTrace string
	Misc       map[string]interface{}
}

func WrapError(err error, msgf string, msgArgs ...interface{}) AppError {
	return AppError{
		Inner:      err,
		Message:    fmt.Sprintf(msgf, msgArgs...),
		StackTrace: string(debug.Stack()),
		Misc:       make(map[string]interface{}),
	}
}

func (err AppError) Error() string {
	return err.Message
}
