package domain_errors

import "errors"

var (
	ErrInvalidDate    = errors.New("invalid date format provided")
	ErrFlightNotFound = errors.New("no flights found matching criteria")
	ErrDatabaseQuery  = errors.New("database query failed unexpectedly")
	ErrInvalidID      = errors.New("invalid or malformed instance ID provided")
)
