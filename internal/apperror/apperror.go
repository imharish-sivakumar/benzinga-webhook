// Package apperror provides utilities to handle and map custom validation errors.
package apperror

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var (
	errRequired            = errors.New("is required")
	errMustBePositive      = errors.New("must be a positive number")
	errMustBeAtLeast3Chars = errors.New("must be at least 3 characters long")
	errInvalidDateTime     = errors.New("must be a valid datetime in RFC3339 format")
	errInvalidIP           = errors.New("must be a valid IP address")
	errInvalidPhoneFormat  = errors.New("must match format 555-1212-123")
)

var customErrors = map[string]error{
	"LogEntry.UserID.required":                      errRequired,
	"LogEntry.UserID.gte":                           errMustBePositive,
	"LogEntry.Total.required":                       errRequired,
	"LogEntry.Total.gt":                             errMustBePositive,
	"LogEntry.Title.required":                       errRequired,
	"LogEntry.Title.min":                            errMustBeAtLeast3Chars,
	"LogEntry.Meta.required":                        errRequired,
	"LogEntry.Logins.required":                      errRequired,
	"LogEntry.Logins.datetime":                      errInvalidDateTime,
	"LogEntry.Logins.ip":                            errInvalidIP,
	"LogEntry.Meta.PhoneNumbers.Home.required":      errRequired,
	"LogEntry.Meta.PhoneNumbers.Home.phoneformat":   errInvalidPhoneFormat,
	"LogEntry.Meta.PhoneNumbers.Mobile.required":    errRequired,
	"LogEntry.Meta.PhoneNumbers.Mobile.phoneformat": errInvalidPhoneFormat,
}

// CustomValidationError converts validator errors and JSON decoding errors into a standardized format.
func CustomValidationError(err error) []map[string]string {
	errList := make([]map[string]string, 0)

	var (
		validationErr validator.ValidationErrors
	)

	switch {
	case errors.As(err, &validationErr):
		for _, e := range validationErr {
			field := e.StructNamespace()
			key := field + "." + e.Tag()

			errMsg := fmt.Sprintf("%s is invalid", field)
			if v, ok := customErrors[key]; ok {
				errMsg = v.Error()
			}

			errList = append(errList, map[string]string{e.Field(): errMsg})
		}
	}
	return errList
}
