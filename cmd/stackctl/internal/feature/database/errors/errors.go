package errors

import "fmt"

type DatabaseError struct {
	Op  string
	Err error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database operation '%s' failed: %v", e.Op, e.Err)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

func NewDatabaseError(op string, err error) error {
	return &DatabaseError{Op: op, Err: err}
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for '%s': %s", e.Field, e.Message)
}

func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

type UserAlreadyExistsError struct {
	Username string
}

func (e *UserAlreadyExistsError) Error() string {
	return fmt.Sprintf("user '%s' already exists", e.Username)
}

func NewUserAlreadyExistsError(username string) error {
	return &UserAlreadyExistsError{Username: username}
}

type UserNotFoundError struct {
	Username string
}

func (e *UserNotFoundError) Error() string {
	return fmt.Sprintf("user '%s' not found", e.Username)
}

func NewUserNotFoundError(username string) error {
	return &UserNotFoundError{Username: username}
}

type ConnectionError struct {
	Host string
	Port int
	Err  error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("failed to connect to %s:%d: %v", e.Host, e.Port, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

func NewConnectionError(host string, port int, err error) error {
	return &ConnectionError{Host: host, Port: port, Err: err}
}
