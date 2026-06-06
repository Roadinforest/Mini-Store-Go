package apperror

import "fmt"

const (
	CodeInternal        = "INTERNAL_ERROR"
	CodeValidation      = "VALIDATION_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeBadRequest      = "BAD_REQUEST"
	CodeOutOfStock      = "OUT_OF_STOCK"
	CodeServiceDisabled = "SERVICE_DISABLED"
)

type Error struct {
	Code    string
	Message string
	Cause   error
	Details interface{}
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func New(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func Wrap(code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func WithDetails(err *Error, details interface{}) *Error {
	err.Details = details
	return err
}
