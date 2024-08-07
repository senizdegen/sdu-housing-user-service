package apperror

import (
	"encoding/json"
	"fmt"
)

var (
	ErrNotFound = NewAppError("not found", "NS-000010", "")
)

type AppError struct {
	Err              error  `json:"-"`
	Message          string `json:"message,omitempty"`
	DeveloperMessage string `json:"developer_message,omitempty"`
	Code             string `json:"code,omitempty"`
}

func NewAppError(message, code, developerMessage string) *AppError {
	return &AppError{
		Err:              fmt.Errorf(message),
		Code:             code,
		Message:          message,
		DeveloperMessage: developerMessage,
	}
}

func (ae *AppError) Error() string {
	return ae.Err.Error()
}

func (ae *AppError) Unwrap() error {
	return ae.Err
}

func (ae *AppError) Marshal() []byte {
	bytes, err := json.Marshal(ae)
	if err != nil {
		return nil
	}
	return bytes
}

func BadRequestError(message string) *AppError {
	return NewAppError(message, "NS-000002", "something wrong with user data")
}

func systemError(developerMessage string) *AppError {
	return NewAppError("system error", "NS-000001", developerMessage)
}
