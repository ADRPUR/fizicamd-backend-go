package services

import "fmt"

type ServiceError struct {
	Status  int
	Message string
}

func (e ServiceError) Error() string {
	return e.Message
}

func ErrNotFound(msg string) error {
	return ServiceError{Status: 404, Message: msg}
}

func ErrBadRequest(msg string) error {
	return ServiceError{Status: 400, Message: msg}
}

func ErrForbidden(msg string) error {
	return ServiceError{Status: 403, Message: msg}
}

func ErrUnauthorized(msg string) error {
	return ServiceError{Status: 401, Message: msg}
}

func WrapError(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
